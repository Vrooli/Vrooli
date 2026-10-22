# E1 authenticated actor/workspace qualification — 2026-10-20

Directive: `ORCH-E1-ACTOR-WORKSPACE-QUALIFICATION-20261020`  
Parent: `5d07e651-eaa6-48ec-89b4-5dc359059193`  
Task/run: `20794cb4-ba98-4e0b-adc3-fb018a746390` / `9d57688b-1f9a-400f-a5ab-f1af59a827a1`  
Scope: read-only source qualification; no product source/test, auth configuration, database, runtime, test lifecycle, provider/user I/O, or credential inspection.

## Finding

Authentication support exists at the shared API boundary, but the approved authenticated actor/workspace parity gate is **not proven by current evidence**. Planner handlers do not currently consume the verified principal to select/authorize a workspace, and storage is singleton. However, the existing Planner domain/security contract explicitly assigns subject-to-workspace binding and per-record authorization to Planner's workspace/domain API layer. The shared Principal contract is sufficient; the missing behavior is wholly actionable in approved Planner paths. UI/CLI/Connect transport and the retained same-ID receipt do not establish runtime actor/workspace parity.

## Source evidence

### API authentication and principal

- `scenarios/personal-planner/api/main.go:113-120` loads `authn.FromEnvironment(os.Getenv)` and passes a non-nil shared config only when enabled; `:127-130` supplies it to `apiserver.Run`.
- `packages/api-core/authn/authn.go:115-136` defines environment-selected provider adapters and Scenario Authenticator issuer/audience/JWKS/cookie configuration. `:228-245` authenticates requests; absent providers/proof fail as missing/unavailable, and bearer proof is rejected unless a bearer-capable provider is configured.
- `packages/api-core/authn/jwt.go:103-121` reads a bearer Authorization credential or configured cookie. `:171-218` verifies issuer, audience, expiry and subject, then creates the verified principal; subject resolves from user_id then sub, actor kind from verified claims/config.
- `packages/api-core/server/server.go:142-145,197-205` describes authentication as passive context capture (read-only handlers remain available; writers must enforce humans) and wraps configured handlers in authn middleware.
- `packages/api-core/authn/authn.go:387-428` calls Authenticate and attaches successful principal/status to request context. `packages/api-core/identity/identity.go:93-114,222-229` defines canonical Principal (actor kind, subject, owner, source, verification, etc.) and exposes only verified context principals. This contract contains no Planner workspace ID/binding.

### Planner handlers and storage

- `scenarios/personal-planner/api/handlers/calendar/module.go:18-25` constructs calendar repository/service/handlers without an identity/workspace resolver.
- `scenarios/personal-planner/api/handlers/calendar/events_http.go:32-55,61-83` passes `req.Context()` into list/get/create/update domain calls, but does not extract principal or workspace identity.
- `scenarios/personal-planner/api/handlers/calendar/connect_handler.go:164-198` Connect List/Get/Create/Update methods pass request context into domain calls, but perform no principal extraction or workspace selection.
- `scenarios/personal-planner/api/internal/calendar/events.go:12-31,33-49` models `Subject` as an ordinary event field, distinct from authenticated Principal. `:63-75` delegates list/get without principal selection.
- `scenarios/personal-planner/api/internal/calendar/events_sqlite.go:18-43,120-133` queries events without workspace/actor predicates and inserts the caller-provided event `Subject`; it does not derive actor/workspace from context.
- `scenarios/personal-planner/api/handlers/workspace/connect_handler.go:28-45,48-65` calls singleton profile/availability service methods with context but no identity binding.
- `scenarios/personal-planner/api/internal/workspace/sqlite.go:28-38,51-64` selects and updates using fixed `profileID`; profile is created as one default record. This is the implementation gap against the already approved workspace contract, not evidence that a shared contract is missing.
- `scenarios/personal-planner/docs/concepts/DOMAINS.md:44-47,89-95` assigns the workspace domain responsibility to bind authenticated subject to one private workspace and enforce INV-01. `scenarios/personal-planner/docs/internal/SECURITY.md:48-67,125-140` specifies scenario-authenticator identity, subject/workspace/capability resolution from authenticated context, no trusted client tenant/owner, per-record authorization, and explicitly records missing enforcement as a Planner security gap. The workspace domain owns applying this mapping; scenario-authenticator owns identity.

### UI, CLI, Connect

