-- Seed Data for Landing Manager

-- Insert default admin user (OT-P0-008: ADMIN-AUTH)
-- Email: admin@localhost
-- Password: changeme123
-- IMPORTANT: Change this password in production!
INSERT INTO admin_users (email, password_hash) VALUES
('admin@localhost', '$2a$10$nhmpbhFPQUZZwEH.qaYHCeiKBWDvr8z5Z7eM4v62MmNwm.0N.5xeG')
ON CONFLICT (email) DO NOTHING;

-- Insert default A/B testing variants (OT-P0-014 through OT-P0-018)
INSERT INTO variants (slug, name, description, weight, status) VALUES
('control', 'Control (Original)', 'Original landing page design', 50, 'active'),
('variant-a', 'Variant A', 'Experimental variant A', 50, 'active')
ON CONFLICT (slug) DO NOTHING;

-- Insert default content sections for control variant (OT-P0-012, OT-P0-013)
INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
(1, 'hero', '{"headline": "Build Landing Pages in Minutes", "subheadline": "Create beautiful, conversion-optimized landing pages with A/B testing and analytics built-in", "cta_text": "Get Started Free", "cta_url": "/signup", "background_color": "#0f172a"}', 1, TRUE),
(1, 'features', '{"title": "Everything You Need", "items": [{"title": "A/B Testing", "description": "Test variants and optimize conversions", "icon": "Zap"}, {"title": "Analytics", "description": "Track visitor behavior and metrics", "icon": "BarChart"}, {"title": "Stripe Integration", "description": "Accept payments instantly", "icon": "CreditCard"}]}', 2, TRUE),
(1, 'pricing', '{"title": "Simple Pricing", "plans": [{"name": "Starter", "price": "$29", "features": ["5 landing pages", "Basic analytics", "Email support"], "cta_text": "Start Free Trial"}, {"name": "Pro", "price": "$99", "features": ["Unlimited pages", "Advanced analytics", "Priority support", "Custom domains"], "cta_text": "Get Started", "highlighted": true}]}', 3, TRUE),
(1, 'cta', '{"headline": "Ready to Launch Your Landing Page?", "subheadline": "Join thousands of marketers using Landing Manager", "cta_text": "Start Building Now", "cta_url": "/signup"}', 4, TRUE)
ON CONFLICT DO NOTHING;
