-- Period Tracker Seed Data
-- Sample data for development and testing (no real personal information)

-- Insert sample users (for multi-tenant testing)
-- Note: In production, users are created through the authentication system
INSERT INTO users (id, username, email_encrypted, password_hash, timezone, preferences) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'demo_user_1', 'encrypted_email_1', '$2a$10$dummy_hash_for_demo_user_1', 'America/New_York', 
     '{"theme": "light", "notifications": true, "prediction_days": 90}'),
    ('550e8400-e29b-41d4-a716-446655440002', 'demo_user_2', 'encrypted_email_2', '$2a$10$dummy_hash_for_demo_user_2', 'Europe/London',
     '{"theme": "dark", "notifications": false, "prediction_days": 60}'),
    ('550e8400-e29b-41d4-a716-446655440003', 'demo_user_3', 'encrypted_email_3', '$2a$10$dummy_hash_for_demo_user_3', 'Asia/Tokyo',
     '{"theme": "auto", "notifications": true, "prediction_days": 120}')
ON CONFLICT (id) DO NOTHING;

-- Sample historical cycles for demo_user_1 (for testing prediction algorithms)
INSERT INTO cycles (id, user_id, start_date, end_date, cycle_length, flow_intensity, is_predicted) VALUES
    ('660e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', '2024-01-01', '2024-01-05', 28, 'medium', false),
    ('660e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', '2024-01-29', '2024-02-02', 30, 'heavy', false),
    ('660e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440001', '2024-02-28', '2024-03-04', 27, 'medium', false),
    ('660e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440001', '2024-03-26', '2024-03-30', 29, 'light', false),
    ('660e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440001', '2024-04-24', '2024-04-28', 28, 'medium', false)
ON CONFLICT (id) DO NOTHING;

-- Sample daily symptoms for demo_user_1 (for testing pattern recognition)
INSERT INTO daily_symptoms (user_id, symptom_date, mood_rating, energy_level, stress_level, cramp_intensity, flow_level) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', '2024-04-24', 6, 7, 4, 5, 'medium'),
    ('550e8400-e29b-41d4-a716-446655440001', '2024-04-25', 5, 6, 5, 7, 'heavy'),
    ('550e8400-e29b-41d4-a716-446655440001', '2024-04-26', 4, 5, 6, 8, 'heavy'),
    ('550e8400-e29b-41d4-a716-446655440001', '2024-04-27', 5, 6, 4, 4, 'medium'),
    ('550e8400-e29b-41d4-a716-446655440001', '2024-04-28', 6, 7, 3, 2, 'light'),
    ('550e8400-e29b-41d4-a716-446655440001', '2024-04-29', 7, 8, 2, 0, 'none'),
    ('550e8400-e29b-41d4-a716-446655440001', '2024-04-30', 8, 8, 2, 0, 'none')
ON CONFLICT (user_id, symptom_date) DO NOTHING;

-- Sample predictions for demo_user_1
INSERT INTO predictions (user_id, predicted_start_date, predicted_end_date, predicted_length, confidence_score, algorithm_version) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', '2024-05-22', '2024-05-26', 28, 0.85, 'v1.0.0'),
    ('550e8400-e29b-41d4-a716-446655440001', '2024-06-19', '2024-06-23', 28, 0.82, 'v1.0.0'),
    ('550e8400-e29b-41d4-a716-446655440001', '2024-07-17', '2024-07-21', 28, 0.78, 'v1.0.0')
ON CONFLICT (id) DO NOTHING;

-- Sample detected patterns for demo_user_1
INSERT INTO detected_patterns (user_id, pattern_type, pattern_description_encrypted, correlation_strength, confidence_level, algorithm_version) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'mood_dip_day2', 'encrypted_pattern_desc_1', -0.73, 'high', 'v1.0.0'),
    ('550e8400-e29b-41d4-a716-446655440001', 'energy_low_pms', 'encrypted_pattern_desc_2', -0.68, 'medium', 'v1.0.0'),
    ('550e8400-e29b-41d4-a716-446655440001', 'cramps_peak_day2', 'encrypted_pattern_desc_3', 0.81, 'high', 'v1.0.0')
ON CONFLICT (id) DO NOTHING;

-- Sample medications for demo_user_2
INSERT INTO medications (user_id, name_encrypted, dosage_encrypted, frequency, is_birth_control, reminder_times) VALUES
    ('550e8400-e29b-41d4-a716-446655440002', 'encrypted_med_name_1', 'encrypted_dosage_1', 'daily', true, '["08:00", "20:00"]'),
    ('550e8400-e29b-41d4-a716-446655440002', 'encrypted_med_name_2', 'encrypted_dosage_2', 'as_needed', false, '[]')
ON CONFLICT (id) DO NOTHING;

-- Sample audit logs (for testing privacy compliance features)
INSERT INTO audit_logs (user_id, action, table_name, ip_address, details) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'data_accessed', 'cycles', '127.0.0.1', '{"endpoint": "/api/v1/cycles", "method": "GET"}'),
    ('550e8400-e29b-41d4-a716-446655440001', 'data_modified', 'daily_symptoms', '127.0.0.1', '{"endpoint": "/api/v1/symptoms", "method": "POST"}'),
    ('550e8400-e29b-41d4-a716-446655440002', 'data_accessed', 'predictions', '127.0.0.1', '{"endpoint": "/api/v1/predictions", "method": "GET"}')
ON CONFLICT (id) DO NOTHING;

-- Update statistics to ensure good query performance
ANALYZE users;
ANALYZE cycles;
ANALYZE daily_symptoms;
ANALYZE predictions;
ANALYZE detected_patterns;
ANALYZE medications;
ANALYZE audit_logs;

-- Insert some helpful metadata
INSERT INTO detected_patterns (user_id, pattern_type, pattern_description_encrypted, correlation_strength, confidence_level, algorithm_version, is_active) VALUES
    ('550e8400-e29b-41d4-a716-446655440003', 'system_info', 'encrypted_system_ready_message', 1.0, 'high', 'v1.0.0', true)
ON CONFLICT (id) DO NOTHING;

-- Print setup confirmation
DO $$
BEGIN
    RAISE NOTICE 'Period Tracker seed data loaded successfully';
    RAISE NOTICE 'Sample users created: 3';
    RAISE NOTICE 'Sample cycles created: 5 for demo_user_1';
    RAISE NOTICE 'Sample symptoms created: 7 days for demo_user_1';
    RAISE NOTICE 'Sample predictions created: 3 for demo_user_1';
    RAISE NOTICE 'Sample patterns detected: 3 for demo_user_1';
    RAISE NOTICE 'Sample medications created: 2 for demo_user_2';
    RAISE NOTICE 'Privacy: All sensitive data is encrypted at application layer';
END $$;