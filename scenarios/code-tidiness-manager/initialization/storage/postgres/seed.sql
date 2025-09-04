-- Code Tidiness Manager Seed Data
-- Initial cleanup rules and patterns

SET search_path TO tidiness, public;

-- Insert default cleanup rules
INSERT INTO cleanup_rules (rule_name, category, pattern, pattern_type, action_template, description, priority, enabled, auto_apply, confidence_threshold, created_by)
VALUES 
    -- Backup files
    ('backup_files', 'file_cleanup', '*.bak', 'glob', 'rm -f {file_path}', 'Remove backup files (.bak)', 80, true, false, 0.95, 'system'),
    ('tilde_backup', 'file_cleanup', '*~', 'glob', 'rm -f {file_path}', 'Remove tilde backup files', 80, true, false, 0.95, 'system'),
    ('orig_files', 'file_cleanup', '*.orig', 'glob', 'rm -f {file_path}', 'Remove .orig files from merges', 75, true, false, 0.90, 'system'),
    ('old_files', 'file_cleanup', '*.old', 'glob', 'rm -f {file_path}', 'Remove .old backup files', 75, true, false, 0.90, 'system'),
    
    -- Temporary files
    ('temp_files', 'file_cleanup', '*/tmp/*', 'glob', 'rm -f {file_path}', 'Remove temporary files in tmp directories', 70, true, false, 0.85, 'system'),
    ('swp_files', 'file_cleanup', '.*.swp', 'glob', 'rm -f {file_path}', 'Remove vim swap files', 85, true, false, 0.95, 'system'),
    ('ds_store', 'file_cleanup', '.DS_Store', 'exact', 'rm -f {file_path}', 'Remove macOS .DS_Store files', 90, true, true, 0.99, 'system'),
    ('thumbs_db', 'file_cleanup', 'Thumbs.db', 'exact', 'rm -f {file_path}', 'Remove Windows Thumbs.db files', 90, true, true, 0.99, 'system'),
    
    -- Empty directories
    ('empty_dirs', 'directory_cleanup', '', 'custom', 'rmdir {file_path}', 'Remove empty directories', 60, true, false, 0.80, 'system'),
    
    -- Node.js specific
    ('node_modules_backup', 'nodejs_cleanup', 'node_modules.bak', 'exact', 'rm -rf {file_path}', 'Remove backed up node_modules', 70, true, false, 0.85, 'system'),
    ('npm_debug_log', 'nodejs_cleanup', 'npm-debug.log*', 'glob', 'rm -f {file_path}', 'Remove npm debug logs', 85, true, false, 0.95, 'system'),
    ('yarn_error_log', 'nodejs_cleanup', 'yarn-error.log', 'exact', 'rm -f {file_path}', 'Remove yarn error logs', 85, true, false, 0.95, 'system'),
    
    -- Python specific
    ('pyc_files', 'python_cleanup', '*.pyc', 'glob', 'rm -f {file_path}', 'Remove Python compiled files', 80, true, false, 0.90, 'system'),
    ('pycache_dirs', 'python_cleanup', '__pycache__', 'exact', 'rm -rf {file_path}', 'Remove Python cache directories', 80, true, false, 0.90, 'system'),
    ('python_egg', 'python_cleanup', '*.egg-info', 'glob', 'rm -rf {file_path}', 'Remove Python egg info directories', 75, true, false, 0.85, 'system'),
    
    -- Go specific
    ('go_test_cache', 'go_cleanup', 'go.sum.bak', 'exact', 'rm -f {file_path}', 'Remove Go sum backup files', 75, true, false, 0.90, 'system'),
    ('go_binary_backup', 'go_cleanup', '*.exe~', 'glob', 'rm -f {file_path}', 'Remove Go binary backups', 75, true, false, 0.90, 'system'),
    
    -- IDE specific
    ('idea_dir', 'ide_cleanup', '.idea', 'exact', NULL, 'IntelliJ IDEA configuration (manual review)', 40, true, false, 0.70, 'system'),
    ('vscode_dir', 'ide_cleanup', '.vscode', 'exact', NULL, 'VS Code configuration (manual review)', 40, true, false, 0.70, 'system'),
    
    -- Build artifacts
    ('build_dist_backup', 'build_cleanup', 'dist.bak', 'exact', 'rm -rf {file_path}', 'Remove dist backup directories', 70, true, false, 0.85, 'system'),
    ('build_out_backup', 'build_cleanup', 'build.old', 'exact', 'rm -rf {file_path}', 'Remove old build directories', 70, true, false, 0.85, 'system'),
    
    -- Log files
    ('old_logs', 'log_cleanup', '*.log.[0-9]*', 'regex', 'rm -f {file_path}', 'Remove rotated log files', 65, true, false, 0.80, 'system'),
    ('debug_logs', 'log_cleanup', 'debug.log', 'exact', 'rm -f {file_path}', 'Remove debug log files', 70, true, false, 0.85, 'system'),
    
    -- Docker specific
    ('dockerfile_backup', 'docker_cleanup', 'Dockerfile.bak', 'exact', 'rm -f {file_path}', 'Remove Dockerfile backups', 75, true, false, 0.90, 'system'),
    ('dockerignore_backup', 'docker_cleanup', '.dockerignore.bak', 'exact', 'rm -f {file_path}', 'Remove dockerignore backups', 75, true, false, 0.90, 'system'),
    
    -- Git specific (careful with these)
    ('git_conflict_markers', 'git_cleanup', '*.rej', 'glob', NULL, 'Git conflict rejection files (manual review)', 30, true, false, 0.60, 'system'),
    ('git_patch_files', 'git_cleanup', '*.patch', 'glob', NULL, 'Git patch files (manual review)', 30, true, false, 0.60, 'system'),
    
    -- Documentation
    ('markdown_backup', 'doc_cleanup', '*.md.bak', 'glob', 'rm -f {file_path}', 'Remove markdown backup files', 80, true, false, 0.95, 'system'),
    ('readme_backup', 'doc_cleanup', 'README*.bak', 'glob', 'rm -f {file_path}', 'Remove README backup files', 80, true, false, 0.95, 'system'),
    
    -- Test artifacts  
    ('coverage_backup', 'test_cleanup', 'coverage.bak', 'exact', 'rm -rf {file_path}', 'Remove coverage backup directories', 70, true, false, 0.85, 'system'),
    ('test_results_old', 'test_cleanup', 'test-results.old', 'exact', 'rm -rf {file_path}', 'Remove old test result directories', 70, true, false, 0.85, 'system'),
    
    -- Complex patterns (no auto-cleanup)
    ('duplicate_scenarios', 'architectural', NULL, 'custom', NULL, 'Detect scenarios with overlapping functionality', 20, true, false, 0.50, 'system'),
    ('unused_imports', 'code_quality', NULL, 'custom', NULL, 'Detect unused import statements', 30, true, false, 0.60, 'system'),
    ('dead_code', 'code_quality', NULL, 'custom', NULL, 'Detect unreachable/unused code', 25, true, false, 0.50, 'system'),
    ('large_files', 'performance', NULL, 'custom', NULL, 'Detect unusually large files (>10MB)', 35, true, false, 0.70, 'system'),
    ('credential_patterns', 'security', NULL, 'custom', NULL, 'Detect potential hardcoded credentials', 10, true, false, 0.30, 'system')
