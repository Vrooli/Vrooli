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

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_resumes_score ON resumes(score DESC);
CREATE INDEX IF NOT EXISTS idx_resumes_status ON resumes(status);
CREATE INDEX IF NOT EXISTS idx_job_descriptions_status ON job_descriptions(status);
CREATE INDEX IF NOT EXISTS idx_candidate_matches_score ON candidate_matches(match_score DESC);
CREATE INDEX IF NOT EXISTS idx_candidate_matches_resume ON candidate_matches(resume_id);
CREATE INDEX IF NOT EXISTS idx_candidate_matches_job ON candidate_matches(job_id);