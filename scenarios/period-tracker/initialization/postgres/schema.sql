-- Period Tracker Database Schema
-- Privacy-first menstrual health tracking with multi-tenant support
-- All sensitive fields are encrypted at the application layer

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table for multi-tenant support
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email_encrypted TEXT, -- Encrypted at application layer
    password_hash VARCHAR(255) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true
);

-- Index for efficient user lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active) WHERE is_active = true;

-- Menstrual cycles table
CREATE TABLE IF NOT EXISTS cycles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    end_date DATE, -- NULL until cycle ends
    cycle_length INTEGER, -- Computed when cycle ends
    flow_intensity VARCHAR(20) CHECK (flow_intensity IN ('spotting', 'light', 'medium', 'heavy', 'very_heavy')),
    notes_encrypted TEXT, -- Encrypted at application layer
    is_predicted BOOLEAN DEFAULT false,
    confidence_score FLOAT CHECK (confidence_score >= 0 AND confidence_score <= 1),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for cycle queries
CREATE INDEX IF NOT EXISTS idx_cycles_user_id ON cycles(user_id);
CREATE INDEX IF NOT EXISTS idx_cycles_start_date ON cycles(start_date);
CREATE INDEX IF NOT EXISTS idx_cycles_user_date ON cycles(user_id, start_date);
CREATE INDEX IF NOT EXISTS idx_cycles_predicted ON cycles(is_predicted);

-- Daily symptoms tracking
CREATE TABLE IF NOT EXISTS daily_symptoms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cycle_id UUID REFERENCES cycles(id) ON DELETE SET NULL,
    symptom_date DATE NOT NULL,
    -- Physical symptoms (encrypted JSON array)
    physical_symptoms_encrypted TEXT,
    -- Mood and emotional symptoms  
    mood_rating INTEGER CHECK (mood_rating >= 1 AND mood_rating <= 10),
    energy_level INTEGER CHECK (energy_level >= 1 AND energy_level <= 10),
    stress_level INTEGER CHECK (stress_level >= 1 AND stress_level <= 10),
    -- Pain tracking
    cramp_intensity INTEGER CHECK (cramp_intensity >= 0 AND cramp_intensity <= 10),
    headache_intensity INTEGER CHECK (headache_intensity >= 0 AND headache_intensity <= 10),
    breast_tenderness INTEGER CHECK (breast_tenderness >= 0 AND breast_tenderness <= 10),
    back_pain INTEGER CHECK (back_pain >= 0 AND back_pain <= 10),
    -- Flow tracking
    flow_level VARCHAR(20) CHECK (flow_level IN ('none', 'spotting', 'light', 'medium', 'heavy', 'very_heavy')),
    -- Additional notes (encrypted)
    notes_encrypted TEXT,
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    -- Ensure one entry per user per date
    UNIQUE(user_id, symptom_date)
);

-- Indexes for symptom queries
CREATE INDEX IF NOT EXISTS idx_daily_symptoms_user_id ON daily_symptoms(user_id);
CREATE INDEX IF NOT EXISTS idx_daily_symptoms_date ON daily_symptoms(symptom_date);
CREATE INDEX IF NOT EXISTS idx_daily_symptoms_user_date ON daily_symptoms(user_id, symptom_date);
CREATE INDEX IF NOT EXISTS idx_daily_symptoms_cycle ON daily_symptoms(cycle_id);

-- Cycle predictions for AI-driven forecasting
CREATE TABLE IF NOT EXISTS predictions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    predicted_start_date DATE NOT NULL,
    predicted_end_date DATE,
    predicted_length INTEGER,
    confidence_score FLOAT CHECK (confidence_score >= 0 AND confidence_score <= 1),
    algorithm_version VARCHAR(50) NOT NULL,
    input_data_hash VARCHAR(255), -- For tracking what data was used
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true -- For managing prediction history
);

