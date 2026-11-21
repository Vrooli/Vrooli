# Agent Conformance

## North Star

Every coding-agent consumer declares an Agent Manager dependency that is not
explicitly disabled and requests portable `roleRef` profiles rather than
concrete runners or models. Scenario-owned agent assets — profiles and
workflows — live in one unified location, `.vrooli/agent-manager/`, declared
through a single `config.declarations` block. Agent Manager owns the read-only
validation; Test Genie discovers it only through the provider descriptor.

## The rungs and their gates

L0 requires the dependency. L1 requires every scenario-owned declaration file to
be declared under `config.declarations.sources`, valid, and owned by the target
— for profiles, role-only; for workflows, a well-formed catalog definition. L2
requires every `roleRef` (in profiles and in workflow run nodes) to resolve
through Agent Manager's role catalog. L3 requires the direct-spawn boundary:
coding-agent executables must not be constructed outside Agent Manager. L4 is
clean conformance across all dimensions.

### Workflow rungs

A declared workflow climbs the same ladder with workflow-specific gates:

- **declared** — the workflow file is listed in `config.declarations.sources`;
  an undeclared `.vrooli/agent-manager/*.json` workflow is an orphan.
- **valid** — the definition parses and carries no blocking catalog diagnostic
  (structure, budgets, reachability, continuations).
- **owned** — `owner` matches the target scenario and `key` is prefixed by it.
- **CEL-clean** — every branch edge `condition` compiles against Agent Manager's
  workflow CEL environment and returns a boolean. A syntax or type error is a
  violation reported with its edge path, caught at registration rather than
  mid-execution.
- **placeholder-clean** — every `{{.name}}` a run/continue prompt references is
  backed by a declared binding on that node. An unbound placeholder is a
  violation; an unused binding is a non-blocking warning surfaced at reconcile,
  not a conformance finding.

## Old-layout rejection

The unified layout is the only readable location; there is no dual-read
fallback. Two conditions are violations regardless of anything else:

- a declaration file remaining under the retired `.vrooli/agent-profiles/` or
  `.vrooli/agent-workflows/` directories (`declaration_legacy_layout`);
- a legacy `config.profiles` or `config.workflows` block in `service.json`
  (`declaration_legacy_block`).

Both are reported with the remediation to move the file into
`.vrooli/agent-manager/` and declare it under `config.declarations.sources`.

## What each finding means

Dependency findings identify missing or disabled integration. Profile findings
identify unreadable, undeclared, legacy, invalid, or incorrectly owned
scenario configuration. Workflow findings identify a malformed, invalid,
undeclared, or incorrectly owned workflow definition, including precise CEL and
prompt-placeholder defects. Role findings identify a role that cannot be
resolved. Legacy-layout and legacy-block findings identify declarations that
have not moved to the unified location. Direct-spawn findings identify a narrow
executable-construction pattern next to a known coding-agent command and are
advisory.

## The canonical fix

Declare `dependencies.scenarios.agent-manager` with a `config.declarations`
block, move each profile and workflow file into `.vrooli/agent-manager/` with
its `schemaVersion` set, register every file under
`config.declarations.sources`, and replace runner/model/policy inputs with
`roleRef`. Route execution through Agent Manager rather than spawning a
coding-agent executable from the consumer.

## How to verify

Run `agent-manager declarations reconcile-scenario --scenario <scenario>
--dry-run`, then run the Test Genie `agent-conformance` phase.
