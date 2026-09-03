# Scenario freshness and rebuilds

Vrooli uses the manifest freshness engine in `packages/cli-core/cliutil` for
scenario artifacts. The engine evaluates declared inputs by file content and
the stat cache; it does not use source-versus-artifact mtime as a freshness
verdict and it does not expose a stale-check bypass flag.

The retired `--no-stale-check` global is tolerated until 2026-12-01: the root
parser consumes it, prints one warning to stderr, and dispatches the command
unchanged. The tolerance table lives in `internal/cli/rootcli/rootcli.go`
(`retiredGlobals`). Processes that spawn `vrooli` must not rely on it; the
invoker registry test in `internal/cli/rootcli/invokers` fails on any
registered argv that still carries a retired global.

## Freshness contract

An artifact is stale when it is missing, has no manifest, or its manifest no
longer matches the declared inputs. A missing manifest is therefore a
one-time build condition. Successful builds stamp only the component that was
built, allowing API, UI, and other components to rebuild independently.

Input enumeration uses `git ls-files --cached --others --exclude-standard`
inside a work tree and a `WalkDir` fallback outside one. Git supplies the file
set only; the verdict remains content-based, so committing a change cannot by
itself make an artifact fresh or stale.

## Lifecycle behavior

Builder specifications in `internal/lifecycle/components.go` declare inputs,
skip suffixes, digest keys, closure resolution, outputs, and commands. The
generic lifecycle evaluator uses those fields for Go, Node, and Python/uv
components. Fresh components are skipped unless setup is explicitly forced;
independent stale components may build concurrently within the memory budget.

Proto generation is invoked through the direct `protogen` command and receives
the changed schema scope. `make` is not required on the scenario start path.

## Diagnostics

Use lifecycle logs and component freshness verdicts to identify the content
cause. The rebuild-ledger evidence folder contains the retained baseline,
focused regression proofs, and validation commands:
`docs/architecture/evidence/rebuild-ledger/README.md`.

For the next investigation, `vrooli scenario timings --since <date>` reports
the setup and build timing window without reintroducing a second freshness
implementation.
