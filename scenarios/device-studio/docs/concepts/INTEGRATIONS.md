# Integrations — Device Studio


Contract updated during delivery. Project/photographic render integration is implemented; physical-shot composition and optional creation/account/publication integrations remain in progress. Qualification below is scoped to actual evidence, not a released desktop claim.

Separate source-verified shared code, proposed adoption and unresolved owner contracts. No listed optional service is a hidden local-start prerequisite.

## Launch-video consumer operation

`device-studio.launch-video-shot` is the governed consumer workflow for a saved
shot. It uses Program Runtime's typed Device Studio bindings to save an explicit
shot revision, reopen the returned immutable revision, submit one render with a
caller-stable idempotency key, and read that job's current owner status. The
program returns the exact project/shot/revision/job references, source and output
provenance available from the owner, and the supported `device-studio renders
export` command after success. That CLI command remains the byte-download owner:
it checks the succeeded job, MIME, declared size and SHA-256, decodes PNG or
checks the MP4 container, and publishes without replacing a destination. The
program does not call a shell, wait in a private polling loop, or maintain a
second job ledger. A queued/running job is resumed by observing the same job ID;
cancellation uses the owner's expected version.

The operation consumes a project and media that already exist in Device Studio.
It does not package or import a project. Portable project bundles and six
independently reopenable saved demonstrations remain separate delivery work.
The launch-video skill owns the assembled film, audio mix, CTA and whole-film
review; this operation supplies only the immutable device shot and its media
receipt.

## Dependency Inventory

| Dependency | Role | Required for | Current qualification |
|---|---|---|---|
| api-core/platform-go | Storage, identity, discovery and owned process lifecycle | Product core | Shared code exists; adoption and per-OS tests owed |
| React Component Library | Shell/tokens/control primitives | UI build | Template adoption; full account surface is a placeholder |
| FFmpeg/FFprobe | Decode, inspect, encode | Local video | Managed pair qualified for bounded current-host outputs; desktop redistribution remains separate |
| Blender renderer pack | Physical 3D output | Local 3D | Managed5.2.2LTS with qualified current-host CPU/CUDA/OptiX studies; public job/desktop integration underway |
| BAS | Authentic capture | Optional capture action | Import existing media remains independent |
| Image Tools | Photographic candidate generation and preparation | Direct API adapter exists but is absent from `.vrooli/service.json`; unavailable behavior is feature-scoped and must not be inferred from the manifest. |
| AI Gateway | Geometry generation and selected model-backed routes | Declared optional; a selected route fails when the gateway is unavailable. |
| Workspace Sandbox | Contained geometry compilation | Declared optional; geometry compile fails when its protected sandbox is unavailable. |
| Audio Tools | Capability discovery | Declared optional; no hidden startup prerequisite. |
| Agent Manager | Bounded creation agent | Explicit agent action | Actual hard-cap/isolation capability must be checked |
| Asset Studio | Generic shared publication | Shared library release | Byte/media admission/provenance adapters owed |
| Backdrop Studio | Suitable ambient assets | Optional scenes | Does not define room/support geometry |
| LPBS/credential authority | Linked account, credit accounting and secrets | Optional cloud/BYOK | Scoped lease-to-spend adapter and callback repairs owed |
| scenario-to-desktop/deployment owners | Private runtime, download/update/signing | Desktop delivery | Actual OS release evidence owed |

## Vrooli Resources

No external database/cache/GPU service is a mandatory scaffold dependency. SQLite is in-process. Local AI resources such as Ollama are optional choices only when their capability is available and appropriate. Dependencies and artifacts are declared through the lifecycle/SDA owners; this document never authorizes raw package installation or a private host-repair script.

## Scenario dependency declarations and source boundary

`.vrooli/service.json` currently declares these three optional scenario peers with
`startup_policy: try_start`:

| Scenario | Manifest status | Source-backed use |
|---|---|---|
| `audio-tools` | optional, `try_start` | Capability discovery is registered; feature-level adoption remains scoped to the capability contract. |
| `ai-gateway` | optional, `try_start` | Model-backed geometry and library-intake routes resolve the gateway when selected. |
| `workspace-sandbox` | optional, `try_start` | Geometry compilation creates a protected workspace and refuses if the sandbox cannot enforce the required containment. |

The photographic adapter also resolves Image Tools directly (`api/internal/photographic/client.go`),
but Image Tools is not declared in the current service manifest. Its absence makes that
photographic operation unavailable; it does not affect project editing or other local paths.
This direct integration needs service-manifest owner reconciliation. The declared optional
peers are not hidden startup prerequisites, and a declaration alone does not prove a route is
qualified.

Asset Studio is implemented but lacks the required artifact-byte and generalized model/movie admission route in inspected code. Its identity-free release path does not prove Device Studio craft review. Private projects remain usable; shared publication waits on the correct owner. Account linking yields a bound signed lease, not automatically a consumer access token accepted by funded inference. Never bridge it by copying website tokens or charging a shared host account implicitly.

## Third-Party Services

Free model providers supply source bytes only through permitted download routes. Track license, author, exact version/digest and operation-specific permissions. Generation providers are explicit local/BYOK/Vrooli choices. Media upload requires project consent independently of account linkage. Generic image controls, actual provider model identity and cost receipts must be qualified end-to-end. Do not assume built-in chat image generation is callable by the scenario. No paid remote render service is selected by this design.

## Failure Modes

| Failure | Product response | Repair owner |
|---|---|---|
| Capture service absent | Import recording remains available | BAS integration |
| Renderer/backend unavailable | Explain capability; offer explicit compatible local fallback | Tool/control-plane owner |
| Missing Asset Studio adapter | Keep private result; shared release pending | Asset Studio |
| Expired/revoked account or insufficient credit | Refuse new funded dispatch; preserve local projects | LPBS/account adapter |
| Unknown provider cost | Keep pending/unknown; reject unbounded new paid dispatch | Provider/gateway/commerce |
| Agent runner lacks hard aggregate limit | Refuse hard-cap workflow; explain supported option | Agent Manager |
| Unavailable external services offline | Keep local editing/replay/render with installed assets | Respective integration |

A failed local call never silently selects a paid cloud source.

## Cross-References

[ARCHITECTURE.md](ARCHITECTURE.md), [monetization](../business/MONETIZATION.md), [deployment](../operations/DEPLOYMENT.md), [research](../research/README.md), [problems](../internal/PROBLEMS.md).

[Configuration and policy boundaries](../reference/configuration.md).
