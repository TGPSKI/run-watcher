package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func laws(rep *Report) []string {
	var out []string
	for _, f := range rep.Findings {
		out = append(out, string(f.Law))
	}
	return out
}

func has(rep *Report, law string) bool {
	for _, f := range rep.Findings {
		if f.Law == law {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------------ L1

func TestLintFlagsMutation(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "watch-job.sh", `#!/usr/bin/env bash
snapshot() {
  rm -rf "$OUT/tmp"
}
`)
	rep, err := lintViewer(p)
	if err != nil {
		t.Fatal(err)
	}
	if !has(rep, "L1") {
		t.Fatalf("expected L1 for rm in a viewer, got %v", laws(rep))
	}
	if rep.worst() != Error {
		t.Fatal("a viewer that deletes must be an error, not a warning")
	}
}

func TestLintTeardownIsExemptFromL1(t *testing.T) {
	// The teardown is SUPPOSED to kill things (L15). Flagging it would
	// train people to ignore L1 findings.
	d := t.TempDir()
	p := write(t, d, "stoprun.py", `
import os, signal
os.killpg(pgid, signal.SIGTERM)
`)
	rep, err := lintViewer(p)
	if err != nil {
		t.Fatal(err)
	}
	if has(rep, "L1") {
		t.Fatalf("teardown must be exempt from L1, got %v", laws(rep))
	}
}

func TestLintPragmaRequiresReason(t *testing.T) {
	d := t.TempDir()
	withReason := write(t, d, "a.py", `
os.remove(debris)   # watchctl:allow L1 sweep path, never called by render
`)
	rep, _ := lintViewer(withReason)
	if has(rep, "L1") {
		t.Fatal("a pragma with a reason should suppress L1")
	}

	bare := write(t, d, "b.py", `
os.remove(debris)   # watchctl:allow L1
`)
	rep2, _ := lintViewer(bare)
	if !has(rep2, "L1") {
		t.Fatal("a bare pragma with no reason must NOT suppress; that is how a law " +
			"quietly stops applying")
	}
}

func TestLintOsKillNeedsPlatformGuard(t *testing.T) {
	d := t.TempDir()
	unguarded := write(t, d, "live.py", `
def alive(pid):
    os.kill(pid, 0)
    return True
`)
	rep, _ := lintViewer(unguarded)
	if !has(rep, "L1") {
		t.Fatalf("unguarded os.kill must be flagged: on Windows signal 0 is CTRL_C_EVENT, "+
			"got %v", laws(rep))
	}

	guarded := write(t, d, "live2.py", `
def alive(pid):
    if not pid or os.name != "posix":
        return False
    try:
        os.kill(pid, 0)
    except OSError as e:
        return e.errno == errno.EPERM
    return True
`)
	rep2, _ := lintViewer(guarded)
	if has(rep2, "L1") {
		t.Fatalf("guarded os.kill must pass, got %v", laws(rep2))
	}
}

// ------------------------------------------------------------------ L2

func TestLintUnguardedParse(t *testing.T) {
	d := t.TempDir()
	bad := write(t, d, "reader.py", `
def rows(path):
    return [json.loads(l) for l in open(path)]
`)
	rep, _ := lintViewer(bad)
	if !has(rep, "L2") {
		t.Fatalf("an unguarded parse must be flagged: a torn final line is normal, got %v",
			laws(rep))
	}

	good := write(t, d, "reader2.py", `
NULL_BAND = 2.4

def rows(path):
    out = []
    for l in open(path):
        try:
            out.append(json.loads(l))
        except ValueError:
            continue          # half-written final line: normal, skipped
    return out
`)
	rep2, _ := lintViewer(good)
	if has(rep2, "L2") {
		t.Fatalf("a guarded parse must pass, got %v", laws(rep2))
	}
}

// ------------------------------------------------------------------ L3

func TestLintMtimeLiveness(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "live.py", `
def is_running(path):
    age = time.time() - os.path.getmtime(path)
    return age < 90
`)
	rep, _ := lintViewer(p)
	if !has(rep, "L3") {
		t.Fatalf("mtime near a liveness word must be flagged, got %v", laws(rep))
	}
}

func TestLintMtimeAwayFromLivenessIsFine(t *testing.T) {
	// mtime for change detection is legitimate; only liveness is the bug.
	d := t.TempDir()
	p := write(t, d, "loader.py", `
def newest_mtime(paths):
    return max(os.path.getmtime(p) for p in paths)
`)
	rep, _ := lintViewer(p)
	if has(rep, "L3") {
		t.Fatalf("mtime for change detection is fine, got %v", laws(rep))
	}
}

func TestLintStoreWithoutSidecar(t *testing.T) {
	d := t.TempDir()
	bad := write(t, d, "s.py", `
con = sqlite3.connect(out / "opencode.db")
`)
	rep, _ := lintViewer(bad)
	if !has(rep, "L3") {
		t.Fatalf("a store read with no sidecar mention must warn, got %v", laws(rep))
	}

	good := write(t, d, "s2.py", `
con = sqlite3.connect(out / "opencode.db")
fresh = max(mtime(db), mtime(str(db) + "-wal"))
`)
	rep2, _ := lintViewer(good)
	for _, f := range rep2.Findings {
		if f.Law == "L3" && f.Line == 0 {
			t.Fatal("acknowledging the WAL sidecar must clear the file-level L3 warning")
		}
	}
}

// ------------------------------------------------------------------ L5

func TestLintDotProgressTrack(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "w.sh", `
bar="$(printf '%*s' "$pct" '' | tr ' ' '.')"
`)
	rep, _ := lintViewer(p)
	if !has(rep, "L5") {
		t.Fatalf("a dot progress track must warn: it reads as truncated output, got %v",
			laws(rep))
	}
}

