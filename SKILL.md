---
name: run-watcher
description: "Build a live-render TUI that is the control center, debugging surface, and anomaly detector — for a long-running job, or for a continuous system like logs or traffic where the question is not 'did it work' but 'is this normal'. Routes across four phases: evidence inventory, the watcher, the browser, the live view."
metadata:
  author: TGPSKI
  version: "1.0"
compatibility: "POSIX shell, Python 3 (stdlib only). No dependencies, no framework, no web stack."
---

# Run Watcher

Build the instrument before you trust the results. This workflow produces a
read-only terminal renderer for something you have to watch — a long-running
job, or a system that never stops. One focused PR per phase.

The claim it is built on:

> Most defects in a concurrent long-running system are visible as **shape**
> long before they are visible as **values** — and a live render is the only
> instrument that shows shape.

A results file says `pass`. It cannot say *"this column looks too easy."*

## Design Principles

- **Ownership**: The person who will actually watch the run builds it. The
  screen encodes *their* eye for the job — what counts as a wrong shape is
  domain knowledge, and it cannot be delegated to someone who will never sit
  in front of it.
- **Source of truth**: The job's own on-disk evidence — streams, stores, marker
  files, the process table. Read what already exists. **Never add
  instrumentation to the observed system to feed the viewer**; that inverts the
  pattern and leaves two systems to maintain.
- **Safety over completeness**: The observer must never be able to affect the
  observed. A viewer that can crash the run — or crash itself on a half-written
  line — is worse than no viewer, because you will stop launching it.
- **Ask less, infer more**: Derive the evidence inventory by reading the output
  directory and the process table. Only ask for what disk cannot answer: which
  question the screen exists to settle, and which shapes are wrong.
- **Prefer simple**: Phase 2 alone is frequently the whole answer. Do not build
  a curses browser until you have named a view the watcher cannot show.

## Prerequisites

- **Something that actually runs.** This workflow reads real output — a job's
  working directories, or a continuous system's logs and rollups. A watcher
  validated against nothing is unvalidated.
- **Start from an up-to-date primary branch.** Progress detection reads files
  merged to the primary branch — not local commits, uncommitted changes, or
  prior session output.
- **Start with a clean chat session.** Prior conversation causes the agent to
  conflate previous output with actual repository state.

## State Machine

Each phase produces one PR. The merge advances the state machine.

1. **Detect** — read the filesystem to determine the current phase
2. **Execute** — Inspect → Decide → Generate
3. **Validate** — run the artifact against a real job for one session
4. **Checkpoint** — STOP. Commit, push, create PR.
5. **Merge** — reviewed and merged, outside this workflow
6. **Resume** — new session, invoke the router again

Step 3 is not optional and is not a formality. **A watcher that has never
watched anything is a hypothesis.** Roughly a third of what a new screen flags
is wrong *in the screen*, and every reading taken through a lying instrument is
suspect.

## Entry Point

**Inspect** the job before asking anything:

```bash
# a job: working directories, finished artifacts, live processes
ls -d ${TMPDIR:-/tmp}/*out* 2>/dev/null | head
ls results/ runs/ archives/ 2>/dev/null | head
ps -eo pid,etimes,args | grep -v grep | head -20

# a continuous system: raw sources, rollups, and whether the collector is alive
ls -lt /var/log/*/ stats/ 2>/dev/null | head -20
ls -lt *.csv *.json 2>/dev/null | head          # exported aggregates
crontab -l 2>/dev/null | grep -iE 'fetch|collect|analy'
```

**Ask**: "Is this what you want to watch?" — confirm the output root, the name
of the thing, and **which of the two kinds it is** (below).

## What kind of system is this?

The question the screen answers differs, and it changes what Phase 1 asks for.

| Kind | The question | Examples |
|---|---|---|
| **A job** — starts, runs, ends | *"Did it work, and is it going to finish?"* | A benchmark grid, a migration, a batch, a build matrix |
| **A continuous system** — never ends | *"Is this normal?"* | Access logs, traffic, queue depth, a service's own metrics |

Both are in scope; the pattern's oldest instances are the second kind. Where
they differ:

| | Job | Continuous |
|---|---|---|
| Progress | against a known denominator | there isn't one — trend instead |
| A number out of range | usually a defect | usually just traffic |
| The noise floor (**L9**) | often skippable | **mandatory** — there is a daily and weekly shape, and without the band the screen invents an incident every Monday |
| "Finished" | a state to render | doesn't exist; a flat line means the *collector* died (**L3**) |
| **L1** risk | reading a temp dir | reading production. Read exported aggregates, never the live path |

