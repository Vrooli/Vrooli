# React-Vite Template Level 5 Temporal Flow Plan

**Plan file**: `path:docs/plans/react-vite-template-level-5-temporal-flow-plan.md`  
**Authored**: 2026-05-10  
**Owner**: meta-optimization / template-validation  
**Primary target**: `path:templates/scenarios/react-vite/`  
**Status**: ready for implementation

---

## 1. Purpose

Upgrade the `react-vite` scenario template from a strong Level 4 temporal-flow
reference to a true Level 5 reference.

The template already has the right Level 4 shape: domain-owned workflow code,
full transition matrices, representative traces, declarative `*.spec.json`
contracts, and conformance tests for both API and UI attachment-upload flows.
The missing Level 5 layer is the formal verification loop:

1. A domain-local Quint model is checked by the Quint toolchain.
2. Deterministic artifacts are generated from the model.
3. Production Go and TypeScript transition functions replay those artifacts in
   tests.
4. Validation fails when artifacts are stale or production behavior diverges.

This plan should leave future generated scenarios with a professional,
maintainable, copyable example of maximum temporal-flow maturity. The result
must be accurate enough that future agents do not confuse "a formal-looking
file exists" with "a checked formal model protects production behavior."

---

## 2. Required Reading

Run before implementing:

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read temporal-flow-audit screaming-architecture-audit test documentation-health reference-pattern-fitness
```

Read current template context:

- `path:templates/scenarios/react-vite/docs/concepts/FLOWS.md`
- `path:templates/scenarios/react-vite/docs/internal/TESTING.md`
- `path:templates/scenarios/react-vite/docs/internal/SEAMS.md`
- `path:templates/scenarios/react-vite/docs/internal/TEMPLATE-GENERATION-CONTRACT.md`
- `path:templates/scenarios/react-vite/docs/internal/TEMPLATE-MAINTENANCE.md`
- `path:templates/scenarios/react-vite/template.json`
- `path:templates/scenarios/react-vite/.vrooli/service.json`
- `path:templates/scenarios/react-vite/Makefile`
- `path:templates/scenarios/react-vite/api/internal/notes/attachment_workflow.go`
- `path:templates/scenarios/react-vite/api/internal/notes/attachment_workflow_test.go`
- `path:templates/scenarios/react-vite/api/internal/notes/attachment_upload_workflow.spec.json`
- `path:templates/scenarios/react-vite/api/internal/testutil/modeltest/`
- `path:templates/scenarios/react-vite/ui/src/features/notes/AttachmentUploadWorkflow.ts`
- `path:templates/scenarios/react-vite/ui/src/features/notes/AttachmentUploadWorkflow.test.ts`
- `path:templates/scenarios/react-vite/ui/src/features/notes/AttachmentUploadWorkflow.spec.json`
- `path:templates/scenarios/react-vite/ui/src/test-utils/modeltest/`

Tooling references:

- Quint CLI docs: `https://quint.sh/docs/quint`
- Quint model-based testing docs: `https://quint.sh/docs/model-based-testing`
- Quint model checker docs: `https://quint.sh/docs/model-checkers`

Discovery evidence from 2026-05-10:

```bash
quint --version
# 0.32.0

quint verify --help
# supports --invariants, --out-itf, --max-steps, --backend apalache|tlc

quint run --help
# supports --mbt, --seed, --n-traces, --max-steps, --out-itf

java -version
# OpenJDK 17 is installed locally
```

---

## 3. Greenfield Constraint

This is greenfield template work. Do not include compatibility shims, legacy
wrappers, dead code, unused re-exports, renamed `_unused` variables, or
migration paths for scenarios generated from older template revisions.

The template is the canonical source for new scenarios. It should teach the
best pattern directly. If the implementation reveals an older helper or doc
path that is now inferior, replace it cleanly inside this template instead of
preserving both patterns.

---

## 4. Hard Rules

1. **No documentation-only formal specs.** A Quint model is adopted only when
   `quint typecheck`, `quint test`, `quint verify`, generated artifacts, and
   production replay tests are all wired together.
2. **No package installs.** Quint and Java are project host tools. Use the host
   `quint` command and existing Node/Go runtimes. Do not add npm, Go, or system
   dependencies unless the operator explicitly approves.
