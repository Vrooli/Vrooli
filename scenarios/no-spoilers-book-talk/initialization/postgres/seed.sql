-- No Spoilers Book Talk Seed Data
-- Initial data for testing and demonstration

-- Insert sample books for testing (using classic public domain works)
INSERT INTO books (id, title, author, file_path, file_type, total_chunks, total_words, processing_status, processed_at, metadata) VALUES
(
    uuid_generate_v4(),
    'Pride and Prejudice',
    'Jane Austen',
    'data/books/pride-and-prejudice.txt',
    'txt',
    245,
    122000,
    'completed',
    NOW(),
    '{"isbn": "9780141439518", "publication_year": 1813, "genre": "Romance", "public_domain": true, "difficulty_level": "intermediate"}'::jsonb
),
(
    uuid_generate_v4(),
    'The Adventures of Sherlock Holmes',
    'Arthur Conan Doyle',
    'data/books/sherlock-holmes.txt', 
    'txt',
    189,
    95000,
    'completed',
    NOW(),
    '{"isbn": "9780486474915", "publication_year": 1892, "genre": "Mystery", "public_domain": true, "difficulty_level": "intermediate"}'::jsonb
),
(
    uuid_generate_v4(),
    'Alice''s Adventures in Wonderland',
    'Lewis Carroll',
    'data/books/alice-in-wonderland.txt',
    'txt',
    78,
    26500,
    'completed', 
    NOW(),
    '{"isbn": "9780486275437", "publication_year": 1865, "genre": "Fantasy", "public_domain": true, "difficulty_level": "beginner"}'::jsonb
);

-- Get the book IDs for setting up sample user progress
DO $$ 
DECLARE
    pride_prejudice_id UUID;
    sherlock_id UUID;
    alice_id UUID;
