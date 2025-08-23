-- Seed data for Personal Relationship Manager

-- Sample contacts
INSERT INTO contacts (name, nickname, relationship_type, birthday, email, notes, interests, tags) VALUES
('Sarah Johnson', 'Sarah', 'friend', '1990-03-15', 'sarah@example.com', 'Met at college, loves hiking', ARRAY['hiking', 'photography', 'coffee', 'travel'], ARRAY['college', 'close friend']),
('Michael Chen', 'Mike', 'colleague', '1985-07-22', 'mchen@work.com', 'Great team member, has two kids', ARRAY['technology', 'basketball', 'cooking'], ARRAY['work', 'mentor']),
('Emma Wilson', 'Em', 'family', '1995-12-10', 'emma.w@email.com', 'Cousin, studying medicine', ARRAY['books', 'yoga', 'medicine', 'cats'], ARRAY['family', 'cousin']),
('David Martinez', 'Dave', 'friend', '1988-05-30', 'dave.m@email.com', 'Neighbor, plays guitar', ARRAY['music', 'guitar', 'craft beer', 'vinyl records'], ARRAY['neighbor', 'musician']);

-- Sample events
INSERT INTO events (contact_id, event_type, event_date, recurring, reminder_days_before, notes) VALUES
(1, 'birthday', '2024-03-15', true, 7, 'Likes adventure-related gifts'),
(2, 'birthday', '2024-07-22', true, 7, 'Prefers practical gifts'),
(3, 'birthday', '2024-12-10', true, 14, 'Send a card early'),
(1, 'custom', '2024-06-01', false, 3, 'Graduation party');

-- Sample interactions
INSERT INTO interactions (contact_id, interaction_type, interaction_date, description, sentiment) VALUES
(1, 'in-person', '2024-01-10', 'Coffee catch-up downtown', 'positive'),
(2, 'call', '2024-01-08', 'Discussed project timeline', 'neutral'),
(3, 'text', '2024-01-12', 'Happy New Year message', 'positive'),
(4, 'in-person', '2024-01-05', 'New Year party at his place', 'positive');

-- Sample gift ideas and history
INSERT INTO gifts (contact_id, gift_name, description, price, occasion, status, notes) VALUES
(1, 'Hiking Backpack', 'Osprey 40L daypack', 150.00, 'Birthday', 'idea', 'She mentioned needing a new pack'),
(1, 'Photography Book', 'National Geographic Best Photos', 35.00, 'Christmas', 'given', 'Loved it!'),
(2, 'Wireless Earbuds', 'For his daily runs', 120.00, 'Birthday', 'idea', 'He lost his old ones'),
(3, 'Medical Textbook', 'Gray''s Anatomy for Students', 85.00, 'Graduation', 'purchased', 'For her studies'),
(4, 'Guitar Picks Set', 'Custom engraved picks', 25.00, 'Birthday', 'given', 'Really appreciated the personal touch');

-- Sample reminders
INSERT INTO reminders (contact_id, reminder_type, reminder_date, message) VALUES
(1, 'birthday', '2024-03-08', 'Sarah''s birthday is coming up in 7 days!'),
(2, 'check-in', '2024-02-01', 'Check in with Mike about the project'),
(3, 'birthday', '2024-11-26', 'Emma''s birthday in 2 weeks - order gift online'),
(4, 'event', '2024-02-14', 'Dave''s band playing at local venue tonight');

-- Sample milestones
INSERT INTO milestones (contact_id, milestone_type, milestone_date, description) VALUES
(1, 'friendship', '2018-09-01', 'First day we met at orientation'),
(2, 'professional', '2022-01-15', 'Started working together on Alpha project'),
(3, 'education', '2024-05-15', 'Emma''s medical school graduation'),
(4, 'personal', '2023-07-01', 'Helped me move to new apartment');