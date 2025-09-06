-- Contact Book Seed Data
-- Sample data demonstrating the social intelligence engine capabilities

-- =============================================================================
-- SAMPLE PERSONS
-- =============================================================================

INSERT INTO persons (id, full_name, display_name, emails, phones, pronouns, metadata, tags, notes, social_profiles) VALUES
(
    uuid_generate_v4(),
    'Sarah Chen',
    'Sarah',
    ARRAY['sarah.chen@techcorp.com', 'sarahc.personal@gmail.com'],
    ARRAY['+1-555-0123', '+1-555-9876'],
    'she/her',
    '{
        "birthday": "1990-03-15",
        "timezone": "America/New_York",
        "dietary_restrictions": ["vegetarian"],
        "accessibility_needs": [],
        "gift_preferences": ["books", "tea", "plants"]
    }',
    ARRAY['colleague', 'friend', 'tech', 'photography'],
    'Prefers morning meetings. Has two cats. Excellent at React development. Loves hiking on weekends.',
    '{
        "linkedin": "https://linkedin.com/in/sarahchen",
        "github": "https://github.com/sarahc-dev",
        "twitter": "@sarah_codes"
    }'
),
(
    uuid_generate_v4(),
    'Marcus Rodriguez',
    'Marcus',
    ARRAY['marcus.rodriguez@designstudio.co'],
    ARRAY['+1-555-2468'],
    'he/him',
    '{
        "birthday": "1987-11-22",
        "timezone": "America/Los_Angeles",
        "dietary_restrictions": [],
        "accessibility_needs": [],
        "gift_preferences": ["art supplies", "coffee", "vinyl records"]
    }',
    ARRAY['designer', 'musician', 'coffee-enthusiast'],
    'UX designer with a passion for music. Plays guitar in a local band. Always knows the best coffee shops.',
    '{
        "linkedin": "https://linkedin.com/in/marcusrodriguez",
        "instagram": "@marcus_designs",
        "spotify": "marcusmusic"
    }'
),
(
    uuid_generate_v4(),
    'Dr. Emily Watson',
    'Emily',
    ARRAY['emily.watson@university.edu', 'em.watson@gmail.com'],
    ARRAY['+1-555-1357'],
    'she/her',
    '{
        "birthday": "1985-07-08",
        "timezone": "America/Chicago",
        "dietary_restrictions": ["gluten-free"],
        "accessibility_needs": [],
        "gift_preferences": ["science books", "puzzle games", "wine"]
    }',
    ARRAY['professor', 'researcher', 'science', 'family'],
    'Computer Science professor at State University. Research focus on AI ethics. Sister to James Watson.',
    '{
        "linkedin": "https://linkedin.com/in/dr-emily-watson",
        "orcid": "0000-0002-1825-0097"
    }'
),
(
    uuid_generate_v4(),
    'James Watson',
    'Jamie',
    ARRAY['james.watson@startup.io'],
    ARRAY['+1-555-8024'],
    'he/him',
    '{
        "birthday": "1992-07-08",
        "timezone": "America/Chicago",
        "dietary_restrictions": [],
        "accessibility_needs": [],
        "gift_preferences": ["tech gadgets", "board games", "craft beer"]
    }',
    ARRAY['entrepreneur', 'tech', 'gaming', 'family'],
    'Startup founder working on fintech. Brother to Dr. Emily Watson. Avid board game collector.',
    '{
        "linkedin": "https://linkedin.com/in/james-watson-startup",
        "twitter": "@jamie_builds"
    }'
),
(
    uuid_generate_v4(),
    'Alex Kim',
    'Alex',
    ARRAY['alex@freelance.dev'],
    ARRAY['+1-555-3691'],
    'they/them',
    '{
        "birthday": "1993-12-03",
        "timezone": "America/Denver",
        "dietary_restrictions": ["vegan"],
        "accessibility_needs": ["hearing_impaired"],
        "gift_preferences": ["sustainable products", "experiences", "donations"]
    }',
    ARRAY['freelancer', 'developer', 'sustainability', 'accessibility'],
    'Full-stack developer focused on accessibility and sustainability. Prefers text/email communication. Passionate about environmental causes.',
    '{
        "github": "https://github.com/alexkimdev",
        "mastodon": "@alex@tech.lgbt"
    }'
);

