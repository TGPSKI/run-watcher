// Command watchctl inspects run watchers and the jobs they watch.
//
//	watchctl evidence --root DIR     inventory the on-disk evidence (Phase 1)
//	watchctl lint     --viewer FILE  check a viewer against the design laws
//	watchctl plan     --file FILE    check a WATCHING.md plan is complete
//
// Deterministic and read-only. It performs no writes, opens no network
// connection, and never executes anything it finds. That is not incidental:
// this tool exists to enforce L1 -- the observer must never be able to affect
// the observed -- and a linter that violated its own first law would be
// worthless.
//
// The law checks are deliberately a SUBSET. L6 (rank by the question), L11
// (name your callouts after what they ruined), L16 (encode each fix as a
// display rule) and L17 (the screen produces suspicion) are judgment, not
// syntax; no linter can decide them and this one does not pretend to. What is
// here is the mechanical residue: the things that have a textual signature and
// have each shipped at least once.
//
// Stdlib only.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const usage = `watchctl -- inspect run watchers and the jobs they watch

  watchctl evidence --root DIR [--json]    inventory on-disk evidence (Phase 1)
  watchctl lint --viewer FILE [--json]     check a viewer against the laws
  watchctl plan --file FILE [--json]       check a WATCHING.md plan

Exit 0 when clean, 1 on findings, 2 on usage or I/O error.
`

// ---------------------------------------------------------------- findings

// Severity separates "this will mislead you" from "this can hurt the run".
type Severity string

const (
	Error Severity = "error"
	Warn  Severity = "warn"
)

type Finding struct {
	Law      string   `json:"law"`
	Severity Severity `json:"severity"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Message  string   `json:"message"`
	Evidence string   `json:"evidence,omitempty"`
}

type Report struct {
	Target   string    `json:"target"`
	Findings []Finding `json:"findings"`
}

func (r *Report) add(f Finding) { r.Findings = append(r.Findings, f) }

func (r *Report) worst() Severity {
	for _, f := range r.Findings {
		if f.Severity == Error {
			return Error
		}
	}
	return Warn
}

// ---------------------------------------------------------------- pragmas

// A pragma suppresses one law on one line, and REQUIRES a reason. A bare
// suppression is how a law quietly stops applying; making the reason
// mandatory means the next reader can judge whether it still holds.
//
//	# watchctl:allow L1 sweep() is the teardown path, never called by render
var pragmaRe = regexp.MustCompile(`watchctl:allow\s+(L\d+)\s+(\S.*)$`)

func allowed(line, law string) bool {
	m := pragmaRe.FindStringSubmatch(line)
	return m != nil && m[1] == law && strings.TrimSpace(m[2]) != ""
}

// ---------------------------------------------------------------- lint

// L1: things that mutate. A viewer reads. The teardown is a separate program
// (L15), so any of these inside a viewer is either a bug or needs a pragma.
var mutators = []struct {
	re   *regexp.Regexp
	what string
}{
	{regexp.MustCompile(`\brm\s+-|\brm\s+"`), "rm"},
	{regexp.MustCompile(`\bmv\s+|\bshutil\.move\b`), "move"},
	{regexp.MustCompile(`\bkill\s+-|\bos\.killpg\b|\bpkill\b`), "kill"},
	{regexp.MustCompile(`\btruncate\b|\bos\.truncate\b`), "truncate"},
	{regexp.MustCompile(`\bos\.remove\b|\bos\.unlink\b|\bshutil\.rmtree\b`), "delete"},
	{regexp.MustCompile(`\bopen\([^)]*["'][wa]`), "open for write"},
	{regexp.MustCompile(`\bchmod\b|\bos\.chmod\b`), "chmod"},
}

// L1: os.kill(pid, 0) is a probe on POSIX and a real Ctrl-C on Windows, where
// signal.CTRL_C_EVENT == 0. The guard must be present in the same function.
var osKillRe = regexp.MustCompile(`os\.kill\(`)
var posixGuardRe = regexp.MustCompile(`os\.name\s*!=\s*["']posix["']|os\.name\s*==\s*["']posix["']|sys\.platform`)

