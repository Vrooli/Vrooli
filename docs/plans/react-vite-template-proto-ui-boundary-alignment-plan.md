# React Vite Template Proto UI Boundary Alignment Plan

## 1. Purpose

Bring the Prompt Manager UI/interoperability skills and the `react-vite`
scenario template into one clean, greenfield architecture for UI↔API contracts.
The target is a professional default that makes domain additions hard to drift:
proto schemas are the source of truth, API/UI/CLI boundaries use proto tooling,
UI API clients live in an explicit `src/api/` service layer, and validation
rules are documented as one coherent policy rather than scattered guidance.

This plan intentionally excludes proto `service`/RPC generation as current work.
It should be mentioned only as a future enhancement once the JSON/proto boundary
pattern is stable.

## 2. Required Reading

Future agents executing this plan must run:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health skill-authoring-practice
prompt-manager skill read react-coherence react-stability interoperability-steer
prompt-manager skill read api-steer cli-steer seam-discovery-and-enforcement utils-unification
```

Read these source files before editing:

```bash
sed -n '1,260p' scenarios/prompt-manager/store/skills/packs/core/react-coherence/SKILL.md
sed -n '1,380p' scenarios/prompt-manager/store/skills/packs/core/react-stability/SKILL.md
sed -n '1,520p' scenarios/prompt-manager/store/skills/packs/core/interoperability-steer/SKILL.md
sed -n '1,140p' packages/proto/README.md
find templates/scenarios/react-vite/ui/src -maxdepth 3 -type f | sort
```

## 3. Hard Rule: Greenfield, No Compatibility Shims

This is greenfield template and skill alignment work. Do not add compatibility
wrappers, legacy re-exports, fallback API client locations, or prose that says
"we used to do X." The finished guidance should state the intended architecture
directly.

Existing generated scenarios are out of scope. This plan updates the template
and the skills so future work starts from the right structure.

## 4. Problem Statement

The template has started moving in the right direction:

- `ui/src/lib/api.ts` centralizes `protoFetch`, `ApiError`, and envelope parsing.
- `ui/src/lib/notes.ts` is a thin domain client over generated proto descriptors.
- CLI changes are moving to `cliapp.ArgSchema`, `RunContext`, and typed proto calls.

But the current state is not yet the intended long-term shape:

1. The UI API boundary lives under `src/lib/`, which is vague and encourages
   future utility/API mixing. The clearer boundary is `src/api/`.
2. `protoFetch` serializes requests with `toJsonString` but does not explicitly
   request proto field names. Buf v2 defaults outbound JSON to lowerCamel; Vrooli
   APIs generally use snake_case/proto field names.
3. API ingress still decodes JSON requests with `encoding/json`, not
   `protojson`. That can reject or drop payloads differently from UI/CLI proto
   serialization, especially for fields with underscores, timestamps, enums,
   optional presence, and nested messages.
4. Prompt Manager skills contain stale or overly broad guidance:
   - `interoperability-steer` references `jsonOptions: { useProtoNames: true }`,
     but current Buf v2 uses `useProtoFieldName` for writing and has no read-side
     `useProtoNames` option.
   - `react-stability` presents Zod mirrors as the default validation chain, which
     risks creating a second source of truth beside proto.
   - `react-coherence` describes an ideal `shared/services` layer, while
     `interoperability-steer` points to `ui/src/api/`; the two should converge on
     a clear rule for scenario templates.
5. The template docs still describe `lib/api` and `lib/notes` as canonical, which
   will teach new domains the wrong final location.

## 5. Scope

### In scope

- Update these Prompt Manager skills:
  - `scenarios/prompt-manager/store/skills/packs/core/react-coherence/SKILL.md`
  - `scenarios/prompt-manager/store/skills/packs/core/react-stability/SKILL.md`
  - `scenarios/prompt-manager/store/skills/packs/core/interoperability-steer/SKILL.md`
- Update `packages/proto/README.md` TypeScript casing examples to the current
  Buf v2 API.
- Update `templates/scenarios/react-vite/` UI/API boundary structure and docs.
- Add tests that lock the intended proto JSON behavior at the UI and API
  boundaries.
- Keep the notes domain as the canonical worked example.

### Out of scope

- Migrating existing scenarios.
- Introducing proto `service`/RPC generation.
- Adding new dependencies without explicit approval.
- Reworking unrelated UI design, i18n, gamepad/spatial nav, or CLI behavior.

## 6. Current Technical Context

| Area | Current state |
|---|---|
| UI API client | `templates/scenarios/react-vite/ui/src/lib/api.ts` owns `protoFetch`; `lib/notes.ts` owns note calls. |
| Desired UI API client | `ui/src/api/client.ts` for substrate, `ui/src/api/<domain>.ts` for domain clients, `ui/src/lib/` for pure utilities only. |
| API request decoding | `api/internal/httpx/decode.go` uses `encoding/json.Decoder` with `DisallowUnknownFields`. |
| API response encoding | Notes handler writes proto responses via `protojson.MarshalOptions{UseProtoNames: true}`. |
| UI proto read behavior | Buf v2 `fromJson` accepts both proto field names and JSON/lowerCamel names; `ignoreUnknownFields` controls unknown-field tolerance. |
| UI proto write behavior | Buf v2 `toJsonString` defaults to lowerCamel; `{ useProtoFieldName: true }` emits snake_case/proto field names. |
| Skill drift | Skills still mention `useProtoNames` / `jsonOptions` and Zod mirrors as default runtime validation. |

## 7. Target End State

### UI structure

The template uses this shape:

```text
ui/src/
  api/
    client.ts        # protoFetch, ApiError, decodeApiError, makeApiError
    health.ts        # fetchHealth
    notes.ts         # listNotes/createNote/getNote
  features/
    health/
    notes/
  components/
  hooks/
  i18n/
  lib/
    utils.ts         # pure local utilities only
