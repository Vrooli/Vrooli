# Flows — React Component Library

This document is the canonical workflow map for ordered behavior.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Validation |
|---|---|---|---|---|
| Index library | components | CLI/API/UI `components index` | Manifests and version folders are validated and reflected in SQLite | Indexer, repository, handler, CLI tests |
| Apply component | adoptions | CLI/API/UI `adoptions apply` | Selected version source is copied into a target scenario with provenance and an adoption row | Service, handler, CLI, UI tests |
| Assisted extract/adopt | workflows | CLI/API/UI `workflows start` | RCL records a scoped Agent Manager task/run and exposes honest queued/running/terminal status | Workflow service, handler, CLI tests |
| Promotion readiness | workflows | CLI/API/UI `workflows promotion-readiness` | Read-only evidence report joins parity, examples, dependency closure, origin replacement, and drift | Workflow readiness service, handler, CLI, UI tests |
| Refresh drift | adoptions | CLI/API/UI `adoptions refresh` | Adoption rows receive separate library-version and local-edit statuses | Service status matrix and UI tests |
| Reapply component | adoptions | CLI/API `adoptions reapply` | Adopted file is overwritten from a selected version; local edits require confirmation | Service and handler tests |
| Diff versions/adoptions | versions | CLI/API/UI diff request | Server returns aligned line diff rows | Versions service and handler tests |
| Graduate scenario component | components / experience | Scenario UI component becomes reusable | TSX, examples, and experience-component claims land in the catalog as one versioned contract | Catalog conformance, preview e2e, experience phase |
| Preview workspace experiment | preview | User focuses a rendered specimen and applies temporary JSON props | Exactly that iframe rerenders from an in-memory shallow merge; Reset/reload restores the indexed example | Component editor UI tests, preview harness tests, preview E2E, BAS workflow |

## Apply Component

1. Resolve the component and its manifest-pinned dependency closure.
2. Use requested version, or the manifest latest when no version is
   supplied.
3. Reject an existing target path unless overwrite is confirmed.
4. Write a provenance header with library id, version, adoption id,
   applied timestamp, and source sha.
5. Copy the full editable source body into the target scenario.
6. Insert one direct parent adoption plus a provenance row for each materialized
   asset exactly once. Dependency rows retain their originating asset, library,
   and version so hooks can report mediated effective usage and link back to
   the parent component adoption.

## Assisted Work

1. The user supplies extract source or adopt target plus an idempotency key.
2. RCL persists a queued workflow before dispatching a narrow server-authored
   Agent Manager task/run.
3. Refresh/stop/retry operate on the durable ledger; no browser talks to Agent
   Manager and no polling loop claims success.
4. Terminal agent state remains evidence only. Direct RCL ingest/apply/reapply
   still performs parity, validation, overwrite, and provenance decisions.

## Promotion Readiness

`GetPromotionReadiness` is the read-only checkpoint between harvest and calling
an asset canonical. It reads the selected version's origin parity report
(including an explicit waiver), dependency closure, examples, and the origin
scenario's recorded replacement adoption plus drift status. It is ready only
when all those facts are available and clean. It never writes catalog or
scenario files, and Agent Manager terminal state is not proof of promotion.

The explicit mutation sequence remains: ingest the source, create the selected
release, apply or reapply it to the origin with the existing confirmations,
refresh drift, and run the returned validation command. Candidate suggestions
are separately read-only and do not authorize rollout.

## Refresh Drift

Refresh computes two dimensions:

- `library_version_status`: `current`, `behind`, `deprecated`,
  `missing`, or `unknown`.
- `local_status`: `clean`, `modified`, `missing`, or `unknown`.

This lets the UI distinguish a clean but behind copy from a locally
edited copy that is also behind.

## Graduate Scenario Component

Reusable component graduation carries code, examples, and experience
claims together. Do not move TSX alone.

1. Identify the scenario-local component and the page or state claims
   that prove why it exists.
2. Move the reusable TSX into
   `library/components/<Slug>/versions/<version>/<Slug>.tsx` and keep
   its `@libraryId`, `@version`, `@status`, and `@deps` headers aligned
   with `component.json`.
3. Move the component's representative states into
   `library/components/<Slug>/versions/<version>/examples.json`. Keep
   the examples data-only; use the `$` vocabulary for React nodes,
   icons, handlers, row keys, columns, and filters.
