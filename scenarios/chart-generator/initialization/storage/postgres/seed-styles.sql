-- Chart Generator Default Styles Seed Data
-- Populates the database with professional, high-quality chart styles

-- Insert default professional chart styles
INSERT INTO chart_styles (id, name, category, description, color_palette, typography, spacing, animations, is_public, is_default, created_by)
VALUES 

-- Professional Corporate Style
(
    gen_random_uuid(),
    'Professional',
    'professional',
    'Clean, corporate styling perfect for business reports and presentations. Features subtle colors and clear typography.',
    '["#2563eb", "#64748b", "#f59e0b", "#10b981", "#ef4444", "#8b5cf6", "#06b6d4", "#84cc16"]',
    '{
        "fontFamily": "Inter, system-ui, sans-serif",
        "fontSize": {
            "title": 18,
            "axis": 12,
            "legend": 11,
            "tooltip": 10
        },
        "fontWeight": {
            "title": 600,
            "axis": 500,
            "legend": 400
        }
    }',
    '{
        "margin": {"top": 20, "right": 30, "bottom": 40, "left": 50},
        "padding": {"chart": 16, "legend": 12},
        "gridOpacity": 0.3,
        "borderWidth": 1
    }',
    '{
        "enabled": true,
        "duration": 750,
        "easing": "cubic-bezier(0.4, 0, 0.2, 1)",
        "stagger": 100
    }',
    true,
    true,
    'system'
),

-- Minimal Clean Style
(
    gen_random_uuid(),
    'Minimal',
    'minimal',
    'Ultra-clean, distraction-free design focusing on data clarity. Perfect for academic papers and technical documentation.',
    '["#374151", "#6b7280", "#9ca3af", "#d1d5db", "#f3f4f6", "#e5e7eb"]',
    '{
        "fontFamily": "system-ui, -apple-system, sans-serif",
        "fontSize": {
            "title": 16,
            "axis": 11,
            "legend": 10,
            "tooltip": 9
        },
        "fontWeight": {
            "title": 500,
            "axis": 400,
            "legend": 400
        }
    }',
    '{
        "margin": {"top": 16, "right": 20, "bottom": 32, "left": 40},
        "padding": {"chart": 12, "legend": 8},
        "gridOpacity": 0.2,
        "borderWidth": 0
    }',
    '{
        "enabled": false,
        "duration": 0,
        "easing": "linear",
        "stagger": 0
    }',
    true,
    false,
    'system'
),

-- Vibrant Presentation Style
(
    gen_random_uuid(),
    'Vibrant',
    'creative',
    'Bold, eye-catching colors perfect for marketing presentations and creative reports that need to make an impact.',
    '["#f59e0b", "#ef4444", "#8b5cf6", "#10b981", "#3b82f6", "#f97316", "#ec4899", "#84cc16"]',
    '{
        "fontFamily": "Inter, system-ui, sans-serif",
        "fontSize": {
            "title": 20,
            "axis": 13,
            "legend": 12,
            "tooltip": 11
        },
        "fontWeight": {
            "title": 700,
            "axis": 500,
            "legend": 500
        }
    }',
    '{
        "margin": {"top": 24, "right": 40, "bottom": 48, "left": 60},
        "padding": {"chart": 20, "legend": 16},
        "gridOpacity": 0.25,
        "borderWidth": 2
    }',
    '{
        "enabled": true,
        "duration": 1000,
        "easing": "cubic-bezier(0.68, -0.55, 0.265, 1.55)",
        "stagger": 150
    }',
    true,
    false,
    'system'
),

-- Dark Professional Style
(
    gen_random_uuid(),
    'Dark Professional',
    'professional',
    'Sophisticated dark theme perfect for modern dashboards and executive presentations. Easy on the eyes with high contrast.',
    '["#60a5fa", "#34d399", "#fbbf24", "#f87171", "#c084fc", "#22d3ee", "#a3e635", "#fb7185"]',
    '{
        "fontFamily": "Inter, system-ui, sans-serif",
        "fontSize": {
            "title": 18,
            "axis": 12,
            "legend": 11,
            "tooltip": 10
        },
        "fontWeight": {
            "title": 600,
            "axis": 500,
            "legend": 400
        },
        "color": "#f8fafc"
    }',
    '{
        "margin": {"top": 20, "right": 30, "bottom": 40, "left": 50},
        "padding": {"chart": 16, "legend": 12},
        "gridOpacity": 0.2,
        "borderWidth": 1,
        "backgroundColor": "#1e293b"
    }',
    '{
        "enabled": true,
        "duration": 800,
        "easing": "cubic-bezier(0.25, 0.46, 0.45, 0.94)",
        "stagger": 120
    }',
    true,
    false,
    'system'
),

