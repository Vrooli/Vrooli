-- Tech Tree Designer Database Schema
-- Stores the complete civilization technology roadmap from basic tools to digital twins

-- Main tech tree container
CREATE TABLE IF NOT EXISTS tech_trees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(128) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    tree_type VARCHAR(50) NOT NULL DEFAULT 'official', -- official, experimental, draft
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, archived, proposed
    is_active BOOLEAN DEFAULT true,
    parent_tree_id UUID REFERENCES tech_trees(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tech_trees_type_status ON tech_trees(tree_type, status);

-- Technology sectors (manufacturing, healthcare, finance, etc.)
CREATE TABLE IF NOT EXISTS sectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tree_id UUID NOT NULL REFERENCES tech_trees(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL, -- manufacturing, healthcare, finance, education, software, governance
    description TEXT,
    progress_percentage FLOAT DEFAULT 0.0 CHECK (progress_percentage >= 0.0 AND progress_percentage <= 100.0),
    position_x FLOAT DEFAULT 0.0,
    position_y FLOAT DEFAULT 0.0,
    color VARCHAR(7) DEFAULT '#3B82F6', -- hex color for visualization
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tree_id, name)
);

-- Progression stages within each sector (foundation → operations → analytics → integration → digital_twin)
-- Supports hierarchical relationships: stages can have child stages for arbitrary depth trees
CREATE TABLE IF NOT EXISTS progression_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sector_id UUID NOT NULL REFERENCES sectors(id) ON DELETE CASCADE,
    parent_stage_id UUID REFERENCES progression_stages(id) ON DELETE CASCADE, -- Hierarchical parent (NULL = root-level)
    stage_type VARCHAR(50) NOT NULL, -- foundation, operational, analytics, integration, digital_twin
    stage_order INTEGER NOT NULL, -- 1-5 for the standard progression
    name VARCHAR(255) NOT NULL,
    description TEXT,
    progress_percentage FLOAT DEFAULT 0.0 CHECK (progress_percentage >= 0.0 AND progress_percentage <= 100.0),
    examples JSONB DEFAULT '[]'::jsonb, -- Array of example systems/tools for this stage
    position_x FLOAT DEFAULT 0.0,
    position_y FLOAT DEFAULT 0.0,
    has_children BOOLEAN DEFAULT false, -- Denormalized flag for UI performance
    children_loaded BOOLEAN DEFAULT false, -- Client-side lazy loading tracking
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.table_constraints
        WHERE constraint_name = 'progression_stages_sector_id_stage_type_key'
          AND table_name = 'progression_stages'
    ) THEN
        ALTER TABLE progression_stages
        DROP CONSTRAINT progression_stages_sector_id_stage_type_key;
    END IF;
END;
$$;

-- Dependencies between stages (within and across sectors)
CREATE TABLE IF NOT EXISTS stage_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dependent_stage_id UUID NOT NULL REFERENCES progression_stages(id) ON DELETE CASCADE,
    prerequisite_stage_id UUID NOT NULL REFERENCES progression_stages(id) ON DELETE CASCADE,
    dependency_type VARCHAR(50) DEFAULT 'required', -- required, recommended, optional
    dependency_strength FLOAT DEFAULT 1.0 CHECK (dependency_strength >= 0.0 AND dependency_strength <= 1.0),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(dependent_stage_id, prerequisite_stage_id),
    CHECK (dependent_stage_id != prerequisite_stage_id)
);

-- Mapping of Vrooli scenarios to tech tree stages
CREATE TABLE IF NOT EXISTS scenario_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name VARCHAR(255) NOT NULL,
    stage_id UUID NOT NULL REFERENCES progression_stages(id) ON DELETE CASCADE,
    contribution_weight FLOAT DEFAULT 1.0 CHECK (contribution_weight >= 0.0 AND contribution_weight <= 1.0),
    completion_status VARCHAR(50) DEFAULT 'not_started', -- not_started, in_progress, completed, blocked
    priority INTEGER DEFAULT 1 CHECK (priority >= 1 AND priority <= 5), -- 1=P0 (critical), 5=P2 (nice to have)
    estimated_impact FLOAT DEFAULT 1.0 CHECK (estimated_impact >= 0.0 AND estimated_impact <= 10.0),
    last_status_check TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scenario_name, stage_id)
);

