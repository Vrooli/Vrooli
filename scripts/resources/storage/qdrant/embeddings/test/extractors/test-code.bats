#!/usr/bin/env bats
# Test code extractor independently

load "../helpers/setup.bash"

setup() {
    # Set up test environment
    TEST_DIR="$BATS_TEST_TMPDIR/code_test"
    mkdir -p "$TEST_DIR"
    
    # Source the extractor
    source "$EMBEDDING_ROOT/extractors/code.sh"
    
    # Create test fixtures
    setup_code_fixtures
}

teardown() {
    rm -rf "$TEST_DIR"
}

setup_code_fixtures() {
    # Copy code test fixtures
    mkdir -p "$TEST_DIR/lib" "$TEST_DIR/src"
    cp "$FIXTURE_ROOT/code/email-service.sh" "$TEST_DIR/lib/"
    cp "$FIXTURE_ROOT/code/api-routes.ts" "$TEST_DIR/src/"
    cp "$FIXTURE_ROOT/code/database-queries.sql" "$TEST_DIR/lib/"
}

@test "code extractor finds code files" {
    cd "$TEST_DIR"
    
    run qdrant::extract::find_code_files "."
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"email-service.sh"* ]]
    [[ "$output" == *"api-routes.ts"* ]]
    [[ "$output" == *"database-queries.sql"* ]]
}

@test "code extractor handles empty directory" {
    empty_dir="$TEST_DIR/empty"
    mkdir -p "$empty_dir"
    cd "$empty_dir"
    
    run qdrant::extract::find_code_files "."
    
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "code extractor processes bash functions correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::bash_functions "lib/email-service.sh"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"email::send_notification"* ]]
    [[ "$output" == *"email::validate_address"* ]]
    [[ "$output" == *"email::fetch_emails"* ]]
    [[ "$output" == *"email::process_batch"* ]]
}

@test "code extractor processes TypeScript functions correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::typescript_functions "src/api-routes.ts"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"GET /api/emails"* ]]
    [[ "$output" == *"POST /api/emails"* ]]
    [[ "$output" == *"setupEmailWebSocket"* ]]
}

@test "code extractor processes SQL queries correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::sql_queries "lib/database-queries.sql"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"CREATE TABLE"* ]]
    [[ "$output" == *"SELECT"* ]]
    [[ "$output" == *"UPDATE emails"* ]]
    [[ "$output" == *"email analytics"* ]]
}

@test "code extractor handles missing file" {
    cd "$TEST_DIR"
    
    run qdrant::extract::bash_functions "nonexistent.sh"
    
    [ "$status" -ne 0 ]
}

@test "code extractor extracts function documentation" {
    cd "$TEST_DIR"
    
    run qdrant::extract::bash_functions "lib/email-service.sh"
    
    [ "$status" -eq 0 ]
    # Should extract function comments
    [[ "$output" == *"Send notification email"* ]]
    [[ "$output" == *"Arguments:"* ]]
    [[ "$output" == *"Returns:"* ]]
}

@test "code extractor processes batch correctly" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/code_output.txt"
    
    run qdrant::extract::code_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    
    # Check output contains expected content
    grep -q "email::send_notification" "$output_file"
    grep -q "GET /api/emails" "$output_file"
    grep -q "CREATE TABLE" "$output_file"
    grep -q "---FUNCTION---" "$output_file"
    grep -q "---ENDPOINT---" "$output_file"
    grep -q "---QUERY---" "$output_file"
}

@test "code extractor detects API endpoints with methods" {
    cd "$TEST_DIR"
    
    run qdrant::extract::typescript_functions "src/api-routes.ts"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"GET /api/emails"* ]]
    [[ "$output" == *"POST /api/emails/:id/categorize"* ]]
    [[ "$output" == *"POST /api/emails/batch/process"* ]]
}

@test "code extractor extracts function parameters" {
    cd "$TEST_DIR"
    
    run qdrant::extract::bash_functions "lib/email-service.sh"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"\$1 - Recipient email"* ]]
    [[ "$output" == *"\$2 - Subject line"* ]]
    [[ "$output" == *"\$3 - Message body"* ]]
}

@test "code extractor handles code with no functions" {
    cd "$TEST_DIR"
    
    # Create file with just variables and comments
    cat > "src/config.ts" << 'EOF'
// Configuration file
export const API_VERSION = "v1";
export const DEFAULT_TIMEOUT = 5000;

// Database settings
const DB_CONFIG = {
  host: 'localhost',
  port: 5432
};
EOF
    
    run qdrant::extract::typescript_functions "src/config.ts"
    
    [ "$status" -eq 0 ]
    # Should still extract some content
    [ -n "$output" ]
}

@test "code extractor filters node_modules correctly" {
    cd "$TEST_DIR"
    
    # Create file in node_modules (should be ignored)
    mkdir -p "node_modules/some-package"
    echo "function ignored() { }" > "node_modules/some-package/index.js"
    
    run qdrant::extract::find_code_files "."
    
    [ "$status" -eq 0 ]
    [[ "$output" != *"node_modules"* ]]
}

@test "code extractor detects design patterns" {
    cd "$TEST_DIR"
    
    # Create file with common patterns
    cat > "src/patterns.ts" << 'EOF'
// Singleton pattern
class DatabaseConnection {
    private static instance: DatabaseConnection;
    
    public static getInstance(): DatabaseConnection {
        if (!DatabaseConnection.instance) {
            DatabaseConnection.instance = new DatabaseConnection();
        }
        return DatabaseConnection.instance;
    }
}

// Factory pattern
class EmailFactory {
    static createEmail(type: string): Email {
        switch(type) {
            case 'notification': return new NotificationEmail();
            case 'marketing': return new MarketingEmail();
            default: throw new Error('Unknown email type');
        }
    }
}

// Observer pattern
class EventEmitter {
    private listeners: Map<string, Function[]> = new Map();
    
    on(event: string, callback: Function): void {
        // Observer implementation
    }
}
EOF
    
    run qdrant::extract::detect_patterns "src/patterns.ts"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Singleton"* ]]
    [[ "$output" == *"Factory"* ]]
    [[ "$output" == *"Observer"* ]]
}

@test "code extractor handles syntax errors gracefully" {
    cd "$TEST_DIR"
    
    # Create file with syntax errors
    cat > "src/broken.ts" << 'EOF'
function broken( {
    // Missing closing brace and parameter
    return "invalid syntax"
}

// Missing export keyword
const INVALID = 
EOF
    
    run qdrant::extract::typescript_functions "src/broken.ts"
    
    # Should handle gracefully, not crash
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "code extractor output format is consistent" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/format_test.txt"
    run qdrant::extract::code_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    
    # Check format consistency
    local function_count=$(grep -c "---FUNCTION---" "$output_file")
    local endpoint_count=$(grep -c "---ENDPOINT---" "$output_file")
    local query_count=$(grep -c "---QUERY---" "$output_file")
    local separator_count=$(grep -c "---SEPARATOR---" "$output_file")
    
    # Should have found functions, endpoints, and queries
    [ "$function_count" -gt 0 ]
    [ "$endpoint_count" -gt 0 ]
    [ "$query_count" -gt 0 ]
}