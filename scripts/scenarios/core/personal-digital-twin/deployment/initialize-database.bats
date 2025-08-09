#!/usr/bin/env bats
# Tests for initialize-database.sh

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_service_test "postgres"
    
    # Set up test scenario directory structure
    export TEST_SCENARIO_DIR="${BATS_TEST_TMPDIR}/personal-digital-twin"
    export TEST_POSTGRES_DIR="${TEST_SCENARIO_DIR}/initialization/storage/postgres"
    mkdir -p "${TEST_POSTGRES_DIR}"
    
    # Create sample schema and seed files
    cat > "${TEST_POSTGRES_DIR}/schema.sql" <<'EOF'
CREATE TABLE IF NOT EXISTS personas (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS memories (
    id SERIAL PRIMARY KEY,
    persona_id INTEGER REFERENCES personas(id),
    content TEXT NOT NULL,
    importance_score FLOAT DEFAULT 0.0,
    created_at TIMESTAMP DEFAULT NOW()
);
EOF
    
    cat > "${TEST_POSTGRES_DIR}/seed.sql" <<'EOF'
INSERT INTO personas (name) VALUES 
    ('Default Persona'),
    ('Assistant Persona');

INSERT INTO memories (persona_id, content, importance_score) VALUES
    (1, 'Initial memory', 0.8),
    (2, 'Assistant initialization', 0.9);
EOF
    
    # Mock environment variables
    export POSTGRES_HOST="localhost"
    export POSTGRES_PORT="5433"
    export POSTGRES_USER="postgres"
    export POSTGRES_PASSWORD="postgres"
    export POSTGRES_DB="digital_twin"
}

teardown() {
    vrooli_cleanup_test
}

@test "script should wait for PostgreSQL to be ready" {
    # Mock that PostgreSQL is initially not ready, then becomes ready
    mock::postgres::set_ready "false"
    mock::postgres::set_ready_after_attempts "3"
    
    run timeout 10s bash "${BATS_TEST_DIRNAME}/initialize-database.sh"
    
    assert_success
    assert_output_contains "PostgreSQL is ready"
}

@test "script should execute schema creation successfully" {
    # Mock that PostgreSQL is ready
    mock::postgres::set_ready "true"
    # Mock successful schema execution
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "true"
    
    run bash "${BATS_TEST_DIRNAME}/initialize-database.sh"
    
    assert_success
    assert_output_contains "Database schema initialized successfully"
}

@test "script should load seed data when seed file exists" {
    # Mock successful scenario with seed data
    mock::postgres::set_ready "true"
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "true"
    mock::postgres::set_seed_executable "${TEST_POSTGRES_DIR}/seed.sql" "true"
    
    run bash "${BATS_TEST_DIRNAME}/initialize-database.sh"
    
    assert_success
    assert_output_contains "Database schema initialized successfully"
    assert_output_contains "Loading seed data"
    assert_output_contains "Seed data loaded successfully"
}

@test "script should handle missing schema file" {
    # Remove schema file
    rm -f "${TEST_POSTGRES_DIR}/schema.sql"
    
    mock::postgres::set_ready "true"
    
    run bash "${BATS_TEST_DIRNAME}/initialize-database.sh"
    
    assert_failure
    # The psql command will fail when trying to execute a non-existent file
}

@test "script should handle schema execution failure" {
    mock::postgres::set_ready "true"
    # Mock schema execution failure
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "false"
    
    run bash "${BATS_TEST_DIRNAME}/initialize-database.sh"
    
    assert_failure
    assert_output_contains "Failed to initialize database schema"
}

@test "script should handle seed data loading failure" {
    mock::postgres::set_ready "true"
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "true"
    # Mock seed data loading failure
    mock::postgres::set_seed_executable "${TEST_POSTGRES_DIR}/seed.sql" "false"
    
    run bash "${BATS_TEST_DIRNAME}/initialize-database.sh"
    
    assert_failure
    assert_output_contains "Failed to load seed data"
}

@test "script should continue without seed data if seed file doesn't exist" {
    # Remove seed file
    rm -f "${TEST_POSTGRES_DIR}/seed.sql"
    
    mock::postgres::set_ready "true"
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "true"
    
    run bash "${BATS_TEST_DIRNAME}/initialize-database.sh"
    
    assert_success
    assert_output_contains "Database initialization complete"
    refute_output_contains "Loading seed data"
}

@test "script should use environment variables for connection" {
    # Set custom environment variables
    export POSTGRES_HOST="custom-host"
    export POSTGRES_PORT="9999"
    export POSTGRES_USER="custom-user"
    export POSTGRES_PASSWORD="custom-pass"
    export POSTGRES_DB="custom-db"
    
    mock::postgres::set_ready "true"
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "true"
    
    run bash "${BATS_TEST_DIRNAME}/initialize-database.sh"
    
    assert_success
    # Verify the connection parameters were used (this would be validated by the mock)
}

@test "script should use proper log functions" {
    mock::postgres::set_ready "true"
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "true"
    
    run bash "${BATS_TEST_DIRNAME}/initialize-database.sh"
    
    assert_success
    # Verify log output format
    assert_output_contains "[INFO]"
    assert_output_contains "[SUCCESS]"
}