// L2: a parse that can meet a half-written final line.
var parseRe = regexp.MustCompile(`json\.loads\(|json\.load\(`)
var guardRe = regexp.MustCompile(`\btry\b|\bexcept\b|contextlib\.suppress`)

// L3: mtime near a liveness word is the bug that has shipped in every
// generation. The proximity window is deliberately small to keep this honest.
var mtimeRe = regexp.MustCompile(`getmtime|st_mtime|stat -c %Y|stat -f %m`)

// Boundaries are [^a-z] rather than \b so that snake_case identifiers match:
// \brunning\b never fires on `is_running`, because the underscore before it is
// a word character. That miss made this check silently useless -- it passed a
// file whose liveness function was named exactly the way people name them.
// The letter-only boundary still rejects `delivered`, where `live` is preceded
// by a letter.
var livenessRe = regexp.MustCompile(`(?i)(?:^|[^a-z])(alive|live|running|busy|stall|active|dead)(?:[^a-z]|$)`)

// L3: a journaled store read without acknowledging its sidecar.
var storeRe = regexp.MustCompile(`\.db\b|\.sqlite\b`)
var sidecarRe = regexp.MustCompile(`-wal|\.wal\b|-journal|\.journal\b|WAL`)

// L5: a progress track made of dots reads as truncated output, not as 0%.
var dotTrackRe = regexp.MustCompile(`tr ' ' '\.'|["']\.["']\s*\*\s*|\.{6,}`)

// L9: the noise floor, rendered.
var nullBandRe = regexp.MustCompile(`(?i)null_band|noise_floor|NOISE_FLOOR`)

const proximity = 6 // lines, for the L3 mtime/liveness join

func lintViewer(path string) (*Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	rep := &Report{Target: path}
	base := filepath.Base(path)

	// A file that IS the teardown is exempt from L1 wholesale -- it is
	// supposed to kill things. It must still never delete results (L15).
	teardown := strings.Contains(base, "stop") || strings.Contains(base, "sweep") ||
		strings.Contains(base, "teardown")

	var sawStore, sawSidecar, sawNullBand, sawParse bool

	for i, ln := range lines {
		n := i + 1
		trimmed := strings.TrimSpace(ln)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") && !strings.Contains(trimmed, "watchctl:") {
			// Comments cannot mutate anything; skip them for L1 so a
			// comment explaining a past bug is not itself a finding.
			if !strings.HasPrefix(trimmed, "#") {
				continue
			}
		}
		isComment := strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//")

		// ---- L1 ----------------------------------------------------
		if !teardown && !isComment {
			for _, m := range mutators {
				if m.re.MatchString(ln) && !allowed(ln, "L1") {
					rep.add(Finding{
						Law: "L1", Severity: Error, File: path, Line: n,
						Message: fmt.Sprintf("viewer performs a mutating operation (%s). "+
							"A viewer reads; the teardown is a separate program (L15). "+
							"If this is genuinely safe, add: watchctl:allow L1 <reason>", m.what),
						Evidence: trimmed,
					})
					break
				}
			}
		}
		if osKillRe.MatchString(ln) && !isComment {
			if !windowMatches(lines, i, 25, posixGuardRe) && !allowed(ln, "L1") {
				rep.add(Finding{
					Law: "L1", Severity: Error, File: path, Line: n,
					Message: "os.kill() with no platform guard nearby. On Windows " +
						"signal.CTRL_C_EVENT == 0, so a signal-0 'probe' delivers a real " +
						"Ctrl-C to the target's console group. Guard on os.name or decline to answer.",
					Evidence: trimmed,
				})
			}
		}

		// ---- L2 ----------------------------------------------------
		if parseRe.MatchString(ln) && !isComment {
			sawParse = true
			if !windowMatches(lines, i, 8, guardRe) && !allowed(ln, "L2") {
				rep.add(Finding{
					Law: "L2", Severity: Warn, File: path, Line: n,
					Message: "parse with no try/except in range. Every file a viewer reads is " +
						"being appended to concurrently; a half-written final line is normal " +
						"and must be skipped, not raised.",
					Evidence: trimmed,
				})
			}
		}

		// ---- L3 ----------------------------------------------------
		if mtimeRe.MatchString(ln) && !isComment {
			if windowMatches(lines, i, proximity, livenessRe) && !allowed(ln, "L3") {
				rep.add(Finding{
					Law: "L3", Severity: Error, File: path, Line: n,
					Message: "liveness appears to be inferred from file mtime. This has " +
						"produced a distinct false-liveness bug in every generation: killed " +
						"runs shown as live for minutes, and a stash rewriting every archive's " +
						"end time. Probe the process table, or read a timestamp inside the file.",
					Evidence: trimmed,
				})
			}
		}
		if storeRe.MatchString(ln) && !isComment {
			sawStore = true
		}
		if sidecarRe.MatchString(ln) {
			sawSidecar = true
		}

		// ---- L5 ----------------------------------------------------
		if dotTrackRe.MatchString(ln) && !isComment && !allowed(ln, "L5") {
			rep.add(Finding{
				Law: "L5", Severity: Warn, File: path, Line: n,
				Message: "a progress track of dots reads as truncated output rather than 0%. " +
					"Bracket the bar and pick a track character that cannot be an ellipsis.",
				Evidence: trimmed,
			})
		}

		// ---- L9 ----------------------------------------------------
		if nullBandRe.MatchString(ln) {
			sawNullBand = true
		}
	}

	// ---- file-level findings ------------------------------------------
	if sawStore && !sawSidecar {
		rep.add(Finding{
			Law: "L3", Severity: Warn, File: path, Line: 0,
			Message: "reads a database file but never mentions a WAL or journal sidecar. " +
				"In WAL mode the main file's mtime only moves at checkpoints -- measured at " +
				"282s stale against a 29s-fresh WAL, which showed a working worker as hung.",
		})
	}
	if sawParse && !sawNullBand {
		rep.add(Finding{
			Law: "L9", Severity: Warn, File: path, Line: 0,
			Message: "no noise floor declared. A difference smaller than run-to-run variation " +
				"looks exactly like a finding. Declare NULL_BAND and render below it dim, or " +
				"render the band as UNKNOWN if it has never been measured.",
		})
	}
	return rep, nil
}

