---
name: run-watcher-phase-04
description: "Phase 4: The live view -- render work in flight by joining the evidence a running worker leaves on disk. Probed liveness, stall detection, and a teardown that never deletes results."
metadata:
  author: TGPSKI
  version: "1.0"
parent: run-watcher
---

# Phase 4: The Live View

Every previous phase reads *archives*. A record is written only when a unit
finishes, so at long timeouts and low concurrency the screen sits at 0% for
minutes while the machine is fully loaded.

The evidence already exists — a stream being appended to, a store being
written, a marker naming the unit. Nothing joins them. This phase is that join.

**Typical PR**: 1
**Files produced**: a live reader module, a `live` view in the browser, a
`stop` command

## Prerequisites

- Phase 3 browser merged (or Phase 2, if you skipped 3).
- **The screen has actually been observed sitting empty during real work.** If
  results land fast enough to watch, this phase buys nothing.

**Carry forward from Phases 1–3**: the evidence table with its "written when"
column, the liveness signal, the loader API, the cursor-anchoring helper, and
the callout list.

---

## Step 1: The reader

**Generate** a module of pure functions with no curses import and no state, so
it is testable without a terminal and the renderer is testable without a job:

```python
"""What is happening right now, read off disk without touching the run.

A record is written only after a unit finishes, so the results file cannot
show work in flight. The evidence is already there: the worker streams
NDJSON to <out>/stdout.txt as it arrives, and the runner drops a marker
naming the unit. This joins them.

Strictly read-only, and deliberately tolerant: every file it reads is being
written concurrently by a process that does not know it is being watched.
A half-written final line is normal and is skipped, not raised.       [L2]
A viewer that can crash a run is worse than no viewer.                [L1]
"""

def out_dirs() -> list[str]:        ...   # working dirs that exist
def busy_out_dirs() -> set[str]:    ...   # those a live process names
def snapshot() -> list[dict]:       ...   # per unit: what, how far, how long
def activity(out_dir) -> list[dict]:...   # recent events, newest first
def sweep(dry=False):               ...   # debris only — renderer never calls
```

| Rule | Law |
|---|---|
| Every read wrapped; unreadable → skipped, torn line → skipped, missing key → default | L2 |
| Nothing in this module writes, signals, or locks | L1 |
| `sweep()` is the sole exception and is never reachable from the render loop | L15 |

---

## Step 2: Liveness, probed

**Inspect** the signal recorded in Phase 1.

| Signal | Implementation |
|---|---|
| Worker names its working dir in `argv` | Scan `ps`, match. **Preferred** — direct, no heuristic |
| Marker file carries a pid | Probe the pid, with the guard below |
| Neither | File age, labelled on screen as an inference, not a fact |

**Generate** the pid probe with its platform guard. This comment stays:

```python
def _alive(pid: int) -> bool:
    """Does this pid exist? POSIX only -- see below.

    `os.kill(pid, 0)` is the standard existence probe on POSIX, where signal
    0 means "check, do not send". On Windows it is not a probe at all:
    `signal.CTRL_C_EVENT == 0`, so the call delivers a real Ctrl-C to the
    target's console group. A selftest wrote a marker carrying its own pid,
    this function checked it, and the check raised KeyboardInterrupt inside
    an unrelated subprocess several tests later.

    A liveness check must never be able to affect what it observes, so on
    Windows it declines to answer and the caller falls back to evidence
    that cannot misfire.                                              [L1]"""
    if not pid or os.name != "posix":
        return False
    try:
        os.kill(pid, 0)
    except OSError as e:
        return e.errno == errno.EPERM      # exists, not ours
    return True
```

**Journaled stores** (L3): if the evidence table lists a store with a sidecar,
freshness is `max(mtime(db), mtime(db-wal), mtime(db-journal))`. Reading the
main file alone reports a working unit as hung — measured at 282s stale against
a 29s-fresh WAL.

---

## Step 3: Stall and debris

**Generate** two thresholds, far apart (L4):

```python
# Units whose stream has not moved in this long are shown as stalled rather
# than dropped: a hung worker is exactly the thing you want to see.
STALE_S = 180
# Beyond this an out-dir is leftover debris from a killed run, not a run.
ABANDONED_S = 1800
```

| State | Rendered |
|---|---|
| Live, moving | normally |
| Live, not moving past `STALE_S` | **stalled**, highlighted, in place |
| No live process, under `ABANDONED_S` | dimmed, marked ended |
| No live process, over `ABANDONED_S` | omitted; counted in a debris line |

Never let the debris rule swallow the stall rule.

---

## Step 4: Read what the parent stream cannot see

**Inspect** the evidence table for quantities whose source structurally cannot
contain them.

