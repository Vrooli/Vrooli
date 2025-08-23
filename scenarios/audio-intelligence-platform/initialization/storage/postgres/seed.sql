-- Audio Intelligence Platform - Seed Data
-- Populates the database with demo transcriptions, configurations, and sample data

-- Insert demo transcriptions for testing and demonstration
INSERT INTO audio_intelligence_platform.transcriptions (
    id, filename, original_file_path, file_size_bytes, content_type, duration_seconds,
    transcription_text, word_timestamps, confidence_score, language_detected,
    whisper_model_used, processing_time_ms, status, embedding_status,
    minio_audio_path, minio_transcript_path, session_id, user_identifier
) VALUES 
    ('550e8400-e29b-41d4-a716-446655440001'::uuid,
     'demo-audio-episode-01.mp3',
     '/uploads/demo-audio-episode-01.mp3',
     5242880, -- 5MB
     'audio/mpeg',
     300, -- 5 minutes
     'Welcome to our audio intelligence demonstration! Today we''re discussing the future of artificial intelligence and how it will impact content creation. AI tools are becoming increasingly sophisticated, allowing creators to automate transcription, generate summaries, and even create entirely new content. The implications for productivity and creativity are enormous.',
     '[{"word": "Welcome", "start": 0.0, "end": 0.8}, {"word": "to", "start": 0.8, "end": 1.0}, {"word": "our", "start": 1.0, "end": 1.2}]'::jsonb,
     0.95,
     'en',
     'base',
     12500,
     'completed',
     'completed',
     'audio-files/demo-audio-episode-01.mp3',
     'transcriptions/demo-audio-episode-01.txt',
     'demo-session-001',
     'demo-user'
    ),
    
    ('550e8400-e29b-41d4-a716-446655440002'::uuid,
     'interview-tech-founder.wav',
     '/uploads/interview-tech-founder.wav',
     12582912, -- 12MB
     'audio/wav',
     720, -- 12 minutes
     'In this interview, we explore the journey of building a successful startup from scratch. Our guest shares insights about product development, market fit, and scaling challenges. Key topics include fundraising strategies, team building, and the importance of customer feedback in shaping product direction.',
     '[{"word": "In", "start": 0.0, "end": 0.2}, {"word": "this", "start": 0.2, "end": 0.4}, {"word": "interview", "start": 0.4, "end": 1.0}]'::jsonb,
     0.92,
     'en',
     'base',
     28900,
     'completed',
     'completed',
     'audio-files/interview-tech-founder.wav',
     'transcriptions/interview-tech-founder.txt',
     'demo-session-002',
     'demo-user'
    ),
    
    ('550e8400-e29b-41d4-a716-446655440003'::uuid,
     'research-discussion.m4a',
     '/uploads/research-discussion.m4a',
     8388608, -- 8MB
     'audio/m4a',
     480, -- 8 minutes
     'Our research team discusses recent findings in machine learning and natural language processing. The conversation covers breakthrough developments in transformer architectures, multimodal AI systems, and practical applications in business automation.',
     '[{"word": "Our", "start": 0.0, "end": 0.3}, {"word": "research", "start": 0.3, "end": 0.8}, {"word": "team", "start": 0.8, "end": 1.1}]'::jsonb,
     0.88,
     'en',
     'base',
     19200,
     'completed',
     'pending',
     'audio-files/research-discussion.m4a',
     'transcriptions/research-discussion.txt',
     'demo-session-001',
     'demo-user'
    );

