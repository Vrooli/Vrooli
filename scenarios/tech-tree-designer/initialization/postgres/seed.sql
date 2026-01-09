-- Tech Tree Designer Seed Data
-- Populates the complete civilization technology roadmap

-- Insert the main civilization tech tree and a draft sandbox
INSERT INTO tech_trees (id, slug, name, description, version, tree_type, status, is_active)
VALUES
('550e8400-e29b-41d4-a716-446655440000', 'civilization-official', 'Civilization Digital Twin Roadmap', 
 'Complete technology development pathway from individual productivity tools to civilization-scale digital twins and meta-simulations', 
 '1.0.0', 'official', 'active', true),
('660e8400-e29b-41d4-a716-446655440000', 'civilization-draft-alpha', 'Civilization Draft Alpha',
 'Sandbox tree cloned from the official roadmap for experimentation',
 '1.0.0', 'draft', 'active', false);

UPDATE tech_trees SET parent_tree_id = '550e8400-e29b-41d4-a716-446655440000'
WHERE id = '660e8400-e29b-41d4-a716-446655440000';

-- Insert core technology sectors
INSERT INTO sectors (id, tree_id, name, category, description, position_x, position_y, color) VALUES
-- Individual & Personal Tools (Foundation Layer)
('550e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440000', 'Personal Productivity', 'individual', 
 'Individual tools for task management, note-taking, time tracking, and personal optimization', 
 100, 100, '#10B981'),

-- Business & Organizational Systems  
('550e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440000', 'Manufacturing Systems', 'manufacturing',
 'Complete manufacturing ecosystem from design to production to supply chain optimization',
 300, 200, '#3B82F6'),
 
('550e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440000', 'Healthcare Systems', 'healthcare',
 'Healthcare technology progression from records management to population health digital twins',
 500, 200, '#EF4444'),
 
('550e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440000', 'Financial Systems', 'finance',
 'Financial technology ecosystem from accounting to market simulation and economic modeling',
 700, 200, '#F59E0B'),
 
('550e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440000', 'Education Systems', 'education',
 'Educational technology from learning management to adaptive curriculum and skill development',
 900, 200, '#8B5CF6'),

-- Governance & Society
('550e8400-e29b-41d4-a716-446655440006', '550e8400-e29b-41d4-a716-446655440000', 'Governance Systems', 'governance',
 'Government and civic technology from document management to policy simulation and democratic innovation',
 1100, 200, '#06B6D4'),

-- Science & Research
('550e8400-e29b-41d4-a716-446655440007', '550e8400-e29b-41d4-a716-446655440000', 'Research & Science', 'science',
 'Scientific research tools from data collection to hypothesis testing to knowledge synthesis',
 300, 400, '#84CC16'),

-- Software Engineering (Meta-capability)
('550e8400-e29b-41d4-a716-446655440008', '550e8400-e29b-41d4-a716-446655440000', 'Software Engineering', 'software',
 'Software development tools and platforms that accelerate all other domains',
 100, 300, '#EC4899');

-- Insert progression stages for Personal Productivity sector
INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description, examples, position_x, position_y) VALUES
-- Personal Productivity stages
('550e8400-e29b-41d4-a716-446655441001', '550e8400-e29b-41d4-a716-446655440001', 'foundation', 1, 
 'Individual Task Management', 
 'Basic tools for managing personal tasks, notes, calendars, and productivity',
 '["task-planner", "notes", "calendar", "time-tracker"]'::jsonb,
 120, 80),
 
('550e8400-e29b-41d4-a716-446655441002', '550e8400-e29b-41d4-a716-446655440001', 'operational', 2,
 'Personal Automation', 
 'Automated workflows for routine personal tasks and decision-making',
 '["habit-tracker", "routine-automator", "decision-assistant", "personal-dashboard"]'::jsonb,
 120, 100),
 
('550e8400-e29b-41d4-a716-446655441003', '550e8400-e29b-41d4-a716-446655440001', 'analytics', 3,
 'Self-Analytics', 
 'Personal performance tracking, habit analysis, and self-optimization',
 '["personal-metrics", "habit-analyzer", "goal-tracker", "productivity-insights"]'::jsonb,
 120, 120),
 
('550e8400-e29b-41d4-a716-446655441004', '550e8400-e29b-41d4-a716-446655440001', 'integration', 4,
 'Life Integration', 
 'Integrated personal ecosystem connecting work, health, finance, and relationships',
 '["life-dashboard", "personal-crm", "health-finance-integration", "unified-assistant"]'::jsonb,
 120, 140),
 
