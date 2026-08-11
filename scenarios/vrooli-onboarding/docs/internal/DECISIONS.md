# Decisions

Settled positions with their tradeoffs and revisit triggers. Rationale is prose
on purpose — a decision recorded without its cost gets relitigated.

## Manifests and operator state are the only configuration authority

Onboarding internals, browser navigation, and any generated file are disposable.

**Tradeoff**: two files must be read and reconciled for every effective value,
instead of one. **Why it wins**: a single merged file would need onboarding to
author manifest content, which makes the wizard a code generator and makes
upgrades destructive.

## The write API is a field-scoped merge patch, not a document

`internal/operatorstate` loads, merges the patch, validates, and writes
atomically under a lock.

**Tradeoff**: more ceremony for callers than "PUT the document". **Why it
wins**: a partial writer performing a total write silently deletes every field it
does not model — including `trust_posture` and `core`, which govern the install's
security stance. Adding a missing struct field fixes one instance; patches remove
the class. **Revisit if**: the document becomes small and stable enough that every
writer can model all of it, which the roadmap makes unlikely.

## One writer, enforced structurally

A structural test asserts exactly one write site for the state file.

**Tradeoff**: a new legitimate writer costs a code review conversation. **Why it
wins**: that conversation is the point. Two authorities for one decision is how
resource enablement ended up split between operator state and the repository
service manifest.

## Apply is part of the wizard, not a follow-up

**Tradeoff**: onboarding becomes a mutating surface with real failure modes,
rather than a safe read-and-record tool. **Why it wins**: a wizard that records
preferences and never acts leaves the operator with no way to observe that setup
happened, and no operator remembers a documented follow-up command.

## Apply orders and reports; the control plane performs

Every action delegates to its owning control-plane handler.

**Tradeoff**: onboarding cannot fix a host problem it can clearly see. **Why it
wins**: the repository contract reserves host detection and remediation for the
control plane, and a private implementation in a scenario is exactly the drift
that contract exists to prevent.

## `Applied` and `Verified` are different states

An item can install cleanly and still fail its probe.

**Tradeoff**: two states to explain instead of one. **Why it wins**: collapsing
them makes validation an echo of apply rather than an independent gate, which is
where a tool on disk but not on `PATH` hides.

## Required gaps block; optional gaps offer a recorded degraded continue

**Tradeoff**: an operator can finish setup in a knowingly incomplete state.
**Why it wins**: blocking on an optional gap trains operators to bypass the
wizard entirely, which is strictly worse. Recording the acknowledgement keeps the
degraded state visible to later diagnosis.

## Native secure storage is the local credential authority; unsupported targets fail closed

An unreachable native store never falls back to the encrypted file store.

**Tradeoff**: a host with a transiently wedged keyring cannot provision at all.
**Why it wins**: credentials split across two backends by session health are
worse than the degraded state `doctor` reports, because the split is invisible
until a value cannot be found.

## Integration-hub is the only deferred V2 capability, and it creates nothing

**Tradeoff**: a visibly empty step in the flow. **Why it wins**: a step that
fabricates a connection so the flow looks complete produces state nothing can
honour, and it is discovered at the worst possible time.

## Profiles wait for a second concrete profile

**Tradeoff**: the goal-intake step stays deferred, so first-run has no
express lane. **Why it wins**: a profile format designed against one example fits
one example. `active_profile` is already reserved in the schema, so the wait costs
nothing structurally.

## The desktop bundle is a separate install

A bundled app resolves state under its own app-data root, not the repository
document.

**Tradeoff**: configuring the desktop app does not configure a local tier-1
install on the same machine. **Why it wins**: a standalone app that writes into a
repository it may not own is worse. **Revisit if**: shared sign-in and entitlement
work needs one identity across tiers on a machine — that is an identity concern,
and it should not be solved by merging state documents.
