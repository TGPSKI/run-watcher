# Watching the adherence battery

Phase 1 plan, reconstructed from the live view built for
[adherence-suite](https://github.com/TGPSKI/adherence-suite) on 2026-08-03/04.
Golden example: validates under `watchctl plan`.

The job: a deterministic agent-adherence benchmark. Six arms, thirty-four tasks
derived from real merged PRs, no model anywhere in the scoring path. Three
concurrent workers, 900-second timeouts, 120 trials per battery.

## Evidence on disk

| Source | Path pattern | Written when | Carries |
|---|---|---|---|
| stream | `/tmp/adh-out-*/stdout.txt` | continuously, appended | NDJSON: calls, tool uses, tokens |
| store | `/tmp/adh-out-*/*.db` **+ `-wal`** | continuously, WAL mode | child sessions, per-call tokens |
| marker | `/tmp/adh-out-*/.run` | at start | run id, scenario, arm, trial, pid |
| process table | `ps` | live | liveness, process group |
| result | `runs/probe.jsonl` | **on completion only** | verdict, cost, provenance |
| proxy log | recording proxy | continuously | authoritative token and round-trip counts |

**The decisive row is `result`.** It is written only after a trial finishes and
is graded, so at 900-second timeouts and three workers the screen sits at 0%
for minutes while the box is fully loaded. That gap is the entire argument for
Phase 4, and this table is where it became visible.

**The trap is `store`.** SQLite runs in WAL mode, so live writes land in
`*.db-wal` and the main file's mtime only moves at checkpoints. Freshness is
`max(mtime)` across the siblings. Measured: 282s stale against a 29s-fresh WAL,
which showed a working delegation as hung.

**Liveness signal**: the runner passes the output directory to the worker as a
command-line argument, so it sits in `ps` for the whole life of the trial. That
is a direct signal with no heuristic in it, and it replaced a file-age
inference that reported killed runs as live for three more minutes.

## The question

Ranking axis differs by section, because this screen answers three questions at
once and they have different orders:

| Section | Ranked by | Because |
|---|---|---|
| running | elapsed, descending | "is anything stuck" |
| summary | scenario tag | "which cell is discriminating" — a stable order you can scan |
| graded | finished time, newest first | "did the last thing I changed help" |

## Scoping

Scoped to **one run**. Without it the loader globs every `*.jsonl` in `runs/`
and renders the live battery pooled with every smoke test that ever landed
there — numbers belonging to no single experiment.

The results-side filter deliberately does **not** apply to the running section.

## Callouts

| Name | Glyph | Means | Already changed |
|---|---|---|---|
| stalled | count | live but the stream has not moved in 180s | a delegation shown as hung was fine; the viewer was watching `db` instead of `db-wal` |
| ungradeable | `ung` column | a check could not run — harness fault, never a model failure | adapter faults were scoring as `fail` until this became a distinct verdict |
| abandoned | `abnd`, red | the unit gave up; never passes, and drags the median cost down | a cheap-looking cell with abandons is not cheap |
| p90 divergence | red | the tail has run away from the median | a median that halves while the tail doubles is not a saving |
| subagent gap | `sub` column | child calls the parent stream structurally cannot see | it read 26 where 66 had happened |

## Noise floor

```
NULL_BAND = UNKNOWN
```

Never measured on this rig. The primary outcome is reported as a paired
geometric mean with a bootstrap CI rather than against a single-run band, so
the screen renders no significance colouring at all — deliberately, rather than
inventing a threshold.

## Sections and space

| Section | Bounded by | Space |
|---|---|---|
| running | the concurrency limit (3) | whole |
| summary | the cell count | whole |
| graded | nothing — grows all run | scrolls, with a scrollbar and `N–M of T` |

Proportional sharing gave graded 22 rows it did not need while truncating a
3-row running table, and on a 24-line terminal pushed graded off the screen
entirely — absent, not truncated, with nothing saying so.

## Stopping point

**Phases 1 → 2 → 3 → 4.** The full path, because results are written only on
completion and the screen was observed sitting empty during real work.

## What this plan got wrong

The battery ran for hours with two tasks reporting 5/5 PASS. Not an error, not
a crash, no warning — a *good result that was wrong*: neither grader could
judge those tasks, so the verdict had degraded to "touched the right file,"
which an agent passes by adding a comment.

Nothing in the evidence table above could have caught it. The results file said
`pass`. It was caught by a human noticing that a column looked too easy, which
is the anomaly-detector role doing the one thing no assertion in this plan was
ever going to do.

The catalogue entry that generalizes it: **a cell at 100% in a set that
otherwise discriminates means the grader is not grading.**
