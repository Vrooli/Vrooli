-- Data Backup Manager Database Schema
-- Stores backup metadata, schedules, and restore points

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create backup jobs table
CREATE TABLE IF NOT EXISTS backup_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('full', 'incremental', 'differential')),
    target VARCHAR(50) NOT NULL CHECK (target IN ('postgres', 'files', 'scenarios', 'minio', 'all')),
    target_identifier VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    size_bytes BIGINT DEFAULT 0,
    compression_ratio DECIMAL(3,2) DEFAULT 0.00,
    storage_path TEXT,
    checksum VARCHAR(64),
    retention_until TIMESTAMP WITH TIME ZONE,
    description TEXT,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create backup schedules table
CREATE TABLE IF NOT EXISTS backup_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    cron_expression VARCHAR(100) NOT NULL,
    backup_type VARCHAR(20) NOT NULL DEFAULT 'full' CHECK (backup_type IN ('full', 'incremental', 'differential')),
    targets TEXT[] NOT NULL,
    retention_days INTEGER NOT NULL DEFAULT 7,
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_run TIMESTAMP WITH TIME ZONE,
    next_run TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create restore points table (logical grouping of related backups)
CREATE TABLE IF NOT EXISTS restore_points (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    backup_job_ids UUID[] NOT NULL,
    description TEXT,
    verified BOOLEAN DEFAULT false,
    verification_date TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create restore operations table
CREATE TABLE IF NOT EXISTS restore_operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    restore_point_id UUID REFERENCES restore_points(id),
    backup_job_id UUID REFERENCES backup_jobs(id),
    targets TEXT[] NOT NULL,
    destination_path TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    verify_before_restore BOOLEAN DEFAULT true,
    verification_results JSONB DEFAULT '{}',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT restore_operations_source_check CHECK (
        (restore_point_id IS NOT NULL) OR (backup_job_id IS NOT NULL)
    )
);

-- Create backup file registry (tracks individual files within backups)
CREATE TABLE IF NOT EXISTS backup_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    backup_job_id UUID NOT NULL REFERENCES backup_jobs(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    relative_path TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    file_hash VARCHAR(64) NOT NULL,
    file_type VARCHAR(50),
    compression_ratio DECIMAL(3,2) DEFAULT 0.00,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create backup verification log
CREATE TABLE IF NOT EXISTS backup_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    backup_job_id UUID NOT NULL REFERENCES backup_jobs(id) ON DELETE CASCADE,
    verification_type VARCHAR(50) NOT NULL CHECK (verification_type IN ('checksum', 'size', 'restore_test', 'full')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('passed', 'failed', 'warning')),
    details JSONB DEFAULT '{}',
    error_message TEXT,
    verification_duration INTERVAL,
    verified_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    verified_by VARCHAR(255)
);

-- Create backup statistics table
CREATE TABLE IF NOT EXISTS backup_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    date DATE NOT NULL,
    target VARCHAR(50) NOT NULL,
    total_backups INTEGER DEFAULT 0,
    successful_backups INTEGER DEFAULT 0,
    failed_backups INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    average_duration INTERVAL,
    average_compression_ratio DECIMAL(3,2) DEFAULT 0.00,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date, target)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_backup_jobs_status ON backup_jobs(status);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_type ON backup_jobs(type);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_target ON backup_jobs(target);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_started_at ON backup_jobs(started_at);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_retention ON backup_jobs(retention_until);

CREATE INDEX IF NOT EXISTS idx_backup_schedules_enabled ON backup_schedules(enabled);
CREATE INDEX IF NOT EXISTS idx_backup_schedules_next_run ON backup_schedules(next_run);

CREATE INDEX IF NOT EXISTS idx_restore_operations_status ON restore_operations(status);
CREATE INDEX IF NOT EXISTS idx_restore_operations_started_at ON restore_operations(started_at);

CREATE INDEX IF NOT EXISTS idx_backup_files_job_id ON backup_files(backup_job_id);
CREATE INDEX IF NOT EXISTS idx_backup_files_path ON backup_files(relative_path);

CREATE INDEX IF NOT EXISTS idx_backup_verifications_job_id ON backup_verifications(backup_job_id);
CREATE INDEX IF NOT EXISTS idx_backup_verifications_status ON backup_verifications(status);

CREATE INDEX IF NOT EXISTS idx_backup_statistics_date ON backup_statistics(date);
CREATE INDEX IF NOT EXISTS idx_backup_statistics_target ON backup_statistics(target);

-- Create triggers for updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_backup_jobs_updated_at
    BEFORE UPDATE ON backup_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_backup_schedules_updated_at
    BEFORE UPDATE ON backup_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_restore_points_updated_at
    BEFORE UPDATE ON restore_points
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_restore_operations_updated_at
    BEFORE UPDATE ON restore_operations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_backup_statistics_updated_at
    BEFORE UPDATE ON backup_statistics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Insert default backup schedules