3. **Domain ownership stays local.** Formal models for `notes` attachment upload
   live beside the owning API/UI workflow files. Shared mechanics may live in
   test utilities or a small template-local script, but product state vocabulary
   must stay in the owning domain.
4. **Artifacts must be deterministic.** Generated files must not include
   timestamps, absolute paths, machine-specific temp directories, random
   ordering, or local usernames. Use stable seeds and normalized output.
5. **Generated artifacts must be checked in.** Tests must consume generated
   artifacts from the template. A `--check` mode must fail when artifacts are
   stale.
6. **Production code is not generated from Quint in this pass.** The checked
   model validates production Go/TypeScript transition functions. It does not
   replace them.
7. **Plain CRUD remains plain CRUD.** Do not make every domain formal-methods
   shaped. The Level 5 example is for temporal flows with named lifecycle
   states and illegal transitions.

---

## 5. Problem Statement

`react-vite` now teaches temporal-flow maturity through Level 4:

- API workflow model:
  `path:templates/scenarios/react-vite/api/internal/notes/attachment_workflow.go`
- API matrix/spec/trace tests:
  `path:templates/scenarios/react-vite/api/internal/notes/attachment_workflow_test.go`
- API declarative spec:
  `path:templates/scenarios/react-vite/api/internal/notes/attachment_upload_workflow.spec.json`
- UI workflow model:
  `path:templates/scenarios/react-vite/ui/src/features/notes/AttachmentUploadWorkflow.ts`
- UI matrix/spec/trace tests:
  `path:templates/scenarios/react-vite/ui/src/features/notes/AttachmentUploadWorkflow.test.ts`
- UI declarative spec:
  `path:templates/scenarios/react-vite/ui/src/features/notes/AttachmentUploadWorkflow.spec.json`

The Level 4 setup is useful and should remain. It proves every hand-authored
state/event pair and keeps JSON specs aligned with production tests.

The gap is that the formal-model status still says `not_adopted`, and the
template docs explicitly reserve Level 5 for a checked model that produces
artifacts replayed by production tests. Since Quint and Java host tools now
exist, the template can become the first real Level 5 reference instead of
documenting Level 5 as future work.

The risk is doing this superficially. A standalone `.qnt` file beside the
workflow would make the template look mature while adding a new drift surface.
The implementation must close the loop end to end.

---

## 6. Scope

### In Scope

- Add checked Quint models for the API and UI attachment-upload workflows.
- Add a small deterministic artifact-generation/check script using existing
  runtimes and the host `quint` command.
- Add generated formal artifacts that production Go/TypeScript tests replay.
- Extend API `modeltest` helpers to validate formal artifact metadata and
  replay generated transitions/traces against production Go transition logic.
- Extend UI `modeltest` helpers to validate formal artifact metadata and replay
  generated transitions/traces against production TypeScript transition logic.
- Update the existing workflow tests to consume the generated artifacts.
- Update `*.spec.json` `formalModel` blocks from `not_adopted` to `adopted`
  only after the loop is executable.
- Wire formal validation into `Makefile`, lifecycle/test commands, and template
  deep validation without making routine docs-only checks expensive.
- Update template docs and docs manifest maturity claims.
- Add scenario-level `hostTools` declarations for `quint` and `java` if the
  generated scenario's own test lifecycle invokes formal checks.

### Out of Scope

- Installing Quint, Java, Apalache, TLC, or npm packages.
- Migrating existing generated scenarios.
- Building a generic project-wide formal-model framework before the template
  pattern proves itself.
- Generating production Go or TypeScript transition functions from Quint.
- Replacing current Level 4 matrix/spec tests.
- Adding formal models for CRUD-only notes behavior.
- Reworking unrelated template validation, proto generation, Connect-RPC, UI
  design, CLI command structure, or lifecycle supervision.

---

## 7. Current Technical Context

### Current Maturity

The template is Level 4 for the attachment-upload temporal flows:

- Pure workflow transition functions exist in API and UI.
- Tests cover every state/event pair.
- Tests replay representative traces.
- Tests compare hand-authored matrices/traces with declarative JSON specs.
- `formalModel.status` is `not_adopted`.

### Existing API Workflow Semantics

API states:

- `received`
- `bytes_stored`
- `metadata_recorded`
- `failed`

API events:

- `store_bytes`
- `record_metadata`
- `fail`

API terminal states:

