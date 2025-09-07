-- Pregnancy Tracker Database Schema
-- Privacy-first design with encryption support and multi-tenant isolation

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create schema for pregnancy tracker
CREATE SCHEMA IF NOT EXISTS pregnancy_tracker;
SET search_path TO pregnancy_tracker, public;

-- Pregnancies table (core entity)
CREATE TABLE IF NOT EXISTS pregnancies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    lmp_date DATE NOT NULL,
    conception_date DATE,
    due_date DATE NOT NULL,
    current_week INTEGER DEFAULT 0,
    current_day INTEGER DEFAULT 0,
    pregnancy_type VARCHAR(50) DEFAULT 'singleton' CHECK (pregnancy_type IN ('singleton', 'twins', 'multiples')),
    outcome VARCHAR(50) DEFAULT 'ongoing' CHECK (outcome IN ('ongoing', 'live_birth', 'loss', 'terminated', 'unknown')),
    outcome_date DATE,
    baby_count INTEGER DEFAULT 1,
    risk_factors JSONB DEFAULT '[]'::jsonb,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, created_at)
);

-- Daily logs table (encrypted sensitive data)
CREATE TABLE IF NOT EXISTS daily_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pregnancy_id UUID NOT NULL REFERENCES pregnancies(id) ON DELETE CASCADE,
    log_date DATE NOT NULL,
    weight_value DECIMAL(5,2),
    weight_unit VARCHAR(10) DEFAULT 'lbs',
    blood_pressure_systolic INTEGER,
    blood_pressure_diastolic INTEGER,
    symptoms JSONB DEFAULT '[]'::jsonb,
    mood INTEGER CHECK (mood >= 1 AND mood <= 10),
    energy INTEGER CHECK (energy >= 1 AND energy <= 10),
    sleep_hours DECIMAL(3,1),
    water_intake INTEGER,
    exercise_minutes INTEGER,
    notes_encrypted BYTEA, -- Encrypted with user key
    photos JSONB DEFAULT '[]'::jsonb, -- Array of photo URLs or base64
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pregnancy_id, log_date)
);

-- Appointments table
CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pregnancy_id UUID NOT NULL REFERENCES pregnancies(id) ON DELETE CASCADE,
    appointment_date TIMESTAMP WITH TIME ZONE NOT NULL,
    appointment_type VARCHAR(50) NOT NULL CHECK (appointment_type IN ('ob', 'ultrasound', 'lab', 'specialist', 'class', 'other')),
    provider_name VARCHAR(255),
    provider_phone VARCHAR(50),
    location_name VARCHAR(255),
    location_address TEXT,
    purpose TEXT,
    notes TEXT,
    results JSONB DEFAULT '{}'::jsonb,
    reminder_sent BOOLEAN DEFAULT FALSE,
    reminder_time TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'completed', 'cancelled', 'rescheduled')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Kick counts table
CREATE TABLE IF NOT EXISTS kick_counts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pregnancy_id UUID NOT NULL REFERENCES pregnancies(id) ON DELETE CASCADE,
    session_start TIMESTAMP WITH TIME ZONE NOT NULL,
    session_end TIMESTAMP WITH TIME ZONE,
    kick_count INTEGER NOT NULL DEFAULT 0,
    duration_minutes INTEGER,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Contraction timer table
CREATE TABLE IF NOT EXISTS contractions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pregnancy_id UUID NOT NULL REFERENCES pregnancies(id) ON DELETE CASCADE,
    contraction_start TIMESTAMP WITH TIME ZONE NOT NULL,
    contraction_end TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    intensity INTEGER CHECK (intensity >= 1 AND intensity <= 10),
    interval_minutes INTEGER, -- Time since last contraction
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Emergency information table (encrypted)
CREATE TABLE IF NOT EXISTS emergency_info (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL UNIQUE,
    blood_type VARCHAR(10),
    rh_factor VARCHAR(10),
    allergies_encrypted BYTEA, -- Encrypted JSON array
    medications_encrypted BYTEA, -- Encrypted JSON array
    medical_conditions_encrypted BYTEA, -- Encrypted JSON array
    ob_name VARCHAR(255),
    ob_phone VARCHAR(50),
    ob_practice VARCHAR(255),
    hospital_preference VARCHAR(255),
    hospital_address TEXT,
    hospital_phone VARCHAR(50),
    emergency_contacts_encrypted BYTEA, -- Encrypted JSON array
    insurance_info_encrypted BYTEA, -- Encrypted JSON
    birth_plan_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Medications/Supplements tracking
CREATE TABLE IF NOT EXISTS medications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pregnancy_id UUID NOT NULL REFERENCES pregnancies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    dosage VARCHAR(100),
    frequency VARCHAR(100),
    start_date DATE NOT NULL,
    end_date DATE,
    prescribed_by VARCHAR(255),
    reason TEXT,
    notes TEXT,
    reminder_enabled BOOLEAN DEFAULT FALSE,
    reminder_times JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Weight tracking with goals
CREATE TABLE IF NOT EXISTS weight_tracking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pregnancy_id UUID NOT NULL REFERENCES pregnancies(id) ON DELETE CASCADE,
    week_number INTEGER NOT NULL,
    weight_value DECIMAL(5,2) NOT NULL,
    weight_unit VARCHAR(10) DEFAULT 'lbs',
    goal_min DECIMAL(5,2),
    goal_max DECIMAL(5,2),
    notes TEXT,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pregnancy_id, week_number)
);

