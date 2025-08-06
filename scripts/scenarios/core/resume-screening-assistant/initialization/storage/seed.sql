-- Resume Screening Assistant Seed Data
-- This file contains initial data for the Resume Screening Assistant scenario

-- Insert sample job descriptions
INSERT INTO job_descriptions (job_title, company_name, description, required_skills, preferred_skills, experience_required, location, salary_range) VALUES
(
    'Senior Software Engineer',
    'Tech Innovations Inc',
    'We are looking for a Senior Software Engineer to join our dynamic team. The ideal candidate will have experience in full-stack development with modern frameworks.',
    '["JavaScript", "React", "Node.js", "PostgreSQL", "Git"]'::jsonb,
    '["TypeScript", "Docker", "AWS", "CI/CD", "GraphQL"]'::jsonb,
    5,
    'San Francisco, CA',
    '$120,000 - $160,000'
),
(
    'Data Scientist',
    'Analytics Pro LLC',
    'Join our data science team to build predictive models and extract insights from large datasets. Experience with machine learning and statistical analysis required.',
    '["Python", "SQL", "Machine Learning", "Statistics", "Pandas"]'::jsonb,
    '["R", "TensorFlow", "PyTorch", "Spark", "Tableau"]'::jsonb,
    3,
    'Remote',
    '$90,000 - $130,000'
),
(
    'Product Manager',
    'StartupVenture Co',
    'Looking for a Product Manager to lead product strategy and work closely with engineering and design teams.',
    '["Product Strategy", "Agile", "Requirements Gathering", "Stakeholder Management"]'::jsonb,
    '["Technical Background", "SQL", "Analytics Tools", "UX Design"]'::jsonb,
    4,
    'Austin, TX',
    '$100,000 - $140,000'
);

-- Insert sample resume data (placeholder entries)
INSERT INTO resumes (candidate_name, email, resume_text, parsed_skills, experience_years, education_level, score, status) VALUES
(
    'John Smith',
    'john.smith@email.com',
    'Experienced software engineer with 6 years in full-stack development...',
    '["JavaScript", "React", "Node.js", "PostgreSQL", "Python", "Docker"]'::jsonb,
    6,
    'Bachelor''s Degree',
    85,
    'reviewed'
),
(
    'Sarah Johnson',
    'sarah.johnson@email.com',
    'Data scientist with expertise in machine learning and statistical analysis...',
    '["Python", "SQL", "Machine Learning", "TensorFlow", "Statistics", "R"]'::jsonb,
    4,
    'Master''s Degree',
    92,
    'reviewed'
),
(
    'Mike Wilson',
    'mike.wilson@email.com',
    'Product manager with 5 years of experience in agile development environments...',
    '["Product Strategy", "Agile", "Scrum", "Analytics", "SQL"]'::jsonb,
    5,
    'MBA',
    78,
    'pending'
);

-- Insert sample candidate matches
INSERT INTO candidate_matches (resume_id, job_id, match_score, skills_match, experience_match, overall_fit, notes) VALUES
(1, 1, 88.5, '{"matching": ["JavaScript", "React", "Node.js", "PostgreSQL"], "missing": ["Git"], "bonus": ["Python", "Docker"]}'::jsonb, true, 'excellent', 'Strong technical background with relevant experience'),
(2, 2, 95.0, '{"matching": ["Python", "SQL", "Machine Learning", "Statistics"], "missing": ["Pandas"], "bonus": ["TensorFlow", "R"]}'::jsonb, true, 'excellent', 'Perfect fit for data science role'),
(3, 3, 82.0, '{"matching": ["Product Strategy", "Agile"], "missing": ["Requirements Gathering"], "bonus": ["SQL"]}'::jsonb, true, 'good', 'Good product management experience with some technical background');