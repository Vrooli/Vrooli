-- Migration: Add hierarchical stage support (parent-child relationships)
-- This enables arbitrary-depth tech trees where stages can have children

-- Add parent_stage_id column to progression_stages
ALTER TABLE progression_stages
ADD COLUMN IF NOT EXISTS parent_stage_id UUID REFERENCES progression_stages(id) ON DELETE CASCADE;

-- Add index for parent lookups
CREATE INDEX IF NOT EXISTS idx_progression_stages_parent_id ON progression_stages(parent_stage_id);

-- Add has_children helper column (denormalized for performance)
ALTER TABLE progression_stages
ADD COLUMN IF NOT EXISTS has_children BOOLEAN DEFAULT false;

-- Add children_loaded flag for lazy loading
ALTER TABLE progression_stages
ADD COLUMN IF NOT EXISTS children_loaded BOOLEAN DEFAULT false;

-- Function to update has_children flag when stages are added/removed
CREATE OR REPLACE FUNCTION update_parent_has_children()
RETURNS TRIGGER AS $$
BEGIN
    -- Update parent's has_children flag
    IF TG_OP = 'INSERT' AND NEW.parent_stage_id IS NOT NULL THEN
        UPDATE progression_stages
        SET has_children = true
        WHERE id = NEW.parent_stage_id;
    ELSIF TG_OP = 'DELETE' AND OLD.parent_stage_id IS NOT NULL THEN
        -- Check if parent still has other children
        UPDATE progression_stages
        SET has_children = EXISTS(
            SELECT 1 FROM progression_stages
            WHERE parent_stage_id = OLD.parent_stage_id AND id != OLD.id
        )
        WHERE id = OLD.parent_stage_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.parent_stage_id IS DISTINCT FROM NEW.parent_stage_id THEN
        -- Update old parent
        IF OLD.parent_stage_id IS NOT NULL THEN
            UPDATE progression_stages
            SET has_children = EXISTS(
                SELECT 1 FROM progression_stages
                WHERE parent_stage_id = OLD.parent_stage_id AND id != NEW.id
            )
            WHERE id = OLD.parent_stage_id;
        END IF;
        -- Update new parent
        IF NEW.parent_stage_id IS NOT NULL THEN
            UPDATE progression_stages
            SET has_children = true
            WHERE id = NEW.parent_stage_id;
        END IF;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Trigger for maintaining has_children flag
DROP TRIGGER IF EXISTS trigger_update_parent_has_children ON progression_stages;
CREATE TRIGGER trigger_update_parent_has_children
    AFTER INSERT OR UPDATE OR DELETE ON progression_stages
    FOR EACH ROW
    EXECUTE FUNCTION update_parent_has_children();

-- Backfill has_children for existing data
UPDATE progression_stages p
SET has_children = EXISTS(
    SELECT 1 FROM progression_stages c WHERE c.parent_stage_id = p.id
);

-- Add constraint: prevent circular dependencies (a stage cannot be its own ancestor)
CREATE OR REPLACE FUNCTION check_stage_hierarchy_cycle()
RETURNS TRIGGER AS $$
DECLARE
    current_id UUID;
    depth INTEGER := 0;
    max_depth INTEGER := 100; -- Prevent infinite loops
BEGIN
    IF NEW.parent_stage_id IS NULL THEN
        RETURN NEW;
    END IF;

    -- Cannot be parent of itself
    IF NEW.id = NEW.parent_stage_id THEN
        RAISE EXCEPTION 'A stage cannot be its own parent';
    END IF;

    -- Check for cycles by walking up the tree
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

DROP TRIGGER IF EXISTS trigger_check_stage_hierarchy_cycle ON progression_stages;
CREATE TRIGGER trigger_check_stage_hierarchy_cycle
    BEFORE INSERT OR UPDATE ON progression_stages
    FOR EACH ROW
    EXECUTE FUNCTION check_stage_hierarchy_cycle();

-- Add comment explaining the hierarchical structure
COMMENT ON COLUMN progression_stages.parent_stage_id IS
    'Hierarchical parent stage ID - enables arbitrary-depth tech trees. NULL = root-level stage (direct child of sector)';
COMMENT ON COLUMN progression_stages.has_children IS
    'Denormalized flag indicating if this stage has child stages - optimizes UI rendering';
COMMENT ON COLUMN progression_stages.children_loaded IS
    'Client-side tracking flag for lazy loading - not enforced by database';