- `metadata_recorded`
- `failed`

Important API invariant:

- Metadata cannot be recorded before bytes are stored.

### Existing UI Workflow Semantics

UI states:

- `idle`
- `selected`
- `uploading`
- `succeeded`
- `failed`

UI events:

- `select`
- `start`
- `succeed`
- `fail`
- `reset`

Important UI invariants:

- `succeeded` carries a non-empty `fileName`.
- `failed` carries a non-empty `message`.
- `selected`, `uploading`, and `failed` carry enough file context to submit or
  retry.

### Existing Toolchain

Root host tools now include `quint` and `java`. Local checks on 2026-05-10
showed:

- `quint --version` returns `0.32.0`.
- `quint verify` exposes Apalache/TLC verification flags.
- `quint run --mbt --out-itf` exposes model-based trace generation.
- `java -version` reports OpenJDK 17.

### Template Validation Context

`vrooli scenario template validate --mode deep --template react-vite` generates
a real temporary scenario and runs test-genie against it. Any Level 5 pipeline
must survive both template-local focused tests and generated-scenario deep
validation.

---

## 8. Target End State

The `react-vite` template contains a fully executable Level 5 temporal-flow
reference:

```text
templates/scenarios/react-vite/
  tools/temporal-model/
    README.md
    generate.mjs
    generated-artifact.schema.json

  api/internal/notes/
    attachment_upload_workflow.qnt
    attachment_upload_workflow.formal.generated.json
    attachment_upload_workflow.spec.json
    attachment_workflow.go
    attachment_workflow_test.go

  ui/src/features/notes/
    AttachmentUploadWorkflow.qnt
    AttachmentUploadWorkflow.formal.generated.json
    AttachmentUploadWorkflow.spec.json
    AttachmentUploadWorkflow.ts
    AttachmentUploadWorkflow.test.ts
```

The exact script path may change during implementation if an existing template
script folder is a better fit, but the ownership rule must not change: Quint
models and generated artifacts stay beside the workflow they validate.

The generated artifact format is stable, minimal, and intentionally reusable:

```json
{
  "schemaVersion": 1,
  "flowId": "notes.attachment-upload.api",
  "source": {
    "modelPath": "api/internal/notes/attachment_upload_workflow.qnt",
    "modelSha256": "<sha256>",
    "quintVersion": "0.32.0"
  },
  "commands": {
    "typecheck": ["quint", "typecheck", "..."],
    "test": ["quint", "test", "..."],
    "verify": ["quint", "verify", "..."],
    "run": ["quint", "run", "..."]
  },
  "states": ["received"],
  "events": ["store_bytes"],
  "transitions": [
    {
      "from": "received",
      "event": "store_bytes",
      "to": "bytes_stored",
      "wantError": false
    }
  ],
  "traces": [
    {
      "name": "generated_success_001",
      "initial": "received",
      "steps": [
        { "event": "store_bytes", "want": "bytes_stored" }
      ]
    }
  ],
  "checks": {
    "typechecked": true,
    "tested": true,
    "verified": true,
    "generatedFromModel": true
  }
}
```

The final shape should keep volatile Quint/Apalache output out of git and check
in only normalized artifacts.

---

## 9. Implementation Strategy

### Phase 1: Quint Modeling Spike

Goal: prove the exact Quint syntax and commands before touching the template
broadly.

Actions:

1. Create a temporary scratch Quint model outside committed template paths.
2. Model the API attachment-upload workflow first because its state shape is
   simpler than the UI discriminated union.
3. Verify these commands work locally:

   ```bash
   quint typecheck /tmp/attachment_upload_workflow.qnt
   quint test /tmp/attachment_upload_workflow.qnt --seed 20260510
   quint verify /tmp/attachment_upload_workflow.qnt --invariants invariant --max-steps 8
   quint run /tmp/attachment_upload_workflow.qnt --mbt --seed 20260510 --n-traces 8 --max-steps 8 --out-itf /tmp/attachment_upload_{seq}.itf.json
   ```

4. Confirm generated ITF output contains enough information to normalize
   state/event traces.
5. Record the chosen Quint action, invariant, and witness names in the plan
   implementation notes or final handoff.

Acceptance criteria:

