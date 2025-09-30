-- Test data for Invoice Generator
-- Provides default company and sample data for development

-- Insert default company
INSERT INTO companies (id, name, email, phone, address_line1, city, state_province, postal_code, country, invoice_prefix)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Default Company',
    'billing@company.com',
    '+1 555-0100',
    '123 Business St',
    'New York',
    'NY',
    '10001',
    'USA',
    'INV'
) ON CONFLICT (id) DO NOTHING;

-- Insert test clients
INSERT INTO clients (id, company_id, name, email, phone, address_line1, city, state_province, postal_code, country)
VALUES
    ('11111111-1111-1111-1111-111111111111', '00000000-0000-0000-0000-000000000001',
     'Acme Corp', 'billing@acme.com', '+1 555-0101',
     '456 Main St', 'San Francisco', 'CA', '94105', 'USA'),
    ('22222222-2222-2222-2222-222222222222', '00000000-0000-0000-0000-000000000001',
     'Tech Solutions Inc', 'accounts@techsolutions.com', '+1 555-0102',
     '789 Tech Way', 'Austin', 'TX', '78701', 'USA')
ON CONFLICT (id) DO NOTHING;

-- Insert default tax rates
INSERT INTO tax_rates (company_id, name, rate, is_default)
VALUES 
    ('00000000-0000-0000-0000-000000000001', 'Standard Tax', 10.00, true),
    ('00000000-0000-0000-0000-000000000001', 'Reduced Tax', 5.00, false),
    ('00000000-0000-0000-0000-000000000001', 'No Tax', 0.00, false)
ON CONFLICT DO NOTHING;