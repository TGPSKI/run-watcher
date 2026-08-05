# `assets/tui/` — the drawing layer, vendored from pane

Five stdlib-only modules, ~570 lines, vendored byte-identically from
[**pane**](https://github.com/TGPSKI/pane) — the canonical upstream of this
package. **Copy them into the target repository in Phase 3. Do not rewrite
them, and do not improve them.**

This is the one part of a run watcher that ports as *code* rather than as
rules, and the reason is the test worth remembering:

> **If it knows what the numbers mean, it is not framework.**

Nothing in here knows what a trial, an arm, a request or a repository is. It
can put a string inside the terminal bounds, refuse to draw in a window too
small to be honest, render a bar, prompt for a search, and say `1-22/73`.
That is all — which is exactly why it has survived four codebases unchanged
while every layer above it was rewritten from scratch each time.

## Provenance

Extracted on 2026-07-13 from two live server-analytics dashboards, vendored
byte-identically since, and promoted to its own repository —
[pane](https://github.com/TGPSKI/pane) — on 2026-08-04. The files here match,
to the byte, `src/pane/` upstream and the copies in:

- [leather](https://github.com/TGPSKI/leather) — `examples/14-sig-triage/eval/scripts/tui/`
- [adherence-suite](https://github.com/TGPSKI/adherence-suite) — `src/adherence/tui/`

Verify it yourself rather than believing it — pane ships the checker:

```bash
cd pane && make vendor-check    # diffs every portfolio copy against src/pane
```

To refresh this copy after an upstream change:

```bash
cd pane && tools/vendor.sh ../run-watcher/assets/tui
```

See [LINEAGE.md](../../LINEAGE.md) here and
[pane's LINEAGE.md](https://github.com/TGPSKI/pane/blob/main/LINEAGE.md) for
why the byte-identity claim matters.

## What each module gives you

| File | Provides | Laws it carries |
|---|---|---|
| `framework.py` | `TuiApp` (bounds-checked `_put`, colour pairs, run loop, footer, **`scroll_indicator`**), `curses_main` | **L8** — the `N–M of T` indicator is `scroll_indicator`, not something you write per project |
| `charts.py` | `bar_chart`: single or stacked series, peak markers, half-blocks, aggregation binning (long series bin rather than fall off the right edge), outlier clipping (`clip_ratio` caps the y-axis at a robust bound and labels what ran past it) | **L5**'s spirit at the axis: the y-max must describe a bar the user can actually see |
| `fmt.py` | `compact_num`, `human_bytes`, `duration`, `pct`, `sparkline`, safe `to_int`/`to_float` | **L5** — the coercers return a default rather than raising on a torn value |
| `interact.py` | `prompt_search` (bottom-row `/`), `filter_rows`, `hbar`, `cycle` | Extracted at the third duplication; takes the app as an argument so `framework.py` stays byte-identical |
| `windows.py` | `trailing_hours`, `trailing_days`, `trend`, `stack_cells` | see below |

Two things in `framework.py` are load-bearing and easy to delete by accident:

- **`_put` clips to the terminal and swallows `curses.error`.** A viewer that
  raises when a row is one column too wide is a viewer you stop launching.
- **The min-size guard in `run()`** prints *"Terminal too small (need 60x16)"*
  instead of drawing garbage. Absent, empty, and unreadable are three different
  states (**L5**), and a wrapped table reads as corrupt data.

## `windows.py`, and why it ships anyway

All five modules are here, including `windows.py`, which looks at first like
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
that is supposed to be the safe part. This is why pane's `tools/vendor.sh`
copies the whole package or nothing.

**Genuinely unused, and safe to ignore:** `read_csv` in `framework.py`, and the
three time-series functions above. They survive for byte-identity, not because
you need them — a watcher's reader owns loading, and results are JSONL.

The lesson generalizes past this file: **"unused" and "unreachable" are not the
same claim.** A lazy import inside a branch is invisible to every check that
reads the top of a file.

## Testing what you build on it

pane also ships `pty_smoke.py` (not vendored here — it's test infra, and
POSIX-only): drive any TUI in a real pseudo-terminal, feed it keys, assert a
clean exit with no traceback. Point it at your watcher from your own test
suite rather than writing a second harness.

## What you still have to write

Everything that knows anything:

- the reader (Phase 4), which turns on-disk evidence into records
- the loader shared with your report (**L10**)
- the views, the columns, the callouts, the noise floor
- every law from the [design laws](../../references/design-laws.md) except L8's
  indicator, which you get for free

The framework is roughly 570 lines. A useful watcher is 300–2,500 more. Copying
this is the last shortcut you get.