-- Corporate Brand Style
(
    gen_random_uuid(),
    'Corporate Blue',
    'corporate',
    'Conservative corporate styling with blue color scheme, perfect for financial reports and formal business documentation.',
    '["#1e40af", "#3b82f6", "#60a5fa", "#93c5fd", "#dbeafe", "#1f2937", "#4b5563", "#9ca3af"]',
    '{
        "fontFamily": "system-ui, -apple-system, BlinkMacSystemFont, sans-serif",
        "fontSize": {
            "title": 17,
            "axis": 12,
            "legend": 11,
            "tooltip": 10
        },
        "fontWeight": {
            "title": 600,
            "axis": 500,
            "legend": 400
        }
    }',
    '{
        "margin": {"top": 18, "right": 28, "bottom": 36, "left": 48},
        "padding": {"chart": 14, "legend": 10},
        "gridOpacity": 0.28,
        "borderWidth": 1
    }',
    '{
        "enabled": true,
        "duration": 700,
        "easing": "cubic-bezier(0.4, 0, 0.2, 1)",
        "stagger": 80
    }',
    true,
    false,
    'system'
),

-- Scientific/Technical Style
(
    gen_random_uuid(),
    'Scientific',
    'technical',
    'Precise, technical styling optimized for scientific publications and research papers. Emphasizes clarity and accuracy.',
    '["#059669", "#0891b2", "#7c3aed", "#dc2626", "#ea580c", "#4338ca", "#be185d", "#166534"]',
    '{
        "fontFamily": "Georgia, Times, serif",
        "fontSize": {
            "title": 16,
            "axis": 11,
            "legend": 10,
            "tooltip": 9
        },
        "fontWeight": {
            "title": 500,
            "axis": 400,
            "legend": 400
        }
    }',
    '{
        "margin": {"top": 16, "right": 24, "bottom": 36, "left": 48},
        "padding": {"chart": 12, "legend": 8},
        "gridOpacity": 0.35,
        "borderWidth": 1
    }',
    '{
        "enabled": false,
        "duration": 0,
        "easing": "linear",
        "stagger": 0
    }',
    true,
    false,
    'system'
),

-- Financial Charts Style
(
    gen_random_uuid(),
    'Financial',
    'professional',
    'Specialized for financial data with red/green color coding and precise formatting. Ideal for stock charts and financial reports.',
    '["#10b981", "#ef4444", "#3b82f6", "#f59e0b", "#6b7280", "#1f2937", "#059669", "#dc2626"]',
    '{
        "fontFamily": "Monaco, Consolas, monospace",
        "fontSize": {
            "title": 16,
            "axis": 11,
            "legend": 10,
            "tooltip": 9
        },
        "fontWeight": {
            "title": 600,
            "axis": 500,
            "legend": 500
        }
    }',
    '{
        "margin": {"top": 16, "right": 32, "bottom": 32, "left": 56},
        "padding": {"chart": 12, "legend": 8},
        "gridOpacity": 0.4,
        "borderWidth": 1
    }',
    '{
        "enabled": true,
        "duration": 600,
        "easing": "cubic-bezier(0.25, 0.46, 0.45, 0.94)",
        "stagger": 50
    }',
    true,
    false,
    'system'
),

-- Colorful Creative Style
(
    gen_random_uuid(),
    'Creative Rainbow',
    'creative',
    'Playful, colorful design perfect for creative agencies, marketing materials, and presentations that need personality.',
    '["#ff6b6b", "#4ecdc4", "#45b7d1", "#96ceb4", "#feca57", "#ff9ff3", "#54a0ff", "#5f27cd"]',
    '{
        "fontFamily": "Inter, system-ui, sans-serif",
        "fontSize": {
            "title": 22,
            "axis": 13,
            "legend": 12,
            "tooltip": 11
        },
        "fontWeight": {
            "title": 700,
            "axis": 500,
            "legend": 500
        }
    }',
    '{
        "margin": {"top": 28, "right": 36, "bottom": 44, "left": 56},
        "padding": {"chart": 20, "legend": 16},
        "gridOpacity": 0.2,
        "borderWidth": 0
    }',
    '{
        "enabled": true,
        "duration": 1200,
        "easing": "cubic-bezier(0.68, -0.55, 0.265, 1.55)",
        "stagger": 200
    }',
    true,
    false,
    'system'
);

