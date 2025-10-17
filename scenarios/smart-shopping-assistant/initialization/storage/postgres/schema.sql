-- Smart Shopping Assistant Database Schema
-- Version: 1.0.0

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Shopping Profiles table
CREATE TABLE IF NOT EXISTS shopping_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID, -- From scenario-authenticator
    name VARCHAR(255) NOT NULL,
    preferences JSONB DEFAULT '{}',
    budget_limits JSONB DEFAULT '{}',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Products table
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(500), -- ID from retailer
    name VARCHAR(500) NOT NULL,
    category VARCHAR(255),
    subcategory VARCHAR(255),
    description TEXT,
    features JSONB DEFAULT '{}',
    specifications JSONB DEFAULT '{}',
    brand VARCHAR(255),
    model VARCHAR(255),
    upc VARCHAR(50),
    asin VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(external_id)
);

-- Price History table
CREATE TABLE IF NOT EXISTS price_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    retailer VARCHAR(255) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    original_price DECIMAL(10, 2),
    currency VARCHAR(3) DEFAULT 'USD',
    availability VARCHAR(50),
    shipping_cost DECIMAL(10, 2),
    url TEXT,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_price_history_product_time (product_id, recorded_at DESC)
);

-- Purchase History table
CREATE TABLE IF NOT EXISTS purchases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID REFERENCES shopping_profiles(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    price DECIMAL(10, 2) NOT NULL,
    quantity INTEGER DEFAULT 1,
    retailer VARCHAR(255),
    purchase_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    affiliate_revenue DECIMAL(10, 2) DEFAULT 0,
    notes TEXT,
    satisfaction_rating INTEGER CHECK (satisfaction_rating >= 1 AND satisfaction_rating <= 5),
    return_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Price Alerts table
CREATE TABLE IF NOT EXISTS price_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID REFERENCES shopping_profiles(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    target_price DECIMAL(10, 2),
    alert_type VARCHAR(50) NOT NULL, -- below_target, percentage_drop, back_in_stock
    alert_config JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_triggered TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Shopping Lists table
CREATE TABLE IF NOT EXISTS shopping_lists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID REFERENCES shopping_profiles(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_shared BOOLEAN DEFAULT false,
    shared_with JSONB DEFAULT '[]', -- Array of profile IDs
    items JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Product Alternatives table
CREATE TABLE IF NOT EXISTS product_alternatives (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    alternative_id UUID REFERENCES products(id) ON DELETE CASCADE,
    alternative_type VARCHAR(50), -- similar, generic, used, rental, refurbished
    similarity_score DECIMAL(3, 2) CHECK (similarity_score >= 0 AND similarity_score <= 1),
    price_difference DECIMAL(10, 2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, alternative_id)
);

-- Affiliate Links table
CREATE TABLE IF NOT EXISTS affiliate_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    retailer VARCHAR(255) NOT NULL,
    affiliate_network VARCHAR(255), -- Amazon Associates, ShareASale, etc.
    original_url TEXT NOT NULL,
    affiliate_url TEXT NOT NULL,
    commission_rate DECIMAL(5, 2),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Purchase Patterns table
CREATE TABLE IF NOT EXISTS purchase_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID REFERENCES shopping_profiles(id) ON DELETE CASCADE,
    product_category VARCHAR(255),
    pattern_type VARCHAR(50), -- recurring, seasonal, event-based
    frequency_days INTEGER,
    average_quantity DECIMAL(10, 2),
    average_spend DECIMAL(10, 2),
    confidence_score DECIMAL(3, 2) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    last_analyzed TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Gift Recommendations table
CREATE TABLE IF NOT EXISTS gift_recommendations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID REFERENCES shopping_profiles(id) ON DELETE CASCADE,
    recipient_contact_id UUID, -- From contact-book scenario
    occasion VARCHAR(255),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    recommendation_score DECIMAL(3, 2) CHECK (recommendation_score >= 0 AND recommendation_score <= 1),
    reason TEXT,
    was_purchased BOOLEAN DEFAULT false,
    feedback VARCHAR(50), -- loved, liked, neutral, disliked
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Reviews Summary table (aggregated from multiple sources)
CREATE TABLE IF NOT EXISTS reviews_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    source VARCHAR(255), -- Amazon, BestBuy, etc.
    average_rating DECIMAL(2, 1) CHECK (average_rating >= 0 AND average_rating <= 5),
    total_reviews INTEGER,
    pros JSONB DEFAULT '[]',
    cons JSONB DEFAULT '[]',
    sentiment_score DECIMAL(3, 2) CHECK (sentiment_score >= -1 AND sentiment_score <= 1),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_profiles_user_id ON shopping_profiles(user_id);
CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_brand ON products(brand);
CREATE INDEX idx_purchases_profile_date ON purchases(profile_id, purchase_date DESC);
CREATE INDEX idx_price_alerts_profile_active ON price_alerts(profile_id, is_active);
CREATE INDEX idx_patterns_profile_type ON purchase_patterns(profile_id, pattern_type);
CREATE INDEX idx_gift_recs_profile_occasion ON gift_recommendations(profile_id, occasion);

-- Create update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_shopping_profiles_updated_at BEFORE UPDATE ON shopping_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_price_alerts_updated_at BEFORE UPDATE ON price_alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_shopping_lists_updated_at BEFORE UPDATE ON shopping_lists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_affiliate_links_updated_at BEFORE UPDATE ON affiliate_links
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_purchase_patterns_updated_at BEFORE UPDATE ON purchase_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();