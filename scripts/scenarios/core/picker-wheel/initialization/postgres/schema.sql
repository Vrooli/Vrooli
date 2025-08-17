-- Picker Wheel Database Schema
-- Fun random selection wheel with saved configurations

CREATE TABLE IF NOT EXISTS wheels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    options JSONB NOT NULL, -- Array of {label, color, weight, icon}
    theme VARCHAR(50) DEFAULT 'neon',
    sound_enabled BOOLEAN DEFAULT true,
    confetti_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    times_used INTEGER DEFAULT 0,
    is_public BOOLEAN DEFAULT false,
    session_id VARCHAR(255),
    tags TEXT[]
);

CREATE TABLE IF NOT EXISTS spin_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wheel_id UUID REFERENCES wheels(id) ON DELETE CASCADE,
    result VARCHAR(255) NOT NULL,
    spin_duration_ms INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    session_id VARCHAR(255),
    metadata JSONB -- Additional context like fairness score, angle, etc
);

CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) UNIQUE NOT NULL,
    default_theme VARCHAR(50) DEFAULT 'neon',
    sound_volume DECIMAL(3,2) DEFAULT 0.5,
    animation_speed VARCHAR(20) DEFAULT 'normal',
    favorite_wheels UUID[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_wheels_public ON wheels(is_public);
CREATE INDEX idx_wheels_tags ON wheels USING GIN(tags);
CREATE INDEX idx_history_wheel ON spin_history(wheel_id);
CREATE INDEX idx_history_session ON spin_history(session_id);
CREATE INDEX idx_history_timestamp ON spin_history(timestamp DESC);

-- Sample data for fun preset wheels
INSERT INTO wheels (name, description, options, theme, tags, is_public) VALUES
(
    'Dinner Decider',
    'Can''t decide what to eat? Let the wheel choose!',
    '[
        {"label": "Pizza 🍕", "color": "#FF6B6B", "weight": 1},
        {"label": "Sushi 🍱", "color": "#4ECDC4", "weight": 1},
        {"label": "Tacos 🌮", "color": "#FFD93D", "weight": 1},
        {"label": "Burger 🍔", "color": "#6C5CE7", "weight": 1},
        {"label": "Pasta 🍝", "color": "#A8E6CF", "weight": 1},
        {"label": "Salad 🥗", "color": "#95E77E", "weight": 0.5},
        {"label": "Surprise Me! 🎲", "color": "#FF1744", "weight": 0.5}
    ]'::jsonb,
    'food',
    ARRAY['food', 'decision', 'dinner'],
    true
),
(
    'Yes or No',
    'Simple decision maker',
    '[
        {"label": "YES! ✅", "color": "#4CAF50", "weight": 1},
        {"label": "NO ❌", "color": "#F44336", "weight": 1},
        {"label": "Maybe? 🤔", "color": "#FFC107", "weight": 0.2}
    ]'::jsonb,
    'minimal',
    ARRAY['decision', 'simple', 'binary'],
    true
),
(
    'Team Picker',
    'Randomly select team members',
    '[
        {"label": "Team Alpha", "color": "#E91E63", "weight": 1},
        {"label": "Team Beta", "color": "#9C27B0", "weight": 1},
        {"label": "Team Gamma", "color": "#3F51B5", "weight": 1},
        {"label": "Team Delta", "color": "#2196F3", "weight": 1}
    ]'::jsonb,
    'corporate',
    ARRAY['teams', 'work', 'selection'],
    true
),
(
    'D20 Roll',
    'Digital 20-sided die',
    '[
        {"label": "1", "color": "#B71C1C", "weight": 1},
        {"label": "2", "color": "#880E4F", "weight": 1},
        {"label": "3", "color": "#4A148C", "weight": 1},
        {"label": "4", "color": "#311B92", "weight": 1},
        {"label": "5", "color": "#1A237E", "weight": 1},
        {"label": "6", "color": "#0D47A1", "weight": 1},
        {"label": "7", "color": "#01579B", "weight": 1},
        {"label": "8", "color": "#006064", "weight": 1},
        {"label": "9", "color": "#004D40", "weight": 1},
        {"label": "10", "color": "#1B5E20", "weight": 1},
        {"label": "11", "color": "#33691E", "weight": 1},
        {"label": "12", "color": "#827717", "weight": 1},
        {"label": "13", "color": "#F57F17", "weight": 1},
        {"label": "14", "color": "#FF6F00", "weight": 1},
        {"label": "15", "color": "#E65100", "weight": 1},
        {"label": "16", "color": "#BF360C", "weight": 1},
        {"label": "17", "color": "#3E2723", "weight": 1},
        {"label": "18", "color": "#212121", "weight": 1},
        {"label": "19", "color": "#263238", "weight": 1},
        {"label": "20 🎯", "color": "#FFD700", "weight": 1}
    ]'::jsonb,
    'gaming',
    ARRAY['gaming', 'dice', 'd20', 'rpg'],
    true
);