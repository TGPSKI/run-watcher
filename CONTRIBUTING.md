# Contributing

This repository ships a pattern, a workflow that generates instances of it, a
linter, and two golden examples. Contributions are welcome in all four.

## Development setup

```bash
git clone https://github.com/TGPSKI/run-watcher
cd run-watcher
command make help          # every target, with a description
command make check         # everything CI runs
```

Requirements: Go (version from `scripts/go.mod`) and a POSIX shell. Nothing
else. `scripts/` is its own Go module so the repository root stays
language-neutral.

## Development commands

| Command | Does |
|---|---|
| `make check` | Full CI: lint + fmt-check + test + golden validation |
| `make test` | Go tests for `watchctl` — law checks, plan checks, evidence scan |
| `make lint` | `go vet`, plus: every phase file declares `parent: run-watcher`, and every `references/` path named in `SKILL.md` resolves |
| `make fmt` / `fmt-check` | Format Go / fail if unformatted |
| `make check-examples` | Validate both golden Phase 1 plans under `watchctl plan` |
| `make lint-viewer VIEWER=…` | Run the law checks against a real viewer |
| `make regen-findings SIBLINGS=…` | Regenerate `examples/*/findings.txt` from local clones |

## The rule that governs this repository

> **A law is only admitted with the incident that produced it.**

`references/design-laws.md` is L1–L17, each stated with the concrete failure it
prevents. A rule without its cost gets rationalized away by the next person who
finds it inconvenient — that is not a hypothetical, it is why the Windows
`os.kill` comment has survived three refactors in two repositories.

If you have a rule but no incident, it belongs in
`references/anomaly-catalogue.md` as a shape to watch for, not in the laws.

## Contribution types

### Adding or amending a law

1. State the incident. What ran, what it showed, what it cost.
2. **Append. Never renumber.** Generated code in other repositories cites `L3`
   by number; renumbering silently invalidates comments you cannot see.
3. If the law has a textual signature, consider a `watchctl` check — see below.
4. Update `PATTERN.md`'s law table and `LINEAGE.md` if the incident came from a
   new generation.

### Changing a `watchctl` check

The linter is deliberately a **subset**. L6, L11, L16 and L17 are judgment, not
syntax, and no check should pretend otherwise. A linter people learn to ignore
is worse than one that stays narrow.

Before opening the PR:

1. Add a test for both directions — the violation *and* the compliant form. The
   compliant-form test is the one that stops false positives shipping.
2. Run the check against both example source repositories.
3. Regenerate `examples/*/findings.txt` with `make regen-findings`. Never
   hand-edit them; they are recorded output.
4. If the finding count changed, say why in the PR body.

### Adding an example

An example needs all three:

- `WATCHING.md` — a Phase 1 plan that validates clean under `watchctl plan`
- `README.md` — what it demonstrates, and **which phases it declined and why**
- `findings.txt` — verbatim `watchctl lint` output against the real viewer

Examples must come from a job that actually ran. A constructed example teaches
the tool, not the pattern.

**Never modify a source repository to make an example look better.** The
existing examples contain true positives, including one in the author's own
code. That is the point.

### Editing a phase

Phase files follow Inspect–Decide–Generate, encode all conditional logic as
status–action tables, and end with a PR checkpoint. Conditional logic in prose
will be asked to become a table.

Every phase declares `parent: run-watcher` — `make lint` enforces it.

## PR requirements

- `make check` passes. CI runs exactly those steps, so a green local run is a
  green CI run.
- Golden plans stay clean. If a new `plan` rule breaks them, the rule is
  probably wrong — the goldens are drawn from real, working practice.
- One logical change per PR. Law changes, linter changes, and example changes
  are separate reviews.

## What not to contribute

- **A check that guesses.** Precision over recall, always.
- **A law you have not paid for.** The catalogue is the right home for
  suspicions.
- **A generated watcher.** This repository ships the pattern, not instances.
  Instances live in the repositories whose jobs they watch.
