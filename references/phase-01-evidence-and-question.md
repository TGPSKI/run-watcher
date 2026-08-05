---
name: run-watcher-phase-01
description: "Phase 1: Evidence and question -- inventory what the job already leaves on disk, name the one question the screen settles, and name the wrong shapes. Produces a plan, no renderer."
metadata:
  author: TGPSKI
  version: "1.0"
parent: run-watcher
---

# Phase 1: Evidence and Question

Inventory the evidence, name the question, name the wrong shapes. **This phase
writes a plan and no renderer.**

**Typical PR**: 1
**Files produced**: `docs/WATCHING.md`

Do not generate a screen while the question it answers is unresolved. A screen
built before the question is a screen organized around whatever was easiest to
read, which is how you end up ranking by arrival order.

## Prerequisites

- A job that runs and leaves output. Ideally one running now, or a recent
  output directory still on disk.
- The repo's docs convention identified (where `WATCHING.md` belongs).

---

## Step 1: Inventory the evidence

Everything the viewer will ever show must already exist on disk. If it does
not, the answer is **not** to add instrumentation — see Step 4.

**Inspect** — read, do not ask:

```bash
# working directories of in-flight units
ls -dlt ${TMPDIR:-/tmp}/*{out,work,run}* 2>/dev/null | head

# what a single unit leaves behind
find <one-working-dir> -type f -printf '%10s  %p\n' 2>/dev/null | sort -rn | head -20

# finished artifacts
ls -lt results/ runs/ archives/ 2>/dev/null | head

# live processes, and whether they name their working dir on the command line
ps -eo pid,pgid,etimes,args | grep -v grep | grep <worker-name>

# journaled stores hide their live writes in a sidecar
ls -la --time-style=full-iso <store>* 2>/dev/null
```

| Status | Action |
|--------|--------|
| Workers name their working dir in `argv` | **Best case.** Liveness is a `ps` read with no heuristic (L3). Record this |
| Workers do not name it, but a marker file carries a pid | Second best. Record the marker path and its schema |
| Neither | Record it as a gap. Phase 4 will need one added **to the runner**, not to the viewer |
| A store exists with a `-wal` / `.journal` sidecar | Record it. The main file's mtime lies between checkpoints (L3) |
| Streams are NDJSON | Record the nesting path to the fields you need — these are usually buried |

**Generate** — the evidence table, into the plan. For a **job**:

