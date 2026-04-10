# Configuration Reference

## Runtime Files
- `config/settings.json` for persisted UI/API settings.
- `profiles/metadata.json` for auto-steer profile index.

## Important Settings Domains
- Processor behavior and concurrency.
- Agent execution limits and defaults.
- Auto steer profile assignment and iteration limits.

## Environment
- `CORS_ALLOWED_ORIGINS` for API CORS policy.

[CODE: api/pkg/settings/settings.go]
[CODE: api/pkg/settings/provider.go]
[CODE: ui/src/hooks/useSettings.ts]
