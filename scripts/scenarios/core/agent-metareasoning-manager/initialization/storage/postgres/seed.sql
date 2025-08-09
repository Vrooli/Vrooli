-- Agent Metareasoning Manager - Initial Data Seed
-- Populates database with sample prompts, workflows, templates, and test data

-- Insert reasoning templates from our template library
INSERT INTO templates (name, description, pattern, structure, example_usage, best_practices, limitations, tags, is_public) VALUES

-- Analytical Templates
('Basic Pros and Cons', 'Simple template for weighing advantages and disadvantages with scoring', 'pros_cons_analysis', 
'{"prompt": "Analyze {{subject}} by listing pros and cons", "sections": ["pros", "cons", "summary"], "output_format": "structured_list", "scoring": true}',
'Should we adopt a 4-day work week? Evaluating cloud providers.', 
'Assign weights 1-10 to each item. Focus on measurable impacts. Consider both short and long-term effects.',
'May oversimplify complex decisions. Weight assignments can be subjective.',
ARRAY['analytical', 'decision-making', 'basic'], true),

('Comprehensive SWOT Analysis', 'Full SWOT analysis with strategic recommendations and positioning', 'swot_analysis',
'{"prompt": "Perform SWOT analysis for {{subject}} in context of {{context}}", "sections": ["strengths", "weaknesses", "opportunities", "threats", "strategy"], "scoring": true, "strategic_positioning": true}',
'Strategic analysis of market entry. Business unit performance evaluation.',
'Include at least 4 items per category. Consider internal vs external factors. Link to strategic actions.',
'Requires good understanding of competitive environment. Can become overly complex.',
ARRAY['strategic', 'business', 'comprehensive'], true),

('Risk Assessment Matrix', 'Evaluate risks by probability and impact with mitigation strategies', 'risk_assessment',
'{"prompt": "Assess risks for {{action}} considering {{constraints}}", "dimensions": ["probability", "impact", "mitigation"], "risk_levels": ["low", "medium", "high", "critical"], "matrix_scoring": true}',
'Project risk assessment. Investment decision risks. Technology adoption risks.',
'Use 5-point scale for probability and impact. Include early warning indicators. Develop mitigation plans.',
'Risk perception can vary. Historical data may not predict future risks accurately.',
ARRAY['risk-management', 'analytical', 'planning'], true),

-- Critical Thinking Templates
('Devils Advocate Challenge', 'Challenge assumptions and find weaknesses in proposals', 'devil_advocate',
'{"prompt": "Challenge the following proposal: {{proposal}}", "approach": "adversarial", "focus_areas": ["assumptions", "logic_gaps", "overlooked_risks", "alternative_views"]}',
'Challenging new product launches. Questioning strategic decisions. Testing business models.',
'Be genuinely adversarial but constructive. Question fundamental assumptions. Consider worst-case scenarios.',
'Can create analysis paralysis. May discourage innovation if overused.',
ARRAY['critical-thinking', 'challenge', 'adversarial'], true),

('Assumption Validator', 'Identify and validate underlying assumptions systematically', 'assumption_checking',
'{"prompt": "Identify assumptions in {{statement}} and evaluate their validity", "validation_criteria": ["evidence", "logic", "precedent", "expert_opinion"], "confidence_scoring": true}',
'Validating research conclusions. Checking strategic plans. Questioning technical designs.',
'Test each assumption independently. Seek contradicting evidence. Use multiple validation methods.',
'Can slow decision-making. Some assumptions may be impossible to fully validate.',
ARRAY['critical-thinking', 'validation', 'systematic'], true),

