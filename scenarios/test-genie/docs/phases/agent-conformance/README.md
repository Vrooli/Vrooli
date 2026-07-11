# Agent Conformance

## North Star

Every coding-agent consumer declares an Agent Manager dependency that is not
explicitly disabled and requests portable `roleRef` profiles rather than
concrete runners or models. Agent Manager owns the read-only validation; Test
Genie discovers it only through the provider descriptor.

## The rungs and their gates

L0 requires the dependency. L1 requires every scenario-owned profile file to be declared, valid, owned by the target, and role-only. L2 requires every `roleRef` to resolve through Agent Manager's role catalog. L3 requires the direct-spawn boundary: coding-agent executables must not be constructed outside Agent Manager. L4 is clean conformance across all dimensions.

## What each finding means

Dependency findings identify missing or disabled integration. Profile findings identify unreadable, undeclared, legacy, invalid, or incorrectly owned scenario configuration. Role findings identify a role that cannot be resolved. Direct-spawn findings identify a narrow executable-construction pattern next to a known coding-agent command and are advisory.

## The canonical fix

Declare `dependencies.scenarios.agent-manager`, register each `.vrooli/agent-profiles/*.json` source, and replace runner/model/policy inputs with `roleRef`. Route execution through Agent Manager rather than spawning a coding-agent executable from the consumer.

## How to verify

Run `agent-manager profile reconcile-scenario --scenario <scenario> --dry-run`, then run the Test Genie `agent-conformance` phase.
