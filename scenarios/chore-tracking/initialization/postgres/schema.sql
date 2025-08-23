-- Chore Tracking Schema
-- Gamified household chore management with points and rewards

-- Users/Family members
CREATE TABLE IF NOT EXISTS chore_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    avatar_emoji VARCHAR(10) DEFAULT '🙂',
    total_points INTEGER DEFAULT 0,
    level INTEGER DEFAULT 1,
    streak_days INTEGER DEFAULT 0,
    last_active DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Chore definitions
CREATE TABLE IF NOT EXISTS chores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL, -- kitchen, bathroom, bedroom, outdoor, etc.
    icon_emoji VARCHAR(10) DEFAULT '🧹',
    base_points INTEGER NOT NULL DEFAULT 10,
    difficulty VARCHAR(20) DEFAULT 'medium', -- easy, medium, hard, epic
    frequency VARCHAR(20) DEFAULT 'weekly', -- daily, weekly, biweekly, monthly, as_needed
    estimated_minutes INTEGER DEFAULT 15,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Chore assignments and schedules
CREATE TABLE IF NOT EXISTS chore_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chore_id UUID REFERENCES chores(id) ON DELETE CASCADE,
    user_id UUID REFERENCES chore_users(id) ON DELETE CASCADE,
    assigned_date DATE NOT NULL,
    due_date DATE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    points_earned INTEGER,
    bonus_multiplier DECIMAL(3,2) DEFAULT 1.0, -- for streaks, speed bonuses, etc.
    status VARCHAR(20) DEFAULT 'pending', -- pending, in_progress, completed, overdue, skipped
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Rewards that can be claimed with points
CREATE TABLE IF NOT EXISTS rewards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    cost_points INTEGER NOT NULL,
    category VARCHAR(50), -- treat, privilege, item, experience
    icon_emoji VARCHAR(10) DEFAULT '🎁',
    quantity_available INTEGER DEFAULT -1, -- -1 for unlimited
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Reward claims history
CREATE TABLE IF NOT EXISTS reward_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES chore_users(id) ON DELETE CASCADE,
    reward_id UUID REFERENCES rewards(id) ON DELETE CASCADE,
    points_spent INTEGER NOT NULL,
    claimed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    fulfilled_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'claimed' -- claimed, fulfilled, cancelled
);

-- Achievements/Badges
CREATE TABLE IF NOT EXISTS achievements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    criteria_type VARCHAR(50) NOT NULL, -- streak, total_points, chores_completed, speed, etc.
    criteria_value INTEGER NOT NULL,
    badge_emoji VARCHAR(10) DEFAULT '🏆',
    points_bonus INTEGER DEFAULT 0,
    rarity VARCHAR(20) DEFAULT 'common', -- common, rare, epic, legendary
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User achievements
CREATE TABLE IF NOT EXISTS user_achievements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES chore_users(id) ON DELETE CASCADE,
    achievement_id UUID REFERENCES achievements(id) ON DELETE CASCADE,
    earned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, achievement_id)
);

-- Chore completion history for analytics
CREATE TABLE IF NOT EXISTS chore_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID REFERENCES chore_assignments(id) ON DELETE CASCADE,
    user_id UUID REFERENCES chore_users(id) ON DELETE CASCADE,
    chore_id UUID REFERENCES chores(id) ON DELETE CASCADE,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    time_taken_minutes INTEGER,
    quality_rating INTEGER CHECK (quality_rating >= 1 AND quality_rating <= 5),
    points_earned INTEGER NOT NULL,
    notes TEXT
);

-- Household/Group settings
CREATE TABLE IF NOT EXISTS household_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_name VARCHAR(100) DEFAULT 'Our Home',
    point_multiplier_weekday DECIMAL(3,2) DEFAULT 1.0,
    point_multiplier_weekend DECIMAL(3,2) DEFAULT 1.5,
    streak_bonus_percentage INTEGER DEFAULT 10,
    reminder_time TIME DEFAULT '09:00:00',
    theme_name VARCHAR(50) DEFAULT 'cozy',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_assignments_user_status ON chore_assignments(user_id, status);
CREATE INDEX idx_assignments_due_date ON chore_assignments(due_date);
CREATE INDEX idx_history_user_completed ON chore_history(user_id, completed_at);
CREATE INDEX idx_claims_user_status ON reward_claims(user_id, status);

-- Create update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_chore_users_updated_at BEFORE UPDATE ON chore_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_chores_updated_at BEFORE UPDATE ON chores
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_assignments_updated_at BEFORE UPDATE ON chore_assignments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rewards_updated_at BEFORE UPDATE ON rewards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_settings_updated_at BEFORE UPDATE ON household_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();