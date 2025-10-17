#!/bin/bash
# Business logic validation phase - validates end-to-end business workflows
source "$(dirname "${BASH_SOURCE[0]}")/../../../../scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 180-second target and require runtime
testing::phase::init --target-time "180s" --require-runtime

# Get API port dynamically
API_PORT=$(vrooli scenario port data-structurer API_PORT 2>/dev/null || echo "15774")
API_BASE_URL="http://localhost:$API_PORT"

echo "🔍 Testing business workflows at $API_BASE_URL..."

# Test 1: Complete Schema Creation Workflow
echo ""
echo "Test 1: Schema Creation Workflow"
SCHEMA_ID=""
TEST_SCHEMA_NAME="business-test-schema-$(date +%s)"

# Create schema
echo "  Creating test schema..."
CREATE_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/schemas" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"$TEST_SCHEMA_NAME\",
        \"description\": \"Test schema for business workflow validation\",
        \"schema_definition\": {
            \"type\": \"object\",
            \"properties\": {
                \"name\": {\"type\": \"string\"},
                \"email\": {\"type\": \"string\"},
                \"age\": {\"type\": \"number\"}
            },
            \"required\": [\"name\", \"email\"]
        }
    }")

if echo "$CREATE_RESPONSE" | jq -e '.id' >/dev/null 2>&1; then
    SCHEMA_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id')
    log::success "  ✅ Schema created successfully: $SCHEMA_ID"
    testing::phase::add_test passed
else
    testing::phase::add_error "  ❌ Failed to create schema"
    testing::phase::add_test failed
    testing::phase::end_with_summary "Business workflow validation failed early"
    exit 1
fi

# Retrieve schema
echo "  Retrieving created schema..."
GET_RESPONSE=$(curl -s "$API_BASE_URL/api/v1/schemas/$SCHEMA_ID")
if echo "$GET_RESPONSE" | jq -e ".id == \"$SCHEMA_ID\"" >/dev/null 2>&1; then
    log::success "  ✅ Schema retrieval successful"
    testing::phase::add_test passed
else
    testing::phase::add_error "  ❌ Failed to retrieve schema"
    testing::phase::add_test failed
fi

# Test 2: Data Processing Workflow
echo ""
echo "Test 2: Data Processing Workflow"
echo "  Processing text data with schema..."

PROCESS_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/process" \
    -H "Content-Type: application/json" \
    -d "{
        \"schema_id\": \"$SCHEMA_ID\",
        \"input_type\": \"text\",
        \"input_data\": \"John Doe, email: john.doe@example.com, age 30 years old\"
    }")

if echo "$PROCESS_RESPONSE" | jq -e '.processing_id' >/dev/null 2>&1; then
    PROCESSING_ID=$(echo "$PROCESS_RESPONSE" | jq -r '.processing_id')
    log::success "  ✅ Data processing initiated: $PROCESSING_ID"
    testing::phase::add_test passed

    # Check if processing completed
    STATUS=$(echo "$PROCESS_RESPONSE" | jq -r '.status')
    if [ "$STATUS" = "completed" ]; then
        log::success "  ✅ Processing completed successfully"
        testing::phase::add_test passed

        # Validate structured data extraction
        if echo "$PROCESS_RESPONSE" | jq -e '.structured_data.name' >/dev/null 2>&1; then
            EXTRACTED_NAME=$(echo "$PROCESS_RESPONSE" | jq -r '.structured_data.name')
            log::success "  ✅ Data extracted: name=$EXTRACTED_NAME"
            testing::phase::add_test passed
        else
            testing::phase::add_warning "  ⚠️  No structured data in response"
        fi

        # Check confidence score
        if echo "$PROCESS_RESPONSE" | jq -e '.confidence_score' >/dev/null 2>&1; then
            CONFIDENCE=$(echo "$PROCESS_RESPONSE" | jq -r '.confidence_score')
            log::success "  ✅ Confidence score: $CONFIDENCE"
        fi
    else
        testing::phase::add_warning "  ⚠️  Processing status: $STATUS (may be async)"
    fi
else
    testing::phase::add_error "  ❌ Failed to initiate processing"
    testing::phase::add_test failed
fi

# Test 3: Data Retrieval Workflow
echo ""
echo "Test 3: Data Retrieval Workflow"
echo "  Retrieving processed data for schema..."

DATA_RESPONSE=$(curl -s "$API_BASE_URL/api/v1/data/$SCHEMA_ID")
if echo "$DATA_RESPONSE" | jq -e '.data' >/dev/null 2>&1; then
    DATA_COUNT=$(echo "$DATA_RESPONSE" | jq '.data | length')
    log::success "  ✅ Retrieved $DATA_COUNT processed items"
    testing::phase::add_test passed
else
    testing::phase::add_error "  ❌ Failed to retrieve processed data"
    testing::phase::add_test failed
fi

# Test 4: Schema Template Workflow
echo ""
echo "Test 4: Schema Template Workflow"
echo "  Listing schema templates..."

TEMPLATES_RESPONSE=$(curl -s "$API_BASE_URL/api/v1/schema-templates")
if echo "$TEMPLATES_RESPONSE" | jq -e '.templates' >/dev/null 2>&1; then
    TEMPLATE_COUNT=$(echo "$TEMPLATES_RESPONSE" | jq '.templates | length')
    log::success "  ✅ Found $TEMPLATE_COUNT schema templates"
    testing::phase::add_test passed

    if [ "$TEMPLATE_COUNT" -ge 7 ]; then
        log::success "  ✅ Expected number of templates available (7+)"
    else
        testing::phase::add_warning "  ⚠️  Expected at least 7 templates, found $TEMPLATE_COUNT"
    fi
else
    testing::phase::add_error "  ❌ Failed to retrieve templates"
    testing::phase::add_test failed
fi

# Test 5: Schema Cleanup
echo ""
echo "Test 5: Schema Cleanup"
echo "  Deleting test schema..."

DELETE_RESPONSE=$(curl -s -X DELETE "$API_BASE_URL/api/v1/schemas/$SCHEMA_ID")
if echo "$DELETE_RESPONSE" | jq -e '.status == "deleted"' >/dev/null 2>&1; then
    log::success "  ✅ Schema deleted successfully"
    testing::phase::add_test passed
else
    testing::phase::add_warning "  ⚠️  Schema deletion may have failed (cleanup issue)"
fi

# End with summary
testing::phase::end_with_summary "Business workflow validation completed"
