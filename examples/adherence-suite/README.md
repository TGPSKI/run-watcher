# Example: the adherence battery

**Generation 3 — the live view.** The full path, and the example to read when
your screen sits empty while the machine is busy.

| | |
|---|---|
| Repository | [adherence-suite](https://github.com/TGPSKI/adherence-suite) |
| Job | Deterministic agent-adherence benchmark: 6 arms, 34 PR-derived tasks, 120 trials, 3 concurrent workers, 900s timeouts |
| Built | 2026-08-03/04 |
| Phases | 1 → 2 → 3 → 4 |
| Artifacts | `src/adherence/live.py` (629 lines, the reader), `src/adherence/matrix_tui.py` (1,869 lines, the renderer), `src/adherence/stoprun.py` (the teardown) |

## What it demonstrates

**The Phase 4 trigger, in the evidence table.** A result row is written only
after a trial finishes and is graded. At 900-second timeouts and three workers,
that is minutes of a viewer showing 0% while three agents are very much doing
something. The `written when` column is where that becomes visible — which is
why Phase 1 makes it mandatory.

**Liveness probed, not inferred (L3).** The runner passes the output directory
to the worker as an argument, so it sits in `ps` for the life of the trial. It
replaced a file-age heuristic that reported killed runs as live for three more
minutes.

**The WAL trap (L3, again).** SQLite in WAL mode writes to `*.db-wal`; the main
file's mtime moves only at checkpoints. Measured at 282s stale against a
29s-fresh WAL, which showed a working delegation as hung. `watchctl evidence`
attaches sidecars to their principal specifically so this appears in Phase 1
rather than being discovered in Phase 4.

**The platform guard (L1).** `os.kill(pid, 0)` is a probe on POSIX and a real
Ctrl-C on Windows, where `signal.CTRL_C_EVENT == 0`. The comment recording that
is reproduced verbatim in `phase-04-live-view.md`, because it is the clearest
statement of the first law in either codebase.

**Priority space allocation (L8).** Running and summary are bounded, so they
render whole; graded grows all run, so it absorbs the scrolling.

**An honestly UNKNOWN noise floor (L9).** Never measured on this rig, so the
screen renders no significance colouring at all rather than inventing a
threshold. `watchctl plan` accepts an explicit `UNKNOWN` and rejects silence.

## The finding no plan would have caught

Two tasks reported 5/5 PASS for hours. Not an error, not a crash, no warning —
a good result that was wrong. Neither grader could judge those tasks, so
`all_pass` had degraded to "touched the right file," which an agent passes by
adding a comment.

The results file said `pass`. It was caught by a human noticing that a column
looked too easy.

This is the clearest case for the anomaly-detector role in either example:
nobody would have written the assertion, and the shape was only wrong *in
context* — 100% is not an error until it sits next to the tasks that have no
grader.

## What the linter finds on the real reader

Recorded verbatim in [`findings.txt`](findings.txt).

```
live.py:580 [error] L1: viewer performs a mutating operation (delete).
    shutil.rmtree(d, ignore_errors=True)
live.py [warn] L9: no noise floor declared.
```

**The L1 finding is a true positive and the tool is right to raise it.** That
call is inside `sweep()`, the debris-removal function — documented in the
module docstring as "the only non-read-only function here, and the renderer
never calls it." But *documented* is not *checkable*. The correct resolution is
the pragma, which forces the reason into the line itself:

```python
shutil.rmtree(d, ignore_errors=True)   # watchctl:allow L1 sweep(); never reachable from render
```

This is exactly the case the pragma exists for, and it is why a bare
`watchctl:allow L1` with no reason does not suppress anything.

The L9 warning is correct and answered by the plan: the floor is genuinely
`UNKNOWN` on this rig.

## Validate the plan

```bash
make check-examples
go run scripts/watchctl.go plan --file examples/adherence-suite/WATCHING.md
```

## Read it in this order

1. [`WATCHING.md`](WATCHING.md) — the plan, including what it got wrong
2. [`live.py`](https://github.com/TGPSKI/adherence-suite/blob/main/src/adherence/live.py) — the module docstring, then `_alive()`
3. [`matrix_tui.py`](https://github.com/TGPSKI/adherence-suite/blob/main/src/adherence/matrix_tui.py) — `view_live()` and its space-allocation comment
4. [`SESSION-LOG.md`](https://github.com/TGPSKI/adherence-suite/blob/main/docs/SESSION-LOG.md) — the full defect catalogue this example is drawn from
