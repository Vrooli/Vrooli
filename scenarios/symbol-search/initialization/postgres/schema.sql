-- Symbol Search Database Schema
-- Creates tables for Unicode character data with optimized indexing

-- Unicode Categories table
CREATE TABLE IF NOT EXISTS categories (
    code VARCHAR(2) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT
);

-- Unicode Blocks table  
CREATE TABLE IF NOT EXISTS character_blocks (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    start_codepoint INTEGER NOT NULL,
    end_codepoint INTEGER NOT NULL,
    description TEXT
);

-- Main Characters table
CREATE TABLE IF NOT EXISTS characters (
    codepoint VARCHAR(10) PRIMARY KEY,     -- U+1F600
    decimal INTEGER NOT NULL UNIQUE,       -- 128512
    name TEXT NOT NULL,                    -- GRINNING FACE
    category VARCHAR(2) NOT NULL,          -- So (Symbol, other)
    block VARCHAR(50) NOT NULL,            -- Emoticons
    unicode_version VARCHAR(10) NOT NULL,  -- 6.1
    description TEXT,                      -- Optional extended description
    html_entity VARCHAR(20),               -- &#128512;
    css_content VARCHAR(20),               -- \1F600
    properties JSONB                       -- Additional Unicode properties
);

-- Foreign key constraints
ALTER TABLE characters 
ADD CONSTRAINT fk_characters_category 
FOREIGN KEY (category) REFERENCES categories(code);

ALTER TABLE characters 
ADD CONSTRAINT fk_characters_block 
FOREIGN KEY (block) REFERENCES character_blocks(name);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_characters_name ON characters USING gin(to_tsvector('english', name));
CREATE INDEX IF NOT EXISTS idx_characters_name_trgm ON characters USING gin(name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_characters_description ON characters USING gin(to_tsvector('english', coalesce(description, '')));
CREATE INDEX IF NOT EXISTS idx_characters_category ON characters(category);
CREATE INDEX IF NOT EXISTS idx_characters_block ON characters(block);
CREATE INDEX IF NOT EXISTS idx_characters_unicode_version ON characters(unicode_version);
CREATE INDEX IF NOT EXISTS idx_characters_decimal ON characters(decimal);
CREATE INDEX IF NOT EXISTS idx_characters_search_composite ON characters(category, block, unicode_version);

-- Full-text search index combining name and description
CREATE INDEX IF NOT EXISTS idx_characters_fulltext ON characters 
USING gin(to_tsvector('english', name || ' ' || coalesce(description, '')));

-- Enable trigram extension for fuzzy matching
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Analyze tables for query optimization
ANALYZE categories;
ANALYZE character_blocks;
ANALYZE characters;