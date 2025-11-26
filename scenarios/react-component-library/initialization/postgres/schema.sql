-- React Component Library Schema

-- Component libraries (collections of components)
CREATE TABLE IF NOT EXISTS component_libraries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- React components
CREATE TABLE IF NOT EXISTS components (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  library_id UUID NOT NULL REFERENCES component_libraries(id) ON DELETE CASCADE,
  display_name TEXT NOT NULL,
  description TEXT,
  version TEXT NOT NULL,
  file_path TEXT NOT NULL,
  source_path TEXT NOT NULL,
  tags TEXT[] DEFAULT '{}',
  category TEXT,
  tech_stack TEXT[] DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Component versions (historical snapshots)
CREATE TABLE IF NOT EXISTS component_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  component_id UUID NOT NULL REFERENCES components(id) ON DELETE CASCADE,
  version TEXT NOT NULL,
  content TEXT NOT NULL,
  changelog TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(component_id, version)
);

-- Adoption records (tracking which scenarios use which components)
CREATE TABLE IF NOT EXISTS adoption_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  component_id UUID NOT NULL REFERENCES components(id) ON DELETE CASCADE,
  component_library_id UUID NOT NULL REFERENCES component_libraries(id) ON DELETE CASCADE,
  scenario_name TEXT NOT NULL,
  adopted_path TEXT NOT NULL,
  version TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('current', 'behind', 'modified', 'unknown')),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(component_id, scenario_name, adopted_path)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_components_library_id ON components(library_id);
CREATE INDEX IF NOT EXISTS idx_components_category ON components(category);
CREATE INDEX IF NOT EXISTS idx_component_versions_component_id ON component_versions(component_id);
CREATE INDEX IF NOT EXISTS idx_adoption_records_component_id ON adoption_records(component_id);
CREATE INDEX IF NOT EXISTS idx_adoption_records_scenario_name ON adoption_records(scenario_name);

-- Create a default library
INSERT INTO component_libraries (id, name, description)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Library', 'Default component library')
ON CONFLICT DO NOTHING;
