-- Add Exchange Rates Table
-- Supports multi-currency invoicing with automatic conversion

CREATE TABLE IF NOT EXISTS exchange_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    base_currency VARCHAR(3) NOT NULL,
    target_currency VARCHAR(3) NOT NULL,
    rate DECIMAL(15,6) NOT NULL,
    source VARCHAR(100) DEFAULT 'manual', -- manual, api, cache
    valid_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Ensure we don't have duplicate active rates for same currency pair
    CONSTRAINT unique_active_rate UNIQUE (base_currency, target_currency, valid_from)
);

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_exchange_rates_currencies
ON exchange_rates(base_currency, target_currency);

CREATE INDEX IF NOT EXISTS idx_exchange_rates_valid
ON exchange_rates(valid_from, valid_until);

-- Add exchange rate tracking to invoices
ALTER TABLE invoices
ADD COLUMN IF NOT EXISTS exchange_rate DECIMAL(15,6),
ADD COLUMN IF NOT EXISTS base_currency VARCHAR(3),
ADD COLUMN IF NOT EXISTS converted_total DECIMAL(15,2);

-- Function to get latest exchange rate
CREATE OR REPLACE FUNCTION get_exchange_rate(
    p_base_currency VARCHAR(3),
    p_target_currency VARCHAR(3),
    p_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
RETURNS DECIMAL(15,6) AS $$
DECLARE
    v_rate DECIMAL(15,6);
BEGIN
    -- Same currency = 1.0
    IF p_base_currency = p_target_currency THEN
        RETURN 1.0;
    END IF;

    -- Get most recent rate for the date
    SELECT rate INTO v_rate
    FROM exchange_rates
    WHERE base_currency = p_base_currency
      AND target_currency = p_target_currency
      AND valid_from <= p_date
      AND (valid_until IS NULL OR valid_until > p_date)
    ORDER BY valid_from DESC
    LIMIT 1;

    -- If no rate found, try inverse rate
    IF v_rate IS NULL THEN
        SELECT 1.0 / rate INTO v_rate
        FROM exchange_rates
        WHERE base_currency = p_target_currency
          AND target_currency = p_base_currency
          AND valid_from <= p_date
          AND (valid_until IS NULL OR valid_until > p_date)
        ORDER BY valid_from DESC
        LIMIT 1;
    END IF;

    RETURN v_rate;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update exchange rate on invoice creation/update
CREATE OR REPLACE FUNCTION update_invoice_exchange_rate()
RETURNS TRIGGER AS $$
DECLARE
    v_company_currency VARCHAR(3);
    v_rate DECIMAL(15,6);
BEGIN
    -- Get company's default currency
    SELECT default_currency INTO v_company_currency
    FROM companies
    WHERE id = NEW.company_id;

    -- If invoice currency differs from company currency, apply exchange rate
    IF NEW.currency != v_company_currency THEN
        NEW.base_currency := v_company_currency;
        NEW.exchange_rate := get_exchange_rate(NEW.currency, v_company_currency, NEW.issue_date::TIMESTAMP);

        IF NEW.exchange_rate IS NOT NULL THEN
            NEW.converted_total := NEW.total_amount * NEW.exchange_rate;
        END IF;
    ELSE
        NEW.base_currency := NULL;
        NEW.exchange_rate := NULL;
        NEW.converted_total := NULL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to invoices
DROP TRIGGER IF EXISTS trigger_update_invoice_exchange_rate ON invoices;
CREATE TRIGGER trigger_update_invoice_exchange_rate
    BEFORE INSERT OR UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION update_invoice_exchange_rate();

-- Insert common exchange rates (USD as base)
-- These will be replaced by API fetched rates in production
INSERT INTO exchange_rates (base_currency, target_currency, rate, source) VALUES
    ('USD', 'EUR', 0.92, 'manual'),
    ('USD', 'GBP', 0.79, 'manual'),
    ('USD', 'JPY', 149.50, 'manual'),
    ('USD', 'CAD', 1.35, 'manual'),
    ('USD', 'AUD', 1.52, 'manual'),
    ('USD', 'CHF', 0.88, 'manual'),
    ('USD', 'CNY', 7.24, 'manual'),
    ('USD', 'INR', 83.12, 'manual')
ON CONFLICT (base_currency, target_currency, valid_from) DO NOTHING;