4. Copy the canonical experience contract into
   `library/components/<Slug>/versions/<version>/experience-contract.json`.
   That immutable version directory, not RCL's scenario-level `experience/`
   folder, is the reusable contract authority.
5. Replace the origin scenario's full component contract with a direct page
   library pin, or retain a local component only as an explicit additive
   wrapper. A wrapper declares `component.extends` and an extension purpose;
   it cannot reuse canonical lifecycle-state or claim identifiers.
6. Run `react-component-library components index --json` so SQLite
   projections for versions, dependencies, examples, and design
   affinities match Git.
7. Run the catalog gates: `pnpm run catalog:check` and
   `pnpm run test:preview-e2e` from `ui/`.
8. Run `test-genie execute react-component-library experience --json`
   so Experience Manager captures the preview harness and reconciles
component machine claims against the BAS accessibility tree.

## Preview Workspace Experiment

1. The editor queries indexed examples and renders each named example in its
   own sandboxed harness iframe.
2. The user can focus a specimen for inspection or add it to the bounded,
   deterministic comparison workspace; this changes host UI state only.
3. Try props presents the indexed `props` as JSON. Apply accepts only a JSON
   object, posts it to the matching registered iframe, and the harness
   shallow-merges it over indexed props using the existing `$` resolver.
4. A malformed or non-object value remains in the host with an inline error;
   it never reaches the iframe. A mismatched identity or message origin is
   ignored.
5. Reset, iframe reload, editor navigation, or unmount discards the override
   and restores the indexed props. No source file, SQLite row, localStorage
   entry, or write RPC is involved.
6. `setup` is not executed in this flow. It remains an indexed field awaiting
   a separate runtime contract.

## Component Test Contracts

RCL component tests are opt-in, declarative, and version-pinned. Place a
`test-contract.json` beside the selected version's source and examples. A
component contract names catalog examples; a hook contract names fixtures.
The runner resolves the manifest dependency closure in pinned order, checks the
restricted action/assertion vocabulary, and persists a normalized report.

```sh
react-component-library components test <component-id> --version 1.0.0 --closure true
react-component-library components test-list <component-id>
react-component-library components test-show <report-id>
react-component-library components test-rerun <report-id>
```

No contract can name a file, command, or arbitrary setup. In particular,
`examples.setup` is preview data and is never executed by the test runner. An
asset without a contract stays previewable and is reported as **blocked**
(uncovered), never as a passing test. Test Genie owns the scenario-level
provider phase; the catalog Test tab exposes the same durable report history.
Report URLs use `?tab=tests&testReport=<report-id>` so a durable CLI report can
be opened directly in the catalog. The preview browser sweep evaluates each
indexed example's safe role, text, and attribute expectations against the
rendered iframe DOM in both light and dark modes.

## Deferred Flows

## Voice Input Capability

`useVoiceInput` and `VoiceInputButton` are a linked, RCL-only capability. The
hook owns one browser capture owner, one timeout clock, and one idempotent
terminal funnel. Its injected adapter owns same-origin transport; an adopter
owns settings, transcript placement, and domain actions. The capability never
imports a scenario endpoint, provider, resource, or audio-capture package.

1. `start` acquires the injected media capture, subscribes to device-end, then
   connects the injected adapter. A start cue follows only after recording is
   active; there is no prewarm path.
2. In `always-on` mode, settled segments are forwarded in arrival order and
   silence is a segment boundary, never a capture stop. In `timeout` mode the
   injected clock is the sole countdown owner.
3. Explicit stop, timer expiry, device end, permission denial, and adapter
   failure all converge on one cleanup operation. It clears the timer,
   unsubscribes, stops the capture/adapter, and emits at most one stop cue.
4. Catalog examples are controlled, fake-backed visual specimens. They never
   request microphone permission or execute example setup in the preview iframe.

Web-console adoption, durable audio-tools streaming/recovery, provider/resource
simplification, and deletion of copied implementations remain follow-on work
after explicit user approval of the manual RCL checklist.

| Flow | Risk | Next Step |
|---|---|---|
| Draft/release lifecycle UI | Released version immutability is enforced by convention, not a full workflow model. | Add draft/release commands and tests when editing releases expands. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md)
- [`DATA.md`](DATA.md)
- [`../internal/TESTING.md`](../internal/TESTING.md)
