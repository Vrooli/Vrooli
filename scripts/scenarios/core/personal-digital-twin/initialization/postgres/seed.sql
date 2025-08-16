-- Personal Digital Twin - Seed Data
-- Initial data for testing and demonstration

-- Insert sample personas
INSERT INTO personas (id, name, description, base_model, personality_traits, knowledge_domains, conversation_style)
VALUES 
  (
    '550e8400-e29b-41d4-a716-446655440001',
    'Alex - Coding Expert', 
    'A specialized AI persona trained on software development best practices, coding patterns, and technical documentation.',
    'llama3.2',
    '{"creativity": 0.6, "analytical": 0.9, "empathy": 0.4, "precision": 0.9}',
    '["software-development", "programming", "system-architecture", "code-review"]',
    '{"tone": "professional", "verbosity": "detailed", "code_examples": true}'
  ),
  (
    '550e8400-e29b-41d4-a716-446655440002',
    'Morgan - Project Manager',
    'An AI persona specialized in project management, team coordination, and strategic planning.',
    'llama3.2', 
    '{"creativity": 0.7, "analytical": 0.8, "empathy": 0.8, "leadership": 0.9}',
    '["project-management", "team-coordination", "strategic-planning", "process-optimization"]',
    '{"tone": "encouraging", "verbosity": "concise", "action_oriented": true}'
  ),
  (
    '550e8400-e29b-41d4-a716-446655440003',
    'Sam - Research Assistant',
    'A research-focused AI persona trained on academic papers, research methodologies, and data analysis.',
    'llama3.2',
    '{"creativity": 0.8, "analytical": 0.9, "empathy": 0.5, "curiosity": 0.9}',
    '["research", "data-analysis", "academic-writing", "literature-review"]',
    '{"tone": "academic", "verbosity": "comprehensive", "citations": true}'
  )
ON CONFLICT (id) DO NOTHING;

-- Insert sample data sources
INSERT INTO data_sources (id, persona_id, source_type, source_config, sync_frequency, total_documents, total_size_bytes, status)
VALUES
  (
    '660e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    'local_files',
    '{"path": "/data/coding-resources", "include_patterns": ["*.md", "*.py", "*.js", "*.ts"], "exclude_patterns": ["node_modules", "*.tmp"]}',
    'daily',
    25,
    52428800,
    'active'
  ),
  (
    '660e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440002',
    'github',
    '{"repository": "company/project-templates", "include_types": ["issues", "pull_requests", "documentation"]}',
    'weekly',
    15,
    31457280,
    'active'
  ),
  (
    '660e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440003',
    'local_files',
    '{"path": "/data/research-papers", "include_patterns": ["*.pdf", "*.tex", "*.bib"], "recursive": true}',
    'manual',
    42,
    104857600,
    'active'
  )
ON CONFLICT (id) DO NOTHING;

-- Insert sample processed documents
INSERT INTO documents (id, persona_id, source_id, original_path, document_type, metadata, chunk_count, token_count, processing_status, processed_at)
VALUES
  (
    '770e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440001',
    '/data/coding-resources/best-practices.md',
    'markdown',
    '{"language": "english", "topics": ["clean-code", "testing", "documentation"], "word_count": 2500}',
    12,
    3200,
    'completed',
    NOW() - INTERVAL '2 days'
  ),
  (
    '770e8400-e29b-41d4-a716-446655440002', 
    '550e8400-e29b-41d4-a716-446655440002',
    '660e8400-e29b-41d4-a716-446655440002',
    'github:company/project-templates#123',
    'github_issue',
    '{"issue_number": 123, "labels": ["planning", "template"], "assignees": ["alice", "bob"]}',
    8,
    1800,
    'completed',
    NOW() - INTERVAL '1 day'
  ),
  (
    '770e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440003', 
    '660e8400-e29b-41d4-a716-446655440003',
    '/data/research-papers/ml-paper-2024.pdf',
    'pdf',
    '{"title": "Advances in Machine Learning", "authors": ["Dr. Jane Smith", "Dr. John Doe"], "year": 2024, "pages": 24}',
    28,
    8500,
    'completed',
    NOW() - INTERVAL '3 hours'
  )
ON CONFLICT (id) DO NOTHING;

-- Insert sample conversations for testing
INSERT INTO conversations (id, persona_id, session_id, user_id, messages, total_tokens_used)
VALUES
  (
    '880e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    'session_001', 
    'user_123',
    '[
      {"role": "user", "content": "What are the best practices for writing clean code?", "timestamp": "2024-01-01T10:00:00Z"},
      {"role": "assistant", "content": "Based on my knowledge, here are the key principles for writing clean code: 1) Use meaningful names for variables and functions, 2) Keep functions small and focused on a single responsibility, 3) Write comprehensive tests...", "timestamp": "2024-01-01T10:00:15Z"}
    ]',
    150
  ),
  (
    '880e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440002',
    'session_002',
    'user_456', 
    '[
      {"role": "user", "content": "How should I approach planning a new software project?", "timestamp": "2024-01-01T14:30:00Z"},
      {"role": "assistant", "content": "Great question! Here is my recommended approach for project planning: 1) Start with requirements gathering and stakeholder alignment, 2) Break down the project into manageable phases, 3) Create realistic timelines with buffer time...", "timestamp": "2024-01-01T14:30:20Z"}
    ]',
    180
  )
ON CONFLICT (id) DO NOTHING;

-- Insert sample training jobs
INSERT INTO training_jobs (id, persona_id, job_type, model_name, technique, training_config, status, created_at)
VALUES
  (
    '990e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    'fine_tune',
    'llama3.2',
    'lora',
    '{"learning_rate": 0.0001, "epochs": 3, "batch_size": 4, "r": 16, "alpha": 32}',
    'completed',
    NOW() - INTERVAL '1 day'
  ),
  (
    '990e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440002', 
    'fine_tune',
    'llama3.2',
    'lora',
    '{"learning_rate": 0.0001, "epochs": 3, "batch_size": 4, "r": 16, "alpha": 32}',
    'queued',
    NOW() - INTERVAL '2 hours'
  )
ON CONFLICT (id) DO NOTHING;

-- Insert sample API tokens for external access
INSERT INTO api_tokens (id, persona_id, token_hash, name, permissions, expires_at)
VALUES
  (
    'aa0e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    'b5d4c8a7f9e2d1c6b4a3e8f7d2c5b9a6e3f8d1c4b7a2e5f8c1d4b9a6e3f8d2c5',
    'app-personalizer-integration',
    '["read", "chat"]',
    NOW() + INTERVAL '1 year'
  ),
  (
    'aa0e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440002',
    'c6e5d9b8g0f3e2d7c5b4f9a8e4d7c2f5b8e1d4c7b0a5f8d2c5b9a7e4f9d3c6',
    'research-integration',
    '["read", "chat", "train"]',
    NOW() + INTERVAL '1 year'  
  )
ON CONFLICT (id) DO NOTHING;

-- Update personas with current stats
UPDATE personas SET 
  document_count = (SELECT COUNT(*) FROM documents WHERE documents.persona_id = personas.id),
  total_tokens = (SELECT COALESCE(SUM(token_count), 0) FROM documents WHERE documents.persona_id = personas.id),
  last_updated = NOW()
WHERE id IN (
  '550e8400-e29b-41d4-a716-446655440001',
  '550e8400-e29b-41d4-a716-446655440002', 
  '550e8400-e29b-41d4-a716-446655440003'
);