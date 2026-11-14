# Runbook — Scenario to Android

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup
make start
make status
make logs
make stop
make test
```

Do not start API/UI binaries directly. The lifecycle owns process
naming, ports, health checks, and logs.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `scenario-to-android status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

## Backup / Restore

The generated template uses local SQLite state. Product scenarios must
define backup and restore procedures before production deployment.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | deferred | deferred | Define before deployment. |
| Blob files | deferred | deferred | Define if binary/blob domains remain. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Android Ramp Operations

The ramp owns artifact orchestration and evidence review; device verbs remain
owned by `device-control`, and web-content automation remains owned by BAS.
Use the CLI or equivalent API routes:

```bash
ANDROID_SOURCE_REF=/absolute/path/to/scenario/ui/dist \
  scenario-to-android android build
scenario-to-android android targets
scenario-to-android android matrix-catalog
ANDROID_ARTIFACT_DIGEST=sha256:<artifact-sha256> \
ANDROID_ARTIFACT_REF=android-debug:<artifact-sha256> \
ANDROID_ARTIFACT_PATH=/absolute/path/to/app.apk \
  scenario-to-android android matrix-create
scenario-to-android android matrix-start <run-id>
scenario-to-android android matrix-wait <run-id>
scenario-to-android android matrix-abort <run-id>
```

To run only the installed-app offline/online chapter on an isolated USB
Android target, create a matrix with the exact journey and target selected:

```bash
ANDROID_ARTIFACT_DIGEST=sha256:<artifact-sha256> \
ANDROID_ARTIFACT_REF=android-debug:<artifact-sha256> \
ANDROID_ARTIFACT_PATH=/absolute/path/to/app.apk \
ANDROID_ARTIFACT_PACKAGE=<installed-package> \
ANDROID_TARGET_ID=<exact-usb-target-id> \
ANDROID_JOURNEY_IDS=android-scenario-app-offline-transition-v1 \
  scenario-to-android android matrix-create
```

Then start and wait on the returned server-owned matrix ID. The selected target
must be attached over USB, have no competing owner, and already contain the
fixture at the version named by the selected immutable artifact metadata. This
journey runs offline and online BAS assertions, restores
connectivity and original display/lock state, and confirms the package remains
at the expected version without installing or uninstalling it.

For durable app upgrades, first install the baseline artifact with `android
app-install` while the package is absent. Build the next artifact with the
same package and signing identity and a strictly higher
`ANDROID_ARTIFACT_VERSION_CODE`, then run `android app-update`. The selected
scenario must register its cold/warm flow at
`bas/cases/mobile/conformance-flow.json` and the
`scenario-app-update-preservation` flow at
`bas/cases/mobile/update-preservation-flow.json`. Scenario-to-Android checks
both files and registry entries before device lease acquisition or installation.
The update-preservation flow asserts persisted app state on the installed
version before replacement and again immediately after update, before other
BAS input. A failed pre-update assertion prevents replacement. The update path
refuses unknown or non-increasing installed versions and leaves the updated
package in place for inspection after a post-update failure.

To repeat BAS checks against an installed package without replacing or deleting
it, use `android app-verify` with the expected package and version settings. It
uses the scenario's registered read-only preservation flow for saved-state,
cold-start, and warm-resume assertions and restores the display and lock state.
The flow must declare observer execution, `reset: none`, and use only
navigation, wait, assertion, screenshot, or extraction actions. If Device
Control omits installed `versionCode`, the run records that limit and does not
claim exact APK byte identity. This verification path skips Device Control
video and log capture while external recording owners remain unreconciled.

`matrix-abort` waits for the matrix owner to persist cancellation. Keep the
owner running until it reports a terminal result; if a request ends before
then, use `matrix-wait` to continue observing the owner.

`android build` produces a debug APK and AAB without an operator signing
identity. A matrix cell is pass-eligible only when device-control proves the
target, BAS completes the registered flow, and the redacted recording has
bounded offsets and a checksum. Unavailable targets must remain unavailable;
do not substitute a host-only or idle recording.

For review tooling, open the UI target matrix, run review, and readiness pages
at `/targets`, `/runs`, and `/readiness`. When deployment-manager identity is
configured (`DEPLOYMENT_MANAGER_URL`, `DEPLOYMENT_MANAGER_PROFILE_ID`, and
`DEPLOYMENT_MANAGER_GIT_COMMIT`), completed matrix gates report reference-only
verdicts to its evidence service.

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