// ------------------------------------------------------------------ plan

func TestPlanRequiresWrittenWhenColumn(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "WATCHING.md", `# Watching

## Evidence on disk

| Source | Path | Carries |
|---|---|---|
| stream | out/stdout.txt | calls |

## The question

Which arm is winning.

## Callouts

| Name | Glyph | Means | Already changed |
|---|---|---|---|
| STALE | ! | artifacts outlive the run | the 12-hour battery |

NULL_BAND = 2.4
`)
	rep, err := checkPlan(p)
	if err != nil {
		t.Fatal(err)
	}
	if !has(rep, "P1") {
		t.Fatalf("a plan without 'written when' must fail: that column decides whether a "+
			"live view is needed, got %v", laws(rep))
	}
}

func TestPlanCompleteIsClean(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "WATCHING.md", `# Watching

## Evidence on disk

| Source | Path | Written when | Carries |
|---|---|---|---|
| stream | out/stdout.txt | continuously, appended | calls, tokens |
| result | runs/probe.jsonl | on completion only | verdict |

## The question

Ranking axis: pass rate, descending.

## Callouts

| Name | Glyph | Means | Already changed |
|---|---|---|---|
| STALE | ! | artifacts outlive the run | the 12-hour battery reported idle |

## Noise floor

NULL_BAND = 2.4 points
`)
	rep, err := checkPlan(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Findings) != 0 {
		t.Fatalf("a complete plan must be clean, got %v", rep.Findings)
	}
}

func TestPlanCalloutWithoutIncident(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "WATCHING.md", `# Watching

## Evidence

| Source | Written when |
|---|---|
| stream | continuously |

## The question

Which is stuck.

## Callouts

| Name | Glyph | Means | Already changed |
|---|---|---|---|
| GUESS | ! | might happen | — |

## Noise floor

UNKNOWN
`)
	rep, _ := checkPlan(p)
	if !has(rep, "L11") {
		t.Fatalf("a callout with no incident must warn, got %v", laws(rep))
	}
}

func TestPlanUnknownFloorIsAccepted(t *testing.T) {
	// Declaring UNKNOWN is correct behaviour. Inventing a number is not.
	d := t.TempDir()
	p := write(t, d, "WATCHING.md", `# W

## Evidence

| Source | Written when |
|---|---|
| stream | continuously |

## The question

Which arm wins.

## Callouts

| Name | Glyph | Means | Already changed |
|---|---|---|---|
| STALE | ! | outlives the run | battery reported idle |

## Noise floor

NULL_BAND = UNKNOWN -- never measured on this rig
`)
	rep, _ := checkPlan(p)
	if has(rep, "L9") {
		t.Fatalf("an explicit UNKNOWN floor must be accepted, got %v", laws(rep))
	}
}

// ------------------------------------------------------------------ evidence

func TestEvidenceClassifiesAndFindsPhase4Trigger(t *testing.T) {
	d := t.TempDir()
	write(t, d, "adh-out-1/stdout.txt", "{}\n")
	write(t, d, "runs/results.jsonl", "{}\n")

	inv, err := scanEvidence(d)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, s := range inv.Sources {
		kinds = append(kinds, s.Kind)
	}
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, "stream") || !strings.Contains(joined, "result") {
		t.Fatalf("expected stream and result, got %v", kinds)
	}
	var gaps string
	for _, g := range inv.Gaps {
		gaps += g + "\n"
	}
	if !strings.Contains(gaps, "marker") {
		t.Fatalf("a missing marker must be reported as a gap, got: %s", gaps)
	}
}

func TestEvidenceAttachesSidecars(t *testing.T) {
	d := t.TempDir()
	write(t, d, "out/store.db", "")
	write(t, d, "out/store.db-wal", "")

	inv, err := scanEvidence(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range inv.Sources {
		if s.Kind == "store" {
			if len(s.Sidecars) == 0 {
				t.Fatal("the WAL sidecar must attach to its store, not stand alone -- " +
					"that attachment is what makes L3 visible in Phase 1")
			}
			return
		}
	}
	t.Fatal("no store classified")
}

func TestEvidenceResultOnlyIsPhase4Trigger(t *testing.T) {
	d := t.TempDir()
	write(t, d, "runs/results.jsonl", "{}\n")

	inv, _ := scanEvidence(d)
	var gaps string
	for _, g := range inv.Gaps {
		gaps += g + "\n"
	}
	if !strings.Contains(gaps, "Phase 4") {
		t.Fatalf("results-only must name the Phase 4 trigger, got: %s", gaps)
	}
}
