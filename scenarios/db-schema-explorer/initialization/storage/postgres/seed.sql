-- Database Schema Explorer - Seed Data
-- Initial data for testing and demonstration

SET search_path TO db_explorer, public;

-- Insert default database connections
INSERT INTO database_connections (connection_name, database_type, host, port, database_name, schema_name, is_active, metadata)
VALUES 
    ('main', 'postgres', 'localhost', 5432, 'main', 'public', true, 
     '{"description": "Main Vrooli database", "category": "production"}'),
    ('test', 'postgres', 'localhost', 5432, 'test', 'public', true,
     '{"description": "Test database for scenarios", "category": "development"}')
ON CONFLICT (connection_name) DO NOTHING;

-- Insert sample query patterns
INSERT INTO query_patterns (pattern_name, pattern_type, pattern_template, usage_count, success_rate, tables_involved)
VALUES
    ('List All Tables', 'metadata', 
     'SELECT table_name FROM information_schema.tables WHERE table_schema = $1',
     42, 100.0, ARRAY['information_schema.tables']),
    
    ('Count Table Rows', 'aggregation',
     'SELECT COUNT(*) FROM $table_name',
     38, 98.5, ARRAY[]::TEXT[]),
    
    ('Find Foreign Keys', 'metadata',
     'SELECT constraint_name, table_name, column_name FROM information_schema.key_column_usage WHERE constraint_name LIKE ''%_fkey''',
     25, 100.0, ARRAY['information_schema.key_column_usage']),
    
    ('Table Column Details', 'metadata',
     'SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = $1',
     35, 100.0, ARRAY['information_schema.columns']),
    
    ('Recent Records', 'filter',
     'SELECT * FROM $table_name ORDER BY created_at DESC LIMIT 10',
     28, 95.0, ARRAY[]::TEXT[])
ON CONFLICT DO NOTHING;

-- Insert sample visualization layouts
INSERT INTO visualization_layouts (name, database_name, schema_name, layout_type, layout_data, is_default, is_shared)
VALUES
    ('Default Layout', 'main', 'public', 'standard',
     '{"zoom": 1.0, "center": {"x": 0, "y": 0}, "selectedTables": [], "collapsedTables": []}',
     true, true),
    
    ('Compact View', 'main', 'public', 'compact',
     '{"zoom": 0.8, "center": {"x": 0, "y": 0}, "showOnlyRelated": true, "minimizeColumns": true}',
     false, true),
    
    ('Relationship Focus', 'main', 'public', 'relationship',
     '{"zoom": 1.2, "center": {"x": 0, "y": 0}, "highlightForeignKeys": true, "hideUnrelatedTables": true}',
     false, true)
ON CONFLICT DO NOTHING;

-- Insert sample query history (will be enriched with Qdrant vectors in runtime)
INSERT INTO query_history (
    natural_language, 
    generated_sql, 
    database_name, 
    schema_name,
    execution_time_ms, 
    result_count, 
    query_type, 
    tables_used,
    user_feedback,
    confidence_score
)
VALUES
    ('Show me all tables in the database',
     'SELECT table_name, table_type FROM information_schema.tables WHERE table_schema = ''public'';',
     'main', 'public', 125, 15, 'SELECT', ARRAY['information_schema.tables'],
     'helpful', 95),
    
    ('How many users are in the system?',
     'SELECT COUNT(*) as user_count FROM users;',
     'main', 'public', 45, 1, 'SELECT', ARRAY['users'],
     'helpful', 90),
    
    ('Find tables with more than 10 columns',
     'SELECT table_name, COUNT(column_name) as column_count FROM information_schema.columns WHERE table_schema = ''public'' GROUP BY table_name HAVING COUNT(column_name) > 10;',
     'main', 'public', 230, 3, 'SELECT', ARRAY['information_schema.columns'],
     'helpful', 88),
    
    ('Show recent activity',
     'SELECT * FROM activity_logs ORDER BY created_at DESC LIMIT 100;',
     'main', 'public', 180, 100, 'SELECT', ARRAY['activity_logs'],
     null, 85),
    
    ('What indexes exist on the users table?',
     'SELECT indexname, indexdef FROM pg_indexes WHERE tablename = ''users'';',
     'main', 'public', 95, 4, 'SELECT', ARRAY['pg_indexes'],
     'helpful', 92)
ON CONFLICT DO NOTHING;

-- Create a simple function to generate test schema snapshot
CREATE OR REPLACE FUNCTION generate_test_snapshot(p_database_name VARCHAR)
RETURNS UUID AS $$
DECLARE
    v_snapshot_id UUID;
BEGIN
    INSERT INTO schema_snapshots (
        database_name,
        schema_name,
        tables_count,
        columns_count,
        relationships_count,
        indexes_count,
        schema_data,
        version
    )
    VALUES (
        p_database_name,
        'public',
        15,  -- Example counts
        127,
        23,
        38,
        '{
            "tables": [
                {"name": "users", "columns": 12, "rows_estimate": 1000},
                {"name": "projects", "columns": 8, "rows_estimate": 500},
                {"name": "tasks", "columns": 15, "rows_estimate": 5000}
            ],
            "relationships": [
                {"from": "tasks.project_id", "to": "projects.id"},
                {"from": "tasks.user_id", "to": "users.id"},
                {"from": "projects.owner_id", "to": "users.id"}
            ]
        }'::JSONB,
        '1.0.0'
    )
    RETURNING id INTO v_snapshot_id;
    
    RETURN v_snapshot_id;
END;
$$ LANGUAGE plpgsql;

-- Generate initial snapshots for default connections
SELECT generate_test_snapshot('main');
SELECT generate_test_snapshot('test');

-- Add sample schema analysis results
INSERT INTO schema_analysis (
    snapshot_id,
    analysis_type,
    findings,
    severity,
    recommendations
)
SELECT 
    ss.id,
    'consistency',
    '{"issues": [
        {"type": "naming", "description": "Inconsistent naming convention: some tables use snake_case, others use camelCase"},
        {"type": "missing_pk", "description": "Table ''audit_logs'' missing primary key"}
    ]}'::JSONB,
    'warning',
    '{"suggestions": [
        "Standardize table naming to snake_case",
        "Add primary key to audit_logs table"
    ]}'::JSONB
FROM schema_snapshots ss
WHERE ss.database_name = 'main'
LIMIT 1;

-- Create helpful notice
DO $$
BEGIN
    RAISE NOTICE 'Database Schema Explorer seed data loaded successfully!';
    RAISE NOTICE 'Default connections created: main, test';
    RAISE NOTICE 'Sample query patterns and layouts added';
    RAISE NOTICE 'Run the application to start exploring your databases!';
END $$;