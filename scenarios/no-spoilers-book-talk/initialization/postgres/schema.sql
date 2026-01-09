-- No Spoilers Book Talk Database Schema
-- Manages book metadata, user progress, and conversation history

-- Extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Books table: Core book information and metadata
CREATE TABLE books (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    author VARCHAR(300),
    file_path TEXT NOT NULL,
    file_type VARCHAR(20) NOT NULL, -- txt, epub, pdf, etc.
    file_size_bytes BIGINT,
    total_chunks INTEGER DEFAULT 0,
    total_words INTEGER DEFAULT 0,
    total_characters INTEGER DEFAULT 0,
    chapters JSONB DEFAULT '[]'::jsonb, -- Array of chapter info {title, start_position, end_position}
    metadata JSONB DEFAULT '{}'::jsonb, -- Additional book metadata (ISBN, publication date, etc.)
    processing_status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    processing_error TEXT,
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- User progress tracking for each book
CREATE TABLE user_progress (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL, -- User identifier
    current_position INTEGER DEFAULT 0, -- Current reading position (chunk number)
    position_type VARCHAR(20) DEFAULT 'chunk', -- chunk, chapter, page, percentage
    position_value NUMERIC DEFAULT 0, -- Actual position value (chapter 5, page 100, 45.5%)
    last_read_chunk_id INTEGER, -- Last chunk the user has read
    reading_session_count INTEGER DEFAULT 0,
    total_reading_time_minutes INTEGER DEFAULT 0,
    notes TEXT, -- User's reading notes
    favorite_quotes JSONB DEFAULT '[]'::jsonb,
    reading_goals JSONB DEFAULT '{}'::jsonb, -- Reading goals and preferences
    updated_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure one progress record per user per book
    UNIQUE(book_id, user_id)
);

CREATE TABLE book_talk_conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL,
    user_message TEXT NOT NULL,
    ai_response TEXT NOT NULL,
    user_position INTEGER NOT NULL, -- User's position when asking question
    position_type VARCHAR(20) NOT NULL, -- Position type at time of question
    context_chunks_used JSONB DEFAULT '[]'::jsonb, -- Array of chunk IDs used for context
    sources_referenced JSONB DEFAULT '[]'::jsonb, -- Specific text passages referenced
    position_boundary_respected BOOLEAN DEFAULT true, -- Whether spoiler prevention was enforced
    response_quality_score NUMERIC(3,2), -- Optional response quality tracking
    processing_time_ms INTEGER, -- Response generation time
    conversation_metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Book chunks metadata (complementing Qdrant vector storage)
CREATE TABLE book_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    chunk_number INTEGER NOT NULL, -- Sequential chunk number (0-indexed)
    chapter_number INTEGER, -- Chapter this chunk belongs to
    position_start INTEGER NOT NULL, -- Starting character position in original text
    position_end INTEGER NOT NULL, -- Ending character position in original text
    word_count INTEGER DEFAULT 0,
    character_count INTEGER DEFAULT 0,
    content_preview TEXT, -- First 200 characters for debugging/preview
    embedding_status VARCHAR(20) DEFAULT 'pending', -- pending, completed, failed
    qdrant_point_id UUID, -- Reference to Qdrant vector point ID
    chunk_metadata JSONB DEFAULT '{}'::jsonb, -- Additional chunk-specific metadata
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure unique chunk numbering per book
    UNIQUE(book_id, chunk_number)
);

-- Reading analytics for insights and improvements
CREATE TABLE reading_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL,
    analytics_date DATE DEFAULT CURRENT_DATE,
    pages_read INTEGER DEFAULT 0,
    reading_time_minutes INTEGER DEFAULT 0,
    questions_asked INTEGER DEFAULT 0,
    concepts_discussed JSONB DEFAULT '[]'::jsonb,
    difficult_sections JSONB DEFAULT '[]'::jsonb, -- Sections where user asked many questions
    reading_pace NUMERIC(5,2), -- Pages per hour
    comprehension_score NUMERIC(3,2), -- Derived from question quality
    analytics_metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- One analytics record per user per book per day
    UNIQUE(book_id, user_id, analytics_date)
);

