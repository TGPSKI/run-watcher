# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-08-04

### Added

- `PATTERN.md` — the run-watcher pattern: the claim, the three roles, the four
  phases, and what the pattern refuses.
- `SKILL.md` — multi-phase router with a decline gate and two stop conditions.
- `references/design-laws.md` — L1–L17, each stated with the incident that
  produced it.
- `references/anomaly-catalogue.md` — shape → likely cause → cheapest
  confirming check.
- `references/phase-01..04` — evidence and question, the watcher, the browser,
  the live view.
- `scripts/watchctl.go` — deterministic, read-only linter and inventory tool.
  Subcommands `evidence`, `lint`, `plan`. Checks the mechanically-checkable
  subset of the laws; refuses to guess at the rest.
- `examples/sig-triage` — generations 1 and 1b, with Phase 4 deliberately
  declined.
- `examples/adherence-suite` — generation 3, the full path.
- `LINEAGE.md` — where each law came from, with dates and incidents.
