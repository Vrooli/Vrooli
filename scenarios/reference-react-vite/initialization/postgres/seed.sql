-- Seed data for reference-react-vite demonstration
-- All inserts use ON CONFLICT DO NOTHING for idempotency

-- Sample project for demonstration
INSERT INTO projects (id, name, description, status, color)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Getting Started',
    'A sample project to help you explore the reference scenario',
    'active',
    '#3B82F6'
) ON CONFLICT (id) DO NOTHING;

-- Sample tasks for the demonstration project
INSERT INTO tasks (id, project_id, title, description, status, priority)
VALUES (
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000001',
    'Explore the API',
    'Try the /api/v1/tasks, /api/v1/projects, and /api/v1/notes endpoints',
    'pending',
    2
) ON CONFLICT (id) DO NOTHING;

INSERT INTO tasks (id, project_id, title, description, status, priority)
VALUES (
    '00000000-0000-0000-0000-000000000102',
    '00000000-0000-0000-0000-000000000001',
    'Review the code structure',
    'Examine the screaming architecture: domain/, handlers/, repository/',
    'pending',
    2
) ON CONFLICT (id) DO NOTHING;

INSERT INTO tasks (id, project_id, title, description, status, priority)
VALUES (
    '00000000-0000-0000-0000-000000000103',
    '00000000-0000-0000-0000-000000000001',
    'Run the test suite',
    'Execute test-genie to verify all phases pass',
    'pending',
    1
) ON CONFLICT (id) DO NOTHING;

-- Sample note on the first task
INSERT INTO notes (id, task_id, content, author)
VALUES (
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000101',
    'The API follows REST conventions with consistent error handling and pagination.',
    'Reference Guide'
) ON CONFLICT (id) DO NOTHING;
