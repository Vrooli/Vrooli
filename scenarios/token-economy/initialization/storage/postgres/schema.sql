-- Token Economy Schema
-- Immutable ledger for token transactions with full audit trail

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enum types
CREATE TYPE token_type AS ENUM ('fungible', 'non-fungible');
CREATE TYPE wallet_type AS ENUM ('user', 'scenario', 'treasury');
CREATE TYPE transaction_type AS ENUM ('mint', 'transfer', 'burn', 'swap');
CREATE TYPE transaction_status AS ENUM ('pending', 'confirmed', 'failed');
CREATE TYPE achievement_rarity AS ENUM ('common', 'rare', 'epic', 'legendary');

-- Households table (multi-tenancy support)
CREATE TABLE IF NOT EXISTS households (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tokens table
CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    symbol VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    type token_type NOT NULL,
    total_supply DECIMAL(36,18) DEFAULT 0,
    max_supply DECIMAL(36,18),
    decimals INT DEFAULT 18,
    creator_scenario VARCHAR(100) NOT NULL,
    metadata JSONB DEFAULT '{}',
    icon_url VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(household_id, symbol)
);

-- Wallets table
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    user_id UUID,
    scenario_name VARCHAR(100),
    address VARCHAR(42) UNIQUE NOT NULL,
    type wallet_type NOT NULL,
    encrypted_key TEXT,
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT wallet_owner_check CHECK (
        (type = 'user' AND user_id IS NOT NULL) OR
        (type = 'scenario' AND scenario_name IS NOT NULL) OR
        (type = 'treasury')
    )
);

-- Balances table (current state)
CREATE TABLE IF NOT EXISTS balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
    amount DECIMAL(36,18) NOT NULL DEFAULT 0,
    locked_amount DECIMAL(36,18) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(wallet_id, token_id),
    CONSTRAINT balance_non_negative CHECK (amount >= 0 AND locked_amount >= 0)
);

-- Transactions table (immutable ledger)
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hash VARCHAR(66) UNIQUE NOT NULL,
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    from_wallet UUID REFERENCES wallets(id),
    to_wallet UUID REFERENCES wallets(id),
    token_id UUID NOT NULL REFERENCES tokens(id),
    amount DECIMAL(36,18) NOT NULL,
    type transaction_type NOT NULL,
    status transaction_status NOT NULL DEFAULT 'pending',
    metadata JSONB DEFAULT '{}',
    error_message TEXT,
    gas_used INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TIMESTAMP WITH TIME ZONE,
    block_number BIGINT,
    CONSTRAINT transaction_wallets_check CHECK (
        (type = 'mint' AND from_wallet IS NULL AND to_wallet IS NOT NULL) OR
        (type = 'burn' AND from_wallet IS NOT NULL AND to_wallet IS NULL) OR
        (type IN ('transfer', 'swap') AND from_wallet IS NOT NULL AND to_wallet IS NOT NULL)
    )
);

-- Achievements table (NFT metadata)
CREATE TABLE IF NOT EXISTS achievements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    scenario VARCHAR(100) NOT NULL,
    achievement_type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    criteria JSONB DEFAULT '{}',
    icon_url VARCHAR(500),
    rarity achievement_rarity NOT NULL DEFAULT 'common',
    points INT DEFAULT 0,
    earned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    UNIQUE(token_id)
);

-- Exchange rates table
CREATE TABLE IF NOT EXISTS exchange_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    from_token UUID NOT NULL REFERENCES tokens(id),
    to_token UUID NOT NULL REFERENCES tokens(id),
    rate DECIMAL(36,18) NOT NULL,
    liquidity DECIMAL(36,18) DEFAULT 0,
    is_automatic BOOLEAN DEFAULT false,
    created_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(household_id, from_token, to_token),
    CONSTRAINT different_tokens CHECK (from_token != to_token),
    CONSTRAINT positive_rate CHECK (rate > 0)
);

-- Scenario permissions table
CREATE TABLE IF NOT EXISTS scenario_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    scenario_name VARCHAR(100) NOT NULL,
    can_mint BOOLEAN DEFAULT false,
    can_burn BOOLEAN DEFAULT false,
    mint_limit DECIMAL(36,18),
    daily_mint_limit DECIMAL(36,18),
    allowed_tokens UUID[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(household_id, scenario_name)
);

-- Household rules table (parent controls)
CREATE TABLE IF NOT EXISTS household_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    user_id UUID,
    daily_spend_limit DECIMAL(36,18),
    weekly_spend_limit DECIMAL(36,18),
    monthly_spend_limit DECIMAL(36,18),
    allowed_scenarios TEXT[],
    blocked_scenarios TEXT[],
    can_swap BOOLEAN DEFAULT true,
    can_transfer BOOLEAN DEFAULT true,
    requires_approval_above DECIMAL(36,18),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(household_id, user_id)
);

-- Pending approvals table
CREATE TABLE IF NOT EXISTS pending_approvals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    transaction_id UUID REFERENCES transactions(id),
    requester_id UUID NOT NULL,
    approver_id UUID,
    action VARCHAR(50) NOT NULL,
    details JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    household_id UUID REFERENCES households(id) ON DELETE CASCADE,
    user_id UUID,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    old_value JSONB,
    new_value JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Analytics aggregates table (for performance)
CREATE TABLE IF NOT EXISTS token_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    transaction_count INT DEFAULT 0,
    volume DECIMAL(36,18) DEFAULT 0,
    unique_wallets INT DEFAULT 0,
    average_transaction DECIMAL(36,18),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(token_id, period_start, period_end)
);

