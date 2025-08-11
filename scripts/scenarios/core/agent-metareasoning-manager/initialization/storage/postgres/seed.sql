-- Agent Metareasoning Manager Seed Data
-- Initial workflow configurations and example patterns

-- Insert initial workflow metrics for all configured workflows
INSERT INTO workflow_metrics (workflow_name, workflow_platform) VALUES 
    ('pros-cons', 'n8n'),
    ('swot', 'n8n'),
    ('risk-assessment', 'n8n'),
    ('self-review', 'n8n'),
    ('reasoning-chain', 'n8n'),
    ('decision', 'windmill')
ON CONFLICT (workflow_name) DO NOTHING;

-- Insert common reasoning patterns
INSERT INTO reasoning_patterns (pattern_name, pattern_type, description, tags) VALUES
    ('Analytical Pros/Cons', 'analytical', 'Weighted analysis of advantages and disadvantages', ARRAY['decision-making', 'evaluation', 'comparison']),
    ('Strategic SWOT', 'strategic', 'Comprehensive SWOT analysis for strategic planning', ARRAY['strategy', 'planning', 'analysis']),
    ('Risk Matrix', 'risk', 'Probability × Impact risk assessment framework', ARRAY['risk', 'mitigation', 'assessment']),
    ('Devil''s Advocate', 'critical', 'Challenge assumptions and find flaws in reasoning', ARRAY['critical-thinking', 'validation', 'challenge']),
    ('Six Thinking Hats', 'creative', 'Multi-perspective analysis using de Bono method', ARRAY['creativity', 'perspectives', 'comprehensive']),
    ('Self-Reflection Loop', 'metacognitive', 'Iterative self-review and improvement', ARRAY['self-improvement', 'validation', 'metacognition']),
    ('Assumption Validator', 'critical', 'Identify and validate underlying assumptions', ARRAY['assumptions', 'validation', 'evidence']),
    ('Bias Detector', 'critical', 'Detect cognitive biases in reasoning', ARRAY['bias', 'objectivity', 'validation']),
    ('Contrarian Analysis', 'critical', 'Generate opposing viewpoints and alternatives', ARRAY['contrarian', 'alternatives', 'challenge'])
ON CONFLICT DO NOTHING;

-- Insert example workflow chains
INSERT INTO workflow_chains (chain_name, chain_description, workflow_sequence) VALUES
    ('Complete Decision Analysis', 
     'Full decision analysis: SWOT → Risk Assessment → Pros/Cons → Self-Review',
     '["swot", "risk-assessment", "pros-cons", "self-review"]'::jsonb),
    ('Quick Evaluation', 
     'Rapid evaluation: Pros/Cons → Risk Assessment',
     '["pros-cons", "risk-assessment"]'::jsonb),
    ('Strategic Planning', 
     'Strategic analysis: SWOT → Risk Assessment → Decision Analysis',
     '["swot", "risk-assessment", "decision"]'::jsonb),
    ('Critical Review', 
     'Critical thinking chain: Assumption Check → Bias Detection → Devil''s Advocate',
     '["assumption-check", "bias-detection", "devils-advocate"]'::jsonb)
ON CONFLICT DO NOTHING;

-- Insert initial model performance baselines
INSERT INTO model_performance (model_name, workflow_type, avg_response_time_ms, quality_score) VALUES
    ('llama3.2', 'pros-cons', 3000, 0.85),
    ('llama3.2', 'swot', 5000, 0.88),
    ('llama3.2', 'risk-assessment', 4500, 0.87),
    ('llama3.2', 'self-review', 6000, 0.82),
    ('mistral', 'pros-cons', 2500, 0.83),
    ('mistral', 'swot', 4000, 0.85),
    ('mistral', 'risk-assessment', 3800, 0.84),
    ('codellama', 'self-review', 5500, 0.86),
    ('codellama', 'risk-assessment', 4000, 0.89)
ON CONFLICT (model_name, workflow_type) DO NOTHING;

-- Sample execution history (for demonstration/testing)
INSERT INTO execution_history (
    resource_type,
    resource_id,
    workflow_type,
    input_data,
    output_data,
    execution_time_ms,
    status,
    model_used,
    metadata
) VALUES
    ('n8n', 'demo-001', 'pros-cons',
     '{"input": "Should we migrate to microservices?", "context": "Legacy monolith with 5-person team"}'::jsonb,
     '{"pros": [{"item": "Better scalability", "weight": 8}, {"item": "Independent deployments", "weight": 7}], "cons": [{"item": "Increased complexity", "weight": 9}, {"item": "Team size concerns", "weight": 8}], "recommendation": "Consider gradual migration", "confidence": 75}'::jsonb,
     3200, 'success', 'llama3.2',
     '{"demo": true, "category": "architecture"}'::jsonb),
    
    ('n8n', 'demo-002', 'swot',
     '{"input": "New SaaS product launch", "context": "B2B market, established competitors"}'::jsonb,
     '{"strengths": ["Innovative features", "Strong team"], "weaknesses": ["Limited marketing budget", "No brand recognition"], "opportunities": ["Growing market", "Partnership potential"], "threats": ["Established competitors", "Economic uncertainty"], "strategic_position": "Differentiation strategy recommended"}'::jsonb,
     5500, 'success', 'llama3.2',
     '{"demo": true, "category": "product-strategy"}'::jsonb),
    
    ('n8n', 'demo-003', 'risk-assessment',
     '{"action": "Cloud migration", "constraints": "6-month timeline, limited budget"}'::jsonb,
     '{"risks": [{"description": "Data loss during migration", "probability": 2, "impact": 5, "mitigation": "Comprehensive backup strategy"}, {"description": "Downtime exceeding SLA", "probability": 3, "impact": 4, "mitigation": "Phased migration approach"}], "overall_risk": "Medium", "recommendation": "Proceed with phased approach"}'::jsonb,
     4800, 'success', 'codellama',
     '{"demo": true, "category": "infrastructure"}'::jsonb)
ON CONFLICT DO NOTHING;

-- Create a view for quick workflow performance overview
CREATE OR REPLACE VIEW workflow_performance_summary AS
SELECT 
    wm.workflow_name,
    wm.workflow_platform,
    wm.total_executions,
    wm.success_rate,
    wm.avg_execution_time_ms,
    wm.last_execution,
    COUNT(DISTINCT mp.model_name) as models_tested,
    AVG(mp.quality_score) as avg_quality_score
FROM workflow_metrics wm
LEFT JOIN model_performance mp ON wm.workflow_name = mp.workflow_type
GROUP BY wm.workflow_name, wm.workflow_platform, wm.total_executions, 
         wm.success_rate, wm.avg_execution_time_ms, wm.last_execution
ORDER BY wm.total_executions DESC;

-- Create a view for recent execution insights
CREATE OR REPLACE VIEW recent_execution_insights AS
SELECT 
    eh.workflow_type,
    eh.model_used,
    eh.status,
    eh.execution_time_ms,
    eh.created_at,
    CASE 
        WHEN eh.execution_time_ms < 3000 THEN 'fast'
        WHEN eh.execution_time_ms < 6000 THEN 'normal'
        ELSE 'slow'
    END as performance_category,
    eh.input_data->>'input' as input_summary
FROM execution_history eh
WHERE eh.created_at > CURRENT_TIMESTAMP - INTERVAL '7 days'
ORDER BY eh.created_at DESC
LIMIT 100;