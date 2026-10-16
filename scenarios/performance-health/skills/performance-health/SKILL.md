---
name: "performance-health"
description: "Run bounded packaged-launch experiments and inspect comparable performance evidence through Performance Health."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["performance", "packaged-launch", "experiments", "reporting"]
  icon: "activity"
  status: "active"
  revision: 1
  createdAt: "2026-10-07T00:00:00Z"
  updatedAt: "2026-10-07T00:00:00Z"
  requires:
    scenarios: ["performance-health", "scenario-to-desktop", "program-runtime"]
    commands: ["performance-health packaged-experiment", "program-runtime library run"]
  origin: {kind: "authored"}
---
## Tools focus: Performance Health packaged experiments

Use Performance Health to validate packaged-launch inputs, collect owner-produced
samples, and report measured comparisons. The desktop ramp owns builds and
journey execution; Performance Health owns capture validation and comparison.
Do not treat a successful validation or a partial report as a completed
experiment.

### Before acting

Read `path:docs/architecture/packaging-strategies.md` §§5.1–5.2 and 6.1–6.4,
then inspect `performance-health packaged --help` and the current
`packaged-experiment.v1` contract. Confirm all four declared subjects have
current runnable journey IDs, the experiment environment fields are observed,
the desktop build prerequisites are ready, and host capacity can support serial
builds and cold/warm launches. Do not start captures while an owner repair,
missing artifact prerequisite, or active host pressure is unresolved.

### Choose the operation

| Situation | One step |
|---|---|
| Need to validate and run the complete desktop comparison | Run `performance-health.packaged-experiment` with the full `packaged-experiment.v1` manifest through `program-runtime library run`. The program validates the matrix before building or capturing. Branch on the program contract's `status` and `errors[0].class`; retain the returned pipeline IDs, completed cells, missing data, and JSON/Markdown report. **[S3]** |
| Need a report for an existing experiment | Run `performance-health packaged report --experiment <id>` and retain its comparability and missing-data fields. **[S1]** |
| One cell needs a standalone capture outside the governed matrix | Use `performance-health packaged capture --help`; verify the requested artifact, digest, profile attribution, condition, and journey against the owner output before comparing it. **[S1]** |

For the governed comparison, the program validates the complete four-subject,
six-cell matrix and builds each cell through scenario-to-desktop. Supply
observed host fingerprint, display, profiler mode, and power state, plus one
current journey ID per subject. Caller artifact paths and digests are not used
as build inputs; each capture must use the artifact returned by that cell's
pipeline. The expected matrix has 480 launches: 24 built artifacts, cold and
warm conditions, and 10 runs per condition. Every cell also requires journey
replay evidence.

The JSON and Markdown reports are comparable only where host fingerprint,
artifact digest, display, profiler mode, and power state match. Read per-cell
toggle attribution, medians, parity, reliability, and missing-data fields. An
unavailable or incomplete measure remains unknown; it is never zero or a pass.

### In-use settings

| Symptom | Setting move |
|---|---|
| A manifest describes another platform | Set `platform` to the actual supported target and rebuild that target's cells; preserve the previous experiment as a separate record. |
| A journey or environment field is unknown | Leave the experiment unstarted until its owner can provide observed evidence. |
| A cell fails or the report marks it non-comparable | Keep the returned error and missing-data reason; repair the named owner or evidence prerequisite before a new experiment ID. |

### Troubleshooting & Edge Cases

| Symptom | First check | Action |
|---|---|---|
| Program preparation returns `skill_incomplete` or a missing binding | `program-runtime library get performance-health.packaged-experiment --json` and `program-runtime bindings describe <binding>` | Repair the owning declaration or binding; do not bypass Program Runtime with direct API calls. |
| A cell has no AppImage or pipeline ID | Inspect the returned `built_pipeline_ids` and the scenario-to-desktop pipeline result | Treat the cell as failed and retain its pipeline evidence; do not substitute a stale artifact. |
| A report says parity, reliability, or environment data is missing | Read the report's `missing_data` entries | Keep the comparison incomplete until the owner supplies evidence. |
| Host pressure, owner validation failures, or build prerequisites are active | Check the documented reopening conditions in the active E3 work record | Do not launch the matrix or Test Genie until those conditions are satisfied. |

Repeated pipeline wait or artifact-selection gaps belong in the scenario-to-
desktop CLI/program contract; repeated comparison gaps belong in
Performance Health. Keep this skill as routing judgment rather than duplicating
those command contracts.
