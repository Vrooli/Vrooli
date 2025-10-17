#!/bin/bash
#
# Documentation Test Phase for knowledge-observatory
# Integrates with centralized Vrooli testing infrastructure
#

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Validating documentation..."

# Check required documentation files exist
doc_files=("README.md" "PRD.md" "PROBLEMS.md")

for doc in "${doc_files[@]}"; do
    if [ -f "$doc" ]; then
        testing::phase::success "Found $doc"

        # Check file is not empty
        if [ ! -s "$doc" ]; then
            testing::phase::error "$doc is empty"
            testing::phase::end_with_summary "Empty documentation file" 1
        fi
    else
        testing::phase::error "Missing required documentation: $doc"
        testing::phase::end_with_summary "Missing documentation" 1
    fi
done

# Validate README structure
testing::phase::log "Checking README.md structure..."

required_sections=("Overview" "Usage" "Features")
for section in "${required_sections[@]}"; do
    if grep -qi "^## .*$section" README.md || grep -qi "^# .*$section" README.md; then
        testing::phase::success "README has '$section' section"
    else
        testing::phase::warn "README missing '$section' section"
    fi
done

# Validate PRD structure
testing::phase::log "Checking PRD.md structure..."

prd_sections=("Requirements" "Architecture" "Value")
for section in "${prd_sections[@]}"; do
    if grep -qi "$section" PRD.md; then
        testing::phase::success "PRD mentions '$section'"
    else
        testing::phase::warn "PRD may be missing '$section' information"
    fi
done

# Check for broken file references in documentation
testing::phase::log "Checking for broken file references..."

broken_refs=0
for doc in "${doc_files[@]}"; do
    if [ -f "$doc" ]; then
        # Look for file paths that might not exist
        # Match patterns like: ./file, ../file, /path/to/file (but not URLs)
        while IFS= read -r ref; do
            # Extract just the file path, removing markdown link syntax
            file_ref=$(echo "$ref" | sed 's/.*(\([^)]*\)).*/\1/' | sed 's/.*`\([^`]*\)`.*/\1/' | tr -d '`')

            # Skip URLs
            if [[ "$file_ref" =~ ^https?:// ]]; then
                continue
            fi

            # Check if relative path exists
            if [[ "$file_ref" =~ ^\.\.?/ ]] || [[ "$file_ref" =~ ^/ ]]; then
                if [ ! -e "$file_ref" ] && [ ! -e "${TESTING_PHASE_SCENARIO_DIR}/${file_ref}" ]; then
                    testing::phase::warn "Potentially broken reference in $doc: $file_ref"
                    ((broken_refs++))
                fi
            fi
        done < <(grep -oE '\./[a-zA-Z0-9_/.-]+|\.\./[a-zA-Z0-9_/.-]+|`[a-zA-Z0-9_/.-]+`' "$doc" 2>/dev/null || true)
    fi
done

if [ "$broken_refs" -gt 0 ]; then
    testing::phase::warn "Found $broken_refs potentially broken file references"
else
    testing::phase::success "No obvious broken file references found"
fi

# Check API documentation consistency
testing::phase::log "Checking API documentation..."

if [ -f "PRD.md" ]; then
    # Check that documented endpoints match what's in the code
    if [ -f "api/main.go" ]; then
        # Extract endpoint paths from code
        code_endpoints=$(grep -oE '"\/(api\/v[0-9]+\/[a-zA-Z0-9/_-]+)"' api/main.go | sort -u | wc -l)
        # Count endpoint mentions in PRD
        prd_endpoints=$(grep -oE '\/api\/v[0-9]+\/[a-zA-Z0-9/_-]+' PRD.md | sort -u | wc -l)

        testing::phase::log "Found $code_endpoints endpoints in code, $prd_endpoints documented in PRD"

        if [ "$code_endpoints" -gt "$prd_endpoints" ]; then
            testing::phase::warn "Some API endpoints may not be documented in PRD"
        else
            testing::phase::success "API endpoints appear to be documented"
        fi
    fi
fi

testing::phase::end_with_summary "Documentation validation completed"
