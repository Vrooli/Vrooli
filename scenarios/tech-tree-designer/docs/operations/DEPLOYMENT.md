# Deployment — Tech Tree Designer

## Purpose Of This Document

Distinguish local implementation from qualified support for repository-wide design.

## Supported Tiers

Local Vrooli is the existing runtime shape, not certification of all new targets. Desktop/mobile packaging, remote multi-user, SaaS and enterprise operation require separate qualification. Compact browser review is a UX target, not a claim of on-device backend portability.

## Runtime Requirements

Lifecycle owns API/UI ports, process identity and health. Embedded SQLite stores current metadata; SDA supplies observed interface data. Proposed experiments require qualified control-plane isolation of runtime identities, databases, ports and effects. No additional resource is approved by this documentation.

## Packaging

Preserve the lifecycle-built Go API/CLI, Vite UI and shared generated proto contracts. Reuse artifact owners for code generation and publication. Document owner capability/version compatibility and unsupported operations before releasing generic bundle consumers.

## Release Checklist

- [ ] Applicable requirements have fresh evidence, not only declared completion.
- [ ] General artifact/revision/grant/conflict/recovery invariants pass real owner-path qualification.
- [ ] Draft publication and ungranted runtime/billing effects are denied.
- [ ] Operator experiences are grounded and validated, including partial/stale/error states.
- [ ] Performance cohorts and budgets are approved and measured.
- [ ] Retention, backup/restore and interrupted-apply recovery are qualified.
- [ ] Deployment-specific identity, secrets and access control are verified.
- [ ] Documentation, API/CLI references and evidence describe the same supported surface.

## Rollback

Coordinate source and storage compatibility with their owners. Do not use destructive repository resets or database restoration as automatic compensation for partial application. Inspect per-entry receipts, preserve newer unrelated work and obtain applicable recovery authority.

## Cross-References

- [Runbook](RUNBOOK.md)
- [Observability](OBSERVABILITY.md)
- [Security](../internal/SECURITY.md)