-- Insert sample AI analyses
INSERT INTO audio_intelligence_platform.ai_analyses (
    id, transcription_id, analysis_type, prompt_used, result_text,
    ai_model_used, processing_time_ms, tokens_used, confidence_score,
    session_id, user_identifier
) VALUES 
    ('650e8400-e29b-41d4-a716-446655440001'::uuid,
     '550e8400-e29b-41d4-a716-446655440001'::uuid,
     'summary',
     'Please provide a concise summary of this transcription.',
     'This audio recording introduces a discussion about artificial intelligence''s impact on content creation. The speakers highlight how AI tools are enabling creators to automate transcription and summary generation, while also creating entirely new content. They emphasize the significant implications for both productivity and creativity in the content creation industry.',
     'llama3.1:8b',
     2800,
     145,
     0.91,
     'demo-session-001',
     'demo-user'
    ),
    
    ('650e8400-e29b-41d4-a716-446655440002'::uuid,
     '550e8400-e29b-41d4-a716-446655440001'::uuid,
     'key_insights',
     'Extract the key insights and main points from this transcription.',
     '• AI tools are becoming sophisticated enough to automate content creation tasks\n• Transcription automation is a major productivity boost for creators\n• AI can generate summaries and create new content, not just process existing content\n• The impact on creativity and productivity is expected to be enormous\n• Content creators should prepare for significant workflow changes',
     'llama3.1:8b',
     3200,
     178,
     0.89,
     'demo-session-001',
     'demo-user'
    ),
    
    ('650e8400-e29b-41d4-a716-446655440003'::uuid,
     '550e8400-e29b-41d4-a716-446655440002'::uuid,
     'summary',
     'Please provide a concise summary of this transcription.',
     'This interview focuses on the entrepreneurial journey of building a startup from the ground up. The guest discusses critical aspects including product development, finding market fit, and overcoming scaling challenges. Key areas covered include effective fundraising strategies, building strong teams, and leveraging customer feedback to guide product development decisions.',
     'llama3.1:8b',
     4100,
     198,
     0.93,
     'demo-session-002',
     'demo-user'
    );

-- Insert demo user sessions
INSERT INTO audio_intelligence_platform.user_sessions (
    id, session_id, user_identifier, preferences, search_history, favorite_transcriptions,
    transcriptions_count, analyses_count, searches_count, total_audio_duration_seconds
) VALUES 
    ('750e8400-e29b-41d4-a716-446655440001'::uuid,
     'demo-session-001',
     'demo-user',
     '{"theme": "light", "auto_analyze": true, "export_format": "txt", "whisper_model": "base"}'::jsonb,
     '["artificial intelligence", "content creation", "productivity tools", "AI automation"]'::jsonb,
     ARRAY['550e8400-e29b-41d4-a716-446655440001'::uuid, '550e8400-e29b-41d4-a716-446655440003'::uuid],
     2,
     2,
     4,
     780
    ),
    
    ('750e8400-e29b-41d4-a716-446655440002'::uuid,
     'demo-session-002',
     'demo-user',
     '{"theme": "dark", "auto_analyze": false, "export_format": "pdf", "whisper_model": "small"}'::jsonb,
     '["startup interviews", "entrepreneurship", "business strategy"]'::jsonb,
     ARRAY['550e8400-e29b-41d4-a716-446655440002'::uuid],
     1,
     1,
     3,
     720
    );

-- Insert sample search queries
INSERT INTO audio_intelligence_platform.search_queries (
    id, query_text, results_count, max_similarity_score, processing_time_ms,
    session_id, user_identifier
) VALUES 
    ('850e8400-e29b-41d4-a716-446655440001'::uuid,
     'artificial intelligence content creation',
     2,
     0.94,
     450,
     'demo-session-001',
     'demo-user'
    ),
    
    ('850e8400-e29b-41d4-a716-446655440002'::uuid,
     'startup fundraising strategies',
     1,
     0.87,
     380,
     'demo-session-002',
     'demo-user'
    ),
    
    ('850e8400-e29b-41d4-a716-446655440003'::uuid,
     'machine learning research',
     1,
     0.82,
     420,
     'demo-session-001',
     'demo-user'
    );

-- Insert sample resource metrics
INSERT INTO audio_intelligence_platform.resource_metrics (
    resource_type, operation, duration_ms, success, input_size_bytes, output_size_bytes,
    tokens_used, transcription_id, session_id, request_metadata
) VALUES 
    ('whisper', 'transcribe', 12500, true, 5242880, 2048, null,
     '550e8400-e29b-41d4-a716-446655440001'::uuid, 'demo-session-001',
     '{"model": "base", "language": "en", "audio_duration": 300}'::jsonb),
     
    ('ollama', 'analyze', 2800, true, 1024, 512, 145,
     '550e8400-e29b-41d4-a716-446655440001'::uuid, 'demo-session-001',
     '{"model": "llama3.1:8b", "analysis_type": "summary", "temperature": 0.7}'::jsonb),
     
    ('ollama', 'embed', 680, true, 512, 384, null,
     '550e8400-e29b-41d4-a716-446655440001'::uuid, 'demo-session-001',
     '{"model": "nomic-embed-text", "text_length": 512}'::jsonb),
     
    ('qdrant', 'search', 450, true, 384, 1536, null,
     null, 'demo-session-001',
     '{"query": "artificial intelligence", "collection": "transcription-embeddings", "limit": 50}'::jsonb),
     
    ('minio', 'upload', 2100, true, 5242880, 0, null,
     '550e8400-e29b-41d4-a716-446655440001'::uuid, 'demo-session-001',
     '{"bucket": "audio-files", "content_type": "audio/mpeg", "filename": "demo-audio-episode-01.mp3"}'::jsonb);

