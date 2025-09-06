-- Seed data for Feature Request Voting System
-- This creates example scenarios and feature requests for testing

-- Insert example scenarios
INSERT INTO scenarios (id, name, display_name, description, auth_config) VALUES
    ('11111111-1111-1111-1111-111111111111', 'study-buddy', 'Study Buddy', 'AI-powered study assistant with flashcards and quizzes', '{"mode": "authenticated"}'),
    ('22222222-2222-2222-2222-222222222222', 'retro-game-launcher', 'Retro Game Launcher', '80s arcade-style game launcher', '{"mode": "public"}'),
    ('33333333-3333-3333-3333-333333333333', 'system-monitor', 'System Monitor', 'Real-time system monitoring dashboard', '{"mode": "authenticated"}')
ON CONFLICT (name) DO NOTHING;

-- Insert example users
INSERT INTO users (id, external_id, email, username, display_name) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'auth_user_1', 'alice@example.com', 'alice', 'Alice Developer'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'auth_user_2', 'bob@example.com', 'bob', 'Bob Builder'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', 'auth_user_3', 'charlie@example.com', 'charlie', 'Charlie Coder')
ON CONFLICT (external_id) DO NOTHING;

-- Insert example feature requests for Study Buddy
INSERT INTO feature_requests (scenario_id, title, description, status, priority, creator_id, position) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Add spaced repetition algorithm', 'Implement Anki-style spaced repetition for better memory retention', 'proposed', 'high', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 0),
    ('11111111-1111-1111-1111-111111111111', 'Dark mode support', 'Add a toggle for dark mode to reduce eye strain during night study sessions', 'in_development', 'medium', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 0),
    ('11111111-1111-1111-1111-111111111111', 'Export to PDF', 'Allow exporting study notes and flashcards to PDF format', 'under_review', 'medium', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 0),
    ('11111111-1111-1111-1111-111111111111', 'Collaborative study rooms', 'Create virtual study rooms where multiple users can study together', 'proposed', 'high', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 1),
    ('11111111-1111-1111-1111-111111111111', 'Voice-to-text notes', 'Add voice recording with automatic transcription for quick note-taking', 'shipped', 'high', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 0);

-- Insert example feature requests for Retro Game Launcher
INSERT INTO feature_requests (scenario_id, title, description, status, priority, creator_id, position) VALUES
    ('22222222-2222-2222-2222-222222222222', 'Add CRT filter effect', 'Nostalgic CRT monitor effect with scanlines and curve', 'proposed', 'low', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 0),
    ('22222222-2222-2222-2222-222222222222', 'Gamepad support', 'Support for USB and Bluetooth gamepads', 'in_development', 'high', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 0),
    ('22222222-2222-2222-2222-222222222222', 'High score leaderboards', 'Global and friends-only leaderboards for each game', 'proposed', 'medium', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 1);

-- Insert example feature requests for System Monitor
INSERT INTO feature_requests (scenario_id, title, description, status, priority, creator_id, position) VALUES
    ('33333333-3333-3333-3333-333333333333', 'Custom alert thresholds', 'Allow users to set custom thresholds for CPU, memory, and disk alerts', 'under_review', 'high', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 0),
    ('33333333-3333-3333-3333-333333333333', 'Historical data export', 'Export system metrics history to CSV or JSON', 'proposed', 'medium', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 0),
    ('33333333-3333-3333-3333-333333333333', 'Docker container monitoring', 'Monitor individual Docker containers resource usage', 'wont_fix', 'low', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 0);

-- Add some example votes
INSERT INTO votes (feature_request_id, user_id, value) 
SELECT 
    fr.id,
    u.id,
    CASE 
        WHEN random() > 0.3 THEN 1 
        ELSE -1 
    END
FROM feature_requests fr
CROSS JOIN users u
WHERE random() < 0.6 -- 60% chance of voting
ON CONFLICT (feature_request_id, user_id) DO NOTHING;

-- Add example comments
INSERT INTO comments (feature_request_id, user_id, content) 
SELECT 
    fr.id,
    u.id,
    CASE floor(random() * 5)::int
        WHEN 0 THEN 'This would be really helpful for my workflow!'
        WHEN 1 THEN 'I think this should be higher priority.'
        WHEN 2 THEN 'Great idea! I have some suggestions on implementation...'
        WHEN 3 THEN 'This is already partially possible with the current features.'
        ELSE 'Looking forward to seeing this implemented!'
    END
FROM feature_requests fr
CROSS JOIN users u
WHERE random() < 0.3 -- 30% chance of commenting
LIMIT 15;

-- Set scenario permissions (make first user owner of each scenario)
INSERT INTO scenario_permissions (scenario_id, user_id, role, can_propose, can_vote, can_moderate, can_edit_settings) VALUES
    ('11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'owner', true, true, true, true),
    ('22222222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'owner', true, true, true, true),
    ('33333333-3333-3333-3333-333333333333', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'owner', true, true, true, true),
    ('11111111-1111-1111-1111-111111111111', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'contributor', true, true, false, false),
    ('11111111-1111-1111-1111-111111111111', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 'contributor', true, true, false, false)
ON CONFLICT (scenario_id, user_id) DO NOTHING;

-- Add some status change history
INSERT INTO status_changes (feature_request_id, user_id, from_status, to_status, reason)
SELECT 
    id,
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    'proposed',
    status,
    'Initial review and prioritization'
FROM feature_requests
WHERE status != 'proposed'
LIMIT 5;