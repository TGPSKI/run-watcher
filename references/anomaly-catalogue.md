---
name: run-watcher-anomaly-catalogue
description: "Shape on screen → likely cause → the cheapest check that confirms or kills it."
metadata:
  author: TGPSKI
  version: "1.0"
parent: run-watcher
---

# Anomaly Catalogue

What a wrong shape usually means, and the cheapest check that settles it.

Use this two ways: as a **triage table** when something looks off, and as a
**design input** in Phase 1 — every row here is a shape some screen was
eventually built to make visible, so the columns you need are largely already
listed below.

**Read L17 first.** Every entry is a hypothesis, not a verdict. The screen
produces suspicion; a shell produces evidence.

---

## Liveness and progress

| Shape on screen | Likely cause | Cheapest check |
|---|---|---|
| 0% for minutes while the box is loaded | Results written only on completion — there is no in-flight view. **This is the Phase 4 trigger** | `ps` for workers; `ls -la` the working dirs for growing files |
| Rows claim to run; the machine is idle | Liveness inferred from mtime (L3) | `ps -eo pid,args \| grep <worker>` — compare to the row count |
| A worker shown stalled that is fine | Watching the wrong file. Journaled stores write to a sidecar (`-wal`, `.journal`, `.tmp`) | `ls -la --time-style=full-iso <store>*` — compare mtimes across siblings |
| Worker count below the concurrency limit, no error | A worker died silently, or the pool never scaled up | Count process-group members; check for zombies separately |
| Orphaned workers surviving between batches | Kills sent to a pid, not a process group | `ps -eo pid,pgid,args`; look for pgids with no live leader |
| Progress bar moves, then jumps backward | Two runs writing the same output root | Check the run id on each row; scope the viewer to one run |

---

## Arrival patterns

| Shape on screen | Likely cause | Cheapest check |
|---|---|---|
| Completions land in perfect submission order on N workers | Results buffered by index — `pool.map` rather than `as_completed` | Compare each row's *finished* timestamp to its position |
| Results arrive in bursts of exactly N | A barrier between stages that does not need to be there | Look for a collect-all between stages |
| Nothing for a long gap, then everything | Buffered writer never flushed; or an all-or-nothing final write | `ls -la` the output during the gap — is it growing? |

---

## Verdicts that are too good

| Shape on screen | Likely cause | Cheapest check |
|---|---|---|
| A cell at 100%, or every trial passing | **The grader is not grading.** A check that reports "skip" leaves the verdict resting on whatever remains | Print the per-check breakdown for one passing row. Ask what the verdict reduces to when the skips are removed |
| One task easy for every arm, in a set that discriminates | The task lacks the surface the grader compares | Confirm the grader's input exists for that task at all |
| Pass rate outside the useful band in most cells | Ceiling or floor effects — the instrument cannot discriminate | Pooled pass rate per scenario; count how many land in band |

The first row is the most dangerous shape in this document. It is not an
error, does not crash, and produces no warning. It is a **good result that is
wrong**, and the only thing that catches it is a human noticing that a column
looks too easy.

**Design rule that follows**: a check that cannot run must return
`ungradeable`, never `skip` — and a unit nothing can grade must never be able
to yield a pass.

---

## Costs and counts

| Shape on screen | Likely cause | Cheapest check |
|---|---|---|
| A cell that looks cheap and also has give-ups | Give-ups drag the median down. Cheap because it did nothing (L16) | Median cost excluding abandons vs. including |
| A count that is always exactly 0 | The counter reads a source that structurally cannot contain it — e.g. a parent stream carries none of a child's calls | Count the same thing from a second source. A 0 that never moves is a wiring bug, not a measurement |
| A timed-out unit recorded as zero work | The kill discarded buffered evidence | Size the working dir before teardown. Megabytes discarded means salvage is needed |
| p90 far past the median, median falling | A tail that is not saving anything; the average improved by abandoning the hard cases | Plot the distribution, not the summary |
| Two sources disagree about the same quantity | One is authoritative by construction, the other is self-reported | Name the authoritative one *once*, in writing, and gate on the disagreement |

