---
name: run-watcher-design-laws
description: "The laws every phase obeys, each stated with the concrete failure it exists to prevent."
metadata:
  author: TGPSKI
  version: "1.0"
parent: run-watcher
---

# Design Laws

Every law here is a law because violating it cost something real. The failure
is stated with each one — a rule without its failure gets rationalized away at
the first inconvenience.

Cite these by number in generated code comments. A comment that says *why* a
line exists survives refactoring; a comment that says what it does does not.

---

## L1 — The observer must never be able to affect the observed

Absolute. Everything else on this list is negotiable.

**The failure**: A liveness probe used `os.kill(pid, 0)` — the standard POSIX
existence check, where signal 0 means "check, do not send." On Windows,
`signal.CTRL_C_EVENT == 0`. The same call delivers a real Ctrl-C to the
target's console group. A selftest wrote a marker carrying its own pid, the
viewer checked it, and *the check* raised `KeyboardInterrupt` inside an
unrelated subprocess several tests later — sixteen seconds into CI, with a
traceback pointing at code that had nothing to do with it.

**The rule**: When a check cannot be made safe on a platform, it declines to
answer and the caller falls back to evidence that cannot misfire. A monitor
that can kill what it monitors is not a monitor.

**Applies to**: signals, locks, opening files with write intent, anything that
mutates atime where the job reads atime, and any cleanup routine the renderer
can reach.

---

## L2 — Tolerate partial writes; never raise on them

**The failure**: Every file the viewer reads is being written concurrently by
a process that does not know it is being watched. A half-written final JSON
line is *normal*, not exceptional.

**The rule**: Unreadable file → skip. Torn line → skip. Missing key → default.
Nothing raises. A viewer that can crash a six-hour job is worse than no viewer;
a viewer that crashes *itself* is worse than one showing a stale row, because
you will stop launching it and then you are flying blind by habit.

---

## L3 — Liveness is probed, not inferred

**The failures** — three variants, three different projects, same root:

| Inference | What it produced |
|---|---|
| Archive end time from file mtime | A `git stash` rewrote every mtime; all archives reported the same end time and 10-hour durations |
| "The stream file moved recently" = running | Four rows claiming to run for three minutes after teardown, while `ps` showed nothing |
| "The database file hasn't moved" = stalled | SQLite in WAL mode writes to `db-wal`. Main file 282s stale, WAL 29s fresh. A working delegation shown as hung |

**The rule**: Prefer a direct signal. If the runner passes the output directory
to the worker as an argument, it sits in `ps` for the life of the trial — that
is liveness with no heuristic in it. Where you must read a file, read the last
timestamp *inside* it, not the filesystem's metadata about it.

**Filesystem metadata is not evidence about your program.** It is evidence
about the filesystem, and something else is always touching it.

---

## L4 — Show stalled; never drop it

**The failure**: A hung worker filtered out of the display is invisible in
exactly the situation you built the display for.

**The rule**: Two thresholds, far apart. Past the first (minutes), render
**stalled** — loudly, in place. Past the second (tens of minutes), treat it as
debris from a killed run. Conflating them hides hangs behind cleanup.

---

## L5 — Absent, empty, and filtered are three different states

**The failure**: "Nothing on screen" has several causes needing different
responses. Blaming the filter when the run simply has not written a result yet
sends the operator hunting for a filter they never set. An empty progress bar
rendered `................` reads as truncated output, not as 0%.

**The rule**: Distinct glyphs for distinct absences, and say which one it is.

| Glyph | Means |
|---|---|
| `-` | Measured, and the value is zero |
| `?` | Not recorded — an archive predating the field |
| `~` | Inferred, not read. A guess must never be mistaken for a record |
| blank | Not yet computed |

Bracket progress bars, and pick a track character that cannot be read as an
ellipsis.

**And**: a results-side filter must not apply to the running table. Silently
hiding a running job is the worst behaviour a monitor has available to it.

---

## L6 — Rank by the question, not by arrival

**The failure**: Run order buries the answer. The question asked of the screen
is always *"which one is winning"* or *"which one is stuck"* — never *"what
happened most recently."*

**The rule**: Sort by the axis the screen exists to settle. Put the thing that
changes second-to-second where the eye lands first; finished rows that will
never move again go below it.

---

## L7 — The cursor anchors to identity, not position

**The failure**: Rows finish and new rows start every tick. A positional cursor
points at a different object each time the list changes length — worst in a
detail pane, which swaps out from under you mid-read.

**The rule**: Anchor to something unique that outlives the row (an output
directory path, a run id). Re-find it each tick. When the anchored row genuinely
ends, stay in range and re-anchor *explicitly*, rather than silently tracking
whatever slid into the slot.

---

## L8 — Allocate space by priority, not proportion

