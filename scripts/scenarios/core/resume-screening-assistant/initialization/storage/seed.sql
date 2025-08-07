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
),
(
    'DevOps Engineer',
    'CloudFirst Solutions',
    'Seeking a DevOps Engineer to manage our cloud infrastructure and implement CI/CD pipelines. Experience with Kubernetes and AWS required.',
    '["Kubernetes", "AWS", "Docker", "CI/CD", "Linux", "Terraform"]'::jsonb,
    '["Ansible", "Monitoring", "Security", "Python", "GitOps"]'::jsonb,
    3,
    'Seattle, WA',
    '$110,000 - $145,000'
),
(
    'Frontend Developer',
    'DesignTech Studio',
    'Join our frontend team to build beautiful, responsive user interfaces. Strong knowledge of modern JavaScript frameworks required.',
    '["JavaScript", "React", "CSS", "HTML", "Webpack"]'::jsonb,
    '["TypeScript", "Vue.js", "SASS", "Testing", "Figma"]'::jsonb,
    2,
    'New York, NY',
    '$75,000 - $105,000'
),
(
    'AI Research Scientist',
    'FutureTech Labs',
    'Research position focused on developing next-generation machine learning models. PhD in Computer Science or related field preferred.',
    '["Machine Learning", "Deep Learning", "Python", "PyTorch", "Research", "Mathematics"]'::jsonb,
    '["Publications", "TensorFlow", "NLP", "Computer Vision", "Statistics"]'::jsonb,
    2,
    'Boston, MA',
    '$140,000 - $180,000'
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
),
(
    'Alex Chen',
    'alex.chen@email.com',
    'DevOps engineer with 4 years of experience in cloud infrastructure and automation. Expertise in Kubernetes, AWS, and CI/CD pipelines.',
    '["Kubernetes", "AWS", "Docker", "Terraform", "Linux", "Python", "Ansible"]'::jsonb,
    4,
    'Bachelor''s Degree',
    89,
    'reviewed'
),
(
    'Emma Rodriguez',
    'emma.rodriguez@email.com',
    'Frontend developer specializing in React and modern JavaScript. 3 years of experience building responsive web applications.',
    '["JavaScript", "React", "TypeScript", "CSS", "HTML", "Webpack", "Testing"]'::jsonb,
    3,
    'Bachelor''s Degree',
    81,
    'pending'
),
(
    'Dr. James Park',
    'james.park@email.com',
    'AI researcher with PhD in Computer Science. Published multiple papers on deep learning and natural language processing.',
    '["Machine Learning", "Deep Learning", "Python", "PyTorch", "TensorFlow", "NLP", "Research"]'::jsonb,
    3,
    'PhD',
    95,
    'reviewed'
),
(
    'Lisa Thompson',
    'lisa.thompson@email.com',
    'Junior developer with 1 year of experience in web development. Strong foundation in JavaScript and React.',
    '["JavaScript", "React", "HTML", "CSS", "Git"]'::jsonb,
    1,
    'Bachelor''s Degree',
    65,
    'pending'
);

-- Insert sample candidate matches
INSERT INTO candidate_matches (resume_id, job_id, match_score, skills_match, experience_match, overall_fit, notes) VALUES
-- Senior Software Engineer matches
(1, 1, 88.5, '{"matching": ["JavaScript", "React", "Node.js", "PostgreSQL"], "missing": ["Git"], "bonus": ["Python", "Docker"]}'::jsonb, true, 'excellent', 'Strong technical background with relevant experience'),
(6, 1, 72.0, '{"matching": ["JavaScript", "React"], "missing": ["Node.js", "PostgreSQL", "Git"], "bonus": []}', false, 'maybe', 'Junior developer with potential but needs more experience'),

-- Data Scientist matches  
(2, 2, 95.0, '{"matching": ["Python", "SQL", "Machine Learning", "Statistics"], "missing": ["Pandas"], "bonus": ["TensorFlow", "R"]}'::jsonb, true, 'excellent', 'Perfect fit for data science role'),

-- Product Manager matches
(3, 3, 82.0, '{"matching": ["Product Strategy", "Agile"], "missing": ["Requirements Gathering"], "bonus": ["SQL"]}'::jsonb, true, 'good', 'Good product management experience with some technical background'),

-- DevOps Engineer matches
(4, 4, 92.0, '{"matching": ["Kubernetes", "AWS", "Docker", "Terraform", "Linux"], "missing": ["CI/CD"], "bonus": ["Python", "Ansible"]}'::jsonb, true, 'excellent', 'Excellent DevOps skills with strong cloud experience'),
(1, 4, 65.0, '{"matching": ["Docker"], "missing": ["Kubernetes", "AWS", "CI/CD", "Linux", "Terraform"], "bonus": ["Python"]}'::jsonb, false, 'maybe', 'Software engineer with some containerization experience'),

-- Frontend Developer matches
(5, 5, 85.0, '{"matching": ["JavaScript", "React", "CSS", "HTML"], "missing": ["Webpack"], "bonus": ["TypeScript", "Testing"]}'::jsonb, true, 'excellent', 'Strong frontend skills with modern framework experience'),
(6, 5, 78.0, '{"matching": ["JavaScript", "React", "HTML", "CSS"], "missing": ["Webpack"], "bonus": ["Git"]}'::jsonb, false, 'good', 'Junior developer perfect for entry-level frontend position'),

-- AI Research Scientist matches
(7, 6, 98.0, '{"matching": ["Machine Learning", "Deep Learning", "Python", "PyTorch"], "missing": [], "bonus": ["TensorFlow", "NLP", "Research"]}'::jsonb, true, 'excellent', 'PhD with perfect research background and publications'),
(2, 6, 75.0, '{"matching": ["Machine Learning", "Python"], "missing": ["Deep Learning", "PyTorch", "Research", "Mathematics"], "bonus": ["TensorFlow"]}'::jsonb, false, 'maybe', 'Data science background but lacks deep learning research experience');