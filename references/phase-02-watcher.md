---
name: run-watcher-phase-02
description: "Phase 2: The watcher -- one screen on a tick, ranked by the question, with named callouts. Shell or curses; Step 0 decides. Often the whole answer."
metadata:
  author: TGPSKI
  version: "1.0"
parent: run-watcher
---

# Phase 2: The Watcher

**One screen, redrawn on a tick, ranked by the question from Phase 1, with the
callouts rendered loudly.** That is the deliverable. The substrate is a
decision, and it is the first thing this phase makes.

**Typical PR**: 1
**Files produced**: `<scripts>/watch-<job>.sh` **or** `<pkg>/<job>_tui.py` +
`<pkg>/tui/`, plus a `watch` make target

## Prerequisites

- Phase 1 plan merged.

**Carry forward from Phase 1**: the evidence table (paths and what each carries),
the ranking axis, the callout list with glyphs, `NULL_BAND`, the liveness
signal, and the stopping point. Do not re-ask for any of these.

---

## Step 0: Choose the substrate

**Inspect** the Phase 1 plan — specifically the question, and the stopping
point recorded in its Step 5.

**The two substrates**, since the choice is meaningless without them:

**Shell + ANSI.** You `printf` whole frames and move the cursor with escape
codes (`\033[H` to go home, `\033[K` to clear a line). The terminal is a
teletype you are repainting. `read -t` doubles as both the tick and the
keyboard. Zero dependencies, and the output pipes — `NOCOLOR=1 … | head` is a
paste into an issue.

**Curses.** The stdlib `curses` module, wrapping the C `ncurses` library. You
address individual cells — `addstr(y, x, ...)` — set colour pairs, read keys
without Enter, and it computes the minimal characters needed to update the
screen. The name is a pun on *cursor optimization*, from the 1978 BSD library
that minimized bytes sent over slow serial lines; it is the actual API name,
not a term of art invented here. `halfdelay(20)` gives a 2-second tick that
still responds to a keypress instantly.

| | shell + ANSI | curses |
|---|---|---|
| Drawing | whole frames, you manage the escape codes | cell-addressed, diffed for you |
| Tick + input | `read -t` is both | `halfdelay` is both |
| Pipeable to a file or `head` | **yes** | no |
| Detail panes, independent scrolling | painful | tractable |
| Dependencies | none | none (stdlib) |
| Cost to start | ~250 lines | a `cp` of `assets/tui/`, then the app |

**Decide**:

1. "Does one ranked table answer the question, or do you already know you need
   several views and a detail pane?"

| Status | Substrate | Why |
|--------|-----------|-----|
| One ranked table answers it | **Shell** — Steps 1–6 below | ~250 lines, no Python, no framework. Two of the five generations in `LINEAGE.md` stopped here permanently |
| You need tabs or a detail pane **and you already know it** | **Curses** — copy `assets/tui/`, then jump to `phase-03-browser.md` Step 3 and return here for Steps 2, 4, 5, 6 | The framework is a `cp`. Writing the render in shell first and porting it is the throwaway work |
| The screen must show work *in flight* | **Curses**, and read `phase-04-live-view.md` before building | An in-flight view wants a detail pane almost immediately |
| You cannot tell | **Shell.** Build it, watch a real run, then decide | The cheapest way to find out which views you need is to miss one |

**This ordering is not a ladder.** Generation 0 in `LINEAGE.md` went straight to
curses over live server logs and was right to; generations 1 and 2 chose shell
with full knowledge of the curses version and were also right. What decides it
is the question, not seniority.

**What does NOT change with the substrate**: every rule in Steps 2–6. Ranked by
the question (L6), named callouts with glyphs (L11), two liveness thresholds
(L4), distinct glyphs for distinct absences (L5), and no mutation anywhere
(L1). Those are the phase. The redraw loop is an implementation detail.

| Status | Action |
|--------|--------|
| Chose curses | Do **Step 0.5**, then skip Step 1 and do Steps 2–6 |
| Chose shell | Continue to Step 1 |

### Step 0.5 — curses only: copy the drawing layer

```bash
cp -r <skill>/assets/tui <target>/<pkg>/tui
```

384 stdlib-only lines, vendored byte-identically across three codebases. Read
`assets/tui/README.md` first, copy all four modules, and **do not rewrite
them** — `phase-03-browser.md` Step 3 explains what you get and how rewriting
it goes wrong. Then use `halfdelay(20)` as the tick in place of Step 1's
`read -t`, and continue at Step 2.

