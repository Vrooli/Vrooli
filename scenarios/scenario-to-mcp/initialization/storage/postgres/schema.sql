-- Scenario to MCP Database Schema
-- Manages MCP endpoints, registry, and usage analytics

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS mcp;

-- MCP Endpoints table - tracks all MCP-enabled scenarios
CREATE TABLE IF NOT EXISTS mcp.endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name VARCHAR(255) NOT NULL UNIQUE,
    scenario_path VARCHAR(500) NOT NULL,
    mcp_port INTEGER,
    manifest JSONB,
    status VARCHAR(50) DEFAULT 'inactive' CHECK (status IN ('active', 'inactive', 'error', 'pending')),
    last_health_check TIMESTAMP WITH TIME ZONE,
    health_status JSONB,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    mcp_version VARCHAR(20) DEFAULT '1.0.0',
    auto_start BOOLEAN DEFAULT true,
    capabilities JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- MCP Tool Usage Analytics
CREATE TABLE IF NOT EXISTS mcp.tool_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID REFERENCES mcp.endpoints(id) ON DELETE CASCADE,
    tool_name VARCHAR(255) NOT NULL,
    invocation_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    total_response_time_ms BIGINT DEFAULT 0,
    avg_response_time_ms FLOAT DEFAULT 0,
    last_used TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- MCP Tool Definitions - cached from manifests
CREATE TABLE IF NOT EXISTS mcp.tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID REFERENCES mcp.endpoints(id) ON DELETE CASCADE,
    tool_name VARCHAR(255) NOT NULL,
    description TEXT,
    input_schema JSONB,
    output_schema JSONB,
    examples JSONB DEFAULT '[]'::jsonb,
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(endpoint_id, tool_name)
);

-- MCP Agent Sessions - track Claude-code sessions adding MCP
CREATE TABLE IF NOT EXISTS mcp.agent_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name VARCHAR(255) NOT NULL,
    agent_type VARCHAR(100) DEFAULT 'claude-code',
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    start_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    end_time TIMESTAMP WITH TIME ZONE,
    logs TEXT,
    result JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- MCP Registry Events - audit log of registry changes
CREATE TABLE IF NOT EXISTS mcp.registry_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    endpoint_id UUID REFERENCES mcp.endpoints(id) ON DELETE SET NULL,
    scenario_name VARCHAR(255),
    event_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- MCP Templates - reusable MCP server templates
CREATE TABLE IF NOT EXISTS mcp.templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    template_type VARCHAR(50) DEFAULT 'generic' CHECK (template_type IN ('generic', 'api', 'data', 'workflow', 'ui')),
    template_code TEXT NOT NULL,
    manifest_template JSONB,
    tags TEXT[] DEFAULT '{}',
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_endpoints_scenario_name ON mcp.endpoints(scenario_name);
CREATE INDEX idx_endpoints_status ON mcp.endpoints(status);
CREATE INDEX idx_endpoints_mcp_port ON mcp.endpoints(mcp_port);
CREATE INDEX idx_tool_usage_endpoint_id ON mcp.tool_usage(endpoint_id);
CREATE INDEX idx_tool_usage_tool_name ON mcp.tool_usage(tool_name);
CREATE INDEX idx_tools_endpoint_id ON mcp.tools(endpoint_id);
CREATE INDEX idx_agent_sessions_scenario ON mcp.agent_sessions(scenario_name);
CREATE INDEX idx_agent_sessions_status ON mcp.agent_sessions(status);
CREATE INDEX idx_registry_events_created_at ON mcp.registry_events(created_at DESC);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION mcp.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update trigger to tables with updated_at
CREATE TRIGGER update_endpoints_updated_at BEFORE UPDATE ON mcp.endpoints
    FOR EACH ROW EXECUTE FUNCTION mcp.update_updated_at_column();

CREATE TRIGGER update_tool_usage_updated_at BEFORE UPDATE ON mcp.tool_usage
    FOR EACH ROW EXECUTE FUNCTION mcp.update_updated_at_column();

CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON mcp.templates
    FOR EACH ROW EXECUTE FUNCTION mcp.update_updated_at_column();

-- View for active MCP endpoints with tool counts
CREATE OR REPLACE VIEW mcp.active_endpoints AS
SELECT 
    e.id,
    e.scenario_name,
    e.mcp_port,
    e.status,
    e.last_health_check,
    COUNT(DISTINCT t.id) as tool_count,
    COUNT(DISTINCT u.id) as usage_records,
    SUM(u.invocation_count) as total_invocations,
    e.created_at,
    e.updated_at
FROM mcp.endpoints e
LEFT JOIN mcp.tools t ON e.id = t.endpoint_id
LEFT JOIN mcp.tool_usage u ON e.id = u.endpoint_id
WHERE e.status = 'active'
GROUP BY e.id;

