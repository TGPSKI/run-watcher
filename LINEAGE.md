# Lineage

Run Watcher was not designed on a whiteboard. It accreted across five
generations in twenty-five days, in three codebases, and the accretion is the
most useful thing about it: each generation added exactly what the previous one
turned out to be missing, and the laws were extracted afterward from what had
already gone wrong.

Everything below is checkable. Dates come from `git log`.

## The twenty-five days

| Date | Generation | Artifact | Added |
|---|---|---|---|
| 2026-07-11 | **0** — the analytics TUIs | `sh-web-analytics/ui/terminal_graphs.py` (1,054 lines), then `sh-github-analytics` (1,227) | Curses views over live operational data; the screen as the place you look |
| 2026-07-13 | **0b** — the framework | `shared/tui/{framework,charts,fmt,windows}.py`, 384 lines | Bounds-checked draw, min-size guard, footer renderer, **scroll indicator** |
| 2026-07-15/16 | — | redesign across all three `sh-*` apps | The conventions settle under real daily use |
| 2026-07-29 | **1** — the watcher | `watch-matrix.sh`, 273 lines of bash | Ranked-not-run-ordered; named callouts; in-place ANSI redraw with `read -t` as the tick |
| 2026-07-30 | **1b** — the browser | `matrix-tui.py`, 660 lines of curses | Tabbed views, a facet picker, one shared loader, `NULL_BAND` |
| 2026-07-30 | **2** — the re-application | `15-trust-repair/eval/watch-matrix.sh`, 128 lines | Proof the pattern ports as rules, not code |
| 2026-08-03 | — | the `no-out` reattribution | L17 — the screen produces suspicion, not evidence |
| 2026-08-03/04 | **3** — the live view | `matrix_tui.py` (1,869) + `live.py` (629) | Work in flight; probed liveness; process control |

Nothing here took a quarter to mature, and the pattern did not begin in an
eval. It began in operations.

## Generation 0 — the analytics TUIs

The oldest ancestor is not a benchmark viewer. It is a pair of curses dashboards
over live server data: `sh-web-analytics` (nginx access logs across several
domains) and `sh-github-analytics` (repository traffic). Both predate the eval
suite by more than two weeks.

