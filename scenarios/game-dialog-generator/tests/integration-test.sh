#!/bin/bash

# Game Dialog Generator Integration Tests 🌿🎮
# Jungle Platformer Adventure Theme

set -e

# Colors for jungle theme output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_PORT=${API_PORT:-8080}
UI_PORT=${UI_PORT:-3200}
API_BASE_URL="http://localhost:${API_PORT}"
UI_BASE_URL="http://localhost:${UI_PORT}"

print_header() {
    echo -e "${GREEN}🌿=".repeat(50)"${NC}"
    echo -e "${GREEN}🎮 Game Dialog Generator Integration Tests 🌿${NC}"
    echo -e "${GREEN}🌿=".repeat(50)"${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_test() {
    echo -e "${YELLOW}🧪 Testing: $1${NC}"
}

# Wait for service to be ready
wait_for_service() {
    local url="$1"
    local name="$2"
    local max_attempts=30
    local attempt=1
    
    print_info "Waiting for $name to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url" >/dev/null 2>&1; then
            print_success "$name is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$name failed to start after $max_attempts attempts"
    return 1
}

# Test API endpoint
test_api_endpoint() {
    local method="$1"
    local endpoint="$2" 
    local data="$3"
    local expected_status="$4"
    local description="$5"
    
    print_test "$description"
    
    local response
    local status
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$API_BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$API_BASE_URL$endpoint")
    fi
    
    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$status" -eq "$expected_status" ]; then
        print_success "API test passed ($status)"
        echo "$body"
        return 0
    else
        print_error "API test failed (expected $expected_status, got $status)"
        echo "Response: $body"
        return 1
    fi
}

# Test CLI command
test_cli_command() {
    local command="$1"
    local expected_exit="$2"
    local description="$3"
    
    print_test "$description"
    
    if eval "$command" >/dev/null 2>&1; then
        local exit_code=$?
        if [ $exit_code -eq $expected_exit ]; then
            print_success "CLI test passed"
            return 0
        else
            print_error "CLI test failed (expected exit $expected_exit, got $exit_code)"
            return 1
        fi
    else
        print_error "CLI command failed to execute"
        return 1
    fi
}

