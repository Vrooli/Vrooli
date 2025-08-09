-- Brand Manager Database Seed Data

-- Insert brand templates
INSERT INTO templates (id, name, category, style_config, color_schemes, font_families, prompt_templates, is_active) VALUES 
(
  '550e8400-e29b-41d4-a716-446655440001',
  'Modern Tech',
  'technology',
  '{"design_style": "minimalist", "color_scheme": "monochromatic", "typography": "sans-serif", "logo_style": "geometric"}',
  '[
    {"name": "Deep Blue", "primary": "#1e40af", "secondary": "#3b82f6", "accent": "#60a5fa", "background": "#f8fafc", "text": "#0f172a"},
    {"name": "Emerald", "primary": "#047857", "secondary": "#059669", "accent": "#10b981", "background": "#f0fdf4", "text": "#064e3b"}
  ]',
  '[
    {"heading": "Inter", "body": "Inter", "accent": "JetBrains Mono"},
    {"heading": "Poppins", "body": "Open Sans", "accent": "Fira Code"}
  ]',
  '{
    "slogan": "Generate a concise, tech-forward slogan for {brandName} that emphasizes innovation, reliability, and user experience. Keep it under 6 words.",
    "description": "Write a professional brand description for {brandName} in the {industry} industry. Focus on cutting-edge technology, user-centric design, and market leadership. 2-3 sentences.",
    "ad_copy": "Create compelling ad copy for {brandName} that highlights technical excellence and user benefits. Include a clear call-to-action. 50-75 words.",
    "logo_prompt": "minimalist logo design, geometric shapes, {brandColors.primary} and {brandColors.secondary}, clean lines, modern typography, scalable vector, professional"
  }',
  true
),
(
  '550e8400-e29b-41d4-a716-446655440002',
  'Creative Agency',
  'design',
  '{"design_style": "bold", "color_scheme": "vibrant", "typography": "mixed", "logo_style": "artistic"}',
  '[
    {"name": "Rainbow", "primary": "#8b5cf6", "secondary": "#ec4899", "accent": "#f59e0b", "background": "#fefefe", "text": "#18181b"},
    {"name": "Sunset", "primary": "#dc2626", "secondary": "#ea580c", "accent": "#facc15", "background": "#fffbeb", "text": "#7c2d12"}
  ]',
  '[
    {"heading": "Montserrat", "body": "Source Sans Pro", "accent": "Pacifico"},
    {"heading": "Playfair Display", "body": "Lato", "accent": "Dancing Script"}
  ]',
  '{
    "slogan": "Create an inspiring, creative slogan for {brandName} that captures artistic vision and innovation. Make it memorable and energetic.",
    "description": "Write an engaging brand description for {brandName} that showcases creativity, artistic excellence, and client collaboration. Emphasize unique vision and results.",
    "ad_copy": "Develop vibrant ad copy for {brandName} that inspires creativity and showcases artistic capabilities. Use energetic language with strong visual appeal.",
    "logo_prompt": "creative artistic logo, dynamic composition, {brandColors.primary} gradient, expressive design, contemporary art style, memorable symbol"
  }',
  true
),
(
  '550e8400-e29b-41d4-a716-446655440003',
  'Professional Services',
  'business',
  '{"design_style": "classic", "color_scheme": "conservative", "typography": "serif", "logo_style": "traditional"}',
  '[
    {"name": "Navy Trust", "primary": "#1e3a8a", "secondary": "#3730a3", "accent": "#6366f1", "background": "#f9fafb", "text": "#111827"},
    {"name": "Forest Professional", "primary": "#14532d", "secondary": "#166534", "accent": "#22c55e", "background": "#f7fee7", "text": "#0f2d1a"}
  ]',
  '[
    {"heading": "Merriweather", "body": "Source Sans Pro", "accent": "Merriweather Sans"},
    {"heading": "Playfair Display", "body": "Inter", "accent": "Source Serif Pro"}
  ]',
  '{
    "slogan": "Generate a trustworthy, professional slogan for {brandName} that conveys expertise, reliability, and client success. Focus on trust and results.",
    "description": "Write a professional brand description for {brandName} that establishes authority, experience, and client-focused service. Emphasize track record and expertise.",
    "ad_copy": "Create authoritative ad copy for {brandName} that builds trust and demonstrates expertise. Include client benefits and professional credentials.",
    "logo_prompt": "professional logo design, traditional elements, {brandColors.primary} monogram, clean typography, corporate identity, timeless design"
  }',
  true
);