## Is this worth a watcher?

**Decline** when any two of these hold. Say so plainly and stop.

| Signal | Meaning |
|---|---|
| A job that finishes in under ~2 minutes | Just run it again |
| No concurrency **and** no history | The interesting facts are about a set — over workers, or over time. One value now is a number, not a shape |
| Leaves no evidence anywhere but in the live system | You would have to instrument it, or query production — both invert the pattern |
| A job that will run once | The instrument never amortizes |

A continuous system is rarely declined on the first two: it has history by
definition, and it is by definition going to be looked at again.

The clearest signal you *do* need one: **someone has typed a variant of
`watch 'ls results/ | wc -l'` — or `tail -f access.log | grep` — more than
twice.**

## Detect Current Progress

Detection is based on files on disk from the primary branch. Use Glob and Read.

| Check | What to Look For | Indicates |
|-------|-----------------|-----------|
| Plan | `docs/WATCHING.md` (or the repo's docs convention) | Phase 1 complete |
| Watcher | `**/watch-*.sh` | Phase 2 complete |
| Loader + browser | a shared loader module **and** `**/*-tui.py` | Phase 3 complete |
| Live reader | a module exposing `snapshot()` over in-flight work | Phase 4 complete |

### Determine Phase

| Detected State | Recommended Action |
|----------------|-------------------|
| Nothing found | Start at **Phase 1** |
| Plan exists, no watcher | **Phase 2** |
| Watcher exists, and a needed view is missing from it | **Phase 3** |
| Watcher exists, and no view is missing | **Stop.** Phase 2 was the whole answer |
| Browser exists, job shows 0% while workers are busy | **Phase 4** |
| Browser exists, results land fast enough to watch | **Stop.** Phase 4 buys nothing |
| Live reader exists | Workflow complete — maintain via `references/design-laws.md` |
| Watcher exists but has never been run against a live job | **Validate first**, then re-detect |

## Route to Phase

**Ask**: "You appear to be at [detected phase]. Would you like to:
1. Continue from [detected phase]
2. Start from a different phase
3. Review what already exists"

| User Choice | Action |
|-------------|--------|
| Continue detected phase | Load the corresponding phase file |
| Different phase | Ask which phase, then load that file |
| Review | Summarize the existing viewer surface, then re-ask |

## Variant Paths

| Scenario | Phases |
|----------|--------|
| Standard | 1 → 2 → stop, or 1 → 2 → 3 → 4 |
| Archives only, no in-flight question | 1 → 2 → 3 |
| Second job or system in a repo that already has a watcher | 1 → 2, obeying `design-laws.md`; do not reuse the code |
| Watcher exists, is wrong, and misled someone | `references/design-laws.md`, then re-enter at the failing phase |

A watcher does **not** port between jobs as code. It ports as rules.
The best evidence for this pattern in the wild is a 128-line re-application
written fresh the day after a 273-line original, obeying the same laws.

## Phase Files

| Phase | File | Description | PR |
|-------|------|-------------|-----|
| 1 | `references/phase-01-evidence-and-question.md` | Inventory the on-disk evidence; name the question and the wrong shapes | 1 |
| 2 | `references/phase-02-watcher.md` | The watcher: one screen ranked by the question, shell **or** curses — Step 0 decides | 1 |
| 3 | `references/phase-03-browser.md` | The browser: one shared loader, tabbed views, the noise floor. Copies `assets/tui/` rather than rewriting it | 1 |
| 4 | `references/phase-04-live-view.md` | In-flight rendering, probed liveness, teardown | 1 |
| — | `references/design-laws.md` | The laws every phase obeys, each with the failure it prevents | — |
| — | `references/anomaly-catalogue.md` | Shape → likely cause → cheapest confirming check | — |

Phases run in order. Phase 1 writes only a plan. Do not generate a renderer
while the question it answers is unresolved.

## What NOT to Do

- **Never let the viewer write to, signal, or lock anything the job uses.**
  See the `os.kill(pid, 0)` case in `design-laws.md` — a liveness *check* that
  delivered a real Ctrl-C.
- **Never infer liveness from file mtime.** It has produced a distinct
  false-liveness bug in every generation. Probe it.
- **Never give the viewer its own aggregation code.** One loader, or two
  surfaces will disagree and you will chase a rendering bug as though it were a
  finding.
- **Never treat the screen as evidence.** It produces suspicion. Confirmation
  happens in a shell, against the archives.
- **Never build Phase 3 or 4 speculatively.** Build them when you can name the
  question the current screen cannot answer.