('550e8400-e29b-41d4-a716-446655441005', '550e8400-e29b-41d4-a716-446655440001', 'digital_twin', 5,
 'Personal Digital Twin', 
 'Complete personal simulation for life optimization and decision modeling',
 '["personal-digital-twin", "life-simulator", "decision-modeling", "future-planning"]'::jsonb,
 120, 160);

-- Manufacturing System stages
INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description, examples, position_x, position_y) VALUES
('550e8400-e29b-41d4-a716-446655442001', '550e8400-e29b-41d4-a716-446655440002', 'foundation', 1,
 'Product Lifecycle Management',
 'Core systems for product design, documentation, and lifecycle management',
 '["PLM", "CAD systems", "product-catalog", "design-repository"]'::jsonb,
 320, 180),
 
('550e8400-e29b-41d4-a716-446655442002', '550e8400-e29b-41d4-a716-446655440002', 'operational', 2,
 'Manufacturing Execution',
 'Real-time production control, quality management, and equipment monitoring',
 '["MES", "SCADA", "quality-control", "equipment-monitoring"]'::jsonb,
 320, 200),
 
('550e8400-e29b-41d4-a716-446655442003', '550e8400-e29b-41d4-a716-446655440002', 'analytics', 3,
 'Production Analytics',
 'Manufacturing intelligence, performance optimization, and predictive maintenance',
 '["production-dashboards", "predictive-maintenance", "yield-optimization", "cost-analysis"]'::jsonb,
 320, 220),
 
('550e8400-e29b-41d4-a716-446655442004', '550e8400-e29b-41d4-a716-446655440002', 'integration', 4,
 'Industrial IoT Integration',
 'Connected factory with supply chain integration and automated orchestration',
 '["IIoT platform", "supply-chain-integration", "automated-logistics", "vendor-portals"]'::jsonb,
 320, 240),
 
('550e8400-e29b-41d4-a716-446655442005', '550e8400-e29b-41d4-a716-446655440002', 'digital_twin', 5,
 'Factory Digital Twin',
 'Complete manufacturing simulation with supply chain optimization and scenario planning',
 '["factory-digital-twin", "supply-chain-simulator", "production-optimization", "what-if-analysis"]'::jsonb,
 320, 260);

-- Healthcare System stages
INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description, examples, position_x, position_y) VALUES
('550e8400-e29b-41d4-a716-446655443001', '550e8400-e29b-41d4-a716-446655440003', 'foundation', 1,
 'Electronic Health Records',
 'Patient data management, clinical documentation, and health information systems',
 '["EHR", "patient-portal", "clinical-documentation", "health-records"]'::jsonb,
 520, 180),
 
('550e8400-e29b-41d4-a716-446655443002', '550e8400-e29b-41d4-a716-446655440003', 'operational', 2,
 'Clinical Operations',
 'Hospital management, clinical workflows, and medical device integration',
 '["hospital-management", "clinical-workflows", "medical-devices", "treatment-protocols"]'::jsonb,
 520, 200),
 
('550e8400-e29b-41d4-a716-446655443003', '550e8400-e29b-41d4-a716-446655440003', 'analytics', 3,
 'Clinical Decision Support',
 'Medical analytics, clinical decision support, and population health insights',
 '["clinical-decision-support", "medical-analytics", "population-health", "outcome-analysis"]'::jsonb,
 520, 220),
 
('550e8400-e29b-41d4-a716-446655443004', '550e8400-e29b-41d4-a716-446655440003', 'integration', 4,
 'Health Information Exchange',
 'Interoperable health systems with coordinated care and data sharing',
 '["health-information-exchange", "care-coordination", "telemedicine", "health-networks"]'::jsonb,
 520, 240),
 
('550e8400-e29b-41d4-a716-446655443005', '550e8400-e29b-41d4-a716-446655440003', 'digital_twin', 5,
 'Population Health Twin',
 'Complete health system simulation with epidemic modeling and public health optimization',
 '["population-health-twin", "epidemic-modeling", "healthcare-optimization", "public-health-simulation"]'::jsonb,
 520, 260);

-- Financial System stages  
INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description, examples, position_x, position_y) VALUES
('550e8400-e29b-41d4-a716-446655444001', '550e8400-e29b-41d4-a716-446655440004', 'foundation', 1,
 'Enterprise Resource Planning',
 'Core financial management, accounting, and business resource planning',
 '["ERP", "accounting-system", "financial-planning", "budget-management"]'::jsonb,
 720, 180),
 
