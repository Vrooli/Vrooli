# Flows — React Component Library

This document is the canonical workflow map for ordered behavior.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Validation |
|---|---|---|---|---|
| Index library | components | CLI/API/UI `components index` | Manifests and version folders are validated and reflected in SQLite | Indexer, repository, handler, CLI tests |
| Apply component | adoptions | CLI/API/UI `adoptions apply` | Selected version source is copied into a target scenario with provenance and an adoption row | Service, handler, CLI, UI tests |
| Preflight styling contract | adoptions | CLI/API `adoptions preflight` | Dependency, style-fit, version, maturity, and token findings are combined into one read-only adoptability verdict | Service and handler tests |
| Sync scenario ramp | adoptions | CLI/API `adoptions tokens-sync` | Missing closure tokens are added only inside the managed design-token region | Ramp parser and service tests |
| Prune scenario ramp | adoptions | CLI/API `adoptions tokens-prune` | Unused managed declarations are reported, then removed only with explicit apply | Ramp parser and service tests |
| Batch apply | adoptions | API/CLI batch apply | Several roots share one union closure and persistence transaction | Service, handler, and CLI tests |
| Assisted extract/adopt | workflows | CLI/API/UI `workflows start` | RCL records a scoped Agent Manager task/run and exposes honest queued/running/terminal status | Workflow service, handler, CLI tests |
| Promotion readiness | workflows | CLI/API/UI `workflows promotion-readiness` | Read-only evidence report joins parity, examples, dependency closure, origin replacement, and drift | Workflow readiness service, handler, CLI, UI tests |
| Refresh drift | adoptions | CLI/API/UI `adoptions refresh` | Adoption rows receive separate library-version and local-edit statuses | Service status matrix and UI tests |
| Reapply component | adoptions | CLI/API `adoptions reapply` | Adopted file is overwritten from a selected version; local edits require confirmation | Service and handler tests |
| Diff versions/adoptions | versions | CLI/API/UI diff request | Server returns aligned line diff rows | Versions service and handler tests |
| Graduate scenario component | components / experience | Scenario UI component becomes reusable | TSX, story contract, and experience-component claims land in the catalog as one versioned contract | Catalog conformance, preview e2e, experience phase |
| Story workbench | preview | User selects a named story and varies generated Args or explicit environment fixtures | Exactly that iframe rerenders from validated effective story args; Reset/reload restores the named baseline | Component editor UI tests, preview harness tests, preview E2E, BAS workflow |

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
edited copy that is also behind. A fixed-version hash mismatch is reported as
`source_drifted`, which means the released bytes changed under the adoption
and must be repaired as a version-integrity issue.

## Adoptability preflight

Preflight resolves the complete pinned closure without writing files. It
returns the dependency verdict, style-fit verdict, version status, achieved
maturity rung, required and defined CSS properties, and the blocking decision.
Apply and reapply use the same verdict. A missing managed token ramp is
remediated with `adoptions tokens-sync`; an explicit override is recorded as
an operator decision rather than hiding the finding.

## Asset update and batch adoption

Asset source changes are published as a new version. Released version bytes
are immutable, so reindexing a changed released folder fails. Operators then
refresh adopters, classify local drift, and reapply only clean copies. Several
roots can be submitted as one batch so shared dependencies are resolved once
and target collisions are rejected before any write.

## Graduate Scenario Component

Reusable component graduation carries code, a story contract, and experience
claims together. Do not move TSX alone.

1. Identify the scenario-local component and the page or state claims
   that prove why it exists.
2. Move the reusable TSX into
   `library/components/<Slug>/versions/<version>/<Slug>.tsx` and keep
   its `@libraryId`, `@version`, `@status`, and `@deps` headers aligned
   with `component.json`.
3. Define representative states in
   `library/components/<Slug>/versions/<version>/story.json`. Declare the
   public args schema once, then keep stories data-only; use the `$` vocabulary
   only for allowlisted React nodes,
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
   projections for versions, dependencies, stories, and design
   affinities match Git.
7. Run the catalog gates: `pnpm run catalog:check` and
   `pnpm run test:preview-evidence` from `ui/`; this delegates browser capture to BAS and persists the evidence manifest through `CatalogService.CaptureEvidence`.
8. Run `test-genie execute react-component-library experience --json`
   so Experience Manager captures the preview harness and reconciles
component machine claims against the BAS accessibility tree.

## Story Workbench

1. The editor queries indexed stories as named states and renders the selected
   story in its own sandboxed harness iframe.
2. The workbench keeps story navigation, a dominant canvas, generated Args,
   explicit Environment controls, status, and Reset together. Narrow screens
   move contextual tools into focused sheets. Comparison remains intentional
   host UI state only.
3. The Args form is generated from one asset-level schema. Editing a field
   updates only that path, validates the complete effective args, and posts
   the resulting data-only object to the matching iframe. Raw JSON is a
   diagnostic view, never the default input path.
4. A malformed, unsupported, or invalid value is retained in the host with a
   field-addressable error; the last valid iframe render remains. A mismatched
   identity or message origin is ignored.
5. Environment controls select only contract-declared provider/adapter
   fixtures. Internal component state is reached through real interactions or
   public controlled/default APIs; hook internals are never mutated.
6. Reset, iframe reload, editor navigation, or unmount discards the session
   edit and restores the named story. No source file, SQLite row, localStorage
   entry, or write RPC is involved. Arbitrary setup is never executed.

## Component Test Contracts

RCL component tests are version-pinned story runs. The selected version's
`story.json` names valid component stories or hook fixtures.
The runner resolves the manifest dependency closure in pinned order, checks the
restricted action/assertion vocabulary, and persists a normalized report.

```sh
react-component-library components test "<component-id>" --version 1.0.0 --closure true
react-component-library components test-list "<component-id>"
react-component-library components test-show "<report-id>"
react-component-library components test-rerun "<report-id>"
```

No contract can name a file, command, arbitrary setup, or mutable hook
implementation. An asset without a contract is an actionable conformance
failure, never a passing test. Test Genie owns the scenario-level
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
4. Catalog stories are controlled, fake-backed visual specimens. They never
   request microphone permission or execute arbitrary setup in the preview iframe.

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
