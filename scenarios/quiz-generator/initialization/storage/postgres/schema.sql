-- Quiz Generator Database Schema
-- Version: 1.0.0
-- Description: Schema for AI-powered quiz generation and assessment platform

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS quiz_generator;
SET search_path TO quiz_generator;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For text search
CREATE EXTENSION IF NOT EXISTS "btree_gin"; -- For composite indexes

-- Enum types
CREATE TYPE question_type AS ENUM (
  'mcq',           -- Multiple choice
  'true_false',    -- True/False
  'short_answer',  -- Short text answer
  'fill_blank',    -- Fill in the blank
  'matching',      -- Match items
  'ordering'       -- Put items in order
);

CREATE TYPE difficulty_level AS ENUM (
  'easy',
  'medium', 
  'hard'
);

CREATE TYPE quiz_status AS ENUM (
  'draft',
  'published',
  'archived'
);

CREATE TYPE result_status AS ENUM (
  'in_progress',
  'completed',
  'abandoned'
);

-- Main quiz table
CREATE TABLE quizzes (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  title VARCHAR(255) NOT NULL,
  description TEXT,
  source_document TEXT,
  source_type VARCHAR(50), -- pdf, txt, md, docx, url
  difficulty difficulty_level DEFAULT 'medium',
  time_limit INTEGER, -- in seconds, NULL = unlimited
  passing_score INTEGER DEFAULT 70, -- percentage
  status quiz_status DEFAULT 'draft',
  tags TEXT[] DEFAULT '{}',
  metadata JSONB DEFAULT '{}',
  settings JSONB DEFAULT '{}', -- quiz-specific settings
  created_by VARCHAR(255),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  published_at TIMESTAMP WITH TIME ZONE,
  
  -- Statistics
  times_taken INTEGER DEFAULT 0,
  average_score DECIMAL(5,2),
  completion_rate DECIMAL(5,2)
);

