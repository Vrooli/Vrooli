-- Contact Book Database Schema
-- This schema implements the social intelligence engine design
-- with graph-based relationships, time-bounded facts, and consent management

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable PostGIS for geo-coding addresses (optional)
-- CREATE EXTENSION IF NOT EXISTS postgis;

-- =============================================================================
-- CORE ENTITIES
-- =============================================================================

-- Persons: Core contact information with rich normalized data
CREATE TABLE persons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Basic identity
    full_name TEXT NOT NULL,
    display_name TEXT, -- How they prefer to be called
    nicknames TEXT[], -- Array of nicknames
    pronouns TEXT, -- e.g., "they/them", "she/her"
    
    -- Contact information
    emails TEXT[], -- Array of email addresses
    phones TEXT[], -- Array of phone numbers
    
    -- Scenario integration
    scenario_authenticator_id UUID, -- Link to scenario-authenticator profile
    
    -- Rich metadata (stored as JSONB for flexibility)
    metadata JSONB DEFAULT '{}', -- Stores birthday, timezone, dietary restrictions, etc.
    
    -- Tags for flexible categorization
    tags TEXT[],
    
    -- Free-form notes
    notes TEXT,
    
    -- Social media and external profiles
    social_profiles JSONB DEFAULT '{}', -- URLs to social media, professional profiles
    
    -- Computed signals (updated by batch processing)
    computed_signals JSONB DEFAULT '{}', -- Closeness scores, affinity vectors, etc.
    
    -- Soft delete
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Addresses: Time-bounded address information with geo-coding
CREATE TABLE addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Time bounds for this address
    valid_from DATE NOT NULL,
    valid_to DATE, -- NULL means current
    
    -- Address details
    address_type TEXT NOT NULL, -- 'home', 'work', 'mailing', 'temporary'
    street_address TEXT NOT NULL,
    city TEXT NOT NULL,
    state_province TEXT,
    postal_code TEXT,
    country TEXT NOT NULL DEFAULT 'US',
    
    -- Geo-coding (for clustering and travel suggestions)
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    timezone TEXT, -- e.g., 'America/New_York'
    
    -- Household clustering (derived)
    household_id UUID, -- Computed field for people sharing addresses
    
    -- Metadata
    notes TEXT,
    is_primary BOOLEAN DEFAULT false
);

-- Organizations: Companies, schools, clubs, teams
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Basic info
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- 'employer', 'school', 'club', 'team', 'family'
    industry TEXT,
    website TEXT,
    
    -- Contact information
    address_id UUID REFERENCES addresses(id),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    notes TEXT,
    tags TEXT[]
);

-- Person-Organization relationships with time bounds
CREATE TABLE person_organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Time bounds
    valid_from DATE NOT NULL,
    valid_to DATE, -- NULL means current
    
    -- Role and relationship details
    role TEXT, -- 'employee', 'student', 'member', 'volunteer'
    title TEXT, -- Job title, degree, position
    department TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    notes TEXT
);

-- =============================================================================
-- RELATIONSHIP GRAPH
-- =============================================================================

-- Relationships: Graph edges between people
CREATE TABLE relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Graph edge
    from_person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    to_person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    
    -- Relationship details
    relationship_type TEXT NOT NULL, -- 'friend', 'family', 'sibling', 'coworker', 'ex-coworker'
    strength DECIMAL(3, 2) CHECK (strength >= 0 AND strength <= 1), -- 0.0 to 1.0
    recency_score DECIMAL(3, 2) CHECK (recency_score >= 0 AND recency_score <= 1), -- Computed from last_contact
    last_contact_date DATE,
    
    -- Introduction tracking
    introduced_by_person_id UUID REFERENCES persons(id),
    introduction_date DATE,
    introduction_context TEXT,
    
    -- Shared interests and affinity
    shared_interests TEXT[], -- Common interest tags
    affinity_score DECIMAL(3, 2), -- Computed similarity score
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    notes TEXT,
    
    -- Constraints
    CONSTRAINT no_self_relationship CHECK (from_person_id != to_person_id),
    UNIQUE(from_person_id, to_person_id)
);

-- Household clusters (computed from shared addresses)
CREATE TABLE households (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Household details
    name TEXT, -- Optional household name
    primary_address_id UUID REFERENCES addresses(id),
    
    -- Computed membership (updated by batch processing)
    member_count INTEGER DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'
);

-- =============================================================================
-- COMMUNICATION HISTORY (METADATA ONLY)
-- =============================================================================

-- Communication history: Summaries only, never content
CREATE TABLE communication_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Participants
    from_person_id UUID REFERENCES persons(id),
    to_person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    
    -- Communication metadata
    channel TEXT NOT NULL, -- 'email', 'sms', 'call', 'in-person', 'social'
    direction TEXT NOT NULL, -- 'sent', 'received', 'bidirectional'
    communication_date TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Tone and style analysis (privacy-respecting)
    tone TEXT, -- 'formal', 'casual', 'friendly', 'business'
    response_latency_hours INTEGER, -- How long to respond
    message_length TEXT, -- 'short', 'medium', 'long'
    
    -- Context
    subject_category TEXT, -- 'work', 'personal', 'event-planning', 'social'
    
    -- Metadata
    metadata JSONB DEFAULT '{}'
);

