#!/usr/bin/env bats
# Test file trees extractor independently

load "../helpers/setup.bash"

setup() {
    # Set up test environment
    TEST_DIR="$BATS_TEST_TMPDIR/filetrees_test"
    mkdir -p "$TEST_DIR"
    
    # Setup mocks from helpers
    mock_qdrant_collections
    mock_unified_embedding_service
    
    # Source the extractor
    source "$EMBEDDING_ROOT/extractors/filetrees/main.sh"
    
    # Create test fixtures
    setup_filetrees_fixtures
}

teardown() {
    rm -rf "$TEST_DIR"
}

setup_filetrees_fixtures() {
    # Create a realistic directory structure for testing
    mkdir -p "$TEST_DIR/src/components" "$TEST_DIR/src/utils" "$TEST_DIR/lib/api"
    mkdir -p "$TEST_DIR/test/unit" "$TEST_DIR/docs" "$TEST_DIR/config"
    
    # Add some files to make directories meaningful
    echo "export const Button = () => {}" > "$TEST_DIR/src/components/Button.tsx"
    echo "export const Header = () => {}" > "$TEST_DIR/src/components/Header.tsx"
    echo "export const validateEmail = () => {}" > "$TEST_DIR/src/utils/validation.js"
    echo "export const formatDate = () => {}" > "$TEST_DIR/src/utils/formatting.js"
    echo "const apiClient = {}" > "$TEST_DIR/lib/api/client.js"
    echo "describe('Button', () => {})" > "$TEST_DIR/test/unit/Button.test.js"
    echo "# Documentation" > "$TEST_DIR/docs/README.md"
    echo '{"name": "test-app"}' > "$TEST_DIR/config/package.json"
}

@test "file trees extractor finds semantic directories" {
    cd "$TEST_DIR"
    
    run qdrant::extract::find_semantic_directories "."
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"src/components"* ]]
    [[ "$output" == *"src/utils"* ]]
    [[ "$output" == *"lib/api"* ]]
}

@test "file trees extractor handles empty directory" {
    empty_dir="$TEST_DIR/empty"
    mkdir -p "$empty_dir"
    cd "$empty_dir"
    
    run qdrant::extract::find_semantic_directories "."
    
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "file trees extractor analyzes directory purpose correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::analyze_directory_purpose "src/components"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"frontend components"* ]]
}

@test "file trees extractor creates directory summary" {
    cd "$TEST_DIR"
    
    run qdrant::extract::directory_summary "src/components"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"frontend components"* ]]
    [[ "$output" == *"2 files"* ]]
    [[ "$output" == *"TypeScript"* ]]
}

@test "file trees extractor gets file distribution" {
    cd "$TEST_DIR"
    
    run qdrant::extract::get_file_distribution "src"
    
    [ "$status" -eq 0 ]
    # Should return JSON with file counts
    run echo "$output" | jq -e '.typescript > 0'
    [ "$status" -eq 0 ]
    run echo "$output" | jq -e '.javascript > 0'  
    [ "$status" -eq 0 ]
}

@test "file trees extractor identifies key files" {
    cd "$TEST_DIR"
    
    run qdrant::extract::get_key_files "src/components"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Button.tsx"* ]]
    [[ "$output" == *"Header.tsx"* ]]
}

@test "file trees batch extraction creates valid JSON" {
    cd "$TEST_DIR"
    output_file="$BATS_TEST_TMPDIR/test_output.jsonl"
    
    run qdrant::extract::file_trees_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    
    # Validate first line is valid JSON
    first_line=$(head -n 1 "$output_file")
    run echo "$first_line" | jq -e '.content'
    [ "$status" -eq 0 ]
    run echo "$first_line" | jq -e '.metadata.source_path'
    [ "$status" -eq 0 ]
}

@test "file trees extractor handles processing function" {
    cd "$TEST_DIR"
    
    # Create a mock output file to test the flow
    output_file="$TEMP_DIR/file_trees.jsonl"
    mkdir -p "$TEMP_DIR"
    echo '{"content":"test directory","metadata":{"directory_name":"test"}}' > "$output_file"
    
    # Mock the embedding function to return fixed count
    qdrant::embedding::process_jsonl_file() {
        echo "5"  # Return mock count
    }
    export -f qdrant::embedding::process_jsonl_file
    
    run qdrant::embeddings::process_file_trees "test-app"
    
    [ "$status" -eq 0 ]
    [[ "$output" == "5" ]]
}

@test "file trees extractor generates metadata from content" {
    # Test content parsing
    test_content="The src/components directory is a frontend components module containing 2 files"
    
    run qdrant::extract::filetrees_metadata_from_content "$test_content"
    
    [ "$status" -eq 0 ]
    run echo "$output" | jq -e '.directory_name == "components"'
    [ "$status" -eq 0 ]
}

@test "file trees extractor handles large directory structures" {
    # Create a larger directory structure
    for i in {1..10}; do
        mkdir -p "$TEST_DIR/module$i/subdir$i"
        echo "content" > "$TEST_DIR/module$i/file$i.js"
    done
    
    cd "$TEST_DIR"
    
    run qdrant::extract::find_semantic_directories "."
    
    [ "$status" -eq 0 ]
    # Should find at least some of the created directories
    line_count=$(echo "$output" | wc -l)
    [ "$line_count" -gt 5 ]
}