('550e8400-e29b-41d4-a716-446655444002', '550e8400-e29b-41d4-a716-446655440004', 'operational', 2,
 'Trading & Risk Systems',
 'Real-time trading, risk management, and regulatory compliance',
 '["trading-systems", "risk-management", "compliance-monitoring", "payment-processing"]'::jsonb,
 720, 200),
 
('550e8400-e29b-41d4-a716-446655444003', '550e8400-e29b-41d4-a716-446655440004', 'analytics', 3,
 'Financial Analytics',
 'Market analysis, portfolio optimization, and financial intelligence',
 '["market-analytics", "portfolio-optimization", "financial-modeling", "credit-scoring"]'::jsonb,
 720, 220),
 
('550e8400-e29b-41d4-a716-446655444004', '550e8400-e29b-41d4-a716-446655440004', 'integration', 4,
 'Financial Ecosystem',
 'Open banking, fintech integration, and automated financial services',
 '["open-banking", "fintech-integration", "automated-advisory", "financial-apis"]'::jsonb,
 720, 240),
 
('550e8400-e29b-41d4-a716-446655444005', '550e8400-e29b-41d4-a716-446655440004', 'digital_twin', 5,
 'Economic System Twin',
 'Complete economic modeling with market simulation and financial system optimization',
 '["economic-system-twin", "market-simulation", "monetary-policy-modeling", "financial-crisis-prediction"]'::jsonb,
 720, 260);

-- Education System stages
INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description, examples, position_x, position_y) VALUES
('550e8400-e29b-41d4-a716-446655445001', '550e8400-e29b-41d4-a716-446655440005', 'foundation', 1,
 'Learning Management',
 'Student information systems, learning management, and educational content',
 '["SIS", "LMS", "educational-content", "student-portal"]'::jsonb,
 920, 180),
 
('550e8400-e29b-41d4-a716-446655445002', '550e8400-e29b-41d4-a716-446655440005', 'operational', 2,
 'Educational Operations',
 'Classroom management, assessment systems, and educational workflows',
 '["classroom-management", "assessment-systems", "grading-automation", "educational-workflows"]'::jsonb,
 920, 200),
 
('550e8400-e29b-41d4-a716-446655445003', '550e8400-e29b-41d4-a716-446655440005', 'analytics', 3,
 'Learning Analytics',
 'Student performance analysis, adaptive learning, and educational intelligence',
 '["learning-analytics", "adaptive-learning", "performance-tracking", "skill-assessment"]'::jsonb,
 920, 220),
 
('550e8400-e29b-41d4-a716-446655445004', '550e8400-e29b-41d4-a716-446655440005', 'integration', 4,
 'Educational Ecosystem',
 'Integrated learning platforms with credential verification and career pathways',
 '["integrated-learning-platform", "credential-verification", "career-pathways", "skill-marketplace"]'::jsonb,
 920, 240),
 
('550e8400-e29b-41d4-a716-446655445005', '550e8400-e29b-41d4-a716-446655440005', 'digital_twin', 5,
 'Education System Twin',
 'Complete educational simulation with curriculum optimization and workforce development',
 '["education-system-twin", "curriculum-optimization", "workforce-development", "skill-prediction"]'::jsonb,
 920, 260);

-- Software Engineering stages (Meta-capability that accelerates all others)
INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description, examples, position_x, position_y) VALUES
('550e8400-e29b-41d4-a716-446655448001', '550e8400-e29b-41d4-a716-446655440008', 'foundation', 1,
 'Development Tools',
 'Core software development tools, version control, and project management',
 '["IDE", "version-control", "project-management", "code-repository"]'::jsonb,
 120, 280),
 
('550e8400-e29b-41d4-a716-446655448002', '550e8400-e29b-41d4-a716-446655440008', 'operational', 2,
 'Development Operations',
 'CI/CD, automated testing, deployment automation, and monitoring',
 '["CI/CD", "automated-testing", "deployment-automation", "monitoring"]'::jsonb,
 120, 300),
 
('550e8400-e29b-41d4-a716-446655448003', '550e8400-e29b-41d4-a716-446655440008', 'analytics', 3,
 'Code Intelligence',
 'Code analysis, performance optimization, and development analytics',
 '["code-analysis", "performance-optimization", "development-analytics", "technical-debt-tracking"]'::jsonb,
 120, 320),
 