-- Additional sample configuration specific to audio intelligence
INSERT INTO audio_intelligence_platform.app_config (config_key, config_value, config_type, description) VALUES
-- Transcription-specific settings
('transcription.supported_formats', '["mp3", "wav", "m4a", "ogg", "flac", "mp4"]', 'setting', 'Supported audio file formats'),
('transcription.max_duration_minutes', '180', 'setting', 'Maximum audio duration in minutes'),
('transcription.auto_language_detection', 'true', 'setting', 'Enable automatic language detection'),
('transcription.word_timestamps', 'true', 'setting', 'Include word-level timestamps'),
('transcription.default_model', '"base"', 'setting', 'Default Whisper model (tiny, base, small, medium, large)'),

-- AI Analysis settings
('analysis.default_temperature', '0.7', 'setting', 'Default temperature for AI analysis'),
('analysis.max_tokens', '2048', 'setting', 'Maximum tokens for AI responses'),
('analysis.auto_analyze', 'false', 'setting', 'Automatically analyze new transcriptions'),
('analysis.summary_prompt', '"Please provide a concise summary of this transcription."', 'setting', 'Default prompt for summaries'),
('analysis.insights_prompt', '"Extract the key insights and main points from this transcription."', 'setting', 'Default prompt for key insights'),

-- Search settings
('search.embedding_model', '"nomic-embed-text"', 'setting', 'Model used for generating embeddings'),
('search.collection_name', '"transcription-embeddings"', 'setting', 'Qdrant collection name'),
('search.vector_size', '384', 'setting', 'Size of embedding vectors'),
('search.similarity_metric', '"cosine"', 'setting', 'Similarity calculation method'),
('search.default_limit', '20', 'setting', 'Default number of search results'),
('search.min_similarity', '0.7', 'setting', 'Minimum similarity score for results'),

-- UI preferences
('ui.default_view', '"grid"', 'setting', 'Default view mode (grid or list)'),
('ui.export_formats', '["txt", "pdf", "json", "srt"]', 'setting', 'Available export formats'),
('ui.show_confidence_scores', 'true', 'setting', 'Display transcription confidence scores'),
('ui.show_word_timestamps', 'false', 'setting', 'Display word-level timestamps in UI'),
('ui.auto_refresh_interval', '30', 'setting', 'Auto-refresh interval in seconds'),

-- Storage settings
('storage.audio_retention_days', '365', 'setting', 'Days to retain original audio files'),
('storage.transcript_retention_days', '730', 'setting', 'Days to retain transcription text'),
('storage.analysis_retention_days', '180', 'setting', 'Days to retain AI analysis results'),
('storage.max_file_size_mb', '500', 'setting', 'Maximum audio file size in MB'),
('storage.compression_enabled', 'true', 'setting', 'Enable audio compression for storage'),

-- Feature flags
('features.batch_upload', 'true', 'feature_flag', 'Enable batch file upload'),
('features.real_time_transcription', 'false', 'feature_flag', 'Enable real-time transcription'),
('features.speaker_identification', 'false', 'feature_flag', 'Enable speaker identification'),
('features.export_subtitles', 'true', 'feature_flag', 'Enable subtitle export (SRT format)'),
('features.custom_prompts', 'true', 'feature_flag', 'Allow custom AI analysis prompts'),
('features.transcription_editing', 'false', 'feature_flag', 'Allow manual transcription editing')
ON CONFLICT (config_key) DO NOTHING;

-- Display seed data verification
SELECT 
    'Database seeding completed successfully!' as status,
    (SELECT COUNT(*) FROM audio_intelligence_platform.transcriptions) as transcriptions,
    (SELECT COUNT(*) FROM audio_intelligence_platform.ai_analyses) as analyses,
    (SELECT COUNT(*) FROM audio_intelligence_platform.user_sessions) as sessions,
    (SELECT COUNT(*) FROM audio_intelligence_platform.search_queries) as searches,
    (SELECT COUNT(*) FROM audio_intelligence_platform.resource_metrics) as metrics,
    (SELECT COUNT(*) FROM audio_intelligence_platform.app_config) as config_entries;