-- Questions table
CREATE TABLE questions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  quiz_id UUID REFERENCES quizzes(id) ON DELETE CASCADE,
  type question_type NOT NULL,
  question_text TEXT NOT NULL,
  question_html TEXT, -- Rich text version
  options JSONB, -- For MCQ: [{id, text, is_correct}]
  correct_answer JSONB NOT NULL, -- Flexible structure for different types
  explanation TEXT,
  hint TEXT,
  difficulty difficulty_level DEFAULT 'medium',
  points INTEGER DEFAULT 1,
  order_index INTEGER NOT NULL,
  time_limit INTEGER, -- Question-specific time limit
  tags TEXT[] DEFAULT '{}',
  media_url TEXT, -- For image/audio/video questions
  metadata JSONB DEFAULT '{}',
  
  -- Analytics
  times_answered INTEGER DEFAULT 0,
  correct_rate DECIMAL(5,2),
  average_time_seconds INTEGER,
  
  -- Vector embedding reference for semantic search
  embedding_id VARCHAR(255),
  
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Question bank for reusable questions
CREATE TABLE question_bank (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  type question_type NOT NULL,
  category VARCHAR(100),
  subcategory VARCHAR(100),
  question_text TEXT NOT NULL,
  options JSONB,
  correct_answer JSONB NOT NULL,
  explanation TEXT,
  difficulty difficulty_level DEFAULT 'medium',
  tags TEXT[] DEFAULT '{}',
  usage_count INTEGER DEFAULT 0,
  quality_score DECIMAL(3,2), -- 0-5 rating
  metadata JSONB DEFAULT '{}',
  embedding_id VARCHAR(255),
  created_by VARCHAR(255),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Quiz results/attempts
CREATE TABLE quiz_results (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  quiz_id UUID REFERENCES quizzes(id) ON DELETE CASCADE,
  taker_id VARCHAR(255), -- External user reference
  taker_name VARCHAR(255),
  status result_status DEFAULT 'in_progress',
  score INTEGER,
  percentage DECIMAL(5,2),
  passed BOOLEAN,
  time_taken INTEGER, -- seconds
  started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP WITH TIME ZONE,
  submitted_at TIMESTAMP WITH TIME ZONE,
  
  -- Detailed scoring
  correct_answers INTEGER DEFAULT 0,
  incorrect_answers INTEGER DEFAULT 0,
  skipped_answers INTEGER DEFAULT 0,
  total_points INTEGER DEFAULT 0,
  earned_points INTEGER DEFAULT 0,
  
  -- Session data
  ip_address INET,
  user_agent TEXT,
  session_data JSONB DEFAULT '{}'
);

-- Individual question responses
CREATE TABLE question_responses (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  result_id UUID REFERENCES quiz_results(id) ON DELETE CASCADE,
  question_id UUID REFERENCES questions(id) ON DELETE CASCADE,
  given_answer JSONB,
  is_correct BOOLEAN,
  points_earned INTEGER DEFAULT 0,
  time_spent INTEGER, -- seconds
  answered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Quiz templates for common scenarios
CREATE TABLE quiz_templates (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name VARCHAR(255) NOT NULL,
  category VARCHAR(100),
  description TEXT,
  default_question_count INTEGER DEFAULT 10,
  default_difficulty difficulty_level DEFAULT 'medium',
  question_distribution JSONB, -- {mcq: 60, true_false: 20, short_answer: 20}
  tags TEXT[] DEFAULT '{}',
  usage_count INTEGER DEFAULT 0,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Analytics events for tracking
CREATE TABLE analytics_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  event_type VARCHAR(50) NOT NULL,
  entity_type VARCHAR(50), -- quiz, question, result
  entity_id UUID,
  user_id VARCHAR(255),
  event_data JSONB DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_quizzes_status ON quizzes(status);
CREATE INDEX idx_quizzes_tags ON quizzes USING GIN(tags);
CREATE INDEX idx_quizzes_created_at ON quizzes(created_at DESC);
CREATE INDEX idx_quizzes_metadata ON quizzes USING GIN(metadata);

CREATE INDEX idx_questions_quiz_id ON questions(quiz_id);
CREATE INDEX idx_questions_type ON questions(type);
CREATE INDEX idx_questions_difficulty ON questions(difficulty);
CREATE INDEX idx_questions_tags ON questions USING GIN(tags);
CREATE INDEX idx_questions_embedding ON questions(embedding_id);

CREATE INDEX idx_question_bank_category ON question_bank(category, subcategory);
CREATE INDEX idx_question_bank_tags ON question_bank USING GIN(tags);
CREATE INDEX idx_question_bank_embedding ON question_bank(embedding_id);
CREATE INDEX idx_question_bank_text_search ON question_bank USING GIN(to_tsvector('english', question_text));

CREATE INDEX idx_results_quiz_id ON quiz_results(quiz_id);
CREATE INDEX idx_results_taker_id ON quiz_results(taker_id);
CREATE INDEX idx_results_status ON quiz_results(status);
CREATE INDEX idx_results_completed_at ON quiz_results(completed_at DESC);

CREATE INDEX idx_responses_result_id ON question_responses(result_id);
CREATE INDEX idx_responses_question_id ON question_responses(question_id);

CREATE INDEX idx_analytics_event_type ON analytics_events(event_type, created_at DESC);
CREATE INDEX idx_analytics_entity ON analytics_events(entity_type, entity_id);

-- Full text search indexes
CREATE INDEX idx_quizzes_fts ON quizzes USING GIN(
  to_tsvector('english', title || ' ' || COALESCE(description, ''))
);

CREATE INDEX idx_questions_fts ON questions USING GIN(
  to_tsvector('english', question_text || ' ' || COALESCE(explanation, ''))
);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_quizzes_updated_at BEFORE UPDATE ON quizzes
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_questions_updated_at BEFORE UPDATE ON questions
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_question_bank_updated_at BEFORE UPDATE ON question_bank
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Functions for analytics
CREATE OR REPLACE FUNCTION calculate_quiz_statistics(quiz_uuid UUID)
RETURNS TABLE (
  total_attempts INTEGER,
  avg_score DECIMAL,
  completion_percentage DECIMAL,
  avg_time_minutes DECIMAL
) AS $$
BEGIN
  RETURN QUERY
  SELECT
    COUNT(*)::INTEGER as total_attempts,
    AVG(percentage)::DECIMAL as avg_score,
    (COUNT(CASE WHEN status = 'completed' THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0))::DECIMAL as completion_percentage,
    (AVG(time_taken) / 60.0)::DECIMAL as avg_time_minutes
  FROM quiz_results
  WHERE quiz_id = quiz_uuid;
END;
$$ LANGUAGE plpgsql;

-- Function to get question performance
CREATE OR REPLACE FUNCTION get_question_performance(question_uuid UUID)
RETURNS TABLE (
  total_responses INTEGER,
  correct_percentage DECIMAL,
  avg_time_seconds INTEGER,
  difficulty_rating VARCHAR
) AS $$
BEGIN
  RETURN QUERY
  SELECT
    COUNT(*)::INTEGER as total_responses,
    (COUNT(CASE WHEN is_correct THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0))::DECIMAL as correct_percentage,
    AVG(time_spent)::INTEGER as avg_time_seconds,
    CASE
      WHEN (COUNT(CASE WHEN is_correct THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0)) > 80 THEN 'easy'
      WHEN (COUNT(CASE WHEN is_correct THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0)) > 50 THEN 'medium'
      ELSE 'hard'
    END as difficulty_rating
  FROM question_responses
  WHERE question_id = question_uuid;
END;
$$ LANGUAGE plpgsql;

-- Sample data for testing (optional)
-- Uncomment to insert sample quiz template
/*
INSERT INTO quiz_templates (name, category, description, default_question_count, question_distribution) VALUES
  ('Basic Knowledge Check', 'general', 'Standard quiz with mixed question types', 10, '{"mcq": 6, "true_false": 2, "short_answer": 2}'),
  ('Quick Assessment', 'general', 'Fast 5-question quiz', 5, '{"mcq": 3, "true_false": 2}'),
  ('Comprehensive Exam', 'academic', 'Detailed assessment with all question types', 25, '{"mcq": 10, "true_false": 5, "short_answer": 5, "fill_blank": 3, "matching": 2}');
*/

-- Permissions (adjust based on your user roles)
GRANT USAGE ON SCHEMA quiz_generator TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA quiz_generator TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA quiz_generator TO PUBLIC;