('Cognitive Bias Scanner', 'Detect common cognitive biases in reasoning processes', 'bias_detection',
'{"prompt": "Analyze {{reasoning}} for cognitive biases", "bias_types": ["confirmation_bias", "anchoring_bias", "availability_heuristic", "dunning_kruger", "sunk_cost_fallacy"], "mitigation_suggestions": true}',
'Investment decision analysis. Hiring process evaluation. Strategic choice review.',
'Check for multiple bias types. Suggest specific mitigation strategies. Consider systematic biases.',
'Bias detection itself can be biased. Over-focus on biases may hinder decisive action.',
ARRAY['bias-detection', 'critical-thinking', 'metacognition'], true),

-- Reflective Templates  
('Self-Review Loop', 'Agent reviews its own reasoning process iteratively', 'introspection',
'{"prompt": "Review your reasoning for {{decision}} and identify potential biases or errors", "steps": ["initial_reasoning", "bias_check", "assumption_validation", "confidence_assessment"], "iteration_count": 3}',
'Major business decisions. Technical recommendations. Strategic choices.',
'Be honest about uncertainty. Iterate until confidence stabilizes. Focus on process improvement.',
'Can become recursive without progress. May reduce confidence unnecessarily.',
ARRAY['introspection', 'self-improvement', 'metacognition'], true),

-- Creative Templates
('Six Thinking Hats', 'Analyze from multiple perspectives using de Bono methodology', 'multi_perspective',
'{"prompt": "Analyze {{topic}} from six perspectives", "perspectives": {"white": "facts and information", "red": "emotions and intuition", "black": "caution and critical thinking", "yellow": "optimism and benefits", "green": "creativity and alternatives", "blue": "process and control"}}',
'Innovation workshops. Complex problem-solving. Team decision processes.',
'Spend equal time on each perspective. Avoid letting one hat dominate. Include diverse viewpoints.',
'Can be time-consuming. May generate too many alternatives without clear prioritization.',
ARRAY['creative', 'multi-perspective', 'systematic'], true),

('Contrarian Perspective', 'Actively seek dissenting opinions and alternatives', 'yes_man_avoidance',
'{"prompt": "Provide contrarian analysis for {{proposal}}", "requirements": ["must_disagree_with_at_least_one_major_point", "suggest_alternative_approach", "identify_overlooked_negatives"]}',
'Testing consensus decisions. Challenging popular strategies. Avoiding groupthink.',
'Must genuinely oppose the proposal. Provide credible alternatives. Be intellectually honest.',
'Can create unnecessary conflict. May delay decisions when speed is important.',
ARRAY['contrarian', 'alternative-thinking', 'challenge'], true);

-- Insert sample prompts based on our template library
INSERT INTO prompts (name, description, category, pattern, content, variables, metadata, tags, usage_count) VALUES

('Standard Pros/Cons Analyzer', 'Basic weighted pros and cons analysis', 'analytical', 'pros_cons_analysis',
'Analyze the following decision or proposal and create a comprehensive pros and cons list:

{{input}}

Context: {{context}}

Provide:
1. At least 5 pros (advantages, benefits, positive outcomes)
2. At least 5 cons (disadvantages, risks, negative outcomes)  
3. Weight each item on importance (1-10)
4. Overall recommendation
5. Confidence level (0-100%)

Format as JSON with structure: {pros: [{item, weight, explanation}], cons: [{item, weight, explanation}], recommendation: string, confidence: number}',
'["input", "context"]',
'{"temperature": 0.5, "max_tokens": 2000, "preferred_models": ["mistral", "llama3.2"]}',
ARRAY['decision-analysis', 'structured', 'scoring'], 15),

('Strategic SWOT Analysis', 'Comprehensive strategic position analysis', 'strategic', 'swot_analysis',
'Perform a detailed SWOT analysis for {{input}} in the context of {{context}}.

**STRENGTHS** (Internal positive factors):
- Identify at least 5 key strengths
- Rate impact level (High/Medium/Low)
- Assess sustainability

**WEAKNESSES** (Internal negative factors):  
- Identify at least 5 key weaknesses
- Rate severity (High/Medium/Low)
- Suggest improvements

**OPPORTUNITIES** (External positive factors):
- Identify at least 4 opportunities
- Rate probability of success
- Estimate timeframe

**THREATS** (External negative factors):
- Identify at least 4 threats  
- Rate risk level and probability
- Suggest mitigation

Determine strategic position and provide 3-5 strategic recommendations.',
'["input", "context"]',
'{"temperature": 0.6, "max_tokens": 2500, "preferred_models": ["llama3.2", "mistral"]}',
ARRAY['strategic-planning', 'business', 'comprehensive'], 8),