-- Communication preferences (learned from history)
CREATE TABLE communication_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Preferred communication patterns
    preferred_channels TEXT[], -- ['email', 'sms'] in order of preference
    preferred_tone TEXT, -- 'formal', 'casual', 'friendly'
    typical_response_latency_hours INTEGER,
    preferred_message_length TEXT, -- 'short', 'medium', 'long'
    
    -- Timing preferences
    best_contact_times JSONB, -- Time windows when they typically respond
    timezone TEXT,
    
    -- Computed from communication history
    response_reliability_score DECIMAL(3, 2), -- How reliably they respond
    
    -- Metadata
    metadata JSONB DEFAULT '{}'
);

-- =============================================================================
-- EVENTS AND SOCIAL CONTEXT
-- =============================================================================

-- Events: Parties, meetings, gatherings for relationship context
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Event details
    name TEXT NOT NULL,
    event_type TEXT, -- 'wedding', 'birthday', 'business-meeting', 'social'
    event_date DATE,
    location TEXT,
    
    -- Event metadata
    metadata JSONB DEFAULT '{}',
    notes TEXT
);

-- Event participation and RSVP tracking
CREATE TABLE event_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- RSVP status
    rsvp_status TEXT, -- 'invited', 'attending', 'declined', 'maybe', 'no-response'
    response_date TIMESTAMP WITH TIME ZONE,
    
    -- Event-specific role
    role TEXT, -- 'guest', 'speaker', 'organizer', 'vip'
    
    -- Seating and logistics (for wedding-planner integration)
    table_assignment TEXT,
    dietary_restrictions TEXT[],
    accessibility_needs TEXT[],
    plus_one_count INTEGER DEFAULT 0,
    travel_info JSONB, -- Flight details, hotel, etc.
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    notes TEXT,
    
    UNIQUE(event_id, person_id)
);

-- =============================================================================
-- PREFERENCES AND INTERESTS
-- =============================================================================

-- Personal preferences and interests
CREATE TABLE person_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Dietary and lifestyle
    dietary_restrictions TEXT[], -- 'vegan', 'vegetarian', 'gluten-free', 'kosher', etc.
    allergies TEXT[],
    accessibility_needs TEXT[],
    
    -- Personal preferences
    interests TEXT[], -- Hobbies, topics they enjoy
    music_preferences TEXT[],
    alcohol_preference TEXT, -- 'drinks', 'non-drinker', 'occasionally'
    gift_preferences TEXT[], -- What they typically like as gifts
    
    -- Religion and culture (for respectful interaction)
    religious_considerations TEXT[],
    cultural_considerations TEXT[],
    
    -- Metadata
    metadata JSONB DEFAULT '{}'
);

-- =============================================================================
-- PRIVACY AND CONSENT MANAGEMENT
-- =============================================================================

-- Consent management: What can be stored and used per person
CREATE TABLE consent_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Consent scopes
    scope TEXT NOT NULL, -- 'dietary', 'accessibility', 'calendar', 'communication_analysis', 'photos'
    
    -- Consent details
    granted BOOLEAN NOT NULL,
    granted_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE, -- Optional expiration
    
    -- Context
    granted_by TEXT, -- 'self', 'implied', 'third-party'
    granularity TEXT, -- 'full', 'limited', 'metadata-only'
    
    -- Metadata
    notes TEXT,
    metadata JSONB DEFAULT '{}',
    
    UNIQUE(person_id, scope)
);

-- =============================================================================
-- COMPUTED SIGNALS AND ANALYTICS
-- =============================================================================

-- Computed closeness scores and social metrics (updated nightly)
CREATE TABLE social_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    computed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Closeness and centrality metrics
    overall_closeness_score DECIMAL(5, 4), -- 0.0000 to 1.0000
    social_centrality_score DECIMAL(5, 4), -- How connected they are in the graph
    mutual_connection_count INTEGER,
    
    -- Communication patterns
    avg_response_latency_hours INTEGER,
    communication_frequency_score DECIMAL(3, 2),
    last_interaction_days INTEGER,
    
    -- Interest similarity (TF-IDF based)
    top_shared_interests TEXT[],
    affinity_vector JSONB, -- Vector representation for similarity calculations
    
    -- Priority for relationship maintenance
    maintenance_priority_score DECIMAL(3, 2), -- Higher = needs attention
    
    -- Metadata
    computation_version TEXT, -- Track algorithm versions
    metadata JSONB DEFAULT '{}'
);

-- =============================================================================
-- INDEXES FOR PERFORMANCE
-- =============================================================================

-- Core lookups
CREATE INDEX idx_persons_emails ON persons USING GIN (emails);
CREATE INDEX idx_persons_phones ON persons USING GIN (phones);
CREATE INDEX idx_persons_scenario_auth ON persons (scenario_authenticator_id) WHERE scenario_authenticator_id IS NOT NULL;
CREATE INDEX idx_persons_tags ON persons USING GIN (tags);
CREATE INDEX idx_persons_deleted ON persons (deleted_at) WHERE deleted_at IS NULL;

