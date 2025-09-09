-- Graph Studio Database Schema
-- Version: 1.0.0
-- Description: Core schema for graph storage, versioning, and metadata

-- Create database if not exists (run as superuser)
-- CREATE DATABASE graph_studio;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Graphs table: Stores all graph documents
CREATE TABLE IF NOT EXISTS graphs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL, -- Plugin identifier (mind-maps, bpmn, etc.)
    description TEXT,
    data JSONB DEFAULT '{}', -- Graph-specific data structure
    metadata JSONB DEFAULT '{}', -- Additional metadata
    version INTEGER DEFAULT 1,
    created_by VARCHAR(255) DEFAULT 'system',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tags JSONB DEFAULT '[]', -- Array of tags for categorization
    permissions JSONB DEFAULT '{"public": true}', -- Access control
    
    -- Indexes for performance
    INDEX idx_graphs_type ON graphs(type),
    INDEX idx_graphs_created_at ON graphs(created_at DESC),
    INDEX idx_graphs_updated_at ON graphs(updated_at DESC),
    INDEX idx_graphs_tags ON graphs USING gin(tags),
    INDEX idx_graphs_metadata ON graphs USING gin(metadata)
);

-- Graph versions table: Stores version history
CREATE TABLE IF NOT EXISTS graph_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    graph_id UUID NOT NULL REFERENCES graphs(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    data JSONB NOT NULL, -- Snapshot of graph data at this version
    change_description TEXT,
    created_by VARCHAR(255) DEFAULT 'system',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique version numbers per graph
    UNIQUE(graph_id, version_number),
    INDEX idx_graph_versions_graph_id ON graph_versions(graph_id),
    INDEX idx_graph_versions_created_at ON graph_versions(created_at DESC)
);

-- Graph conversions table: Tracks format conversions
CREATE TABLE IF NOT EXISTS graph_conversions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_graph_id UUID REFERENCES graphs(id) ON DELETE SET NULL,
    target_graph_id UUID REFERENCES graphs(id) ON DELETE SET NULL,
    source_format VARCHAR(100) NOT NULL,
    target_format VARCHAR(100) NOT NULL,
    conversion_rules JSONB DEFAULT '{}', -- Rules/options used for conversion
    status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    
    INDEX idx_conversions_source ON graph_conversions(source_graph_id),
    INDEX idx_conversions_target ON graph_conversions(target_graph_id),
    INDEX idx_conversions_status ON graph_conversions(status)
);

-- Graph relationships table: Links between related graphs
CREATE TABLE IF NOT EXISTS graph_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_graph_id UUID NOT NULL REFERENCES graphs(id) ON DELETE CASCADE,
    target_graph_id UUID NOT NULL REFERENCES graphs(id) ON DELETE CASCADE,
    relationship_type VARCHAR(100) NOT NULL, -- parent, child, related, derived_from
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(source_graph_id, target_graph_id, relationship_type),
    INDEX idx_relationships_source ON graph_relationships(source_graph_id),
    INDEX idx_relationships_target ON graph_relationships(target_graph_id),
    INDEX idx_relationships_type ON graph_relationships(relationship_type)
);

-- Plugin registry table: Tracks available plugins and their capabilities
CREATE TABLE IF NOT EXISTS plugins (
    id VARCHAR(100) PRIMARY KEY, -- Plugin identifier
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL, -- visualization, process, architecture, semantic
    description TEXT,
    formats JSONB DEFAULT '[]', -- Supported formats
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 100,
    metadata JSONB DEFAULT '{}', -- Icon, color, capabilities, etc.
    installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_plugins_category ON plugins(category),
    INDEX idx_plugins_enabled ON plugins(enabled)
);

-- Graph templates table: Reusable graph templates
CREATE TABLE IF NOT EXISTS graph_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL, -- Plugin identifier
    description TEXT,
    template_data JSONB NOT NULL, -- Template structure
    category VARCHAR(100), -- business, technical, creative, etc.
    metadata JSONB DEFAULT '{}',
    usage_count INTEGER DEFAULT 0,
    created_by VARCHAR(255) DEFAULT 'system',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_templates_type ON graph_templates(type),
    INDEX idx_templates_category ON graph_templates(category),
    INDEX idx_templates_usage ON graph_templates(usage_count DESC)
);

-- Graph collaborations table: Track collaborative editing sessions
CREATE TABLE IF NOT EXISTS graph_collaborations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    graph_id UUID NOT NULL REFERENCES graphs(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    action VARCHAR(100) NOT NULL, -- view, edit, comment
    changes JSONB DEFAULT '{}', -- Delta/changes made
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_collaborations_graph ON graph_collaborations(graph_id),
    INDEX idx_collaborations_user ON graph_collaborations(user_id),
    INDEX idx_collaborations_session ON graph_collaborations(session_id),
    INDEX idx_collaborations_timestamp ON graph_collaborations(timestamp DESC)
);

-- Graph analytics table: Usage and performance metrics
CREATE TABLE IF NOT EXISTS graph_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    graph_id UUID REFERENCES graphs(id) ON DELETE CASCADE,
    plugin_id VARCHAR(100),
    event_type VARCHAR(100) NOT NULL, -- created, viewed, edited, converted, rendered
    user_id VARCHAR(255),
    metadata JSONB DEFAULT '{}', -- Additional event data
    duration_ms INTEGER, -- Operation duration in milliseconds
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_analytics_graph ON graph_analytics(graph_id),
    INDEX idx_analytics_plugin ON graph_analytics(plugin_id),
    INDEX idx_analytics_event ON graph_analytics(event_type),
    INDEX idx_analytics_timestamp ON graph_analytics(timestamp DESC)
);