('Risk Assessment Framework', 'Systematic risk evaluation with mitigation planning', 'risk-management', 'risk_assessment',
'Conduct comprehensive risk assessment for {{action}} considering {{constraints}}.

For each risk:
1. Risk Description
2. Category (Technical/Financial/Operational/Strategic/Legal/Reputational)  
3. Probability (Very Low/Low/Medium/High/Very High)
4. Impact (Minimal/Minor/Moderate/Major/Critical)
5. Risk Score (Probability × Impact)
6. Current Controls
7. Additional Mitigation
8. Early Warning Signs
9. Contingency Plan

Provide at least 8 distinct risks. Include overall risk posture and top priorities.',
'["action", "constraints"]', 
'{"temperature": 0.4, "max_tokens": 2500, "preferred_models": ["llama3.2", "codellama"]}',
ARRAY['risk-analysis', 'mitigation', 'systematic'], 12);

-- Insert workflow records for our n8n workflows
INSERT INTO workflows (name, description, platform, pattern, workflow_data, input_schema, output_schema, dependencies, tags, execution_count) VALUES

('Pros and Cons Analyzer', 'n8n workflow for weighted pros/cons analysis', 'n8n', 'pros_cons_analysis',
'{"id": "pros-cons-analyzer", "webhook_path": "/webhook/analyze-pros-cons", "nodes": 6, "estimated_duration": 8000}',
'{"required": ["input"], "optional": ["model", "context"]}',
'{"analysis": {"pros": "array", "cons": "array", "recommendation": "string"}, "scores": {"net_score": "number"}}',
'[]',
ARRAY['analytical', 'decision-support', 'scoring'], 0),

('SWOT Analysis Workflow', 'n8n workflow for strategic SWOT analysis', 'n8n', 'swot_analysis', 
'{"id": "swot-analysis", "webhook_path": "/webhook/swot-analysis", "nodes": 6, "estimated_duration": 15000}',
'{"required": ["input"], "optional": ["context", "model"]}',
'{"swot_analysis": {"strengths": "array", "weaknesses": "array", "opportunities": "array", "threats": "array"}, "strategic_position": "string"}',
'[]',
ARRAY['strategic', 'business', 'comprehensive'], 0),

('Risk Assessment Workflow', 'n8n workflow for comprehensive risk analysis', 'n8n', 'risk_assessment',
'{"id": "risk-assessment", "webhook_path": "/webhook/risk-assessment", "nodes": 6, "estimated_duration": 18000}',
'{"required": ["action"], "optional": ["constraints", "model"]}', 
'{"risk_assessment": {"risks": "array"}, "risk_metrics": {"total_risks": "number", "overall_risk_posture": "string"}}',
'[]',
ARRAY['risk-management', 'analysis', 'mitigation'], 0),

('Self-Review Loop', 'n8n workflow for reasoning introspection', 'n8n', 'introspection',
'{"id": "self-review", "webhook_path": "/webhook/self-review", "nodes": 8, "estimated_duration": 25000}',
'{"required": ["decision"], "optional": ["model"]}',
'{"review_process": {"initial_confidence": "number", "final_assessment": "object"}, "learning_outcomes": {"iteration_count": "number"}}',
'[]', 
ARRAY['introspection', 'self-improvement', 'metacognition'], 0),

