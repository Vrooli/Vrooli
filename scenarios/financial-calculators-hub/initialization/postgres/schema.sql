-- Financial Calculators Hub Database Schema

-- Create database if not exists
SELECT 'CREATE DATABASE financial_calculators_hub'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'financial_calculators_hub')\gexec

\c financial_calculators_hub;

-- Create calculations table to store all calculation history
CREATE TABLE IF NOT EXISTS calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calculator_type VARCHAR(50) NOT NULL,
    inputs JSONB NOT NULL,
    outputs JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    user_id UUID,
    notes TEXT,
    INDEX idx_calculator_type (calculator_type),
    INDEX idx_created_at (created_at),
    INDEX idx_user_id (user_id)
);

-- Create saved_scenarios table for grouping calculations
CREATE TABLE IF NOT EXISTS saved_scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    calculations UUID[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    share_token VARCHAR(100) UNIQUE,
    INDEX idx_share_token (share_token),
    INDEX idx_created_at (created_at)
);

-- Create net_worth_tracker table for tracking assets and liabilities
CREATE TABLE IF NOT EXISTS net_worth_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    entry_date DATE NOT NULL,
    assets JSONB NOT NULL DEFAULT '{}',
    liabilities JSONB NOT NULL DEFAULT '{}',
    net_worth DECIMAL(15, 2) GENERATED ALWAYS AS (
        (assets->>'total')::DECIMAL - (liabilities->>'total')::DECIMAL
    ) STORED,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_entry_date (user_id, entry_date),
    INDEX idx_entry_date (entry_date),
    UNIQUE(user_id, entry_date)
);

-- Create tax_calculations table
CREATE TABLE IF NOT EXISTS tax_calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    tax_year INTEGER NOT NULL,
    income DECIMAL(15, 2) NOT NULL,
    filing_status VARCHAR(50) NOT NULL,
    deductions JSONB DEFAULT '{}',
    credits JSONB DEFAULT '{}',
    tax_owed DECIMAL(15, 2),
    effective_rate DECIMAL(5, 2),
    marginal_rate DECIMAL(5, 2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_year (user_id, tax_year)
);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER update_saved_scenarios_updated_at 
    BEFORE UPDATE ON saved_scenarios
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_net_worth_updated_at
    BEFORE UPDATE ON net_worth_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;