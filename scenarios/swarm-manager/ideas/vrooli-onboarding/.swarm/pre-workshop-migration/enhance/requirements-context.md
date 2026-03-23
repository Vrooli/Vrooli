# Requirements Context: Vrooli Onboarding

## Validation Approach

### Secret Validation Testing
- **Format validation**: Each supported resource type needs format pattern tests (e.g. OpenRouter keys must match `sk-or-*` pattern)
- **Whitespace handling**: Verify trimming of leading/trailing whitespace, newlines, and invisible Unicode characters from pasted keys
- **API ping validation**: Mock external API endpoints for testing; real pings only in integration/smoke tests
- **Failure UX**: Verify that validation failures produce actionable error messages (not just "invalid key")

### Config Generation Testing
- **Transitive dependency resolution**: Given a set of chosen resources, verify all required dependencies are included in generated service.json
- **Schema compliance**: Generated service.json must pass validation against config/schema.json
- **Idempotency**: Running config generation with the same inputs twice produces the same output
- **Existing config preservation**: If user already has a service.json, generation should merge (not overwrite) existing settings

### Health Check Integration Testing
- **Polling behavior**: Verify health status updates propagate to both CLI and UI within acceptable latency
- **Startup grace period**: Resources need time to start; verify "starting up" state is distinct from "failed"
- **Flaky health**: Verify that transient health check failures don't cause permanent "unhealthy" display

### Progress Persistence Testing
- **Abandon and resume**: Start onboarding, close session, reopen — verify progress is preserved
- **Concurrent access**: If CLI and UI are both open, verify consistent state
- **Reset capability**: User can restart onboarding from scratch

## Technical Constraints

1. **API-first mandate**: All logic must reside in the API. CLI and UI must not contain business logic — only presentation and API calls.
2. **Dual-frontend parity**: Every feature available in the UI must also be available via CLI, and vice versa.
3. **Non-technical user assumption**: All user-facing text must avoid jargon. Error messages must suggest next actions.
4. **V1 resource scope is fixed**: Only coding agents (claude-code, codex, opencode) and AI providers (openrouter, ollama). No scope creep to other resources.
5. **No embedded third-party config**: Autoheal and prompt-manager are deep-link only in v1. Do not build configuration panels for them.
6. **File-based config**: Configuration reads/writes target .vrooli/service.json and .vrooli/secrets.json. No database required for onboarding state (use file-based persistence consistent with Vrooli patterns).

## Dependency Relationships

```
User Choices
  → Config Generation API
    → service.json (with transitive deps resolved)
    → secrets.json (with validated keys)
      → Resource Enablement (via vrooli CLI)
        → Health Check Polling
          → Status Dashboard
```

Each stage depends on the previous. The onboarding flow must handle failures at any stage gracefully and allow retry without re-doing completed steps.
