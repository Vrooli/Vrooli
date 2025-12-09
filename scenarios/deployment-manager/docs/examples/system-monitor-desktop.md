# Example: System Monitor — Desktop Planning

`system-monitor` depends on high-frequency data collection, persistent storage, and optional AI summaries. It is a perfect stress test for Tier 2.

## Dependency Snapshot

| Dependency | Notes |
|------------|-------|
| API (Go) | Lightweight, can be bundled. |
| UI (Vite/React) | Already a production bundle. |
| Postgres | Heavy for desktop; consider SQLite. |
| Redis | Used for caching; could be swapped for in-process cache. |
| Ollama | Optional AI summaries; not realistic on low-power desktops. Consider OpenRouter. |

## Deployment Thoughts

- **Fitness Scores**: Postgres (0.3), Redis (0.5), Ollama (0.0) → Desktop fitness currently <0.5.
- **Swaps Needed**:
  - Postgres → SQLite or LiteFS.
  - Redis → Embedded cache.
  - Ollama → Optional OpenRouter integration with user-provided key.
- **Secrets**:
  - Infrastructure DB passwords removed.
  - Generate local encryption key for SQLite file.
  - Prompt for OpenRouter key if AI is enabled.

## Next Actions

1. Enter deployment metadata for each dependency.
2. Use deployment-manager (once available) to simulate swaps and produce bundle manifest.
3. Update scenario-to-desktop template to run the Go API + background collectors locally.
4. Define first-run wizard (choose sensors, configure AI, set data retention limits).

Documenting this plan now prevents re-learning the same lessons that Picker Wheel taught us.
