# Vrooli Onboarding

Vrooli Onboarding decides **what a Vrooli install runs and under what
permissions**, applies that decision to the host, and proves it worked.

It is scenario-first: operators choose capabilities, and resources, credentials,
host tools, safeguards, and operating mode are derived from manifests. Every
decision commits through one control-plane write authority, so the browser, the
terminal, an agent, and a remote coordinator reach the same result.

## Read in this order

1. [Quickstart](QUICKSTART.md) — first run in five minutes.
2. [Wizard flow](WIZARD_FLOW.md) — the nine steps and their contracts.
3. [Architecture](concepts/ARCHITECTURE.md) — read models, write authority, tier resolution.
4. [Configuration](reference/configuration.md) — every operator-controllable decision and where it lands.

## The one-paragraph model

Manifests declare what exists. `.vrooli/operator-state.json` records what this
install chose. Onboarding is the surface between them and the only thing that
turns a choice into applied host state. It owns no database, stores no credential
value, and writes operator state only through `internal/operatorstate`.

## Working here

Contract changes have an order: document the configurability in
[`/docs/configuration/`](../../../docs/configuration/), add the operational
target and requirement, add the experience claim, then implement and tag the test
with its `[REQ:ID]`. Statuses are earned by requirement sync from passing
evidence, never set by hand.

The integrations step is deliberately empty until integration-hub ships. It
declares the deferral and creates nothing.

What is not true yet is recorded in [Problems](internal/PROBLEMS.md). Read it
before assuming a documented behaviour is live.
