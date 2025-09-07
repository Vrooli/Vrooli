-- Data Structurer Seed Data
-- Provides initial schema templates and examples for common use cases

-- Insert common schema templates that other scenarios can use immediately

-- Contact/Person Schema Template (for email-outreach-manager, personal-relationship-manager)
INSERT INTO schema_templates (name, category, description, schema_definition, example_data, tags) VALUES 
(
    'contact-person',
    'contacts',
    'Standard person/contact schema with name, email, phone, and social profiles',
    '{
        "type": "object",
        "properties": {
            "first_name": {"type": "string", "description": "First name"},
            "last_name": {"type": "string", "description": "Last name"},
            "email": {"type": "string", "format": "email", "description": "Primary email address"},
            "phone": {"type": "string", "description": "Phone number"},
            "company": {"type": "string", "description": "Company name"},
            "title": {"type": "string", "description": "Job title"},
            "linkedin_url": {"type": "string", "format": "uri", "description": "LinkedIn profile URL"},
            "location": {"type": "string", "description": "Geographic location"},
            "notes": {"type": "string", "description": "Additional notes"}
        },
        "required": ["first_name", "email"]
    }',
    '{
        "first_name": "John",
        "last_name": "Smith",
        "email": "john.smith@example.com",
        "phone": "+1-555-123-4567",
        "company": "Acme Corp",
        "title": "Software Engineer",
        "linkedin_url": "https://linkedin.com/in/johnsmith",
        "location": "San Francisco, CA",
        "notes": "Met at tech conference"
    }',
    ARRAY['contacts', 'people', 'networking', 'crm']
);

-- Company/Organization Schema Template (for competitor-change-monitor, research-assistant)
INSERT INTO schema_templates (name, category, description, schema_definition, example_data, tags) VALUES 
(
    'company-organization',
    'business',
    'Company or organization information with key business details',
    '{
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Company name"},
            "domain": {"type": "string", "description": "Primary domain/website"},
            "industry": {"type": "string", "description": "Industry or sector"},
            "size": {"type": "string", "enum": ["startup", "small", "medium", "large", "enterprise"], "description": "Company size"},
            "founded_year": {"type": "integer", "description": "Year founded"},
            "headquarters": {"type": "string", "description": "HQ location"},
            "description": {"type": "string", "description": "Company description"},
            "revenue": {"type": "string", "description": "Annual revenue"},
            "employees": {"type": "integer", "description": "Number of employees"},
            "funding": {"type": "string", "description": "Funding stage or amount"},
            "competitors": {"type": "array", "items": {"type": "string"}, "description": "Main competitors"},
            "technologies": {"type": "array", "items": {"type": "string"}, "description": "Technologies used"}
        },
        "required": ["name", "domain"]
    }',
    '{
        "name": "TechCorp Inc",
        "domain": "techcorp.com",
        "industry": "Software",
        "size": "medium",
        "founded_year": 2015,
        "headquarters": "Austin, TX",
        "description": "B2B SaaS platform for project management",
        "revenue": "$50M ARR",
        "employees": 200,
        "funding": "Series B - $25M",
        "competitors": ["Asana", "Monday.com", "Notion"],
        "technologies": ["React", "Node.js", "PostgreSQL", "AWS"]
    }',
    ARRAY['companies', 'business', 'competitive-intelligence', 'research']
);

