SHELL := /bin/bash
.DEFAULT_GOAL := help

SCRIPTS_DIR := scripts
CTL         := go run $(SCRIPTS_DIR)/watchctl.go
EXAMPLES    := sig-triage adherence-suite

# --- Quality ---

.PHONY: test
test: ## Run Go tests for watchctl (law checks, plan checks, evidence scan)
	cd $(SCRIPTS_DIR) && go test -v -count=1 ./...

.PHONY: lint
lint: ## Vet Go code and check every phase file declares its parent router
	cd $(SCRIPTS_DIR) && go vet ./...
	@for f in references/phase-*.md references/design-laws.md references/anomaly-catalogue.md; do \
		grep -q '^parent: run-watcher$$' "$$f" || { echo "$$f: missing 'parent: run-watcher'"; exit 1; }; \
	done; echo "phase files: parent declared"
	@for f in $$(grep -oE 'references/[a-z0-9-]+\.md' SKILL.md | sort -u); do \
		test -f "$$f" || { echo "SKILL.md references missing file: $$f"; exit 1; }; \
	done; echo "router references: resolve"

.PHONY: check-assets
check-assets: ## The vendored drawing layer must import cleanly and stay stdlib-only
	@# Import, not just parse. charts.py reaches windows.py through a LAZY
	@# import inside a branch, so anything that only reads the top of a file
	@# calls windows.py unused -- and dropping it plants an ImportError that
	@# fires the first time a caller passes stacked segments.
	@PYTHONPATH=assets python3 -c "from tui import charts, fmt, framework, windows" \
		|| { echo 'assets/tui: does not import as a package'; exit 1; }
	@PYTHONPATH=assets python3 -c "from tui.windows import stack_cells" \
		|| { echo 'assets/tui: charts.py needs windows.stack_cells'; exit 1; }
	@bad="$$(grep -hoE '^[[:space:]]*(import|from) [a-z_.]+' assets/tui/*.py \
		| awk '{print $$2}' | sort -u \
		| grep -vE '^(csv|curses|math|datetime|pathlib|__future__|\.fmt|\.windows)$$')"; \
	if [ -n "$$bad" ]; then \
		echo "assets/tui: non-stdlib import(s): $$bad"; \
		echo "  the drawing layer must stay dependency-free"; exit 1; \
	fi
	@echo "assets/tui: imports cleanly, stdlib-only"

.PHONY: fmt
fmt: ## Format Go code
	cd $(SCRIPTS_DIR) && gofmt -w .

.PHONY: fmt-check
fmt-check: ## Check Go formatting (CI)
	@cd $(SCRIPTS_DIR) && test -z "$$(gofmt -l .)" || (echo "Go files need formatting:"; gofmt -l .; exit 1)

.PHONY: check
check: lint fmt-check test check-assets check-examples ## Full CI: lint + fmt-check + test + golden validation

# --- Golden examples ---

.PHONY: check-examples
check-examples: ## Validate both golden Phase 1 plans
	@for e in $(EXAMPLES); do \
		echo "=== $$e ==="; \
		$(CTL) plan --file examples/$$e/WATCHING.md || exit 1; \
	done

.PHONY: regen-findings
regen-findings: ## Regenerate recorded findings from local sibling clones: make regen-findings SIBLINGS=../
	@if [ -z "$(SIBLINGS)" ]; then echo "Usage: make regen-findings SIBLINGS=/path/containing/clones"; exit 1; fi
	@$(CTL) lint --viewer $(SIBLINGS)/leather/examples/14-sig-triage/eval/scripts/watch-matrix.sh \
		> examples/sig-triage/findings.txt 2>&1 || true
	@$(CTL) lint --viewer $(SIBLINGS)/adherence-suite/src/adherence/live.py \
		> examples/adherence-suite/findings.txt 2>&1 || true
	@echo "regenerated findings for: $(EXAMPLES)"

# --- Tool wrappers ---

.PHONY: evidence
evidence: ## Inventory a job's on-disk evidence: make evidence ROOT=/tmp
	@if [ -z "$(ROOT)" ]; then echo "Usage: make evidence ROOT=/path/to/output/root"; exit 1; fi
	$(CTL) evidence --root $(ROOT)

.PHONY: lint-viewer
lint-viewer: ## Check a viewer against the design laws: make lint-viewer VIEWER=path/to/watch.sh
	@if [ -z "$(VIEWER)" ]; then echo "Usage: make lint-viewer VIEWER=path/to/viewer"; exit 1; fi
	$(CTL) lint --viewer $(VIEWER)

.PHONY: check-plan
check-plan: ## Check a Phase 1 plan is complete: make check-plan PLAN=docs/WATCHING.md
	@if [ -z "$(PLAN)" ]; then echo "Usage: make check-plan PLAN=path/to/WATCHING.md"; exit 1; fi
	$(CTL) plan --file $(PLAN)

# --- Help ---

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
