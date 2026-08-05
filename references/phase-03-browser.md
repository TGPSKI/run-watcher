---
name: run-watcher-phase-03
description: "Phase 3: The browser -- one shared loader, tabbed curses views, a detail pane, and the noise floor rendered. Build only when you can name a view the watcher cannot show."
metadata:
  author: TGPSKI
  version: "1.0"
parent: run-watcher
---

# Phase 3: The Browser

The watcher answers *"what is happening."* It cannot answer *"why is this one
winning."* This phase adds tabbed views over the same data, through the same
loader.

**Typical PR**: 1
**Files produced**: a shared loader module, `<scripts>/<job>-tui.py`, a
vendored `tui/` package, a `matrix` make target

## Prerequisites

- Phase 2 watcher merged and run against a real job.
- **A named view the watcher cannot show.** If you cannot name one, stop —
  building this speculatively produces tabs nobody opens.

**Carry forward from Phases 1–2**: the evidence table, the filter grammar and
knob names, the callout list, `NULL_BAND`, the ranking axis, and the table
renderer the watcher calls.

---

## Step 1: Extract the loader first

This step comes first because doing it second means writing the aggregation
twice, and then living with two numbers (L10).

**Inspect**:

| Status | Action |
|--------|--------|
| Phase 2 calls a `table.py`-style renderer | Split it: a loader returning records, and a formatter. Both surfaces call the loader |
| Phase 2 computes inline in bash | Move the computation into a Python loader. The watcher calls it too — do not leave two implementations |
| A registered analysis already implements a statistic you want to show | **Import that module.** Never write a second implementation of a test that appears in a result |

**Generate** a loader with no rendering in it:

```python
def load_cells() -> list[dict]:
    """One record per unit. Unfiltered — filtering happens at render."""

def matches(tag: str, pattern: str) -> bool:
    """The Phase 2 filter grammar, shared so both surfaces agree."""

def newest_mtime() -> float:
    """For change detection only — never for liveness (L3)."""
```

Write to the module the report already uses, or a new one both then import.

**The test that this step succeeded**: change an aggregation rule in one place
and both the watcher and the browser change.

---

## Step 2: Name the views

**Decide** — one tab per question, and each must be a question the others
cannot answer:

1. "What are the two-to-four questions you ask of finished results?"

| Common view | Answers | Notes |
|---|---|---|
| cells | "which unit, ranked" | The default. Carries the detail pane |
| rollup | "which condition, aggregated" | Add a baselined bar chart |
| pairs | "is the difference real" | Import the registered test (Step 1) |
| cost | "what did it cost to get that" | Mark the Pareto frontier `★` |
| reference | "what is this unit / condition even asking for" | **Static.** See Step 5 |

| Status | Action |
|--------|--------|
| Fewer than two distinct questions | Stop. Add a sort to the watcher instead |
| More than five | Some are the same question with a different sort. Merge them |

---

## Step 3: The curses shell

**Do not generate the drawing layer. Copy it.**

`assets/tui/` in this skill holds the whole package — `framework.py`,
`charts.py`, `fmt.py`, `windows.py` — 384 stdlib-only lines vendored
byte-identically across three codebases since 2026-07-13. Copy all four:
`charts.py` reaches `windows.py` through a lazy import inside a branch, so a
trimmed copy works until the first stacked bar and then raises. Copy them into the target repository beside your
app:

```bash
cp -r <skill>/assets/tui <target>/<pkg>/tui
```

| Status | Action |
|--------|--------|
| Target has no TUI package | Copy `assets/tui/` verbatim. Read `assets/tui/README.md` first |
| Target already vendors this framework | Diff against `assets/tui/`; keep the target's copy if identical, and do not "modernize" it |
| Target has a different curses base | Keep theirs. Two frameworks is worse than an older one |

You get `TuiApp` (bounds-checked `_put`, colour pairs, a run loop with a
min-size guard, a footer renderer, and **`scroll_indicator` — which is L8's
`N–M of T`, already written**), plus `curses_main`, `bar_chart`, and the safe
formatters.

Rewriting it is the most common way this phase goes wrong: an agent
re-derives a `_put` without the clip, or without swallowing `curses.error`, and
the viewer starts dying on rows one column too wide.

**Generate** only the app on top of it.

```python
self.curses.halfdelay(20)   # 2s tick: getch returns -1, so we refresh
```

