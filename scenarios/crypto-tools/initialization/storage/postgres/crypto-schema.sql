-- Crypto-Tools Database Schema
-- Enterprise-grade cryptographic operations platform

-- Cryptographic operations tracking
CREATE TABLE IF NOT EXISTS crypto_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation_type VARCHAR(50) NOT NULL CHECK (operation_type IN ('hash', 'encrypt', 'decrypt', 'sign', 'verify', 'keygen')),
    algorithm VARCHAR(100) NOT NULL,
    
    -- Operation details
    input_size BIGINT,
    output_size BIGINT,
    execution_time_ms INTEGER,
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    
    -- Security context
    key_id UUID,
    certificate_id UUID,
    
    -- Metadata
    user_id VARCHAR(255),
    session_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Cryptographic keys management
CREATE TABLE IF NOT EXISTS crypto_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    key_type VARCHAR(50) NOT NULL CHECK (key_type IN ('symmetric', 'rsa', 'ecdsa', 'ed25519')),
    key_size INTEGER,
    usage VARCHAR(50)[] DEFAULT ARRAY[]::VARCHAR[],
    
    -- Key material (encrypted)
    public_key TEXT,
    private_key_encrypted TEXT,
    key_fingerprint VARCHAR(255) UNIQUE,
    
    -- Security properties
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired', 'compromised')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    
    -- Rotation policy
    rotation_policy JSONB DEFAULT '{}',
    rotated_from UUID REFERENCES crypto_keys(id),
    
    -- HSM integration
    hsm_key_id VARCHAR(255),
    hsm_provider VARCHAR(100),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    audit_trail JSONB DEFAULT '[]'
);

-- X.509 Certificates
CREATE TABLE IF NOT EXISTS certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject VARCHAR(500) NOT NULL,
    issuer VARCHAR(500) NOT NULL,
    serial_number VARCHAR(255) UNIQUE NOT NULL,
    
    -- Certificate data
    certificate_pem TEXT NOT NULL,
    certificate_chain TEXT[],
    private_key_id UUID REFERENCES crypto_keys(id),
    
    -- Validity
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    revocation_status VARCHAR(50) DEFAULT 'valid' CHECK (revocation_status IN ('valid', 'revoked', 'expired')),
    revocation_reason TEXT,
    revoked_at TIMESTAMP WITH TIME ZONE,
    
    -- Certificate properties
    key_usage TEXT[],
    extended_key_usage TEXT[],
    subject_alt_names TEXT[],
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_validated_at TIMESTAMP WITH TIME ZONE
);

-- Security events and audit log
CREATE TABLE IF NOT EXISTS security_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    
    -- Entity references
    entity_id UUID,
    entity_type VARCHAR(50) CHECK (entity_type IN ('key', 'certificate', 'user', 'system')),
    
    -- Event details
    event_data JSONB NOT NULL,
    source_ip INET,
    user_agent TEXT,
    user_id VARCHAR(255),
    
    -- Response
    remediation_required BOOLEAN DEFAULT false,
    remediation_actions JSONB,
    
    -- Timestamps
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    correlation_id UUID
);

-- Compliance policies
CREATE TABLE IF NOT EXISTS compliance_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    policy_type VARCHAR(50) NOT NULL CHECK (policy_type IN ('fips_140_2', 'common_criteria', 'gdpr', 'hipaa', 'sox')),
    
    -- Policy configuration
    requirements JSONB NOT NULL,
    validation_rules JSONB NOT NULL,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    effective_from TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    effective_until TIMESTAMP WITH TIME ZONE,
    
    -- Audit
    created_by VARCHAR(255),
    approved_by VARCHAR(255),
    last_validation TIMESTAMP WITH TIME ZONE,
    compliance_status VARCHAR(50) DEFAULT 'pending' CHECK (compliance_status IN ('compliant', 'non_compliant', 'pending'))
);

-- Create indexes for performance
CREATE INDEX idx_crypto_ops_type ON crypto_operations(operation_type);
CREATE INDEX idx_crypto_ops_created ON crypto_operations(created_at DESC);
CREATE INDEX idx_crypto_ops_user ON crypto_operations(user_id);

CREATE INDEX idx_keys_status ON crypto_keys(status) WHERE status = 'active';
CREATE INDEX idx_keys_type ON crypto_keys(key_type);
CREATE INDEX idx_keys_fingerprint ON crypto_keys(key_fingerprint);
CREATE INDEX idx_keys_expires ON crypto_keys(expires_at) WHERE expires_at IS NOT NULL;

CREATE INDEX idx_certs_serial ON certificates(serial_number);
CREATE INDEX idx_certs_subject ON certificates(subject);
CREATE INDEX idx_certs_validity ON certificates(valid_until) WHERE revocation_status = 'valid';
CREATE INDEX idx_certs_key ON certificates(private_key_id);

CREATE INDEX idx_events_type ON security_events(event_type);
CREATE INDEX idx_events_severity ON security_events(severity);
CREATE INDEX idx_events_entity ON security_events(entity_type, entity_id);
CREATE INDEX idx_events_timestamp ON security_events(timestamp DESC);

CREATE INDEX idx_compliance_active ON compliance_policies(policy_type) WHERE is_active = true;

-- Statistics view for dashboard
CREATE OR REPLACE VIEW crypto_statistics AS
SELECT 
    COUNT(*) FILTER (WHERE operation_type = 'hash') as hash_operations,
    COUNT(*) FILTER (WHERE operation_type = 'encrypt') as encryption_operations,
    COUNT(*) FILTER (WHERE operation_type = 'decrypt') as decryption_operations,
    COUNT(*) FILTER (WHERE operation_type = 'sign') as signature_operations,
    COUNT(*) FILTER (WHERE operation_type = 'verify') as verification_operations,
    COUNT(*) FILTER (WHERE operation_type = 'keygen') as key_generation_operations,
    AVG(execution_time_ms) as avg_execution_time,
    COUNT(*) FILTER (WHERE success = false) as failed_operations,
    COUNT(*) as total_operations
FROM crypto_operations
WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '24 hours';

-- Key rotation tracking
CREATE OR REPLACE VIEW key_rotation_status AS
SELECT 
    k.id,
    k.name,
    k.key_type,
    k.expires_at,
    k.status,
    CASE 
        WHEN k.expires_at < CURRENT_TIMESTAMP THEN 'expired'
        WHEN k.expires_at < CURRENT_TIMESTAMP + INTERVAL '30 days' THEN 'expiring_soon'
        ELSE 'valid'
    END as rotation_status
FROM crypto_keys k
WHERE k.status = 'active'
ORDER BY k.expires_at ASC NULLS LAST;