# Main test execution
main() {
    print_header
    
    local test_count=0
    local passed_count=0
    local failed_count=0
    
    # Wait for services
    if ! wait_for_service "$API_BASE_URL/health" "API Server"; then
        exit 1
    fi
    
    if ! wait_for_service "$UI_BASE_URL/health" "UI Server"; then
        exit 1
    fi
    
    echo ""
    print_info "🌿 Running Jungle Adventure Tests 🎮"
    echo ""
    
    # Test 1: API Health Check
    test_count=$((test_count + 1))
    if test_api_endpoint "GET" "/health" "" 200 "API health check with jungle theme"; then
        passed_count=$((passed_count + 1))
    else
        failed_count=$((failed_count + 1))
    fi
    echo ""
    
    # Test 2: Character Creation
    test_count=$((test_count + 1))
    character_data='{
        "name": "Test Jungle Hero",
        "personality_traits": {
            "brave": 0.8,
            "humorous": 0.6,
            "loyal": 0.7
        },
        "background_story": "A brave monkey created for integration testing",
        "voice_profile": {
            "pitch": "medium"
        }
    }'
    
    if character_response=$(test_api_endpoint "POST" "/api/v1/characters" "$character_data" 201 "Create test jungle character"); then
        character_id=$(echo "$character_response" | jq -r '.character_id' 2>/dev/null || echo "")
        passed_count=$((passed_count + 1))
        print_info "Created character with ID: $character_id"
    else
        failed_count=$((failed_count + 1))
        character_id=""
    fi
    echo ""
    
    # Test 3: Character Listing
    test_count=$((test_count + 1))
    if test_api_endpoint "GET" "/api/v1/characters" "" 200 "List jungle characters"; then
        passed_count=$((passed_count + 1))
    else
        failed_count=$((failed_count + 1))
    fi
    echo ""
    
    # Test 4: Dialog Generation (if character was created)
    if [ -n "$character_id" ]; then
        test_count=$((test_count + 1))
        dialog_data="{
            \"character_id\": \"$character_id\",
            \"scene_context\": \"A peaceful jungle clearing for testing\",
            \"emotion_state\": \"confident\"
        }"
        
        if test_api_endpoint "POST" "/api/v1/dialog/generate" "$dialog_data" 200 "Generate dialog for test character"; then
            passed_count=$((passed_count + 1))
        else
            failed_count=$((failed_count + 1))
        fi
        echo ""
    fi
    
    # Test 5: Project Creation
    test_count=$((test_count + 1))
    project_data='{
        "name": "Test Jungle Adventure Game",
        "description": "Integration test project for jungle platformer",
        "settings": {
            "theme": "jungle-adventure"
        }
    }'
    
    if test_api_endpoint "POST" "/api/v1/projects" "$project_data" 201 "Create test game project"; then
        passed_count=$((passed_count + 1))
    else
        failed_count=$((failed_count + 1))
    fi
    echo ""
    
    # Test 6: UI Health Check
    test_count=$((test_count + 1))
    if test_api_endpoint "GET" "/health" "" 200 "UI server health with jungle theme"; then
        passed_count=$((passed_count + 1))
    else
        failed_count=$((failed_count + 1))
    fi
    echo ""
    
    # Test 7: CLI Help Command
    if command -v game-dialog-generator >/dev/null 2>&1; then
        test_count=$((test_count + 1))
        if test_cli_command "game-dialog-generator --help" 0 "CLI help command with jungle theme"; then
            passed_count=$((passed_count + 1))
        else
            failed_count=$((failed_count + 1))
        fi
        echo ""
        
        # Test 8: CLI Status Command
        test_count=$((test_count + 1))
        if test_cli_command "game-dialog-generator status" 0 "CLI status command"; then
            passed_count=$((passed_count + 1))
        else
            failed_count=$((failed_count + 1))
        fi
        echo ""
    else
        print_info "CLI not installed, skipping CLI tests"
    fi
    
    # Test Results Summary
    echo ""
    print_header
    print_info "🌿 Jungle Adventure Test Results 🎮"
    echo ""
    
    if [ $failed_count -eq 0 ]; then
        print_success "All $test_count tests passed! 🌿🎮"
        print_success "Game Dialog Generator is ready for jungle adventures!"
        echo ""
        print_info "Next steps:"
        echo "  1. 🐒 Create characters: $UI_BASE_URL/characters"
        echo "  2. 💬 Generate dialog: $UI_BASE_URL/dialog"
        echo "  3. 🎮 Manage projects: $UI_BASE_URL/projects"
        echo "  4. 💻 Use CLI: game-dialog-generator --help"
        exit 0
    else
        print_error "$failed_count out of $test_count tests failed"
        print_error "Game Dialog Generator needs attention before jungle adventures can begin"
        exit 1
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        print_error "Missing dependencies: ${missing_deps[*]}"
        print_info "Install with: sudo apt-get install ${missing_deps[*]} (Ubuntu/Debian)"
        exit 1
    fi
}

# Show usage
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --api-port PORT    API server port (default: $API_PORT)"
    echo "  --ui-port PORT     UI server port (default: $UI_PORT)"
    echo "  --help             Show this help message"
    echo ""
    echo "🌿 Game Dialog Generator Integration Tests 🎮"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --api-port)
            API_PORT="$2"
            API_BASE_URL="http://localhost:${API_PORT}"
            shift 2
            ;;
        --ui-port)
            UI_PORT="$2"
            UI_BASE_URL="http://localhost:${UI_PORT}"
            shift 2
            ;;
        --help|-h)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Run the tests
check_dependencies
main "$@"