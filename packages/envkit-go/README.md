# envkit-go

The repository-wide process environment boundary (Go package `envkit`). An environment is an
exec boundary, not an ownership boundary, so relative variables are never inherited into a
process that belongs to another scenario or resource. Names are folded for the target platform.

Relative identity includes service endpoints, lifecycle/process labels and
instance/storage namespaces. Caller-session identity includes provider
`*_AGENT_TAG` markers, run/harness locators, Agent Manager store attribution,
and terminal/provider session plumbing. `Resource` and `ForeignScenario`
drop these inherited observations; `SameScenario` retains its own context.
`DelegatedAgent` retains the launcher-composed caller session, but drops
service-relative ownership. Existing credential rules remain separate.

Independent managed providers use `WithOverlay` with `Resource`: derive from
the host first, then explicitly overlay the child's verified identity, home,
store and any child-specific tag. The overlay also deduplicates names using
the platform's case rules. Host installation roots, PATH/HOME, toolchain
settings and resource/provider credential availability are preserved; this is
an ownership boundary, not an authentication downgrade or a general secret
filter.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
