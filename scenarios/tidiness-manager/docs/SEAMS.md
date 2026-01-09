# Tidiness Manager Seams

- **Entry/presentation**: HTTP handlers (`handlers.go`, `handlers_campaigns.go`) decode/validate inputs, normalize scenario names through `ScanCoordinator`, and delegate orchestration. Refactor recommendations now live in the handlers file as a thin wrapper that calls coordination instead of running scans directly.
- **Coordination/orchestration**: `ScanCoordinator` owns scan flow ordering, scenario scoping, and persistence hand-offs. It now also seeds file metrics on demand for refactor recommendations, using `ScenarioLocator` for paths and the existing light scan flow rather than handler-level env lookups. Auto-campaign orchestration stays in `AutoCampaignOrchestrator` with DB-backed state transitions, concurrency checks, and centralized campaign input normalization (defaults and safety caps).
- **Domain/integration**: `RefactorRecommender` merges visited-tracker data with file metrics and ranking rules; `LightScanner` and `SmartScanner` handle IO-heavy analysis; `TidinessStore` persists metrics/issues/history to Postgres; `CampaignManager` integrates with visited-tracker APIs.
- **Cross-cutting concerns**: Logging flows through `Server.log`/`logJSON`; scan workflows guard against oversized inputs and path traversal; scenario discovery and caching live in `ScenarioLocator` instead of per-handler env lookups.

## Recent Boundary Tightening
- Refactor recommendation auto-scan logic moved out of HTTP handler/domain into `ScanCoordinator.EnsureFileMetrics`, centralizing scanning, persistence, and scenario-path validation.
- Refactor recommendation inputs now pass through `ScanCoordinator.NormalizeScenarioName`, keeping traversal checks in one place before DB access.
- Auto-campaign creation rules (defaults and max limits) now live inside `AutoCampaignOrchestrator.normalizeCampaignConfig`, so both HTTP handlers and any future callers reuse the same validation instead of duplicating it per entrypoint.

## Guidance for Future Changes
- When adding new scan or recommendation entrypoints, route scenario identity and auto-scan behavior through `ScanCoordinator` rather than reintroducing env-based path building.
- New auto-campaign entrypoints should call `AutoCampaignOrchestrator` directly; avoid reintroducing handler-level defaults or caps and let the orchestrator guard inputs.
- Prefer extending `TidinessStore`/`CampaignManager` for persistence/integration behaviors instead of embedding SQL or HTTP calls inside handlers or scanners.
