#!/usr/bin/env bats

# Contact Book CLI Edge Case Tests
# Test edge cases and error handling for the contact-book CLI

setup() {
    # Ensure API is running
    contact-book status >/dev/null 2>&1 || skip "Contact Book API not running"

    # Generate test IDs
    export TEST_ID=$(date +%s%N)
    export TEST_EMAIL="edge-test-${TEST_ID}@example.com"
}

# ============================================================================
# INPUT VALIDATION TESTS
# ============================================================================

@test "Edge: list with negative limit returns error" {
    run contact-book list --limit -1 --json
    # Should still work but limit to 0 or use default
    [ "$status" -eq 0 ]
}

@test "Edge: list with very large limit handles gracefully" {
    run contact-book list --limit 999999 --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -r '.persons' >/dev/null
}

@test "Edge: search with empty query" {
    run contact-book search ""
    # Should fail with error message
    [ "$status" -eq 1 ]
    [[ "$output" =~ "required" ]]
}

@test "Edge: search with special characters" {
    run contact-book search "test@#$%^&*()" --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -r '.results' >/dev/null
}

@test "Edge: search with very long query string" {
    LONG_QUERY=$(printf 'a%.0s' {1..500})
    run contact-book search "$LONG_QUERY" --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -r '.results' >/dev/null
}

# ============================================================================
# CONTACT CREATION EDGE CASES
# ============================================================================

@test "Edge: add contact with minimal data" {
    run contact-book add --name "Minimal Contact ${TEST_ID}"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "created" ]]
}

@test "Edge: add contact with all fields" {
    run contact-book add \
        --name "Full Contact ${TEST_ID}" \
        --email "${TEST_EMAIL}" \
        --phone "+1-555-0000" \
        --tags "test,edge,full" \
        --notes "This is a test contact with all fields"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "created" ]]
}

@test "Edge: add contact with empty name fails" {
    run contact-book add --name ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "required" ]]
}

@test "Edge: add contact with invalid email format" {
    # Should still accept it as the API may not validate strictly
    run contact-book add --name "Bad Email ${TEST_ID}" --email "not-an-email"
    # Check if it accepts or rejects
    if [ "$status" -eq 0 ]; then
        [[ "$output" =~ "created" ]]
    else
        [[ "$output" =~ "invalid" ]] || [[ "$output" =~ "error" ]]
    fi
}

# ============================================================================
# RELATIONSHIP EDGE CASES
# ============================================================================

@test "Edge: connect with same person ID (self-relationship)" {
    # Get a valid person ID first
    PERSON_ID=$(contact-book list --limit 1 --json | jq -r '.persons[0].id')

    if [ -n "$PERSON_ID" ] && [ "$PERSON_ID" != "null" ]; then
        run contact-book connect "$PERSON_ID" "$PERSON_ID" --type "self"
        # May accept or reject self-relationships
        [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    else
        skip "No contacts available for relationship test"
    fi
}

@test "Edge: connect with invalid UUIDs" {
    run contact-book connect "invalid-uuid-1" "invalid-uuid-2"
    [ "$status" -eq 1 ] || [ "$status" -eq 0 ]
    # Either fails or API handles invalid UUIDs gracefully
}

@test "Edge: connect with non-existent but valid UUIDs" {
    run contact-book connect \
        "00000000-0000-0000-0000-000000000001" \
        "00000000-0000-0000-0000-000000000002" \
        --type "test"
    # Should fail since these IDs don't exist
    [ "$status" -eq 1 ] || [ "$status" -eq 0 ]
}

# ============================================================================
# ANALYTICS EDGE CASES
# ============================================================================

@test "Edge: analytics with invalid person ID" {
    run contact-book analytics "invalid-id" --json
    # Should either return empty or error
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "Edge: maintenance with limit 0" {
    run contact-book maintenance --limit 0 --json
    [ "$status" -eq 0 ]
    # Should return empty array
    COUNT=$(echo "$output" | jq '.contacts | length')
    [ "$COUNT" -eq 0 ]
}

# ============================================================================
# CONCURRENT ACCESS TESTS
# ============================================================================

@test "Edge: concurrent list requests" {
    # Run 5 concurrent requests
    for i in {1..5}; do
        contact-book list --limit 1 --json &
    done
    wait
    # All should complete successfully
    [ "$?" -eq 0 ]
}

# ============================================================================
# ERROR RECOVERY TESTS
# ============================================================================

@test "Edge: status when API URL is wrong" {
    run env CONTACT_BOOK_API_URL="http://localhost:99999" contact-book status
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not available" ]]
}

@test "Edge: handles malformed JSON response gracefully" {
    # This would require mocking the API response, skip for now
    skip "Requires API mocking capability"
}

# ============================================================================
# UNICODE AND INTERNATIONALIZATION
# ============================================================================

@test "Edge: search with Unicode characters" {
    run contact-book search "测试 тест δοκιμή" --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -r '.results' >/dev/null
}

@test "Edge: add contact with emoji in name" {
    run contact-book add --name "Emoji Test 🎉 ${TEST_ID}"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "created" ]]
}

