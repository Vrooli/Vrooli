-- Seed data for Funnel Builder
-- Sample templates and test data

SET search_path TO funnel_builder;

-- Insert sample funnel templates
INSERT INTO funnel_templates (name, slug, description, category, template_data, metrics) VALUES
(
    'Lead Generation Funnel',
    'lead-generation',
    'Capture high-quality leads with qualifying questions',
    'Marketing',
    '{
        "name": "Lead Generation Funnel",
        "steps": [
            {
                "type": "quiz",
                "title": "What are you looking for?",
                "content": {
                    "question": "What brings you here today?",
                    "options": [
                        {"id": "1", "text": "Growing my business", "icon": "📈"},
                        {"id": "2", "text": "Finding new customers", "icon": "🎯"},
                        {"id": "3", "text": "Improving conversions", "icon": "🚀"}
                    ]
                }
            },
            {
                "type": "form",
                "title": "Get Your Free Guide",
                "content": {
                    "fields": [
                        {"type": "text", "label": "Full Name", "required": true},
                        {"type": "email", "label": "Email", "required": true},
                        {"type": "tel", "label": "Phone", "required": false}
                    ],
                    "submitText": "Get Free Access"
                }
            },
            {
                "type": "cta",
                "title": "Success!",
                "content": {
                    "headline": "Check Your Email!",
                    "subheadline": "We sent your free guide to your inbox.",
                    "buttonText": "Go to Dashboard",
                    "buttonUrl": "/dashboard"
                }
            }
        ],
        "settings": {
            "theme": {"primaryColor": "#0ea5e9"},
            "progressBar": true
        }
    }'::jsonb,
    '{"avg_conversion_rate": 32, "typical_completion_time": 165}'::jsonb
),
(
    'Product Launch Funnel',
    'product-launch',
    'Build excitement and drive sales for new products',
    'Sales',
    '{
        "name": "Product Launch Funnel",
        "steps": [
            {
                "type": "content",
                "title": "Introducing Our Latest Innovation",
                "content": {
                    "headline": "The Future is Here",
                    "body": "Discover how our latest product can transform your business.",
                    "buttonText": "Learn More"
                }
            },
            {
                "type": "quiz",
                "title": "Personalize Your Experience",
                "content": {
                    "question": "What is your biggest challenge?",
                    "options": [
                        {"id": "1", "text": "Time management", "icon": "⏰"},
                        {"id": "2", "text": "Team collaboration", "icon": "👥"},
                        {"id": "3", "text": "Project tracking", "icon": "📋"}
                    ]
                }
            },
            {
                "type": "form",
                "title": "Get Early Access",
                "content": {
                    "fields": [
                        {"type": "text", "label": "Full Name", "required": true},
                        {"type": "email", "label": "Work Email", "required": true},
                        {"type": "text", "label": "Company", "required": true},
                        {"type": "select", "label": "Company Size", "options": ["1-10", "11-50", "51-200", "200+"]}
                    ],
                    "submitText": "Get Early Access"
                }
            },
            {
                "type": "cta",
                "title": "Welcome to the Future",
                "content": {
                    "headline": "You are on the List!",
                    "subheadline": "Get ready for exclusive early access.",
                    "buttonText": "Share with Friends",
                    "buttonUrl": "/share",
                    "urgency": "Only 100 early access spots remaining!"
                }
            }
        ],
        "settings": {
            "theme": {"primaryColor": "#8b5cf6"},
            "progressBar": true,
            "exitIntent": true
        }
    }'::jsonb,
    '{"avg_conversion_rate": 28, "typical_completion_time": 210}'::jsonb
),
(
    'Interactive Quiz Funnel',
    'quiz-funnel',
    'Engage visitors with personalized quiz results',
    'Engagement',
    '{
        "name": "Interactive Quiz Funnel",
        "steps": [
            {
                "type": "content",
                "title": "Discover Your Type",
                "content": {
                    "headline": "What Kind of Entrepreneur Are You?",
                    "body": "Take our 2-minute quiz to discover your entrepreneurial style and get personalized recommendations.",
                    "buttonText": "Start Quiz"
                }
            },
            {
                "type": "quiz",
                "title": "Question 1 of 3",
                "content": {
                    "question": "How do you prefer to work?",
                    "options": [
                        {"id": "1", "text": "Solo, focused and independent", "icon": "🧘"},
                        {"id": "2", "text": "With a small team", "icon": "👥"},
                        {"id": "3", "text": "Leading large groups", "icon": "👑"}
                    ]
                }
            },
            {
                "type": "quiz",
                "title": "Question 2 of 3",
                "content": {
                    "question": "What motivates you most?",
                    "options": [
                        {"id": "1", "text": "Financial freedom", "icon": "💰"},
                        {"id": "2", "text": "Making an impact", "icon": "🌍"},
                        {"id": "3", "text": "Creative expression", "icon": "🎨"}
                    ]
                }
            },
            {
                "type": "quiz",
                "title": "Question 3 of 3",
                "content": {
                    "question": "How do you handle risk?",
                    "options": [
                        {"id": "1", "text": "Carefully calculated", "icon": "📊"},
                        {"id": "2", "text": "Moderate risk-taker", "icon": "⚖️"},
                        {"id": "3", "text": "All-in adventurer", "icon": "🎲"}
                    ]
                }
            },
            {
                "type": "form",
                "title": "Get Your Results",
                "content": {
                    "fields": [
                        {"type": "email", "label": "Email", "required": true, "placeholder": "Get your personalized report"},
                        {"type": "text", "label": "First Name", "required": true}
                    ],
                    "submitText": "Get My Results"
                }
            },
            {
                "type": "content",
                "title": "Your Results",
                "content": {
                    "headline": "You are a Visionary Leader!",
                    "body": "Based on your answers, you have the traits of a visionary entrepreneur. Check your email for your complete personalized report with actionable insights.",
                    "buttonText": "Share Your Results"
                }
            }
        ],
        "settings": {
            "theme": {"primaryColor": "#f59e0b"},
            "progressBar": true
        }
    }'::jsonb,
    '{"avg_conversion_rate": 38, "typical_completion_time": 180}'::jsonb
),
(
    'Webinar Registration',
    'webinar-registration',
    'Get attendees for your online events and webinars',
    'Events',
    '{
        "name": "Webinar Registration",
        "steps": [
            {
                "type": "content",
                "title": "Free Live Training",
                "content": {
                    "headline": "How to Scale Your Business to 7 Figures",
                    "body": "Join our exclusive live training where we reveal the exact strategies that helped 500+ entrepreneurs reach their goals.",
                    "media": {"type": "image", "url": "/webinar-hero.jpg"},
                    "buttonText": "Reserve My Spot"
                }
            },
            {
                "type": "form",
                "title": "Register for Free",
                "content": {
                    "fields": [
                        {"type": "text", "label": "First Name", "required": true},
                        {"type": "text", "label": "Last Name", "required": true},
                        {"type": "email", "label": "Email", "required": true},
                        {"type": "tel", "label": "Phone (for reminders)", "required": false}
                    ],
                    "submitText": "Save My Seat"
                }
            },
            {
                "type": "cta",
                "title": "You are Registered!",
                "content": {
                    "headline": "Your Spot is Reserved!",
                    "subheadline": "Check your email for login details and calendar invite.",
                    "buttonText": "Add to Calendar",
                    "buttonUrl": "/calendar",
                    "urgency": "Limited to 500 attendees - 423 spots taken!"
                }
            }
        ],
        "settings": {
            "theme": {"primaryColor": "#10b981"},
            "progressBar": true,
            "exitIntent": true
        }
    }'::jsonb,
    '{"avg_conversion_rate": 45, "typical_completion_time": 120}'::jsonb
);

