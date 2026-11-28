# Scoped environment configuration

Scenario environment has four propagation relationships: same-scenario,
foreign-scenario, resource, and delegated-agent. Child processes derive their
environment through `packages/envkit-go`, which applies platform case folding
and the required platform floor at the boundary.

Cross-scenario addresses are not configuration. Consumers declare the peer in
`dependencies.scenarios` and resolve its current address through discovery.
Resource variables are satisfiable only when the resource is declared; shared
resource seams are owned by packages through
`adoption.owns_resource_environment`.

The producer census lives in `packages/envresolve-go/census_baseline.json`.
The current baseline is 718 reads across 87 scenarios: class A (undeclared
resource) is zero and class B (cross-scenario address) is zero. Remaining class C values are local
configuration, credentials, toolchain settings, or ambient platform inputs.

The boundary lint is `.ast-grep/rules/no-direct-cmd-env-inheritance.yml`:

```bash
ast-grep scan --rule .ast-grep/rules/no-direct-cmd-env-inheritance.yml \
  --globs '**/*.go' --globs '!**/*_test.go' internal packages resources scenarios
```

The current scan reports zero production assignments requiring migration
or an explicit host-tool boundary. They are tracked as remaining Phase 3/10
work; the rule is intentionally warning-level until that list reaches zero.