-- Indexes for predictions
CREATE INDEX IF NOT EXISTS idx_predictions_user_id ON predictions(user_id);
CREATE INDEX IF NOT EXISTS idx_predictions_start_date ON predictions(predicted_start_date);
CREATE INDEX IF NOT EXISTS idx_predictions_active ON predictions(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_predictions_user_active ON predictions(user_id, is_active) WHERE is_active = true;

-- Pattern detection results for AI insights
CREATE TABLE IF NOT EXISTS detected_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pattern_type VARCHAR(100) NOT NULL, -- e.g., 'headache_correlation', 'mood_pattern'
    pattern_description_encrypted TEXT, -- Human-readable description (encrypted)
    correlation_strength FLOAT CHECK (correlation_strength >= -1 AND correlation_strength <= 1),
    statistical_significance FLOAT,
    data_points_count INTEGER,
    first_observed DATE,
    last_observed DATE,
    algorithm_version VARCHAR(50),
    confidence_level VARCHAR(20) CHECK (confidence_level IN ('low', 'medium', 'high')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for patterns
CREATE INDEX IF NOT EXISTS idx_patterns_user_id ON detected_patterns(user_id);
CREATE INDEX IF NOT EXISTS idx_patterns_type ON detected_patterns(pattern_type);
CREATE INDEX IF NOT EXISTS idx_patterns_active ON detected_patterns(is_active) WHERE is_active = true;

-- Medication and supplement tracking
CREATE TABLE IF NOT EXISTS medications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name_encrypted TEXT NOT NULL, -- Encrypted medication name
    dosage_encrypted TEXT, -- Encrypted dosage information
    frequency VARCHAR(50), -- daily, weekly, as_needed
    start_date DATE,
    end_date DATE,
    reminder_times JSONB, -- Array of time strings for reminders
    is_birth_control BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    notes_encrypted TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for medications
CREATE INDEX IF NOT EXISTS idx_medications_user_id ON medications(user_id);
CREATE INDEX IF NOT EXISTS idx_medications_active ON medications(is_active) WHERE is_active = true;

-- Medication logs for tracking adherence
CREATE TABLE IF NOT EXISTS medication_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    medication_id UUID NOT NULL REFERENCES medications(id) ON DELETE CASCADE,
    log_date DATE NOT NULL,
    time_taken TIME,
    was_taken BOOLEAN NOT NULL,
    notes_encrypted TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for medication logs
CREATE INDEX IF NOT EXISTS idx_med_logs_user_id ON medication_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_med_logs_date ON medication_logs(log_date);
CREATE INDEX IF NOT EXISTS idx_med_logs_medication ON medication_logs(medication_id);

-- Export requests for medical appointments
CREATE TABLE IF NOT EXISTS export_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    export_type VARCHAR(50) NOT NULL, -- 'medical_summary', 'cycle_history', 'symptom_report'
    date_range_start DATE NOT NULL,
    date_range_end DATE NOT NULL,
    file_path TEXT, -- Where the exported file is stored
    file_hash VARCHAR(255), -- For integrity verification
    expiry_date TIMESTAMP WITH TIME ZONE, -- When to delete the export
    is_anonymized BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for exports
CREATE INDEX IF NOT EXISTS idx_exports_user_id ON export_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_exports_expiry ON export_requests(expiry_date);

-- Partner sharing permissions (for consensual data sharing)
CREATE TABLE IF NOT EXISTS sharing_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shared_with_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    shared_with_external_id VARCHAR(255), -- For non-Vrooli partners
    permission_type VARCHAR(50) NOT NULL, -- 'cycle_predictions', 'symptom_summary', 'full_access'
    is_active BOOLEAN DEFAULT true,
    expiry_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for sharing
CREATE INDEX IF NOT EXISTS idx_sharing_user_id ON sharing_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_sharing_with ON sharing_permissions(shared_with_user_id);
CREATE INDEX IF NOT EXISTS idx_sharing_active ON sharing_permissions(is_active) WHERE is_active = true;

-- Audit logs for privacy compliance
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL, -- 'data_accessed', 'data_modified', 'data_exported', 'data_shared'
    table_name VARCHAR(100),
    record_id UUID,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for audit logs
CREATE INDEX IF NOT EXISTS idx_audit_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at);

-- Create update timestamp trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update triggers to relevant tables
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cycles_updated_at BEFORE UPDATE ON cycles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_daily_symptoms_updated_at BEFORE UPDATE ON daily_symptoms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_patterns_updated_at BEFORE UPDATE ON detected_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_medications_updated_at BEFORE UPDATE ON medications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sharing_updated_at BEFORE UPDATE ON sharing_permissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Row Level Security (RLS) for multi-tenant data isolation
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE cycles ENABLE ROW LEVEL SECURITY;
ALTER TABLE daily_symptoms ENABLE ROW LEVEL SECURITY;
ALTER TABLE predictions ENABLE ROW LEVEL SECURITY;
ALTER TABLE detected_patterns ENABLE ROW LEVEL SECURITY;
ALTER TABLE medications ENABLE ROW LEVEL SECURITY;
ALTER TABLE medication_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE export_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE sharing_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- RLS Policies (will be set by application based on authenticated user)
-- These are placeholder policies - the application will set proper policies based on session context

-- Create a view for cycle statistics (commonly queried data)
CREATE OR REPLACE VIEW cycle_statistics AS
SELECT 
    user_id,
    COUNT(*) as total_cycles,
    AVG(cycle_length) as avg_cycle_length,
    STDDEV(cycle_length) as cycle_length_variation,
    MIN(start_date) as first_cycle_date,
    MAX(start_date) as last_cycle_date,
    COUNT(*) FILTER (WHERE flow_intensity = 'heavy') as heavy_flow_count
FROM cycles 
WHERE is_predicted = false AND cycle_length IS NOT NULL
GROUP BY user_id;

-- Create a view for recent symptoms (for quick dashboard queries)
CREATE OR REPLACE VIEW recent_symptoms AS
SELECT 
    user_id,
    symptom_date,
    mood_rating,
    energy_level,
    cramp_intensity,
    flow_level,
    created_at
FROM daily_symptoms 
WHERE symptom_date >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY user_id, symptom_date DESC;

-- Performance optimization: Create partial indexes for common queries
CREATE INDEX IF NOT EXISTS idx_cycles_recent ON cycles(user_id, start_date DESC) 
    WHERE start_date >= CURRENT_DATE - INTERVAL '1 year';

CREATE INDEX IF NOT EXISTS idx_symptoms_recent ON daily_symptoms(user_id, symptom_date DESC) 
    WHERE symptom_date >= CURRENT_DATE - INTERVAL '3 months';

-- Comments for documentation
COMMENT ON TABLE users IS 'Multi-tenant user accounts with encrypted personal information';
COMMENT ON TABLE cycles IS 'Menstrual cycle records with flow tracking and predictions';
COMMENT ON TABLE daily_symptoms IS 'Daily symptom logging for pattern recognition';
COMMENT ON TABLE predictions IS 'AI-generated cycle predictions with confidence scores';
COMMENT ON TABLE detected_patterns IS 'Machine learning detected health patterns';
COMMENT ON TABLE medications IS 'User medication and supplement tracking';
COMMENT ON TABLE audit_logs IS 'Privacy compliance audit trail for all data access';

-- Grant permissions (will be refined by the application)
-- The application should use a dedicated database user with minimal privileges