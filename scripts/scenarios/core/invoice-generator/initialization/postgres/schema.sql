-- Invoice Generator Database Schema
-- Professional invoice generation and management system

-- Companies/Organizations
CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    logo_url TEXT,
    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    city VARCHAR(100),
    state_province VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(100),
    phone VARCHAR(50),
    email VARCHAR(255),
    website VARCHAR(255),
    tax_id VARCHAR(100),
    default_payment_terms INTEGER DEFAULT 30, -- Days
    default_currency VARCHAR(3) DEFAULT 'USD',
    invoice_prefix VARCHAR(20) DEFAULT 'INV',
    next_invoice_number INTEGER DEFAULT 1001,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Clients
CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    city VARCHAR(100),
    state_province VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(100),
    tax_id VARCHAR(100),
    payment_terms INTEGER, -- Override company default if set
    currency VARCHAR(3),
    notes TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Invoice Templates
CREATE TABLE IF NOT EXISTS invoice_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_data JSONB NOT NULL, -- Template configuration and styling
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Invoices
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
    template_id UUID REFERENCES invoice_templates(id) ON DELETE SET NULL,
    invoice_number VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'draft', -- draft, sent, viewed, paid, overdue, cancelled
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE,
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- Amounts
    subtotal DECIMAL(15,2) DEFAULT 0,
    tax_amount DECIMAL(15,2) DEFAULT 0,
    discount_amount DECIMAL(15,2) DEFAULT 0,
    total_amount DECIMAL(15,2) DEFAULT 0,
    paid_amount DECIMAL(15,2) DEFAULT 0,
    balance_due DECIMAL(15,2) DEFAULT 0,
    
    -- Tax settings
    tax_rate DECIMAL(5,2),
    tax_name VARCHAR(100),
    
    -- Discount settings
    discount_rate DECIMAL(5,2),
    discount_type VARCHAR(20), -- percentage, fixed
    
    -- Additional fields
    notes TEXT,
    terms TEXT,
    footer_text TEXT,
    
    -- Payment info
    payment_method VARCHAR(100),
    payment_reference VARCHAR(255),
    paid_date DATE,
    
    -- Tracking
    sent_date TIMESTAMP,
    viewed_date TIMESTAMP,
    reminder_count INTEGER DEFAULT 0,
    last_reminder_date TIMESTAMP,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(company_id, invoice_number)
);

-- Invoice Items
CREATE TABLE IF NOT EXISTS invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID REFERENCES invoices(id) ON DELETE CASCADE,
    item_order INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    quantity DECIMAL(10,2) DEFAULT 1,
    unit_price DECIMAL(15,2) NOT NULL,
    unit VARCHAR(50), -- hours, units, pieces, etc.
    tax_rate DECIMAL(5,2),
    discount_rate DECIMAL(5,2),
    line_total DECIMAL(15,2) NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products/Services catalog (optional, for quick item selection)
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    unit_price DECIMAL(15,2) NOT NULL,
    unit VARCHAR(50),
    tax_rate DECIMAL(5,2),
    category VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Payments
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID REFERENCES invoices(id) ON DELETE CASCADE,
    amount DECIMAL(15,2) NOT NULL,
    payment_date DATE NOT NULL,
    payment_method VARCHAR(100),
    reference_number VARCHAR(255),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Recurring Invoice Templates
CREATE TABLE IF NOT EXISTS recurring_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    client_id UUID REFERENCES clients(id) ON DELETE CASCADE,
    template_id UUID REFERENCES invoice_templates(id) ON DELETE SET NULL,
    
    -- Recurrence settings
    frequency VARCHAR(20) NOT NULL, -- daily, weekly, monthly, quarterly, yearly
    interval_count INTEGER DEFAULT 1, -- Every X periods
    start_date DATE NOT NULL,
    end_date DATE,
    next_invoice_date DATE,
    
    -- Invoice defaults
    payment_terms INTEGER,
    tax_rate DECIMAL(5,2),
    discount_rate DECIMAL(5,2),
    discount_type VARCHAR(20),
    
    -- Content
    items JSONB NOT NULL, -- Array of invoice items
    notes TEXT,
    terms TEXT,
    
    is_active BOOLEAN DEFAULT true,
    last_generated_date DATE,
    total_generated INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tax Rates
