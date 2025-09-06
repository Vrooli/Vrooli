-- Home Automation Intelligence - Seed Data
-- Initial data for development and testing

-- Insert demo user profiles
INSERT INTO home_profiles (user_id, name, permissions, preferences) VALUES
    (
        '550e8400-e29b-41d4-a716-446655440001', -- Mock admin user ID
        'Admin User',
        '{
            "devices": ["*"],
            "scenes": ["*"],
            "automation_create": true,
            "admin_access": true
        }'::jsonb,
        '{
            "default_scene": "home",
            "energy_optimization": true,
            "notification_preferences": {
                "device_failures": true,
                "automation_executions": false,
                "energy_alerts": true
            },
            "theme": "dark",
            "dashboard_layout": "grid"
        }'::jsonb
    ),
    (
        '550e8400-e29b-41d4-a716-446655440002', -- Mock regular user ID
        'Family Member',
        '{
            "devices": ["light.living_room", "light.bedroom", "switch.coffee_maker"],
            "scenes": ["morning", "evening", "movie_night"],
            "automation_create": false,
            "admin_access": false
        }'::jsonb,
        '{
            "default_scene": "comfort",
            "energy_optimization": false,
            "notification_preferences": {
                "device_failures": true,
                "automation_executions": true,
                "energy_alerts": false
            }
        }'::jsonb
    ),
    (
        '550e8400-e29b-41d4-a716-446655440003', -- Mock child user ID  
        'Kid Profile',
        '{
            "devices": ["light.bedroom_kid"],
            "scenes": ["bedtime"],
            "automation_create": false,
            "admin_access": false
        }'::jsonb,
        '{
            "default_scene": "play",
            "energy_optimization": false,
            "parental_controls": true,
            "notification_preferences": {
                "device_failures": false,
                "automation_executions": false,
                "energy_alerts": false
            }
        }'::jsonb
    );

-- Insert demo device states (mock Home Assistant devices)
INSERT INTO device_states (device_id, entity_id, name, device_type, state, attributes, available) VALUES
    (
        'light.living_room',
        'light.living_room',
        'Living Room Lights',
        'lights',
        '{"on": true, "brightness": 80, "color_temp": 250}'::jsonb,
        '{"friendly_name": "Living Room Lights", "supported_features": 43, "min_mireds": 153, "max_mireds": 500}'::jsonb,
        true
    ),
    (
        'light.bedroom',
        'light.bedroom', 
        'Bedroom Lights',
        'lights',
        '{"on": false, "brightness": 0}'::jsonb,
        '{"friendly_name": "Bedroom Lights", "supported_features": 41}'::jsonb,
        true
    ),
    (
        'light.bedroom_kid',
        'light.bedroom_kid',
        'Kids Bedroom Light',
        'lights',
        '{"on": false, "brightness": 0, "color": {"r": 255, "g": 180, "b": 50}}'::jsonb,
        '{"friendly_name": "Kids Bedroom Light", "supported_features": 63}'::jsonb,
        true
    ),
    (
        'switch.coffee_maker',
        'switch.coffee_maker',
        'Coffee Maker',
        'switches',
        '{"on": false}'::jsonb,
        '{"friendly_name": "Coffee Maker", "device_class": "outlet"}'::jsonb,
        true
    ),
    (
        'sensor.temperature',
        'sensor.living_room_temperature',
        'Living Room Temperature',
        'sensors',
        '{"temperature": 72.5, "unit": "°F"}'::jsonb,
        '{"friendly_name": "Living Room Temperature", "device_class": "temperature", "unit_of_measurement": "°F"}'::jsonb,
        true
    ),
    (
        'sensor.humidity',
        'sensor.living_room_humidity',
        'Living Room Humidity',
        'sensors', 
        '{"humidity": 45.2, "unit": "%"}'::jsonb,
        '{"friendly_name": "Living Room Humidity", "device_class": "humidity", "unit_of_measurement": "%"}'::jsonb,
        true
    ),
    (
        'climate.thermostat',
        'climate.main_thermostat',
        'Main Thermostat',
        'climate',
        '{"temperature": 72, "target_temp": 72, "mode": "heat", "fan": "auto"}'::jsonb,
        '{"friendly_name": "Main Thermostat", "supported_features": 17, "min_temp": 45, "max_temp": 85}'::jsonb,
        true
    ),
    (
        'lock.front_door',
        'lock.front_door',
        'Front Door Lock',
        'locks',
        '{"locked": true}'::jsonb,
        '{"friendly_name": "Front Door Lock", "device_class": "lock"}'::jsonb,
        true
    );

