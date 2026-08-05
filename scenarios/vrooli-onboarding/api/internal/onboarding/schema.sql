
CREATE TABLE IF NOT EXISTS onboarding_progress (
    id          SERIAL PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT 'default',
    current_step    INT NOT NULL DEFAULT 0,
    completed_steps JSONB NOT NULL DEFAULT '[]'::jsonb,
    config_data     JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id)
);

