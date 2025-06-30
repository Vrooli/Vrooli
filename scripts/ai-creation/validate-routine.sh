#!/usr/bin/env bash
# validate-routine.sh - Standalone routine JSON validation
# This script provides comprehensive validation for routine JSON files

set -e

# Helper function to safely check if command succeeds
safe_check() {
    local cmd="$1"
    local file="$2"
    if eval "$cmd" "$file" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function: Validate single-step routine configurations
validate_single_step_config() {
    local json_file="$1"
    local resource_sub_type="$2"
    local errors=0
    
    echo "🔧 Validating $resource_sub_type configuration..."
    
    case "$resource_sub_type" in
        "RoutineGenerate")
            # Check for callDataGenerate
            if ! safe_check "jq -e '.versions[0].config | has(\"callDataGenerate\")'" "$json_file"; then
                echo "  ❌ Missing callDataGenerate configuration"
                ((errors++))
            else
                # Check for required generate fields
                if ! safe_check "jq -e '.versions[0].config.callDataGenerate.schema | has(\"prompt\")'" "$json_file"; then
                    echo "  ❌ Missing prompt in callDataGenerate"
                    ((errors++))
                fi
                if ! safe_check "jq -e '.versions[0].config.callDataGenerate.schema | has(\"botStyle\")'" "$json_file"; then
                    echo "  ❌ Missing botStyle in callDataGenerate"
                    ((errors++))
                fi
            fi
            ;;
        "RoutineApi")
            # Check for callDataApi
            if ! safe_check "jq -e '.versions[0].config | has(\"callDataApi\")'" "$json_file"; then
                echo "  ❌ Missing callDataApi configuration"
                ((errors++))
            else
                # Check for required API fields
                if ! safe_check "jq -e '.versions[0].config.callDataApi.schema | has(\"endpoint\")'" "$json_file"; then
                    echo "  ❌ Missing endpoint in callDataApi"
                    ((errors++))
                fi
                if ! safe_check "jq -e '.versions[0].config.callDataApi.schema | has(\"method\")'" "$json_file"; then
                    echo "  ❌ Missing method in callDataApi"
                    ((errors++))
                fi
            fi
            ;;
        "RoutineCode")
            # Check for callDataCode
            if ! safe_check "jq -e '.versions[0].config | has(\"callDataCode\")'" "$json_file"; then
                echo "  ❌ Missing callDataCode configuration"
                ((errors++))
            else
                # Check for required code fields
                if ! safe_check "jq -e '.versions[0].config.callDataCode.schema | has(\"inputTemplate\")'" "$json_file"; then
                    echo "  ❌ Missing inputTemplate in callDataCode"
                    ((errors++))
                fi
                if ! safe_check "jq -e '.versions[0].config.callDataCode.schema | has(\"outputMappings\")'" "$json_file"; then
                    echo "  ❌ Missing outputMappings in callDataCode"
                    ((errors++))
                fi
            fi
            ;;
        "RoutineWeb")
            # Check for callDataWeb
            if ! safe_check "jq -e '.versions[0].config | has(\"callDataWeb\")'" "$json_file"; then
                echo "  ❌ Missing callDataWeb configuration"
                ((errors++))
            else
                # Check for required web fields
                if ! safe_check "jq -e '.versions[0].config.callDataWeb.schema | has(\"queryTemplate\")'" "$json_file"; then
                    echo "  ❌ Missing queryTemplate in callDataWeb"
                    ((errors++))
                fi
                if ! safe_check "jq -e '.versions[0].config.callDataWeb.schema | has(\"searchEngine\")'" "$json_file"; then
                    echo "  ❌ Missing searchEngine in callDataWeb"
                    ((errors++))
                fi
            fi
            ;;
        "RoutineInternalAction")
            # Check for callDataAction
            if ! safe_check "jq -e '.versions[0].config | has(\"callDataAction\")'" "$json_file"; then
                echo "  ❌ Missing callDataAction configuration"
                ((errors++))
            else
                # Check for required action fields
                if ! safe_check "jq -e '.versions[0].config.callDataAction.schema | has(\"toolName\")'" "$json_file"; then
                    echo "  ❌ Missing toolName in callDataAction"
                    ((errors++))
                fi
                if ! safe_check "jq -e '.versions[0].config.callDataAction.schema | has(\"inputTemplate\")'" "$json_file"; then
                    echo "  ❌ Missing inputTemplate in callDataAction"
                    ((errors++))
                fi
            fi
            ;;
        "RoutineData")
            # Check for callDataData
            if ! safe_check "jq -e '.versions[0].config | has(\"callDataData\")'" "$json_file"; then
                echo "  ❌ Missing callDataData configuration"
                ((errors++))
            else
                # Check for required data fields
                if ! safe_check "jq -e '.versions[0].config.callDataData.schema | has(\"operation\")'" "$json_file"; then
                    echo "  ❌ Missing operation in callDataData"
                    ((errors++))
                fi
                if ! safe_check "jq -e '.versions[0].config.callDataData.schema | has(\"inputFormat\")'" "$json_file"; then
                    echo "  ❌ Missing inputFormat in callDataData"
                    ((errors++))
                fi
            fi
            ;;
        "RoutineMultiStep")
            # Check for graph configuration
            if ! safe_check "jq -e '.versions[0].config | has(\"graph\")'" "$json_file"; then
                echo "  ❌ Missing graph configuration for multi-step routine"
                ((errors++))
            else
                # Check for graph type
                local graph_type
                graph_type=$(jq -r '.versions[0].config.graph.__type // ""' "$json_file" 2>/dev/null || echo "")
                if [[ "$graph_type" != "Sequential" ]] && [[ "$graph_type" != "BPMN-2.0" ]]; then
                    echo "  ❌ Invalid graph type: $graph_type (should be Sequential or BPMN-2.0)"
                    ((errors++))
                fi
            fi
            ;;
        *)
            echo "  ⚠️  Unknown resource sub type: $resource_sub_type"
            ((errors++))
            ;;
    esac
    
    return $errors
}

