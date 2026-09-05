---
name: "command-center"
description: "Read Director Swarm outcome evidence, prepare the morning walk, and distinguish measured results from missing or stale evidence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  status: "active"
  revision: 1
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["command-center", "prompt-manager", "program-runtime", "source-ledger", "vrooli-memory"]
    commands: ["command-center", "prompt-manager skill read", "program-runtime library run", "source-ledger journal", "vrooli-memory learning"]
  origin:
    kind: "authored"
---
## Tools focus: Command Center

Use this skill for reading the instrument or choosing a morning walk capability. The display and the prep program read owner evidence; neither grants authority to act on findings.

| Task | Step |
|---|---|
| Prepare a briefing for another agent or the operator | Read `prompt-manager skill read command-center-vision-walk-prep`. **[S0]** |
| Conduct or resume the strategic conversation | Read `prompt-manager skill read morning-vision-walk`. **[S0]** |
| Inspect measured outcomes without sample figures | Run `command-center walk read`. **[S1]** |
| Inspect ranked missing instrumentation | Run `command-center focus`. **[S1]** |
| Improve this instrument or its walk capabilities | Read `prompt-manager skill read command-center-improve`. **[S0]** |

Keep coverage, trust, and empirical verdicts separate. A zero with valid provenance is a real zero. A sample, stale value, missing timestamp, or unavailable source cannot support a current success claim. The denominator belongs to the objective/capability owner.

### In-use settings

| Symptom | Move |
|---|---|
| Readings are capped | Increase `command-center walk read --limit N` up to 100; retain the reported total and truncation flag. |
| Briefing detail is too large | The prep skill chooses a smaller per-source limit; do not hide unavailable sources. |

### Troubleshooting & Edge Cases

An unavailable source is a named gap. Inspect its existing owner guidance; request no restart merely to make the board green. The ordinary read commands stay S1 because each is one stable operation. Repeated cross-source work belongs in the prep program. Source recovery and product decisions remain with their owners and the caller's authority.
