#!/usr/bin/env bats

SCRIPT_PATH="$BATS_TEST_DIRNAME/lib/api.sh"

@test "debug search function" {
    run bash -c "
        set -e
        export SEARXNG_TEST_MODE=yes
        export SEARXNG_PORT=8100
        export SEARXNG_BASE_URL='http://localhost:8100'
        
        # Mock is_healthy
        searxng::is_healthy() { return 0; }
        
        # Mock curl
        curl() {
            echo '{\"results\": [{\"title\": \"Test\"}]}'
            return 0
        }
        
        # Source api.sh
        source '$SCRIPT_PATH' 2>&1
        
        # Try to run search
        searxng::search 'test query' 2>&1
    "
    
    echo "Status: $status"
    echo "Output: $output"
    [ "$status" -eq 0 ]
}