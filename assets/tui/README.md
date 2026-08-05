# `assets/tui/` — the drawing layer, to be copied verbatim

Three stdlib-only modules, 326 lines. **Copy them into the target repository in
Phase 3. Do not rewrite them, and do not improve them.**

This is the one part of a run watcher that ports as *code* rather than as
rules, and the reason is the test worth remembering:

> **If it knows what the numbers mean, it is not framework.**

Nothing in here knows what a trial, an arm, a request or a repository is. It
can put a string inside the terminal bounds, refuse to draw in a window too
small to be honest, render a bar, and say `1-22/73`. That is all — which is
exactly why it has survived three codebases unchanged while every layer above
it was rewritten from scratch each time.

## Provenance

Extracted on 2026-07-13 from two live server-analytics dashboards, and vendored
byte-identically since. The files here match, to the byte, the copies in:

- [leather](https://github.com/TGPSKI/leather) — `examples/14-sig-triage/eval/scripts/tui/`
- [adherence-suite](https://github.com/TGPSKI/adherence-suite) — `src/adherence/tui/`

Verify it yourself rather than believing it:

```bash
diff assets/tui/framework.py /path/to/leather/examples/14-sig-triage/eval/scripts/tui/framework.py
```

See [LINEAGE.md](../../LINEAGE.md) for why that matters.

## What each module gives you

| File | Provides | Laws it carries |
|---|---|---|
| `framework.py` | `TuiApp` (bounds-checked `_put`, colour pairs, run loop, footer, **`scroll_indicator`**), `curses_main` | **L8** — the `N–M of T` indicator is `scroll_indicator`, not something you write per project |
| `charts.py` | `bar_chart` and friends | — |
| `fmt.py` | `compact_num`, `human_bytes`, `duration`, `pct`, `sparkline`, safe `to_int`/`to_float` | **L5** — the coercers return a default rather than raising on a torn value |

Two things in `framework.py` are load-bearing and easy to delete by accident:

- **`_put` clips to the terminal and swallows `curses.error`.** A viewer that
  raises when a row is one column too wide is a viewer you stop launching.
- **The min-size guard in `run()`** prints *"Terminal too small (need 60x16)"*
  instead of drawing garbage. Absent, empty, and unreadable are three different
  states (**L5**), and a wrapped table reads as corrupt data.

## `windows.py`, and why it ships anyway

All four modules are here, including `windows.py`, which looks at first like
pure analytics vestige — `trailing_hours`, `trailing_days` and `trend` are
time-series helpers whose docstring names the "hourly CSV wire format", and no
watcher in three generations has called any of them.

It ships because of one line in `charts.py`:

```python
if segments:
    from .windows import stack_cells
```

`stack_cells` is a **charting** function that happens to live in the
time-series module. Dropping `windows.py` leaves `bar_chart` working for every
call the existing watchers make, and raising `ImportError` the first time
someone passes stacked segments — a latent failure planted in the one component
that is supposed to be the safe part.

Keeping the package whole also keeps it byte-identical to the public copies,
which is what makes the `diff` above worth running.

**Genuinely unused, and safe to ignore:** `read_csv` in `framework.py`, and the
three time-series functions above. They survive for byte-identity, not because
you need them — a watcher's reader owns loading, and results are JSONL.

The lesson generalizes past this file: **"unused" and "unreachable" are not the
same claim.** A lazy import inside a branch is invisible to every check that
reads the top of a file.

## What you still have to write

Everything that knows anything:

- the reader (Phase 4), which turns on-disk evidence into records
- the loader shared with your report (**L10**)
- the views, the columns, the callouts, the noise floor
- every law from the [design laws](../../references/design-laws.md) except L8's
  indicator, which you get for free

The framework is roughly 300 lines. A useful watcher is 300–2,500 more. Copying
this is the last shortcut you get.
