# Configuration Schema Population Plan

## Context

The configuration-schema bundle (commit `d1e2cd616e onboarding rework p1`) added five new fields across four schemas: `risk` on safeguards, `system_required` on scenarios, `runtime.{kind, auto_restart_default}` on scenarios, the `secretDescriptor` def in `common.schema.json` referenced from `resource.credentials.env`, and the new `operator-state.json` schema. All fields are optional — nothing populated them.

This plan populates the new fields on a known worked-example set: 7 safeguards, 6 scenarios, 2 resources. The set is deliberately small. It is the same set referenced as worked examples in `docs/configuration/`, so populating here makes the docs honest. Phase-3 (opportunistic population of the rest) happens in normal scenario/resource maintenance and is not part of this plan.

This is **not greenfield** in the usual sense — we are *adding* fields to existing manifests, not rewriting them. Existing fields and structure must be preserved verbatim.

## Required Reading

No prompt-manager skills are materially applicable to this plan; the work is mechanical-with-judgment. The substrate to read before editing:

- `docs/configuration/architecture.md` — source-of-truth table and resolution order
- `docs/configuration/host/safeguards.md` — what `risk` low/medium/high mean
- `docs/configuration/scenarios.md` — what `system_required` and `runtime.kind` mean
- `docs/configuration/secrets.md` — `secretDescriptor` shape and intent
- `.vrooli/schemas/safeguard.schema.json`, `service.schema.json`, `resource.schema.json`, `common.schema.json` — field definitions

## Scope

In scope (this plan):
- Set `risk` on all 7 safeguards under `internal/safeguards/`
- Set `service.system_required` and `runtime.{kind, auto_restart_default}` on 6 scenarios: `vrooli-onboarding`, `secrets-manager`, `web-console`, `swarm-manager`, `agent-manager`, `workspace-sandbox`
- Enrich `credentials.env` to use `secretDescriptor` shape on 2 resources: `gemini`, `openrouter`

Out of scope (intentionally deferred):
- Other scenarios (~50+) — populated opportunistically as they are touched for unrelated work
- Other resources beyond `gemini` and `openrouter` — same
- Type unification across schemas (`healthCheck` consolidation, `secretReference` re-use, snake_case/camelCase reconciliation) — separate plan
- Wiring onboarding API/UI to read the new fields — separate plan

## Constraints

- **Additive only.** Do not reorder, rename, or remove existing fields. Use targeted `Edit` operations, not file rewrites.
- **No mass-update scripts** (per `CLAUDE.md`). Each file is edited individually with judgment.
- **JSON validity preserved.** Trailing-comma rules, schema-reference paths, and existing field ordering are kept intact.
- **Schema-default values are not written.** If a value matches the schema's default (e.g. `system_required: false`, `risk: "low"`), do not write it — leave the field absent so the default applies. Only write non-default values.
- **No code changes.** This is manifest-only work. No Go, no TypeScript, no docs (the docs already exist and should not be edited here).

## Phase 1 — Safeguards (7 files)

For each `internal/safeguards/<name>/safeguard.json`:

1. Read the manifest plus its Go handler under `internal/safeguards/<name>/` to confirm what the safeguard actually modifies.
2. Classify by the schema's definitions (verbatim from `safeguard.schema.json`):
   - `low` = no system state changes outside Vrooli's tree (probes, reads)
   - `medium` = writes config files outside Vrooli's tree or modifies networking rules
   - `high` = modifies kernel parameters or requires root in ways that broadly affect host behavior
3. Add the `risk` field. Place it after `platforms` (where the existing description field positions it). Skip writing if classification is `low` (matches schema default).
4. Reasoning recorded inline as a one-line `notes` addition only if non-obvious; do not write a justification comment.

Files in this phase:
- `internal/safeguards/clock/safeguard.json`
- `internal/safeguards/dns-resolution/safeguard.json`
- `internal/safeguards/docker-host-firewall/safeguard.json`
- `internal/safeguards/kernel-config/safeguard.json`
- `internal/safeguards/nat-protection/safeguard.json`
- `internal/safeguards/remote-session-protection/safeguard.json`
- `internal/safeguards/tcp-tuning/safeguard.json`

The classifications are not pre-decided in this plan. The executing agent makes the call from each handler's actual behavior.

