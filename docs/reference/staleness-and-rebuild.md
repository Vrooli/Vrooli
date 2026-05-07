# Staleness detection and self-rebuild — `internal/buildinfo`

`vrooli` and `vrooli-api` self-detect when their compiled binary is older
than the source tree they live in, and re-exec into a freshly built version.
This document is the reference for that flow: env vars, on-disk artifacts,
concurrency contract, and the autoheal-specific opt-out.

## Source of truth

- `internal/buildinfo/buildinfo.go` — fingerprint computation, `CheckStaleness`, `RebuildAndReexec`.
- `internal/cli/rootcli/rootcli.go` — wraps `CheckStaleness` + `RebuildAndReexec` for the CLI entry path; surfaces `--no-stale-check`.

## Fingerprint

A SHA-256 over `<rel> <size> <sha256-hex>` lines for every `.go` file under
the targets returned by `fingerprintTargets()`:

| Binary | Targets |
|---|---|
| `vrooli` | `cmd/vrooli`, `internal` |
| `vrooli-api` | `cmd/vrooli-api`, `internal` |

The fingerprint is embedded in the binary at build time via
`-ldflags -X github.com/vrooli/vrooli/internal/buildinfo.Fingerprint=<hex>`.

## Environment variables

| Variable | Purpose |
|---|---|
| `VROOLI_SOURCE_ROOT` | Repository root used for fingerprinting. Defaults to module-root discovery. |
| `VROOLI_ROOT` | Fallback when `VROOLI_SOURCE_ROOT` is unset. |
| `VROOLI_FINGERPRINT_PATHS` | Comma-separated relative paths overriding the default target set. |
| `VROOLI_BUILD_TARGET` | Override `./cmd/<name>` build target used by `RebuildAndReexec`. |
| `VROOLI_REBUILD_FINGERPRINT` | Set automatically on re-exec; trips the loop guard if the same fingerprint asks for a second rebuild in the same chain. |
| `VROOLI_FINGERPRINT_DEBUG` | When truthy (anything except empty / `0` / `false` / `no` / `off`), dump a sorted `<rel> <size> <sha256>` line per fingerprint input to stderr. Diagnostic only. |

## On-disk artifacts (next to the executable)

| File | Purpose |
|---|---|
| `<executable>.lock` | flock(2) target serializing concurrent rebuilders. Auto-released on process death. |
| `<executable>.fp` | Sidecar fingerprint cache; lets sibling processes recognize a freshly rebuilt binary even when their own embedded symbol still reflects the pre-rebuild value. |
| `<executable>.tmp.<pid>` | Transient build output; `os.Rename`d into place atomically. |

### Sidecar precedence

`CheckStaleness` consults the sidecar first:
- present **and** matches current source fingerprint **and** `sidecar.mtime ≥ binary.mtime` → fresh.
- otherwise → fall through to the embedded-fingerprint compare (authoritative).

A developer who runs `make build` (which doesn't touch the sidecar) is
covered by the mtime check: a newer binary mtime invalidates an older
sidecar, and the embedded compare drives the result.

## Rebuild concurrency contract

`RebuildAndReexec`:

1. Compute current fingerprint; if it equals `VROOLI_REBUILD_FINGERPRINT`, return the rebuild-loop error.
2. Acquire `<executable>.lock` via `LOCK_EX` flock. Linux-only.
3. Re-check: if `Fingerprint` (embedded) already matches current source, skip the build and exec straight into the now-fresh binary.
4. Run `go build -o <executable>.tmp.<pid> <buildTarget>` with `-ldflags` injecting the new fingerprint, git commit, and build time.
5. `os.Rename(<tempfile>, <executable>)` — atomic on the same filesystem.
6. Best-effort write `<executable>.fp` (atomic via temp + rename).
7. Release flock; `syscall.Exec` re-execs with `VROOLI_REBUILD_FINGERPRINT=<current>`.

Build failure removes the temp file and returns; the lock is released by the deferred close.

## `--no-stale-check`

Every `vrooli` subcommand accepts `--no-stale-check`, which short-circuits
the staleness check and the rebuild path. Use it for ad-hoc invocations
where you don't want a stale source tree to trigger a rebuild.

### Autoheal opt-out (always on)

`scenarios/vrooli-autoheal` runs many `vrooli` subprocesses concurrently
against the user's working tree. To prevent every check from entering the
rebuild path (and contending on `<executable>` even with the flock),
`scenarios/vrooli-autoheal/api/internal/checks/executor.go` injects
`--no-stale-check` for every `vrooli` invocation. The injection is
idempotent and lives at the single `RealExecutor` seam that all 27+
autoheal call sites traverse.

## Diagnosing rebuild loops

1. `VROOLI_FINGERPRINT_DEBUG=1 vrooli scenario status <name> 2> /tmp/fp.txt` — dumps the per-file inputs that fed the fingerprint.
2. Compare two such dumps from different invocations to localize which file actually changed; the fingerprint header line includes the resulting hex.
3. If the dumps are identical but the binary still reports stale, suspect a stale embedded symbol (no sidecar yet) — manually `make build` to refresh, or rely on the next successful `RebuildAndReexec` to drop a sidecar.

## Tests

- `internal/buildinfo/buildinfo_test.go` — fingerprint determinism, target validation, debug dump on/off, flock serialization, atomic rename, lock release on build failure, sidecar precedence and mtime gating, sidecar write on success.
