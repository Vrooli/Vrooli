-- Game Dialog Generator Database Schema
-- Jungle Platformer Adventure Theme 🌿🎮

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Projects table - organize game development projects
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    characters TEXT[] DEFAULT '{}', -- Array of character UUIDs
    settings JSONB DEFAULT '{}',
    export_format VARCHAR(50) DEFAULT 'json',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Characters table - store game character definitions
CREATE TABLE IF NOT EXISTS characters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    personality_traits JSONB NOT NULL DEFAULT '{}', -- Big Five + custom traits
    background_story TEXT,
    speech_patterns JSONB DEFAULT '{}', -- Vocabulary, sentence structure, catchphrases
    relationships JSONB DEFAULT '{}', -- Relationships with other characters
    voice_profile JSONB DEFAULT '{}', -- Voice synthesis parameters
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Scenes table - define contexts for dialog generation
CREATE TABLE IF NOT EXISTS scenes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    context TEXT NOT NULL, -- Scene description and context
    setting JSONB DEFAULT '{}', -- Environmental details, atmosphere
    mood VARCHAR(100), -- overall emotional tone
    participants TEXT[] DEFAULT '{}', -- Character UUIDs in this scene
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Dialog lines table - store generated dialog with metadata
CREATE TABLE IF NOT EXISTS dialog_lines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    character_id UUID REFERENCES characters(id) ON DELETE CASCADE,
    scene_id UUID REFERENCES scenes(id) ON DELETE CASCADE,
    content TEXT NOT NULL, -- The actual dialog text
    emotion VARCHAR(100), -- Character's emotional state
    context_embedding_id VARCHAR(255), -- Reference to Qdrant vector
    audio_file_path VARCHAR(500), -- Path to generated audio file
    consistency_score DECIMAL(3,2), -- Character consistency score 0.00-1.00
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Character relationships table - model dynamic relationships
CREATE TABLE IF NOT EXISTS character_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    character1_id UUID REFERENCES characters(id) ON DELETE CASCADE,
    character2_id UUID REFERENCES characters(id) ON DELETE CASCADE,
    relationship_type VARCHAR(100), -- friend, enemy, romantic, mentor, etc.
    strength DECIMAL(3,2) CHECK (strength >= -1.00 AND strength <= 1.00), -- -1 to 1 scale
    context JSONB DEFAULT '{}', -- Additional relationship context
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(character1_id, character2_id)
);

-- Dialog templates table - reusable dialog patterns
CREATE TABLE IF NOT EXISTS dialog_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100), -- greeting, combat, discovery, etc.
    template TEXT NOT NULL, -- Template with placeholders
    variables JSONB DEFAULT '{}', -- Variable definitions and constraints
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Export jobs table - track dialog export operations
CREATE TABLE IF NOT EXISTS export_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    export_format VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    file_path VARCHAR(500),
    parameters JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_characters_name ON characters(name);
CREATE INDEX IF NOT EXISTS idx_characters_created_at ON characters(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dialog_lines_character_id ON dialog_lines(character_id);
CREATE INDEX IF NOT EXISTS idx_dialog_lines_scene_id ON dialog_lines(scene_id);
CREATE INDEX IF NOT EXISTS idx_dialog_lines_generated_at ON dialog_lines(generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_scenes_project_id ON scenes(project_id);
CREATE INDEX IF NOT EXISTS idx_character_relationships_char1 ON character_relationships(character1_id);
CREATE INDEX IF NOT EXISTS idx_character_relationships_char2 ON character_relationships(character2_id);
CREATE INDEX IF NOT EXISTS idx_export_jobs_project_id ON export_jobs(project_id);
CREATE INDEX IF NOT EXISTS idx_export_jobs_status ON export_jobs(status);

-- Create a function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to automatically update timestamps
DROP TRIGGER IF EXISTS update_characters_updated_at ON characters;
CREATE TRIGGER update_characters_updated_at 
    BEFORE UPDATE ON characters 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_character_relationships_updated_at ON character_relationships;
CREATE TRIGGER update_character_relationships_updated_at 
    BEFORE UPDATE ON character_relationships 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Views for common queries
CREATE OR REPLACE VIEW character_summary AS
SELECT 
    c.id,
    c.name,
    c.personality_traits,
    c.created_at,
    COUNT(dl.id) as dialog_count,
    AVG(dl.consistency_score) as avg_consistency_score
FROM characters c
LEFT JOIN dialog_lines dl ON c.id = dl.character_id
GROUP BY c.id, c.name, c.personality_traits, c.created_at;

CREATE OR REPLACE VIEW project_stats AS
SELECT 
    p.id,
    p.name,
    p.created_at,
    array_length(p.characters, 1) as character_count,
    COUNT(s.id) as scene_count,
    COUNT(dl.id) as total_dialog_count
FROM projects p
LEFT JOIN scenes s ON p.id = s.project_id
LEFT JOIN dialog_lines dl ON s.id = dl.scene_id
GROUP BY p.id, p.name, p.created_at, p.characters;

-- Insert comment for schema version tracking
COMMENT ON SCHEMA public IS 'Game Dialog Generator v1.0 - Jungle Platformer Theme 🌿🎮';
COMMENT ON TABLE characters IS 'Game character definitions with AI personality modeling';
COMMENT ON TABLE dialog_lines IS 'Generated character dialog with consistency tracking';
COMMENT ON TABLE scenes IS 'Dialog generation contexts and settings';
COMMENT ON TABLE projects IS 'Game development project organization';
COMMENT ON TABLE character_relationships IS 'Dynamic character relationship modeling';

-- Success message
SELECT 'Game Dialog Generator database schema created successfully! 🌿🎮 Ready for jungle adventures!' as status;