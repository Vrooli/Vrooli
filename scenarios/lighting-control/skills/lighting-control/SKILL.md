---
name: "lighting-control"
description: "Choose and verify the shortest supported path through Lighting Control."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["lighting-control", "usage"]
  status: "active"
  revision: 1
---

## Use Lighting Control

Start from the outcome you need. Use the scenario's stable CLI for one deterministic operation, the UI for reviewing or comparing several related results, and the API when another scenario needs a typed machine contract. Keep durable facts in this scenario rather than in an agent's notes.

| Need | Next step |
|---|---|
| Find the supported operation for a specific outcome | Run `lighting-control --help` and choose the matching command group; use the command's `--help` for its input and output contract. [S1] |
| Review state that depends on several related records or needs human judgment | Open the scenario UI and review the source records and status before taking an action. [S0] |
| Call the capability from another scenario | Read `docs/QUICKSTART.md` and the API contract, then use the typed API client rather than scraping UI output. [S0] |
| Verify a result | Re-read the created or changed record with the owning CLI/API query and compare it with the requested outcome. [S1] |
| The requested outcome has no documented operation | Stop before inventing a command or storing a second source of truth; read `docs/START-HERE.md` and route the missing primitive to the scenario owner. [S0] |

Keep each request within its documented scope. Check the returned status and errors before treating a command as successful. Do not repeat a write because the first response is unclear; query the owning record first. Never add credentials, publish private data, or take an external action unless its owning contract explicitly supports it and the operator authorized it.

### Troubleshooting & Edge Cases

| Symptom | First check | Response |
|---|---|---|
| The command is missing | Confirm the scenario CLI is built and read `docs/QUICKSTART.md`. | Use its documented local invocation; do not guess alternate flags. |
| The result is empty or partial | Read the response status and documented source or pagination limits. | Report the observed gap; do not interpret missing data as proof of absence. |
| A write response is interrupted | Query the owning record by its stable identifier before retrying. | Retry only when the owner contract says the operation is safe to repeat. |