('550e8400-e29b-41d4-a716-446655448004', '550e8400-e29b-41d4-a716-446655440008', 'integration', 4,
 'Platform Engineering',
 'Platform-as-a-Service, microservices orchestration, and system integration',
 '["platform-as-service", "microservices-orchestration", "api-management", "system-integration"]'::jsonb,
 120, 340),
 
('550e8400-e29b-41d4-a716-446655448005', '550e8400-e29b-41d4-a716-446655440008', 'digital_twin', 5,
 'Software Development Twin',
 'Complete development process simulation with automated optimization and AI-driven development',
 '["development-process-twin", "automated-development", "ai-code-generation", "system-optimization"]'::jsonb,
 120, 360);

-- Insert key cross-sector dependencies (foundation stages generally depend on software engineering)
INSERT INTO stage_dependencies (dependent_stage_id, prerequisite_stage_id, dependency_type, dependency_strength, description) VALUES
-- Personal productivity depends on software development tools
('550e8400-e29b-41d4-a716-446655441001', '550e8400-e29b-41d4-a716-446655448001', 'required', 0.8, 
 'Personal productivity tools require software development foundation'),

-- Manufacturing foundation depends on software tools
('550e8400-e29b-41d4-a716-446655442001', '550e8400-e29b-41d4-a716-446655448001', 'required', 0.9,
 'Manufacturing PLM systems require robust software development capabilities'),

-- Healthcare EHR depends on software foundation  
('550e8400-e29b-41d4-a716-446655443001', '550e8400-e29b-41d4-a716-446655448001', 'required', 0.9,
 'Electronic health records require mature software development practices'),

-- Financial ERP depends on software foundation
('550e8400-e29b-41d4-a716-446655444001', '550e8400-e29b-41d4-a716-446655448001', 'required', 0.9,
 'Financial ERP systems require sophisticated software engineering'),

-- Education LMS depends on software foundation
('550e8400-e29b-41d4-a716-446655445001', '550e8400-e29b-41d4-a716-446655448001', 'required', 0.8,
 'Learning management systems require software development capabilities'),

-- Digital twin stages depend on integration stages within same sector
('550e8400-e29b-41d4-a716-446655441005', '550e8400-e29b-41d4-a716-446655441004', 'required', 1.0,
 'Personal digital twin requires complete life integration'),
('550e8400-e29b-41d4-a716-446655442005', '550e8400-e29b-41d4-a716-446655442004', 'required', 1.0,
 'Factory digital twin requires IIoT integration'),
('550e8400-e29b-41d4-a716-446655443005', '550e8400-e29b-41d4-a716-446655443004', 'required', 1.0,
 'Population health twin requires health information exchange'),
('550e8400-e29b-41d4-a716-446655444005', '550e8400-e29b-41d4-a716-446655444004', 'required', 1.0,
 'Economic system twin requires financial ecosystem integration'),
('550e8400-e29b-41d4-a716-446655445005', '550e8400-e29b-41d4-a716-446655445004', 'required', 1.0,
 'Education system twin requires educational ecosystem integration');

-- Insert cross-sector connections showing how domains enhance each other
INSERT INTO sector_connections (source_sector_id, target_sector_id, connection_type, strength, description, examples) VALUES
-- Software engineering accelerates all other sectors
('550e8400-e29b-41d4-a716-446655440008', '550e8400-e29b-41d4-a716-446655440002', 'capability_enablement', 0.9,
 'Software engineering provides tools and platforms for manufacturing systems',
 '["PLM software", "manufacturing execution systems", "industrial IoT platforms"]'::jsonb),

('550e8400-e29b-41d4-a716-446655440008', '550e8400-e29b-41d4-a716-446655440003', 'capability_enablement', 0.9,
 'Software engineering enables healthcare technology development',
 '["EHR systems", "medical device software", "telemedicine platforms"]'::jsonb),

-- Healthcare data flows to manufacturing (biotech, medical devices)
('550e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440002', 'data_flow', 0.6,
 'Healthcare insights drive medical device manufacturing and biotech development',
 '["medical device requirements", "biomarker data", "clinical trial results"]'::jsonb),

-- Financial systems enable all business operations
('550e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440002', 'resource_sharing', 0.8,
 'Financial systems provide funding and cost optimization for manufacturing',
 '["supply chain financing", "equipment leasing", "cost optimization"]'::jsonb),

