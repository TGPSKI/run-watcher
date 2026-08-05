# The Run Watcher Pattern

> Build a live-render TUI that is the control center, debugging surface, and
> anomaly detector for a long-running job — or for a system that never stops.

A **run watcher** is a read-only terminal renderer that joins the evidence a
running system already leaves on disk and shows it as shape, on a tick, in one
screen. It is not a dashboard, not a log viewer, and not a report. It is an
instrument you keep open while the thing runs.

It applies to two kinds of system, and the pattern's oldest instances are the
second kind:

| Kind | The question | Examples |
|---|---|---|
| **A job** — starts, runs, ends | *"Did it work, and will it finish?"* | A benchmark grid, a migration, a batch |
| **A continuous system** — never ends | *"Is this normal?"* | Access logs, traffic, queue depth, service metrics |

Everything below holds for both. Where they diverge is noted; the sharpest
divergence is **L9**, which a job can often skip and a continuous system
cannot, because traffic has a daily and weekly shape and a screen without a
band manufactures an incident every Monday.

The pattern sits beside [Directed Contexts](https://github.com/TGPSKI/directed-contexts)
and [Directed Workflows](https://github.com/TGPSKI/directed-workflows): where a
directed context encodes a *territory* an agent operates inside and a directed
workflow encodes a *process* an agent walks a user through, a run watcher
encodes an *instrument* the user reads while the process runs.

---

## The claim

> Most defects in a concurrent long-running system are visible as **shape**
> long before they are visible as **values** — and a live render is the only
> instrument that shows shape.

Three workers where you expect four is a shape. A pass column that is all green
is a shape. Completions arriving in perfect submission order on a pool of three
is a shape. None of those appear in a log line, and every one of them is one
glance on a screen.

## What it exists against

| Artifact | Tells you | Cannot tell you |
|---|---|---|
| A log you tail | What happened most recently | Anything about the *set* — a log has no set |
| A results file | The final verdicts | Anything at all, until it is over |

The failure mode both permit is specific and expensive: **you do not discover
that the run was invalid until the run is over.** You burn six hours, read the
output, and find that every unit was handed the wrong input, or that the
timeout discarded the evidence, or that a whole class of work scored 100%
because nothing was grading it.

---

## Core model

```text
                    ┌──────────────────────────────┐
   the job ────────▶│  evidence already on disk    │
   (untouched)      │  streams · stores · markers  │
                    │  the process table           │
                    └──────────────┬───────────────┘
                                   │  read-only, tolerant
                                   ▼
                    ┌──────────────────────────────┐
                    │  reader   pure functions     │  testable without a terminal
                    └──────────────┬───────────────┘
                                   │  records
                                   ▼
                    ┌──────────────────────────────┐
                    │  renderer  data → rows       │  testable without a job
                    └──────────────┬───────────────┘
                                   ▼
                          one screen, on a tick
```

The arrow runs one way. Nothing in the renderer can reach the job, and the
`sweep`/teardown path — the only code that removes anything — is never callable
from the render loop.

## Vocabulary

| Term | Means |
|---|---|
| **Evidence** | Something the job writes for its own reasons, that the viewer happens to read. Not instrumentation added for the viewer |
| **Shape** | A property of the set rather than of a row — a count, an ordering, a distribution, an absence |
| **Callout** | A named, glyph-prefixed warning for a silent behaviour change, documented with the run it already ruined |
| **Noise floor** | The measured single-run variation, rendered so a difference below it cannot look like a finding |
| **Stall** | Live but not moving. Rendered loudly, never filtered out |
| **Debris** | An output directory whose run is long gone. A separate, much larger threshold than a stall |

---

## The three roles

A surface that serves only one of these is a report, a debugger, or an alarm.
The pattern is the conjunction.

### Control center

Where the run is understood and steered.

- Scoped to **one** run. A viewer that globs every result file renders the live
  batch pooled with every smoke test that ever landed there — numbers that
  belong to no single experiment and look authoritative anyway.
- Progress against a **known denominator**, counting only what has landed.
- Live data and static ground truth **visibly separated**, so a stale table
  cannot be read as a fresh one.
- Reading and spending in **different rooms**: anything that consumes real
  resources lives behind a separate entry point and an explicit flag.
- Control means **stop, not edit**. The teardown kills process groups and
  sweeps debris. It never deletes results.

### Debugging surface

Where the hypothesis is formed. The shell is where it is confirmed.

- Every row suspicious on its own; detail one keystroke away.
- The field that explains a verdict shown on exactly the rows whose verdict
  needs explaining.
- A depth-accurate legend — a key that does nothing is tried once and then the
  legend is never trusted again.

### Anomaly detector

The role that pays for the whole thing.

> The screen is a continuously-evaluated assertion set that you read instead of
> run.

Two properties make this beat assertions for this class of defect:

1. **It catches what you did not think to assert.** Nobody writes
   `assert results_do_not_arrive_in_submission_order`. But a table where every
   completion lands in index order, on three concurrent workers, is *visibly
   weird* — and that weirdness is the finding.
2. **It catches what is only wrong in context.** A 100% pass rate is not an
   error. A 100% pass rate on the units that have no grader is a catastrophe.
   The screen puts them adjacent; a test would have to already know the
   relationship, which means already knowing the bug.

---

## The laws

Seventeen, each stated with the concrete failure that produced it. The full
text with incidents is in [references/design-laws.md](references/design-laws.md);
generated code cites them by number.

| # | Law |
|---|---|
| L1 | The observer must never be able to affect the observed |
| L2 | Tolerate partial writes; never raise on them |
| L3 | Liveness is probed, not inferred |
| L4 | Show stalled; never drop it |
| L5 | Absent, empty, and filtered are three different states |
| L6 | Rank by the question, not by arrival |
| L7 | The cursor anchors to identity, not position |
| L8 | Allocate space by priority, not proportion |
| L9 | Render the noise floor |
| L10 | One data path, so no two surfaces can disagree |
| L11 | Name your callouts after the experiment they already ruined |
| L12 | Show the field that explains the verdict, on the rows whose verdict needs it |
| L13 | The legend must be depth-accurate |
| L14 | Keep reading and spending in different rooms |
| L15 | Control means stop, not edit |
| L16 | Encode each fix as a display rule |
| L17 | The screen produces suspicion, not evidence |

**L1 is absolute. Everything else is negotiable.**

A rule without its incident gets rationalized away at the first inconvenience,
which is why the law text carries the failure and why generated code carries
the citation.

---

## The four phases

Each produces one PR. Two of the transitions are gates that say *stop*.

| Phase | Produces | Leave it when |
|---|---|---|
| 1 — Evidence and question | `WATCHING.md`: the on-disk evidence table, the question the screen settles, the wrong shapes | Every path resolves against a real output directory |
| 2 — The watcher | `watch-<job>.sh` — in-place redraw, ranked, callouts | It has run against a live job **and a dying one** |
| 3 — The browser | Shared loader + tabbed curses views | Both surfaces show identical numbers for every quantity |
| 4 — The live view | In-flight reader, probed liveness, teardown | Rows stop claiming to be live within one tick of a kill |

| Stop condition | Because |
|---|---|
| Phase 2 exists and no view is missing | Phase 2 is frequently the whole answer. 273 lines of bash caught three run-changing defects |
| Results land fast enough to watch | Phase 4 buys nothing |

## What the pattern refuses

**The inversion.** If most of the columns you want require fields the job does
not emit, the pattern declines. Adding a field to the *runner* because the run
should record it is fine — it becomes evidence whether or not anyone watches.
Adding one *for the viewer* means the observer is changing the observed, and
you now maintain two systems instead of one.

**Speculative depth.** Phases 3 and 4 are built when you can name the question
the current screen cannot answer. Not before.

**Jobs that do not need one.** Decline when any two hold: finishes in under a
couple of minutes, no concurrency, leaves no intermediate evidence, will run
once.

---

## Portability

A watcher does **not** port between jobs as code. It ports as rules — with one
exception, and the exception is instructive.

The best evidence for the rule is in [LINEAGE.md](LINEAGE.md): a 128-line
re-application written the day after a 273-line original, in the same
repository, for a different experiment — sharing no code and obeying every law.

What transfers: rank by the question, name your callouts after what they
already ruined, distinct glyphs for distinct absences, probe liveness, one data
path.

**The exception is the drawing layer.** `shared/tui/` — bounds-checked put, a
min-size guard, a footer renderer, a scroll indicator — has been vendored
byte-identically into three codebases across twenty-five days, because it
encodes nothing about any particular job. That is the test for what belongs in
a framework: if it knows what the numbers *mean*, it is not framework, and
copying it is how a watcher inherits the wrong question.

## Relationship to the sibling patterns

| Repository | Encodes | Answers |
|---|---|---|
| [directed-contexts](https://github.com/TGPSKI/directed-contexts) | A territory an agent operates inside | "What do I own and what are its invariants?" |
| [directed-workflows](https://github.com/TGPSKI/directed-workflows) | A process an agent walks a user through | "What is the next step and what does it produce?" |
| [abductive-triage](https://github.com/TGPSKI/abductive-triage) | A method for locating a fault | "Is this real, and where is it?" |
| **run-watcher** | An instrument the user reads while the process runs | "Is what I am watching actually what I think it is?" |

The connection to abductive triage is direct and deliberate: the watcher's
anomaly catalogue produces the *symptom*, and triage's coordinate-resolution
gate is what stops you from investigating a system when the instrument was
simply pointed at the wrong place. **L17 is the same claim from the other
side.**