-- Insert demo smart scenes
INSERT INTO smart_scenes (name, description, created_by, device_states, conditions, icon, usage_count) VALUES
    (
        'Good Morning',
        'Perfect start to your day - gradual lighting and coffee',
        (SELECT id FROM home_profiles WHERE name = 'Admin User'),
        '{
            "light.living_room": {"on": true, "brightness": 60, "color_temp": 250},
            "light.bedroom": {"on": true, "brightness": 30},
            "switch.coffee_maker": {"on": true},
            "climate.thermostat": {"target_temp": 72}
        }'::jsonb,
        '{
            "time_range": {"start": "06:00", "end": "10:00"},
            "days": ["monday", "tuesday", "wednesday", "thursday", "friday"]
        }'::jsonb,
        '🌅',
        15
    ),
    (
        'Movie Night',
        'Dim lights for the perfect movie experience',
        (SELECT id FROM home_profiles WHERE name = 'Admin User'),
        '{
            "light.living_room": {"on": true, "brightness": 10, "color": {"r": 255, "g": 100, "b": 0}},
            "light.bedroom": {"on": false}
        }'::jsonb,
        '{
            "time_range": {"start": "19:00", "end": "23:00"}
        }'::jsonb,
        '🍿',
        8
    ),
    (
        'Bedtime',
        'Wind down with gentle lighting',
        (SELECT id FROM home_profiles WHERE name = 'Admin User'),
        '{
            "light.living_room": {"on": false},
            "light.bedroom": {"on": true, "brightness": 5, "color_temp": 450},
            "light.bedroom_kid": {"on": true, "brightness": 10, "color": {"r": 255, "g": 200, "b": 100}},
            "lock.front_door": {"locked": true}
        }'::jsonb,
        '{
            "time_range": {"start": "21:00", "end": "23:30"}
        }'::jsonb,
        '🌙',
        22
    ),
    (
        'Away Mode',
        'Security mode when nobody is home',
        (SELECT id FROM home_profiles WHERE name = 'Admin User'),
        '{
            "light.living_room": {"on": false},
            "light.bedroom": {"on": false},
            "light.bedroom_kid": {"on": false},
            "switch.coffee_maker": {"on": false},
            "lock.front_door": {"locked": true},
            "climate.thermostat": {"target_temp": 68}
        }'::jsonb,
        '{
            "presence": "away",
            "duration_min": 30
        }'::jsonb,
        '🏠🔒',
        5
    );

