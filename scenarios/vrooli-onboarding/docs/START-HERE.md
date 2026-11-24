# Vrooli Onboarding

Vrooli Onboarding decides **what a Vrooli install runs and under what
permissions**, applies that decision to the host, and proves it worked.

It is scenario-first: operators choose capabilities, and resources, credentials,
host tools, safeguards, and operating mode are derived from manifests. Every
decision commits through one control-plane write authority, so the browser, the
terminal, an agent, and a remote coordinator reach the same result.

## Read in this order

1. [Quickstart](QUICKSTART.md) — first run in five minutes.
2. [Wizard flow](WIZARD_FLOW.md) — the ten steps and their contracts.
3. [Architecture](concepts/ARCHITECTURE.md) — read models, write authority, tier resolution.
4. [Configuration](reference/configuration.md) — every operator-controllable decision and where it lands.

## The one-paragraph model

Manifests declare what exists. `~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` records what this
install chose. Onboarding is the surface between them and the only thing that
turns a choice into applied host state. It owns no database, stores no credential
value, and writes operator state only through `internal/operatorstate`.

## First-run selection model

A profile may recommend capabilities without selecting a scenario. If the
Scenarios step has no optional selection, choose a scenario from the catalog to
continue; the required recovery plane remains included automatically. Shared
core-set policy starts with `agent-manager`, `prompt-manager`, `web-console`,
and `vrooli-autoheal` ([owner source](../../../packages/api-core/coreset/coreset.go)).
The Core Set step previews their manifest dependency closure, so its
scenario/resource count can be much larger than the scenarios you selected
directly. The credential screen also includes descriptors declared by
system-required scenarios and their dependencies; its input count is not the
number of secrets you must enter. Resources and credential requirements are
derived from this resulting closure. Review the explanation and required vs
optional labels rather than treating every listed item as a separate profile
choice. These choices are onboarding draft state; host changes are owned by the
explicit Apply step.

Readiness separates required credentials needing attention from optional
credentials deferred until needed. Operator setup-question answers that are not
secret-kind are saved in the target-scoped onboarding draft and restored when
you return. Secret-kind text stays in the current browser tab's memory across
setup navigation and is cleared by a full reload; it is never included in the
draft. A credential's final value is sent only when you choose its secure-save
action. Do not treat any other question type as secret storage: secret answers
must be declared with the secret kind.

## Working here

Contract changes have an order: document the configurability in
[`/docs/configuration/`](../../../docs/configuration/), add the operational
target and requirement, add the experience claim, then implement and tag the test
with its `[REQ:ID]`. Statuses are earned by requirement sync from passing
evidence, never set by hand.

The integrations step reports declared requirements as deferred and creates no
bindings. Integration Hub now owns an OpenRouter connection pilot, but Onboarding
does not query it or connect manifest declarations to its bindings. See
[Integrations](concepts/INTEGRATIONS.md) for the current owner boundary.

What is not true yet is recorded in [Problems](internal/PROBLEMS.md). Read it
before assuming a documented behaviour is live.
