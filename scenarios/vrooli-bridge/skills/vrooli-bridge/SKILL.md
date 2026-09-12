---
name: "vrooli-bridge"
description: "Qualify Vrooli Bridge deployments through explicit target authority, durable Test Genie receipts, and retained evidence without claiming certification from admission alone."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["vrooli-bridge", "qualification", "deployment", "targets", "evidence", "test-genie"]
  icon: "shield-check"
  status: "active"
  revision: 1
  createdAt: "2026-09-09T00:00:00Z"
  updatedAt: "2026-09-09T00:00:00Z"
  requires:
    scenarios: ["vrooli-bridge", "test-genie", "program-runtime"]
    commands: ["program-runtime library run", "test-genie validation", "vrooli scenario test"]
  learning:
    scope: "vrooli-bridge-usage"
    capture: "every attempt"
  origin: {kind: "authored"}
---
## Tools focus: Bridge Qualification

Use `vrooli-bridge.qualification` for bounded qualification admission. Supply
an explicit target list, matching owner authority, an opaque authorization
reference, and the candidate digest. The program refuses before the owner call
when authority is missing, preserves existing cell states, and delegates
admission, idempotency, execution, and receipts to Test Genie.

Admission is not proof of success. Retain the receipt and its producer evidence;
wait exactly once when the receipt is non-terminal. A repeated request key with
matching inputs attaches to the same durable receipt. A changed input under the
same key is a typed conflict. Native target mutation, credentials, and release
promotion remain outside this program and require their separately governed
owner contracts.

Required reading: `path:scenarios/vrooli-bridge/docs/internal/TESTING.md` and
`path:scenarios/vrooli-bridge/docs/operations/DEPLOYMENT.md`.
