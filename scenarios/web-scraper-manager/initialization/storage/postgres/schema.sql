-- Web Scraper Manager Database Schema
-- Unified management for Huginn, Browserless, and Agent-S2 scrapers

-- Agent configurations across platforms
CREATE TABLE scraping_agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('huginn', 'browserless', 'agent-s2', 'hybrid')),
    agent_type VARCHAR(100),
    configuration JSONB NOT NULL,
    schedule_cron VARCHAR(100),
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    tags TEXT[]
);

-- Scraping targets
CREATE TABLE scraping_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES scraping_agents(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    selector_config JSONB,
    authentication JSONB,
    headers JSONB,
    proxy_config JSONB,
    rate_limit_ms INTEGER DEFAULT 1000,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Scraping results
CREATE TABLE scraping_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES scraping_agents(id),
    target_id UUID REFERENCES scraping_targets(id),
    run_id UUID,
    status VARCHAR(20) CHECK (status IN ('pending', 'running', 'success', 'failed', 'partial')),
    data JSONB,
    screenshots TEXT[],
    extracted_count INTEGER,
    error_message TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    execution_time_ms INTEGER
);

-- Data transformations
CREATE TABLE data_transformations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES scraping_agents(id),
    name VARCHAR(255) NOT NULL,
    transform_type VARCHAR(50) CHECK (transform_type IN ('filter', 'map', 'aggregate', 'enrich', 'validate')),
    configuration JSONB NOT NULL,
    order_index INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Export configurations
CREATE TABLE export_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    agent_ids UUID[],
    format VARCHAR(50) CHECK (format IN ('json', 'csv', 'xml', 'parquet', 'api', 'webhook')),
    destination_config JSONB,
    transform_ids UUID[],
    schedule_cron VARCHAR(100),
    enabled BOOLEAN DEFAULT true,
    last_export TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Monitoring and alerts
CREATE TABLE monitoring_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    agent_id UUID REFERENCES scraping_agents(id),
    rule_type VARCHAR(50) CHECK (rule_type IN ('success_rate', 'data_quality', 'performance', 'availability')),
    condition JSONB NOT NULL,
    alert_config JSONB,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent metrics
CREATE TABLE agent_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES scraping_agents(id),
    metric_date DATE NOT NULL,
    runs_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    data_points_collected INTEGER DEFAULT 0,
    avg_execution_time_ms INTEGER,
    total_bytes_scraped BIGINT DEFAULT 0,
    cost_estimate DECIMAL(10,2),
    UNIQUE(agent_id, metric_date)
);

-- Proxy management
CREATE TABLE proxy_pool (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proxy_url TEXT NOT NULL,
    proxy_type VARCHAR(20) CHECK (proxy_type IN ('http', 'https', 'socks5')),
    location VARCHAR(100),
    credentials JSONB,
    health_status VARCHAR(20) DEFAULT 'unknown' CHECK (health_status IN ('healthy', 'degraded', 'failed', 'unknown')),
    last_health_check TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API endpoints for scraped data
CREATE TABLE api_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    agent_ids UUID[],
    query_config JSONB,
    cache_ttl_seconds INTEGER DEFAULT 3600,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    auth_required BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Workflow orchestration
CREATE TABLE scraping_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    steps JSONB NOT NULL,
    trigger_config JSONB,
    error_handling JSONB,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_agents_platform ON scraping_agents(platform);
CREATE INDEX idx_agents_enabled ON scraping_agents(enabled);
CREATE INDEX idx_agents_tags ON scraping_agents USING GIN(tags);
CREATE INDEX idx_results_agent ON scraping_results(agent_id);
CREATE INDEX idx_results_status ON scraping_results(status);
CREATE INDEX idx_results_run ON scraping_results(run_id);
CREATE INDEX idx_metrics_agent_date ON agent_metrics(agent_id, metric_date);
CREATE INDEX idx_proxy_health ON proxy_pool(health_status, enabled);