- A scratch model typechecks.
- A scratch model verifies at bounded depth.
- `quint run --mbt --out-itf` emits parseable ITF JSON.
- The implementation knows exactly how transition events will be represented in
  the model so generated traces can be replayed by production tests.

### Phase 2: Add Domain-Local Quint Models

Goal: add the formal source files beside the workflows.

Proposed files:

- `path:templates/scenarios/react-vite/api/internal/notes/attachment_upload_workflow.qnt`
- `path:templates/scenarios/react-vite/ui/src/features/notes/AttachmentUploadWorkflow.qnt`

Model requirements:

- Use the same flow IDs as the existing JSON specs:
  - `notes.attachment-upload.api`
  - `notes.attachment-upload.ui`
- Declare finite state and event domains.
- Encode the existing transition rules exactly.
- Encode terminal-state behavior exactly.
- Encode invariants in model-checkable form.
- Include named tests for successful and rejected traces.
- Avoid UI/file payload details that Quint does not need. Model payload-bearing
  states as abstract lifecycle states, while production TypeScript invariant
  tests keep checking payload-specific rules.

Quality bar:

- Anyone reading the Quint model should recognize the same states/events as the
  Go/TypeScript workflow and JSON spec.
- No transition should exist only in one representation.

### Phase 3: Add Deterministic Artifact Generation

Goal: build a small, dependency-free generator/checker for formal artifacts.

Proposed files:

- `path:templates/scenarios/react-vite/tools/temporal-model/generate.mjs`
- `path:templates/scenarios/react-vite/tools/temporal-model/generated-artifact.schema.json`
- `path:templates/scenarios/react-vite/tools/temporal-model/README.md`

Command shape:

```bash
node tools/temporal-model/generate.mjs
node tools/temporal-model/generate.mjs --check
node tools/temporal-model/generate.mjs --flow notes.attachment-upload.api
```

Generator responsibilities:

- Discover a small explicit registry of formal flows.
- Run `quint --version` and require the expected major/minor version or record
  the exact version and fail if generated artifacts were produced by a different
  version than the one executing `--check`.
- Run `quint typecheck`.
- Run `quint test`.
- Run `quint verify` with bounded depth and named invariants.
- Run `quint run --mbt` with fixed seeds and fixed trace counts.
- Normalize ITF files into stable artifact JSON.
- Hash the `.qnt` source and relevant generator version string.
- Sort arrays and object keys deterministically.
- In `--check` mode, compare would-be output to the checked-in artifact and
  fail with a clear message if stale.
- Clean temporary ITF output automatically.

Do not:

- Add npm dependencies.
- Depend on absolute paths in generated JSON.
- Store raw Apalache output unless it is normalized and stable.
- Include generated timestamps.

### Phase 4: Extend API Modeltest Helpers

Goal: make Go production tests consume formal artifacts.

Likely files:

- `path:templates/scenarios/react-vite/api/internal/testutil/modeltest/formal.go`
- `path:templates/scenarios/react-vite/api/internal/testutil/modeltest/formal_test.go`
- possibly extend `path:templates/scenarios/react-vite/api/internal/testutil/modeltest/spec.go`

New helper shape:

```go
artifact := modeltest.LoadFormalArtifact(t, "attachment_upload_workflow.formal.generated.json")
modeltest.AssertFormalArtifactFresh(t, artifact, "attachment_upload_workflow.qnt")
modeltest.AssertFormalTransitionsReplay(
    t,
    artifact,
    notes.AllAttachmentUploadStatuses(),
    notes.AllAttachmentUploadEvents(),
    transition,
)
modeltest.AssertFormalTracesReplay(t, artifact, transition)
```

API helper responsibilities:

- Parse generated artifacts.
- Validate schema version, flow ID, model hash, state/event coverage, and
  declared check results.
- Replay generated transitions against the production Go transition function.
- Replay generated traces step by step.
- Fail if a generated artifact references unknown production state/event names.
- Fail if a production state/event is missing when the artifact claims to
  include full transition coverage.

### Phase 5: Extend UI Modeltest Helpers

Goal: make TypeScript production tests consume formal artifacts.

Likely files:

- `path:templates/scenarios/react-vite/ui/src/test-utils/modeltest/formal.ts`
- `path:templates/scenarios/react-vite/ui/src/test-utils/modeltest/formal.test.ts`
- `path:templates/scenarios/react-vite/ui/src/test-utils/index.ts`