-- Partner access table
CREATE TABLE IF NOT EXISTS partner_access (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pregnancy_id UUID NOT NULL REFERENCES pregnancies(id) ON DELETE CASCADE,
    partner_user_id VARCHAR(255) NOT NULL,
    access_level VARCHAR(50) DEFAULT 'read_only' CHECK (access_level IN ('read_only', 'contribute', 'full')),
    access_granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    access_revoked_at TIMESTAMP WITH TIME ZONE,
    invite_code VARCHAR(100) UNIQUE,
    invite_expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    permissions JSONB DEFAULT '{
        "view_logs": true,
        "view_appointments": true,
        "view_photos": false,
        "add_notes": false,
        "view_medical": false
    }'::jsonb,
    UNIQUE(pregnancy_id, partner_user_id)
);

-- Baby information (for tracking multiple babies)
CREATE TABLE IF NOT EXISTS babies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pregnancy_id UUID NOT NULL REFERENCES pregnancies(id) ON DELETE CASCADE,
    baby_number INTEGER NOT NULL DEFAULT 1,
    nickname VARCHAR(100),
    gender VARCHAR(50) CHECK (gender IN ('male', 'female', 'unknown', 'not_sharing')),
    name VARCHAR(255),
    birth_date DATE,
    birth_time TIME,
    birth_weight_value DECIMAL(4,2),
    birth_weight_unit VARCHAR(10) DEFAULT 'lbs',
    birth_length_value DECIMAL(4,2),
    birth_length_unit VARCHAR(10) DEFAULT 'inches',
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pregnancy_id, baby_number)
);

-- Search content table (for full-text search)
CREATE TABLE IF NOT EXISTS search_content (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_type VARCHAR(50) NOT NULL,
    week_number INTEGER,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    citations JSONB DEFAULT '[]'::jsonb,
    tags JSONB DEFAULT '[]'::jsonb,
    search_vector tsvector,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create search index
CREATE INDEX IF NOT EXISTS idx_search_content_vector ON search_content USING gin(search_vector);
CREATE INDEX IF NOT EXISTS idx_search_content_week ON search_content(week_number);
CREATE INDEX IF NOT EXISTS idx_search_content_type ON search_content(content_type);

-- Audit log for compliance
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    table_name VARCHAR(100),
    record_id UUID,
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_pregnancies_user_id ON pregnancies(user_id);
CREATE INDEX IF NOT EXISTS idx_daily_logs_pregnancy_id ON daily_logs(pregnancy_id);
CREATE INDEX IF NOT EXISTS idx_daily_logs_date ON daily_logs(log_date);
CREATE INDEX IF NOT EXISTS idx_appointments_pregnancy_id ON appointments(pregnancy_id);
CREATE INDEX IF NOT EXISTS idx_appointments_date ON appointments(appointment_date);
CREATE INDEX IF NOT EXISTS idx_kick_counts_pregnancy_id ON kick_counts(pregnancy_id);
CREATE INDEX IF NOT EXISTS idx_contractions_pregnancy_id ON contractions(pregnancy_id);
CREATE INDEX IF NOT EXISTS idx_emergency_info_user_id ON emergency_info(user_id);
CREATE INDEX IF NOT EXISTS idx_medications_pregnancy_id ON medications(pregnancy_id);
CREATE INDEX IF NOT EXISTS idx_partner_access_pregnancy_id ON partner_access(pregnancy_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at);

-- Update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_pregnancies_updated_at BEFORE UPDATE ON pregnancies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_appointments_updated_at BEFORE UPDATE ON appointments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_emergency_info_updated_at BEFORE UPDATE ON emergency_info
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_search_content_updated_at BEFORE UPDATE ON search_content
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update search vector
CREATE OR REPLACE FUNCTION update_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := to_tsvector('english', 
        COALESCE(NEW.title, '') || ' ' || 
        COALESCE(NEW.content, '') || ' ' ||
        COALESCE(NEW.tags::text, '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_search_vector_trigger
    BEFORE INSERT OR UPDATE ON search_content
    FOR EACH ROW EXECUTE FUNCTION update_search_vector();

-- Row Level Security (RLS) for multi-tenancy
ALTER TABLE pregnancies ENABLE ROW LEVEL SECURITY;
ALTER TABLE daily_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE appointments ENABLE ROW LEVEL SECURITY;
ALTER TABLE kick_counts ENABLE ROW LEVEL SECURITY;
ALTER TABLE contractions ENABLE ROW LEVEL SECURITY;
ALTER TABLE emergency_info ENABLE ROW LEVEL SECURITY;
ALTER TABLE medications ENABLE ROW LEVEL SECURITY;
ALTER TABLE weight_tracking ENABLE ROW LEVEL SECURITY;
ALTER TABLE partner_access ENABLE ROW LEVEL SECURITY;
ALTER TABLE babies ENABLE ROW LEVEL SECURITY;

-- Create RLS policies (these would be created per user session in production)
-- Example policy for pregnancies table
CREATE POLICY pregnancy_isolation ON pregnancies
    FOR ALL
    USING (user_id = current_setting('app.current_user_id', true));

-- Grant permissions
GRANT USAGE ON SCHEMA pregnancy_tracker TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA pregnancy_tracker TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA pregnancy_tracker TO PUBLIC;