-- Book collections/lists for organization
CREATE TABLE book_collections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT false,
    book_ids JSONB DEFAULT '[]'::jsonb, -- Array of book UUIDs
    collection_metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance optimization
CREATE INDEX idx_books_title ON books(title);
CREATE INDEX idx_books_author ON books(author);
CREATE INDEX idx_books_processing_status ON books(processing_status);
CREATE INDEX idx_books_created_at ON books(created_at);

CREATE INDEX idx_user_progress_book_user ON user_progress(book_id, user_id);
CREATE INDEX idx_user_progress_user_id ON user_progress(user_id);
CREATE INDEX idx_user_progress_updated_at ON user_progress(updated_at);

CREATE INDEX idx_book_talk_conversations_book_id ON book_talk_conversations(book_id);
CREATE INDEX idx_book_talk_conversations_user_id ON book_talk_conversations(user_id);
CREATE INDEX idx_book_talk_conversations_book_user ON book_talk_conversations(book_id, user_id);
CREATE INDEX idx_book_talk_conversations_created_at ON book_talk_conversations(created_at);

CREATE INDEX idx_book_chunks_book_id ON book_chunks(book_id);
CREATE INDEX idx_book_chunks_book_chunk_num ON book_chunks(book_id, chunk_number);
CREATE INDEX idx_book_chunks_chapter ON book_chunks(book_id, chapter_number);
CREATE INDEX idx_book_chunks_embedding_status ON book_chunks(embedding_status);

CREATE INDEX idx_reading_analytics_user_book ON reading_analytics(user_id, book_id);
CREATE INDEX idx_reading_analytics_date ON reading_analytics(analytics_date);

CREATE INDEX idx_book_collections_user_id ON book_collections(user_id);
CREATE INDEX idx_book_collections_is_public ON book_collections(is_public);

-- Update triggers for timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_books_updated_at BEFORE UPDATE ON books
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_progress_updated_at BEFORE UPDATE ON user_progress
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_book_collections_updated_at BEFORE UPDATE ON book_collections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Views for common queries
CREATE VIEW book_progress_summary AS
SELECT 
    b.id as book_id,
    b.title,
    b.author,
    b.total_chunks,
    up.user_id,
    up.current_position,
    up.position_type,
    up.position_value,
    CASE 
        WHEN b.total_chunks > 0 THEN 
            ROUND((up.current_position::NUMERIC / b.total_chunks::NUMERIC) * 100, 2)
        ELSE 0 
    END as percentage_complete,
    up.reading_session_count,
    up.total_reading_time_minutes,
    up.updated_at as progress_updated_at
FROM books b
LEFT JOIN user_progress up ON b.id = up.book_id;

CREATE VIEW book_talk_conversation_stats AS
SELECT 
    book_id,
    user_id,
    COUNT(*) as total_conversations,
    AVG(processing_time_ms) as avg_response_time_ms,
    COUNT(*) FILTER (WHERE position_boundary_respected = true) as spoiler_safe_conversations,
    COUNT(*) FILTER (WHERE position_boundary_respected = false) as spoiler_risk_conversations,
    MAX(created_at) as last_conversation_at
FROM book_talk_conversations
GROUP BY book_id, user_id;

-- Comments for schema documentation
COMMENT ON TABLE books IS 'Core book information and processing status';
COMMENT ON TABLE user_progress IS 'Individual user reading progress for each book';
COMMENT ON TABLE book_talk_conversations IS 'Chat history between users and AI about book content';
COMMENT ON TABLE book_chunks IS 'Metadata for book text chunks stored in Qdrant vector database';
COMMENT ON TABLE reading_analytics IS 'Daily reading analytics for insights and progress tracking';
COMMENT ON TABLE book_collections IS 'User-created book lists and collections';

COMMENT ON COLUMN books.chapters IS 'JSON array of chapter metadata: [{title, start_position, end_position, word_count}]';
COMMENT ON COLUMN user_progress.current_position IS 'Current reading position as chunk number (0-indexed)';
COMMENT ON COLUMN book_talk_conversations.context_chunks_used IS 'Array of chunk IDs that were used to generate the AI response';
COMMENT ON COLUMN book_talk_conversations.position_boundary_respected IS 'Whether the response avoided spoilers from future content';