BEGIN
    -- Get book IDs
    SELECT id INTO pride_prejudice_id FROM books WHERE title = 'Pride and Prejudice';
    SELECT id INTO sherlock_id FROM books WHERE title = 'The Adventures of Sherlock Holmes';
    SELECT id INTO alice_id FROM books WHERE title = 'Alice''s Adventures in Wonderland';

    -- Sample user progress for different reading scenarios
    INSERT INTO user_progress (book_id, user_id, current_position, position_type, position_value, reading_session_count, total_reading_time_minutes, notes) VALUES
    (
        pride_prejudice_id,
        'demo-user-1',
        58, -- About 1/4 through the book
        'chunk',
        58,
        12,
        480, -- 8 hours of reading
        'Fascinating character development. Elizabeth''s first impressions of Darcy are so strong!'
    ),
    (
        sherlock_id,
        'demo-user-1', 
        45, -- About 1/4 through
        'chunk',
        45,
        8,
        320, -- 5+ hours
        'Love the deduction methods. Watson''s perspective makes Holmes more relatable.'
    ),
    (
        alice_id,
        'demo-user-2',
        78, -- Finished the book
        'chunk', 
        78,
        5,
        150, -- 2.5 hours
        'Completed! The wordplay and logic puzzles were delightful. Perfect for discussing literary nonsense.'
    ),
    (
        pride_prejudice_id,
        'book-club-member',
        120, -- Halfway through
        'chunk',
        120,
        15,
        600, -- 10 hours
        'Reading for book club. Taking detailed notes on social commentary and class dynamics.'
    );

    -- Sample book collections
    INSERT INTO book_collections (user_id, name, description, book_ids, is_public) VALUES
    (
        'demo-user-1',
        'Classic Literature Journey',
        'Working through essential classic novels',
        jsonb_build_array(pride_prejudice_id::text, sherlock_id::text),
        false
    ),
    (
        'demo-user-2', 
        'Quick Reads & Fantasy',
        'Shorter books and fantasy adventures',
        jsonb_build_array(alice_id::text),
        true
    ),
    (
        'book-club-member',
        'Monthly Book Club Selections',
        'Books we''re reading in our monthly club',
        jsonb_build_array(pride_prejudice_id::text),
        false
    );

    -- Sample conversation history (for demonstration)
    INSERT INTO conversations (book_id, user_id, user_message, ai_response, user_position, position_type, context_chunks_used, sources_referenced, position_boundary_respected, processing_time_ms) VALUES
    (
        pride_prejudice_id,
        'demo-user-1',
        'What do you think about Elizabeth''s first impression of Mr. Darcy? Was she being too judgmental?',
        'Elizabeth''s initial reaction to Darcy is quite understandable given the circumstances of their first meeting at the Meryton ball. When she overhears him saying she is "tolerable, but not handsome enough to tempt" him, it''s natural that she would form a negative impression. However, Austen is showing us how first impressions can be deceiving - both Elizabeth''s judgment of Darcy and our understanding of his character will evolve. At this point in your reading, Elizabeth is responding to what she sees as pride and arrogance, which reflects the social dynamics of the time.',
        58,
        'chunk',
        '[12, 15, 23, 28]'::jsonb,
        '["Mr. Darcy walked off; and Elizabeth remained with no very cordial feelings toward him", "tolerable, but not handsome enough to tempt me"]'::jsonb,
        true,
        1450
    ),
    (
        sherlock_id,
        'demo-user-1',
        'How does Holmes'' observation method work? Can you explain his approach without spoiling future cases?',
        'Based on what you''ve read so far, Holmes'' method relies on careful observation of physical details that others overlook. He notices things like wear patterns on clothing, calluses on hands, ink stains, or mud on shoes - then uses logical deduction to understand what these details reveal about a person''s profession, habits, or recent activities. Watson often serves as our guide, showing us how remarkable these seemingly simple observations are. Holmes combines this acute attention to detail with broad knowledge across many fields, allowing him to make connections that seem almost magical but are actually quite logical once explained.',
        45,
        'chunk',
        '[8, 12, 19, 22, 31]'::jsonb,
        '["You see, but you do not observe", "From a drop of water, a logician could infer the possibility of an Atlantic or a Niagara"]'::jsonb,
        true,
        1680
    );

    -- Sample reading analytics
    INSERT INTO reading_analytics (book_id, user_id, analytics_date, pages_read, reading_time_minutes, questions_asked, reading_pace, comprehension_score) VALUES
    (pride_prejudice_id, 'demo-user-1', CURRENT_DATE - INTERVAL '1 day', 25, 120, 3, 12.5, 0.85),
    (pride_prejudice_id, 'demo-user-1', CURRENT_DATE - INTERVAL '2 days', 30, 100, 2, 18.0, 0.90),
    (sherlock_id, 'demo-user-1', CURRENT_DATE - INTERVAL '1 day', 20, 80, 4, 15.0, 0.75),
    (alice_id, 'demo-user-2', CURRENT_DATE - INTERVAL '3 days', 40, 90, 1, 26.7, 0.95);

END $$;

-- Create some sample book chunks (these would normally be created by the book processing pipeline)
DO $$
DECLARE
    pride_prejudice_id UUID;
BEGIN
    SELECT id INTO pride_prejudice_id FROM books WHERE title = 'Pride and Prejudice';
    
    -- Sample chunks for testing position-based queries
    INSERT INTO book_chunks (book_id, chunk_number, chapter_number, position_start, position_end, word_count, character_count, content_preview, embedding_status) VALUES
    (pride_prejudice_id, 0, 1, 0, 1200, 198, 1200, 'It is a truth universally acknowledged, that a single man in possession of a good fortune, must be in want of a wife. However little known the feelings...', 'completed'),
    (pride_prejudice_id, 1, 1, 1201, 2400, 205, 1199, 'When Jane and Elizabeth were alone, the former, who had been cautious in her praise of Mr. Bingley before, expressed to her sister just how very much...', 'completed'),
    (pride_prejudice_id, 2, 1, 2401, 3600, 189, 1199, 'Not all that Mrs. Bennet, however, with the assistance of her five daughters, could ask on the subject, was sufficient to draw from her husband any...', 'completed');
END $$;

-- Add indexes for sample queries
CREATE INDEX IF NOT EXISTS idx_sample_conversations ON conversations(book_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sample_analytics ON reading_analytics(user_id, analytics_date DESC);

-- Update book statistics based on sample data
UPDATE books SET 
    total_chunks = (SELECT COUNT(*) FROM book_chunks WHERE book_chunks.book_id = books.id),
    updated_at = NOW()
WHERE processing_status = 'completed';