-- Indexes for performance
CREATE INDEX idx_tokens_household ON tokens(household_id);
CREATE INDEX idx_tokens_symbol ON tokens(symbol);
CREATE INDEX idx_wallets_household ON wallets(household_id);
CREATE INDEX idx_wallets_user ON wallets(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_wallets_scenario ON wallets(scenario_name) WHERE scenario_name IS NOT NULL;
CREATE INDEX idx_wallets_address ON wallets(address);
CREATE INDEX idx_balances_wallet ON balances(wallet_id);
CREATE INDEX idx_balances_token ON balances(token_id);
CREATE INDEX idx_transactions_household ON transactions(household_id);
CREATE INDEX idx_transactions_from ON transactions(from_wallet) WHERE from_wallet IS NOT NULL;
CREATE INDEX idx_transactions_to ON transactions(to_wallet) WHERE to_wallet IS NOT NULL;
CREATE INDEX idx_transactions_created ON transactions(created_at);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_achievements_wallet ON achievements(wallet_id);
CREATE INDEX idx_achievements_scenario ON achievements(scenario);
CREATE INDEX idx_exchange_rates_tokens ON exchange_rates(from_token, to_token);
CREATE INDEX idx_audit_logs_household ON audit_logs(household_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);

-- Functions for atomic operations
CREATE OR REPLACE FUNCTION transfer_tokens(
    p_from_wallet UUID,
    p_to_wallet UUID,
    p_token_id UUID,
    p_amount DECIMAL(36,18),
    p_metadata JSONB DEFAULT '{}'
) RETURNS UUID AS $$
DECLARE
    v_transaction_id UUID;
    v_from_balance DECIMAL(36,18);
    v_household_id UUID;
BEGIN
    -- Start transaction
    v_transaction_id := uuid_generate_v4();
    
    -- Get household ID
    SELECT household_id INTO v_household_id
    FROM wallets WHERE id = p_from_wallet;
    
    -- Lock and check balance
    SELECT amount INTO v_from_balance
    FROM balances
    WHERE wallet_id = p_from_wallet AND token_id = p_token_id
    FOR UPDATE;
    
    IF v_from_balance IS NULL OR v_from_balance < p_amount THEN
        RAISE EXCEPTION 'Insufficient balance';
    END IF;
    
    -- Update balances
    UPDATE balances
    SET amount = amount - p_amount,
        updated_at = CURRENT_TIMESTAMP
    WHERE wallet_id = p_from_wallet AND token_id = p_token_id;
    
    INSERT INTO balances (wallet_id, token_id, amount)
    VALUES (p_to_wallet, p_token_id, p_amount)
    ON CONFLICT (wallet_id, token_id)
    DO UPDATE SET
        amount = balances.amount + p_amount,
        updated_at = CURRENT_TIMESTAMP;
    
    -- Record transaction
    INSERT INTO transactions (
        id, hash, household_id, from_wallet, to_wallet,
        token_id, amount, type, status, metadata,
        created_at, confirmed_at
    ) VALUES (
        v_transaction_id,
        encode(sha256((v_transaction_id::text || p_from_wallet::text || p_to_wallet::text || p_amount::text)::bytea), 'hex'),
        v_household_id,
        p_from_wallet,
        p_to_wallet,
        p_token_id,
        p_amount,
        'transfer',
        'confirmed',
        p_metadata,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    );
    
    RETURN v_transaction_id;
END;
$$ LANGUAGE plpgsql;

-- Function to mint tokens
CREATE OR REPLACE FUNCTION mint_tokens(
    p_token_id UUID,
    p_to_wallet UUID,
    p_amount DECIMAL(36,18),
    p_scenario VARCHAR(100),
    p_metadata JSONB DEFAULT '{}'
) RETURNS UUID AS $$
DECLARE
    v_transaction_id UUID;
    v_household_id UUID;
    v_max_supply DECIMAL(36,18);
    v_current_supply DECIMAL(36,18);
BEGIN
    v_transaction_id := uuid_generate_v4();
    
    -- Get token info and check supply limits
    SELECT household_id, max_supply, total_supply
    INTO v_household_id, v_max_supply, v_current_supply
    FROM tokens
    WHERE id = p_token_id
    FOR UPDATE;
    
    IF v_max_supply IS NOT NULL AND (v_current_supply + p_amount) > v_max_supply THEN
        RAISE EXCEPTION 'Would exceed maximum supply';
    END IF;
    
    -- Update total supply
    UPDATE tokens
    SET total_supply = total_supply + p_amount,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = p_token_id;
    
    -- Add to wallet balance
    INSERT INTO balances (wallet_id, token_id, amount)
    VALUES (p_to_wallet, p_token_id, p_amount)
    ON CONFLICT (wallet_id, token_id)
    DO UPDATE SET
        amount = balances.amount + p_amount,
        updated_at = CURRENT_TIMESTAMP;
    
    -- Record transaction
    INSERT INTO transactions (
        id, hash, household_id, from_wallet, to_wallet,
        token_id, amount, type, status, metadata,
        created_at, confirmed_at
    ) VALUES (
        v_transaction_id,
        encode(sha256((v_transaction_id::text || p_token_id::text || p_to_wallet::text || p_amount::text)::bytea), 'hex'),
        v_household_id,
        NULL,
        p_to_wallet,
        p_token_id,
        p_amount,
        'mint',
        'confirmed',
        p_metadata,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    );
    
    RETURN v_transaction_id;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tokens_updated_at BEFORE UPDATE ON tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_wallets_updated_at BEFORE UPDATE ON wallets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_exchange_rates_updated_at BEFORE UPDATE ON exchange_rates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_scenario_permissions_updated_at BEFORE UPDATE ON scenario_permissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_household_rules_updated_at BEFORE UPDATE ON household_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Initial data for testing
INSERT INTO households (id, name) VALUES 
    ('00000000-0000-0000-0000-000000000001', 'Test Household')
ON CONFLICT DO NOTHING;