New helper shape:

```ts
import formalArtifact from "./AttachmentUploadWorkflow.formal.generated.json";

assertFormalArtifactFresh(formalArtifact, {
  modelPath: "ui/src/features/notes/AttachmentUploadWorkflow.qnt",
});
assertFormalTransitionsReplay(
  formalArtifact,
  attachmentUploadStatuses,
  attachmentUploadEvents,
  transitionStatus,
);
assertFormalTracesReplay(formalArtifact, transitionStatus);
```

UI helper responsibilities mirror the Go helpers. The UI state factory already
maps abstract status strings into payload-bearing TypeScript states. Keep that
adapter in the test file, not inside the generic formal helper.

### Phase 6: Update Attachment Workflow Tests

Goal: preserve Level 4 tests and add Level 5 tests.

API updates:

- Import/load `attachment_upload_workflow.formal.generated.json`.
- Add a focused test such as
  `TestAttachmentUploadWorkflow_ReplaysFormalModelArtifacts`.
- Keep existing transition matrix, trace, spec-conformance, and unknown-state
  tests.

UI updates:

- Import `AttachmentUploadWorkflow.formal.generated.json`.
- Add a focused test such as
  `it("replays generated formal model artifacts", ...)`.
- Keep existing matrix, trace, spec-conformance, and invariant tests.

The new tests must prove three separate failure modes:

- stale model/artifact hash fails,
- generated trace diverging from production transition fails,
- generated transition containing unknown state/event fails.

These failure modes can be covered in helper unit tests rather than by mutating
the real generated artifacts.

### Phase 7: Update Manifest, Lifecycle, and Make Targets

Goal: make formal validation part of normal generated-scenario quality gates.

Template `.vrooli/service.json`:

- Add scenario-level `hostTools` for `quint` and `java` if lifecycle `test`
  invokes formal checks in generated scenarios.
- Use `required: true` for `quint` and `java` if the test phase cannot pass
  without them.
- Keep reasons explicit:
  - `quint`: checked temporal-flow models and generated conformance artifacts
  - `java`: model checker runtime used by `quint verify`

Makefile:

- Add `formal` or `temporal-models` target:

  ```make
  temporal-models:
  	@node tools/temporal-model/generate.mjs --check
  ```

- Include the target in `test` only if `vrooli scenario test` / test-genie also
  invokes it. Avoid a Makefile-only gate that deep validation does not exercise.

Lifecycle test phase:

- Prefer adding a test-genie phase or scenario-local command that runs:

  ```bash
  node tools/temporal-model/generate.mjs --check
  ```

- If test-genie already has a suitable custom command or phase mechanism, use
  that instead of adding a one-off lifecycle hack.

Orientation:

- Add a generated-scenario orientation check that formal artifacts are fresh
  before the scaffold is considered healthy.

### Phase 8: Update Documentation

Goal: make the Level 5 contract clear and hard to misread.

Update:

- `path:templates/scenarios/react-vite/docs/concepts/FLOWS.md`
- `path:templates/scenarios/react-vite/docs/internal/TESTING.md`
- `path:templates/scenarios/react-vite/docs/internal/SEAMS.md`
- `path:templates/scenarios/react-vite/docs/internal/TEMPLATE-GENERATION-CONTRACT.md`
- `path:templates/scenarios/react-vite/docs/internal/TEMPLATE-MAINTENANCE.md`
- `path:templates/scenarios/react-vite/docs/START-HERE.md`
- `path:templates/scenarios/react-vite/docs/manifest.json`

Documentation must say:

- The attachment-upload API/UI workflows are the Level 5 reference.
- Level 5 means checked model plus generated artifacts plus production replay
  tests plus stale-artifact failure.
- Quint models are not accepted as standalone docs.
- Plain CRUD domains should not copy this ceremony unless they have lifecycle
  constraints.
- Formal artifacts are checked in because generated scenarios need a passing
  baseline and reviewers need stable diffs.

### Phase 9: Template Validation

Goal: prove both template source and generated scenario outputs are clean.

Run focused validation first:

```bash
node templates/scenarios/react-vite/tools/temporal-model/generate.mjs --check
(cd templates/scenarios/react-vite/api && GOWORK=off go test ./internal/notes ./internal/testutil/modeltest)
(cd templates/scenarios/react-vite/ui && corepack pnpm test -- --run AttachmentUploadWorkflow modeltest)
```

