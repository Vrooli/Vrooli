-- Resume Screening Assistant Database Schema
-- This file contains the database schema for the Resume Screening Assistant scenario

-- Table for storing resume data
CREATE TABLE IF NOT EXISTS resumes (
    id SERIAL PRIMARY KEY,
    candidate_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE,
    resume_text TEXT,
    resume_file_path VARCHAR(500),
    parsed_skills JSONB,
    experience_years INTEGER,
    education_level VARCHAR(100),
    score INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing job descriptions
CREATE TABLE IF NOT EXISTS job_descriptions (
    id SERIAL PRIMARY KEY,
    job_title VARCHAR(255) NOT NULL,
    company_name VARCHAR(255),
    description TEXT NOT NULL,
    required_skills JSONB,
    preferred_skills JSONB,
    experience_required INTEGER,
    location VARCHAR(255),
    salary_range VARCHAR(100),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing candidate-job matches
CREATE TABLE IF NOT EXISTS candidate_matches (
    id SERIAL PRIMARY KEY,
    resume_id INTEGER REFERENCES resumes(id),
    job_id INTEGER REFERENCES job_descriptions(id),
    match_score DECIMAL(5,2),
    skills_match JSONB,
    experience_match BOOLEAN,
    overall_fit VARCHAR(50),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(resume_id, job_id)
);

-- Table for storing job embeddings for fast vector operations
CREATE TABLE IF NOT EXISTS job_embeddings (
    id SERIAL PRIMARY KEY,
    job_id INTEGER REFERENCES job_descriptions(id) ON DELETE CASCADE,
    embedding_vector FLOAT[384],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id)
);

-- Table for storing resume embeddings for fast vector operations
CREATE TABLE IF NOT EXISTS resume_embeddings (
    id SERIAL PRIMARY KEY,
    resume_id INTEGER REFERENCES resumes(id) ON DELETE CASCADE,
    embedding_vector FLOAT[384],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(resume_id)
);

-- Table for search history and analytics
CREATE TABLE IF NOT EXISTS search_history (
    id SERIAL PRIMARY KEY,
    query_text TEXT NOT NULL,
    query_type VARCHAR(50) DEFAULT 'semantic', -- semantic, keyword, hybrid
    results_count INTEGER DEFAULT 0,
    execution_time_ms INTEGER,
    filters JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_session VARCHAR(255)
);

-- Table for file upload tracking
CREATE TABLE IF NOT EXISTS file_uploads (
    id SERIAL PRIMARY KEY,
    original_filename VARCHAR(500) NOT NULL,
    file_path VARCHAR(1000) NOT NULL,
    file_size INTEGER,
    file_type VARCHAR(100),
    processing_status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    error_message TEXT,
    job_id INTEGER REFERENCES job_descriptions(id),
    resume_id INTEGER REFERENCES resumes(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP
);

-- Enhanced indexes for performance
CREATE INDEX IF NOT EXISTS idx_resumes_score ON resumes(score DESC);
CREATE INDEX IF NOT EXISTS idx_resumes_status ON resumes(status);
CREATE INDEX IF NOT EXISTS idx_resumes_created_at ON resumes(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_descriptions_status ON job_descriptions(status);
CREATE INDEX IF NOT EXISTS idx_job_descriptions_created_at ON job_descriptions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_candidate_matches_score ON candidate_matches(match_score DESC);
CREATE INDEX IF NOT EXISTS idx_candidate_matches_resume ON candidate_matches(resume_id);
CREATE INDEX IF NOT EXISTS idx_candidate_matches_job ON candidate_matches(job_id);
CREATE INDEX IF NOT EXISTS idx_job_embeddings_job_id ON job_embeddings(job_id);
CREATE INDEX IF NOT EXISTS idx_resume_embeddings_resume_id ON resume_embeddings(resume_id);
CREATE INDEX IF NOT EXISTS idx_search_history_created_at ON search_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_search_history_query_type ON search_history(query_type);
CREATE INDEX IF NOT EXISTS idx_file_uploads_status ON file_uploads(processing_status);
CREATE INDEX IF NOT EXISTS idx_file_uploads_job_id ON file_uploads(job_id);
CREATE INDEX IF NOT EXISTS idx_file_uploads_created_at ON file_uploads(created_at DESC);