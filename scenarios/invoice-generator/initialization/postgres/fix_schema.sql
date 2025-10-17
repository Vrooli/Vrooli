-- Fix missing columns in existing invoices table
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS last_reminder_sent DATE;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS recurring_invoice_id UUID;

-- Ensure recurring_invoices table exists (it's in the schema but might not be applied)
-- The full table is already in schema.sql, this is just to ensure it exists