# API Endpoints

Onboarding business operations use generated unary Connect-RPC procedures.
The machine-readable source is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json);
the API registry and endpoint generator keep it aligned with the running
handlers. Requests that support another node carry a `target` field and use the
same generated procedure address for local and remote calls.

## Operational REST exceptions

These are the only HTTP routes. They are dependency-free `ops_probe` health
checks for lifecycle managers, load balancers, and curl. See
[`REST_EXCEPTIONS.md`](../internal/REST_EXCEPTIONS.md) for the ownership rule.

| Method | Path | Use |
|---|---|---|
| GET | `/health` | Infrastructure health probe |
| GET | `/api/v1/health` | Client-facing health alias |

## Connect services

Each row is invoked with `POST /<fully-qualified-service>/<method>` by a
generated client. The exact fully-qualified paths, request fields, governance,
and CLI mappings are in `.vrooli/endpoints.json`.

| Service | Methods | Purpose |
|---|---|---|
| `OperatorInputsService` | `ListOperatorInputs`, `ResolveOperatorInputs` | Read pending questions and submit typed answers. |
| `ReadinessService` | `GetReadiness`, `AcknowledgeDegradedReadiness` | Compose readiness and acknowledge a digest-bound degraded set. |
| `ApplyService` | `StartApply`, `GetApplyRun`, `GetApplyPlan` | Start the server-owned apply and inspect its plan or run. |
| `SessionService` | `GetSession`, `AdvanceSessionStep`, `GetStepModel` | Share the stable wizard pointer and step model. |
| `SelectionService` | `ListScenarios`, `GetCoreSet`, `GetRecommendation`, `AcceptRecommendation`, `GetClosure`, `GetUnion`, `CreateHandoff` | Resolve scenario choices, dependencies, core protection, and the Bridge handoff. |
| `ProfileService` | `ListProfiles`, `EvaluateProfile` | List data-owned purpose profiles and evaluate bounded answers into visible questions and recommendations. |
| `CapabilitiesService` | `ListCapabilities`, `GetCapabilityStatus`, `PreviewCapability`, `ApplyCapability` | Inspect and apply typed operator capabilities. |
| `CredentialsService` | `ListCredentials`, `ProvisionCredential`, `DiagnoseCredentials` | Read metadata, relay a write-only value, and diagnose the authority. |
| `HostService` | `ListHostRequirements`, `GetHostFacts`, `ListTargets`, `PatchHostSafeguardConfig`, `SetNotificationRecipient` | Read host requirements and delegate the two governed writes. |
| `OperatorStateService` | `GetOperatorState`, `PatchOperatorState` | Read or field-mask patch the single operator-state document. |
| `ResourcesService` | `ListResources`, `GetResource`, `GetResourceHealth`, `ListDerivedResources` | Inspect runtime resources, health, and selection-derived resources. |
| `GlossaryService` | `SearchGlossary` | Search the static onboarding vocabulary. |

## Error and security contract

Connect errors preserve `invalid_argument`, `unauthenticated`,
`permission_denied`, `unavailable`, `deadline_exceeded`, `aborted`, and
`internal` distinctions. Mutation authorization consumes the verified principal
installed by `api-core/authn` and the shared onboarding capability; credential
values are write-only. See [`ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md)
and [`SECURITY.md`](../internal/SECURITY.md).