('550e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440003', 'resource_sharing', 0.7,
 'Financial systems enable healthcare funding and insurance processing',
 '["healthcare payments", "insurance processing", "medical billing"]'::jsonb),

-- Education feeds talent into all sectors
('550e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440008', 'capability_enablement', 0.8,
 'Education systems develop software engineering talent and skills',
 '["programming education", "technical training", "skill certification"]'::jsonb);

-- Insert strategic milestones on path to superintelligence
INSERT INTO strategic_milestones (id, tree_id, name, description, milestone_type, required_sectors, business_value_estimate) VALUES
('550e8400-e29b-41d4-a716-446655450001', '550e8400-e29b-41d4-a716-446655440000',
 'Individual Productivity Mastery', 
 'Complete personal productivity and self-optimization capabilities',
 'sector_complete',
 '["550e8400-e29b-41d4-a716-446655440001"]'::jsonb,
 10000000),

('550e8400-e29b-41d4-a716-446655450002', '550e8400-e29b-41d4-a716-446655440000',
 'Core Sector Digital Twins', 
 'Manufacturing, Healthcare, Finance, and Education digital twins operational',
 'cross_sector_integration',
 '["550e8400-e29b-41d4-a716-446655440002", "550e8400-e29b-41d4-a716-446655440003", "550e8400-e29b-41d4-a716-446655440004", "550e8400-e29b-41d4-a716-446655440005"]'::jsonb,
 1000000000),

('550e8400-e29b-41d4-a716-446655450003', '550e8400-e29b-41d4-a716-446655440000',
 'Civilization Digital Twin', 
 'Complete society-scale simulation integrating all sectors',
 'civilization_twin',
 '["550e8400-e29b-41d4-a716-446655440002", "550e8400-e29b-41d4-a716-446655440003", "550e8400-e29b-41d4-a716-446655440004", "550e8400-e29b-41d4-a716-446655440005", "550e8400-e29b-41d4-a716-446655440006"]'::jsonb,
 100000000000),

('550e8400-e29b-41d4-a716-446655450004', '550e8400-e29b-41d4-a716-446655440000',
 'Meta-Civilization Simulation', 
 'Superintelligent system capable of optimizing governance, economics, and society',
 'meta_simulation',
 '["550e8400-e29b-41d4-a716-446655440002", "550e8400-e29b-41d4-a716-446655440003", "550e8400-e29b-41d4-a716-446655440004", "550e8400-e29b-41d4-a716-446655440005", "550e8400-e29b-41d4-a716-446655440006", "550e8400-e29b-41d4-a716-446655440007"]'::jsonb,
 1000000000000);

-- Sample scenario mappings for existing Vrooli scenarios
INSERT INTO scenario_mappings (scenario_name, stage_id, contribution_weight, completion_status, priority, estimated_impact) VALUES
-- Personal productivity scenarios
('task-planner', '550e8400-e29b-41d4-a716-446655441001', 1.0, 'completed', 1, 8.0),
('notes', '550e8400-e29b-41d4-a716-446655441001', 0.8, 'completed', 1, 7.0),
('calendar', '550e8400-e29b-41d4-a716-446655441001', 0.9, 'completed', 1, 7.5),

-- Manufacturing scenarios
('graph-studio', '550e8400-e29b-41d4-a716-446655442001', 0.3, 'completed', 2, 6.0),

-- Healthcare scenarios  
('personal-relationship-manager', '550e8400-e29b-41d4-a716-446655443001', 0.2, 'in_progress', 3, 5.0),

-- Education scenarios
('kids-dashboard', '550e8400-e29b-41d4-a716-446655445001', 0.4, 'completed', 2, 6.5),
('study-buddy', '550e8400-e29b-41d4-a716-446655445001', 0.5, 'completed', 2, 7.0),

-- Software engineering scenarios
('code-smell', '550e8400-e29b-41d4-a716-446655448003', 0.6, 'in_progress', 2, 6.0),
('app-debugger', '550e8400-e29b-41d4-a716-446655448003', 0.7, 'completed', 2, 7.0),

-- Research scenarios
('research-assistant', '550e8400-e29b-41d4-a716-446655447001', 0.8, 'completed', 1, 8.5);

-- Update timestamps
UPDATE tech_trees SET updated_at = CURRENT_TIMESTAMP;
UPDATE sectors SET updated_at = CURRENT_TIMESTAMP;
UPDATE progression_stages SET updated_at = CURRENT_TIMESTAMP;
