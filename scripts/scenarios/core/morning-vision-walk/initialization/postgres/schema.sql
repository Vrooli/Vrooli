-- Morning Vision Walk Database Schema
-- Stores conversation history, insights, and daily planning data

CREATE SCHEMA IF NOT EXISTS vision_walk;

-- Users table
CREATE TABLE IF NOT EXISTS vision_walk.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Conversation sessions
CREATE TABLE IF NOT EXISTS vision_walk.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES vision_walk.users(id) ON DELETE CASCADE,
    title VARCHAR(500),
    date DATE DEFAULT CURRENT_DATE,
    mood VARCHAR(50),
    energy_level INTEGER CHECK (energy_level >= 1 AND energy_level <= 10),
    weather JSONB,
    location JSONB,
    duration_minutes INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Conversation messages
CREATE TABLE IF NOT EXISTS vision_walk.messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES vision_walk.sessions(id) ON DELETE CASCADE,
    role VARCHAR(20) CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    audio_url TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Daily insights generated from conversations
CREATE TABLE IF NOT EXISTS vision_walk.insights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES vision_walk.sessions(id) ON DELETE CASCADE,
    user_id UUID REFERENCES vision_walk.users(id) ON DELETE CASCADE,
    type VARCHAR(50), -- 'breakthrough', 'pattern', 'concern', 'opportunity'
    title VARCHAR(500),
    content TEXT,
    importance INTEGER CHECK (importance >= 1 AND importance <= 5),
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Daily plans and tasks
CREATE TABLE IF NOT EXISTS vision_walk.daily_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES vision_walk.sessions(id) ON DELETE CASCADE,
    user_id UUID REFERENCES vision_walk.users(id) ON DELETE CASCADE,
    date DATE DEFAULT CURRENT_DATE,
    focus_areas TEXT[],
    main_goal TEXT,
    energy_allocation JSONB, -- How to distribute energy throughout the day
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tasks extracted from conversations
CREATE TABLE IF NOT EXISTS vision_walk.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    daily_plan_id UUID REFERENCES vision_walk.daily_plans(id) ON DELETE CASCADE,
    session_id UUID REFERENCES vision_walk.sessions(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    priority VARCHAR(20) CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    category VARCHAR(100),
    estimated_time_minutes INTEGER,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'deferred')),
    completed_at TIMESTAMP,
    notes TEXT,
    external_ref JSONB, -- References to other systems like task-planner
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Vrooli system state snapshots
CREATE TABLE IF NOT EXISTS vision_walk.system_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES vision_walk.sessions(id) ON DELETE CASCADE,
    snapshot_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    active_scenarios INTEGER,
    resource_status JSONB,
    recent_achievements JSONB,
    current_challenges JSONB,
    metrics JSONB
);

-- Recurring themes and patterns
CREATE TABLE IF NOT EXISTS vision_walk.themes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES vision_walk.users(id) ON DELETE CASCADE,
    theme VARCHAR(500),
    description TEXT,
    frequency INTEGER DEFAULT 1,
    first_mentioned TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_mentioned TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    related_insights UUID[]
);

-- Integration with other scenarios
CREATE TABLE IF NOT EXISTS vision_walk.integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES vision_walk.sessions(id) ON DELETE CASCADE,
    scenario_name VARCHAR(100),
    action_type VARCHAR(50), -- 'query', 'create', 'update'
    payload JSONB,
    response JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_sessions_user_date ON vision_walk.sessions(user_id, date);
CREATE INDEX idx_messages_session ON vision_walk.messages(session_id);
CREATE INDEX idx_insights_user ON vision_walk.insights(user_id);
CREATE INDEX idx_insights_type ON vision_walk.insights(type);
CREATE INDEX idx_daily_plans_date ON vision_walk.daily_plans(date);
CREATE INDEX idx_tasks_status ON vision_walk.tasks(status);
CREATE INDEX idx_tasks_priority ON vision_walk.tasks(priority);
CREATE INDEX idx_themes_user ON vision_walk.themes(user_id);

-- Create updated_at trigger
CREATE OR REPLACE FUNCTION vision_walk.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON vision_walk.users
    FOR EACH ROW EXECUTE FUNCTION vision_walk.update_updated_at_column();

CREATE TRIGGER update_tasks_updated_at BEFORE UPDATE ON vision_walk.tasks
    FOR EACH ROW EXECUTE FUNCTION vision_walk.update_updated_at_column();