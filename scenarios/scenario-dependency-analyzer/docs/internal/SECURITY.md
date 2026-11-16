# Security

## Purpose Of This Document

Capture SDA security posture and gaps.

## Data Sensitivity

SDA processes repository metadata, manifests, imports, and generated graph evidence.

## Auth And Authorization

Local lifecycle deployment currently relies on local operator trust boundaries.

## Secrets

Do not persist source contents or secrets in graph evidence. Secret-like test fixtures must be clearly marked as examples.

## Threat Model

Primary risks are stale dependency evidence, poisoned local manifests, and accidental secret exposure in generated artifacts.

## Security Gaps

Dependency vulnerability findings are reported by the scenario test security phase.
As of 2026-06-14, the blocking dependency issues are resolved:

- API and CLI Go modules pin `go 1.25.11`, with local `GOTOOLCHAIN=auto` so the patched toolchain is selected for SDA builds.
- API dependencies use patched `github.com/rs/cors`, `golang.org/x/net`, `golang.org/x/crypto`, and `golang.org/x/sys` versions.
- UI dev tooling uses patched Vite-era dependencies; `pnpm audit --audit-level moderate` reports no known vulnerabilities.
- `security-health validate scenario scenario-dependency-analyzer` passes with no error or warning findings.

The remaining security-health entries are info-level only. Current notable classes are local gitignored fingerprint placeholders, test-helper ignored-error observations, and OSV matches inside nested package-manager fixture lockfiles under `node_modules`.

## Hardening Notes

The API server sets explicit read-header, read, write, and idle timeouts. Generated/exported local artifacts use private file modes by default. Scanner paths skip symlink entries before reading repository files, and reviewed gosec suppressions document root-constrained file access and fixed-executable subprocess calls.

Dependency installation is a governed mutation boundary. JavaScript additions
and lockfile refreshes pass `--ignore-scripts`; frozen reproduction is exposed
as a separate command contract (`npm ci`, frozen pnpm/Bun, immutable Yarn,
locked Cargo, and equivalent Go/Python/Poetry profiles). Resolution records the
profile so callers can distinguish lockfile reproduction from a manifest
mutation. Protected build exceptions are explicit, owner-attributed, and
limited to guided, guarded, or enforcing policy modes.

Install admission is one routine for dry runs and applies: a preview plans
exactly what `--apply` will reserve, so "approved" means the install can run.
A request that governance refuses is never planned.

pnpm and npm installs are resolved by the package manager itself, in a
throwaway copy of the surface's `package.json`, lockfile and workspace policy
(`pnpm add --lockfile-only`, `npm install --package-lock-only`), with lifecycle
scripts off, the public registry pinned, the seven-day release-age floor
applied and registry metadata isolated in the staging directory. Admission
sizes only the locked packages that node_modules does not already hold: a
registry package from its published metadata and archive size, a local
`file:` package from its own source files (it must be a real directory beneath
`packages/`), and a tarball URL dependency by downloading it once into memory,
checking the bytes against the lockfile's sha512 integrity, and measuring the
unpacked size. A tarball URL the request did not name is refused. Apply
publishes the staged bytes that admission measured (refusing if the surface
changed since), links node_modules with a frozen-lockfile install, and on any
failure restores the previous inputs and relinks. The pnpm store is whatever
`pnpm store path` reports in the surface, which is the store the scenario
lifecycle uses; a surface linked against another store (an older pnpm major)
gets its manifest and lockfile updated and node_modules left for the
lifecycle, because pnpm would otherwise recreate it.

Go install admission probes the pinned `.mod` and `.zip` artifacts from the
official module proxy, counts compressed and expanded bytes without writing
them during planning, and rejects custom proxy/checksum/workspace settings and
closure limits. A package of a repository module is used in place and adds no
source; the external modules its import closure needs are named by
`go list -deps` run offline (`GOPROXY=off`, `-mod=readonly`, `GOTOOLCHAIN=local`)
in the owning module, and sized like a published module. A published module is sized from
the requirement closure it would add: modules the target `go.mod` already
requires at the same or a newer version, and that the module cache already
holds, cost nothing. An install must name an exact version, and the refusal
names the proxy's latest release when it does not. The executor pins the proxy,
checksum database, local toolchain and cache paths used by the probe, and a
failed `go get` restores `go.mod` and `go.sum`. The planner reserves the managed
module cache, temporary space, and a policy-bounded build cache; the
build-cache peak is a reservation bound, not a measured runtime peak. A
successful size plan still requires a fresh per-operation shared capacity
lease before any package-manager write.

Staging directories (`.sda-stage-*`, beside the surface or in the temp
directory for an external workspace) are removed when the request finishes;
any older than an hour belongs to a process that died and is swept by the next
staging in the same directory.

## Cross-References

- `../operations/RUNBOOK.md`
- `ERROR-HANDLING.md`
