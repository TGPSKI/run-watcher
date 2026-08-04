# AGENTS.md — run-watcher

This repository ships a **pattern**, a **directed workflow** that generates
instances of it, a **deterministic linter**, and **two golden examples**. It
does not ship a watcher. Every code path here inspects or generates; none of it
runs alongside a job.

## The one gate that filters every change

> **A law is only admitted with the incident that produced it.**

`references/design-laws.md` is L1–L17, and each carries the concrete failure it
exists to prevent. That is not decoration. A rule stated without its cost gets
rationalized away by the next person who finds it inconvenient, and the reason
`L1` survives in generated code is that its comment explains what it cost.

If you add a law, name the run it already changed. If you cannot, it is not a
law yet — put it in `anomaly-catalogue.md` as a shape to watch for.

## Repository layout

| Path | Contains | Changes when |
|---|---|---|
| `SKILL.md` | The router. Detects state, routes to a phase, **never generates** | Phase boundaries or stop conditions change |
| `PATTERN.md` | The specification: the claim, the three roles, the laws, what it refuses | The pattern itself changes |
| `LINEAGE.md` | Where each law came from, with dates and incidents | A new generation ships, or an attribution is corrected |
| `references/phase-0*.md` | The four phases, Inspect–Decide–Generate | A phase's outputs or gates change |
| `references/design-laws.md` | L1–L17 with their failures | A law is added, or an incident is corrected |
| `references/anomaly-catalogue.md` | Shape → cause → cheapest check | A new shape is identified in the field |
| `scripts/watchctl.go` | The deterministic linter and inventory tool | A law gains a textual signature |
| `examples/*/` | Golden Phase 1 plans + recorded linter findings | An example's source repository changes |

## Working principles

**The linter is a subset, and says so.** L6, L11, L16 and L17 are judgment, not
syntax. `watchctl` checks only what has a textual signature. Do not add a check
that guesses — a linter people learn to ignore is worse than one that stays
narrow.

**A law without a `watchctl` check is not a failure.** Most laws are unenforced
here and always will be. The tool's docstring states this explicitly; keep it
true.

**Golden plans must stay clean.** `examples/*/WATCHING.md` are validated by
`make check-examples`. If a `plan` check gains a rule, the goldens must satisfy
it or the rule is wrong.

**Findings files are recorded output, not aspirations.**
`examples/*/findings.txt` is the verbatim result of running `watchctl lint`
against the real viewer in its source repository. Regenerate them; never
hand-edit. They currently contain **true positives**, including one in the
author's own code, and that is the point — an example that only shows compliant
code is not an example.

**Never modify a source repository to make an example look better.** If
`live.py` earns an L1 finding, the finding stays and the example explains it.

## Development workflow

```bash
make help              # every target with a description
make check             # full CI: lint + fmt-check + test + golden validation
make test              # Go tests for watchctl
make check-examples    # validate both golden plans
make lint-viewer VIEWER=path/to/file    # run the law checks on a real viewer
```

`scripts/` is its own Go module (`scripts/go.mod`) so the repository root stays
language-neutral — it is a Markdown pattern repository that happens to ship one
tool.

## Constraints

| Constraint | Why |
|---|---|
| Go stdlib only in `scripts/` | The tool must run anywhere with a toolchain and nothing else |
| `watchctl` performs no writes, no network, no execution | It exists to enforce L1; a linter that violated its own first law would be worthless |
| Generated watchers are POSIX shell or stdlib Python | The instrument must never be a dependency problem at 2am |
| Every phase file declares `parent: run-watcher` | Phase files are independently invocable and must trace back to their router |

## Domain concepts

| Term | Means |
|---|---|
| **Evidence** | Something the job writes for its own reasons that the viewer happens to read. Not instrumentation added for the viewer |
| **Shape** | A property of the set rather than a row — a count, an ordering, a distribution, an absence |
| **Callout** | A named, glyph-prefixed warning for a silent behaviour change, documented with the run it already ruined |
| **The inversion** | Adding fields to the observed system to feed the observer. The pattern refuses it |
| **Stop condition** | A phase transition that says *do not continue*. Two exist and they are load-bearing |

## Change process

Phase and law edits are ordinary PRs. Two things need more care:

1. **Renumbering laws.** Generated code in other repositories cites `L3` by
   number. Add at the end; never renumber.
2. **Changing a `watchctl` check.** Run it against both example source repos
   before and after, and update `examples/*/findings.txt` from the actual
   output. A check whose behaviour changed silently is a check that will be
   distrusted.

## Contributing summary

- A law needs its incident.
- The linter stays narrow and honest about being a subset.
- Golden plans and findings files are regenerated, not edited.