-- Insert sample brands for demonstration
INSERT INTO brands (id, name, short_name, slogan, ad_copy, description, brand_colors, logo_url, favicon_url, assets, metadata) VALUES 
(
  '650e8400-e29b-41d4-a716-446655440001',
  'TechFlow Solutions',
  'TechFlow',
  'Innovation at Speed',
  'Transform your business with cutting-edge technology solutions. TechFlow delivers scalable, secure, and user-friendly applications that drive growth and efficiency. Ready to accelerate your digital transformation? Contact us today.',
  'TechFlow Solutions specializes in enterprise-grade software development and digital transformation. We combine innovative technology with user-centric design to deliver solutions that scale with your business.',
  '{"primary": "#1e40af", "secondary": "#3b82f6", "accent": "#60a5fa", "background": "#f8fafc", "text": "#0f172a"}',
  'http://minio:9000/brand-logos/techflow/logo.svg',
  'http://minio:9000/brand-logos/techflow/favicon.ico',
  '[
    {"type": "logo", "format": "svg", "url": "http://minio:9000/brand-logos/techflow/logo.svg", "size": "512x512"},
    {"type": "logo", "format": "png", "url": "http://minio:9000/brand-logos/techflow/logo.png", "size": "512x512"},
    {"type": "favicon", "format": "ico", "url": "http://minio:9000/brand-logos/techflow/favicon.ico", "size": "32x32"}
  ]',
  '{"template_id": "550e8400-e29b-41d4-a716-446655440001", "generated_at": "2024-01-15T10:00:00Z", "version": "1.0"}'
),
(
  '650e8400-e29b-41d4-a716-446655440002',
  'Creative Spark Studio',
  'Creative',
  'Ignite Your Vision',
  'Bring your boldest ideas to life with Creative Spark Studio. Our award-winning team crafts stunning visual experiences that captivate audiences and drive results. From brand identity to digital campaigns, we make magic happen. Spark your success today!',
  'Creative Spark Studio is a full-service creative agency specializing in brand identity, digital design, and marketing campaigns. We turn innovative ideas into compelling visual stories.',
  '{"primary": "#8b5cf6", "secondary": "#ec4899", "accent": "#f59e0b", "background": "#fefefe", "text": "#18181b"}',
  'http://minio:9000/brand-logos/creativespark/logo.svg',
  'http://minio:9000/brand-logos/creativespark/favicon.ico',
  '[
    {"type": "logo", "format": "svg", "url": "http://minio:9000/brand-logos/creativespark/logo.svg", "size": "512x512"},
    {"type": "logo", "format": "png", "url": "http://minio:9000/brand-logos/creativespark/logo.png", "size": "512x512"},
    {"type": "favicon", "format": "ico", "url": "http://minio:9000/brand-logos/creativespark/favicon.ico", "size": "32x32"}
  ]',
  '{"template_id": "550e8400-e29b-41d4-a716-446655440002", "generated_at": "2024-01-15T11:30:00Z", "version": "1.0"}'
);

-- Insert sample campaigns
INSERT INTO campaigns (id, name, description, brand_ids, status) VALUES 
(
  '750e8400-e29b-41d4-a716-446655440001',
  'Q1 2024 Launch Campaign',
  'Coordinated brand launch campaign for new technology clients',
  '{"650e8400-e29b-41d4-a716-446655440001"}',
  'active'
),
(
  '750e8400-e29b-41d4-a716-446655440002', 
  'Creative Portfolio Refresh',
  'Brand refresh campaign for creative agency clients',
  '{"650e8400-e29b-41d4-a716-446655440002"}',
  'planning'
);

-- Insert sample exports
INSERT INTO exports (id, brand_id, export_type, target_app, export_path, manifest, status) VALUES 
(
  '850e8400-e29b-41d4-a716-446655440001',
  '650e8400-e29b-41d4-a716-446655440001',
  'package',
  'demo-react-app',
  '/exports/techflow-demo-package',
  '{
    "name": "TechFlow Solutions Brand Package",
    "version": "1.0.0",
    "brand_id": "650e8400-e29b-41d4-a716-446655440001",
    "files": {
      "manifest.json": "App manifest with brand data",
      "favicon.ico": "32x32 favicon",
      "logo.svg": "Scalable logo",
      "logo.png": "512x512 logo",
      "brand-config.json": "Brand configuration data"
    },
    "generated_at": "2024-01-15T12:00:00Z"
  }',
  'completed'
);

