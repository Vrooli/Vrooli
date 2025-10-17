#!/usr/bin/env bats
# Test docs extractor independently

load "../helpers/setup.bash"

setup() {
    # Set up test environment
    TEST_DIR="$BATS_TEST_TMPDIR/docs_test"
    mkdir -p "$TEST_DIR"
    
    # Source the extractor
    source "$EMBEDDING_ROOT/extractors/docs/main.sh"
    
    # Create test fixtures
    setup_docs_fixtures
}

teardown() {
    rm -rf "$TEST_DIR"
}

setup_docs_fixtures() {
    # Copy docs test fixtures
    mkdir -p "$TEST_DIR/docs"
    cp "$FIXTURE_ROOT/docs/ARCHITECTURE.md" "$TEST_DIR/docs/"
    cp "$FIXTURE_ROOT/docs/LESSONS_LEARNED.md" "$TEST_DIR/docs/"
}

@test "docs extractor finds documentation files" {
    cd "$TEST_DIR"
    
    run qdrant::extract::docs_batch "." "$TEST_DIR/docs_output.txt"
    
    [ "$status" -eq 0 ]
    [ -f "$TEST_DIR/docs_output.txt" ]
}

@test "docs extractor handles empty directory" {
    empty_dir="$TEST_DIR/empty"
    mkdir -p "$empty_dir"
    cd "$empty_dir"
    
    run qdrant::extract::docs_batch "." "$empty_dir/empty_output.txt"
    
    [ "$status" -eq 0 ]
}

@test "docs extractor processes single doc correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::standard_doc "docs/ARCHITECTURE.md"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"ARCHITECTURE.md"* ]]
}

@test "docs extractor handles missing file" {
    cd "$TEST_DIR"
    
    run qdrant::extract::standard_doc "nonexistent.md"
    
    [ "$status" -ne 0 ]
}

@test "docs extractor extracts embedding markers correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::standard_doc "docs/ARCHITECTURE.md"
    
    [ "$status" -eq 0 ]
    # Should extract marked sections
    [[ "$output" == *"EMBED:DECISION"* ]]
    [[ "$output" == *"EMBED:PATTERN"* ]]
    [[ "$output" == *"EMBED:SECURITY"* ]]
    [[ "$output" == *"EMBED:PERFORMANCE"* ]]
}

@test "docs extractor processes lessons learned correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::standard_doc "docs/LESSONS_LEARNED.md"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"EMBED:SUCCESS"* ]]
    [[ "$output" == *"EMBED:FAILURE"* ]]
    [[ "$output" == *"EMBED:INSIGHT"* ]]
    [[ "$output" == *"Early User Feedback"* ]]
    [[ "$output" == *"Over-Engineering"* ]]
}

@test "docs extractor processes batch correctly" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/docs_output.txt"
    
    run qdrant::extract::docs_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    
    # Check output contains expected content
    grep -q "PostgreSQL vs MongoDB" "$output_file"
    grep -q "Early User Feedback" "$output_file"
    grep -q -- "---SECTION---" "$output_file"
}

@test "docs extractor handles files without embedding markers" {
    cd "$TEST_DIR"
    
    # Create doc without embedding markers
    echo -e "# Simple Doc\n\nThis is just regular content without markers." > "docs/SIMPLE.md"
    
    run qdrant::extract::standard_doc "docs/SIMPLE.md"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Simple Doc"* ]]
    [[ "$output" == *"regular content"* ]]
}

@test "docs extractor preserves markdown formatting" {
    cd "$TEST_DIR"
    
    run qdrant::extract::standard_doc "docs/ARCHITECTURE.md"
    
    [ "$status" -eq 0 ]
    # Should preserve important formatting
    [[ "$output" == *"**Context:**"* ]]
    [[ "$output" == *"**Decision:**"* ]]
    [[ "$output" == *"- Strong consistency"* ]]
}

@test "docs extractor handles nested sections" {
    cd "$TEST_DIR"
    
    run qdrant::extract::standard_doc "docs/LESSONS_LEARNED.md"
    
    [ "$status" -eq 0 ]
    # Should handle nested embedding sections
    [[ "$output" == *"What Worked Well"* ]]
    [[ "$output" == *"What Didn't Work"* ]]
    [[ "$output" == *"Technical Insights"* ]]
}

@test "docs extractor provides coverage report" {
    cd "$TEST_DIR"
    
    run qdrant::extract::docs_coverage "."
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"ARCHITECTURE.md"* ]]
    [[ "$output" == *"LESSONS_LEARNED.md"* ]]
}

@test "docs extractor identifies missing standard docs" {
    cd "$TEST_DIR"
    
    run qdrant::extract::docs_coverage "."
    
    [ "$status" -eq 0 ]
    # Should note missing standard docs
    [[ "$output" == *"SECURITY.md"* ]] || [[ "$output" == *"missing"* ]]
    [[ "$output" == *"PATTERNS.md"* ]] || [[ "$output" == *"missing"* ]]
}

@test "docs extractor handles very large documents" {
    cd "$TEST_DIR"
    
    # Create a large document
    {
        echo "# Large Document"
        for i in {1..100}; do
            echo "<!-- EMBED:TEST:START -->"
            echo "## Section $i"
            echo "Content for section $i with lots of detail and information."
            for j in {1..10}; do
                echo "Line $j of section $i with additional content."
            done
            echo "<!-- EMBED:TEST:END -->"
        done
    } > "docs/LARGE.md"
    
    run qdrant::extract::standard_doc "docs/LARGE.md"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Section 1"* ]]
    [[ "$output" == *"Section 100"* ]]
}

@test "docs extractor output format is consistent" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/format_test.txt"
    run qdrant::extract::docs_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    
    # Check format consistency
    local section_count=$(grep -c -- "---SECTION---" "$output_file")
    local separator_count=$(grep -c -- "---SEPARATOR---" "$output_file")
    
    # Should have appropriate separators
    [ "$section_count" -gt 0 ]  # Should have sections
    [ "$separator_count" -ge 0 ]  # May have file separators
}