-- Seed data for Bedtime Story Generator
-- Initial story themes and sample configuration

-- Insert default story themes
INSERT INTO story_themes (name, description, emoji, color) VALUES
    ('Adventure', 'Exciting journeys and brave explorers', '🗺️', '#FF6B6B'),
    ('Animals', 'Friendly creatures and animal friends', '🦁', '#4ECDC4'),
    ('Fantasy', 'Magic, dragons, and enchanted lands', '🦄', '#9B59B6'),
    ('Space', 'Astronauts and cosmic adventures', '🚀', '#3498DB'),
    ('Ocean', 'Underwater worlds and sea creatures', '🐠', '#00B4D8'),
    ('Forest', 'Woodland tales and nature stories', '🌲', '#27AE60'),
    ('Friendship', 'Stories about making and keeping friends', '🤝', '#F39C12'),
    ('Bedtime', 'Calming stories perfect for sleep', '🌙', '#34495E'),
    ('Dinosaurs', 'Prehistoric adventures with dinosaurs', '🦕', '#E67E22'),
    ('Fairy Tales', 'Classic tales with a modern twist', '👑', '#E91E63')
ON CONFLICT (name) DO NOTHING;

-- Insert default user preferences (for demo/first run)
INSERT INTO user_preferences (
    preferred_age_group,
    preferred_themes,
    preferred_length,
    auto_nightlight_time
) VALUES (
    '6-8',
    ARRAY['Adventure', 'Animals', 'Friendship'],
    'medium',
    '21:00:00'
) ON CONFLICT DO NOTHING;

-- Insert a welcome story
INSERT INTO stories (
    title,
    content,
    age_group,
    theme,
    story_length,
    reading_time_minutes,
    page_count,
    character_names
) VALUES (
    'Welcome to Story Time!',
    E'# Welcome to Your Magical Bookshelf!\n\n## Page 1\n\nHello there, young reader! 📚\n\nThis is your very own magical bookshelf, where wonderful stories come to life just for you.\n\n## Page 2\n\nEvery night, new adventures await you here. You can:\n- Generate brand new stories with the magic story button\n- Save your favorite tales to read again\n- Track your reading adventures\n\n## Page 3\n\nThe room changes with the time of day:\n- ☀️ Bright and sunny during the day\n- 🕯️ Cozy lamp light in the evening  \n- 🌙 Gentle nightlight when it''s bedtime\n\n## Page 4\n\nAre you ready to begin your first adventure?\n\nSweet dreams and happy reading! 🌟',
    '3-5',
    'Bedtime',
    'short',
    3,
    4,
    '[]'::jsonb
) ON CONFLICT DO NOTHING;