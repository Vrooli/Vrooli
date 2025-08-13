-- Resource Experimenter - Seed Data
-- Sample templates and scenarios for experimentation

-- Insert common experiment templates
INSERT INTO experiment_templates (name, description, prompt_template, target_scenario_pattern, resource_category) VALUES
('Add Storage Resource', 'Add a new storage resource to an existing scenario', 'Take the existing {{TARGET_SCENARIO}} scenario and integrate {{NEW_RESOURCE}} storage. Modify the initialization files to include {{NEW_RESOURCE}} configuration, update the service.json to enable the new resource, and create any necessary database schemas or storage configurations.', '*', 'storage'),
('Add AI Resource', 'Integrate an AI model or service into a scenario', 'Enhance the {{TARGET_SCENARIO}} scenario by adding {{NEW_RESOURCE}} AI capabilities. Update workflows to utilize the AI service, create necessary prompts or model configurations, and integrate AI processing into the business logic.', '*', 'ai'),
('Add Automation Resource', 'Add workflow automation capabilities', 'Expand {{TARGET_SCENARIO}} with {{NEW_RESOURCE}} automation. Create new automation workflows, configure triggers and actions, and integrate the automation platform into the scenario\'s business processes.', '*', 'automation'),
('Dashboard Enhancement', 'Add monitoring or analytics to dashboard scenarios', 'Enhance the {{TARGET_SCENARIO}} dashboard by integrating {{NEW_RESOURCE}}. Add new metrics collection, visualization components, real-time monitoring capabilities, and necessary data pipelines.', '*dashboard*', 'monitoring'),
('Assistant Enhancement', 'Add capabilities to AI assistant scenarios', 'Upgrade the {{TARGET_SCENARIO}} assistant with {{NEW_RESOURCE}} capabilities. Integrate new data sources, enhance processing abilities, add new interaction modalities, and expand the assistant\'s knowledge base.', '*assistant*', 'ai');

-- Insert available scenarios for experimentation
INSERT INTO available_scenarios (name, display_name, description, path, current_resources, resource_categories, complexity_level, experimentation_friendly) VALUES
('analytics-dashboard', 'Analytics Dashboard', 'Business analytics and monitoring dashboard', '/scripts/scenarios/core/analytics-dashboard', ARRAY['postgres', 'n8n', 'windmill'], ARRAY['storage', 'automation'], 'intermediate', true),
('research-assistant', 'Research Assistant', 'AI-powered research and document analysis platform', '/scripts/scenarios/core/research-assistant', ARRAY['postgres', 'qdrant', 'minio', 'ollama', 'n8n'], ARRAY['storage', 'ai', 'automation'], 'advanced', true),
('idea-generator', 'Idea Generator', 'Creative idea generation and development platform', '/scripts/scenarios/core/idea-generator', ARRAY['postgres', 'qdrant', 'ollama', 'n8n'], ARRAY['storage', 'ai', 'automation'], 'intermediate', true),
('document-manager', 'Document Manager', 'Intelligent document processing and management', '/scripts/scenarios/core/document-manager', ARRAY['postgres', 'qdrant', 'n8n'], ARRAY['storage', 'automation'], 'intermediate', true),
('brand-manager', 'Brand Manager', 'Brand asset generation and management platform', '/scripts/scenarios/core/brand-manager', ARRAY['postgres', 'minio', 'comfyui', 'n8n'], ARRAY['storage', 'ai', 'automation'], 'advanced', true),
('audio-intelligence-platform', 'Audio Intelligence Platform', 'Audio processing and transcription platform', '/scripts/scenarios/core/audio-intelligence-platform', ARRAY['postgres', 'whisper', 'qdrant', 'n8n'], ARRAY['storage', 'ai', 'automation'], 'advanced', true),
('image-generation-pipeline', 'Image Generation Pipeline', 'AI-powered image creation and management', '/scripts/scenarios/core/image-generation-pipeline', ARRAY['postgres', 'comfyui', 'minio'], ARRAY['storage', 'ai'], 'intermediate', true),
('prompt-manager', 'Prompt Manager', 'AI prompt management and optimization platform', '/scripts/scenarios/core/prompt-manager', ARRAY['postgres'], ARRAY['storage'], 'simple', true),
('scenario-generator-v1', 'Scenario Generator V1', 'AI-powered scenario creation system', '/scripts/scenarios/core/scenario-generator-v1', ARRAY['postgres', 'claude-code'], ARRAY['storage', 'ai'], 'intermediate', false);

-- Insert some sample experiment records for demonstration
INSERT INTO experiments (name, description, prompt, target_scenario, new_resource, status) VALUES
('Add Redis to Analytics Dashboard', 'Integrate Redis caching to improve dashboard performance', 'Add Redis caching capabilities to the analytics-dashboard scenario to speed up data retrieval and enable real-time updates', 'analytics-dashboard', 'redis', 'completed'),
('Add Vector Search to Document Manager', 'Enhance document search with Qdrant vector database', 'Integrate Qdrant vector search into the document-manager scenario to enable semantic document search and similarity matching', 'document-manager', 'qdrant', 'completed'),
('Add MinIO to Idea Generator', 'Add file storage capabilities for generated content', 'Integrate MinIO object storage into the idea-generator scenario to store generated documents, images, and other assets', 'idea-generator', 'minio', 'requested');