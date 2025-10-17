#!/usr/bin/env bash
# Java Code Extractor Library
# Extracts classes, methods, interfaces, and metadata from Java files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract methods from Java file
# 
# Finds method definitions and their visibility
#
# Arguments:
#   $1 - Path to Java file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with method information
#######################################
extractor::lib::java::extract_methods() {
    local file="$1"
    local context="${2:-java}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting Java methods from: $file" >&2
    
    # Find method definitions (public, private, protected, or package-private)
    local methods=$(grep -E "^[[:space:]]*(public|private|protected|static)[[:space:]].*[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*\(" "$file" 2>/dev/null | \
        sed -E 's/.*[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*\(.*/\1/' || echo "")
    
    if [[ -n "$methods" ]]; then
        while IFS= read -r method_name; do
            [[ -z "$method_name" ]] && continue
            
            # Skip common non-method matches
            [[ "$method_name" =~ ^(class|interface|enum|import|package)$ ]] && continue
            
            # Get method signature and visibility
            local method_line=$(grep -E "[[:space:]]+$method_name[[:space:]]*\(" "$file" | head -1)
            local visibility="package"
            local is_static="false"
            local return_type=""
            
            if [[ "$method_line" =~ public ]]; then
                visibility="public"
            elif [[ "$method_line" =~ private ]]; then
                visibility="private"
            elif [[ "$method_line" =~ protected ]]; then
                visibility="protected"
            fi
            
            [[ "$method_line" =~ static ]] && is_static="true"
            
            # Extract return type (rough approximation)
            if [[ "$method_line" =~ [[:space:]]([A-Za-z_][A-Za-z0-9_<>]*)[[:space:]]+$method_name ]]; then
                return_type="${BASH_REMATCH[1]}"
            fi
            
            # Look for JavaDoc comments
            local description=""
            local line_num=$(grep -n "$method_name.*(" "$file" | head -1 | cut -d: -f1)
            
            if [[ -n "$line_num" ]]; then
                local desc_line=$((line_num - 1))
                while [[ $desc_line -gt 0 ]]; do
                    local line_content=$(sed -n "${desc_line}p" "$file")
                    if [[ "$line_content" =~ ^[[:space:]]*\*[[:space:]]*(.+)$ ]]; then
                        local comment="${BASH_REMATCH[1]}"
                        if [[ -z "$description" ]]; then
                            description="$comment"
                        else
                            description="$comment $description"
                        fi
                        ((desc_line--))
                    elif [[ "$line_content" =~ ^[[:space:]]*//[[:space:]]*(.+)$ ]]; then
                        description="${BASH_REMATCH[1]}"
                        break
                    else
                        break
                    fi
                done
            fi
            
            # Determine method type
            local method_type="method"
            if [[ "$method_name" == "main" ]]; then
                method_type="main"
            elif [[ "$method_name" =~ ^get[A-Z] ]]; then
                method_type="getter"
            elif [[ "$method_name" =~ ^set[A-Z] ]]; then
                method_type="setter"
            elif [[ "$method_name" =~ ^is[A-Z] ]]; then
                method_type="getter"
            fi
            
            # Build content
            local content="Method: $method_name | Context: $context | Parent: $parent_name"
            content="$content | Visibility: $visibility | Type: $method_type"
            [[ -n "$return_type" ]] && content="$content | Returns: $return_type"
            [[ "$is_static" == "true" ]] && content="$content | Static: true"
            [[ -n "$description" ]] && content="$content | Description: $description"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg method_name "$method_name" \
                --arg method_type "$method_type" \
                --arg visibility "$visibility" \
                --arg return_type "$return_type" \
                --arg is_static "$is_static" \
                --arg description "$description" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "java",
                        method_name: $method_name,
                        method_type: $method_type,
                        visibility: $visibility,
                        return_type: $return_type,
                        is_static: ($is_static == "true"),
                        description: $description,
                        content_type: "code_function",
                        extraction_method: "java_parser"
                    }
                }' | jq -c
        done <<< "$methods"
    fi
}

