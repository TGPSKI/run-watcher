# Example: SIG-triage ablation matrix

**Generations 1 and 1b — the watcher and the browser.** The origin of the
pattern, and the example to read first if you are deciding how much to build.

| | |
|---|---|
| Repository | [leather](https://github.com/TGPSKI/leather), `examples/14-sig-triage/eval/` |
| Job | Pre-registered ablation matrix: Kubernetes issue → SIG triage, 22 classes, 250-issue gold corpus |
| Built | 2026-07-29 (watcher), 2026-07-30 (browser) |
| Phases | 1 → 2 → 3. **Phase 4 deliberately not built** |
| Artifacts | `scripts/watch-matrix.sh` (273 lines bash), `scripts/matrix-tui.py` (660 lines curses), `VIEWING.md` |

## What it demonstrates

**Phase 2 is a real terminus.** The bash watcher caught three run-changing
defects on its own, and they are named in its header comment — which is where
the workflow's L11 comes from:

> Three callouts exist because each has already silently changed an experiment.

**`read -t` as the tick.** One call waits out the redraw interval *and* collects
a keystroke, so sorting is interactive with no second thread, no curses
dependency, and no change to the redraw model.

**The noise floor rendered (L9).** `NULL_BAND = 2.4` — one constant and two
colour pairs, and a difference too small to mean anything can no longer look
like a finding.

**One loader (L10).** Both surfaces read through `matrixdata.py`, and the
browser imports the *registered* McNemar implementation rather than writing a
second one that could drift.

**Distinct absences (L5).** `-` is zero, `?` is an archive predating the field,
`~` is inferred — "so a guess is never mistaken for a record."

## Why Phase 4 was not built

A draw finishes in minutes, so the archive table is useful on its own and the
screen never sat empty. The workflow's stop condition applies:

> Results land fast enough to watch → **Phase 4 buys nothing.**

This is the example to point at when a workflow's momentum is pushing you
toward building everything.

## Validate the plan

```bash
make check-examples                                   # both examples
go run scripts/watchctl.go plan --file examples/sig-triage/WATCHING.md
```

## What the linter finds on the real watcher

Recorded verbatim in [`findings.txt`](findings.txt), because an example that
only shows compliant code is not an example.

```
watch-matrix.sh:120 [warn] L5: a progress track of dots reads as truncated
    output rather than 0%.
    bar="$(printf ... | tr ' ' '#')$(printf ... | tr ' ' '.')"
watch-matrix.sh [warn] L9: no noise floor declared.
```

Both are true and neither is serious:

- **L5** is a genuine miss. The *filled* portion of the bar uses `#`, but the
  *track* uses `.` — exactly the case the law exists for. Generation 3 fixed it
  independently, which is how the law got written down.
- **L9** is correct but points at the wrong file: `NULL_BAND` lives in the
  browser, not the watcher. The finding is honest about what *this file*
  declares, and the right response is a pragma or a comment, not a code change.

**No L1, L2 or L3 findings.** The watcher never mutates, guards every read with
`2>/dev/null` and `${x:-0}`, and derives its end times from inside the evidence
log rather than from mtime.

## Read it in this order

1. [`WATCHING.md`](WATCHING.md) — the Phase 1 plan, reconstructed
2. [`watch-matrix.sh`](https://github.com/TGPSKI/leather/blob/main/examples/14-sig-triage/eval/scripts/watch-matrix.sh) — the header comment first
3. [`VIEWING.md`](https://github.com/TGPSKI/leather/blob/main/examples/14-sig-triage/eval/VIEWING.md) — every column, and what each can mislead you about
4. [`matrix-tui.py`](https://github.com/TGPSKI/leather/blob/main/examples/14-sig-triage/eval/scripts/matrix-tui.py) — `NULL_BAND` and its three colour thresholds
