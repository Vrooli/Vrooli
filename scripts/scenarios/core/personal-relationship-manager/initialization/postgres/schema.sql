-- Personal Relationship Manager Database Schema

-- Contacts table
CREATE TABLE IF NOT EXISTS contacts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    nickname VARCHAR(100),
    relationship_type VARCHAR(50), -- family, friend, colleague, etc.
    birthday DATE,
    anniversary DATE,
    email VARCHAR(255),
    phone VARCHAR(50),
    address TEXT,
    notes TEXT,
    interests TEXT[], -- Array of interests
    tags TEXT[], -- Array of tags for grouping
    favorite_color VARCHAR(50),
    clothing_size VARCHAR(20),
    photo_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Events table for tracking important dates
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    contact_id INTEGER REFERENCES contacts(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, -- birthday, anniversary, custom
    event_date DATE NOT NULL,
    recurring BOOLEAN DEFAULT true,
    reminder_days_before INTEGER DEFAULT 7,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Interactions table for tracking communication history
CREATE TABLE IF NOT EXISTS interactions (
    id SERIAL PRIMARY KEY,
    contact_id INTEGER REFERENCES contacts(id) ON DELETE CASCADE,
    interaction_type VARCHAR(50), -- call, text, email, in-person, gift
    interaction_date DATE NOT NULL,
    description TEXT,
    sentiment VARCHAR(20), -- positive, neutral, negative
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Gifts table for tracking gift history and ideas
CREATE TABLE IF NOT EXISTS gifts (
    id SERIAL PRIMARY KEY,
    contact_id INTEGER REFERENCES contacts(id) ON DELETE CASCADE,
    gift_name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2),
    occasion VARCHAR(100),
    given_date DATE,
    status VARCHAR(20) DEFAULT 'idea', -- idea, purchased, given
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    notes TEXT,
    purchase_link TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Reminders table for scheduled notifications
CREATE TABLE IF NOT EXISTS reminders (
    id SERIAL PRIMARY KEY,
    contact_id INTEGER REFERENCES contacts(id) ON DELETE CASCADE,
    reminder_type VARCHAR(50), -- birthday, check-in, event, gift
    reminder_date DATE NOT NULL,
    reminder_time TIME DEFAULT '09:00:00',
    message TEXT,
    sent BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Relationship milestones
CREATE TABLE IF NOT EXISTS milestones (
    id SERIAL PRIMARY KEY,
    contact_id INTEGER REFERENCES contacts(id) ON DELETE CASCADE,
    milestone_type VARCHAR(100),
    milestone_date DATE,
    description TEXT,
    photo_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_contacts_birthday ON contacts(birthday);
CREATE INDEX idx_events_date ON events(event_date);
CREATE INDEX idx_interactions_contact_date ON interactions(contact_id, interaction_date);
CREATE INDEX idx_reminders_date ON reminders(reminder_date);
CREATE INDEX idx_gifts_contact_status ON gifts(contact_id, status);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for contacts table
CREATE TRIGGER update_contacts_updated_at BEFORE UPDATE
    ON contacts FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();