```markdown
## Evidence on disk

| Source | Path pattern | Written when | Carries |
|---|---|---|---|
| stream | `<out>/stdout.txt` | continuously, appended | calls, tool uses, tokens |
| store | `<out>/*.db` + `-wal` | continuously, WAL | child sessions, per-call tokens |
| marker | `<out>/.run` | at start | run id, unit id, condition, pid |
| result | `runs/<run>.jsonl` | on completion only | verdict, cost, provenance |
| process table | `ps` | live | liveness, process group |
```

For a **continuous system**, the same table with different rows — and note that
"written when" now describes a *cadence*, not a lifecycle:

```markdown
| Source | Path pattern | Written when | Carries |
|---|---|---|---|
| raw log | `/var/log/nginx/access.log*` | continuously, appended + rotated | per-request path, status, bytes, UA |
| hourly rollup | `stats/hourly.csv` | every hour by cron | requests, bytes, unique IPs per bucket |
| daily rollup | `stats/daily.csv` | nightly | the same, coarser — the trend axis |
| history store | `stats/rolling.json` | per collection | prior snapshots, for "is this normal" |
| collector run | its own log or a stamp file | per collection | **whether the pipeline is alive** |
```

The **"written when"** column is the load-bearing one either way, but it
answers a different question in each case:

| Kind | What the column decides |
|---|---|
| Job | Whether a live view is needed. Anything written only on completion cannot show work in flight — the entire argument for Phase 4 |
| Continuous | What your resolution actually is, and how stale "now" is allowed to look. A screen fed by an hourly rollup cannot answer a question about the last ten minutes, and must not appear to |

**The last row is the one people forget.** In a continuous system the collector
is a component, and a flat line means either "nothing happened" or "the
collector died" — indistinguishable without evidence about the collector
itself. Record where that lives, or Phase 2 cannot render the difference (L3).

**Read exported aggregates, never the live path.** If the only way to see
something is to query the production database or tail the socket, that is a
reason to *export* it on a cadence, not a reason for the viewer to reach in.
A viewer that becomes load the system did not ask for has violated L1 as surely
as one that sends a signal.

---

## Step 2: Name the question

**Decide** — this cannot be derived from disk. Ask:

1. "When you glance at this screen, what are you actually asking?"

For a **job**:

| Answer shape | The screen ranks by | Section priority |
|---|---|---|
| "Which condition is winning" | The outcome metric, descending | Summary first |
| "Is anything stuck" | Idle time, descending | Running first |
| "Is it going to finish in time" | Remaining work and rate | Progress first |
| "Did the last thing I changed help" | Newest results first | Graded first |

For a **continuous system**, where nothing finishes and the question is *is
this normal*:

| Answer shape | The screen ranks by | Section priority |
|---|---|---|
| "Is anything unusual right now" | Deviation from the band, descending | Current bucket first |
| "What is the shape of today" | Time, ascending — a chart, not a table | Chart first |
| "Who is doing this" | Volume per source, descending | Top-N first |
| "Is the pipeline alive" | Age of the newest record | A single line, always visible |

**Never** answer "what happened most recently." That is a log, and a log is
what you already have (L6).

**And for a continuous system, never answer it with a bare current value.**
"1,204 requests" is not an answer to *is this normal*; "1,204, and the band for
this hour on a Tuesday is 900–1,400" is. If you cannot state the band yet,
record it as `UNKNOWN` in Step 3 rather than shipping a number that invites a
conclusion it cannot support.

---

## Step 3: Name the wrong shapes

This is the anomaly-detector payload and the part only the operator can supply.

**Inspect**: read `references/anomaly-catalogue.md` and check which rows apply
to this job's architecture.

**Decide** — ask, and push for specifics:

1. "What has silently changed an experiment here before?"
2. "What would have to be true on screen for you to stop the run?"
3. "What number, if it were ever exactly zero, would mean a wiring bug rather
   than a measurement?"
4. "Is there a known noise floor for the outcome metric?" (L9)

| Answer | Becomes |
|--------|---------|
| A silent behaviour change | A named callout, `x` glyph (L11) |
| A quantity that can be structurally unreadable | A column that shows `0` and a check that it ever moves |
| A stop condition | A callout plus a rule in the plan for what to do |
| A noise floor | `NULL_BAND`, rendered dim below it (L9) |
| "I don't know yet" | Record as unknown. Do **not** invent a threshold |

**Generate** — into the plan:

```markdown
## Callouts

| Name | Glyph | Means | The experiment it already changed |
|---|---|---|---|
| STALE | `!` | artifacts persist after a run ends, so counting them reports progress for a run that is not running | the 12-hour battery that reported "idle" throughout |

## Noise floor

NULL_BAND = <value> <units>   # source of the estimate, or UNKNOWN
```

A callout with an empty last column is a guess. Either find the incident or
mark it speculative — the header comment is what stops the rule being deleted
by someone who does not know what it cost.

---

## Step 4: Confirm the pattern is not inverted

**Inspect** the gaps from Step 1.

| Status | Action |
|--------|--------|
| Every column in the plan is derivable from existing evidence | Proceed |
| A column needs a field the job does not currently emit | **Stop.** Decide explicitly |
| Most columns need new instrumentation | **Decline.** The observer would be changing the observed; you would maintain two systems |

**Decide** when a field is genuinely missing:

1. "This needs `<field>`, which nothing writes today. Add it to the **runner**
   as part of the run's own record, or drop the column?"

Adding a field to the runner because the *run* should record it is fine — it
becomes evidence, and it is there whether or not anyone watches. Adding a field
*for the viewer* is the inversion this workflow refuses.

---

## Step 5: Decide the stopping point

**Decide**:

1. "Do results land fast enough that a table of finished rows is useful on its
   own?"

| Answer | Route |
|--------|-------|
| Yes, and one ranked table answers the question | Phases: **1 → 2**, then stop |
| Yes, but you need several views | **1 → 2 → 3** |
| No — the screen sits empty while workers are busy | **1 → 2 → 3 → 4** |

Record the decision. Phase 2 is frequently the whole answer, and a 273-line
bash watcher has caught three experiment-changing defects on its own.

---

## Validate

```bash
# every path in the evidence table must resolve against a real directory
# every callout must name a real, observable condition
```

Walk the plan against one real output directory. A path that does not resolve
is a column that will render blank forever.

## PR Checkpoint

**STOP. This phase is complete. Create the PR now. Do NOT proceed to Phase 2.**

**Title**: `docs: what to watch while <job> runs — Phase 1`

**Files to include**:
- `docs/WATCHING.md`
- Any runner change that adds a genuinely missing evidence field (Step 4)

**After creating the PR**, inform the user:

> Plan created. After it is merged, start a new session and invoke the
> workflow — it will detect the plan and route to Phase 2.

---

## Next Phase

`references/phase-02-watcher.md`

**Carry forward**: the evidence table, the ranking axis, the callout list with
glyphs, `NULL_BAND`, the liveness signal chosen in Step 1, and the stopping
point from Step 5.
