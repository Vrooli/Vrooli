-- Network Tools Seed Data
-- Initial test data for development and testing

-- Insert sample network targets
INSERT INTO network_targets (name, target_type, address, port, protocol, tags, is_active) VALUES
('Google DNS', 'host', '8.8.8.8', NULL, 'icmp', ARRAY['dns', 'google', 'critical'], true),
('Cloudflare DNS', 'host', '1.1.1.1', NULL, 'icmp', ARRAY['dns', 'cloudflare', 'critical'], true),
('HTTPBin API', 'api', 'httpbin.org', 443, 'https', ARRAY['test', 'api'], true),
('Local PostgreSQL', 'service', 'localhost', 5432, 'tcp', ARRAY['database', 'local'], true),
('GitHub API', 'api', 'api.github.com', 443, 'https', ARRAY['api', 'github'], true),
('Example Website', 'url', 'example.com', 443, 'https', ARRAY['test', 'website'], true)
ON CONFLICT DO NOTHING;

-- Insert sample API definitions
INSERT INTO api_definitions (name, base_url, version, specification, authentication_methods, endpoints_count) VALUES
('HTTPBin Test API', 'https://httpbin.org', '1.0', 'rest', ARRAY['none'], 20),
('GitHub REST API', 'https://api.github.com', 'v3', 'rest', ARRAY['bearer', 'oauth2'], 100),
('Local Test API', 'http://localhost:8090', '1.0', 'openapi', ARRAY['api_key'], 10)
ON CONFLICT DO NOTHING;

-- Insert sample DNS queries for common domains (for cache testing)
INSERT INTO dns_queries (query, record_type, dns_server, response_time_ms, authoritative, dnssec_valid, ttl, cached_until, answers) VALUES
('google.com', 'A', '8.8.8.8', 12, true, true, 300, NOW() + INTERVAL '5 minutes', 
 '[{"name": "google.com", "type": "A", "ttl": 300, "data": "142.250.185.78"}]'::jsonb),
('cloudflare.com', 'A', '1.1.1.1', 8, true, true, 300, NOW() + INTERVAL '5 minutes',
 '[{"name": "cloudflare.com", "type": "A", "ttl": 300, "data": "104.16.132.229"}]'::jsonb),
('github.com', 'A', '8.8.8.8', 15, true, true, 60, NOW() + INTERVAL '1 minute',
 '[{"name": "github.com", "type": "A", "ttl": 60, "data": "140.82.112.4"}]'::jsonb)
ON CONFLICT DO NOTHING;

-- Insert sample monitoring data for visualization
DO $$
DECLARE
    target_record RECORD;
    i INTEGER;
BEGIN
    -- Get the first network target
    SELECT id INTO target_record FROM network_targets WHERE name = 'Google DNS' LIMIT 1;
    
    IF target_record.id IS NOT NULL THEN
        -- Generate 24 hours of sample latency data (one point per hour)
        FOR i IN 0..23 LOOP
            INSERT INTO monitoring_data (
                target_id, 
                metric_type, 
                timestamp, 
                value, 
                unit, 
                alert_level
            ) VALUES (
                target_record.id,
                'latency',
                NOW() - INTERVAL '1 hour' * i,
                10 + random() * 20, -- Random latency between 10-30ms
                'ms',
                CASE 
                    WHEN random() < 0.9 THEN 'ok'
                    WHEN random() < 0.95 THEN 'warning'
                    ELSE 'critical'
                END
            );
        END LOOP;
    END IF;
END $$;

-- Insert sample scan results
INSERT INTO scan_results (
    target_id,
    scan_type,
    status,
    completed_at,
    response_time_ms,
    results,
    findings
) 
SELECT 
    id,
    'connectivity',
    'success',
    NOW(),
    FLOOR(random() * 50 + 10),
    jsonb_build_object(
        'reachable', true,
        'avg_rtt', FLOOR(random() * 30 + 5),
        'packet_loss', 0
    ),
    '[]'::jsonb
FROM network_targets
WHERE is_active = true
LIMIT 3;

-- Insert sample HTTP request logs
INSERT INTO http_requests (
    url,
    method,
    status_code,
    response_time_ms,
    headers,
    response_headers
) VALUES
('https://httpbin.org/get', 'GET', 200, 150, 
 '{"User-Agent": "Network-Tools/1.0"}'::jsonb,
 '{"Content-Type": "application/json"}'::jsonb),
('https://api.github.com/users/octocat', 'GET', 200, 230,
 '{"Accept": "application/vnd.github.v3+json"}'::jsonb,
 '{"X-RateLimit-Remaining": "59"}'::jsonb),
('https://example.com', 'GET', 200, 89,
 '{"Accept": "text/html"}'::jsonb,
 '{"Content-Type": "text/html; charset=UTF-8"}'::jsonb);

-- Insert sample port scan results
INSERT INTO port_scans (
    target_id,
    port,
    protocol,
    state,
    service,
    response_time_ms
)
SELECT 
    nt.id,
    unnest(ARRAY[22, 80, 443, 3306, 5432, 8080]),
    'tcp',
    CASE 
        WHEN random() < 0.3 THEN 'open'
        WHEN random() < 0.6 THEN 'closed'
        ELSE 'filtered'
    END,
    CASE unnest(ARRAY[22, 80, 443, 3306, 5432, 8080])
        WHEN 22 THEN 'ssh'
        WHEN 80 THEN 'http'
        WHEN 443 THEN 'https'
        WHEN 3306 THEN 'mysql'
        WHEN 5432 THEN 'postgresql'
        WHEN 8080 THEN 'http-proxy'
    END,
    FLOOR(random() * 100 + 1)
FROM network_targets nt
WHERE nt.name = 'Local PostgreSQL'
LIMIT 1;

-- Insert sample alerts
INSERT INTO alerts (
    target_id,
    alert_type,
    severity,
    title,
    message,
    threshold_value,
    actual_value
)
SELECT
    id,
    'high_latency',
    'warning',
    'High latency detected for ' || name,
    'Network latency exceeded threshold of 100ms',
    100.0,
    125.5
FROM network_targets
WHERE name = 'Google DNS'
LIMIT 1;

-- Add a sample SSL certificate record
INSERT INTO ssl_certificates (
    target_id,
    common_name,
    issuer,
    subject,
    not_before,
    not_after,
    is_valid,
    is_expired,
    days_until_expiry,
    signature_algorithm,
    key_size
)
SELECT
    id,
    'github.com',
    'CN=DigiCert TLS RSA SHA256 2020 CA1, O=DigiCert Inc',
    'CN=github.com',
    NOW() - INTERVAL '30 days',
    NOW() + INTERVAL '335 days',
    true,
    false,
    335,
    'SHA256-RSA',
    2048
FROM network_targets
WHERE name = 'GitHub API'
LIMIT 1;