Then run broad template validation:

```bash
vrooli scenario template validate --mode shallow --template react-vite --warning-policy report
vrooli scenario template validate --mode deep --template react-vite --test-preset comprehensive --warning-policy report
```

If deep validation is expensive, run it before final handoff rather than after
every small edit. Do not mark the plan complete until deep validation passes or
the exact blocker is documented with logs and next steps.

---

## 10. Contract Decisions

### Formal Artifact Contract

Generated formal artifacts are committed source artifacts, not cache files.
They are part of the template contract and generated-scenario baseline.

Required fields:

- `schemaVersion`
- `flowId`
- `source.modelPath`
- `source.modelSha256`
- `source.quintVersion`
- `commands`
- `states`
- `events`
- `transitions`
- `traces`
- `checks`

Forbidden fields:

- timestamps
- absolute paths
- username/home-directory fragments
- temp directory paths
- raw nondeterministic ordering

### Stale Artifact Contract

`node tools/temporal-model/generate.mjs --check` must fail when:

- `.qnt` source hash differs,
- generator normalization logic differs,
- Quint version differs in a way the implementation treats as significant,
- generated transitions/traces differ,
- checked-in artifact JSON is not canonical.

### Production Replay Contract

Go and TypeScript tests must fail when:

- a generated transition cannot be applied by production transition logic,
- a generated trace diverges from production transition logic,
- generated artifacts mention an unknown production state or event,
- generated artifacts omit a required production state or event while claiming
  full coverage,
- generated artifacts claim formal checks passed when required check flags are
  false or missing.

### Spec JSON Contract

Existing `*.spec.json` files remain the human-readable declarative workflow
contracts. Their `formalModel` block changes to `adopted` only after formal
validation is wired.

Suggested shape:

```json
"formalModel": {
  "status": "adopted",
  "tool": "Quint 0.32.0",
  "model": "attachment_upload_workflow.qnt",
  "generatedArtifacts": "attachment_upload_workflow.formal.generated.json",
  "driftCheck": "quint typecheck/test/verify plus generated artifact replay in Go tests"
}
```

### Host Tool Contract

If generated scenarios run the formal check as part of their test lifecycle,
the template scenario manifest must declare `quint` and `java` in `hostTools`.
Root-level declarations are not enough for a portable generated-scenario
contract.

---

## 11. Testing Plan

### Unit Tests

API:

```bash
(cd templates/scenarios/react-vite/api && GOWORK=off go test ./internal/testutil/modeltest)
(cd templates/scenarios/react-vite/api && GOWORK=off go test ./internal/notes)
```

UI:

```bash
(cd templates/scenarios/react-vite/ui && corepack pnpm test -- --run modeltest)
(cd templates/scenarios/react-vite/ui && corepack pnpm test -- --run AttachmentUploadWorkflow)
```

Generator:

```bash
node templates/scenarios/react-vite/tools/temporal-model/generate.mjs --check
```

Add generator tests if the script grows past straightforward command
orchestration and normalization. Prefer Node's built-in `node:test` if tests are
needed, to avoid dependencies.

### Integration Tests

Template source:

```bash
vrooli scenario template validate --mode shallow --template react-vite --warning-policy report
```

Generated scenario:

```bash
vrooli scenario template validate --mode deep --template react-vite --test-preset comprehensive --warning-policy report
```

### Drift Tests

Add helper-level tests that intentionally mutate in-memory artifact copies and
assert failures for:

- stale hash,
- unknown state,
- unknown event,
- missing required state/event,
- divergent transition,
- divergent trace,
- false `checks.verified`.

Do not mutate checked-in generated artifact files in tests.

### Manual Sanity Commands

Before final handoff:

```bash
quint --version
java -version
quint typecheck templates/scenarios/react-vite/api/internal/notes/attachment_upload_workflow.qnt
quint typecheck templates/scenarios/react-vite/ui/src/features/notes/AttachmentUploadWorkflow.qnt
```

---

## 12. Rollout and Validation Checklist

- [ ] Scratch Quint model proves syntax, typecheck, test, verify, and MBT trace
  generation.
- [ ] API `.qnt` model exists beside API workflow.
- [ ] UI `.qnt` model exists beside UI workflow.
- [ ] Deterministic generator/checker exists and has no third-party package
  dependency.
