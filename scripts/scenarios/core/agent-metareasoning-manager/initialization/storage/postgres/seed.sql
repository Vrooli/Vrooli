-- Agent Metareasoning Manager Seed Data
-- Lightweight metadata registry for workflows deployed to n8n/Windmill

-- Register metadata for workflows that are deployed via resource injection
INSERT INTO workflow_registry (
    platform,
    platform_id,
    name,
    description,
    category,
    tags,
    capabilities,
    input_schema,
    output_schema
) VALUES 
    ('n8n',
     'pros-cons-analyzer',  -- This matches the workflow name in n8n
     'Pros and Cons Analyzer',
     'Weighted advantages vs disadvantages with scoring',
     'analytical',
     ARRAY['decision-making', 'evaluation', 'comparison'],
     '{"models": ["llama3.2", "mistral"], "output_format": "structured_json", "supports_context": true}'::jsonb,
     '{"input": {"type": "string", "required": true}, "context": {"type": "string", "required": false}, "model": {"type": "string", "required": false}}'::jsonb,
     '{"pros": {"type": "array"}, "cons": {"type": "array"}, "recommendation": {"type": "string"}, "confidence": {"type": "number"}}'::jsonb),
     
    ('n8n',
     'swot-analysis',
     'SWOT Analysis',
     'Strategic Strengths, Weaknesses, Opportunities, Threats analysis',
     'strategic',
     ARRAY['strategic', 'planning', 'analysis', 'business'],
     '{"models": ["llama3.2", "mistral"], "output_format": "structured_json", "supports_context": true}'::jsonb,
     '{"input": {"type": "string", "required": true}, "context": {"type": "string", "required": false}}'::jsonb,
     '{"strengths": {"type": "array"}, "weaknesses": {"type": "array"}, "opportunities": {"type": "array"}, "threats": {"type": "array"}}'::jsonb),
     
    ('n8n',
     'risk-assessment',
     'Risk Assessment',
     'Probability × Impact analysis with mitigation strategies',
     'risk',
     ARRAY['risk', 'mitigation', 'assessment', 'planning'],
     '{"models": ["llama3.2", "codellama"], "output_format": "structured_json", "risk_matrix": true}'::jsonb,
     '{"action": {"type": "string", "required": true}, "constraints": {"type": "string", "required": false}}'::jsonb,
     '{"risks": {"type": "array"}, "overall_risk": {"type": "string"}, "mitigations": {"type": "array"}}'::jsonb),
     
    ('n8n',
     'self-review',
     'Self-Review Loop',
     'Recursive reasoning validation and improvement',
     'metacognitive',
     ARRAY['metacognitive', 'self-improvement', 'validation', 'iterative'],
     '{"models": ["llama3.2"], "iterations": 3, "supports_custom_iterations": true}'::jsonb,
     '{"decision": {"type": "string", "required": true}, "iterations": {"type": "number", "required": false}}'::jsonb,
     '{"initial_assessment": {"type": "object"}, "final_assessment": {"type": "object"}, "improvements": {"type": "array"}}'::jsonb),
     
    ('n8n',
     'reasoning-chain',
     'Reasoning Chain',
     'Orchestrated execution of multiple reasoning patterns',
     'orchestration',
     ARRAY['orchestration', 'complex', 'multi-step', 'comprehensive'],
     '{"models": ["llama3.2"], "chaining": true, "parallel_execution": false}'::jsonb,
     '{"input": {"type": "string", "required": true}, "chain_type": {"type": "string", "required": false}}'::jsonb,
     '{"chain_results": {"type": "array"}, "synthesis": {"type": "object"}}'::jsonb),
     
    ('windmill',
     'metareasoning-orchestrator',
     'Metareasoning Orchestrator',
     'Web UI for managing and executing reasoning workflows',
     'ui',
     ARRAY['dashboard', 'orchestration', 'management'],
     '{"type": "app", "ui": true, "api_integration": true}'::jsonb,
     '{"workflow_selection": {"type": "string"}, "input_data": {"type": "object"}}'::jsonb,
     '{"execution_result": {"type": "object"}, "visualization": {"type": "html"}}'::jsonb),
     
    ('windmill',
     'prompt-tester',
     'Prompt Tester',
     'Interactive prompt testing and optimization tool',
     'testing',
     ARRAY['testing', 'optimization', 'prompt-engineering'],
     '{"type": "app", "ui": true, "model_comparison": true}'::jsonb,
     '{"prompt": {"type": "string"}, "test_cases": {"type": "array"}}'::jsonb,
     '{"test_results": {"type": "array"}, "recommendations": {"type": "object"}}'::jsonb)
ON CONFLICT (platform, platform_id) DO NOTHING;

-- Sample search patterns for semantic search
INSERT INTO search_patterns (
    pattern_text,
    matched_workflows
) VALUES
    ('help me make a decision',
     ARRAY[(SELECT id FROM workflow_registry WHERE platform_id = 'pros-cons-analyzer')]),
    ('analyze my business strategy',
     ARRAY[(SELECT id FROM workflow_registry WHERE platform_id = 'swot-analysis')]),
    ('what could go wrong',
     ARRAY[(SELECT id FROM workflow_registry WHERE platform_id = 'risk-assessment')]),
    ('improve my reasoning',
     ARRAY[(SELECT id FROM workflow_registry WHERE platform_id = 'self-review')]),
    ('complex analysis',
     ARRAY[(SELECT id FROM workflow_registry WHERE platform_id = 'reasoning-chain')])
ON CONFLICT (pattern_text) DO NOTHING;

-- Create helpful views for the API

-- View: Available workflows with usage stats
CREATE OR REPLACE VIEW available_workflows AS
SELECT 
    wr.id,
    wr.platform,
    wr.platform_id,
    wr.name,
    wr.description,
    wr.category,
    wr.tags,
    wr.usage_count,
    wr.avg_execution_time_ms,
    wr.last_used,
    CASE 
        WHEN wr.last_used > CURRENT_TIMESTAMP - INTERVAL '1 day' THEN 'recently_used'
        WHEN wr.last_used > CURRENT_TIMESTAMP - INTERVAL '7 days' THEN 'used_this_week'
        WHEN wr.last_used IS NULL THEN 'never_used'
        ELSE 'inactive'
    END as usage_status
FROM workflow_registry wr
ORDER BY wr.usage_count DESC, wr.name;

-- View: Execution performance summary
CREATE OR REPLACE VIEW execution_performance AS
SELECT 
    wr.name as workflow_name,
    wr.platform,
    COUNT(el.id) as total_executions,
    COUNT(CASE WHEN el.status = 'success' THEN 1 END) as successful,
    COUNT(CASE WHEN el.status = 'failed' THEN 1 END) as failed,
    AVG(el.execution_time_ms) FILTER (WHERE el.status = 'success') as avg_time_ms,
    MIN(el.execution_time_ms) FILTER (WHERE el.status = 'success') as min_time_ms,
    MAX(el.execution_time_ms) FILTER (WHERE el.status = 'success') as max_time_ms
FROM workflow_registry wr
LEFT JOIN execution_log el ON wr.id = el.workflow_id
GROUP BY wr.id, wr.name, wr.platform;

-- View: Agent workflow preferences
CREATE OR REPLACE VIEW agent_workflow_usage AS
SELECT 
    ap.agent_id,
    ap.task_type,
    wr.name as preferred_workflow,
    wr.platform,
    ap.preference_score,
    ap.last_used
FROM agent_preferences ap
JOIN workflow_registry wr ON ap.workflow_id = wr.id
ORDER BY ap.agent_id, ap.preference_score DESC;