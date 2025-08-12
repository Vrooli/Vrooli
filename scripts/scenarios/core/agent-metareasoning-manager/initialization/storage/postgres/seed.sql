-- Agent Metareasoning Manager Seed Data
-- Initial workflow configurations and example patterns

-- Insert built-in workflows from the original workflows.json
INSERT INTO workflows (
    name,
    description,
    type,
    platform,
    webhook_path,
    job_path,
    schema,
    estimated_duration_ms,
    config,
    is_builtin,
    is_active,
    tags,
    created_by
) VALUES 
    ('pros-cons',
     'Weighted advantages vs disadvantages with scoring',
     'pros-cons',
     'n8n',
     'analyze-pros-cons',
     NULL,
     '{"input": {"required": ["input"], "optional": ["context", "model"]}, "output": {"analysis": {"pros": "array of {item, weight, explanation}", "cons": "array of {item, weight, explanation}", "recommendation": "string", "confidence": "number 0-100"}}}'::jsonb,
     8000,
     '{"name": "Pros & Cons Analysis", "webhook_trigger": true, "model_default": "llama3.2", "temperature": 0.5, "max_tokens": 2000}'::jsonb,
     true,
     true,
     ARRAY['decision-making', 'evaluation', 'comparison', 'analytical'],
     'system'),
     
    ('swot',
     'Strategic Strengths, Weaknesses, Opportunities, Threats analysis',
     'swot',
     'n8n',
     'swot-analysis',
     NULL,
     '{"input": {"required": ["input"], "optional": ["context", "model"]}, "output": {"swot_analysis": {"strengths": "array of strategic strengths", "weaknesses": "array of strategic weaknesses", "opportunities": "array of external opportunities", "threats": "array of external threats"}, "strategic_position": "string assessment"}}'::jsonb,
     15000,
     '{"name": "SWOT Analysis", "webhook_trigger": true, "model_default": "llama3.2", "temperature": 0.5, "max_tokens": 2500}'::jsonb,
     true,
     true,
     ARRAY['strategic', 'planning', 'analysis', 'business'],
     'system'),
     
    ('risk-assessment',
     'Probability × Impact analysis with mitigation strategies',
     'risk-assessment',
     'n8n',
     'risk-assessment',
     NULL,
     '{"input": {"required": ["action"], "optional": ["constraints", "model"]}, "output": {"risk_assessment": {"risks": "array of risk objects with probability/impact/mitigation"}, "risk_metrics": {"total_risks": "number", "overall_risk_posture": "string"}}}'::jsonb,
     18000,
     '{"name": "Risk Assessment", "webhook_trigger": true, "model_default": "llama3.2", "temperature": 0.4, "max_tokens": 3000}'::jsonb,
     true,
     true,
     ARRAY['risk', 'mitigation', 'assessment', 'planning'],
     'system'),
     
    ('self-review',
     'Recursive reasoning validation and improvement',
     'self-review',
     'n8n',
     'self-review',
     NULL,
     '{"input": {"required": ["decision"], "optional": ["model", "iterations"]}, "output": {"review_process": {"initial_confidence": "number", "final_assessment": "object with improved reasoning"}, "learning_outcomes": {"iteration_count": "number", "improvement_areas": "array"}}}'::jsonb,
     25000,
     '{"name": "Self-Review Loop", "webhook_trigger": true, "model_default": "llama3.2", "temperature": 0.6, "max_tokens": 3500, "iterations_default": 3}'::jsonb,
     true,
     true,
     ARRAY['metacognitive', 'self-improvement', 'validation', 'iterative'],
     'system'),
     
    ('reasoning-chain',
     'Orchestrated execution of multiple reasoning patterns',
     'reasoning-chain',
     'n8n',
     'reasoning-chain',
     NULL,
     '{"input": {"required": ["input"], "optional": ["chain_type", "model", "custom_steps"]}, "output": {"chain_execution": {"results": "array of step results", "status": "string"}, "synthesis": "object with combined insights"}}'::jsonb,
     45000,
     '{"name": "Multi-Step Reasoning Chain", "webhook_trigger": true, "model_default": "llama3.2", "temperature": 0.5, "max_tokens": 4000}'::jsonb,
     true,
     true,
     ARRAY['orchestration', 'complex', 'multi-step', 'comprehensive'],
     'system'),
     
    ('decision',
     'Comprehensive decision framework with multiple factors',
     'decision',
     'windmill',
     NULL,
     'decision-analyzer',
     '{"input": {"required": ["input"], "optional": ["factors", "weights", "model"]}, "output": {"decision_analysis": {"factors_analysis": "object", "recommendation": "string", "confidence": "number", "reasoning": "string"}}}'::jsonb,
     12000,
     '{"name": "Decision Analysis", "job_based": true, "model_default": "llama3.2", "temperature": 0.5, "max_tokens": 2500}'::jsonb,
     true,
     true,
     ARRAY['decision-making', 'multi-factor', 'weighted', 'analytical'],
     'system')
ON CONFLICT (name, version) DO NOTHING;

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
-- Note: Uses workflow_id references from the workflows table
INSERT INTO execution_history (
    workflow_id,
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
    ((SELECT id FROM workflows WHERE name = 'pros-cons' AND version = 1),
     'n8n', 'demo-001', 'pros-cons',
     '{"input": "Should we migrate to microservices?", "context": "Legacy monolith with 5-person team"}'::jsonb,
     '{"pros": [{"item": "Better scalability", "weight": 8}, {"item": "Independent deployments", "weight": 7}], "cons": [{"item": "Increased complexity", "weight": 9}, {"item": "Team size concerns", "weight": 8}], "recommendation": "Consider gradual migration", "confidence": 75}'::jsonb,
     3200, 'success', 'llama3.2',
     '{"demo": true, "category": "architecture"}'::jsonb),
    
    ((SELECT id FROM workflows WHERE name = 'swot' AND version = 1),
     'n8n', 'demo-002', 'swot',
     '{"input": "New SaaS product launch", "context": "B2B market, established competitors"}'::jsonb,
     '{"strengths": ["Innovative features", "Strong team"], "weaknesses": ["Limited marketing budget", "No brand recognition"], "opportunities": ["Growing market", "Partnership potential"], "threats": ["Established competitors", "Economic uncertainty"], "strategic_position": "Differentiation strategy recommended"}'::jsonb,
     5500, 'success', 'llama3.2',
     '{"demo": true, "category": "product-strategy"}'::jsonb),
    
    ((SELECT id FROM workflows WHERE name = 'risk-assessment' AND version = 1),
     'n8n', 'demo-003', 'risk-assessment',
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