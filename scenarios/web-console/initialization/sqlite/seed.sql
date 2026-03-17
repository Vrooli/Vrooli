-- Default AI provider configuration (idempotent)
INSERT INTO ai_provider_configs (name, enabled, priority, timeout_sec, max_retries)
VALUES
    ('ollama', 1, 1, 30, 0),
    ('openrouter', 1, 2, 30, 0)
ON CONFLICT (name) DO NOTHING;

-- Default shortcut profile (idempotent)
INSERT INTO shortcut_profiles (id, scope, name, shortcuts)
VALUES (
    'default',
    'service',
    'Default',
    '[{"label":"Claude Code","command":"claude --dangerously-skip-permissions","description":"AI coding assistant with full permissions"},{"label":"Codex","command":"codex --yolo","description":"OpenAI Codex CLI in auto-approve mode"}]'
)
ON CONFLICT (id) DO NOTHING;
