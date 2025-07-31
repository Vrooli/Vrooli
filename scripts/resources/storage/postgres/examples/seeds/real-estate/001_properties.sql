-- Real Estate Seed Data: Properties and Listings
-- Description: Sample property data for real estate client automation

-- Create properties table
CREATE TABLE IF NOT EXISTS properties (
    id SERIAL PRIMARY KEY,
    mls_number VARCHAR(50) UNIQUE,
    address VARCHAR(255) NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(50) NOT NULL,
    zip_code VARCHAR(20) NOT NULL,
    price DECIMAL(12,2) NOT NULL,
    bedrooms INTEGER,
    bathrooms DECIMAL(3,1),
    square_feet INTEGER,
    lot_size DECIMAL(10,2),
    property_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    listing_date DATE DEFAULT CURRENT_DATE,
    description TEXT,
    agent_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create property features table
CREATE TABLE IF NOT EXISTS property_features (
    id SERIAL PRIMARY KEY,
    property_id INTEGER REFERENCES properties(id) ON DELETE CASCADE,
    feature VARCHAR(100) NOT NULL,
    value VARCHAR(255)
);

-- Insert sample properties
INSERT INTO properties (mls_number, address, city, state, zip_code, price, bedrooms, bathrooms, square_feet, lot_size, property_type, description) VALUES
('MLS001234', '123 Main Street', 'Springfield', 'IL', '62701', 250000.00, 3, 2.0, 1800, 0.25, 'Single Family', 'Beautiful 3-bedroom home with updated kitchen and hardwood floors.'),
('MLS005678', '456 Oak Avenue', 'Springfield', 'IL', '62702', 180000.00, 2, 1.5, 1200, 0.18, 'Townhouse', 'Cozy townhouse in quiet neighborhood, perfect for first-time buyers.'),
('MLS009876', '789 Pine Road', 'Springfield', 'IL', '62703', 450000.00, 4, 3.5, 2800, 0.50, 'Single Family', 'Luxury home with modern amenities and large backyard.'),
('MLS011111', '321 Elm Drive', 'Springfield', 'IL', '62704', 320000.00, 3, 2.5, 2100, 0.30, 'Single Family', 'Recently renovated home with open floor plan.');

-- Insert sample property features
INSERT INTO property_features (property_id, feature, value) VALUES
(1, 'Parking', '2-car garage'),
(1, 'Heating', 'Central air/heat'),
(1, 'Flooring', 'Hardwood'),
(2, 'Parking', '1-car garage'),
(2, 'Heating', 'Gas forced air'),
(3, 'Parking', '3-car garage'),
(3, 'Heating', 'Central air/heat'),
(3, 'Pool', 'In-ground heated'),
(4, 'Parking', '2-car garage'),
(4, 'Heating', 'Central air/heat'),
(4, 'Kitchen', 'Granite countertops');