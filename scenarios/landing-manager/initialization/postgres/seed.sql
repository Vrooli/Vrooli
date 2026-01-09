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
-- Field names match UI component expectations (title/subtitle, not headline/subheadline)
INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
(1, 'hero', '{"title": "Build Landing Pages in Minutes", "subtitle": "Create beautiful, conversion-optimized landing pages with A/B testing and analytics built-in", "cta_text": "Get Started Free", "cta_url": "/signup"}', 1, TRUE),
(1, 'features', '{"title": "Everything You Need", "subtitle": "Powerful tools to build, test, and optimize your landing pages.", "features": [{"title": "A/B Testing", "description": "Test variants and optimize conversions", "icon": "zap"}, {"title": "Analytics", "description": "Track visitor behavior and metrics", "icon": "shield"}, {"title": "Stripe Integration", "description": "Accept payments instantly", "icon": "sparkles"}]}', 2, TRUE),
(1, 'pricing', '{"title": "Simple Pricing", "subtitle": "Choose the plan that fits your needs. No hidden fees.", "tiers": [{"name": "Starter", "price": "$29", "description": "Perfect for small projects", "features": ["5 landing pages", "Basic analytics", "Email support"], "cta_text": "Start Free Trial", "cta_url": "/checkout?plan=starter"}, {"name": "Pro", "price": "$99", "description": "For growing businesses", "features": ["Unlimited pages", "Advanced analytics", "Priority support", "Custom domains"], "cta_text": "Get Started", "cta_url": "/checkout?plan=pro", "highlighted": true}]}', 3, TRUE),
(1, 'cta', '{"title": "Ready to Launch Your Landing Page?", "subtitle": "Join thousands of marketers using Landing Manager", "cta_text": "Start Building Now", "cta_url": "/signup"}', 4, TRUE),
(1, 'footer', '{"company_name": "Landing Manager", "tagline": "Build better landing pages", "links": [{"label": "Home", "url": "/"}, {"label": "Pricing", "url": "#pricing"}, {"label": "Contact", "url": "/contact"}]}', 99, TRUE)
ON CONFLICT DO NOTHING;

-- Insert default content sections for variant-a (same structure, all variants need content)
INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
(2, 'hero', '{"title": "Build Landing Pages in Minutes", "subtitle": "Create beautiful, conversion-optimized landing pages with A/B testing and analytics built-in", "cta_text": "Get Started Free", "cta_url": "/signup"}', 1, TRUE),
(2, 'features', '{"title": "Everything You Need", "subtitle": "Powerful tools to build, test, and optimize your landing pages.", "features": [{"title": "A/B Testing", "description": "Test variants and optimize conversions", "icon": "zap"}, {"title": "Analytics", "description": "Track visitor behavior and metrics", "icon": "shield"}, {"title": "Stripe Integration", "description": "Accept payments instantly", "icon": "sparkles"}]}', 2, TRUE),
(2, 'pricing', '{"title": "Simple Pricing", "subtitle": "Choose the plan that fits your needs. No hidden fees.", "tiers": [{"name": "Starter", "price": "$29", "description": "Perfect for small projects", "features": ["5 landing pages", "Basic analytics", "Email support"], "cta_text": "Start Free Trial", "cta_url": "/checkout?plan=starter"}, {"name": "Pro", "price": "$99", "description": "For growing businesses", "features": ["Unlimited pages", "Advanced analytics", "Priority support", "Custom domains"], "cta_text": "Get Started", "cta_url": "/checkout?plan=pro", "highlighted": true}]}', 3, TRUE),
(2, 'cta', '{"title": "Ready to Launch Your Landing Page?", "subtitle": "Join thousands of marketers using Landing Manager", "cta_text": "Start Building Now", "cta_url": "/signup"}', 4, TRUE),
(2, 'footer', '{"company_name": "Landing Manager", "tagline": "Build better landing pages", "links": [{"label": "Home", "url": "/"}, {"label": "Pricing", "url": "#pricing"}, {"label": "Contact", "url": "/contact"}]}', 99, TRUE)
ON CONFLICT DO NOTHING;