ON CONFLICT (rule_name) DO UPDATE
SET 
    updated_at = CURRENT_TIMESTAMP,
    description = EXCLUDED.description,
    priority = EXCLUDED.priority;

-- Insert sample debt metrics (optional, for dashboard testing)
INSERT INTO debt_metrics (
    total_debt_score, 
    debt_by_category, 
    debt_by_scenario,
    debt_trend,
    critical_issues,
    high_issues,
    medium_issues,
    low_issues,
    estimated_cleanup_hours
) VALUES (
    142.5,
    '{"file_cleanup": 45, "code_quality": 38, "architectural": 29, "security": 15, "performance": 15.5}'::JSONB,
    '{"scenario-generator": 22, "agent-dashboard": 18, "study-buddy": 15, "system-monitor": 12}'::JSONB,
    'decreasing',
    2,
    8,
    24,
    67,
    8.5
);

-- Create notification preferences table (for future expansion)
CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255),
    notify_critical BOOLEAN DEFAULT true,
    notify_high BOOLEAN DEFAULT true,
    notify_medium BOOLEAN DEFAULT false,
    notify_low BOOLEAN DEFAULT false,
    daily_summary BOOLEAN DEFAULT true,
    weekly_report BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert default notification preferences
INSERT INTO notification_preferences (user_id, email, notify_critical, notify_high, daily_summary)
VALUES ('system', 'admin@localhost', true, true, true)
ON CONFLICT (user_id) DO NOTHING;

-- Create a function to estimate cleanup impact
CREATE OR REPLACE FUNCTION tidiness.estimate_cleanup_impact()
RETURNS TABLE(
    total_files BIGINT,
    total_size_mb DECIMAL,
    automatable_files BIGINT,
    automatable_size_mb DECIMAL,
    manual_review_files BIGINT,
    estimated_time_saved_hours DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    WITH latest_scan AS (
        SELECT scan_id 
        FROM scan_history 
        WHERE status = 'completed' 
        ORDER BY completed_at DESC 
        LIMIT 1
    )
    SELECT 
        COUNT(DISTINCT sr.file_path) as total_files,
        ROUND(SUM(sr.size_bytes) / 1048576.0, 2) as total_size_mb,
        COUNT(DISTINCT sr.file_path) FILTER (WHERE sr.safe_to_automate = true) as automatable_files,
        ROUND(SUM(sr.size_bytes) FILTER (WHERE sr.safe_to_automate = true) / 1048576.0, 2) as automatable_size_mb,
        COUNT(DISTINCT sr.file_path) FILTER (WHERE sr.requires_human_review = true) as manual_review_files,
        ROUND(COUNT(DISTINCT sr.file_path) * 0.05, 2) as estimated_time_saved_hours
    FROM scan_results sr
    JOIN latest_scan ls ON sr.scan_id = ls.scan_id;
END;
$$ LANGUAGE plpgsql;

-- Output confirmation
DO $$
BEGIN
    RAISE NOTICE 'Code Tidiness Manager database initialized with % cleanup rules', 
        (SELECT COUNT(*) FROM cleanup_rules);
END
$$;