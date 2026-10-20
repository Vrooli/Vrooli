# Integrations — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

Separate source-verified shared code, proposed adoption and unresolved owner contracts. No listed optional service is a hidden local-start prerequisite.

## Dependency Inventory

| Dependency | Role | Required for | Current qualification |
|---|---|---|---|
| api-core/platform-go | Storage, identity, discovery and owned process lifecycle | Product core | Shared code exists; adoption and per-OS tests owed |
| React Component Library | Shell/tokens/control primitives | UI build | Template adoption; full account surface is a placeholder |
| FFmpeg/FFprobe | Decode, inspect, encode | Local video | Host tools observed; distributable pair unqualified |
| Blender renderer pack | Physical 3D output | Local 3D | Host 4.0.2 lacks CUDA payload; governed pack owed |
| BAS | Authentic capture | Optional capture action | Import existing media remains independent |
| Image Tools/AI Gateway | Generic raster/provider operations | Chosen inference route | Funded media not certified by text path |
| Agent Manager | Bounded creation agent | Explicit agent action | Actual hard-cap/isolation capability must be checked |
| Asset Studio | Generic shared publication | Shared library release | Byte/media admission/provenance adapters owed |
| Backdrop Studio | Suitable ambient assets | Optional scenes | Does not define room/support geometry |
| LPBS/credential authority | Linked account, credit accounting and secrets | Optional cloud/BYOK | Scoped lease-to-spend adapter and callback repairs owed |
| scenario-to-desktop/deployment owners | Private runtime, download/update/signing | Desktop delivery | Actual OS release evidence owed |

## Vrooli Resources

No external database/cache/GPU service is a mandatory scaffold dependency. SQLite is in-process. Local AI resources such as Ollama are optional choices only when their capability is available and appropriate. Dependencies and artifacts are declared through the lifecycle/SDA owners; this document never authorizes raw package installation or a private host-repair script.

## Scenario Dependencies

Declare actual callable integrations as optional with concrete degraded behavior when wired. At the foundation stage none is implemented merely because it appears here; `.vrooli/service.json` describes the real scaffold. Keep an explicit planned integration table rather than enabling a fleet at startup. Production dependency declarations must be updated with the adopting slice.

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
