# artifactlease

Scenario-facing door to the control plane's artifact ownership records.

storage-manager's reaper decides whether an installed CLI is an orphan from how long its
owner has been missing. Scenario modules cannot import `internal/artifactlease`, so this
package re-exports its types as aliases: a scenario asking whether an artifact may be
reclaimed means exactly what the control plane means.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
