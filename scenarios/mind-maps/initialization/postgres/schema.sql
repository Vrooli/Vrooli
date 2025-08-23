-- Mind Maps Database Schema
-- Stores mind map structures and metadata for visual knowledge organization

-- Mind maps table
CREATE TABLE IF NOT EXISTS mind_maps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    campaign_id VARCHAR(255),
    owner_id VARCHAR(255) NOT NULL DEFAULT 'default-user',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_public BOOLEAN DEFAULT false,
    share_token VARCHAR(255) UNIQUE
);

-- Mind map nodes table
CREATE TABLE IF NOT EXISTS mind_map_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mind_map_id UUID NOT NULL REFERENCES mind_maps(id) ON DELETE CASCADE,
    map_id UUID NOT NULL REFERENCES mind_maps(id) ON DELETE CASCADE, -- Alias for compatibility
    title VARCHAR(255),
    content TEXT NOT NULL,
    node_type VARCHAR(50) DEFAULT 'child', -- root, child, branch, leaf, concept, question, idea
    type VARCHAR(50) DEFAULT 'child', -- Alias for compatibility
    position_x FLOAT DEFAULT 0,
    position_y FLOAT DEFAULT 0,
    parent_id UUID REFERENCES mind_map_nodes(id) ON DELETE CASCADE,
    metadata JSONB DEFAULT '{}', -- color, size, icon, tags, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Connections table for non-hierarchical relationships
CREATE TABLE IF NOT EXISTS mind_map_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mind_map_id UUID NOT NULL REFERENCES mind_maps(id) ON DELETE CASCADE,
    map_id UUID REFERENCES mind_maps(id) ON DELETE CASCADE, -- Alias for compatibility
    from_node_id UUID NOT NULL REFERENCES mind_map_nodes(id) ON DELETE CASCADE,
    to_node_id UUID NOT NULL REFERENCES mind_map_nodes(id) ON DELETE CASCADE,
    relationship_type VARCHAR(100) DEFAULT 'related', -- related, depends_on, conflicts_with, semantic_similarity, etc.
    strength FLOAT DEFAULT 1.0, -- Connection strength (0-1 for semantic similarity)
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_node_id, to_node_id, relationship_type)
);

-- Campaigns table for organizing mind maps
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(255), -- Gradient or hex color
    owner_id VARCHAR(255) NOT NULL DEFAULT 'default-user',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Attachments table for nodes
CREATE TABLE IF NOT EXISTS node_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES mind_map_nodes(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(100),
    file_size INTEGER,
    file_url TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_mind_maps_owner ON mind_maps(owner_id);
CREATE INDEX IF NOT EXISTS idx_mind_maps_campaign ON mind_maps(campaign_id);
CREATE INDEX IF NOT EXISTS idx_mind_maps_public ON mind_maps(is_public);
CREATE INDEX IF NOT EXISTS idx_nodes_mind_map ON mind_map_nodes(mind_map_id);
CREATE INDEX IF NOT EXISTS idx_nodes_parent ON mind_map_nodes(parent_id);
CREATE INDEX IF NOT EXISTS idx_nodes_type ON mind_map_nodes(type);
CREATE INDEX IF NOT EXISTS idx_connections_map ON mind_map_connections(mind_map_id);
CREATE INDEX IF NOT EXISTS idx_connections_from ON mind_map_connections(from_node_id);
CREATE INDEX IF NOT EXISTS idx_connections_to ON mind_map_connections(to_node_id);
CREATE INDEX IF NOT EXISTS idx_campaigns_owner ON campaigns(owner_id);
CREATE INDEX IF NOT EXISTS idx_attachments_node ON node_attachments(node_id);

-- Full text search index on node content
CREATE INDEX IF NOT EXISTS idx_nodes_content_fts ON mind_map_nodes USING gin(to_tsvector('english', content));

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update trigger to tables
CREATE TRIGGER update_mind_maps_updated_at BEFORE UPDATE ON mind_maps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_mind_map_nodes_updated_at BEFORE UPDATE ON mind_map_nodes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_campaigns_updated_at BEFORE UPDATE ON campaigns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