-- Document Metadata Schema Template (for document-manager, research-assistant)
INSERT INTO schema_templates (name, category, description, schema_definition, example_data, tags) VALUES 
(
    'document-metadata',
    'documents',
    'Document metadata extraction for content management and search',
    '{
        "type": "object",
        "properties": {
            "title": {"type": "string", "description": "Document title"},
            "author": {"type": "string", "description": "Document author"},
            "document_type": {"type": "string", "enum": ["report", "article", "whitepaper", "manual", "presentation", "contract", "other"], "description": "Type of document"},
            "date_created": {"type": "string", "format": "date", "description": "Creation date"},
            "subject": {"type": "string", "description": "Primary subject or topic"},
            "keywords": {"type": "array", "items": {"type": "string"}, "description": "Key terms and topics"},
            "summary": {"type": "string", "description": "Brief summary of content"},
            "language": {"type": "string", "description": "Document language"},
            "page_count": {"type": "integer", "description": "Number of pages"},
            "word_count": {"type": "integer", "description": "Approximate word count"},
            "category": {"type": "string", "description": "Content category"},
            "confidentiality": {"type": "string", "enum": ["public", "internal", "confidential", "restricted"], "description": "Confidentiality level"}
        },
        "required": ["title", "document_type"]
    }',
    '{
        "title": "Q3 2024 Market Analysis Report",
        "author": "Research Team",
        "document_type": "report",
        "date_created": "2024-10-15",
        "subject": "Market trends and competitive analysis",
        "keywords": ["market analysis", "competition", "trends", "Q3", "2024"],
        "summary": "Comprehensive analysis of market conditions and competitive landscape for Q3 2024",
        "language": "English",
        "page_count": 45,
        "word_count": 12500,
        "category": "Business Intelligence",
        "confidentiality": "internal"
    }',
    ARRAY['documents', 'content', 'metadata', 'search']
);

-- Product/Service Schema Template (for e-commerce, inventory, competitive analysis)
INSERT INTO schema_templates (name, category, description, schema_definition, example_data, tags) VALUES 
(
    'product-service',
    'products',
    'Product or service information for catalogs and competitive analysis',
    '{
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Product/service name"},
            "description": {"type": "string", "description": "Product description"},
            "category": {"type": "string", "description": "Product category"},
            "price": {"type": "number", "description": "Price in USD"},
            "currency": {"type": "string", "default": "USD", "description": "Currency code"},
            "availability": {"type": "string", "enum": ["available", "limited", "out_of_stock", "discontinued"], "description": "Availability status"},
            "features": {"type": "array", "items": {"type": "string"}, "description": "Key features"},
            "specifications": {"type": "object", "description": "Technical specifications"},
            "tags": {"type": "array", "items": {"type": "string"}, "description": "Product tags"},
            "vendor": {"type": "string", "description": "Vendor or manufacturer"},
            "sku": {"type": "string", "description": "Stock keeping unit"},
            "rating": {"type": "number", "minimum": 0, "maximum": 5, "description": "Average customer rating"},
            "review_count": {"type": "integer", "description": "Number of reviews"}
        },
        "required": ["name", "price"]
    }',
    '{
        "name": "Premium Project Management Software",
        "description": "All-in-one project management solution with advanced analytics",
        "category": "Software/SaaS",
        "price": 29.99,
        "currency": "USD",
        "availability": "available",
        "features": ["Task Management", "Time Tracking", "Analytics", "API Integration"],
        "specifications": {
            "users": "Unlimited",
            "projects": "Unlimited", 
            "storage": "100GB",
            "support": "24/7"
        },
        "tags": ["project-management", "saas", "collaboration", "analytics"],
        "vendor": "ProductCorp",
        "sku": "PM-PREM-001",
        "rating": 4.5,
        "review_count": 1247
    }',
    ARRAY['products', 'services', 'ecommerce', 'competitive-analysis']
);