('Reasoning Chain Orchestrator', 'n8n workflow for multi-step reasoning chains', 'n8n', 'multi_step_reasoning',
'{"id": "reasoning-chain", "webhook_path": "/webhook/reasoning-chain", "nodes": 10, "estimated_duration": 45000}',
'{"required": ["input"], "optional": ["chain_type", "model", "custom_chain"]}',
'{"chain_execution": {"results": "array", "status": "string"}, "synthesis": "object"}',
'["pros-cons-analyzer", "swot-analysis", "risk-assessment", "self-review"]',
ARRAY['orchestration', 'multi-step', 'comprehensive'], 0);

-- Insert reasoning chains configuration
INSERT INTO reasoning_chains (name, description, steps, input_requirements, output_format, use_cases, tags) VALUES

('Comprehensive Analysis Chain', 'Thorough analysis using multiple reasoning approaches', 
'[{"step": "pros_cons", "description": "Initial pros and cons analysis"}, {"step": "devil_advocate", "description": "Challenge the initial analysis"}, {"step": "risk_assessment", "description": "Assess potential risks"}, {"step": "self_review", "description": "Review reasoning process"}]',
'{"input": {"type": "string", "description": "Decision or problem to analyze"}}',
'{"synthesis": "object", "step_results": "array", "confidence": "number"}',
ARRAY['major_decisions', 'strategic_planning', 'complex_problems'],
ARRAY['comprehensive', 'multi-step', 'thorough']),

('Strategic Analysis Chain', 'Business-focused strategic evaluation',
'[{"step": "swot_analysis", "description": "Strategic SWOT analysis"}, {"step": "risk_assessment", "description": "Strategic risk assessment"}, {"step": "pros_cons", "description": "Final pros and cons evaluation"}]',
'{"input": {"type": "string"}, "context": {"type": "string", "description": "Business context"}}',
'{"strategic_recommendation": "string", "swot_summary": "object", "risk_profile": "object"}',
ARRAY['business_strategy', 'market_entry', 'investment_decisions'], 
ARRAY['strategic', 'business', 'planning']),

('Critical Thinking Chain', 'Rigorous critical analysis and assumption validation',
'[{"step": "assumption_check", "description": "Validate underlying assumptions"}, {"step": "bias_detection", "description": "Check for cognitive biases"}, {"step": "devil_advocate", "description": "Challenge from opposing view"}, {"step": "self_review", "description": "Final self-review"}]',
'{"statement": {"type": "string", "description": "Statement or reasoning to analyze"}}',
'{"critical_analysis": "object", "assumption_validity": "array", "bias_assessment": "object"}',
ARRAY['research_validation', 'policy_analysis', 'technical_decisions'],
ARRAY['critical-thinking', 'validation', 'rigorous']),

('Creative Exploration Chain', 'Multi-perspective creative problem-solving',
'[{"step": "multi_perspective", "description": "Six thinking hats analysis"}, {"step": "pros_cons", "description": "Evaluate creative options"}, {"step": "risk_assessment", "description": "Assess implementation risks"}]',
'{"topic": {"type": "string", "description": "Problem or opportunity to explore"}}',
'{"creative_insights": "object", "perspectives": "array", "implementation_plan": "object"}',
ARRAY['innovation', 'problem_solving', 'opportunity_exploration'],
ARRAY['creative', 'multi-perspective', 'innovation']);

-- Insert sample collections
INSERT INTO collections (name, description, type, items, tags, is_public) VALUES

('Decision Analysis Toolkit', 'Essential tools for structured decision-making', 'mixed',
'[{"type": "prompt", "id": "pros-cons-basic"}, {"type": "workflow", "id": "pros-cons-analyzer"}, {"type": "template", "id": "basic-pros-cons"}]',
ARRAY['decision-making', 'toolkit', 'essential'], true),

('Strategic Planning Suite', 'Complete strategic analysis and planning tools', 'mixed',
'[{"type": "prompt", "id": "strategic-swot"}, {"type": "workflow", "id": "swot-analysis"}, {"type": "chain", "id": "strategic-chain"}]',
ARRAY['strategic-planning', 'business', 'suite'], true),