// windowMatches reports whether re matches within n lines either side of i.
func windowMatches(lines []string, i, n int, re *regexp.Regexp) bool {
	lo, hi := i-n, i+n
	if lo < 0 {
		lo = 0
	}
	if hi >= len(lines) {
		hi = len(lines) - 1
	}
	for j := lo; j <= hi; j++ {
		if re.MatchString(lines[j]) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- plan

var (
	planEvidenceRe = regexp.MustCompile(`(?i)^#+\s*evidence`)
	planWrittenRe  = regexp.MustCompile(`(?i)written\s+when`)
	planCalloutRe  = regexp.MustCompile(`(?i)^#+\s*callout`)
	planFloorRe    = regexp.MustCompile(`(?i)null_band|noise floor`)
	planUnknownRe  = regexp.MustCompile(`(?i)\bUNKNOWN\b`)
	planQuestionRe = regexp.MustCompile(`(?i)^#+\s*(the )?question|ranks? by|ranking axis`)
	tableRowRe     = regexp.MustCompile(`^\|.*\|$`)
	sepRowRe       = regexp.MustCompile(`^\|[\s\-:|]+\|$`)
)

func checkPlan(path string) (*Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	rep := &Report{Target: path}

	var hasEvidence, hasWritten, hasCallouts, hasFloor, hasQuestion bool
	inCallouts := false
	calloutRows, unexplained := 0, 0

	for i, ln := range lines {
		switch {
		case planEvidenceRe.MatchString(ln):
			hasEvidence = true
			inCallouts = false
		case planCalloutRe.MatchString(ln):
			hasCallouts, inCallouts = true, true
		case strings.HasPrefix(ln, "#"):
			inCallouts = false
		}
		if planWrittenRe.MatchString(ln) {
			hasWritten = true
		}
		if planFloorRe.MatchString(ln) {
			hasFloor = true
		}
		if planQuestionRe.MatchString(ln) {
			hasQuestion = true
		}

		// A callout row whose final cell is empty is a guess wearing a
		// rule's clothes. The "already changed" column is what stops the
		// rule being deleted by someone who does not know what it cost.
		if inCallouts && tableRowRe.MatchString(ln) && !sepRowRe.MatchString(ln) {
			cells := splitRow(ln)
			if len(cells) >= 2 && !strings.EqualFold(cells[0], "name") {
				calloutRows++
				last := strings.TrimSpace(cells[len(cells)-1])
				if last == "" || last == "-" || last == "—" || last == "?" {
					unexplained++
					rep.add(Finding{
						Law: "L11", Severity: Warn, File: path, Line: i + 1,
						Message: "callout has no recorded incident. Either name the run it " +
							"already changed, or mark it speculative explicitly.",
						Evidence: strings.TrimSpace(ln),
					})
				}
			}
		}
	}

	if !hasEvidence {
		rep.add(Finding{Law: "P1", Severity: Error, File: path,
			Message: "no Evidence section. Phase 1 must inventory what the job already " +
				"leaves on disk before any renderer is designed."})
	}
	if hasEvidence && !hasWritten {
		rep.add(Finding{Law: "P1", Severity: Error, File: path,
			Message: "the evidence table has no 'written when' column. That column is what " +
				"decides whether a live view is needed at all: anything written only on " +
				"completion cannot show work in flight."})
	}
	if !hasQuestion {
		rep.add(Finding{Law: "L6", Severity: Error, File: path,
			Message: "no question or ranking axis recorded. A screen built before the " +
				"question is a screen organized around whatever was easiest to read."})
	}
	if !hasCallouts {
		rep.add(Finding{Law: "L11", Severity: Warn, File: path,
			Message: "no Callouts section. If nothing has silently changed a run yet, say so " +
				"explicitly -- an empty section is a claim, a missing one is an omission."})
	}
	if !hasFloor {
		rep.add(Finding{Law: "L9", Severity: Warn, File: path,
			Message: "no noise floor recorded. State the measured value, or UNKNOWN. Do not " +
				"leave it unstated -- an unstated floor becomes an invented one."})
	} else if planUnknownRe.MatchString(string(raw)) {
		// Explicit UNKNOWN is correct behaviour, not a finding.
		_ = calloutRows
	}
	return rep, nil
}

func splitRow(ln string) []string {
	s := strings.Trim(strings.TrimSpace(ln), "|")
	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// ---------------------------------------------------------------- evidence

type Source struct {
	Kind        string   `json:"kind"`
	Path        string   `json:"path"`
	Sidecars    []string `json:"sidecars,omitempty"`
	WrittenWhen string   `json:"written_when"`
	Note        string   `json:"note,omitempty"`
}

type Inventory struct {
	Root    string   `json:"root"`
	Sources []Source `json:"sources"`
	Gaps    []string `json:"gaps"`
}

// classify maps a filename onto the evidence kinds Phase 1 asks about. The
// "written when" column is the one that matters: it decides whether Phase 4
// is needed at all.
func classify(name string) (kind, when, note string, ok bool) {
	l := strings.ToLower(name)
	switch {
	case strings.HasSuffix(l, ".jsonl") && (strings.Contains(l, "result") || strings.Contains(l, "verdict")):
		return "result", "on completion only",
			"cannot show work in flight -- this is the Phase 4 trigger", true
	case strings.HasSuffix(l, ".jsonl"), strings.Contains(l, "stdout"), strings.Contains(l, "stream"):
		return "stream", "continuously, appended", "tolerate a torn final line (L2)", true
	case strings.HasSuffix(l, ".db"), strings.HasSuffix(l, ".sqlite"):
		return "store", "continuously", "check for a WAL sidecar before trusting mtime (L3)", true
	case strings.HasSuffix(l, ".log"):
		return "log", "continuously, appended", "read timestamps inside, not mtime (L3)", true
	case strings.HasPrefix(l, ".run"), strings.Contains(l, "marker"), strings.Contains(l, "manifest"):
		return "marker", "at start", "carries identity; the anchor for L7", true
	}
	return "", "", "", false
}

func scanEvidence(root string) (*Inventory, error) {
	inv := &Inventory{Root: root, Sources: []Source{}, Gaps: []string{}}
	seen := map[string]*Source{}
	var order []string

	err := filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil // unreadable subtree is a gap, not a failure
		}
		if fi.IsDir() {
			if strings.HasPrefix(fi.Name(), ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		name := fi.Name()
		// Sidecars attach to their principal rather than standing alone.
		if sidecarRe.MatchString(name) {
			principal := sidecarRe.ReplaceAllString(p, "")
			if s, ok := seen[principal]; ok {
				s.Sidecars = append(s.Sidecars, filepath.Base(p))
			}
			return nil
		}
		kind, when, note, ok := classify(name)
		if !ok {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		s := &Source{Kind: kind, Path: rel, WrittenWhen: when, Note: note}
		seen[p] = s
		order = append(order, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(order)
	for _, p := range order {
		inv.Sources = append(inv.Sources, *seen[p])
	}

	kinds := map[string]bool{}
	for _, s := range inv.Sources {
		kinds[s.Kind] = true
		if s.Kind == "store" && len(s.Sidecars) > 0 {
			// Recorded so the plan cannot omit it.
			inv.Gaps = append(inv.Gaps,
				"store "+s.Path+" has a journal sidecar: freshness is max(mtime) across siblings (L3)")
		}
	}
	if !kinds["marker"] {
		inv.Gaps = append(inv.Gaps,
			"no marker file found: nothing on disk names the unit a working directory belongs to. "+
				"Add one to the RUNNER (it is part of the run's own record), never to the viewer.")
	}
	if !kinds["stream"] && !kinds["log"] {
		inv.Gaps = append(inv.Gaps,
			"no continuously-written evidence found: a live view may have nothing to show. "+
				"Confirm before entering Phase 4.")
	}
	if kinds["result"] && !kinds["stream"] {
		inv.Gaps = append(inv.Gaps,
			"results are written only on completion and nothing streams: the screen will sit "+
				"empty while workers are busy. This is the Phase 4 trigger.")
	}
	return inv, nil
}

// ---------------------------------------------------------------- output

func printReport(rep *Report, asJSON bool) int {
	if asJSON {
		b, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(b))
	} else if len(rep.Findings) == 0 {
		fmt.Printf("%s: clean\n", rep.Target)
	} else {
		for _, f := range rep.Findings {
			loc := rep.Target
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", rep.Target, f.Line)
			}
			fmt.Printf("%s [%s] %s: %s\n", loc, f.Severity, f.Law, f.Message)
			if f.Evidence != "" {
				fmt.Printf("    %s\n", f.Evidence)
			}
		}
		fmt.Printf("\n%d finding(s)\n", len(rep.Findings))
	}
	if len(rep.Findings) > 0 && rep.worst() == Error {
		return 1
	}
	if len(rep.Findings) > 0 {
		return 1
	}
	return 0
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	var root, viewer, file string
	var asJSON bool
	fs.StringVar(&root, "root", "", "output root to inventory")
	fs.StringVar(&viewer, "viewer", "", "viewer script or module to lint")
	fs.StringVar(&file, "file", "", "plan file to check")
	fs.BoolVar(&asJSON, "json", false, "emit JSON")
	_ = fs.Parse(os.Args[2:])

	switch cmd {
	case "evidence":
		if root == "" {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
		inv, err := scanEvidence(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "watchctl:", err)
			os.Exit(2)
		}
		if asJSON {
			b, _ := json.MarshalIndent(inv, "", "  ")
			fmt.Println(string(b))
		} else {
			fmt.Printf("%-8s %-40s %s\n", "kind", "path", "written when")
			for _, s := range inv.Sources {
				fmt.Printf("%-8s %-40s %s\n", s.Kind, s.Path, s.WrittenWhen)
			}
			for _, g := range inv.Gaps {
				fmt.Printf("\n  gap: %s\n", g)
			}
		}
	case "lint":
		if viewer == "" {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
		rep, err := lintViewer(viewer)
		if err != nil {
			fmt.Fprintln(os.Stderr, "watchctl:", err)
			os.Exit(2)
		}
		os.Exit(printReport(rep, asJSON))
	case "plan":
		if file == "" {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
		rep, err := checkPlan(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "watchctl:", err)
			os.Exit(2)
		}
		os.Exit(printReport(rep, asJSON))
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}
