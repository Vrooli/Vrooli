---
name: "portal"
description: "Use Portal as the governed front door for chat, context, surface discovery, integrations, and Assistant migration."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  revision: 1
  modes: ["tools"]
  tags: ["portal", "chat", "context", "surfaces", "migration"]
  requires:
    scenarios: ["portal"]
    commands: ["portal"]
  origin:
    kind: "authored"
---

## Tools focus: Portal

Use Portal as the policy boundary for operator chat and context workflows.
Portal resolves target surfaces and reports unavailable optional providers; it
does not grant host control or bypass an owner scenario's authorization.

### Choose one operation

Read rows in order. The first matching row is the next step.

| Situation | One step |
|---|---|
| The command or flags are unknown | Read `portal <group> help`. **[S1]** |
| Portal may be stopped or degraded | Run `portal status`. **[S1]** |
| The operator needs a target surface | Run `portal surfaces list`, then run `portal surfaces resolve` when more than one result matches. **[S1]** |
| The operator needs integration readiness | Run `portal integrations status`. Use `portal integrations override` only with explicit operator direction. **[S1]** |
| The operator needs a context brief | Run `portal brief build` with the prompt, consumer, render mode, and bounded budget. Read the returned evidence before using the brief. **[S1]** |
| A retained context image must be inspected or removed | Run `portal context read` or `portal context delete` for the exact context ID. **[S1]** |
| A context import is uncertain | Run `portal context reconcile-import` or `portal context cancel-import`; never resend pixels before reading publication state. **[S1]** |
| A chat operation has an uncertain agent launch | Run `portal messages agent-admissions`, then read the exact admission before retrying. **[S1]** |
| Legacy Assistant data needs migration | Run `portal assistant-migration inventory`, export to a separate review directory, reconcile the checksum manifest, and capture only reviewed records. **[S1]** |

### Safety and verification

Resolve an ambiguous surface with the operator. Never select the first match.
Keep credentials, private URLs, and image pixels out of skills, workflow files,
and memory. Treat a successful request acknowledgement as incomplete until the
returned state or evidence proves the requested result. Preserve a failed or
uncertain receipt and read the owner state before retrying a mutating operation.

Optional providers may be unavailable. Keep ordinary chat available when an
optional provider is degraded and report the exact missing capability. Do not
replace Portal's typed API with a direct provider call.

Read `prompt-manager skill read program-runtime` when a Portal program binding
is added. A CLI result is not proof of a downstream side effect without its
owner-provided postcondition.

### Troubleshooting & Edge Cases

| Result | Next action |
|---|---|
| `scenario_unreachable` | Read lifecycle status and retain the unknown observation. |
| `provider_unavailable` | Report the provider and capability as unavailable; continue only with the supported fallback. |
| `ambiguous_target` | Ask the operator for the exact surface identity. |
| `permission_denied` | Stop and request the owner-authorized grant; do not retry blindly. |
| `unknown` or missing agent admission | Read `portal messages agent-admissions` and reconcile before any new launch. |
| Assistant checksum mismatch | Stop migration capture and re-export only after the source is reviewed. |