```

Rules:

- `src/api/client.ts` is the only production UI module that calls raw `fetch`.
- `src/api/<domain>.ts` is the only place domain endpoint paths and generated
  request/response descriptors are combined.
- Feature components and hooks call `src/api/<domain>.ts` through React Query.
- `src/lib/` must not contain network clients.

### Contract and serialization policy

- Proto schemas are the source of truth for request, response, and error shapes.
- JSON API ingress/egress uses proto tooling:
  - Go API ingress: `protojson.UnmarshalOptions{DiscardUnknown: false}` by
    default for client requests, with a central helper.
  - Go API egress: `protojson.MarshalOptions{UseProtoNames: true}`.
  - TypeScript UI read: `fromJson(schema, json, { ignoreUnknownFields: true })`
    for responses.
  - TypeScript UI write: `toJsonString(schema, msg, { useProtoFieldName: true })`
    for request bodies.
  - CLI: existing `protojson` helpers remain the standard.
- Multipart/file uploads are explicit exceptions: structured metadata remains
  protojson, file bytes use multipart parts or a two-step upload flow, and
  responses/errors remain protojson.

### Validation policy

- Server-side validation is authoritative.
- Contract-level validation should live in proto/protovalidate where available.
- UI form validation is for user experience only and must not become a second
  hand-maintained contract mirror.
- Zod may be used for UI-only form state or non-proto external payloads. Do not
  instruct agents to mirror every proto message manually in Zod.

### Future enhancement note

Proto `service` definitions/RPC-style operation contracts are a future platform
capability to evaluate after this boundary pattern is stable. Do not implement
RPC generation in this plan.

## 8. Implementation Strategy

### Phase 1: Update skill guidance first

1. Update `interoperability-steer`:
   - Replace stale TypeScript casing guidance with Buf v2 terms:
     - read: `fromJson(schema, json, { ignoreUnknownFields: true })`
     - write: `toJsonString(schema, msg, { useProtoFieldName: true })`
   - State that `fromJson` accepts both proto field names and JSON/lowerCamel
     names, while writes must be explicit to keep Vrooli APIs snake_case.
   - Set `scenarios/{{TARGET}}/ui/src/api/` as the canonical UI↔API boundary
     location.
   - Add the multipart/file-upload exception pattern.
   - Add a short future-RPC note without making it current guidance.

2. Update `react-coherence`:
   - In the service/API row, state that scenario templates use `src/api/` for
     API clients and reserve `src/lib/` for pure utilities.
   - Keep the broader `shared/services` language only for large apps that already
     have a `shared/` hierarchy, but do not present it as the template default.
   - Add a simple ownership table for `api/`, `features/`, `components/`,
     `hooks/`, and `lib/`.

3. Update `react-stability`:
   - Replace the default "generated TS types -> Zod schemas" chain with:
     `proto schema -> generated descriptors/types -> proto boundary parse -> UI`.
   - Clarify Zod's role: UI-only form validation or non-proto inputs, not a
     mandatory mirror of proto messages.
   - Keep the strong TypeScript/ESLint guardrails unchanged.

4. Run the Prompt Manager skill validation/sync workflow used in this repo.
   Discover exact command from `prompt-manager skill --help` if needed.

### Phase 2: Update shared proto documentation

1. Update `packages/proto/README.md` TypeScript examples:
   - Remove `jsonOptions: { useProtoNames: true }`.
   - Show current Buf v2 read/write examples.
   - Explain that Vrooli writes proto field names with
     `{ useProtoFieldName: true }`.

2. Add a short "JSON bodies vs file uploads" note:
   - structured metadata: protojson
   - binary streams: multipart or upload URL
   - responses/errors: protojson

### Phase 3: Move the template UI boundary to `src/api/`

1. Move the current `ui/src/lib/api.ts` substrate to `ui/src/api/client.ts`.
2. Split health into `ui/src/api/health.ts`.
3. Move notes client from `ui/src/lib/notes.ts` to `ui/src/api/notes.ts`.
4. Update imports in:
   - `ui/src/features/health/HealthCard.tsx`
   - `ui/src/features/notes/NotesCard.tsx`
   - UI tests and mocks
5. Keep `ui/src/lib/utils.ts` as the pure utility example.
6. Do not leave re-export shims in `ui/src/lib/`.

### Phase 4: Harden UI proto client behavior

1. In `src/api/client.ts`, define central options:

   ```ts
   const PROTO_READ_OPTIONS = { ignoreUnknownFields: true } as const;
   const PROTO_WRITE_OPTIONS = { useProtoFieldName: true } as const;
   ```

2. Ensure `protoFetch` uses `PROTO_WRITE_OPTIONS` for request serialization.
3. Ensure `decodeApiError` and response decoding use the read options.
4. Add or update tests that prove:
   - snake_case response fields decode into lowerCamel generated TS properties.
   - lowerCamel response fields also decode, if supported by Buf.
   - request bodies emit snake_case/proto field names when the request contains
     an underscored proto field.
   - non-2xx envelopes produce `ApiError` with `code`, `message`, and `status`.

If the notes request has no underscored field, add the casing test at the
`protoFetch` level using an existing generated message descriptor with an
underscored field, not by adding fake production schema.

### Phase 5: Move API ingress to protojson for JSON requests

1. Replace or supplement `api/internal/httpx/decode.go` with a proto-specific
   helper, for example:

   ```go
   func DecodeProtoJSON[T proto.Message](r *http.Request) (T, error)
   ```

2. Use strict unknown-field handling for client requests by default. Protojson's
   default is strict; keep that behavior unless a handler explicitly needs
   leniency.
3. Migrate notes create handler to decode `CreateNoteRequest` through this helper.
4. Keep error translation through the existing `ErrorEnvelope` writer.
5. Add API handler tests proving:
   - snake_case request fields decode.
   - lowerCamel request fields decode when protojson supports them.
   - unknown fields still return `invalid_request`.
   - malformed JSON still returns `invalid_request`.

### Phase 6: Update template docs

Update every template doc that names `lib/api` or `lib/notes`:

- `templates/scenarios/react-vite/docs/concepts/ARCHITECTURE.md`
- `templates/scenarios/react-vite/docs/internal/SEAMS.md`
- `templates/scenarios/react-vite/docs/internal/REPLACING-NOTES.md`
- `templates/scenarios/react-vite/docs/internal/TESTING.md`
- `templates/scenarios/react-vite/proto/v1/*/*.proto` comments if they mention
  old UI paths or stale `useProtoNames` terms.

Documentation should state the final architecture directly:

- `src/api/client.ts` is the proto-typed network substrate.
- `src/api/<domain>.ts` is the per-domain client.
- `src/features/<domain>/` owns UI assembly and local mocks.
- `src/lib/` is pure utilities only.

### Phase 7: Add guardrails against drift

Evaluate lightweight lint or tests that enforce:

- no raw `fetch(` outside `ui/src/api/client.ts` and test utilities.
- no imports from `@vrooli/proto-types` directly inside feature components unless
  the component is intentionally rendering a generated enum/type.
- no production network client files under `ui/src/lib/`.
- no `encoding/json` request decode for proto API messages in template handlers.

Prefer existing ESLint/custom rule infrastructure in the template. Do not add a
new dependency.

## 9. Contract Decisions

- **Casing:** Vrooli JSON APIs emit proto field names/snake_case. UI generated
  TS objects expose lowerCamel properties after `fromJson`.
- **Unknown fields:** UI responses tolerate additive unknown fields. API requests
  reject unknown fields by default.
- **Errors:** Every non-2xx JSON API response uses `ErrorEnvelope`; UI surfaces
  it as `ApiError`; CLI wraps it through `cliapp.WrapAPIError`.
- **Files:** file bytes are not protojson. Metadata and responses are protojson.
- **Forms:** forms may have local draft types. Request payload construction must
  happen in `src/api/<domain>.ts` using generated proto descriptors.

## 10. Testing Plan

Run focused tests first:

```bash
cd templates/scenarios/react-vite/ui && pnpm test -- src/api src/features/notes src/features/health
cd templates/scenarios/react-vite/api && go test ./handlers/notes ./internal/httpx
cd packages/proto && make check
```

Then validate the template through the scenario workflow:

```bash
vrooli scenario template validate react-vite
vrooli scenario generate react-vite --help
```

If a scratch scenario is generated during execution, manage it through scenario
Makefiles or `vrooli scenario` lifecycle commands only. Do not run scenario
binaries directly.

## 11. Rollout / Validation Checklist

- [ ] Skills state the intended architecture without legacy fallback language.
- [ ] `packages/proto/README.md` uses current Buf v2 TypeScript options.
- [ ] Template UI imports API clients from `src/api/`, not `src/lib/`.
- [ ] Raw production `fetch` appears only in `src/api/client.ts`.
- [ ] UI request serialization uses `{ useProtoFieldName: true }`.
- [ ] API JSON request ingress for proto messages uses `protojson`, not
      `encoding/json`.
- [ ] Notes remains a complete CRUD reference across API, UI, CLI, tests, and docs.
- [ ] Template docs and proto comments mention current paths and current options.
- [ ] Tests cover casing, envelopes, malformed JSON, and unknown fields.

## 12. Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Buf v2 API differs across installed versions | Verify against the template's pinned package version and update docs to match that version. |
| Moving `lib/notes` breaks existing test mocks | Update mocks to target `src/api/notes`; do not leave `lib` re-export shims. |
| Protojson ingress changes error text | Tests should assert stable envelope code and meaningful field mention, not exact raw parser wording. |
| Zod guidance removal is overcorrected | Keep Zod explicitly allowed for UI-only form validation and non-proto external payloads. |
| Lint guardrails become noisy | Start with narrow, template-owned checks and clear allowlists for tests/mocks. |

## 13. Non-goals / Prohibited Patterns

- Do not implement proto `service`/RPC generation in this plan.
- Do not add compatibility re-exports from `src/lib/api.ts` or `src/lib/notes.ts`.
- Do not introduce a second hand-written validation contract for proto messages.
- Do not add dependencies without explicit approval.
- Do not move unrelated UI folders just to match an abstract directory tree.
- Do not weaken TypeScript or ESLint strictness.

## 14. Definition Of Done

- The three Prompt Manager skills and the React Vite template describe the same
  UI/API boundary architecture.
- A new domain author can add proto messages, API handlers, CLI commands, and UI
  clients by following one documented path with no casing guesswork.
- JSON request/response/error payloads use proto tooling at every API/CLI/UI
  boundary.
- File upload exceptions are documented as structured metadata plus binary body,
  not as ad hoc JSON drift.
- Tests and template validation pass.
- Final implementation summary includes any remaining deliberate gaps, especially
  whether proto `service`/RPC generation should be scheduled as a later platform
  capability.