| Concern | Rule | Law |
|---|---|---|
| Reload | Only when data can have changed — a tick, or a key that alters what is rendered | — |
| Scrolling | Repaint from the captured frame. A scroll that re-scans 90 archives reads as a *broken* scrollbar, not a slow one | — |
| Cursor | Anchor to a stable identity, re-find each tick | L7 |
| Space | Bounded sections whole; the unbounded one scrolls, with a scrollbar and `N–M of T` | L8 |
| Sort | Key and direction stored **separately**, so flipping does not lose the column | — |
| Legend | Advertise only what *this* view and *this* depth respond to | L13 |
| Empty state | `no cells match this filter — [F] clears it` — never a bare blank | L5 |

Terminal details that each cost a debugging session:

```python
# Cursor keys arrive as CSI (ESC [ A) in normal mode and SS3 (ESC O A) in
# application-cursor mode, which many terminals enable by default. Decoding
# only CSI makes the arrows dead keys on exactly those.

# The filter buffer starts EMPTY, not pre-filled: a pre-filled buffer
# silently appends what you type onto the old pattern.

# The scrollbar draws only when content overflows, in the last column, so it
# never collides with a row that happens to be full width.
```

---

## Step 4: The detail pane

**Generate** a `[space]` detail card on the cells view — and only on views that
have one (L13).

| Rule | Law |
|---|---|
| Every row is suspicious on its own; detail is one keystroke away | — |
| The reason field resolves per verdict — populated on exactly the rows whose verdict needs explaining | L12 |
| The pane scrolls independently and keeps its own offset | L8 |
| Opening detail on a row that then finishes must not swap the content | L7 |

---

## Step 5: The reference tabs, behind a divider

**Inspect**: what an operator must know for a number to mean anything — what
each unit asks for, what each condition *is*, what rules the run is bound by.

| Status | Action |
|--------|--------|
| That information lives only in a long registration document | Add static tabs that read it **off disk**, so they cannot drift |
| It is already on screen | Skip |

**Generate** the tab bar with an explicit divider:

```python
VIEWS = ("cells", "rollup", "cost", "pairs", "tasks", "design")
REFERENCE_FROM = VIEWS.index("tasks")
# Run data first, then the reference pages. They are separated because they
# answer different questions: everything left of the divider changes as the
# run proceeds, everything right of it is ground truth that does not move,
# and mixing them invites reading a static table as a live one.
```

---

## Step 6: Render the statistics

**Generate** the noise floor as a rendering rule, not a footnote (L9):

```python
NULL_BAND = 2.4          # measured single-run noise floor, in points

attr = (RED  if spread > 2 * NULL_BAND else
        YEL  if spread > NULL_BAND     else DIM)

verdict = ("RESOLVED" if p < 0.05 and abs(delta) > NULL_BAND
           else "null" if abs(delta) <= NULL_BAND
           else "underpowered")
```

| Status | Action |
|--------|--------|
| A noise floor was measured | Use it. Cite the measurement in the comment |
| None exists | Render the band as `UNKNOWN` and say so on screen. Do not invent a number |

---

## Validate

```bash
python3 <path>                     # opens on the picker
python3 <path> '<pattern>'         # pre-filtered
make matrix                        # via the repo's target
```

| Check | Expectation |
|---|---|
| Same quantity, watcher vs browser | **Identical.** Any difference is L10 and blocks the PR |
| Arrow keys in your terminal | Work — if not, SS3 decoding is missing |
| Every advertised key | Does something at that depth (L13) |
| Cursor during a live run | Stays on the same object as rows land (L7) |
| Narrow terminal (80×24) | No section silently absent (L8) |
| A difference under the noise floor | Renders dim (L9) |

## PR Checkpoint

**STOP. This phase is complete. Create the PR now.**

**Title**: `matrix: interactive browser over the same loader — Phase 3`

**Files to include**:
- The shared loader module
- `<scripts>/<job>-tui.py`, plus `tui/` copied verbatim from the skill's
  `assets/tui/` (unmodified — a diff against it should be empty)
- The Phase 2 watcher, refactored to call the shared loader
- The `matrix` make target and a `VIEWING.md` documenting every column *and
  what it can mislead you about*

`VIEWING.md` is not optional. The columns that mislead are the ones worth
having, and an undocumented misleading column is a trap you set for yourself.

---

## Next Phase

| Condition | Route |
|---|---|
| The screen sits empty while workers are busy | `references/phase-04-live-view.md` |
| Results land fast enough to watch | **Stop.** Phase 4 buys nothing here |

**Carry forward**: the loader API, `VIEWS` and the divider index, the framework
package, `NULL_BAND`, the cursor-anchoring helper, and the filter grammar.