CREATE TABLE IF NOT EXISTS tax_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    rate DECIMAL(5,2) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Invoice Activity Log
CREATE TABLE IF NOT EXISTS invoice_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID REFERENCES invoices(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL, -- created, updated, sent, viewed, paid, etc.
    description TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_invoices_company ON invoices(company_id);
CREATE INDEX idx_invoices_client ON invoices(client_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_created_at ON invoices(created_at DESC);
CREATE INDEX idx_invoice_items_invoice ON invoice_items(invoice_id);
CREATE INDEX idx_clients_company ON clients(company_id);
CREATE INDEX idx_clients_active ON clients(is_active);
CREATE INDEX idx_products_company ON products(company_id);
CREATE INDEX idx_products_active ON products(is_active);
CREATE INDEX idx_payments_invoice ON payments(invoice_id);
CREATE INDEX idx_payments_date ON payments(payment_date DESC);
CREATE INDEX idx_recurring_active ON recurring_invoices(is_active, next_invoice_date);
CREATE INDEX idx_activities_invoice ON invoice_activities(invoice_id);

-- Full text search
CREATE INDEX idx_clients_search ON clients USING gin(to_tsvector('english', name || ' ' || COALESCE(email, '')));
CREATE INDEX idx_invoices_number_search ON invoices USING gin(invoice_number gin_trgm_ops);
CREATE INDEX idx_products_search ON products USING gin(to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- Function to update invoice totals
CREATE OR REPLACE FUNCTION calculate_invoice_totals()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE invoices 
    SET 
        subtotal = (SELECT COALESCE(SUM(line_total), 0) FROM invoice_items WHERE invoice_id = NEW.invoice_id),
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.invoice_id;
    
    UPDATE invoices 
    SET 
        tax_amount = subtotal * COALESCE(tax_rate, 0) / 100,
        discount_amount = CASE 
            WHEN discount_type = 'percentage' THEN subtotal * COALESCE(discount_rate, 0) / 100
            ELSE COALESCE(discount_rate, 0)
        END,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.invoice_id;
    
    UPDATE invoices 
    SET 
        total_amount = subtotal + tax_amount - discount_amount,
        balance_due = subtotal + tax_amount - discount_amount - paid_amount,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.invoice_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to recalculate invoice totals when items change
CREATE TRIGGER trigger_invoice_totals
    AFTER INSERT OR UPDATE OR DELETE ON invoice_items
    FOR EACH ROW EXECUTE FUNCTION calculate_invoice_totals();

-- Function to update invoice payment status
CREATE OR REPLACE FUNCTION update_invoice_payment_status()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE invoices 
    SET 
        paid_amount = (SELECT COALESCE(SUM(amount), 0) FROM payments WHERE invoice_id = NEW.invoice_id),
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.invoice_id;
    
    UPDATE invoices 
    SET 
        balance_due = total_amount - paid_amount,
        status = CASE 
            WHEN paid_amount >= total_amount THEN 'paid'
            WHEN paid_amount > 0 THEN 'partially_paid'
            ELSE status
        END,
        paid_date = CASE 
            WHEN paid_amount >= total_amount THEN CURRENT_DATE
            ELSE NULL
        END,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.invoice_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update payment status when payments are made
CREATE TRIGGER trigger_payment_status
    AFTER INSERT OR UPDATE OR DELETE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_invoice_payment_status();

-- Function to generate next invoice number
CREATE OR REPLACE FUNCTION get_next_invoice_number(p_company_id UUID)
RETURNS VARCHAR AS $$
DECLARE
    v_prefix VARCHAR(20);
    v_next_number INTEGER;
    v_invoice_number VARCHAR(100);
BEGIN
    SELECT invoice_prefix, next_invoice_number 
    INTO v_prefix, v_next_number
    FROM companies 
    WHERE id = p_company_id;
    
    v_invoice_number := v_prefix || '-' || LPAD(v_next_number::TEXT, 5, '0');
    
    UPDATE companies 
    SET next_invoice_number = next_invoice_number + 1
    WHERE id = p_company_id;
    
    RETURN v_invoice_number;
END;
$$ LANGUAGE plpgsql;