-- Insert demo automation rules
INSERT INTO automation_rules (
    name, description, created_by, trigger_type, trigger_config, conditions, actions, generated_by_ai, source_code
) VALUES
    (
        'Morning Coffee Automation',
        'Start coffee when first person wakes up on weekdays',
        (SELECT id FROM home_profiles WHERE name = 'Admin User'),
        'schedule',
        '{
            "time": "06:30",
            "days": ["monday", "tuesday", "wednesday", "thursday", "friday"]
        }'::jsonb,
        '{
            "presence": "home",
            "temperature": {"min": 65}
        }'::jsonb,
        '[
            {
                "device_id": "switch.coffee_maker",
                "action": "turn_on",
                "parameters": {}
            },
            {
                "device_id": "light.living_room", 
                "action": "turn_on",
                "parameters": {"brightness": 60}
            }
        ]'::jsonb,
        false,
        NULL
    ),
    (
        'Energy Saver Evening',
        'Optimize energy usage in the evening',
        (SELECT id FROM home_profiles WHERE name = 'Admin User'),
        'schedule',
        '{
            "time": "22:00"
        }'::jsonb,
        '{
            "presence": "home"
        }'::jsonb,
        '[
            {
                "device_id": "climate.thermostat",
                "action": "set_temperature", 
                "parameters": {"temperature": 68}
            },
            {
                "device_id": "light.living_room",
                "action": "turn_off",
                "parameters": {}
            }
        ]'::jsonb,
        true,
        'AI generated automation based on energy optimization patterns'
    ),
    (
        'Welcome Home',
        'Activate comfort settings when arriving home',
        (SELECT id FROM home_profiles WHERE name = 'Admin User'),
        'device',
        '{
            "device_id": "lock.front_door",
            "state_change": {"from": "locked", "to": "unlocked"}
        }'::jsonb,
        '{
            "time_range": {"start": "16:00", "end": "23:00"}
        }'::jsonb,
        '[
            {
                "device_id": "light.living_room",
                "action": "turn_on", 
                "parameters": {"brightness": 80}
            },
            {
                "device_id": "climate.thermostat",
                "action": "set_temperature",
                "parameters": {"temperature": 72}
            }
        ]'::jsonb,
        false,
        NULL
    );

-- Insert demo calendar contexts
INSERT INTO calendar_contexts (
    calendar_event_id, context_name, scene_id, automation_overrides, 
    start_time, end_time, all_day
) VALUES
    (
        'cal_event_work_from_home_001',
        'work',
        NULL,
        '{
            "climate.thermostat": {"target_temp": 70},
            "light.living_room": {"brightness": 90},
            "automation_pause": ["energy_saver_evening"]
        }'::jsonb,
        '2024-01-22 09:00:00',
        '2024-01-22 17:00:00',
        false
    ),
    (
        'cal_event_movie_night_001',
        'entertainment',
        (SELECT id FROM smart_scenes WHERE name = 'Movie Night'),
        '{}'::jsonb,
        '2024-01-22 20:00:00', 
        '2024-01-22 23:00:00',
        false
    ),
    (
        'cal_event_vacation_001',
        'away',
        (SELECT id FROM smart_scenes WHERE name = 'Away Mode'),
        '{
            "security_mode": true,
            "energy_saving": "maximum"
        }'::jsonb,
        '2024-01-25 00:00:00',
        '2024-01-30 23:59:59',
        true
    );

-- Insert sample energy usage data (last 24 hours)
INSERT INTO energy_usage (device_id, measurement_type, value, unit, timestamp) VALUES
    -- Living room lights
    ('light.living_room', 'power', 45.5, 'W', NOW() - INTERVAL '1 hour'),
    ('light.living_room', 'energy', 0.045, 'kWh', NOW() - INTERVAL '1 hour'),
    ('light.living_room', 'power', 48.2, 'W', NOW() - INTERVAL '2 hours'),
    ('light.living_room', 'energy', 0.048, 'kWh', NOW() - INTERVAL '2 hours'),
    
    -- Coffee maker
    ('switch.coffee_maker', 'power', 1200.0, 'W', NOW() - INTERVAL '12 hours'),
    ('switch.coffee_maker', 'energy', 0.4, 'kWh', NOW() - INTERVAL '12 hours'),
    ('switch.coffee_maker', 'cost', 0.06, 'USD', NOW() - INTERVAL '12 hours'),
    
    -- Thermostat
    ('climate.thermostat', 'power', 2500.0, 'W', NOW() - INTERVAL '3 hours'),
    ('climate.thermostat', 'energy', 2.5, 'kWh', NOW() - INTERVAL '3 hours'),
    ('climate.thermostat', 'cost', 0.38, 'USD', NOW() - INTERVAL '3 hours');