-- Cross-sector connections showing how progress in one area enables others
CREATE TABLE IF NOT EXISTS sector_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_sector_id UUID NOT NULL REFERENCES sectors(id) ON DELETE CASCADE,
    target_sector_id UUID NOT NULL REFERENCES sectors(id) ON DELETE CASCADE,
    connection_type VARCHAR(50) NOT NULL, -- data_flow, capability_enablement, resource_sharing
    strength FLOAT DEFAULT 1.0 CHECK (strength >= 0.0 AND strength <= 1.0),
    description TEXT,
    examples JSONB DEFAULT '[]'::jsonb, -- Examples of how these sectors connect
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_sector_id, target_sector_id),
    CHECK (source_sector_id != target_sector_id)
);

-- Strategic milestones on the path to superintelligence
CREATE TABLE IF NOT EXISTS strategic_milestones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tree_id UUID NOT NULL REFERENCES tech_trees(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    milestone_type VARCHAR(50) NOT NULL, -- sector_complete, cross_sector_integration, civilization_twin, meta_simulation
    required_sectors JSONB DEFAULT '[]'::jsonb, -- Array of sector IDs required for this milestone
    required_stages JSONB DEFAULT '[]'::jsonb, -- Array of stage IDs required for this milestone
    completion_percentage FLOAT DEFAULT 0.0 CHECK (completion_percentage >= 0.0 AND completion_percentage <= 100.0),
    estimated_completion_date DATE,
    confidence_level FLOAT DEFAULT 0.5 CHECK (confidence_level >= 0.0 AND confidence_level <= 1.0),
    business_value_estimate BIGINT DEFAULT 0, -- Estimated dollar value upon completion
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Analysis results and recommendations from AI strategic planning
CREATE TABLE IF NOT EXISTS strategic_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tree_id UUID NOT NULL REFERENCES tech_trees(id) ON DELETE CASCADE,
    analysis_type VARCHAR(50) NOT NULL, -- path_optimization, bottleneck_identification, impact_analysis
    input_parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    recommendations JSONB NOT NULL DEFAULT '[]'::jsonb,
    confidence_scores JSONB DEFAULT '{}'::jsonb,
    analysis_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    model_used VARCHAR(100), -- Which AI model generated this analysis
    execution_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Progress tracking events for audit trail
CREATE TABLE IF NOT EXISTS progress_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL, -- scenario_started, scenario_completed, stage_updated, milestone_reached
    entity_type VARCHAR(50) NOT NULL, -- scenario, stage, sector, milestone
    entity_id UUID NOT NULL,
    old_value JSONB,
    new_value JSONB,
    change_reason TEXT,
    automated BOOLEAN DEFAULT false, -- true if updated by system, false if manual
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system'
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_sectors_tree_id ON sectors(tree_id);
CREATE INDEX IF NOT EXISTS idx_sectors_category ON sectors(category);
CREATE INDEX IF NOT EXISTS idx_progression_stages_sector_id ON progression_stages(sector_id);
CREATE INDEX IF NOT EXISTS idx_progression_stages_parent_id ON progression_stages(parent_stage_id);
CREATE INDEX IF NOT EXISTS idx_progression_stages_stage_type ON progression_stages(stage_type);
CREATE INDEX IF NOT EXISTS idx_scenario_mappings_scenario_name ON scenario_mappings(scenario_name);
CREATE INDEX IF NOT EXISTS idx_scenario_mappings_stage_id ON scenario_mappings(stage_id);
CREATE INDEX IF NOT EXISTS idx_scenario_mappings_completion_status ON scenario_mappings(completion_status);
CREATE INDEX IF NOT EXISTS idx_stage_dependencies_dependent ON stage_dependencies(dependent_stage_id);
CREATE INDEX IF NOT EXISTS idx_stage_dependencies_prerequisite ON stage_dependencies(prerequisite_stage_id);
CREATE INDEX IF NOT EXISTS idx_strategic_analyses_tree_id ON strategic_analyses(tree_id);
CREATE INDEX IF NOT EXISTS idx_strategic_analyses_timestamp ON strategic_analyses(analysis_timestamp);
CREATE INDEX IF NOT EXISTS idx_progress_events_entity ON progress_events(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_progress_events_timestamp ON progress_events(created_at);

-- Function to automatically update progress percentages based on scenario completion
CREATE OR REPLACE FUNCTION update_stage_progress()
RETURNS TRIGGER AS $$
BEGIN
    -- Recalculate progress for the affected stage
    UPDATE progression_stages 
    SET progress_percentage = (
        SELECT COALESCE(
            SUM(
                CASE 
                    WHEN sm.completion_status = 'completed' THEN sm.contribution_weight * 100
                    WHEN sm.completion_status = 'in_progress' THEN sm.contribution_weight * 50
                    ELSE 0
                END
            ) / NULLIF(SUM(sm.contribution_weight), 0), 
            0
        )
        FROM scenario_mappings sm 
        WHERE sm.stage_id = COALESCE(NEW.stage_id, OLD.stage_id)
    ),
    updated_at = CURRENT_TIMESTAMP
    WHERE id = COALESCE(NEW.stage_id, OLD.stage_id);
    
    -- Update sector progress based on stage progress
    UPDATE sectors 
    SET progress_percentage = (
        SELECT COALESCE(AVG(ps.progress_percentage), 0)
        FROM progression_stages ps 
        WHERE ps.sector_id = sectors.id
    ),
    updated_at = CURRENT_TIMESTAMP
    WHERE id = (
        SELECT ps.sector_id 
        FROM progression_stages ps 
        WHERE ps.id = COALESCE(NEW.stage_id, OLD.stage_id)
    );
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic progress calculation
CREATE TRIGGER trigger_update_stage_progress
    AFTER INSERT OR UPDATE OR DELETE ON scenario_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_stage_progress();

-- Function to log progress events
CREATE OR REPLACE FUNCTION log_progress_event()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO progress_events (event_type, entity_type, entity_id, new_value, automated)
        VALUES ('scenario_mapped', 'scenario', NEW.id, to_jsonb(NEW), true);
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.completion_status != NEW.completion_status THEN
            INSERT INTO progress_events (event_type, entity_type, entity_id, old_value, new_value, automated)
            VALUES ('scenario_status_changed', 'scenario', NEW.id, to_jsonb(OLD), to_jsonb(NEW), true);
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO progress_events (event_type, entity_type, entity_id, old_value, automated)
        VALUES ('scenario_unmapped', 'scenario', OLD.id, to_jsonb(OLD), true);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger for progress event logging
CREATE TRIGGER trigger_log_progress_event
    AFTER INSERT OR UPDATE OR DELETE ON scenario_mappings
    FOR EACH ROW
    EXECUTE FUNCTION log_progress_event();

-- Hierarchical stage support: has_children flag maintenance
CREATE OR REPLACE FUNCTION update_parent_has_children()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.parent_stage_id IS NOT NULL THEN
        UPDATE progression_stages
        SET has_children = true
        WHERE id = NEW.parent_stage_id;
    ELSIF TG_OP = 'DELETE' AND OLD.parent_stage_id IS NOT NULL THEN
        UPDATE progression_stages
        SET has_children = EXISTS(
            SELECT 1 FROM progression_stages
            WHERE parent_stage_id = OLD.parent_stage_id AND id != OLD.id
        )
        WHERE id = OLD.parent_stage_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.parent_stage_id IS DISTINCT FROM NEW.parent_stage_id THEN
        IF OLD.parent_stage_id IS NOT NULL THEN
            UPDATE progression_stages
            SET has_children = EXISTS(
                SELECT 1 FROM progression_stages
                WHERE parent_stage_id = OLD.parent_stage_id AND id != NEW.id
            )
            WHERE id = OLD.parent_stage_id;
        END IF;
        IF NEW.parent_stage_id IS NOT NULL THEN
            UPDATE progression_stages
            SET has_children = true
            WHERE id = NEW.parent_stage_id;
        END IF;
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_parent_has_children
    AFTER INSERT OR UPDATE OR DELETE ON progression_stages
    FOR EACH ROW
    EXECUTE FUNCTION update_parent_has_children();

-- Prevent circular dependencies in stage hierarchy
CREATE OR REPLACE FUNCTION check_stage_hierarchy_cycle()
RETURNS TRIGGER AS $$
DECLARE
    current_id UUID;
    depth INTEGER := 0;
    max_depth INTEGER := 100;
BEGIN
    IF NEW.parent_stage_id IS NULL THEN
        RETURN NEW;
    END IF;
    IF NEW.id = NEW.parent_stage_id THEN
        RAISE EXCEPTION 'A stage cannot be its own parent';
    END IF;
    current_id := NEW.parent_stage_id;
    WHILE current_id IS NOT NULL AND depth < max_depth LOOP
        IF current_id = NEW.id THEN
            RAISE EXCEPTION 'Circular dependency detected in stage hierarchy';
        END IF;
        SELECT parent_stage_id INTO current_id
        FROM progression_stages
        WHERE id = current_id;
        depth := depth + 1;
    END LOOP;
    IF depth >= max_depth THEN
        RAISE EXCEPTION 'Stage hierarchy depth exceeds maximum of %', max_depth;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_check_stage_hierarchy_cycle
    BEFORE INSERT OR UPDATE ON progression_stages
    FOR EACH ROW
    EXECUTE FUNCTION check_stage_hierarchy_cycle();