-- View for MCP registry discovery
CREATE OR REPLACE VIEW mcp.registry AS
SELECT 
    e.scenario_name as name,
    'stdio' as transport,
    'http://localhost:' || e.mcp_port as url,
    'http://localhost:' || e.mcp_port || '/manifest' as manifest_url,
    e.capabilities,
    e.mcp_version as version,
    e.metadata
FROM mcp.endpoints e
WHERE e.status = 'active' AND e.mcp_port IS NOT NULL
ORDER BY e.scenario_name;

-- Function to allocate MCP port for a scenario
CREATE OR REPLACE FUNCTION mcp.allocate_mcp_port(scenario_name_param VARCHAR)
RETURNS INTEGER AS $$
DECLARE
    base_port INTEGER := 4000;
    max_port INTEGER := 5000;
    allocated_port INTEGER;
BEGIN
    -- Find the next available port in the range
    SELECT MIN(port_candidate) INTO allocated_port
    FROM generate_series(base_port, max_port) AS port_candidate
    WHERE NOT EXISTS (
        SELECT 1 FROM mcp.endpoints 
        WHERE mcp_port = port_candidate 
        AND scenario_name != scenario_name_param
    );
    
    RETURN allocated_port;
END;
$$ LANGUAGE plpgsql;

-- Function to record tool usage
CREATE OR REPLACE FUNCTION mcp.record_tool_usage(
    p_scenario_name VARCHAR,
    p_tool_name VARCHAR,
    p_success BOOLEAN,
    p_response_time_ms INTEGER
)
RETURNS VOID AS $$
DECLARE
    v_endpoint_id UUID;
    v_existing_count INTEGER;
    v_existing_success INTEGER;
    v_existing_error INTEGER;
    v_existing_total_time BIGINT;
BEGIN
    -- Get endpoint ID
    SELECT id INTO v_endpoint_id 
    FROM mcp.endpoints 
    WHERE scenario_name = p_scenario_name;
    
    IF v_endpoint_id IS NULL THEN
        RETURN;
    END IF;
    
    -- Get existing stats
    SELECT invocation_count, success_count, error_count, total_response_time_ms
    INTO v_existing_count, v_existing_success, v_existing_error, v_existing_total_time
    FROM mcp.tool_usage
    WHERE endpoint_id = v_endpoint_id AND tool_name = p_tool_name;
    
    IF NOT FOUND THEN
        -- Insert new record
        INSERT INTO mcp.tool_usage (
            endpoint_id, tool_name, invocation_count, 
            success_count, error_count, total_response_time_ms,
            avg_response_time_ms, last_used
        ) VALUES (
            v_endpoint_id, p_tool_name, 1,
            CASE WHEN p_success THEN 1 ELSE 0 END,
            CASE WHEN NOT p_success THEN 1 ELSE 0 END,
            p_response_time_ms,
            p_response_time_ms,
            NOW()
        );
    ELSE
        -- Update existing record
        UPDATE mcp.tool_usage
        SET invocation_count = invocation_count + 1,
            success_count = success_count + CASE WHEN p_success THEN 1 ELSE 0 END,
            error_count = error_count + CASE WHEN NOT p_success THEN 1 ELSE 0 END,
            total_response_time_ms = total_response_time_ms + p_response_time_ms,
            avg_response_time_ms = (total_response_time_ms + p_response_time_ms) / (invocation_count + 1),
            last_used = NOW()
        WHERE endpoint_id = v_endpoint_id AND tool_name = p_tool_name;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Insert default MCP templates
INSERT INTO mcp.templates (name, description, template_type, template_code, manifest_template) VALUES
('basic-api', 'Basic MCP server for API exposure', 'api', 
'const { Server } = require("@modelcontextprotocol/sdk/server/stdio");
const server = new Server({
  name: "scenario_name",
  version: "1.0.0"
});

// Tool implementations
server.setRequestHandler("tools/list", async () => ({
  tools: []
}));

// Tool handlers go here

server.start();',
'{"name": "scenario_name", "version": "1.0.0", "tools": []}'::jsonb),

('data-processor', 'MCP server for data processing scenarios', 'data',
'const { Server } = require("@modelcontextprotocol/sdk/server/stdio");
const server = new Server({
  name: "scenario_name",
  version: "1.0.0"
});

// Data processing tools
// Tools go here

server.start();',
'{"name": "scenario_name", "version": "1.0.0", "tools": []}'::jsonb);

-- Grant permissions
GRANT ALL ON SCHEMA mcp TO PUBLIC;
GRANT ALL ON ALL TABLES IN SCHEMA mcp TO PUBLIC;
GRANT ALL ON ALL SEQUENCES IN SCHEMA mcp TO PUBLIC;
GRANT ALL ON ALL FUNCTIONS IN SCHEMA mcp TO PUBLIC;