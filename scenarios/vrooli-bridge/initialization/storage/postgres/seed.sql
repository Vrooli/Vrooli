-- Vrooli Bridge Seed Data
-- Initial data for testing and demonstration

-- Example project types for reference
INSERT INTO projects (name, path, type, integration_status, metadata) VALUES 
('vrooli-bridge-test', '/tmp/test-projects/sample-npm', 'npm', 'missing', '{"description": "Sample NPM project for testing"}'),
('sample-rust-project', '/tmp/test-projects/sample-rust', 'cargo', 'missing', '{"description": "Sample Rust project for testing"}')
ON CONFLICT (path) DO NOTHING;

-- Add some sample tags
INSERT INTO project_tags (project_id, tag)
SELECT p.id, 'test-project'
FROM projects p
WHERE p.path LIKE '/tmp/test-projects/%'
ON CONFLICT (project_id, tag) DO NOTHING;

INSERT INTO project_tags (project_id, tag)
SELECT p.id, 'demo'
FROM projects p 
WHERE p.name IN ('vrooli-bridge-test', 'sample-rust-project')
ON CONFLICT (project_id, tag) DO NOTHING;