-- Create a demo funnel for testing
INSERT INTO funnels (name, slug, description, status, settings) VALUES
(
    'Demo Sales Funnel',
    'demo-sales-funnel',
    'A demonstration funnel for testing the platform',
    'active',
    '{"theme": {"primaryColor": "#0ea5e9"}, "progressBar": true}'::jsonb
);

-- Add steps to the demo funnel
INSERT INTO funnel_steps (funnel_id, type, position, title, content)
SELECT 
    f.id,
    'quiz',
    0,
    'What are your goals?',
    '{
        "question": "What is your primary business goal?",
        "options": [
            {"id": "growth", "text": "Scale my business", "icon": "📈"},
            {"id": "efficiency", "text": "Improve efficiency", "icon": "⚡"},
            {"id": "revenue", "text": "Increase revenue", "icon": "💵"}
        ]
    }'::jsonb
FROM funnels f WHERE f.slug = 'demo-sales-funnel';

INSERT INTO funnel_steps (funnel_id, type, position, title, content)
SELECT 
    f.id,
    'form',
    1,
    'Get Your Custom Plan',
    '{
        "fields": [
            {"id": "name", "type": "text", "label": "Full Name", "required": true},
            {"id": "email", "type": "email", "label": "Email", "required": true},
            {"id": "company", "type": "text", "label": "Company Name", "required": false}
        ],
        "submitText": "Get My Free Plan"
    }'::jsonb
FROM funnels f WHERE f.slug = 'demo-sales-funnel';