-- Event/Meeting Schema Template (for calendar, scheduling systems)
INSERT INTO schema_templates (name, category, description, schema_definition, example_data, tags) VALUES 
(
    'event-meeting',
    'events',
    'Event or meeting information for calendars and scheduling',
    '{
        "type": "object",
        "properties": {
            "title": {"type": "string", "description": "Event title"},
            "description": {"type": "string", "description": "Event description"},
            "date": {"type": "string", "format": "date", "description": "Event date"},
            "start_time": {"type": "string", "format": "time", "description": "Start time"},
            "end_time": {"type": "string", "format": "time", "description": "End time"},
            "location": {"type": "string", "description": "Event location"},
            "organizer": {"type": "string", "description": "Event organizer"},
            "attendees": {"type": "array", "items": {"type": "string"}, "description": "List of attendees"},
            "event_type": {"type": "string", "enum": ["meeting", "conference", "workshop", "social", "other"], "description": "Type of event"},
            "priority": {"type": "string", "enum": ["low", "medium", "high", "critical"], "description": "Event priority"},
            "status": {"type": "string", "enum": ["planned", "confirmed", "cancelled", "completed"], "description": "Event status"},
            "recurring": {"type": "boolean", "description": "Is this a recurring event"},
            "reminder": {"type": "integer", "description": "Reminder time in minutes before event"}
        },
        "required": ["title", "date", "start_time"]
    }',
    '{
        "title": "Weekly Team Standup",
        "description": "Weekly sync meeting to discuss progress and blockers",
        "date": "2024-09-06",
        "start_time": "09:00",
        "end_time": "09:30",
        "location": "Conference Room A / Zoom",
        "organizer": "Team Lead",
        "attendees": ["Alice", "Bob", "Charlie", "Diana"],
        "event_type": "meeting",
        "priority": "medium",
        "status": "confirmed",
        "recurring": true,
        "reminder": 15
    }',
    ARRAY['events', 'meetings', 'calendar', 'scheduling']
);

-- Financial Transaction Schema Template (for expense tracking, financial analysis)
INSERT INTO schema_templates (name, category, description, schema_definition, example_data, tags) VALUES 
(
    'financial-transaction',
    'finance',
    'Financial transaction data for expense tracking and analysis',
    '{
        "type": "object",
        "properties": {
            "date": {"type": "string", "format": "date", "description": "Transaction date"},
            "amount": {"type": "number", "description": "Transaction amount"},
            "currency": {"type": "string", "default": "USD", "description": "Currency code"},
            "description": {"type": "string", "description": "Transaction description"},
            "category": {"type": "string", "description": "Expense category"},
            "payment_method": {"type": "string", "enum": ["cash", "credit_card", "debit_card", "bank_transfer", "check", "other"], "description": "Payment method"},
            "vendor": {"type": "string", "description": "Vendor or merchant name"},
            "account": {"type": "string", "description": "Account used for transaction"},
            "transaction_type": {"type": "string", "enum": ["expense", "income", "transfer", "refund"], "description": "Type of transaction"},
            "tax_deductible": {"type": "boolean", "description": "Is this tax deductible"},
            "receipt_url": {"type": "string", "format": "uri", "description": "URL to receipt image"},
            "tags": {"type": "array", "items": {"type": "string"}, "description": "Transaction tags"}
        },
        "required": ["date", "amount", "description"]
    }',
    '{
        "date": "2024-09-06",
        "amount": 45.67,
        "currency": "USD",
        "description": "Office supplies - notebooks and pens",
        "category": "Office Expenses",
        "payment_method": "credit_card",
        "vendor": "Office Depot",
        "account": "Business Credit Card",
        "transaction_type": "expense",
        "tax_deductible": true,
        "receipt_url": "https://example.com/receipts/12345.jpg",
        "tags": ["office", "supplies", "business"]
    }',
    ARRAY['finance', 'expenses', 'accounting', 'transactions']
);