**The failure**: Proportional sharing gave an unbounded section 22 rows it did
not need while truncating a 3-row bounded one — and on a 24-line terminal
pushed a whole section off the screen. Absent, not truncated, with nothing
saying so.

**The rule**: Classify each section as bounded or unbounded. Bounded sections
(one row per worker, one per cell) render whole. The unbounded section absorbs
the scrolling, and gets a scrollbar plus an `N–M of T` indicator — without them
the cursor can sit on a row nobody can see.

---

## L9 — Render the noise floor

**The failure**: A difference smaller than the measurement noise looks exactly
like a finding.

**The rule**: One constant.

```python
NULL_BAND = 2.4          # measured single-run noise floor, in points
```

Below it, render dim. Above it, warn. Above twice it, alarm. A verdict is only
`RESOLVED` when it is both significant *and* outside the band.

This is the cheapest statistical honesty available and it operates at the
moment of perception, before motivated reasoning gets a turn.

---

## L10 — One data path, so no two surfaces can disagree

**The failure**: A live view with its own aggregation eventually disagrees with
the report. You then chase a rendering bug *as though it were a finding*, which
is the most expensive kind of wasted afternoon because it feels like science.

**The rule**: Every number flows through the loader the report uses. If an
analysis has a registered implementation (a statistical test, a scorer), import
that one rather than writing a second that can drift.

---

## L11 — Name your callouts after the experiment they already ruined

**The failure**: Silent, non-crashing behaviour changes what is being measured.
A run that paginates when its arm believes it is single-turn is measuring a
different delivery mechanism than its name claims. Nothing errors.

**The rule**: Each callout gets a short uppercase name, a one-line meaning, and
a header comment recording the experiment it already changed. Encode severity
in a **glyph** (`x` critical, `!` advisory, `~` informational) so it survives
`NOCOLOR=1` piping into a paste.

Examples from the field:

| Callout | Meaning |
|---|---|
| `PAGINATED` | An oversized payload silently switched delivery mode |
| `UNATTRIB` | Calls the proxy could not attribute to a stage — summarization fires silently, so unexplained calls are its only trace |
| `STALE` | Artifacts persist after a run ends, so counting them unconditionally reports progress for a run that is not running |
| `TRUNC` | Hit the token ceiling mid-reasoning — the cell measured the budget, not the model |

---

## L12 — Show the field that explains the verdict, on the rows whose verdict needs explaining

**The failure**: An `ungradeable` row rendered with `—` in its reason column is
the single worst cell in a table, because it is blank precisely where the
operator is asking *why*.

**The rule**: The reason column resolves differently per verdict. This is not
"add more columns" — it is making one column context-sensitive.

---

## L13 — The legend must be depth-accurate

**The failure**: A footer advertising `[space] detail` on a view with no detail
pane. The key is tried once, does nothing, and is never trusted again.

**The rule**: Advertise only what the current level responds to. Same for sort
keys: the sort shown must be the one the sort key would change *here*.

---

## L14 — Keep reading and spending in different rooms

**The failure**: A viewer one typo away from launching a battery.

**The rule**: Everything that spends real resources lives behind a separate
entry point and an explicit flag. Reaching for a viewer must never be able to
start a run.

---

## L15 — Control means stop, not edit

**The failure**: The one destructive capability an operator reaches for at 2am
is the one that must not exist.

**The rule**: The teardown kills process groups, reclaims strays, and sweeps
temp. It **deliberately never deletes results.** Removing debris and removing
evidence are different operations and only one of them belongs on a hotkey.

---

## L16 — Encode each fix as a display rule

This is what compounds, and it is the reason the instrument gets better rather
than merely bigger.

**The rule**: When a defect is found, the fix lands in two places — the code,
and the screen.

- Abandoned trials never pass *and* drag the median cost down, so a
  cheap-looking cell containing abandons is not cheap → abandons render red on
  every surface showing a cost.
- A p90 far past the median is a tail that is not saving anything → p90 goes
  red when it diverges.
- An arm that paginated when it believed it was single-turn → a red
  `REFLECTION MODE` line.

The display becomes an accumulated regression suite you read rather than run,
and it keeps catching the *next* instance of a class you have already paid for
once.

---

## L17 — The screen produces suspicion, not evidence

**The failure**: A column made a rare, distributed, non-crashing anomaly
visible and reproducible — 25/26/28/30/34 rows across five draws, ~11%, every
time. The shape was read correctly. The *cause* was assigned wrongly, and went
into a published figure. Forensics later found a bookkeeping defect
underneath — answers were landing under a corrupted id and being scored as
silence — which moved the headline number.

**The rule**: The screen's job ends at *"that looks wrong."* Confirmation is a
separate job, done in a shell, against the archives. Part of what presents as
"the treatment is expensive" is always, potentially, *your instrument for
measuring the treatment having a bug.*

That is not an argument against instruments. It is the discipline that has to
come with one.
