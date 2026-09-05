# Targeted cross-platform validation — 2026-08-08

This record replaces the authored repository-wide baseline for this execution.
The broad baseline was started once, admitted five members, then abandoned at
the operator's direction because its admission queue and full-tree scope were
not an appropriate validation instrument. It is not being rerun or used as a
release gate.

## Local implementation evidence

- `packages/platform-go` owns process attributes, signals, process groups,
  process containment, locking, service installation, re-exec, path resolution,
  and process observation.
- Serial cross-builds passed for Linux amd64/arm64, macOS amd64/arm64, and
  Windows amd64 for the platform package.
- Serial targeted cross-builds passed for the control-plane lifecycle,
  resources, host inventory, process, runtime supervisor, and emulator
  packages on those five targets.
- Focused tests passed for platform, lifecycle, build-info, runtime,
  runtime-supervisor, resources, host inventory, process, Git Control Tower
  baseline storage, scenario-to-desktop runtime, and Test Genie execution
  persistence.
- Test Genie execution persistence now records `host_os`, `host_arch`,
  `host_node`, and `host_fact_digest`, with additive idempotent migration tests.

## Mac mini pre-change evidence

Collected over the verified effort SSH key without reading effort-store values:

- Host: `minimouse`, macOS `15.7.2`, Darwin `24.6.0`, Intel `x86_64`.
- Hardware: 6 cores, 8 GiB RAM, 201 GiB free in the home volume.
- Available: `/usr/bin/git`, `/usr/bin/curl`, passwordless `sudo`.
- Missing from PATH: Go, Node, pnpm, Homebrew.
- Partial install: 98 files under `~/.vrooli/bin`, 93 named `resource-*`; a
  225280-byte `~/.vrooli/state/runtime.db`; Bridge LaunchAgent present.
- Missing: `~/.vrooli/bin/vrooli` and `~/.vrooli/src`.

This is retained as interrupted-install-recovery evidence. Convergence was
completed through the supported Bridge path below; no repository-wide baseline
was used as a gate.

## Known execution findings

- The scenario dependency analyzer's installed CLI exposes `deps install` with
  no usable arguments and `deps reconcile --all` currently fails on malformed
  template module placeholders. This is recorded tooling friction; local module
  wiring was made explicit only where required for the seam migration.
- The agent-manager module currently has unrelated pre-existing compile errors
  in `internal/invocationreadmodel/project.go` (`int`/`int64` mismatch and an
  unused `segmentMinimums` declaration). Its targeted seam tests cannot run
  until that existing module state is repaired.
- The dependency analyzer remains a tooling gap and must be repaired before
  dependency governance can be treated as fully automated.

## Targeted Mac acceptance evidence

- Bridge operations `89100f52-8d75-4f02-b89f-a24fc584ee49`,
  `fa7e5857-e28c-474a-a789-88885c80f305`, and
  `9119401a-c1a3-4f49-ade9-e9e5cf792c42` reached durable `SUCCEEDED` status.
  The latter shipped the final working tree after the Darwin process-table
  fix. The CLI watch connection has a short client deadline and ended with
  `unexpected EOF`; durable status, not that presentation stream, was the
  acceptance authority.
- Final Bridge node projection: `minimouse`, Darwin `amd64`,
  `NODE_STATUS_ONLINE`, revision `cd3feb7c883b9559994dd4640db19b87942212fe+dirty`.
- Native Mac checks passed over the effort SSH key: `vrooli version` reported
  Vrooli CLI `v1.0.0` on Mach-O x86_64; encrypted credentials reported
  initialized; the headless LaunchDaemon was running with stdout/stderr paths
  under `~/.local/state/vrooli-bridge-agent/`.
- Explicit Mac setup checks completed for `minimal` and `development` with
  `resources none`, `scenarios none`, result files, and the protected
  credential-passphrase stdin path. No secret value was printed or persisted
  in this record.
- Native project build produced Mach-O x86_64 `vrooli`, `vrooli-api`, and
  `vrooli-policy-runner`. `vrooli develop --environment development
  --resources none --scenarios none` reached `Vrooli API healthy on port 8092`
  and intentionally skipped the orchestrator. A direct health query reported
  `healthy`, zero orphans, and zero zombies.
- Darwin port diagnostics and managed orphan cleanup passed after replacing
  the Linux-only `ps ... sid=` query with the cross-platform `sess=` form.
- Docker-dependent services were not started: Docker is absent by choice on
  this 8 GiB Mac and those resources are not part of the targeted no-resource
  development profile. Apple Silicon remains unqualified; no Apple Silicon
  hardware was available.

## Targeted validation commands

- `GOWORK=off go test ./...` in `scenarios/vrooli-bridge/api` — passed.
- `GOWORK=off go test ./...` in `scenarios/vrooli-bridge/cli` — passed.
- `GOWORK=off go test ./...` in `scenarios/vrooli-bridge/agent` — passed.
- `go test ./internal/setup ./internal/cli/projectcli ./cmd/vrooli` — passed.
- `go test ./internal/maintenance` and `shellcheck scenarios/vrooli-bridge/bootstrap/bootstrap.sh` — passed.
- `bash scenarios/vrooli-bridge/bootstrap/bootstrap_test.sh` — `PASS=92 FAIL=0`.
- Serial compile-only checks passed for the updated maintenance package and
  `packages/platform-go` on Darwin amd64/arm64 and Windows amd64. Foreign test
  binaries were never executed.

## Teardown evidence

- The documented Mac helper was absent, so its safe teardown actions were
  completed manually: the effort-labeled authorized key was removed with a
  macOS `sed -i .bak` backup that was then deleted; the Mac effort store and
  sudo drop-in were absent afterward.
- The operator teardown helper then removed the local effort store and local
  sudo drop-in. Local `sudo -n true` now fails. A subsequent effort-key SSH
  attempt was refused.
- Bridge-only verification after teardown still reports `minimouse` ONLINE as
  Darwin amd64. The Bridge agent keeps its own paired credential and does not
  depend on the temporary effort SSH key.

## Privilege findings

The one-time-elevation design has two observed finding categories: first-touch
sudo provisioning/checking, and the headless macOS LaunchDaemon fallback,
which necessarily uses passwordless `sudo -n` for root-owned plist installation
and `launchctl` control. These are non-setup privileged workflows and remain
findings even though they are idempotent and secret-free. Teardown confirmed
their removal; the exact repeated command count is retained in the effort
execution history rather than reproduced here.