-- Store the person IDs for reference in relationships
DO $$
DECLARE
    sarah_id UUID;
    marcus_id UUID;
    emily_id UUID;
    james_id UUID;
    alex_id UUID;
BEGIN
    -- Get the generated UUIDs
    SELECT id INTO sarah_id FROM persons WHERE full_name = 'Sarah Chen';
    SELECT id INTO marcus_id FROM persons WHERE full_name = 'Marcus Rodriguez';
    SELECT id INTO emily_id FROM persons WHERE full_name = 'Dr. Emily Watson';
    SELECT id INTO james_id FROM persons WHERE full_name = 'James Watson';
    SELECT id INTO alex_id FROM persons WHERE full_name = 'Alex Kim';

    -- =============================================================================
    -- SAMPLE ORGANIZATIONS
    -- =============================================================================

    INSERT INTO organizations (id, name, type, industry, website, metadata) VALUES
    (uuid_generate_v4(), 'TechCorp Inc.', 'employer', 'Technology', 'https://techcorp.com', '{"size": "500-1000", "stage": "public"}'),
    (uuid_generate_v4(), 'Creative Design Studio', 'employer', 'Design', 'https://designstudio.co', '{"size": "10-50", "stage": "private"}'),
    (uuid_generate_v4(), 'State University', 'school', 'Education', 'https://university.edu', '{"type": "public_university"}'),
    (uuid_generate_v4(), 'FinTech Innovations', 'employer', 'Financial Technology', 'https://startup.io', '{"size": "1-10", "stage": "seed"}');

    -- =============================================================================
    -- SAMPLE RELATIONSHIPS
    -- =============================================================================

    INSERT INTO relationships (from_person_id, to_person_id, relationship_type, strength, last_contact_date, shared_interests, notes) VALUES
    -- Sarah and Marcus: Work colleagues who became friends
    (sarah_id, marcus_id, 'friend', 0.85, CURRENT_DATE - INTERVAL '3 days', ARRAY['tech', 'design'], 'Met at a conference, now work on projects together'),
    (marcus_id, sarah_id, 'friend', 0.85, CURRENT_DATE - INTERVAL '3 days', ARRAY['tech', 'design'], 'Collaborate well on UX/dev projects'),
    
    -- Emily and James: Siblings
    (emily_id, james_id, 'sibling', 0.95, CURRENT_DATE - INTERVAL '1 week', ARRAY['family', 'gaming'], 'Close siblings, regular family game nights'),
    (james_id, emily_id, 'sibling', 0.95, CURRENT_DATE - INTERVAL '1 week', ARRAY['family', 'gaming'], 'Older sister, very supportive'),
    
    -- Sarah and Alex: Professional network
    (sarah_id, alex_id, 'colleague', 0.60, CURRENT_DATE - INTERVAL '2 weeks', ARRAY['tech', 'accessibility'], 'Consulted on accessibility features'),
    (alex_id, sarah_id, 'colleague', 0.65, CURRENT_DATE - INTERVAL '2 weeks', ARRAY['tech', 'accessibility'], 'Great technical mentor'),
    
    -- Marcus and Emily: Met through James
    (marcus_id, emily_id, 'friend', 0.40, CURRENT_DATE - INTERVAL '1 month', ARRAY['coffee'], 'Introduced by James, had coffee a few times'),
    (emily_id, marcus_id, 'friend', 0.40, CURRENT_DATE - INTERVAL '1 month', ARRAY['coffee'], 'Nice guy, interesting design perspective'),
    
    -- Alex and James: Startup ecosystem
    (alex_id, james_id, 'colleague', 0.55, CURRENT_DATE - INTERVAL '2 months', ARRAY['tech', 'entrepreneurship'], 'Met at startup meetup, potential collaboration');

    -- =============================================================================
    -- SAMPLE ADDRESSES
    -- =============================================================================

    INSERT INTO addresses (person_id, valid_from, address_type, street_address, city, state_province, postal_code, country, latitude, longitude, timezone, is_primary) VALUES
    (sarah_id, '2020-01-01', 'home', '123 Oak Street, Apt 4B', 'New York', 'NY', '10001', 'US', 40.7505, -73.9934, 'America/New_York', true),
    (marcus_id, '2019-06-01', 'home', '456 Sunset Blvd', 'Los Angeles', 'CA', '90028', 'US', 34.0969, -118.3267, 'America/Los_Angeles', true),
    (emily_id, '2018-08-15', 'home', '789 University Ave', 'Chicago', 'IL', '60637', 'US', 41.7886, -87.5987, 'America/Chicago', true),
    (james_id, '2022-03-01', 'home', '321 Innovation Drive', 'Chicago', 'IL', '60654', 'US', 41.8919, -87.6051, 'America/Chicago', true),
    (alex_id, '2021-09-01', 'home', '654 Mountain View Rd', 'Denver', 'CO', '80204', 'US', 39.7547, -104.9969, 'America/Denver', true);

    -- =============================================================================
    -- SAMPLE COMMUNICATION HISTORY
    -- =============================================================================

    INSERT INTO communication_history (from_person_id, to_person_id, channel, direction, communication_date, tone, response_latency_hours, message_length, subject_category) VALUES
    -- Recent communications
    (sarah_id, marcus_id, 'email', 'sent', CURRENT_TIMESTAMP - INTERVAL '3 days', 'casual', null, 'medium', 'work'),
    (marcus_id, sarah_id, 'email', 'received', CURRENT_TIMESTAMP - INTERVAL '2 days 20 hours', 'casual', 4, 'medium', 'work'),
    
    (emily_id, james_id, 'sms', 'sent', CURRENT_TIMESTAMP - INTERVAL '1 week', 'casual', null, 'short', 'personal'),
    (james_id, emily_id, 'sms', 'received', CURRENT_TIMESTAMP - INTERVAL '1 week' + INTERVAL '15 minutes', 'casual', 0, 'short', 'personal'),
    
    (sarah_id, alex_id, 'email', 'sent', CURRENT_TIMESTAMP - INTERVAL '2 weeks', 'professional', null, 'long', 'work'),
    (alex_id, sarah_id, 'email', 'received', CURRENT_TIMESTAMP - INTERVAL '2 weeks' + INTERVAL '8 hours', 'professional', 8, 'long', 'work');

    -- =============================================================================
    -- SAMPLE COMMUNICATION PREFERENCES (LEARNED)
    -- =============================================================================

    INSERT INTO communication_preferences (person_id, preferred_channels, preferred_tone, typical_response_latency_hours, preferred_message_length, best_contact_times, timezone) VALUES
    (sarah_id, ARRAY['email', 'sms'], 'casual', 6, 'medium', '{"morning": "9-11", "afternoon": "14-16"}', 'America/New_York'),
    (marcus_id, ARRAY['email'], 'casual', 4, 'medium', '{"afternoon": "13-17"}', 'America/Los_Angeles'),
    (emily_id, ARRAY['email'], 'professional', 12, 'long', '{"morning": "8-10", "evening": "18-20"}', 'America/Chicago'),
    (alex_id, ARRAY['email', 'sms'], 'professional', 8, 'long', '{"afternoon": "12-18"}', 'America/Denver');

    -- =============================================================================
    -- SAMPLE PERSON PREFERENCES
    -- =============================================================================

    INSERT INTO person_preferences (person_id, dietary_restrictions, allergies, accessibility_needs, interests, music_preferences, alcohol_preference, gift_preferences, religious_considerations, cultural_considerations) VALUES
    (sarah_id, ARRAY['vegetarian'], ARRAY[], ARRAY[], ARRAY['photography', 'hiking', 'react', 'cats'], ARRAY['indie', 'folk'], 'occasionally', ARRAY['books', 'tea', 'plants'], ARRAY[], ARRAY[]),
    (marcus_id, ARRAY[], ARRAY['peanuts'], ARRAY[], ARRAY['design', 'music', 'guitar', 'coffee'], ARRAY['jazz', 'rock', 'blues'], 'drinks', ARRAY['art supplies', 'coffee', 'vinyl records'], ARRAY[], ARRAY[]),
    (emily_id, ARRAY['gluten-free'], ARRAY['gluten'], ARRAY[], ARRAY['ai ethics', 'research', 'puzzles', 'wine'], ARRAY['classical', 'ambient'], 'drinks', ARRAY['science books', 'puzzle games', 'wine'], ARRAY[], ARRAY[]),
    (james_id, ARRAY[], ARRAY[], ARRAY[], ARRAY['entrepreneurship', 'fintech', 'board games', 'craft beer'], ARRAY['electronic', 'hip-hop'], 'drinks', ARRAY['tech gadgets', 'board games', 'craft beer'], ARRAY[], ARRAY[]),
    (alex_id, ARRAY['vegan'], ARRAY[], ARRAY['hearing_impaired'], ARRAY['sustainability', 'accessibility', 'programming', 'environment'], ARRAY['electronic', 'ambient'], 'non-drinker', ARRAY['sustainable products', 'experiences', 'donations'], ARRAY[], ARRAY[]);

    -- =============================================================================
    -- SAMPLE CONSENT RECORDS
    -- =============================================================================

    INSERT INTO consent_records (person_id, scope, granted, granted_at, granted_by, granularity) VALUES
    -- Sarah: Full consent for most things
    (sarah_id, 'dietary', true, CURRENT_TIMESTAMP, 'self', 'full'),
    (sarah_id, 'accessibility', true, CURRENT_TIMESTAMP, 'self', 'full'),
    (sarah_id, 'communication_analysis', true, CURRENT_TIMESTAMP, 'self', 'metadata-only'),
    
    -- Marcus: Basic consent
    (marcus_id, 'dietary', true, CURRENT_TIMESTAMP, 'implied', 'full'),
    (marcus_id, 'communication_analysis', true, CURRENT_TIMESTAMP, 'implied', 'metadata-only'),
    
    -- Emily: Academic-level consent
    (emily_id, 'dietary', true, CURRENT_TIMESTAMP, 'self', 'full'),
    (emily_id, 'communication_analysis', true, CURRENT_TIMESTAMP, 'self', 'full'),
    (emily_id, 'calendar', true, CURRENT_TIMESTAMP, 'self', 'limited'),
    
    -- Alex: Privacy-conscious, limited consent
    (alex_id, 'accessibility', true, CURRENT_TIMESTAMP, 'self', 'full'),
    (alex_id, 'dietary', true, CURRENT_TIMESTAMP, 'self', 'full'),
    (alex_id, 'communication_analysis', false, CURRENT_TIMESTAMP, 'self', 'full');

    -- =============================================================================
    -- SAMPLE COMPUTED ANALYTICS
    -- =============================================================================

    INSERT INTO social_analytics (person_id, overall_closeness_score, social_centrality_score, mutual_connection_count, avg_response_latency_hours, communication_frequency_score, last_interaction_days, top_shared_interests, maintenance_priority_score, computation_version) VALUES
    (sarah_id, 0.7500, 0.8200, 3, 6, 0.85, 3, ARRAY['tech', 'design', 'photography'], 0.25, 'v1.0.0'),
    (marcus_id, 0.6800, 0.7000, 2, 4, 0.90, 3, ARRAY['design', 'music', 'coffee'], 0.30, 'v1.0.0'),
    (emily_id, 0.8100, 0.6500, 2, 12, 0.70, 7, ARRAY['science', 'family', 'gaming'], 0.40, 'v1.0.0'),
    (james_id, 0.7800, 0.6800, 3, 2, 0.80, 7, ARRAY['tech', 'gaming', 'entrepreneurship'], 0.35, 'v1.0.0'),
    (alex_id, 0.5500, 0.4200, 1, 8, 0.60, 14, ARRAY['tech', 'accessibility', 'sustainability'], 0.60, 'v1.0.0');

END $$;