#######################################
# Extract classes and interfaces from Java file
# 
# Finds class and interface definitions
#
# Arguments:
#   $1 - Path to Java file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with class information
#######################################
extractor::lib::java::extract_classes() {
    local file="$1"
    local context="${2:-java}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find class definitions
    local classes=$(grep -E "^[[:space:]]*(public[[:space:]]+)?(class|interface|enum)[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*[[:space:]](class|interface|enum)[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\2:\1/' || echo "")
    
    if [[ -n "$classes" ]]; then
        while IFS= read -r class_info; do
            [[ -z "$class_info" ]] && continue
            
            local class_name="${class_info%:*}"
            local class_type="${class_info#*:}"
            
            # Check visibility
            local visibility="package"
            if grep -q "public.*$class_type.*$class_name" "$file" 2>/dev/null; then
                visibility="public"
            fi
            
            # Count methods in class (rough approximation)
            local method_count=0
            local in_class=false
            local brace_count=0
            
            while IFS= read -r line; do
                if [[ "$line" =~ $class_type[[:space:]]+$class_name ]]; then
                    in_class=true
                fi
                
                if [[ "$in_class" == "true" ]]; then
                    # Count braces to track class scope
                    local open_braces="${line//[^\{]/}"
                    local close_braces="${line//[^\}]/}"
                    brace_count=$((brace_count + ${#open_braces} - ${#close_braces}))
                    
                    # Count method definitions
                    if [[ "$line" =~ ^[[:space:]]*(public|private|protected)[[:space:]].*[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*\( ]]; then
                        ((method_count++))
                    fi
                    
                    if [[ $brace_count -eq 0 && "$line" =~ \} ]]; then
                        break
                    fi
                fi
            done < "$file"
            
            # Extract extends/implements information
            local inheritance=""
            local class_line=$(grep "$class_type $class_name" "$file" | head -1)
            if [[ "$class_line" =~ extends[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
                inheritance="extends ${BASH_REMATCH[1]}"
            fi
            if [[ "$class_line" =~ implements[[:space:]]+([^{]+) ]]; then
                local interfaces="${BASH_REMATCH[1]}"
                interfaces=$(echo "$interfaces" | sed 's/,/ /g' | xargs)
                inheritance="$inheritance implements $interfaces"
            fi
            
            # Build content
            local content="$class_type: $class_name | Context: $context | Parent: $parent_name"
            content="$content | Visibility: $visibility | Methods: $method_count"
            [[ -n "$inheritance" ]] && content="$content | $inheritance"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg class_name "$class_name" \
                --arg class_type "$class_type" \
                --arg visibility "$visibility" \
                --arg method_count "$method_count" \
                --arg inheritance "$inheritance" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "java",
                        class_name: $class_name,
                        class_type: $class_type,
                        visibility: $visibility,
                        method_count: ($method_count | tonumber),
                        inheritance: $inheritance,
                        content_type: "code_class",
                        extraction_method: "java_parser"
                    }
                }' | jq -c
        done <<< "$classes"
    fi
}

#######################################
# Extract Maven pom.xml information
# 
# Gets Maven project metadata and dependencies
#
# Arguments:
#   $1 - Path to pom.xml file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with Maven information
#######################################
extractor::lib::java::extract_maven_pom() {
    local file="$1"
    local context="${2:-java}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract basic project info
    local group_id=$(grep -A 1 "<groupId>" "$file" | grep -v "parent" | head -1 | sed 's/<[^>]*>//g' | xargs)
    local artifact_id=$(grep "<artifactId>" "$file" | head -1 | sed 's/<[^>]*>//g' | xargs)
    local version=$(grep -A 1 "<version>" "$file" | grep -v "parent" | head -1 | sed 's/<[^>]*>//g' | xargs)
    local packaging=$(grep "<packaging>" "$file" | head -1 | sed 's/<[^>]*>//g' | xargs)
    
    # Count dependencies
    local dep_count=$(grep -c "<dependency>" "$file" 2>/dev/null || echo "0")
    
    # Get Java version
    local java_version=$(grep -E "(maven.compiler.source|java.version)" "$file" | head -1 | sed 's/<[^>]*>//g' | xargs)
    
    # Build content
    local content="Maven: $artifact_id | Context: $context | Parent: $parent_name"
    [[ -n "$group_id" ]] && content="$content | Group: $group_id"
    [[ -n "$version" ]] && content="$content | Version: $version"
    [[ -n "$packaging" ]] && content="$content | Packaging: $packaging"
    [[ -n "$java_version" ]] && content="$content | Java: $java_version"
    content="$content | Dependencies: $dep_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg group_id "$group_id" \
        --arg artifact_id "$artifact_id" \
        --arg version "$version" \
        --arg packaging "$packaging" \
        --arg java_version "$java_version" \
        --arg dependency_count "$dep_count" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "java",
                group_id: $group_id,
                artifact_id: $artifact_id,
                version: $version,
                packaging: $packaging,
                java_version: $java_version,
                dependency_count: ($dependency_count | tonumber),
                build_tool: "maven",
                content_type: "build_config",
                extraction_method: "maven_pom_parser"
            }
        }' | jq -c
}

#######################################
# Extract Gradle build information
# 
# Gets Gradle project metadata
#
# Arguments:
#   $1 - Path to build.gradle file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with Gradle information
#######################################
extractor::lib::java::extract_gradle_build() {
    local file="$1"
    local context="${2:-java}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract basic info
    local version=$(grep "version[[:space:]]*=" "$file" | sed "s/.*=[[:space:]]*['\"]//; s/['\"].*//")
    local group=$(grep "group[[:space:]]*=" "$file" | sed "s/.*=[[:space:]]*['\"]//; s/['\"].*//")
    
    # Count dependencies
    local dep_count=$(grep -c "implementation\|compile\|api" "$file" 2>/dev/null || echo "0")
    
    # Get Java version
    local java_version=$(grep -E "(sourceCompatibility|targetCompatibility|JavaVersion)" "$file" | head -1 | sed "s/.*['\"]//; s/['\"].*//")
    
    # Check for plugins
    local plugins=$(grep -E "apply plugin|id[[:space:]]*'" "$file" | wc -l)
    
    # Build content
    local content="Gradle: $(basename "$(dirname "$file")") | Context: $context | Parent: $parent_name"
    [[ -n "$group" ]] && content="$content | Group: $group"
    [[ -n "$version" ]] && content="$content | Version: $version"
    [[ -n "$java_version" ]] && content="$content | Java: $java_version"
    content="$content | Dependencies: $dep_count | Plugins: $plugins"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg group "$group" \
        --arg version "$version" \
        --arg java_version "$java_version" \
        --arg dependency_count "$dep_count" \
        --arg plugin_count "$plugins" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "java",
                group_id: $group,
                version: $version,
                java_version: $java_version,
                dependency_count: ($dependency_count | tonumber),
                plugin_count: ($plugin_count | tonumber),
                build_tool: "gradle",
                content_type: "build_config",
                extraction_method: "gradle_build_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from Java files
# 
# Main entry point that extracts classes, methods, and build info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all Java information
#######################################
extractor::lib::java::extract_all() {
    local path="$1"
    local context="${2:-java}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.java)
                extractor::lib::java::extract_methods "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::java::extract_classes "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            pom.xml)
                extractor::lib::java::extract_maven_pom "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            build.gradle)
                extractor::lib::java::extract_gradle_build "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::java::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.java" -o -name "pom.xml" -o -name "build.gradle" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::java::extract_methods
export -f extractor::lib::java::extract_classes
export -f extractor::lib::java::extract_maven_pom
export -f extractor::lib::java::extract_gradle_build
export -f extractor::lib::java::extract_all