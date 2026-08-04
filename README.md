# Run Watcher

> Build a live-render TUI that is the control center, debugging surface, and
> anomaly detector for a long-running concurrent job.

A **run watcher** is a read-only terminal renderer that joins the evidence a
running job already leaves on disk and shows it as shape, on a tick, in one
screen. This repository ships the pattern specification, a directed workflow
that generates one for any job, a deterministic linter, and two golden examples
drawn from real benchmark runs.

The claim it is built on:

> Most defects in a concurrent long-running system are visible as **shape**
> long before they are visible as **values** — and a live render is the only
> instrument that shows shape.

Three workers where you expect four is a shape. A pass column that is all green
is a shape. Completions arriving in perfect submission order on a pool of three
is a shape. None of those appear in a log line, and every one of them is one
glance on a screen.

```text
your-job/
├── docs/WATCHING.md            # Phase 1: evidence, the question, the wrong shapes
├── scripts/watch-<job>.sh      # Phase 2: ranked, in-place redraw, callouts
├── <pkg>/loader.py             # Phase 3: one data path, shared with the report
├── <pkg>/<job>_tui.py          # Phase 3: tabbed views, the noise floor rendered
├── <pkg>/live.py               # Phase 4: work in flight, liveness probed
└── <pkg>/stoprun.py            # Phase 4: teardown that never deletes results
```

Outputs are POSIX shell and stdlib Python. No dependencies, no framework, no
web stack, and nothing that requires a particular agent harness.

The pattern was extracted from three generations of a working watcher built in
six days across [leather](https://github.com/TGPSKI/leather) and
[adherence-suite](https://github.com/TGPSKI/adherence-suite). It sits beside
[directed-contexts](https://github.com/TGPSKI/directed-contexts) (territories
agents operate inside) and
[directed-workflows](https://github.com/TGPSKI/directed-workflows) (processes
agents walk users through), and takes its evidence discipline from
[abductive-triage](https://github.com/TGPSKI/abductive-triage). Full ancestry,
with dates and incidents, is in [LINEAGE.md](LINEAGE.md).

## Quickstart

### Build a watcher for a job

Point your agent at the job's repository and invoke the skill
([`SKILL.md`](SKILL.md)). It will:

1. Inventory what the job already leaves on disk — `watchctl evidence`.
2. Check the job is worth a watcher at all, and decline if not.
3. Ask only what disk cannot answer: the question the screen settles, and which
   shapes are wrong.
4. Generate one phase per PR, stopping as soon as the screen answers the
   question.

```bash
# what the skill runs under the hood
go run scripts/watchctl.go evidence --root /tmp                       # Phase 1
go run scripts/watchctl.go plan     --file docs/WATCHING.md           # Phase 1 gate
go run scripts/watchctl.go lint     --viewer scripts/watch-job.sh     # Phase 2+
```

### Lint an existing viewer

```bash
make lint-viewer VIEWER=path/to/your/watcher.py
```

`watchctl lint` checks the mechanically-checkable laws: mutation inside a
viewer, unguarded parses, mtime-derived liveness, journaled stores read without
their sidecar, dot-track progress bars, and an undeclared noise floor.

## The four phases

Each produces one PR. Two of the transitions are gates that say *stop*.

| Phase | Produces | Leave it when |
|---|---|---|
| 1 — Evidence and question | `WATCHING.md` | Every path resolves against a real output directory |
| 2 — The watcher | `watch-<job>.sh` | It has run against a live job **and a dying one** |
| 3 — The browser | Shared loader + tabbed views | Both surfaces show identical numbers |
| 4 — The live view | In-flight reader, probed liveness, teardown | Rows stop claiming to be live within one tick of a kill |

**Phase 2 is frequently the whole answer.** 273 lines of bash caught three
run-changing defects in the origin example, and that example deliberately never
built Phase 4.

## The laws

Seventeen, each stated with the concrete failure that produced it —
[references/design-laws.md](references/design-laws.md). Generated code cites
them by number, because a rule without its incident gets rationalized away at
the first inconvenience.

**L1 is absolute: the observer must never be able to affect the observed.**
Everything else is negotiable. The clearest illustration is in the law text
itself — a liveness probe using `os.kill(pid, 0)`, the standard POSIX existence
check, where on Windows `signal.CTRL_C_EVENT == 0` and the *check* delivered a
real Ctrl-C into an unrelated subprocess.

## Repository layout

```text
run-watcher/
├── SKILL.md                    # the router: detects, routes, never generates
├── PATTERN.md                  # the pattern specification
├── LINEAGE.md                  # where each law came from, with dates
├── references/
│   ├── phase-01-evidence-and-question.md
│   ├── phase-02-watcher.md
│   ├── phase-03-browser.md
│   ├── phase-04-live-view.md
│   ├── design-laws.md          # L1-L17 with their incidents
│   └── anomaly-catalogue.md    # shape → cause → cheapest check
├── examples/
│   ├── sig-triage/             # generations 1 and 1b; Phase 4 declined
│   └── adherence-suite/        # generation 3; the full path
└── scripts/watchctl.go         # deterministic linter and inventory tool
```

## The anomaly catalogue

[`references/anomaly-catalogue.md`](references/anomaly-catalogue.md) is the part
you reach for mid-incident: shape → likely cause → the cheapest check that
settles it. It covers liveness, arrival patterns, verdicts that are too good,
costs and counts, prompt and delivery, distributed non-crashing losses, and the
instrument itself.

The most dangerous row in it:

> **A cell at 100%, or every trial passing** — the grader is not grading.

That is not an error, does not crash, and produces no warning. It is a good
result that is wrong, and the only thing that catches it is a human noticing
that a column looks too easy.

## Install as a skill

```bash
git clone https://github.com/TGPSKI/run-watcher ~/git/run-watcher
ln -s ~/git/run-watcher ~/.agents/skills/run-watcher
```

Harnesses that read `~/.claude/skills` will pick it up if that path symlinks to
`~/.agents/skills`. The workflow is portable Markdown and does not require any
particular agent runtime.

## Development

```bash
make help              # every target
make check             # full CI: lint + fmt-check + test + golden validation
make check-examples    # validate both golden plans under watchctl
```

## Field note

The essay this pattern was extracted from — including what the instrument got
wrong, and the published figure it revised — is *Watch the run* (Fractal
Engineering, August 2026).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). The one rule worth stating here: a law
is only admitted with the incident that produced it.

## License

GPL-3.0. See [LICENSE](LICENSE).
