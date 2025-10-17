#!/bin/bash

# Tech Tree Designer Database Schema Tests
# Validates database structure and seed data integrity

set -e

# Configuration
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_NAME=${DB_NAME:-vrooli}
export PGPASSWORD=${DB_PASSWORD:-postgres}

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TESTS_PASSED=0
TESTS_FAILED=0

log_test() {
    echo -e "${BLUE}🧪 Testing: $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    ((TESTS_FAILED++))
}

run_query() {
    local query="$1"
    local description="$2"
    
    log_test "$description"
    
    if result=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$query" 2>/dev/null); then
        echo "$result"
        return 0
    else
        log_error "Query failed: $description"
        return 1
    fi
}

test_table_exists() {
    local table_name="$1"
    local description="Table '$table_name' exists"
    
    log_test "$description"
    
    if run_query "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '$table_name');" "Check table $table_name" | grep -q "t"; then
        log_success "$description"
        return 0
    else
        log_error "$description"
        return 1
    fi
}

test_data_count() {
    local table_name="$1"
    local expected_min="$2"
    local description="Table '$table_name' has data (min $expected_min records)"
    
    log_test "$description"
    
    if count=$(run_query "SELECT COUNT(*) FROM $table_name;" "Count records in $table_name" | xargs); then
        if [[ $count -ge $expected_min ]]; then
            log_success "$description - Found $count records"
            return 0
        else
            log_error "$description - Found $count records, expected at least $expected_min"
            return 1
        fi
    else
        return 1
    fi
}

echo -e "${BLUE}🗄️  Tech Tree Designer Database Tests${NC}"
echo -e "${BLUE}======================================${NC}"

# Test database connection
log_test "Database connection"
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
    log_success "Connected to database $DB_NAME"
else
    log_error "Failed to connect to database $DB_NAME"
    exit 1
fi

echo -e "\n${YELLOW}📋 Schema Structure Tests${NC}"

# Test all required tables exist
test_table_exists "tech_trees"
test_table_exists "sectors"
test_table_exists "progression_stages"
test_table_exists "scenario_mappings"
test_table_exists "stage_dependencies"
test_table_exists "sector_connections"
test_table_exists "strategic_milestones"
test_table_exists "strategic_analyses"
test_table_exists "progress_events"

echo -e "\n${YELLOW}📊 Seed Data Validation${NC}"

# Test seed data exists
test_data_count "tech_trees" 1
test_data_count "sectors" 6
test_data_count "progression_stages" 20
test_data_count "strategic_milestones" 3
test_data_count "stage_dependencies" 5

echo -e "\n${YELLOW}🔗 Relationship Integrity Tests${NC}"

# Test foreign key relationships
log_test "Sector-to-tree relationships"
if orphaned=$(run_query "SELECT COUNT(*) FROM sectors WHERE tree_id NOT IN (SELECT id FROM tech_trees);" "Orphaned sectors" | xargs); then
    if [[ $orphaned -eq 0 ]]; then
        log_success "All sectors reference valid tech trees"
    else
        log_error "Found $orphaned orphaned sectors"
    fi
fi

log_test "Stage-to-sector relationships" 
if orphaned=$(run_query "SELECT COUNT(*) FROM progression_stages WHERE sector_id NOT IN (SELECT id FROM sectors);" "Orphaned stages" | xargs); then
    if [[ $orphaned -eq 0 ]]; then
        log_success "All progression stages reference valid sectors"
    else
        log_error "Found $orphaned orphaned progression stages"
    fi
fi

log_test "Scenario mapping relationships"
if orphaned=$(run_query "SELECT COUNT(*) FROM scenario_mappings WHERE stage_id NOT IN (SELECT id FROM progression_stages);" "Orphaned scenario mappings" | xargs); then
    if [[ $orphaned -eq 0 ]]; then
        log_success "All scenario mappings reference valid stages"
    else
        log_error "Found $orphaned orphaned scenario mappings"
    fi
fi

echo -e "\n${YELLOW}⚡ Function and Trigger Tests${NC}"

# Test progress calculation function
log_test "Progress calculation trigger"
if run_query "INSERT INTO scenario_mappings (id, scenario_name, stage_id, completion_status, contribution_weight) 
             VALUES ('test-123', 'test-scenario', (SELECT id FROM progression_stages LIMIT 1), 'completed', 1.0);" \
             "Insert test scenario mapping" > /dev/null; then
    
    # Check if progress was updated
    if updated_progress=$(run_query "SELECT progress_percentage FROM progression_stages WHERE id = (SELECT stage_id FROM scenario_mappings WHERE id = 'test-123');" "Check progress update" | xargs); then
        if [[ $(echo "$updated_progress > 0" | bc -l) -eq 1 ]]; then
            log_success "Progress calculation trigger works - progress: $updated_progress%"
        else
            log_error "Progress calculation trigger failed - progress: $updated_progress%"
        fi
    fi
    
    # Clean up test data
    run_query "DELETE FROM scenario_mappings WHERE id = 'test-123';" "Clean up test data" > /dev/null
fi

echo -e "\n${YELLOW}📈 Data Quality Tests${NC}"

# Test progress percentage constraints
log_test "Progress percentage constraints"
if valid_progress=$(run_query "SELECT COUNT(*) FROM progression_stages WHERE progress_percentage >= 0 AND progress_percentage <= 100;" "Valid progress values" | xargs); then
    total_stages=$(run_query "SELECT COUNT(*) FROM progression_stages;" "Total stages" | xargs)
    if [[ $valid_progress -eq $total_stages ]]; then
        log_success "All progress percentages are valid (0-100%)"
    else
        log_error "Found invalid progress percentages: $((total_stages - valid_progress)) out of $total_stages"
    fi
fi

# Test strategic milestone business values
log_test "Strategic milestone values"
if run_query "SELECT name, business_value_estimate FROM strategic_milestones ORDER BY business_value_estimate DESC;" "Milestone values" | while read -r line; do
    echo "  $line"
done > /dev/null; then
    log_success "Strategic milestone values are accessible"
fi

echo -e "\n${YELLOW}🎯 Civilization Tree Validation${NC}"

# Test civilization tree structure
log_test "Civilization tech tree completeness"
sector_categories=$(run_query "SELECT DISTINCT category FROM sectors;" "Sector categories" | xargs | wc -w)
if [[ $sector_categories -ge 6 ]]; then
    log_success "Found $sector_categories technology sectors (manufacturing, healthcare, software, etc.)"
else
    log_error "Insufficient sector diversity - found only $sector_categories categories"
fi

# Test progression stage types
log_test "Progression stage completeness"
stage_types=$(run_query "SELECT DISTINCT stage_type FROM progression_stages;" "Stage types" | xargs | wc -w)
if [[ $stage_types -ge 5 ]]; then
    log_success "Found $stage_types progression stages (foundation → digital_twin pathway)"
else
    log_error "Incomplete progression pathway - found only $stage_types stage types"
fi

echo -e "\n${BLUE}📊 Test Results Summary${NC}"
echo -e "${BLUE}======================${NC}"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "\n${GREEN}🎉 All database tests passed! Strategic Intelligence data model is intact.${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  Some database tests failed. Check schema and seed data.${NC}"
    exit 1
fi