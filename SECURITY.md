# Security Policy

## Reporting

Report vulnerabilities through [GitHub Security Advisories](https://github.com/TGPSKI/run-watcher/security/advisories/new).
Do not open a public issue for a security report.

## Response timeline

| Stage | Target |
|---|---|
| Acknowledgment | 3 business days |
| Initial assessment | 10 business days |
| Fix or documented mitigation | 30 days for high severity |

## Scope

This repository ships a **pattern specification, a directed workflow, and a
read-only linter**. It runs no service and stores no data. Its attack surface is
therefore mostly *what it tells other people to build*.

Security issues here include:

- **Guidance that produces an unsafe viewer.** The pattern's first law is that
  the observer must never be able to affect the observed. Any phase instruction,
  template, or example that would lead an implementer to generate a viewer
  capable of signalling, writing to, locking, or killing the job it watches is a
  security defect in this repository, not merely a bug.
- **`watchctl` performing any write, execution, or network access.** The tool
  reads files and prints findings. It must never execute a script it is linting,
  follow a symlink out of the tree it was pointed at, or open a socket. A linter
  that violated its own first law would be worthless.
- **A law check that produces a false *pass* on a genuinely dangerous
  construct.** A missed mutation in a viewer is more serious than a false
  positive, because the pragma system means a clean report is read as an
  assurance.
- **Pragma bypass.** `watchctl:allow` requires a reason by design. A way to
  suppress a law without recording why is a defect.
- **Guidance that leaks secrets onto a screen.** A viewer renders whatever the
  job wrote. Phase instructions that would surface credentials, tokens, or
  personal data into a terminal — or into a pasted `NOCOLOR=1` snapshot — are in
  scope.

Not in scope:

- Vulnerabilities in a watcher *you* generated. Those belong to your repository.
- The example source repositories ([leather](https://github.com/TGPSKI/leather),
  [adherence-suite](https://github.com/TGPSKI/adherence-suite)). Report those to
  the repositories themselves.

## Known limitations

**The linter is a deliberate subset.** L6, L11, L16 and L17 are judgment, not
syntax, and `watchctl` does not check them. A clean `watchctl lint` means the
mechanically-checkable laws hold — it is not a certification that a viewer is
safe. This is stated in the tool's own documentation and must stay stated.

**Pragmas are honoured on trust.** `watchctl:allow L1 <reason>` suppresses a
finding on the strength of a human-written justification. The reason is
mandatory so it can be reviewed, but nothing verifies it.

## Supported versions

The latest tagged release. Fixes are not backported.