Those two live in a **private** operations repository, so unlike every other
reference in this document they cannot be followed. The claims made about them
below are the ones that survive that: line counts, dates, a docstring, and a
byte-for-byte diff against code you *can* read in
[leather](https://github.com/TGPSKI/leather) and
[adherence-suite](https://github.com/TGPSKI/adherence-suite).

They matter for three reasons.

**The framework is literally the same code.** `shared/tui/` was extracted from
those two apps on 2026-07-13 — its own docstring says so:

> Extracted from the two live TUIs (sh-web-analytics, sh-github-analytics):
> bounds-checked put, base color pairs, run loop with min-size guard, footer
> renderer, scroll indicator, CSV loading, and the curses.wrapper bootstrap.

All four files are **byte-identical** to the `tui/` package vendored into
leather's eval scripts and again into adherence-suite:

```
charts.py      127 lines   IDENTICAL
fmt.py          91 lines   IDENTICAL
framework.py   108 lines   IDENTICAL
windows.py      58 lines   IDENTICAL
```

So **L8's `N–M of T` indicator is not an eval idea.** It is
`TuiApp.scroll_indicator`, written for a log dashboard sixteen days earlier,
and every later watcher inherited it without rewriting a line.

**The anomaly-detector role was being practised before it was named.** Two
files in those repos are saved TUI screen captures, dated 2026-07-12:
`1d-all-tui-grab-7-12-26.md` and — note the filename —
`tui-capture-maybe-bugs-7-12-26.md`. Capturing the screen in order to inspect
it for anomalies *is* the pattern, three weeks before there was an experiment
to point it at. One capture shows a domain reading `2xx: 0 (0%) · 3xx: 245
(97%)` with top paths `/.env`, `/config.json`, `/wp-login.php`, `/.git/config`
— scanner traffic, legible instantly as shape, invisible in any single log
line.

**It is where the operational instincts came from.** Generation 0 ran against
data that changes whether or not anyone is watching, on a box doing real work.
That is the environment that teaches you a viewer must not be able to disturb
what it observes (**L1**), must survive a narrow terminal rather than emit
garbage (the min-size guard), and must distinguish a real zero from a missing
value — the github capture's `Active: 0` beside `Repos: 89` is exactly the
"count that is always exactly 0" entry in the catalogue.

An eval harness is a *quieter* version of this problem, not a different one.
The pattern arrived at the eval already load-bearing.

## Generation 1 — the watcher

Built for a pre-registered ablation matrix on SIG triage in
[leather](https://github.com/TGPSKI/leather): 22 classes, a 250-issue gold
corpus, arms differing only in harness design, four campaigns across two rigs.

Two decisions from that file survived into everything since, both stated in its
header comment:

> Cells are RANKED, not listed in run order: the question asked of this screen
> is always "which arm is winning", and run order buries that.

Now **L6**.

> Three callouts exist because each has already silently changed an experiment:
> `PAGINATED`, `UNATTRIB`, `STALE`.

Now **L11**. *Each has already silently changed an experiment* — not crashed
it, changed it — is the anomaly-detector thesis, fully formed, before there was
a pattern to put it in.

One more from that generation, in `VIEWING.md`:

> The end time comes from the last timestamp *inside* `run-evidence.log.gz`,
> not the file mtime: any git checkout or stash rewrites mtimes, which once had
> every archive reporting the same end time and 10-hour durations.

Now **L3**. mtime lied here first. It lied twice more in generation three, in
two different ways.

## Generation 1b — the browser

The watcher answers *what is happening*. It cannot answer *why is E2 beating
G*. So a second program read the same archives through the same loader:

> Everything reads the archives through `matrixdata.py` — the same loader and
> the same bridge `table.py` uses, so no two surfaces can disagree about a
> number.

Now **L10**. It also imports the registered analysis path's McNemar
implementation rather than writing a second one that could drift.

The single best idea in that file is one constant:

```python
NULL_BAND = 2.4          # measured single-run noise floor, in points
```

Now **L9** — the screen encodes the statistics, so a difference too small to
mean anything cannot look like a finding. One constant and two colour pairs.

Also from 1b: `-` for zero, `?` for an archive predating the field, `~` for an
inferred value, "so a guess is never mistaken for a record." Now **L5**.

## Generation 2 — the re-application

A different experiment in the same repository — trust repair — got its own
watcher the next day. Half the size, no shared code, its own two callouts:

> - `TRUNC` — the model hit the max_tokens ceiling mid-reasoning and the cell
>   measured the token budget, not the model.
> - `narr` — a `REPAIRED` claim over an empty diff: the honest-report failure
>   mode; the conjunction catches it, this column makes it visible.

This generation is why `PATTERN.md` says a watcher ports as rules rather than
as code, and why the workflow's variant table routes a second experiment to
Phase 2 with `design-laws.md` rather than to a copy of the first watcher.

## The reattribution — where L17 came from

Generation 1b's `no-out` column showed a consistently nonzero value for exactly
one arm: 25, 26, 28, 30, 34 rows across five draws. About 11%, reproducibly.
Noise does not do that.

It was read as the arm's mechanism — a fresh-session boundary making rows
unanswerable — and published as a −15 point arm anchoring a headline span.

Forensics on the archived runs later found something else. The fresh-session
stage, the only stage that never sees the original issue text, transcribes the
issue ID from its own prior output and corrupts the last digit in ~95% of
affected rows. The classification was frequently *right*, sitting in the
evidence log under a ghost ID, scored as silence. Recovering them moves the arm
to roughly −9 and the published span from 22 points to ~21; the defensible
statistic, the span of replicated arm means, is 14.8.

Both halves became law:

- The column made a rare, distributed, non-crashing anomaly **visible and
  reproducible** days before anyone knew what it was. That is the pattern
  working.
- The cause was assigned wrongly anyway. **L17**: the screen produces
  suspicion; a shell produces evidence. Part of what presents as "the treatment
  is expensive" is always, potentially, your instrument having a bug.

`anomaly-catalogue.md` carries the generalized form: *answers present in the
log but absent from the results* is attribution, not generation — and attribute
on the id you dispatched, never on the id the worker echoed.

## Generation 3 — the live view

Built for [adherence-suite](https://github.com/TGPSKI/adherence-suite), a
deterministic agent-adherence benchmark: six arms, thirty-four tasks from real
merged PRs, no model in the scoring path.

Every previous watcher read *archives*. A record is written only after a unit
finishes and is graded, so at 900-second timeouts and three concurrent workers
the screen sat at 0% for minutes with nothing to say — while the box was fully
loaded. The evidence existed and nothing joined it.

`live.py` is that join, and it is where the remaining laws came from:

| Law | The incident |
|---|---|
| **L1** | `os.kill(pid, 0)` is a probe on POSIX. On Windows `CTRL_C_EVENT == 0`, so the *check* delivered a real Ctrl-C into an unrelated subprocess, sixteen seconds into CI |
| **L3** | mtime-as-liveness showed four rows running three minutes after teardown; then watching `db` instead of `db-wal` showed a working delegation as hung, 282s stale against 29s fresh |
| **L7** | A positional cursor swapped the detail pane out mid-read as trials finished |
| **L8** | Proportional space gave the unbounded section 22 rows it did not need and pushed another off a 24-line terminal entirely — absent, not truncated |
| **L15** | A proposed destructive reset was cut down to a teardown that never deletes results |

That session produced roughly thirty-five defects, catalogued in the
repository's [`docs/SESSION-LOG.md`](https://github.com/TGPSKI/adherence-suite/blob/main/docs/SESSION-LOG.md).
Most were invisible in the results file, and several would have produced a
publishable-looking number.

## What the prototypes still teach

Three things the abstraction has not absorbed, recorded so they are not
mistaken for oversights:

1. **Bash is a real terminus, not a stepping stone.** Generation 1 caught three
   run-changing defects on its own, and generation 2 chose bash again with full
   knowledge of the curses version. The workflow's stop conditions exist
   because of this, and `watchctl` lints shell and Python equally.
2. **The callout list is domain knowledge and cannot be generated.** Every
   callout in every generation names something that had already gone wrong in
   that specific rig. Phase 1 asks for them; it does not invent them, and a
   callout whose "already changed" column is empty is marked speculative.
3. **The noise floor has to be measured before it can be rendered.** `NULL_BAND
   = 2.4` came from repeat draws of an identical arm. Where no such measurement
   exists, the pattern requires the band render as `UNKNOWN` rather than
   inviting an invented threshold.

## Sibling mechanisms

| Borrowed | From |
|---|---|
| Multi-phase router, Inspect–Decide–Generate, status-action tables, carry-forward blocks, PR checkpoints | [directed-workflows](https://github.com/TGPSKI/directed-workflows) |
| A deterministic CLI that inspects rather than asks; the decline-when-too-small gate; `references/` beside a root `SKILL.md` | [directed-contexts](https://github.com/TGPSKI/directed-contexts) |
| Evidence labelling, and the discipline that a symptom is not a diagnosis | [abductive-triage](https://github.com/TGPSKI/abductive-triage) |
| Repository frame: identity, governance, GitHub posture, build and quality | [scaffold-repo](https://github.com/TGPSKI/scaffold-repo) |

The connection to abductive triage is the deepest. Its central claim is that
the most common root cause is a coordinate mismatch — the reporter checked the
wrong place — and that this must be resolved before investigating any system.
**L17 is the same claim from the instrument's side.** The catalogue hands you a
symptom; triage stops you from spending a day on it before checking that the
screen was pointed at the right thing.