-- Insert default plugins
INSERT INTO plugins (id, name, category, description, formats, enabled, priority, metadata)
VALUES 
    ('mind-maps', 'Mind Maps', 'visualization', 'Hierarchical thought organization and brainstorming', 
     '["freemind", "xmind", "mermaid"]'::jsonb, true, 1,
     '{"icon": "brain", "color": "#4CAF50"}'::jsonb),
    
    ('network-graphs', 'Network Graphs', 'visualization', 'Relationship and connection modeling',
     '["cytoscape", "graphml", "gexf", "d3"]'::jsonb, true, 2,
     '{"icon": "share-2", "color": "#2196F3"}'::jsonb),
    
    ('bpmn', 'BPMN 2.0', 'process', 'Business Process Model and Notation',
     '["bpmn", "xml"]'::jsonb, true, 1,
     '{"icon": "git-branch", "color": "#FF9800"}'::jsonb),
    
    ('mermaid', 'Mermaid Diagrams', 'visualization', 'Text-based diagramming and charting',
     '["mermaid", "md"]'::jsonb, true, 3,
     '{"icon": "code", "color": "#9C27B0"}'::jsonb),
    
    ('uml', 'UML Diagrams', 'architecture', 'Unified Modeling Language for software design',
     '["plantuml", "xmi", "mermaid"]'::jsonb, false, 1,
     '{"icon": "box", "color": "#795548"}'::jsonb),
    
    ('dmn', 'DMN', 'process', 'Decision Model and Notation',
     '["dmn", "xml"]'::jsonb, false, 2,
     '{"icon": "git-commit", "color": "#607D8B"}'::jsonb),
    
    ('rdf', 'RDF Graphs', 'semantic', 'Resource Description Framework for semantic web',
     '["turtle", "n-triples", "rdf-xml"]'::jsonb, false, 1,
     '{"icon": "globe", "color": "#00BCD4"}'::jsonb)
ON CONFLICT (id) DO UPDATE
SET 
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    formats = EXCLUDED.formats,
    metadata = EXCLUDED.metadata,
    updated_at = CURRENT_TIMESTAMP;

-- Insert sample templates
INSERT INTO graph_templates (name, type, description, template_data, category, metadata)
VALUES
    ('Project Planning', 'mind-maps', 'Template for project planning and organization',
     '{"root": {"text": "Project Name", "children": [
         {"text": "Goals", "children": []},
         {"text": "Timeline", "children": []},
         {"text": "Resources", "children": []},
         {"text": "Risks", "children": []}
     ]}}'::jsonb,
     'business',
     '{"tags": ["project", "planning", "management"]}'::jsonb),
    
    ('Simple Workflow', 'bpmn', 'Basic workflow with start, tasks, and end',
     '{"nodes": [
         {"id": "start", "type": "startEvent", "label": "Start"},
         {"id": "task1", "type": "task", "label": "Task 1"},
         {"id": "task2", "type": "task", "label": "Task 2"},
         {"id": "end", "type": "endEvent", "label": "End"}
     ], "edges": [
         {"source": "start", "target": "task1"},
         {"source": "task1", "target": "task2"},
         {"source": "task2", "target": "end"}
     ]}'::jsonb,
     'business',
     '{"tags": ["workflow", "process", "basic"]}'::jsonb),
    
    ('Network Topology', 'network-graphs', 'Template for network visualization',
     '{"nodes": [
         {"id": "hub", "label": "Central Hub", "type": "hub"},
         {"id": "node1", "label": "Node 1", "type": "endpoint"},
         {"id": "node2", "label": "Node 2", "type": "endpoint"}
     ], "edges": [
         {"source": "hub", "target": "node1"},
         {"source": "hub", "target": "node2"}
     ]}'::jsonb,
     'technical',
     '{"tags": ["network", "topology", "infrastructure"]}'::jsonb)
ON CONFLICT DO NOTHING;

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_graphs_updated_at BEFORE UPDATE ON graphs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_plugins_updated_at BEFORE UPDATE ON plugins
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON graph_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to automatically version graphs on update
CREATE OR REPLACE FUNCTION version_graph_on_update()
RETURNS TRIGGER AS $$
BEGIN
    -- Only create version if data changed
    IF OLD.data IS DISTINCT FROM NEW.data THEN
        INSERT INTO graph_versions (
            graph_id, version_number, data, 
            change_description, created_by
        ) VALUES (
            NEW.id, NEW.version, OLD.data,
            'Automatic version on update', NEW.created_by
        );
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER version_graph_trigger AFTER UPDATE ON graphs
    FOR EACH ROW EXECUTE FUNCTION version_graph_on_update();

-- Create view for graph statistics
CREATE OR REPLACE VIEW graph_statistics AS
SELECT 
    g.type as plugin_id,
    p.name as plugin_name,
    p.category,
    COUNT(DISTINCT g.id) as total_graphs,
    COUNT(DISTINCT gv.id) as total_versions,
    COUNT(DISTINCT gc.source_graph_id) as total_conversions,
    MAX(g.updated_at) as last_activity
FROM graphs g
LEFT JOIN plugins p ON g.type = p.id
LEFT JOIN graph_versions gv ON g.id = gv.graph_id
LEFT JOIN graph_conversions gc ON g.id = gc.source_graph_id
GROUP BY g.type, p.name, p.category;

-- Grant appropriate permissions (adjust as needed)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO graph_studio_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO graph_studio_user;