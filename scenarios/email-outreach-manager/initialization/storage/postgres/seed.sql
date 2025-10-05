-- Seed data for Email Outreach Manager
-- Provides sample templates and campaigns for testing

-- Sample templates
INSERT INTO templates (id, name, subject, html_content, text_content, style_category, created_at)
VALUES
    ('11111111-1111-1111-1111-111111111111',
     'Product Launch Template',
     'Introducing Our New Product',
     '<html><body><h1>Exciting News!</h1><p>Dear {{name}},</p><p>We are thrilled to announce our new product.</p></body></html>',
     'Exciting News!\n\nDear {{name}},\n\nWe are thrilled to announce our new product.',
     'professional',
     NOW()),
    ('22222222-2222-2222-2222-222222222222',
     'Newsletter Template',
     'Monthly Newsletter - {{month}}',
     '<html><body><h1>Monthly Update</h1><p>Hi {{name}},</p><p>Here is what happened this month.</p></body></html>',
     'Monthly Update\n\nHi {{name}},\n\nHere is what happened this month.',
     'friendly',
     NOW())
ON CONFLICT (id) DO NOTHING;

-- Sample campaign
INSERT INTO campaigns (id, name, description, template_id, status, created_at, total_recipients, sent_count)
VALUES
    ('33333333-3333-3333-3333-333333333333',
     'Q4 Product Launch',
     'Launch campaign for new AI features',
     '11111111-1111-1111-1111-111111111111',
     'draft',
     NOW(),
     0,
     0)
ON CONFLICT (id) DO NOTHING;
