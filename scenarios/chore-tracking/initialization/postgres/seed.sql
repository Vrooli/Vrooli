-- Seed data for chore-tracking scenario

-- Insert default chores
INSERT INTO chores (id, name, description, category, frequency, difficulty, estimated_minutes, base_points, icon_emoji) VALUES
('chore-001', 'Wash Dishes', 'Clean all dishes, pots, and pans', 'kitchen', 'daily', 'easy', 15, 5, '🍽️'),
('chore-002', 'Take Out Trash', 'Empty all trash bins and take to curb', 'general', 'twice_weekly', 'easy', 10, 5, '🗑️'),
('chore-003', 'Vacuum Living Room', 'Vacuum carpets and rugs in living area', 'living_room', 'weekly', 'medium', 20, 10, '🏠'),
('chore-004', 'Clean Bathroom', 'Scrub toilet, sink, shower/tub', 'bathroom', 'weekly', 'hard', 45, 20, '🚿'),
('chore-005', 'Do Laundry', 'Wash, dry, and fold one load', 'bedroom', 'twice_weekly', 'medium', 60, 15, '👕'),
('chore-006', 'Mop Floors', 'Mop all hard floor surfaces', 'general', 'weekly', 'medium', 30, 15, '🧹'),
('chore-007', 'Clean Kitchen Counters', 'Wipe down and sanitize all surfaces', 'kitchen', 'daily', 'easy', 10, 5, '🧽'),
('chore-008', 'Water Plants', 'Water all indoor and outdoor plants', 'garden', 'twice_weekly', 'easy', 15, 5, '🌱'),
('chore-009', 'Make Beds', 'Make all beds with fresh sheets if needed', 'bedroom', 'daily', 'easy', 10, 5, '🛏️'),
('chore-010', 'Dust Furniture', 'Dust all surfaces in main living areas', 'living_room', 'weekly', 'medium', 25, 10, '✨'),
('chore-011', 'Clean Windows', 'Clean windows inside and out', 'general', 'monthly', 'hard', 60, 25, '🪟'),
('chore-012', 'Organize Closet', 'Sort and organize clothing and items', 'bedroom', 'monthly', 'hard', 90, 30, '👔'),
('chore-013', 'Deep Clean Fridge', 'Empty, clean, and reorganize refrigerator', 'kitchen', 'monthly', 'hard', 60, 25, '❄️'),
('chore-014', 'Sweep Garage', 'Sweep and organize garage space', 'garage', 'monthly', 'medium', 45, 20, '🚗'),
('chore-015', 'Feed Pets', 'Feed and refresh water for pets', 'pets', 'daily', 'easy', 5, 3, '🐕');

-- Insert default rewards
INSERT INTO rewards (id, name, description, cost_points, category, icon_emoji, is_active) VALUES
('reward-001', 'Movie Night', 'Pick any movie to watch', 50, 'entertainment', '🎬', true),
('reward-002', 'Extra Screen Time', '30 minutes additional screen time', 25, 'entertainment', '📱', true),
('reward-003', 'Favorite Dessert', 'Choose a special dessert', 40, 'food', '🍰', true),
('reward-004', 'Stay Up Late', 'Stay up 1 hour past bedtime', 60, 'privilege', '🌙', true),
('reward-005', 'Skip a Chore', 'Skip one assigned chore', 75, 'privilege', '🎫', true),
('reward-006', 'Pizza Night', 'Order pizza for dinner', 100, 'food', '🍕', true),
('reward-007', 'Video Game Time', '1 hour of gaming', 30, 'entertainment', '🎮', true),
('reward-008', 'Friend Sleepover', 'Have a friend stay over', 150, 'social', '🏠', true),
('reward-009', 'Choose Weekend Activity', 'Pick family weekend activity', 120, 'privilege', '🎯', true),
('reward-010', 'Ice Cream Trip', 'Go out for ice cream', 45, 'food', '🍦', true);

-- Insert achievements
INSERT INTO achievements (id, name, description, criteria_type, criteria_value, badge_emoji, points_bonus, rarity) VALUES
('ach-001', 'First Steps', 'Complete your first chore', 'chores_completed', 1, '👶', 10, 'common'),
('ach-002', 'Week Warrior', 'Complete chores 7 days in a row', 'streak', 7, '⚔️', 50, 'rare'),
('ach-003', 'Monthly Master', 'Complete chores 30 days in a row', 'streak', 30, '👑', 200, 'legendary'),
('ach-004', 'Century Cleaner', 'Complete 100 total chores', 'chores_completed', 100, '💯', 100, 'epic'),
('ach-005', 'Speed Demon', 'Complete 5 chores in one day', 'daily_chores', 5, '⚡', 25, 'rare'),
('ach-006', 'Point Collector', 'Earn 500 total points', 'total_points', 500, '💎', 40, 'rare'),
('ach-007', 'Early Bird', 'Complete a chore before 8 AM', 'time_based', 8, '🐦', 30, 'common'),
('ach-008', 'Night Owl', 'Complete a chore after 9 PM', 'time_based', 21, '🦉', 30, 'common'),
('ach-009', 'Kitchen Master', 'Complete 20 kitchen chores', 'category_chores', 20, '👨‍🍳', 35, 'rare'),
('ach-010', 'Level 10 Hero', 'Reach level 10', 'level', 10, '🎖️', 100, 'epic');

-- Insert sample users (for testing)
INSERT INTO chore_users (id, username, display_name, avatar_emoji, total_points, level, streak_days) VALUES
('11111111-1111-1111-1111-111111111111', 'demo_user', 'Demo Hero', '🦸', 0, 1, 0);

-- Insert initial household settings
INSERT INTO household_settings (household_name, theme_name) VALUES
('ChoreQuest House', 'cozy');