-- Insert some example chart templates using the styles
INSERT INTO chart_templates (id, name, chart_type, description, default_config, style_id, preview_data, is_public, created_by)
VALUES

-- Sales Performance Bar Chart Template
(
    gen_random_uuid(),
    'Quarterly Sales Performance',
    'bar',
    'Standard quarterly sales performance chart with professional styling',
    '{
        "title": "Quarterly Sales Performance",
        "x_axis_label": "Quarter",
        "y_axis_label": "Sales ($)",
        "show_legend": false,
        "show_grid": true,
        "show_values": true,
        "bar_spacing": 0.2
    }',
    (SELECT id FROM chart_styles WHERE name = 'Professional' LIMIT 1),
    '[
        {"x": "Q1 2024", "y": 125000},
        {"x": "Q2 2024", "y": 156000},
        {"x": "Q3 2024", "y": 189000},
        {"x": "Q4 2024", "y": 203000}
    ]',
    true,
    'system'
),

-- Revenue Trend Line Chart Template
(
    gen_random_uuid(),
    'Monthly Revenue Trend',
    'line',
    'Monthly revenue tracking with smooth trend line and professional appearance',
    '{
        "title": "Monthly Revenue Trend",
        "x_axis_label": "Month",
        "y_axis_label": "Revenue ($)",
        "show_legend": false,
        "show_grid": true,
        "line_width": 3,
        "show_points": true,
        "smooth_line": true
    }',
    (SELECT id FROM chart_styles WHERE name = 'Professional' LIMIT 1),
    '[
        {"x": "Jan", "y": 45000},
        {"x": "Feb", "y": 52000},
        {"x": "Mar", "y": 48000},
        {"x": "Apr", "y": 61000},
        {"x": "May", "y": 67000},
        {"x": "Jun", "y": 74000}
    ]',
    true,
    'system'
),

-- Market Share Pie Chart Template
(
    gen_random_uuid(),
    'Market Share Distribution',
    'pie',
    'Market share visualization with clear labels and vibrant colors',
    '{
        "title": "Market Share by Product Category",
        "show_legend": true,
        "show_percentages": true,
        "label_position": "outside",
        "slice_spacing": 2
    }',
    (SELECT id FROM chart_styles WHERE name = 'Vibrant' LIMIT 1),
    '[
        {"x": "Product A", "y": 35},
        {"x": "Product B", "y": 28},
        {"x": "Product C", "y": 22},
        {"x": "Product D", "y": 15}
    ]',
    true,
    'system'
),

-- Financial Performance Scatter Plot Template
(
    gen_random_uuid(),
    'Risk vs Return Analysis',
    'scatter',
    'Financial risk-return analysis with specialized financial styling',
    '{
        "title": "Risk vs Return Analysis",
        "x_axis_label": "Risk (%)",
        "y_axis_label": "Return (%)",
        "show_legend": false,
        "show_grid": true,
        "point_size": 6,
        "show_trend_line": true
    }',
    (SELECT id FROM chart_styles WHERE name = 'Financial' LIMIT 1),
    '[
        {"x": 5.2, "y": 8.1},
        {"x": 7.8, "y": 12.3},
        {"x": 12.1, "y": 15.7},
        {"x": 18.4, "y": 22.9},
        {"x": 25.6, "y": 28.4}
    ]',
    true,
    'system'
);

-- Initialize style usage stats for all styles
INSERT INTO style_usage_stats (style_id, style_name, usage_count, last_used_at, avg_generation_time_ms, success_rate)
SELECT 
    id,
    name,
    0,
    NULL,
    NULL,
    1.0
FROM chart_styles;

-- Log successful seed completion
INSERT INTO chart_generation_history (
    chart_instance_id,
    chart_type,
    data_points,
    style_used,
    generation_time_ms,
    success,
    client_info
) VALUES (
    'seed-initialization',
    'system',
    0,
    'system-setup',
    0,
    true,
    '{"source": "database-seed", "action": "initial-setup"}'
);

-- Display completion message
SELECT 
    'Chart Generator seed data installed successfully' AS status,
    COUNT(*) AS styles_created
FROM chart_styles 
WHERE created_by = 'system';

SELECT 
    'Chart templates created successfully' AS status,
    COUNT(*) AS templates_created
FROM chart_templates 
WHERE created_by = 'system';