- [ ] API formal generated artifact is checked in and stable.
- [ ] UI formal generated artifact is checked in and stable.
- [ ] API modeltest helpers replay formal artifacts against production Go
  transition logic.
- [ ] UI modeltest helpers replay formal artifacts against production
  TypeScript transition logic.
- [ ] Existing Level 4 matrix/spec/trace tests still pass.
- [ ] New Level 5 tests fail on stale or divergent artifacts.
- [ ] `*.spec.json` formalModel blocks accurately say `adopted`.
- [ ] Template manifest declares required host tools if lifecycle tests need
  them.
- [ ] Template docs distinguish CRUD, Level 4, and Level 5 accurately.
- [ ] Shallow template validation passes.
- [ ] Deep template validation passes with comprehensive preset.
- [ ] No compatibility shims, dead code, or unused wrappers were added.

---

## 13. Risks and Mitigations

| Risk | Why It Matters | Mitigation |
|---|---|---|
| Quint syntax/modeling takes longer than expected | A broken or vague model would make the template worse | Do Phase 1 scratch spike first; do not edit docs to claim Level 5 until commands pass |
| Generated artifacts become nondeterministic | Template diffs become noisy and deep validation becomes unreliable | Fixed seeds, sorted JSON, no timestamps, normalized relative paths |
| Formal model drifts from production code | This is the exact failure Level 5 must prevent | Tests replay generated artifacts against Go/TS transitions; `--check` verifies artifacts are fresh |
| Template becomes too heavy for simple scenarios | Future agents may over-model CRUD | Docs must explicitly say plain CRUD stays plain; only temporal flows copy this pattern |
| `quint verify` is slow or flaky in routine tests | Developers stop trusting validation | Keep model small and bounded; reserve heavy checks for generated artifact refresh/check and deep validation if needed |
| Java/Quint missing in generated scenario environment | Tests fail with unclear host issues | Declare scenario `hostTools` when lifecycle tests require them; keep error messages explicit |
| Artifact generator becomes a hidden framework | Template maintainers inherit unnecessary complexity | Keep generator small, explicit, and flow-registry based; do not generalize beyond the two reference flows |
| Raw ITF output does not expose events cleanly | Replay artifacts cannot be precise | Resolve in Phase 1; if needed, model event as explicit state metadata or emit transition rows from model tests instead |

---

## 14. Non-Goals and Prohibited Patterns

Do not:

- add standalone formal docs with no executable check,
- remove current matrix/spec/trace tests,
- generate production transition code from Quint,
- add XState or another state-machine runtime,
- add npm packages for the generator,
- add generic cross-template formal tooling before this template-local pattern
  proves itself,
- hide temporal rules in React components, HTTP handlers, or generic utilities,
- make notes CRUD behavior formal-model-shaped,
- include compatibility paths for older generated scenarios,
- claim Level 5 in docs or specs until deep validation proves the full loop.

---

## 15. Definition of Done

This plan is complete only when all of the following are true:

1. `react-vite` has domain-local Quint models for API and UI attachment-upload
   workflows.
2. `quint typecheck`, `quint test`, and `quint verify` run successfully for
   both models through the template's formal validation path.
3. Deterministic generated formal artifacts are checked in for both workflows.
4. `node tools/temporal-model/generate.mjs --check` fails on stale artifacts and
   passes on committed artifacts.
5. API Go tests replay generated formal artifacts against
   `TransitionAttachmentUpload`.
6. UI Vitest tests replay generated formal artifacts against
   `transitionAttachmentUpload`.
7. Existing Level 4 matrix/spec/trace conformance remains intact.
8. `docs/concepts/FLOWS.md`, `docs/internal/TESTING.md`, and
   `docs/manifest.json` accurately document the Level 5 setup.
9. Scenario host requirements are explicit for Quint/Java if generated scenario
   tests depend on them.
10. Focused API/UI/generator tests pass.
11. `vrooli scenario template validate --mode shallow --template react-vite`
    passes.
12. `vrooli scenario template validate --mode deep --template react-vite
    --test-preset comprehensive --warning-policy report` passes, or any blocker
    is documented with exact failing command, output summary, and next action.
13. The final diff contains no compatibility shims, legacy wrappers, dead code,
    or unused indirection.

