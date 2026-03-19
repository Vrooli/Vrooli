-- Insert default onboarding progress row
INSERT INTO onboarding_progress (user_id, current_step, completed_steps, config_data)
VALUES ('default', 0, '[]'::jsonb, '{}'::jsonb)
ON CONFLICT (user_id) DO NOTHING;
