---
name: "secrets-manager"
description: "Use Secrets Manager as the authority for metadata-safe vault operations, fresh human assurance, bounded grants, agent identity verification, and brokered credential use."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["secrets-manager", "password-manager", "credential-authority", "grants", "broker"]
  status: "active"
  revision: 1
  learning:
    scope: "secrets-manager-usage"
    capture: "every attempt"
  requires:
    scenarios: ["secrets-manager", "scenario-authenticator", "agent-manager"]
    commands: ["secrets-manager broker request-access", "secrets-manager broker create-session", "secrets-manager broker execute", "secrets-manager broker revoke"]
  origin:
    kind: "authored"
---

## Trust boundary

Secrets Manager owns encrypted item custody, workspace membership, grants, access requests, use sessions, origin pinning, revocation, and secret-safe activity. A metadata response is the default. A human reveal requires a fresh Scenario Authenticator session and a one-time action-bound assurance token.

Agent calls carry an opaque Agent Manager identity token. Secrets Manager verifies that token with Agent Manager and derives the principal from the verified run ID. Agent credentials can use the explicitly bounded broker routes, but they cannot perform human administration or mint human reveal assurance.

## Broker use

Use the generated broker contract for agent-facing work:

1. Request access for a specific grant, item, operation, and lifetime.
2. Create a session with the reviewed target origin and bounded lifetime.
3. Execute only the operation allowed by the pinned session.
4. Revoke the session when the task ends or its scope changes.

The broker may send an authorized credential to the destination, but ordinary agent results contain only the allowed projection. Do not copy credential values into prompts, logs, command output, or durable hand-offs.

## Evidence and recovery

Distinguish configured, reachable, externally verified, expired, and revoked sources. Backup coverage is evidence only when it includes a verified drill and a restore checksum. Missing provider evidence stays degraded and must not be presented as successful recovery.
