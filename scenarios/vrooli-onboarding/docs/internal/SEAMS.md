# Desktop wrapper seams

The generated Electron wrapper is a presentation host for the bundled
onboarding runtime. These module anchors describe its explicit process-boundary
seams; they do not own onboarding business decisions.

## target-proxy interceptor

Connect requests that declare a `target` field use the same generated procedure
address for local and remote calls. Empty and `local` targets continue through
the handler. A named node is forwarded through `nodereach` with the request
serialized as protobuf and the response decoded from the generated method
descriptor. Host safeguard and notification-recipient mutations fail at this
source boundary with `failed_precondition`; they never reach Bridge for a
remote target. Offline nodes map to `unavailable`, missing grants to
`permission_denied`, and expired deadlines to `deadline_exceeded`.

## Host requirement capability

The setup wizard resolves non-local selections against `ListTargets` before
profile-session or host-requirement reads. The capability policy recognizes
registered `NODE_KIND_AGENT` and `NODE_KIND_SSH` IDs as remote host candidates;
unknown IDs and attached/control-plane/unspecified node kinds stop with an
unsupported state. Registry failure also blocks target-scoped reads and saves
until the selection can be resolved. Host-tool opt-ins and safeguard writes
remain owned by the local control plane. `ui/src/api/hostCapabilities.ts` is the
shared owner policy for client-adapter guards and wizard affordances; a remote
host candidate remains readable and its controls are visibly disabled. Do not
infer remote host mutation support from successful target discovery or
requirement reads. The target-proxy gate
also rejects the two host write procedures before remote dispatch; direct
low-level Bridge dispatch and real node behavior remain unqualified. The wizard
edits boolean, enum, string and numeric manifest fields. Other schema types are
reported as not editable instead of being coerced to text.

When a remote profile read fails, the wizard refreshes target admission. If the
node has disappeared, its profile surface becomes explicitly unsupported and
offers a target refresh action. If the registry still includes it, the wizard
shows a remote read failure and retains the target-keyed draft. Re-admitting the
same node resumes its saved answers; it does not copy answers to another target.

## Target-bound wizard work

Credential form values and save feedback live in
`ui/src/hooks/useCredentialTargetState.ts`, keyed by target. Provision and
reveal operations capture their selected target; reveal values and timers clear
on switch, and late reveal responses are ignored. Apply run IDs retain the
existing target-keyed local-storage format. `StepReadiness` aborts polling on
target switch/unmount, binds review/start/cancel to the captured target, and
uses a per-target idempotency key until a run is accepted. A failed start
retains its key for safe retry. `StepPlan` uses the target in its React Query
key, so recommendations and acceptance feedback cannot appear under another
target. Mocked target-transition tests cover these contracts; no real Bridge or
remote target behavior is implied.

## Credential diagnosis projection

`google.protobuf.Struct credential_diagnosis` crosses the Connect response as
a JSON object at its root. `ui/src/api/readiness.ts` converts known provider
fields into `CredentialDiagnosis`; consumers should not read a protobuf-style
`.fields` property or cast the map to `any`. The adapter test covers the JSON
shape and malformed provider data. Credential evidence also uses the committed
generated protobuf fields directly; do not add compatibility casts for fields
the generated owner already provides. This read path does not grant remote
credential mutation or secret reveal capabilities.

## main-process

Electron process composition and lifecycle wiring.

## splash-window-module

Startup splash window presentation while the bundled runtime becomes ready.

## window-state-module

Persisted Electron window geometry and visibility state.

## telemetry-module

Best-effort desktop telemetry delivery with no credential values.

## runtime-module

Loopback-only runtime control client and health coordination.

## storage-module

Desktop-local non-secret application metadata. Credentials are delegated to
the native credential authority and never stored here.

## ipc-module

Typed Electron main/renderer IPC boundary.

## auth-module

Shared `api-core/authn` verification and onboarding's provider-neutral human
capability gate. Process credentials are not accepted as human identity.

## bundle-module

Verified bundle path and manifest access.

## runtime-control-client

Renderer-independent client for runtime state and lifecycle operations.

## Connect interceptors

| Seam | Production wiring | Test substitution | Contract |
|---|---|---|---|
| Target interceptor | `api/main.go` passes `targetproxy.Interceptor(s.bridge)` to every target-aware module. | `internal/targetproxy/interceptor_test.go` uses a fake node client and signed request frames. | Empty or `local` targets stay local; named targets use the generated procedure address and map node, scope, and deadline failures to Connect codes. |
| Verified mutation auth | `api-core/authn` installs one verified-principal middleware; `internal/authz` applies the shared human + `vrooli-onboarding:write` gate to every declared mutation. Bundled desktop services receive the runtime-owned local-session token path; the trusted renderer bridge and PTY token binding supply the proof. | Auth tests cover missing/invalid local sessions, agent provenance, forwarded spoofing, trusted renderer admission, and remote principal cases. | Loopback is only an admission input; the runtime-owned session proof is required, and the provider never derives a human principal from the API process's OS user. |
| Error mapping | `handlers/<domain>/connect_handler.go` maps domain errors at the transport edge. | Domain handler tests assert the Connect code and metadata-safe message. | Sentinel errors retain actionable distinctions and never include credential values. |

## Ambient clock

Time-dependent onboarding behavior enters through a clock seam at each module
boundary. The API uses the repository `internal/clock.Real` implementation and
retains `operatorStateNow` as its test injection point; the standalone CLI uses
`cli/internal/clock.Real` and passes the resulting function into long-running
progress reporting. Tests can therefore control expiry, heartbeat, and progress
cadences without replacing process-global time. The only direct `time.Now` calls
in these production paths are the two clock implementations themselves.

The API's root resolver also reads process configuration through
`api/internal/envx.Reader`; production uses `envx.OS`, while tests may inject a
fixed reader when they need to isolate path selection from the host environment.

## Ownership boundaries

The domain service under `api/internal/<domain>/` owns read models, validation,
and repository seams without importing Connect or mux. The handler package owns
request decoding, authorization, error mapping, and module registration.
`api/apply_runner.go` is an orchestration seam owned by the API composition
layer; it is not imported by domain packages. `internal/operatorstate` remains
the only write authority for operator-state decisions, and resources, host,
credentials, and readiness only read or delegate through their owning control
plane.

## Architecture alignment notes

| Area | Drift | Decision | Follow-up |
|---|---|---|---|
| CLI construction | Historical domain registration duplicated the manifest. | The embedded manifest and cli-core primitive loader now own command assembly. | Keep new commands in `manifest.json` and add a generated binding or explicit omission. |
| Health | Lifecycle integrations need a stable unauthenticated HTTP probe. | Keep exactly the two `ops_probe` entries listed in `REST_EXCEPTIONS.md`. | Change only with a lifecycle and load-balancer contract update. |
