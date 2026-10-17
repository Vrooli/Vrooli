# Workspace Sandbox Cross-Platform Readiness Audit

## Last Updated
2026-07-14

## Scope

This audit covers two portability concerns that used to be conflated:

1. **Storage portability** — the DB and file layout (previously the whole
   of this document; still tracked below).
2. **OS-seam portability** — the driver, containment, process, and mount
   layers that make the API build and run beyond Linux. This is the focus
   of the cross-platform-drivers work (docs/AXES.md axis 1).

Every status below is backed by a command that proves it. No claim is made
that a command has not verified.

## Verified Compile Status (OS seam)

The api module builds CGO-free for all three OS-seam targets. Verified via
`make cross-compile` and the equivalent explicit builds on the Linux dev
host (2026-07-14):

| Target | Command | Result |
|--------|---------|--------|
| linux/amd64 (native) | `CGO_ENABLED=0 go build ./...` | ✅ builds |
| darwin/arm64 | `GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 GOWORK=off go build ./...` | ✅ builds |
| windows/amd64 | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOWORK=off go build ./...` | ✅ builds |

These are enforced permanently, not just checked once:

- `make cross-compile` (Makefile target) runs the darwin + windows builds.
- `make check` includes `cross-compile` in the quality gates.
- `api/internal/crosscompile/crosscompile_test.go` (`TestCrossCompile`) is a
  go-test gate that runs the same two cross-builds, so a Linux-only syscall
  leaking outside a `//go:build`-tagged file fails `go test ./...`.

The OS seam is split across build-tagged files so no Linux-only syscall
reaches a non-Linux build:

| Seam | Linux file | Non-Linux file(s) |
|------|-----------|-------------------|
| Filesystem mount | `internal/fsmount/mount_linux.go` | `internal/fsmount/mount_other.go` |
| User namespace | `internal/namespace/kernel_linux.go` | `internal/namespace/kernel_other.go` |
| Change detection whiteouts | `internal/driver/changedetect/whiteout_unix.go` | `internal/driver/changedetect/whiteout_windows.go` |
| Process signals | `internal/process/signals_unix.go` | `internal/process/signals_windows.go` |
| Process attributes | `internal/process/procattr_unix.go` | `internal/process/procattr_windows.go` |
| Containment probe | `internal/driver/containment_linux.go` | `internal/driver/containment_darwin.go`, `internal/driver/containment_other.go` |
| Exec containment backend | `internal/driver/exec/backend_linux.go` | `internal/driver/exec/backend_darwin.go`, `internal/driver/exec/backend_other.go` |

## Per-OS Driver Matrix

Drivers are `api/internal/driver`; the boot-time selection is
`driver.SelectDriver` / `SelectDriverWithPreference` (priority order:
overlayfs-userns → overlayfs-root → fuse-overlayfs → copy). The `copy`
driver is always available and is the fallback every non-Linux host lands
on.

| Driver ID | Linux | macOS | Windows | Notes |
|-----------|:-----:|:-----:|:-------:|-------|
| `overlayfs-userns` | ✅ | ❌ | ❌ | Kernel overlayfs in a user namespace (default when available). |
| `overlayfs-root` | ✅ (root) | ❌ | ❌ | Kernel overlayfs, requires root. |
| `fuse-overlayfs` | ✅ (if installed) | ❌ | ❌ | Unprivileged, slower. |
| `copy` | ✅ | ✅ | ✅ | Cross-platform file-copy driver; no mount, no CoW. |

Copy-driver behavior is proven end-to-end on Linux (representative of the
darwin path, since copy semantics are OS-independent) by
`api/internal/handlers/copy_driver_e2e_test.go`
(`TestCopyDriverEndToEnd`): it forces the copy driver through the durable
driver preference (`driverpref` + `SelectDriverWithPreference`) and drives
create → exec → background process (+logs +structured exit) → diff (add /
modify / delete via the lower-tree walk) → approve/apply → provenance over
the production HTTP stack, asserting the identity layout throughout.

## Per-OS Containment Matrix

Containment backend selection is behind a platform-neutral seam
(`internal/driver/exec/backend*.go`); each backend reports what it enforces
via `GET /api/v1/driver/containment`. Enforcement vocabulary and the full
prose live in `docs/EXECUTION_MODES.md` (§ Per-OS Containment); the matrix
is reproduced here for the audit.