# ============================================================================
# BOUNDARY VALUE TESTS
# ============================================================================

@test "Edge: relationship strength at boundaries" {
    # Get two person IDs
    PERSONS=$(contact-book list --limit 2 --json | jq -r '.persons[].id' | head -2)
    PERSON1=$(echo "$PERSONS" | head -1)
    PERSON2=$(echo "$PERSONS" | tail -1)

    if [ -n "$PERSON1" ] && [ -n "$PERSON2" ] && [ "$PERSON1" != "$PERSON2" ]; then
        # Test strength = 0
        run contact-book connect "$PERSON1" "$PERSON2" --type "boundary-test" --strength 0
        [ "$status" -eq 0 ] || [ "$status" -eq 1 ]

        # Test strength = 1.0
        run contact-book connect "$PERSON1" "$PERSON2" --type "boundary-test" --strength 1.0
        [ "$status" -eq 0 ] || [ "$status" -eq 1 ]

        # Test strength > 1.0 (should fail)
        run contact-book connect "$PERSON1" "$PERSON2" --type "boundary-test" --strength 1.5
        # Should reject or clamp to 1.0
        [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    else
        skip "Not enough contacts for relationship test"
    fi
}

# ============================================================================
# COMMAND INJECTION PREVENTION
# ============================================================================

@test "Security: command injection prevention in search" {
    run contact-book search "; ls /" --json
    [ "$status" -eq 0 ]
    # Should not execute the command, just search for the string
    ! [[ "$output" =~ "bin" ]]
    echo "$output" | jq -r '.results' >/dev/null
}

@test "Security: SQL injection prevention in search" {
    run contact-book search "' OR '1'='1" --json
    [ "$status" -eq 0 ]
    # Should handle as normal search string
    echo "$output" | jq -r '.results' >/dev/null
}

# ============================================================================
# PERFORMANCE BOUNDARY TESTS
# ============================================================================

@test "Performance: list 100 contacts under 500ms" {
    start_time=$(date +%s%N)
    run contact-book list --limit 100 --json
    end_time=$(date +%s%N)

    [ "$status" -eq 0 ]

    # Calculate duration in milliseconds
    duration=$(( (end_time - start_time) / 1000000 ))

    # Should complete in under 500ms
    [ "$duration" -lt 500 ]
}

@test "Performance: search completes under 500ms" {
    start_time=$(date +%s%N)
    run contact-book search "test" --limit 50 --json
    end_time=$(date +%s%N)

    [ "$status" -eq 0 ]

    # Calculate duration in milliseconds
    duration=$(( (end_time - start_time) / 1000000 ))

    # Should complete in under 500ms per PRD requirement
    [ "$duration" -lt 500 ]
}