-- Research Citation Schema Template (for research-assistant, academic work)
INSERT INTO schema_templates (name, category, description, schema_definition, example_data, tags) VALUES 
(
    'research-citation',
    'research',
    'Academic citation and reference information for research management',
    '{
        "type": "object",
        "properties": {
            "title": {"type": "string", "description": "Publication title"},
            "authors": {"type": "array", "items": {"type": "string"}, "description": "List of authors"},
            "publication_year": {"type": "integer", "description": "Year of publication"},
            "journal": {"type": "string", "description": "Journal or conference name"},
            "volume": {"type": "string", "description": "Volume number"},
            "issue": {"type": "string", "description": "Issue number"},
            "pages": {"type": "string", "description": "Page range"},
            "doi": {"type": "string", "description": "Digital Object Identifier"},
            "abstract": {"type": "string", "description": "Publication abstract"},
            "keywords": {"type": "array", "items": {"type": "string"}, "description": "Research keywords"},
            "url": {"type": "string", "format": "uri", "description": "URL to publication"},
            "publication_type": {"type": "string", "enum": ["journal", "conference", "book", "thesis", "preprint", "other"], "description": "Type of publication"},
            "research_field": {"type": "string", "description": "Primary research field"}
        },
        "required": ["title", "authors", "publication_year"]
    }',
    '{
        "title": "Machine Learning Approaches to Document Classification",
        "authors": ["Dr. Jane Smith", "Prof. John Doe", "Dr. Alice Johnson"],
        "publication_year": 2024,
        "journal": "Journal of Artificial Intelligence Research",
        "volume": "78",
        "issue": "2",
        "pages": "123-145",
        "doi": "10.1613/jair.1.12345",
        "abstract": "This paper presents novel approaches to document classification using transformer-based models...",
        "keywords": ["machine learning", "document classification", "transformers", "NLP"],
        "url": "https://jair.org/papers/12345",
        "publication_type": "journal",
        "research_field": "Computer Science"
    }',
    ARRAY['research', 'academic', 'citations', 'publications']
);

-- Update usage counts to 0 for all templates (since they're new)
UPDATE schema_templates SET usage_count = 0;

-- Insert a default general schema for unstructured text extraction
INSERT INTO schemas (name, description, schema_definition, example_data, created_by) VALUES 
(
    'general-text-extraction',
    'General purpose schema for extracting key information from any text document',
    '{
        "type": "object",
        "properties": {
            "title": {"type": "string", "description": "Document or content title"},
            "main_topics": {"type": "array", "items": {"type": "string"}, "description": "Primary topics discussed"},
            "key_points": {"type": "array", "items": {"type": "string"}, "description": "Main points or takeaways"},
            "entities": {
                "type": "object",
                "properties": {
                    "people": {"type": "array", "items": {"type": "string"}, "description": "People mentioned"},
                    "organizations": {"type": "array", "items": {"type": "string"}, "description": "Organizations mentioned"},
                    "locations": {"type": "array", "items": {"type": "string"}, "description": "Locations mentioned"},
                    "dates": {"type": "array", "items": {"type": "string"}, "description": "Important dates mentioned"}
                }
            },
            "summary": {"type": "string", "description": "Brief summary of the content"},
            "sentiment": {"type": "string", "enum": ["positive", "negative", "neutral"], "description": "Overall sentiment"},
            "content_type": {"type": "string", "description": "Type of content (email, article, report, etc.)"}
        },
        "required": ["title", "summary"]
    }',
    '{
        "title": "Product Launch Strategy Meeting",
        "main_topics": ["market research", "timeline", "budget allocation", "team assignments"],
        "key_points": ["Launch date set for Q1 2025", "Budget approved at $500K", "Marketing team leads campaign"],
        "entities": {
            "people": ["Sarah Johnson", "Mike Chen", "Lisa Rodriguez"],
            "organizations": ["Marketing Department", "Product Team"],
            "locations": ["San Francisco office"],
            "dates": ["Q1 2025", "December 15th", "January 30th"]
        },
        "summary": "Team meeting to finalize product launch strategy with approved budget and timeline",
        "sentiment": "positive",
        "content_type": "meeting_notes"
    }',
    'system'
);

-- Create some example processing jobs (for demo purposes)
INSERT INTO processing_jobs (schema_id, input_type, input_data, status, total_items, priority) 
SELECT 
    id,
    'text',
    'Sample text for processing demonstration',
    'completed',
    1,
    5
FROM schemas WHERE name = 'general-text-extraction'
LIMIT 1;