---

## Step 1: The redraw loop *(shell substrate)*

**Inspect**: check whether the repo has an existing watch script whose
conventions should be matched.

| Status | Action |
|--------|--------|
| A `watch-*.sh` exists for another job in this repo | Match its structure, flags, and colour handling. Do **not** import its code |
| No precedent | Use the skeleton below |

**Generate** — the loop. Every line below is load-bearing:

```bash
#!/usr/bin/env bash
# Live status of <job>, ranked by <axis from Phase 1>.
#   LOOP=1 bash <path>          # live, redraw in place
#   NOCOLOR=1 bash <path>       # plain, for piping
#
# Cells are RANKED, not listed in run order: the question asked of this screen
# is always "<question from Phase 1>", and run order buries that.        [L6]
#
# <N> callouts exist because each has already silently changed a run:    [L11]
#   <NAME>  <what it means, and what it cost>

set -u
HERE="$(cd "$(dirname "$0")" && pwd)"   # resolve BEFORE any cd

if [ -z "${NOCOLOR:-}" ]; then
  B=$'\033[1m'; D=$'\033[2m'; R=$'\033[0m'
  GRN=$'\033[32m'; YEL=$'\033[33m'; RED=$'\033[31m'; CYN=$'\033[36m'
else
  B=; D=; R=; GRN=; YEL=; RED=; CYN=
fi

snapshot() { : ; }        # Steps 2-4 fill this in

if [ -n "${LOOP:-}" ]; then
  printf '\033[?25l'                                  # hide cursor
  trap 'printf "\033[?25h\n"; exit 0' INT TERM EXIT   # ALWAYS restore it
  while :; do
    printf '\033[H'                                   # home, do NOT clear
    snapshot | while IFS= read -r line; do
      printf '%s\033[K\n' "$line"                     # erase to EOL: no tails
    done
    printf '\033[J'                                   # drop rows if layout shrank
    read -t "${INTERVAL:-5}" -rsn1 key || true        # the tick IS the input
    case "$key" in q) break ;; esac
  done
else
  snapshot
fi
```

Why each piece:

| Choice | Reason |
|---|---|
| `\033[H` not `clear` | `clear` flashes on every tick and churns scrollback |
| `\033[K` per row | Shorter lines otherwise leave tails from the previous frame |
| `\033[J` at the end | Drops leftover rows when the content shrinks |
| `read -t` as the tick | One call waits out the interval *and* collects a keystroke — interactive sorting with no second thread, no curses |
| `trap ... EXIT` | A hidden cursor that survives Ctrl-C is a broken terminal, and the user will blame the watcher |
| `HERE` before any `cd` | `dirname "$0"` points nowhere after a `cd` |

---

## Step 2: Scope, and the knobs

**Inspect** the output root for how many campaigns/runs share it.

| Status | Action |
|--------|--------|
| One run per output root | A single `FILTER` knob is enough |
| Multiple campaigns share a root | Add a `SCOPE` knob. **Required** — watching all at once is what makes a screen unreadable |
| Multiple machines/rigs | Add a `RIG` knob that also drops the other's whole section |

**Generate**: composable environment knobs, documented in the header and in
`--help`.

```
FILTER=<pat>   prefix · glob · substring · comma-ORs · leading ! negates
SCOPE=...      which campaign; the registered one vs the exploratory pile
SORT=<col>     the ranking axis, overridable
NOCOLOR=1      strip ANSI, for piping or pasting
```

Make the filter grammar forgiving — bare prefix, glob, substring, comma-OR,
`!` negation. An operator mid-incident should not be debugging a filter.

---

## Step 3: The table

**Generate**, in this order top to bottom:

1. **The in-flight unit and its progress.** The only thing that changes second
   to second, so it goes where the eye lands first (L6).
2. A blank line.
3. **The ranked results table.**
4. Reference material (a column glossary behind `HELP=1`).
5. A closing rule and totals — the visual floor, so it stays put no matter what
   is toggled on.

Column rules:

| Rule | Law |
|---|---|
| Render zero as `-`, unrecorded as `?`, inferred as `~` | L5 |
| Bracket the progress bar; use a track character that is not `.` | L5 |
| Below `NULL_BAND`, render dim | L9 |
| Give-ups and truncations in red wherever a cost is shown | L16 |
| Right-align digits; use fixed widths so columns do not dance between ticks | — |