## Phase 2 — Scenarios (6 files)

For each scenario, edit `scenarios/<name>/.vrooli/service.json`:

### System-required scenarios (3)

Set `service.system_required: true` and add a top-level `runtime` block:

```json
"runtime": {
  "kind": "long_running",
  "auto_restart_default": true
}
```

- `scenarios/vrooli-onboarding/.vrooli/service.json`
- `scenarios/secrets-manager/.vrooli/service.json`
- `scenarios/web-console/.vrooli/service.json`

`system_required` is placed inside the existing `service` block. `runtime` is a new top-level sibling of `service`, `ports`, etc. — match the position used in `service.schema.json` (after `service`, before `ports`).

### User-application scenarios (3)

Add the `runtime` block only — do **not** set `system_required` (defaults to false):

```json
"runtime": {
  "kind": "long_running",
  "auto_restart_default": false
}
```

- `scenarios/swarm-manager/.vrooli/service.json`
- `scenarios/agent-manager/.vrooli/service.json`
- `scenarios/workspace-sandbox/.vrooli/service.json`

If any of these six scenarios already have a top-level `runtime` field with conflicting content (none expected based on the verification pass, but check), pause and flag rather than overwriting.

## Phase 3 — Resources (2 files)

### `resources/gemini/resource.json`

Replace the bare-string `credentials.env` entry with a `secretDescriptor` object. Existing shape:

```json
"credentials": {
  "env": ["GEMINI_API_KEY"],
  "secret_ref": "secret/vrooli/gemini"
}
```

Becomes:

```json
"credentials": {
  "env": [
    {
      "env": "GEMINI_API_KEY",
      "label": "Gemini API Key",
      "description": "Google Gemini multimodal LLM. Required for scenarios that call Gemini directly for text generation, vision, or multimodal reasoning.",
      "classification": "user",
      "required": true,
      "obtain_url": "https://aistudio.google.com/app/apikey",
      "default_hint": "Starts with 'AIza...'"
    }
  ],
  "secret_ref": "secret/vrooli/gemini"
}
```

### `resources/openrouter/resource.json`

Same transformation:

```json
"credentials": {
  "env": [
    {
      "env": "OPENROUTER_API_KEY",
      "label": "OpenRouter API Key",
      "description": "OpenRouter unified API gateway across LLM providers. Required for scenarios that route LLM calls through OpenRouter rather than calling providers directly.",
      "classification": "user",
      "required": true,
      "obtain_url": "https://openrouter.ai/keys",
      "default_hint": "Starts with 'sk-or-...'"
    }
  ],
  "secret_ref": "secret/openrouter"
}
```

`secret_ref` is **not** changed — these paths are already in use by `packages/api-core/secrets` and existing scenarios. Changing them is a separate concern.

## Verification

After all three phases:

1. **Schema validation.** Validate every modified file against its schema. Use whichever validator the project standardizes on; if unclear, use `ajv` via `npx`:
   ```
   for f in internal/safeguards/*/safeguard.json; do
     npx --yes ajv-cli validate -s .vrooli/schemas/safeguard.schema.json -d "$f" --strict=false
   done
   ```
   Equivalent loops for the 6 scenario `service.json` files (against `service.schema.json`) and 2 resource `resource.json` files (against `resource.schema.json`).
2. **JSON well-formedness sanity.** `jq . <file>` over all modified files; non-zero exit anywhere is a fail.
3. **No accidental edits.** `git diff --stat` should show exactly 15 files modified (7 safeguards + 6 scenarios + 2 resources). Any additional file in the diff is a pause-and-investigate.
4. **Diff review.** Read each diff. Confirm only the new fields were added; no field reordering, no whitespace churn, no comment additions.

If any validator fails, fix the offending file. Do not proceed to commit until all 15 files validate.

## Done When

- All 15 listed files contain the new fields per Phases 1–3.
- Schema validation passes for all 15.
- `git diff --stat` shows exactly 15 files modified.
- No code or doc changes leaked into the diff.

## Not Done

- Onboarding API/UI does not yet read these fields — separate plan.
- The remaining ~45 scenarios and ~25 resources are still on schema defaults — opportunistic Phase 3, not this plan.
- Type-unification follow-up (healthCheck consolidation, etc.) — separate plan.
