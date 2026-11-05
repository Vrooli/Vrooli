-- Default AI provider configuration (idempotent)
INSERT INTO ai_provider_configs (name, enabled, priority, timeout_sec, max_retries)
VALUES
    ('ollama', 1, 1, 30, 0),
    ('openrouter', 1, 2, 30, 0)
ON CONFLICT (name) DO NOTHING;

-- Default shortcut profile (idempotent seed for fresh DBs).
-- ON CONFLICT DO NOTHING means this never updates an existing row; bumping the
-- shortcut list for already-seeded DBs is handled by reconcileDefaultShortcutProfile
-- (api/shortcut_profiles_sql.go), which upgrades the row only when it is still the
-- unmodified seed. Keep this list in sync with defaultShortcuts (api/shortcut_profiles.go).
INSERT INTO shortcut_profiles (id, scope, name, shortcuts)
VALUES (
    'default',
    'service',
    'Default',
    '[{"label":"Claude Code","command":"claude --dangerously-skip-permissions","description":"AI coding assistant with full permissions"},{"label":"Codex","command":"codex --yolo","description":"OpenAI Codex CLI in auto-approve mode"},{"label":"OpenCode","command":"opencode","description":"OpenCode TUI — conversation captured via its local server API"},{"label":"Grok","command":"grok","description":"xAI Grok CLI — conversation captured from its session transcript"}]'
)
ON CONFLICT (id) DO NOTHING;
