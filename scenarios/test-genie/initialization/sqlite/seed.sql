-- Seed data that mirrors what the CLI/API return for deterministic suite generation.
INSERT INTO suite_requests (
    id,
    scenario_name,
    requested_types,
    coverage_target,
    priority,
    status,
    notes,
    delegation_issue_id
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'sample-scenario',
    '["unit","integration"]',
    95,
    'normal',
    'completed',
    'Initial suite request seeded for dashboard previews',
    NULL
) ON CONFLICT(id) DO UPDATE SET
    status = 'completed',
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE suite_requests.status IN ('queued', 'delegated');
