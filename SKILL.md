---
name: run-watcher
description: "Build a live-render TUI that is the control center, debugging surface, and anomaly detector for a long-running concurrent job. Routes across four phases: evidence inventory, the watcher, the browser, the live view."
metadata:
  author: TGPSKI
  version: "1.0"
compatibility: "POSIX shell, Python 3 (stdlib only). No dependencies, no framework, no web stack."
---

# Run Watcher

Build the instrument before you trust the results. This workflow produces a
read-only terminal renderer for a long-running job — one focused PR per phase.

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

- **A job that actually runs.** This workflow reads real output directories. A
  watcher validated against nothing is unvalidated.
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
ls -d ${TMPDIR:-/tmp}/*out* 2>/dev/null | head        # working directories
ls results/ runs/ archives/ 2>/dev/null | head        # finished artifacts
ps -eo pid,etimes,args | grep -v grep | head -20      # what is running now
```

**Ask**: "Is this the job you want to watch?" — confirm the output root and the
name of the thing being run.

## Is this job worth a watcher?

**Decline** when any two of these hold. Say so plainly and stop.

| Signal | Meaning |
|---|---|
| Finishes in under ~2 minutes | Just run it again |
| No concurrency | The interesting facts are about the set; there is no set |
| Leaves no intermediate evidence on disk | You would have to instrument the job — inverts the pattern |
| Will run once | The instrument never amortizes |

The clearest signal you *do* need one: **the user has typed some variant of
`watch 'ls results/ | wc -l'` more than twice.**

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
| Second experiment in a repo that already has a watcher | 1 → 2, obeying `design-laws.md`; do not reuse the code |
| Watcher exists, is wrong, and misled someone | `references/design-laws.md`, then re-enter at the failing phase |

A watcher does **not** port between experiments as code. It ports as rules.
The best evidence for this pattern in the wild is a 128-line re-application
written fresh the day after a 273-line original, obeying the same laws.

## Phase Files

| Phase | File | Description | PR |
|-------|------|-------------|-----|
| 1 | `references/phase-01-evidence-and-question.md` | Inventory the on-disk evidence; name the question and the wrong shapes | 1 |
| 2 | `references/phase-02-watcher.md` | The watcher: in-place redraw, ranked, named callouts | 1 |
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