('Critical Thinking Arsenal', 'Tools for rigorous critical analysis and validation', 'prompts',
'[{"type": "prompt", "id": "assumption-validator"}, {"type": "prompt", "id": "bias-scanner"}, {"type": "prompt", "id": "devils-advocate"}]',
ARRAY['critical-thinking', 'validation', 'analysis'], true),

('Metareasoning Fundamentals', 'Core patterns for enhanced reasoning quality', 'templates',
'[{"type": "template", "id": "self-review-loop"}, {"type": "template", "id": "assumption-validator"}, {"type": "template", "id": "bias-scanner"}]',
ARRAY['metareasoning', 'fundamentals', 'core'], true);

-- Insert sample execution history for testing
INSERT INTO execution_history (resource_type, resource_id, input_data, output_data, execution_time_ms, status, model_used, metadata) VALUES

('prompt', (SELECT id FROM prompts WHERE name = 'Standard Pros/Cons Analyzer' LIMIT 1),
'{"input": "Should we adopt remote work permanently?", "context": "Tech company with 50 employees"}',
'{"analysis": {"pros": [{"item": "Increased employee satisfaction", "weight": 8}], "cons": [{"item": "Reduced collaboration", "weight": 6}]}, "recommendation": "Adopt with structured collaboration protocols"}',
7500, 'success', 'llama3.2', 
'{"confidence": 82, "reasoning_quality": 8.5}'),

('workflow', (SELECT id FROM workflows WHERE name = 'SWOT Analysis Workflow' LIMIT 1),
'{"input": "Launching AI-powered customer service", "context": "SaaS company expanding services"}',
'{"swot_analysis": {"strengths": ["Technical expertise", "Customer base"], "opportunities": ["Market demand", "Competitive advantage"]}, "strategic_position": "Aggressive growth"}',
14200, 'success', 'mistral',
'{"strategic_confidence": 87, "implementation_priority": "high"}'),

('chain', (SELECT id FROM reasoning_chains WHERE name = 'Comprehensive Analysis Chain' LIMIT 1),
'{"input": "Acquire competitor vs build in-house capability"}',
'{"synthesis": {"recommendation": "Hybrid approach - acquire key talent, build core IP"}, "confidence": 78, "consensus_level": "high"}',
42000, 'success', 'llama3.2',
'{"steps_completed": 4, "decision_complexity": "high", "stakeholder_alignment": "medium"});

-- Insert sample ratings
INSERT INTO ratings (resource_type, resource_id, rating, review) VALUES

('template', (SELECT id FROM templates WHERE name = 'Basic Pros and Cons' LIMIT 1), 4, 'Simple and effective for straightforward decisions'),
('template', (SELECT id FROM templates WHERE name = 'Comprehensive SWOT Analysis' LIMIT 1), 5, 'Excellent for strategic planning - very thorough'),
('workflow', (SELECT id FROM workflows WHERE name = 'Risk Assessment Workflow' LIMIT 1), 4, 'Good risk identification, could use better mitigation suggestions'),
('prompt', (SELECT id FROM prompts WHERE name = 'Standard Pros/Cons Analyzer' LIMIT 1), 4, 'Well-structured output, consistent results across models');

-- Update usage counts and ratings based on sample data
UPDATE prompts SET usage_count = FLOOR(RANDOM() * 20 + 1), average_rating = ROUND((RANDOM() * 2 + 3)::numeric, 2);
UPDATE templates SET usage_count = FLOOR(RANDOM() * 15 + 1);
UPDATE workflows SET execution_count = FLOOR(RANDOM() * 10 + 1), average_duration_ms = FLOOR(RANDOM() * 20000 + 5000);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_execution_history_recent ON execution_history(executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_templates_usage ON templates(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_prompts_rating ON prompts(average_rating DESC) WHERE average_rating IS NOT NULL;