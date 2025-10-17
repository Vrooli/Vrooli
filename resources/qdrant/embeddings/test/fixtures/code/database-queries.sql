-- Email Database Schema and Queries
-- Contains database operations for email management system

-- Create emails table with proper indexing
CREATE TABLE IF NOT EXISTS emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id VARCHAR(255) NOT NULL, -- Gmail/Outlook message ID
    provider_type VARCHAR(50) NOT NULL CHECK (provider_type IN ('gmail', 'outlook')),
    
    -- Email content
    subject TEXT NOT NULL,
    body_text TEXT,
    body_html TEXT,
    sender_email VARCHAR(320) NOT NULL,
    sender_name VARCHAR(255),
    
    -- Classification and metadata
    category VARCHAR(50) DEFAULT 'normal' CHECK (category IN ('important', 'normal', 'spam', 'promotional')),
    priority_score INTEGER DEFAULT 50 CHECK (priority_score BETWEEN 0 AND 100),
    ai_confidence DECIMAL(3,2) DEFAULT 0.00 CHECK (ai_confidence BETWEEN 0 AND 1),
    
    -- Timestamps
    received_at TIMESTAMP WITH TIME ZONE NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Indexing for performance
    UNIQUE(user_id, provider_id, provider_type)
);

-- Create indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_emails_user_received ON emails(user_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_emails_category ON emails(category, user_id);
CREATE INDEX IF NOT EXISTS idx_emails_priority ON emails(priority_score DESC, user_id);
CREATE INDEX IF NOT EXISTS idx_emails_provider ON emails(provider_type, provider_id);
CREATE INDEX IF NOT EXISTS idx_emails_search ON emails USING gin(to_tsvector('english', subject || ' ' || coalesce(body_text, '')));

-- Fetch user emails with pagination and filtering
-- Parameters: $1=user_id, $2=category, $3=search_term, $4=limit, $5=offset
SELECT 
    id,
    subject,
    sender_email,
    sender_name,
    category,
    priority_score,
    ai_confidence,
    received_at,
    processed_at,
    -- Truncate body for list view
    LEFT(body_text, 200) as body_preview
FROM emails 
WHERE user_id = $1
    AND ($2 IS NULL OR category = $2)
    AND ($3 IS NULL OR to_tsvector('english', subject || ' ' || coalesce(body_text, '')) @@ plainto_tsquery('english', $3))
ORDER BY received_at DESC
LIMIT $4 OFFSET $5;

-- Update email category and AI confidence
-- Parameters: $1=email_id, $2=category, $3=ai_confidence, $4=user_id
UPDATE emails 
SET 
    category = $2,
    ai_confidence = $3,
    updated_at = NOW()
WHERE id = $1 AND user_id = $4
RETURNING id, category, ai_confidence, updated_at;

-- Get email analytics for dashboard
-- Parameters: $1=user_id, $2=start_date, $3=end_date
WITH email_stats AS (
    SELECT 
        category,
        COUNT(*) as email_count,
        AVG(priority_score) as avg_priority,
        AVG(ai_confidence) as avg_confidence
    FROM emails 
    WHERE user_id = $1 
        AND received_at >= $2 
        AND received_at <= $3
    GROUP BY category
),
daily_volumes AS (
    SELECT 
        DATE(received_at) as email_date,
        COUNT(*) as daily_count
    FROM emails
    WHERE user_id = $1 
        AND received_at >= $2 
        AND received_at <= $3
    GROUP BY DATE(received_at)
    ORDER BY email_date
),
response_times AS (
    SELECT 
        AVG(EXTRACT(EPOCH FROM (processed_at - received_at))/60) as avg_processing_minutes
    FROM emails
    WHERE user_id = $1 
        AND received_at >= $2 
        AND received_at <= $3
        AND processed_at IS NOT NULL
)
SELECT 
    json_build_object(
        'category_breakdown', (SELECT json_agg(email_stats.*) FROM email_stats),
        'daily_volumes', (SELECT json_agg(daily_volumes.*) FROM daily_volumes),
        'avg_processing_time', (SELECT avg_processing_minutes FROM response_times)
    ) as analytics;

-- Batch update email categories using AI predictions
-- Parameters: $1=user_id, $2=email_ids_array, $3=categories_array, $4=confidences_array
UPDATE emails 
SET 
    category = batch_data.category,
    ai_confidence = batch_data.confidence,
    updated_at = NOW()
FROM (
    SELECT 
        unnest($2::UUID[]) as email_id,
        unnest($3::TEXT[]) as category,
        unnest($4::DECIMAL[]) as confidence
) as batch_data
WHERE emails.id = batch_data.email_id 
    AND emails.user_id = $1
RETURNING emails.id, emails.category, emails.ai_confidence;

-- Find duplicate emails (same subject and sender within 1 hour)
-- Parameters: $1=user_id
SELECT 
    e1.id as original_id,
    e1.received_at as original_time,
    array_agg(e2.id) as duplicate_ids,
    COUNT(e2.id) as duplicate_count
FROM emails e1
JOIN emails e2 ON (
    e1.user_id = e2.user_id 
    AND e1.subject = e2.subject
    AND e1.sender_email = e2.sender_email
    AND e1.id != e2.id
    AND ABS(EXTRACT(EPOCH FROM (e1.received_at - e2.received_at))) < 3600
)
WHERE e1.user_id = $1
    AND e1.received_at > e2.received_at  -- Keep oldest as original
GROUP BY e1.id, e1.received_at
HAVING COUNT(e2.id) > 0
ORDER BY duplicate_count DESC;

-- Create materialized view for email search performance
CREATE MATERIALIZED VIEW IF NOT EXISTS email_search_index AS
SELECT 
    id,
    user_id,
    subject,
    sender_email,
    category,
    received_at,
    setweight(to_tsvector('english', subject), 'A') ||
    setweight(to_tsvector('english', coalesce(body_text, '')), 'B') ||
    setweight(to_tsvector('english', sender_name), 'C') as search_vector
FROM emails;

-- Create index on materialized view
CREATE INDEX IF NOT EXISTS idx_email_search_vector ON email_search_index USING gin(search_vector);

-- Refresh search index (run periodically)
REFRESH MATERIALIZED VIEW email_search_index;

-- Archive old emails to separate table
-- Parameters: $1=cutoff_date
WITH archived_emails AS (
    DELETE FROM emails 
    WHERE received_at < $1
    RETURNING *
)
INSERT INTO emails_archive 
SELECT * FROM archived_emails;

-- Clean up orphaned email attachments
-- Parameters: $1=user_id
DELETE FROM email_attachments 
WHERE user_id = $1 
    AND email_id NOT IN (
        SELECT id FROM emails WHERE user_id = $1
    );