| Enforcement | Linux (`bwrap`) | macOS (`seatbelt`) | Other / copy (`none`) |
|-------------|:---------------:|:------------------:|:---------------------:|
| `filesystem-write-containment` | ✅ | ✅ | ❌ |
| `network-deny` | ✅ | ✅ | ❌ |
| `pid-namespace` | ✅ | ❌ | ❌ |
| `path-illusion` (`workspacePath == /workspace`) | ✅ | ❌ | ❌ |

- The copy driver reports `RequiredContainment() == none`, so
  `workspacePath == mergedDir` and `pathIllusion == false` (identity
  layout). Pinned by `api/internal/driver/contract_test.go`
  (`TestDriverContract_RequiredContainment`) and the E2E above.
- macOS Seatbelt is partial by design (writes + network only). Its
  containment report is unit-tested on the Linux dev host via the pure
  `seatbeltContainmentInfo` generator (`internal/driver/containment.go`,
  `seatbelt_test.go`) and the darwin probe (`containment_darwin.go`).

## Shipped vs Deferred (OS seam)

**Shipped** (verified by build/test on the Linux dev host):

- CGO-free darwin/arm64 + windows/amd64 compile (above).
- macOS Seatbelt exec backend + `rlimit-exec` self-exec shim replacing
  Linux-only `prlimit` (Phase 4; `backend_darwin.go`, `seatbelt.go`,
  `internal/rlimitexec/`).
- Windows process termination: single-process fallback. Windows has no
  POSIX process groups or signals, so `internal/process/signals_windows.go`
  reduces a group-kill target to its absolute PID and force-kills the
  single process. This is the intended Windows behavior, **not** a
  deferred item — the "process-group port" is the single-process fallback.
  Pinned by `internal/process/*_windows.go` and covered by the
  changedetect whiteout Windows test.

**Deferred** (not implemented; no command claims otherwise):

- **APFS `clonefile` acceleration** for the copy driver on macOS. The copy
  driver uses a plain recursive byte copy (`copy.go::copyDirectory`); an
  APFS copy-on-write clone would cut sandbox-create cost on macOS but is
  not implemented.
- **Real-Mac shakeout.** Everything above is verified at the
  compile/unit/Linux-E2E level. No run on physical Apple hardware has been
  performed. The operator checklist for that run is
  `docs/guides/macos-shakeout.md`; it hands off to the mac-mini
  bridge-onboarding plan. Until that checklist passes on a Mac, macOS
  support is "compiles + unit-verified", not "field-verified".

## Storage Portability (unchanged)

### Target Tiers
- [x] Tier 1 Local Stack — works (default).
- [x] Tier 2 Desktop (Electron-style bundles) — feasible as far as storage
  is concerned. Overlayfs is Linux-only; on macOS/Windows the copy driver
  is the path of least resistance (see the driver matrix above).
- [x] Tier 4 Cloud/SaaS — embedded SQLite is appropriate for a single-host
  scenario; cloud deploys scale by running multiple instances each owning
  their DB file (or a persistent volume per pod).
- [ ] Tier 3 Mobile / Tier 5 Enterprise — out of scope; the scenario
  depends on host-level filesystem mounts mobile sandboxes do not provide.

### Environment Variable Status
| Variable | Usage | Policy |
|----------|-------|--------|
| `API_PORT`, `UI_PORT` | Lifecycle-injected service ports | Accepted by the service lifecycle |
| `PROJECT_ROOT` | Root path for sandboxable directories | Accepted as an execution-scope input |
| `VROOLI_ROOT` | Source-tree discovery for integration seams | Consumed only where the shared runtime contract requires it |
| `WORKSPACE_SANDBOX_*` | Explicit feature and policy controls | Each knob is documented and validated individually |

Storage paths and the SQLite database path are service-owned. The API does
not accept application-level directory or database-path overrides, does not
read desktop-specific directory variables, and does not fall back to a
second location when an authoritative path is unavailable.

No `POSTGRES_*` or `DATABASE_URL` references remain.

### Build Status (storage)
- [x] `go build ./...` succeeds.
- [x] `CGO_ENABLED=0 go build ./...` succeeds (modernc.org/sqlite is pure Go).
- [x] No `github.com/mattn/go-sqlite3` import; no CGO directives.

### Network Status
- [x] CORS configurable (`WORKSPACE_SANDBOX_CORS_ORIGINS`).
- [x] Ports configurable via `API_PORT`/`UI_PORT`.
- [x] No fixed `:8080`/`:5432` references.

## Issues Found
- None outstanding for storage portability.
- OS-seam portability is compile- and unit-verified for darwin/windows; the
  only open item is the real-Mac field shakeout (deferred, checklist
  authored).