- `scenarios/personal-planner/ui/src/api/client.ts:12-16` resolves the API base and creates the shared scenario Connect transport; its REST upload path at `:54-59` adds no principal itself. The implementation in `packages/api-base/src/client/connect.ts:11-18` passes base URL/fetch into Connect-Web without adding actor/workspace headers or an explicit credentials policy.
- `scenarios/personal-planner/ui/src/api/calendar.ts:29-35,38-60` uses browser `fetch` for REST with content type/body but no explicit Authorization, actor, or workspace value. This leaves any cookie/session forwarding behavior to shared browser transport/deployment; it does not demonstrate which principal arrives.
- `scenarios/personal-planner/ui/src/api/workspace.ts:1-37` uses generated Connect client requests with profile/availability payloads, no actor/workspace fields.
- `packages/cli-core/cliapp/connect.go:74-85` applies configured CLI HTTP headers and forwards `VROOLI_API_TOKEN` as `X-Agent-Identity-Token` for Connect if absent. This is an agent identity carrier, not proof that a human actor/workspace binding is forwarded. Planner calendar/workspace handlers instantiate generated clients via `scenarios/personal-planner/cli/domains/calendar/handlers.go:25-31` and `cli/domains/workspace/handlers.go:22-27`.
- The shared CLI REST client path is `packages/cli-core/cliutil/apiclient.go:11-23,135-149,162-171`, backed by `packages/cli-core/cliutil/httpclient.go:428-453,475-497`; it applies its configured bearer token and trusted agent identity headers. Current Planner CLI Go sources contain no `APIClient` construction/use, so there is no Planner REST client actor/workspace propagation path to claim; Planner's supported calendar/workspace commands use Connect.
- Planner generated Connect messages expose calendar/profile operation data; `scenarios/personal-planner/api/handlers/calendar/connect_events_test.go:138-150` demonstrates a UI-style REST request with synthetic bearer; `:374-395` demonstrates synthetic provider verification and context subject observation. These are isolated test fixtures, not proof that production handler/storage uses that principal.
- The live same-ID receipt `scenarios/personal-planner/docs/internal/goal/receipts/e1-same-id-live-journey-20261019.md` reports UI-created event IDs/revision 2 read by CLI/Connect, but explicitly says the BAS receipt emitted no authenticated actor ID or workspace ID. It cautions against inferring parity from IDs; this matches the current gap.

## Owner and supported next route

There is **no missing shared-owner contract** to change for this gate. `api-core/identity`/`api-core/authn` own the provider-neutral principal and verified request context (`docs/concepts/IDENTITY-AND-AUTHENTICATION.md:72-85,314-341`); scenario-authenticator is the identity owner. Personal Planner's `workspace` domain owns subject-to-workspace binding, per-record scope enforcement, and attribution within the authoritative API/service/storage paths (`docs/concepts/DOMAINS.md:44-47,89-95`; `docs/internal/SECURITY.md:54-67`). The supported implementation route is the existing Personal Planner E1 workspace/API scope: workspace handlers and storage, plus cross-surface evidence on UI, supported CLI, and Connect using the same authenticated context. This qualification's own admission is read-only, so it stops at evidence and makes no product changes; an implementation slice must be dispatched under that existing Planner scope. No api-core change or scope widening is indicated. This qualification records the finding only; implementation requires its own admitted Planner slice.

## Scope decision and status

- **Changed:** goal-home receipt only.
- **Verified:** source evidence above, including API auth setup, context principal contract, Planner handler/storage behavior, UI REST/Connect transport shape, CLI Connect forwarding and generic REST client contract (with no Planner REST-client use found), and explicit same-ID receipt limitation.
- **Remaining:** dispatch an implementation slice under the already approved Planner workspace/API scope; implement principal-derived workspace selection and per-record scope checks, then gather explicit actor/workspace evidence across browser UI, supported CLI, and Connect. No shared-owner change is identified.
- **Unverified:** actual live authenticated actor/workspace values across browser, CLI and Connect; provider/session behavior for the UI; any principal-to-workspace mapping policy; production authorization semantics. No live/provider/runtime/test/database check was run.
- **Decision:** the gate cannot be proven from current evidence, but it can be repaired wholly within the already approved Personal Planner workspace/API scope. No shared auth change is required. Do not infer parity from localhost/profile labels/event IDs; require explicit actor and resolved workspace evidence across the supported surfaces after Planner implementation.