| Symptom | Cause | Fix |
|---|---|---|
| A child/subagent count that is always exactly `0` | The parent stream carries none of a child's calls | Read child sessions from the store |
| Per-child cost missing | Same | Sum from the store, not the stream |
| A count that never moves at all | Not a measurement — a wiring bug | Cross-check against a second source |

A `0` that never moves is the anomaly. Measured on a real run: the parent
stream reported 26 calls where 66 had happened.

**Decide** before concluding it cannot be done:

1. "Is this genuinely unreadable, or merely not yet read?"

The honest answer is usually the second. A capability gap you assert without
checking is a capability gap you invented.

---

## Step 5: The live view's sections

**Generate** three stacked sections, ordered by what changes fastest (L6):

| Section | Bounded by | Space |
|---|---|---|
| running | the concurrency limit | whole (L8) |
| summary / rollup | the cell count | whole (L8) |
| graded, newest first | nothing — grows all run | scrolls, with a scrollbar |

```python
# Priority, not proportion. Running and summary are bounded, so they fit and
# should simply be shown whole. Graded grows without bound for the life of
# the run, so it is the section that absorbs the scrolling. Sharing the space
# proportionally gave graded 22 rows it did not need while truncating a 3-row
# running table, and on a 24-line terminal pushed graded off the screen
# entirely -- absent, not truncated, with nothing saying so.            [L8]
```

| Rule | Law |
|---|---|
| Progress counts only what has landed on disk | — |
| The results-side filter does **not** apply to the running section | L5 |
| Cursor anchors to the out-dir path, not the row index | L7 |
| Newest graded first — a run in progress is judged by what just landed | L6 |
| Include the start time, with the date; a run crossing midnight sorts wrongly on a bare clock | — |

---

## Step 6: Control — stop, never edit

**Generate** a separate command (`make stop`), not a hotkey (L14, L15):

| Does | Does not |
|---|---|
| Kill process groups, not bare pids | Delete results. **Ever** |
| Reclaim strays whose leader is gone | Touch the archive directory |
| Skip zombies when counting group members | Reset, reinitialize, or "clean" |
| Sweep temp debris | Run from inside the renderer |

```python
# Deliberately never deletes results. Removing debris and removing evidence
# are different operations, and only one of them belongs on a hotkey.  [L15]
```

**Also generate** a start-time refusal: if strays from a previous run exist,
refuse to start rather than launching into a polluted process table.

---

## Step 7: Salvage

**Inspect**: what a timeout kill currently discards.

```bash
du -sh <working-dir>          # before teardown
```

| Status | Action |
|--------|--------|
| Megabytes of events discarded on kill | Add a salvage path — the worker writes what it has before dying |
| Nothing meaningful buffered | Skip |

Measured: a kill discarded 27.6 MB of events, turning a real unit into a
0-call record. Salvage recovered 8 calls and 357,131 tokens from the same run.
**A cut-off unit did not give up**, and recording it as though it did is a
harness fault scored as a subject failure.

---

## Validate

Run against a real job — and against a *dying* one.

| Test | Expectation |
|---|---|
| Start a run, open the live view immediately | Units appear before any result is written |
| Kill the run, keep watching | Rows stop claiming to be live **within one tick** (L3) |
| Open a detail pane, let that unit finish | Content does not swap under you (L7) |
| Resize to 80×24 | Every section present or explicitly scrolled (L8) |
| Truncate a stream file mid-line | Viewer skips it and keeps going (L2) |
| Run the whole battery with the viewer open | The run behaves identically (L1) |
| Compare a child count against the store | They agree, or the disagreement is a finding |

The dying-run test is the one people skip and it is the one that has failed
every time it was first run.

## PR Checkpoint

**STOP. This phase is complete. Create the PR now.**

**Title**: `live: watch work in flight, not just finished rows — Phase 4`

**Files to include**:
- The live reader module
- The `live` view in the browser
- The `stop` command
- Any runner change adding a marker or salvage path
- Tests for the reader (no terminal required)

---

## Workflow Complete

You now have the three roles in one surface:

| Role | Delivered by |
|---|---|
| **Control center** | Scoped views, a known denominator, static/live divider, a teardown that never deletes results |
| **Debugging surface** | Suspicious rows, one-keystroke detail, reasons where verdicts need them, a depth-accurate legend |
| **Anomaly detector** | The whole set at once, the noise floor rendered, callouts named after what they already ruined |

Maintain it through `references/design-laws.md` — specifically **L16**: every
future defect lands in two places, the code and the screen. That is what makes
the instrument compound rather than merely grow.

And keep **L17** in front of you. The screen produces suspicion. A shell
produces evidence. Roughly a third of what a new screen flags will be in the
screen, and fixing that is not wasted time — it means the instrument was lying,
and every reading taken through it was suspect.