# Function: Validate routine JSON structure
validate_routine_json() {
    local json_file="$1"
    local errors=0
    
    echo "🔍 Validating routine JSON: $(basename "$json_file")"
    
    # Check if file exists and is readable
    if [[ ! -f "$json_file" ]]; then
        echo "  ❌ File does not exist: $json_file"
        return 1
    fi
    
    if [[ ! -r "$json_file" ]]; then
        echo "  ❌ File is not readable: $json_file"
        return 1
    fi
    
    # Check if file is valid JSON
    if ! safe_check "jq empty" "$json_file"; then
        echo "  ❌ Invalid JSON format"
        return 1
    fi
    
    # Check required root fields
    local required_fields=("id" "publicId" "resourceType" "versions")
    for field in "${required_fields[@]}"; do
        if ! safe_check "jq -e 'has(\"$field\")'" "$json_file"; then
            echo "  ❌ Missing required field: $field"
            ((errors++))
        fi
    done
    
    # Validate ID format (19-digit snowflake)
    local id_value
    id_value=$(jq -r '.id // ""' "$json_file" 2>/dev/null || echo "")
    if [[ ! "$id_value" =~ ^[0-9]{19}$ ]]; then
        echo "  ❌ Invalid ID format (should be 19-digit numeric string): $id_value"
        ((errors++))
    fi
    
    # Validate publicId format (10-12 character alphanumeric)
    local public_id
    public_id=$(jq -r '.publicId // ""' "$json_file" 2>/dev/null || echo "")
    if [[ ! "$public_id" =~ ^[a-z0-9-]{10,12}$ ]]; then
        echo "  ❌ Invalid publicId format (should be 10-12 char lowercase alphanumeric): $public_id"
        ((errors++))
    fi
    
    # Validate resourceType
    local resource_type
    resource_type=$(jq -r '.resourceType // ""' "$json_file" 2>/dev/null || echo "")
    if [[ "$resource_type" != "Routine" ]]; then
        echo "  ❌ Invalid resourceType (should be 'Routine'): $resource_type"
        ((errors++))
    fi
    
    # Validate versions array
    local versions_count
    versions_count=$(jq '.versions | length' "$json_file" 2>/dev/null || echo 0)
    if [[ $versions_count -eq 0 ]]; then
        echo "  ❌ Empty versions array"
        ((errors++))
    else
        # Validate first version
        local version_fields=("id" "publicId" "versionLabel" "isComplete" "resourceSubType" "config" "translations")
        for field in "${version_fields[@]}"; do
            if ! safe_check "jq -e '.versions[0] | has(\"$field\")'" "$json_file"; then
                echo "  ❌ Missing version field: $field"
                ((errors++))
            fi
        done
        
        # Validate version ID format
        local version_id
        version_id=$(jq -r '.versions[0].id // ""' "$json_file" 2>/dev/null || echo "")
        if [[ ! "$version_id" =~ ^[0-9]{19}$ ]]; then
            echo "  ❌ Invalid version ID format: $version_id"
            ((errors++))
        fi
        
        # Validate config structure
        if ! safe_check "jq -e '.versions[0].config | has(\"__version\")'" "$json_file"; then
            echo "  ❌ Missing config.__version field"
            ((errors++))
        fi
        
        # Validate translations array
        local translations_count
        translations_count=$(jq '.versions[0].translations | length' "$json_file" 2>/dev/null || echo 0)
        if [[ $translations_count -eq 0 ]]; then
            echo "  ❌ Empty translations array"
            ((errors++))
        else
            # Check first translation
            local translation_fields=("id" "language" "name")
            for field in "${translation_fields[@]}"; do
                if ! safe_check "jq -e '.versions[0].translations[0] | has(\"$field\")'" "$json_file"; then
                    echo "  ❌ Missing translation field: $field"
                    ((errors++))
                fi
            done
        fi
    fi
    
    # Validate tags field (should be empty array)
    local tags_check
    tags_check=$(jq '.tags == []' "$json_file" 2>/dev/null || echo false)
    if [[ "$tags_check" != "true" ]]; then
        echo "  ❌ Tags field should be empty array []"
        ((errors++))
    fi
    
    # Validate routine-specific configuration based on resourceSubType
    if [[ $errors -eq 0 ]]; then
        local resource_sub_type
        resource_sub_type=$(jq -r '.versions[0].resourceSubType // ""' "$json_file" 2>/dev/null || echo "")
        
        if [[ -n "$resource_sub_type" ]]; then
            echo ""
            if ! validate_single_step_config "$json_file" "$resource_sub_type"; then
                local config_errors=$?
                ((errors += config_errors))
            fi
        else
            echo "  ⚠️  No resourceSubType found, skipping configuration validation"
        fi
    fi
    
    # Check for form configurations (required for most routine types)
    if [[ $errors -eq 0 ]]; then
        echo ""
        echo "📋 Validating form configurations..."
        
        # Check formInput
        if ! safe_check "jq -e '.versions[0].config | has(\"formInput\")'" "$json_file"; then
            echo "  ⚠️  Missing formInput configuration (recommended for most routines)"
        else
            # Validate formInput structure
            if ! safe_check "jq -e '.versions[0].config.formInput.schema | has(\"elements\")'" "$json_file"; then
                echo "  ❌ formInput missing elements array"
                ((errors++))
            fi
        fi
        
        # Check formOutput
        if ! safe_check "jq -e '.versions[0].config | has(\"formOutput\")'" "$json_file"; then
            echo "  ⚠️  Missing formOutput configuration (recommended for most routines)"
        else
            # Validate formOutput structure
            if ! safe_check "jq -e '.versions[0].config.formOutput.schema | has(\"elements\")'" "$json_file"; then
                echo "  ❌ formOutput missing elements array"
                ((errors++))
            fi
        fi
    fi
    
    # Final validation summary
    if [[ $errors -eq 0 ]]; then
        echo ""
        echo "  ✅ All validations passed"
        return 0
    else
        echo ""
        echo "  ❌ Found $errors validation errors"
        return 1
    fi
}

# Main execution
if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <routine.json> [routine2.json ...]"
    echo "Validates routine JSON files against expected schema"
    exit 1
fi

total_files=0
failed_files=0

for json_file in "$@"; do
    ((total_files++))
    if ! validate_routine_json "$json_file"; then
        ((failed_files++))
    fi
    echo ""
done

echo "📊 Validation Summary:"
echo "  Total files: $total_files"
echo "  Passed: $((total_files - failed_files))"
echo "  Failed: $failed_files"

if [[ $failed_files -gt 0 ]]; then
    exit 1
else
    echo "✅ All files passed validation"
    exit 0
fi