-- Insert sample automation execution logs
INSERT INTO automation_executions (
    automation_id, trigger_source, trigger_data, execution_status,
    devices_affected, execution_time_ms
) VALUES
    (
        (SELECT id FROM automation_rules WHERE name = 'Morning Coffee Automation'),
        'schedule',
        '{"scheduled_time": "06:30", "actual_time": "06:30:02"}'::jsonb,
        'completed',
        '{"switch.coffee_maker", "light.living_room"}',
        1250
    ),
    (
        (SELECT id FROM automation_rules WHERE name = 'Welcome Home'),
        'device',
        '{"device_id": "lock.front_door", "old_state": "locked", "new_state": "unlocked"}'::jsonb,
        'completed',
        '{"light.living_room", "climate.thermostat"}',
        980
    ),
    (
        (SELECT id FROM automation_rules WHERE name = 'Energy Saver Evening'),
        'schedule', 
        '{"scheduled_time": "22:00"}'::jsonb,
        'completed',
        '{"climate.thermostat", "light.living_room"}',
        750
    );

-- Insert sample permission audit entries
INSERT INTO permission_audit_log (
    user_id, profile_id, action_type, resource_type, resource_id,
    permission_granted, additional_data
) VALUES
    (
        '550e8400-e29b-41d4-a716-446655440001',
        (SELECT id FROM home_profiles WHERE name = 'Admin User'),
        'device_control',
        'device',
        'light.living_room',
        true,
        '{"action": "turn_on", "brightness": 80}'::jsonb
    ),
    (
        '550e8400-e29b-41d4-a716-446655440002',
        (SELECT id FROM home_profiles WHERE name = 'Family Member'),
        'device_control',
        'device',
        'switch.coffee_maker',
        true,
        '{"action": "turn_on"}'::jsonb
    ),
    (
        '550e8400-e29b-41d4-a716-446655440003',
        (SELECT id FROM home_profiles WHERE name = 'Kid Profile'),
        'device_control',
        'device',
        'climate.thermostat',
        false,
        '{"action": "set_temperature", "temperature": 75, "reason": "insufficient_permissions"}'::jsonb
    ),
    (
        '550e8400-e29b-41d4-a716-446655440001',
        (SELECT id FROM home_profiles WHERE name = 'Admin User'),
        'automation_create',
        'automation',
        'new_automation_001',
        true,
        '{"generated_by_ai": true, "description": "Turn on porch lights at sunset"}'::jsonb
    );

-- Update last execution times for automation rules
UPDATE automation_rules 
SET last_executed = NOW() - INTERVAL '12 hours',
    execution_count = 1
WHERE name = 'Morning Coffee Automation';

UPDATE automation_rules
SET last_executed = NOW() - INTERVAL '4 hours',
    execution_count = 3  
WHERE name = 'Welcome Home';

UPDATE automation_rules
SET last_executed = NOW() - INTERVAL '2 hours',
    execution_count = 1
WHERE name = 'Energy Saver Evening';

-- Update scene usage timestamps
UPDATE smart_scenes 
SET last_used = NOW() - INTERVAL '12 hours'
WHERE name = 'Good Morning';

UPDATE smart_scenes
SET last_used = NOW() - INTERVAL '2 days'
WHERE name = 'Movie Night';

UPDATE smart_scenes
SET last_used = NOW() - INTERVAL '1 day'
WHERE name = 'Bedtime';

-- Comments for seed data
COMMENT ON TABLE home_profiles IS 'Contains demo user profiles with different permission levels';
COMMENT ON TABLE device_states IS 'Contains mock Home Assistant device states for development';
COMMENT ON TABLE smart_scenes IS 'Contains example scenes for different home contexts';
COMMENT ON TABLE automation_rules IS 'Contains sample automation rules including AI-generated ones';
COMMENT ON TABLE energy_usage IS 'Contains sample energy consumption data for analytics';