-- Real Estate Seed Data: Leads and Contact Management
-- Description: Sample lead and contact data for CRM automation

-- Create leads table
CREATE TABLE IF NOT EXISTS leads (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    lead_source VARCHAR(100),
    lead_type VARCHAR(50) DEFAULT 'buyer',
    status VARCHAR(50) DEFAULT 'new',
    budget_min DECIMAL(12,2),
    budget_max DECIMAL(12,2),
    preferred_areas TEXT[],
    notes TEXT,
    assigned_agent_id INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_contacted TIMESTAMP WITH TIME ZONE
);

-- Create lead activities table
CREATE TABLE IF NOT EXISTS lead_activities (
    id SERIAL PRIMARY KEY,
    lead_id INTEGER REFERENCES leads(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    description TEXT,
    scheduled_date TIMESTAMP WITH TIME ZONE,
    completed_date TIMESTAMP WITH TIME ZONE,
    outcome VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create agents table
CREATE TABLE IF NOT EXISTS agents (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    license_number VARCHAR(50),
    specialties TEXT[],
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample agents
INSERT INTO agents (first_name, last_name, email, phone, license_number, specialties) VALUES
('Sarah', 'Johnson', 'sarah.johnson@realestate.com', '555-0101', 'RE123456', ARRAY['Residential', 'First-time Buyers']),
('Mike', 'Chen', 'mike.chen@realestate.com', '555-0102', 'RE234567', ARRAY['Luxury Homes', 'Investment Properties']),
('Lisa', 'Rodriguez', 'lisa.rodriguez@realestate.com', '555-0103', 'RE345678', ARRAY['Commercial', 'Land Development']);

-- Insert sample leads
INSERT INTO leads (first_name, last_name, email, phone, lead_source, lead_type, status, budget_min, budget_max, preferred_areas, assigned_agent_id) VALUES
('John', 'Smith', 'john.smith@email.com', '555-1001', 'Website Form', 'buyer', 'qualified', 200000, 300000, ARRAY['Downtown', 'Westside'], 1),
('Emma', 'Davis', 'emma.davis@email.com', '555-1002', 'Referral', 'buyer', 'new', 150000, 250000, ARRAY['Suburbs', 'School District 5'], 1),
('Robert', 'Wilson', 'robert.wilson@email.com', '555-1003', 'Open House', 'seller', 'qualified', NULL, NULL, ARRAY['North End'], 2),
('Maria', 'Garcia', 'maria.garcia@email.com', '555-1004', 'Facebook Ad', 'buyer', 'contacted', 300000, 500000, ARRAY['Luxury District'], 2);

-- Insert sample lead activities
INSERT INTO lead_activities (lead_id, activity_type, description, completed_date, outcome) VALUES
(1, 'phone_call', 'Initial qualification call', NOW() - INTERVAL '2 days', 'qualified'),
(1, 'email', 'Sent property listings matching criteria', NOW() - INTERVAL '1 day', 'opened'),
(2, 'email', 'Welcome email with market overview', NOW() - INTERVAL '3 days', 'opened'),
(3, 'meeting', 'Home valuation appointment', NOW() - INTERVAL '1 day', 'listing_agreement_signed'),
(4, 'phone_call', 'Follow-up from Facebook inquiry', NOW() - INTERVAL '1 hour', 'scheduled_showing');

-- Insert scheduled activities
INSERT INTO lead_activities (lead_id, activity_type, description, scheduled_date) VALUES
(1, 'showing', 'Property tour - 123 Main Street', NOW() + INTERVAL '2 days'),
(2, 'phone_call', 'Follow-up call to discuss financing options', NOW() + INTERVAL '1 day'),
(4, 'showing', 'Luxury home tour - 789 Pine Road', NOW() + INTERVAL '3 days');