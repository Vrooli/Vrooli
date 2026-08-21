# Profiles

**Status: deferred.** Profiles are a future feature. This page exists to capture the concept and the constraints so the eventual schema design has a starting point.

## What profiles are

A profile is a named bundle of scenario + resource selections + sensible defaults for an operator who has a specific use case in mind. Examples:

- **`engineering`** — `swarm-manager`, `agent-manager`, `workspace-sandbox`, `web-console` enabled; coding-agent integrations expected; `ollama` and `qdrant` resources enabled for local LLM/embedding.
- **`marketing`** — `landing-page-business-suite`, `browser-automation-studio`, future `rich-media-studio` enabled; AI-UGC / video-provider integrations expected; `comfyui` for local image gen.
- **`homelab`** — minimal personal install with monitoring scenarios enabled and attach-only device integrations configured by the operator.

A profile is *not* a deployment target. Targets are executable bundle contracts such as `bundle.json`; authored tier-fit evidence lives in a scenario manifest's `tier_feasibility` block. A profile is an *operator preference bundle* that pre-fills the wizard.

## Why this is deferred

We have one in-flight install today — the operator's own machine — and no second concrete profile to validate against. Designing a profile schema for one real instance and one imagined one produces wrong shapes. Per the discipline in [`README.md`](README.md): build for one, generalize after three.

The reservation in `operator-state.json` is the only thing committed today: `active_profile` is a `string | null` field, defaulting to null. When profiles ship, this field carries the active selection.

## Constraints on the eventual schema

When the second concrete profile is real and we ship `profile.schema.json`, the schema must:

1. **Reference scenarios and resources by name**, not redefine them. A profile is a selection over the existing manifest list; it's not a parallel catalog.
2. **Override defaults, not introduce new state.** A profile may set `auto_restart` defaults different from a scenario's `runtime.auto_restart_default`, but the override flows through `operator-state.json` like any other operator choice.
3. **Be composable.** The operator should be able to start from a profile and then individually toggle entries; the wizard should track "started from profile X, then made these changes" rather than "this is profile X" if any deviation exists.
4. **Live in the repo or per-user.** Repo-committed profiles are the canonical "official" bundles; per-user profiles in `~/.vrooli/profiles/` are the operator's saved preferences. Both layers, schema-shared.

## Wizard interaction (eventual)

The wizard's optional first step ("what are you trying to do?") will offer profile choices. Selecting one pre-fills subsequent steps. The operator can edit any pre-filled selection, at which point the wizard renders "started from profile X, modified" rather than the profile name.

Profiles never *enforce* selections — they are presets. The system never refuses an operator's selection because it deviates from a profile. The same discipline as `runtime.auto_restart_default` (a recommendation, not a constraint).

## Open questions when this lands

- Should the wizard's goal-intake be the only entry point, or should operators be able to set profiles directly from CLI?
- How do profiles compose with version drift (the operator's install advances; a saved profile from six months ago references scenarios that no longer exist)?
- Are profiles social — shareable across operators, with a registry — or strictly local? The first-shipped version should be local-only; "registry" is a separate scenario.

These are the questions we punt on until the second profile is real.

## See also

- [`README.md`](README.md) — discipline for deferred features
- [`architecture.md#open-work-items`](architecture.md#open-work-items) — full list of what's deferred
- [`operator-state.schema.json`](../../.vrooli/schemas/operator-state.schema.json) — `active_profile` reservation