If a `table.py` or equivalent already renders these rows for the report, **call
it** rather than reimplementing (L10).

---

## Step 4: The callouts

**Inspect**: the callout list carried from Phase 1.

**Generate** one emitter per callout, printed under the unit it belongs to:

```bash
[ "${reflect:-0}" -gt 0 ] && \
  printf '       %sx REFLECTION MODE - payload paginated (%s nav calls)%s\n' \
         "$RED$B" "$pages" "$R"
[ "${other:-0}" -gt 0 ] && \
  printf '       %s! %s unattributed calls (summarization is silent)%s\n' \
         "$YEL" "$other" "$R"
```

| Glyph | Severity | Rendered |
|---|---|---|
| `x` | Critical — the measurement is compromised | bold red |
| `!` | Advisory — something is happening you did not ask for | yellow |
| `~` | Informational — waste, not corruption | yellow, dim |

The glyph carries the severity so it survives `NOCOLOR=1` into a paste (L11).

---

## Step 5: Liveness

**Inspect**: the liveness signal chosen in Phase 1.

| Signal available | Implementation |
|---|---|
| Worker names its dir in `argv` | `ps -eo args` and match. No heuristic (L3) |
| Marker file with a pid | Probe the pid — **and read L1 before writing the probe** |
| Neither | Fall back to file age, and label it explicitly as an inference |

**Generate** the stall rule with two thresholds (L4):

```bash
age=999999
[ -f "$log" ] && age=$(( $(date +%s) - $(command stat -c %Y "$log") ))
stale=""; [ "$age" -gt "${STALE_S:-90}" ]     && stale=" (STALE ${age}s)"
[ "$age" -gt "${ABANDONED_S:-1800}" ] && continue   # debris, not a run
```

Never let the debris threshold swallow the stall threshold. A hung worker is
exactly what you built this for.

---

## Step 6: Refuse to be dangerous

| Rule | Law |
|---|---|
| The script reads. It never writes to, signals, or locks anything the job uses | L1 |
| Every read tolerates a missing or half-written file — `2>/dev/null`, `${x:-0}` | L2 |
| No cleanup, no deletion, no kill in this file | L15 |
| Anything that spends real resources lives in a **separate** entry point with an explicit flag | L14 |

Check the generated script for `rm`, `kill`, `>`, `mv`, `truncate`. There
should be none.

---

## Validate

**Run it against a real job for one full session.** This is the gate.

```bash
bash <path>                              # one-shot: does it render at all
LOOP=1 bash <path>                       # live: watch a real run
NOCOLOR=1 bash <path> | head -40         # pipe-safe, glyphs still legible
bash <path> &                            # while a run is being killed:
                                         # does it show dead workers as live?
```

| Observation | Action |
|---|---|
| Renders correctly, callouts fire when expected | Ship it |
| Rows claim to run after a teardown | L3 violated. Fix before the PR |
| It crashed on a torn line | L2 violated. Fix before the PR |
| A column is blank on every row | The path does not resolve — recheck the Phase 1 evidence table |
| It flickers or leaves tails | Revisit the Step 1 escape sequences |
| The run misbehaved while it was open | **Stop.** L1 violated. Find out how before anything else |

Record what the screen showed during validation in the Phase 1 plan. That
record is the start of the display's regression history (L16).

## PR Checkpoint

**STOP. This phase is complete. Create the PR now.**

**Title**: `watch: live status for <job>, ranked by <axis> — Phase 2`

**Files to include**:
- `<scripts>/watch-<job>.sh`
- The `watch` make target
- The Phase 1 plan, updated with what validation showed

**After creating the PR**, inform the user:

> Watcher shipped. Run it for a while before deciding on Phase 3. The right
> reason to continue is a specific question this screen cannot answer.

---

## Next Phase

| Condition | Route |
|---|---|
| You can name a view this screen cannot show, and Phase 2 was shell | `references/phase-03-browser.md` — all steps |
| Same, but Phase 2 was already curses | `references/phase-03-browser.md` — Steps 1, 2, 4, 5, 6. Step 3 is done |
| You cannot | **Stop here.** This is a complete deliverable |
| Results land only on completion, and the screen sits empty | `references/phase-04-live-view.md` — Phase 3 is optional |

**Carry forward**: the loader/table entry point, the knob names and filter
grammar, the callout emitters, `NULL_BAND`, and the liveness implementation.
