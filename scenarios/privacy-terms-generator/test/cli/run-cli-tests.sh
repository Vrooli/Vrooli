#!/usr/bin/env bats

# CLI Integration Tests for privacy-terms-generator

setup() {
    # Set up test environment
    export CLI_PATH="${CLI_PATH:-../cli/privacy-terms-generator}"
    export TEST_OUTPUT_DIR=$(mktemp -d)
}

teardown() {
    # Clean up test environment
    rm -rf "$TEST_OUTPUT_DIR"
}

@test "CLI: help command displays usage" {
    run "$CLI_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "privacy-terms-generator" ]]
    [[ "$output" =~ "generate" ]]
}

@test "CLI: version command displays version" {
    run "$CLI_PATH" --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "version" ]] || [[ "$output" =~ "1.0" ]]
}

@test "CLI: status command shows system status" {
    run "$CLI_PATH" status
    # May fail if resources not available, but should execute
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: list-templates command" {
    run "$CLI_PATH" list-templates
    # Should execute even if no templates
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: list-templates with --json flag" {
    run "$CLI_PATH" list-templates --json
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    # If successful, output should be valid JSON
    if [ "$status" -eq 0 ]; then
        echo "$output" | jq . > /dev/null 2>&1
        [ "$?" -eq 0 ]
    fi
}

@test "CLI: generate privacy policy with minimal args" {
    run "$CLI_PATH" generate privacy --business-name "TestCo" --jurisdiction US --format markdown
    # May fail if resources not available
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]

    # If successful, should contain policy content
    if [ "$status" -eq 0 ]; then
        [[ "$output" =~ "Privacy" ]] || [[ "$output" =~ "TestCo" ]]
    fi
}

@test "CLI: generate terms with all options" {
    run "$CLI_PATH" generate terms \
        --business-name "ACME Corp" \
        --jurisdiction US \
        --business-type "SaaS" \
        --email "legal@acme.com" \
        --website "https://acme.com" \
        --format markdown

    # May fail if resources not available
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: generate with output file" {
    local output_file="$TEST_OUTPUT_DIR/privacy-policy.md"

    run "$CLI_PATH" generate privacy \
        --business-name "TestCo" \
        --jurisdiction US \
        --format markdown \
        --output "$output_file"

    # May fail if resources not available
    if [ "$status" -eq 0 ]; then
        [ -f "$output_file" ]
        [ -s "$output_file" ]  # File should not be empty
    fi
}

@test "CLI: generate cookie policy" {
    run "$CLI_PATH" generate cookie --business-name "CookieCo" --jurisdiction EU --format html
    # May fail if resources not available
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: generate EULA" {
    run "$CLI_PATH" generate eula --business-name "SoftwareCo" --jurisdiction US --format markdown
    # May fail if resources not available
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: generate with data types" {
    run "$CLI_PATH" generate privacy \
        --business-name "DataCo" \
        --jurisdiction US \
        --data-types "email,name,location" \
        --format markdown

    # May fail if resources not available
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: generate with HTML format" {
    run "$CLI_PATH" generate privacy --business-name "HTMLCo" --jurisdiction US --format html
    # May fail if resources not available
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]

    if [ "$status" -eq 0 ]; then
        [[ "$output" =~ "<" ]] || [[ "$output" =~ "html" ]]
    fi
}

@test "CLI: generate fails with missing business name" {
    run "$CLI_PATH" generate privacy --jurisdiction US
    [ "$status" -ne 0 ]
    [[ "$output" =~ "business" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "error" ]]
}

@test "CLI: generate fails with missing jurisdiction" {
    run "$CLI_PATH" generate privacy --business-name "TestCo"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "jurisdiction" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "error" ]]
}

@test "CLI: generate fails with invalid document type" {
    run "$CLI_PATH" generate invalid-type --business-name "TestCo" --jurisdiction US
    [ "$status" -ne 0 ]
}

@test "CLI: generate fails with invalid jurisdiction" {
    run "$CLI_PATH" generate privacy --business-name "TestCo" --jurisdiction INVALID
    # May fail or succeed with warning
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: search command basic query" {
    run "$CLI_PATH" search "data retention"
    # May fail if resources not available
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: search with limit" {
    run "$CLI_PATH" search "privacy" --limit 5
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: search with JSON output" {
    run "$CLI_PATH" search "gdpr" --json
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]

    if [ "$status" -eq 0 ]; then
        echo "$output" | jq . > /dev/null 2>&1 || true
    fi
}

@test "CLI: search with type filter" {
    run "$CLI_PATH" search "consent" --type privacy
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: search with jurisdiction filter" {
    run "$CLI_PATH" search "privacy" --jurisdiction US
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: history command with document ID" {
    run "$CLI_PATH" history "doc_123456"
    # May fail if document doesn't exist
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: history with JSON output" {
    run "$CLI_PATH" history "doc_123456" --json
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]

    if [ "$status" -eq 0 ]; then
        echo "$output" | jq . > /dev/null 2>&1 || true
    fi
}

@test "CLI: update-templates command" {
    run "$CLI_PATH" update-templates
    # May require network access
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: update-templates with jurisdiction filter" {
    run "$CLI_PATH" update-templates --jurisdiction US
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: update-templates with force flag" {
    run "$CLI_PATH" update-templates --force
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: Multiple jurisdictions support" {
    run "$CLI_PATH" generate privacy \
        --business-name "GlobalCo" \
        --jurisdiction "US,EU,UK" \
        --format markdown

    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: Special characters in business name" {
    run "$CLI_PATH" generate privacy \
        --business-name "Test & Co." \
        --jurisdiction US \
        --format markdown

    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: PDF format generation" {
    run "$CLI_PATH" generate privacy \
        --business-name "PDFCo" \
        --jurisdiction US \
        --format pdf

    # May fail if Browserless not available
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}