-- Insert sample integration requests
INSERT INTO integration_requests (id, brand_id, target_app_path, integration_type, status, request_payload) VALUES 
(
  '950e8400-e29b-41d4-a716-446655440001',
  '650e8400-e29b-41d4-a716-446655440001',
  '/generated-apps/demo-app',
  'full',
  'completed',
  '{
    "brandId": "650e8400-e29b-41d4-a716-446655440001",
    "targetAppPath": "/generated-apps/demo-app",
    "integrationType": "full",
    "createBackup": true,
    "requestedAt": "2024-01-15T13:00:00Z"
  }'
),
(
  '950e8400-e29b-41d4-a716-446655440002',
  '650e8400-e29b-41d4-a716-446655440002',
  '/generated-apps/creative-portfolio',
  'assets-only',
  'processing',
  '{
    "brandId": "650e8400-e29b-41d4-a716-446655440002",
    "targetAppPath": "/generated-apps/creative-portfolio",
    "integrationType": "assets-only",
    "createBackup": true,
    "requestedAt": "2024-01-15T14:00:00Z"
  }'
);

-- Create integration metrics table for analytics
CREATE TABLE IF NOT EXISTS integration_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id TEXT NOT NULL,
    brand_id UUID REFERENCES brands(id),
    status VARCHAR(50),
    duration_seconds INTEGER,
    files_modified INTEGER DEFAULT 0,
    errors_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample metrics
INSERT INTO integration_metrics (id, session_id, brand_id, status, duration_seconds, files_modified, errors_count) VALUES 
(
  'a50e8400-e29b-41d4-a716-446655440001',
  'claude_int_techflow_001',
  '650e8400-e29b-41d4-a716-446655440001', 
  'completed',
  127,
  8,
  0
),
(
  'a50e8400-e29b-41d4-a716-446655440002',
  'claude_int_creative_001',
  '650e8400-e29b-41d4-a716-446655440002',
  'processing',
  NULL,
  3,
  0
);

-- Create activity log table for tracking operations
CREATE TABLE IF NOT EXISTS activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID REFERENCES brands(id),
    action VARCHAR(100) NOT NULL,
    message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample activity log entries
INSERT INTO activity_log (brand_id, action, message, metadata) VALUES 
(
  '650e8400-e29b-41d4-a716-446655440001',
  'brand_created',
  'TechFlow Solutions brand created successfully',
  '{"template": "modern-tech", "generation_time": 23.5}'
),
(
  '650e8400-e29b-41d4-a716-446655440001',
  'integration_started',
  'Started full integration into demo-app',
  '{"target_path": "/generated-apps/demo-app", "type": "full"}'
),
(
  '650e8400-e29b-41d4-a716-446655440001',
  'integration_completed',
  'Successfully integrated TechFlow Solutions brand',
  '{"files_modified": 8, "duration": 127}'
),
(
  '650e8400-e29b-41d4-a716-446655440002',
  'brand_created',
  'Creative Spark Studio brand created successfully',
  '{"template": "creative-agency", "generation_time": 31.2}'
),
(
  '650e8400-e29b-41d4-a716-446655440002',
  'integration_started',
  'Started assets-only integration into creative-portfolio',
  '{"target_path": "/generated-apps/creative-portfolio", "type": "assets-only"}'
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_activity_log_brand ON activity_log(brand_id);
CREATE INDEX IF NOT EXISTS idx_activity_log_action ON activity_log(action);
CREATE INDEX IF NOT EXISTS idx_activity_log_created ON activity_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_integration_metrics_session ON integration_metrics(session_id);
CREATE INDEX IF NOT EXISTS idx_integration_metrics_brand ON integration_metrics(brand_id);

-- Update timestamps for existing records
UPDATE brands SET updated_at = CURRENT_TIMESTAMP;
UPDATE campaigns SET updated_at = CURRENT_TIMESTAMP;