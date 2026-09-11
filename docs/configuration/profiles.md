# Profiles

**Status: active for onboarding.** Profiles are versioned, repository-owned
presets that collect purpose answers and produce explainable scenario
recommendations. They are recommendations, not permissions or deployment
targets.

## What profiles are

A profile is a named questionnaire and rule set for an operator who has a
specific use case in mind. The current data-only profiles are:

- **`local-use`** — local application use or local development.
- **`develop-and-publish`** — application development, desktop publishing, and
  optional local or managed hosting choices.
- **`general-purpose`** — combined local, publishing, desktop, mobile, and
  hosting purposes with conditional questions. Publishing also asks for the
  distribution method, account ownership, payment and mail needs, download
  storage, and connected-target use when those choices apply.
- **`customer-preinstalled`** — customer-facing defaults with optional
  development and managed-hosting capabilities.

The canonical files live in
[`scenarios/vrooli-onboarding/profiles/`](../../scenarios/vrooli-onboarding/profiles/).
Adding a supported profile is a data change. The evaluator accepts bounded
questions and conditions only; it does not execute profile-authored code.

A profile is *not* a deployment target. Targets are executable bundle contracts such as `bundle.json`; authored tier-fit evidence lives in a scenario manifest's `tier_feasibility` block. A profile is an *operator preference bundle* that pre-fills the wizard.

Publishing questions describe applicability only. They do not collect provider
secrets, consent receipts, passwords, tokens, or signing material. The
evaluator turns a selected payment, mail, storage, mobile, or remote-target
choice into a capability recommendation; the owning scenario remains
responsible for its configuration and verification.

A capacity posture is deliberately not a profile. `capacity_posture` answers one
host-placement question (`responsive`, `balanced`, `throughput`, or `minimal`)
and selects capacity rungs, priorities, and transient reserve. It does not
select scenarios or resources, and it does not set the reserved
`active_profile` field.

## Profile contract

Each profile must:

1. **Reference scenarios and resources by name**, not redefine them. A profile is a selection over the existing manifest list; it's not a parallel catalog.
2. **Override defaults, not introduce new state.** A profile may set `auto_restart` defaults different from a scenario's `runtime.auto_restart_default`, but the override flows through `operator-state.json` like any other operator choice.
3. **Be composable.** The operator should be able to start from a profile and then individually toggle entries; the wizard should track "started from profile X, then made these changes" rather than "this is profile X" if any deviation exists.
4. **Carry provenance and a revision.** The onboarding service reports the owner,
   source, and review revision with the profile metadata.
5. **Remain bounded.** Question count, expression depth, collection size, and
   supported operators are capped by the evaluator.

The selected profile is committed as `operator-state.active_profile` through the
typed operator-state authority. Scenario and resource choices remain ordinary
field-scoped operator-state choices, so later manual edits are explicit and
re-enterable.

## Wizard interaction

The welcome step offers the repository-owned profiles and renders only questions
visible for the current answers. It shows validation issues and the resulting
scenario set before the operator accepts. Manual scenario selection remains
available beside the profile path. Selecting a profile never silently overwrites
explicit choices.

Profiles never *enforce* selections — they are presets. The system never refuses an operator's selection because it deviates from a profile. The same discipline as `runtime.auto_restart_default` (a recommendation, not a constraint).

Profile evaluation is available through the same Connect API, CLI group, and UI
client. Desktop bundles must stage the profile JSON next to the onboarding
scenario catalog so the bundled API has the same behavior as a repository run.

## See also

- [`scenarios/vrooli-onboarding/profiles/`](../../scenarios/vrooli-onboarding/profiles/) — canonical profile data
- [`operator-state.schema.json`](../../.vrooli/schemas/operator-state.schema.json) — `active_profile`
- [`scenarios/vrooli-onboarding/docs/WIZARD_FLOW.md`](../../scenarios/vrooli-onboarding/docs/WIZARD_FLOW.md) — interaction contract