-- Relationships and graph queries
CREATE INDEX idx_relationships_from ON relationships (from_person_id);
CREATE INDEX idx_relationships_to ON relationships (to_person_id);
CREATE INDEX idx_relationships_type ON relationships (relationship_type);
CREATE INDEX idx_relationships_strength ON relationships (strength) WHERE strength IS NOT NULL;
CREATE INDEX idx_relationships_recency ON relationships (recency_score) WHERE recency_score IS NOT NULL;
CREATE INDEX idx_relationships_last_contact ON relationships (last_contact_date) WHERE last_contact_date IS NOT NULL;

-- Time-bounded data
CREATE INDEX idx_addresses_valid ON addresses (valid_from, valid_to) WHERE valid_to IS NULL OR valid_to >= CURRENT_DATE;
CREATE INDEX idx_person_orgs_valid ON person_organizations (valid_from, valid_to) WHERE valid_to IS NULL OR valid_to >= CURRENT_DATE;

-- Communication patterns
CREATE INDEX idx_comm_history_person ON communication_history (to_person_id, communication_date DESC);
CREATE INDEX idx_comm_history_channel ON communication_history (channel, communication_date DESC);

-- Events and social context
CREATE INDEX idx_event_participants_person ON event_participants (person_id);
CREATE INDEX idx_event_participants_event ON event_participants (event_id);
CREATE INDEX idx_events_date ON events (event_date) WHERE event_date IS NOT NULL;

-- Analytics and computed data
CREATE INDEX idx_social_analytics_person ON social_analytics (person_id);
CREATE INDEX idx_social_analytics_closeness ON social_analytics (overall_closeness_score DESC) WHERE overall_closeness_score IS NOT NULL;
CREATE INDEX idx_social_analytics_maintenance ON social_analytics (maintenance_priority_score DESC) WHERE maintenance_priority_score IS NOT NULL;

-- Consent management
CREATE INDEX idx_consent_person_scope ON consent_records (person_id, scope);
CREATE INDEX idx_consent_active ON consent_records (person_id, scope) WHERE granted = true AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP);

-- =============================================================================
-- TRIGGERS FOR AUTOMATIC UPDATES
-- =============================================================================

-- Update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_persons_updated_at BEFORE UPDATE ON persons
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_relationships_updated_at BEFORE UPDATE ON relationships
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- VIEWS FOR COMMON QUERIES
-- =============================================================================

-- Active relationships with computed recency
CREATE VIEW active_relationships AS
SELECT 
    r.*,
    CASE 
        WHEN r.last_contact_date IS NULL THEN 0.0
        ELSE GREATEST(0.0, 1.0 - (CURRENT_DATE - r.last_contact_date)::FLOAT / 365.0)
    END AS computed_recency_score
FROM relationships r
WHERE r.strength > 0;

-- Current addresses (valid now)
CREATE VIEW current_addresses AS
SELECT *
FROM addresses
WHERE valid_from <= CURRENT_DATE 
  AND (valid_to IS NULL OR valid_to >= CURRENT_DATE);

-- Current organization memberships
CREATE VIEW current_person_organizations AS
SELECT po.*, o.name as organization_name, o.type as organization_type
FROM person_organizations po
JOIN organizations o ON po.organization_id = o.id
WHERE po.valid_from <= CURRENT_DATE 
  AND (po.valid_to IS NULL OR po.valid_to >= CURRENT_DATE);

-- Person summary with computed signals
CREATE VIEW person_summary AS
SELECT 
    p.*,
    sa.overall_closeness_score,
    sa.maintenance_priority_score,
    sa.last_interaction_days,
    COALESCE(ca.street_address || ', ' || ca.city, 'No address') as current_address
FROM persons p
LEFT JOIN social_analytics sa ON p.id = sa.person_id
LEFT JOIN current_addresses ca ON p.id = ca.person_id AND ca.is_primary = true
WHERE p.deleted_at IS NULL;

-- =============================================================================
-- COMMENTS FOR DOCUMENTATION
-- =============================================================================

COMMENT ON TABLE persons IS 'Core contact information with rich normalized data for social intelligence';
COMMENT ON TABLE relationships IS 'Graph edges representing social connections with strength and recency metrics';
COMMENT ON TABLE organizations IS 'Companies, schools, clubs, and other groups that provide social context';
COMMENT ON TABLE communication_history IS 'Privacy-respecting metadata about communication patterns (never stores content)';
COMMENT ON TABLE consent_records IS 'Privacy-first consent management for what data can be stored and used';
COMMENT ON TABLE social_analytics IS 'Computed social intelligence metrics updated by nightly batch processing';
COMMENT ON COLUMN relationships.strength IS 'Relationship strength from 0.0 (weak) to 1.0 (very strong)';
COMMENT ON COLUMN relationships.recency_score IS 'Computed decay function based on last_contact_date';
COMMENT ON COLUMN social_analytics.affinity_vector IS 'TF-IDF vector for interest-based similarity calculations';
COMMENT ON COLUMN social_analytics.maintenance_priority_score IS 'Priority score for relationship maintenance (higher = needs attention)';