---

## Prompt, input, and delivery

| Shape on screen | Likely cause | Cheapest check |
|---|---|---|
| 1 call, 0 tool uses, completed cleanly | **The input never asked for anything.** The classic symptom is a reply like *"What would you like me to do with this?"* | Read one transcript end to end. Not a summary — the actual text |
| A unit that believes it is single-turn but shows navigation calls | An oversized payload silently switched delivery mode (L11 `PAGINATED`) | grep the run log for the mode-switch line |
| Calls that belong to no stage | Something fires silently — summarization, retries, a background refresh | Total calls minus attributed calls; if nonzero, it has a cause |
| Every unit in one condition near zero, despite correct targeting | The scoring boundary demands something never supplied — e.g. exact internal names | Check whether a correct-but-differently-named solution could pass at all |
| A whole condition delivering the wrong input | A default silently substituted when a path was unset | Assert the input's hash on every row; refuse to start if it is the fallback |

---

## Distributed, non-crashing losses

The hardest class, and the one a live column is uniquely good at.

| Shape on screen | Likely cause | Cheapest check |
|---|---|---|
| A small, **reproducible** fraction of rows producing nothing | Not noise — noise is not reproducible. Something structural drops them | Diff a lost row against a kept one. Look for what the lost path does not see |
| The same fraction across every repeat of one condition | A mechanism, and it is in that condition's structure | Instrument the boundary that condition crosses and nothing else does |
| Answers present in the log but absent from the results | **Attribution, not generation.** The work happened and was filed under the wrong key | grep the evidence log for a lost row's answer. Finding it means the scorer's key is wrong, not the model |

That last row is worth the whole catalogue. A stage that never sees the
original input will transcribe an identifier from its own prior output and
corrupt it. The answer is right, sitting under a ghost id, scored as silence.
Everything looks like a model failure and none of it is.

**Design rule that follows**: attribute on the id you *dispatched*, never on
the id the worker *echoed*.

---

## The instrument itself

Check these before concluding anything about the job.

| Shape on screen | Likely cause | Cheapest check |
|---|---|---|
| A gate failing at an implausible rate | The comparison is running on truncated or differently-scoped inputs | Compare one pair by hand, end to end |
| Two surfaces showing different numbers for one quantity | Two aggregation paths (L10) | Diff the loaders. There should only be one |
| A number that changes when you only changed the sort | Aggregation happening inside the render loop | Move all computation out of render |
| A view empty that should not be | Absent vs. empty vs. filtered conflated (L5) | Clear the filter first, always |
| A finished unit still showing work remaining | A projection divided by *items seen so far* rather than *items that exist* | Check the arithmetic against one complete row: `done` + `left` should equal the configured total, and it will not |
| A sort that disagrees with the column it names | The sort key and the displayed value were computed separately | Sort by the column, read the top row, confirm it is actually the extreme |

**The "seen so far" denominator deserves its own note**, because it is the
instrument bug most likely to survive review: the projection is *plausible at
every moment* and *correct at the end*. Anything divided by a set that grows
during the run — cells that have reported, workers that have checked in,
scenarios that have started — is wrong for the whole middle of the run and
converges only when nobody needs it any more.

Measured instance: `per_cell = expect / len(cells)` read 120/18 = 7 at 86/120,
so every finished 5-trial cell reported 2 trials left and an eta attached to
work already done. The fix was not better arithmetic — it was reading `--trials`
from the batch's own recorded argv, because the runner had already written down
the answer (L10).

**Design rule that follows**: a denominator must come from configuration or
from the job's own record, never from a count of what has been observed. If you
must infer, infer from the fullest unit seen — that can read low, but it never
invents remaining work.

Roughly a third of what a new screen flags will be in the screen. Fix it —
an instrument you have learned to discount detects nothing.
