# Watching the SIG-triage ablation matrix

Phase 1 plan, reconstructed from the watcher that was actually built for this
job in [leather](https://github.com/TGPSKI/leather) on 2026-07-29. This is a
golden example: it validates under `watchctl plan`, and every row below is
derived from a file in that repository rather than invented here.

The job: a pre-registered ablation matrix on Kubernetes issue → SIG triage. 22
classes, a 250-issue gold corpus, arms differing only in harness design, four
campaigns across two rigs.

## Evidence on disk

| Source | Path pattern | Written when | Carries |
|---|---|---|---|
| predictions | `results/runs/<tag>/predictions.jsonl` | on completion only | per-issue prediction, confidence |
| run log | `<session>/run.log` | continuously, appended | tool executions, mode switches, per-agent tokens |
| logprob stream | `<session>/logprobs.jsonl` | continuously, appended | per-call stage attribution |
| artifacts | `<session>/artifacts/{analyze,match}/` | per issue, as produced | progress — count of finished issues |
| manifest | `<session>/run-manifest.json` | at start | run tag, battery, rig |
| evidence log | `results/runs/<tag>/run-evidence.log.gz` | on completion | the authoritative end time |

**The decisive row is `artifacts/`.** Counting files there is the only
in-flight progress signal, and it is also the source of the `STALE` callout —
artifacts persist after a run ends, so counting them unconditionally reports
progress for a run that is not running.

**Not usable for timing**: file mtime anywhere. Any `git checkout` or `stash`
rewrites it, which once had every archive reporting the same end time and
10-hour durations. The end time comes from the last timestamp *inside*
`run-evidence.log.gz`.

## The question

Ranking axis: **accuracy, descending.** The question asked of this screen is
always *"which arm is winning"* — run order buries that, so cells are ranked,
never listed in arrival order.

Secondary sorts, all on the same table: tools, ktok, duration, no-out, cell tag.

## Scoping

Four campaigns share `results/runs/`. Watching all of them at once is what made
the screen unreadable in the first place, so three composable knobs exist:

| Knob | Values | Means |
|---|---|---|
| `RIG` | `4b`, `35b` | one rig; also drops the other's whole section |
| `SCOPE` | `confirmatory`, `exploratory`, `all` | the registered draws vs the earlier atlas |
| `FILTER` | a pattern | prefix · glob · substring · comma-OR · `!` negates |

## Callouts

| Name | Glyph | Means | Already changed |
|---|---|---|---|
| PAGINATED | `x` | an oversized payload put the runtime into reflection mode — paging preamble, tools stripped per turn, N+1 alternating turns | an arm that believed it was single-turn was measuring a different delivery mechanism than its name claimed |
| UNATTRIB | `!` | calls the proxy could not attribute to a stage | context summarization fires silently, so unexplained calls are its only trace |
| STALE | `!` | artifacts persist after a run ends | a 12-hour registered battery reported "idle" for its entire run |
| no-out | column | rows that produced no usable answer | one arm was reproducibly ~11% (25/26/28/30/34 across five draws) — see the reattribution below |

## Noise floor

```
NULL_BAND = 2.4 points
```

Measured from repeat draws of an identical arm. Spreads below it render dim,
above it yellow, above twice it red. A contrast is only `RESOLVED` when it is
both significant *and* outside the band.

## Distinct absences

| Glyph | Means |
|---|---|
| `-` | measured, and zero |
| `?` | an archive written before that field existed |
| `~` | inferred, not read — so a guess is never mistaken for a record |

## Stopping point

**Phases 1 → 2 → 3.** Results land only on completion, but a draw finishes in
minutes rather than hours, so the archive table is useful on its own and Phase
4 was not needed here. Phase 3 was built because "which arm is winning" and
"is the difference real" are different questions and one table cannot answer
both.

## What this plan got wrong

Recorded because a plan that only lists its successes teaches nothing.

The `no-out` column did exactly what Phase 1 designs a column to do — it made a
rare, distributed, non-crashing anomaly visible and reproducible days before
anyone knew what it was. Then the **cause** was assigned wrongly: read as the
arm's mechanism, published as a −15 point arm, and later traced to a
bookkeeping defect in which correct answers were filed under a corrupted id and
scored as silence. The corrected arm is roughly −9.

This is **L17**, and it is why the workflow says the screen produces suspicion
and a shell produces evidence.