INSERT INTO backup_schedules (name, cron_expression, backup_type, targets, retention_days, enabled, created_by) VALUES
    ('Daily Full Backup', '0 2 * * *', 'full', ARRAY['postgres', 'files'], 7, true, 'system'),
    ('Hourly Incremental Backup', '0 * * * *', 'incremental', ARRAY['postgres'], 1, true, 'system'),
    ('Weekly Archive Backup', '0 1 * * 0', 'full', ARRAY['postgres', 'files', 'scenarios'], 30, false, 'system')
ON CONFLICT (name) DO NOTHING;

-- Create view for backup job summary
CREATE OR REPLACE VIEW backup_job_summary AS
SELECT 
    bj.id,
    bj.type,
    bj.target,
    bj.status,
    bj.started_at,
    bj.completed_at,
    bj.size_bytes,
    bj.compression_ratio,
    bj.description,
    EXTRACT(EPOCH FROM (bj.completed_at - bj.started_at)) AS duration_seconds,
    COUNT(bf.id) AS file_count,
    COUNT(bv.id) FILTER (WHERE bv.status = 'passed') AS verifications_passed,
    COUNT(bv.id) FILTER (WHERE bv.status = 'failed') AS verifications_failed
FROM backup_jobs bj
LEFT JOIN backup_files bf ON bj.id = bf.backup_job_id
LEFT JOIN backup_verifications bv ON bj.id = bv.backup_job_id
GROUP BY bj.id, bj.type, bj.target, bj.status, bj.started_at, bj.completed_at, 
         bj.size_bytes, bj.compression_ratio, bj.description;

-- Create view for system status
CREATE OR REPLACE VIEW backup_system_status AS
SELECT 
    COUNT(*) FILTER (WHERE status = 'running') AS active_jobs,
    COUNT(*) FILTER (WHERE status = 'completed' AND started_at > CURRENT_TIMESTAMP - INTERVAL '24 hours') AS jobs_last_24h,
    COUNT(*) FILTER (WHERE status = 'failed' AND started_at > CURRENT_TIMESTAMP - INTERVAL '24 hours') AS failed_jobs_last_24h,
    MAX(completed_at) FILTER (WHERE status = 'completed') AS last_successful_backup,
    SUM(size_bytes) FILTER (WHERE status = 'completed' AND retention_until > CURRENT_TIMESTAMP) AS total_backup_size,
    AVG(compression_ratio) FILTER (WHERE status = 'completed' AND started_at > CURRENT_TIMESTAMP - INTERVAL '7 days') AS avg_compression_ratio
FROM backup_jobs;

-- Create function to cleanup old backups
CREATE OR REPLACE FUNCTION cleanup_old_backups()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
BEGIN
    -- Delete expired backup jobs and their associated files
    WITH deleted_jobs AS (
        DELETE FROM backup_jobs 
        WHERE retention_until < CURRENT_TIMESTAMP 
        AND status IN ('completed', 'failed')
        RETURNING id
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted_jobs;
    
    -- Delete old backup statistics (keep 1 year)
    DELETE FROM backup_statistics 
    WHERE date < CURRENT_DATE - INTERVAL '1 year';
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to update backup statistics
CREATE OR REPLACE FUNCTION update_backup_statistics()
RETURNS VOID AS $$
BEGIN
    INSERT INTO backup_statistics (
        date, target, total_backups, successful_backups, failed_backups, 
        total_size_bytes, average_duration, average_compression_ratio
    )
    SELECT 
        DATE(started_at) as date,
        target,
        COUNT(*) as total_backups,
        COUNT(*) FILTER (WHERE status = 'completed') as successful_backups,
        COUNT(*) FILTER (WHERE status = 'failed') as failed_backups,
        SUM(size_bytes) FILTER (WHERE status = 'completed') as total_size_bytes,
        AVG(completed_at - started_at) FILTER (WHERE status = 'completed') as average_duration,
        AVG(compression_ratio) FILTER (WHERE status = 'completed') as average_compression_ratio
    FROM backup_jobs 
    WHERE DATE(started_at) = CURRENT_DATE
    GROUP BY DATE(started_at), target
    ON CONFLICT (date, target) DO UPDATE SET
        total_backups = EXCLUDED.total_backups,
        successful_backups = EXCLUDED.successful_backups,
        failed_backups = EXCLUDED.failed_backups,
        total_size_bytes = EXCLUDED.total_size_bytes,
        average_duration = EXCLUDED.average_duration,
        average_compression_ratio = EXCLUDED.average_compression_ratio,
        updated_at = CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (adjust as needed for your setup)
-- GRANT ALL ON ALL TABLES IN SCHEMA public TO backup_manager;
-- GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO backup_manager;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO backup_manager;

-- Initialize the schema
SELECT 'Data Backup Manager database schema initialized successfully!' as status;