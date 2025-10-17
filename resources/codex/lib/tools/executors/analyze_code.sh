#!/usr/bin/env bash
################################################################################
# Analyze Code Tool Executor
# 
# Performs static code analysis including syntax, style, complexity, and security
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Analyze Code Tool Implementation
################################################################################

#######################################
# Execute analyze_code tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context (sandbox, local)
# Returns:
#   Analysis result (JSON)
#######################################
tool_analyze_code::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    log::debug "Executing analyze_code tool in $context context"
    
    # Extract arguments
    local code language analysis_type
    code=$(echo "$arguments" | jq -r '.code // ""')
    language=$(echo "$arguments" | jq -r '.language // ""')
    analysis_type=$(echo "$arguments" | jq -r '.analysis_type // "syntax"')
    
    # Validate arguments
    if [[ -z "$code" ]]; then
        echo '{"success": false, "error": "Missing required parameter: code"}'
        return 1
    fi
    
    # Auto-detect language if not provided
    if [[ -z "$language" ]]; then
        language=$(tool_analyze_code::detect_language "$code")
        log::debug "Auto-detected language: $language"
    fi
    
    # Perform analysis based on type
    case "$analysis_type" in
        syntax)
            tool_analyze_code::analyze_syntax "$code" "$language"
            ;;
        style)
            tool_analyze_code::analyze_style "$code" "$language"
            ;;
        complexity)
            tool_analyze_code::analyze_complexity "$code" "$language"
            ;;
        security)
            tool_analyze_code::analyze_security "$code" "$language"
            ;;
        all)
            tool_analyze_code::analyze_comprehensive "$code" "$language"
            ;;
        *)
            echo '{"success": false, "error": "Invalid analysis_type. Valid options: syntax, style, complexity, security, all"}'
            return 1
            ;;
    esac
}

#######################################
# Detect programming language from code
# Arguments:
#   $1 - Code content
# Returns:
#   Language name
#######################################
tool_analyze_code::detect_language() {
    local code="$1"
    
    # Python indicators
    if [[ "$code" =~ (def |import |from .* import|if __name__|print\() ]]; then
        echo "python"
        return
    fi
    
    # JavaScript/TypeScript indicators
    if [[ "$code" =~ (function |const |let |var |=\>|console\.log|require\(|import.*from) ]]; then
        if [[ "$code" =~ (interface |type |as |<.*>) ]]; then
            echo "typescript"
        else
            echo "javascript"
        fi
        return
    fi
    
    # Go indicators
    if [[ "$code" =~ (package |func |import |var |:=|fmt\.Print) ]]; then
        echo "go"
        return
    fi
    
    # Java indicators
    if [[ "$code" =~ (public class|public static void|import java|System\.out) ]]; then
        echo "java"
        return
    fi
    
    # C/C++ indicators
    if [[ "$code" =~ (#include|int main|printf\(|cout|std::) ]]; then
        if [[ "$code" =~ (cout|std::|class.*{|template) ]]; then
            echo "cpp"
        else
            echo "c"
        fi
        return
    fi
    
    # Rust indicators
    if [[ "$code" =~ (fn |let |pub |use |println!|cargo) ]]; then
        echo "rust"
        return
    fi
    
    # Shell/Bash indicators
    if [[ "$code" =~ (#!/bin/bash|#!/bin/sh|\$\{.*\}|if \[|echo ) ]]; then
        echo "bash"
        return
    fi
    
    # Default to unknown
    echo "unknown"
}

#######################################
# Analyze code syntax
# Arguments:
#   $1 - Code content
#   $2 - Language
# Returns:
#   Syntax analysis JSON
#######################################
tool_analyze_code::analyze_syntax() {
    local code="$1"
    local language="$2"
    
    local errors=()
    local warnings=()
    local suggestions=()
    
    case "$language" in
        python)
            tool_analyze_code::analyze_python_syntax "$code" errors warnings suggestions
            ;;
        javascript|typescript)
            tool_analyze_code::analyze_js_syntax "$code" errors warnings suggestions
            ;;
        go)
            tool_analyze_code::analyze_go_syntax "$code" errors warnings suggestions
            ;;
        bash)
            tool_analyze_code::analyze_bash_syntax "$code" errors warnings suggestions
            ;;
        *)
            tool_analyze_code::analyze_generic_syntax "$code" errors warnings suggestions
            ;;
    esac
    
    # Build result
    local errors_json warnings_json suggestions_json
    errors_json=$(printf '%s\n' "${errors[@]}" | jq -R . | jq -s .)
    warnings_json=$(printf '%s\n' "${warnings[@]}" | jq -R . | jq -s .)
    suggestions_json=$(printf '%s\n' "${suggestions[@]}" | jq -R . | jq -s .)
    
    cat << EOF
{
  "success": true,
  "analysis_type": "syntax",
  "language": "$language",
  "errors": $errors_json,
  "warnings": $warnings_json,
  "suggestions": $suggestions_json,
  "error_count": ${#errors[@]},
  "warning_count": ${#warnings[@]},
  "suggestion_count": ${#suggestions[@]}
}
EOF
}

#######################################
# Analyze Python syntax
# Arguments:
#   $1 - Code content
#   $2 - Errors array reference
#   $3 - Warnings array reference  
#   $4 - Suggestions array reference
#######################################
tool_analyze_code::analyze_python_syntax() {
    local code="$1"
    local -n errors_ref=$2
    local -n warnings_ref=$3
    local -n suggestions_ref=$4
    
    # Check for common Python syntax issues
    
    # Indentation issues
    if [[ "$code" =~ ^[[:space:]]*[[:alpha:]] ]] && [[ ! "$code" =~ ^[[:space:]]{4}|^[[:space:]]{8} ]]; then
        errors_ref+=("Inconsistent indentation detected - Python uses 4 spaces per level")
    fi
    
    # Missing colons
    local line_num=1
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*(if|elif|else|for|while|def|class|try|except|finally|with)[[:space:]] ]] && [[ ! "$line" =~ :$ ]]; then
            errors_ref+=("Line $line_num: Missing colon after control statement")
        fi
        ((line_num++))
    done <<< "$code"
    
    # Unused imports (simple check)
    while IFS= read -r line; do
        if [[ "$line" =~ ^import[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
            local imported="${BASH_REMATCH[1]}"
            if [[ ! "$code" =~ $imported\. ]]; then
                warnings_ref+=("Potentially unused import: $imported")
            fi
        fi
    done <<< "$code"
    
    # Suggestions for best practices
    if [[ "$code" =~ print\( ]] && [[ ! "$code" =~ __name__.*==.*__main__ ]]; then
        suggestions_ref+=("Consider using logging instead of print() for non-debug output")
    fi
    
    if [[ "$code" =~ except: ]] && [[ ! "$code" =~ except[[:space:]]+[A-Z] ]]; then
        suggestions_ref+=("Consider catching specific exceptions instead of bare 'except:'")
    fi
}

#######################################
# Analyze JavaScript/TypeScript syntax
# Arguments:
#   $1 - Code content
#   $2 - Errors array reference
#   $3 - Warnings array reference
#   $4 - Suggestions array reference
#######################################
tool_analyze_code::analyze_js_syntax() {
    local code="$1"
    local -n errors_ref=$2
    local -n warnings_ref=$3
    local -n suggestions_ref=$4
    
    # Check for common JS issues
    
    # Missing semicolons (if using semicolon style)
    local line_num=1
    while IFS= read -r line; do
        if [[ "$line" =~ (var|let|const|return)[[:space:]] ]] && [[ ! "$line" =~ ;$ ]] && [[ ! "$line" =~ \{$ ]]; then
            warnings_ref+=("Line $line_num: Missing semicolon")
        fi
        ((line_num++))
    done <<< "$code"
    
    # == vs === usage
    if [[ "$code" =~ [^!]==[^=] ]]; then
        suggestions_ref+=("Consider using '===' instead of '==' for strict equality")
    fi
    
    # var vs let/const
    if [[ "$code" =~ var[[:space:]] ]]; then
        suggestions_ref+=("Consider using 'let' or 'const' instead of 'var' for better scoping")
    fi
    
    # Missing function declarations
    local line_num=1
    while IFS= read -r line; do
        if [[ "$line" =~ function[[:space:]]*\([^)]*\)[[:space:]]*\{ ]] && [[ ! "$line" =~ function[[:space:]]+[a-zA-Z_] ]]; then
            warnings_ref+=("Line $line_num: Anonymous function - consider naming for debugging")
        fi
        ((line_num++))
    done <<< "$code"
}

#######################################
# Analyze Go syntax
# Arguments:
#   $1 - Code content
#   $2 - Errors array reference
#   $3 - Warnings array reference
#   $4 - Suggestions array reference
#######################################
tool_analyze_code::analyze_go_syntax() {
    local code="$1"
    local -n errors_ref=$2
    local -n warnings_ref=$3
    local -n suggestions_ref=$4
    
    # Check for Go-specific issues
    
    # Missing package declaration
    if [[ ! "$code" =~ ^package[[:space:]] ]]; then
        errors_ref+=("Missing package declaration at top of file")
    fi
    
    # Unused variables (simple check)
    if [[ "$code" =~ :=[[:space:]]*[a-zA-Z] ]] && [[ "$code" =~ var[[:space:]]+[a-zA-Z_] ]]; then
        warnings_ref+=("Check for unused variables - Go compiler will error on unused vars")
    fi
    
    # Error handling
    if [[ "$code" =~ err[[:space:]]*:= ]] && [[ ! "$code" =~ if.*err.*!= ]]; then
        suggestions_ref+=("Consider checking error return values with 'if err != nil'")
    fi
    
    # Suggest go fmt
    if [[ "$code" =~ ^[[:space:]]+[{}] ]] || [[ "$code" =~ \}[[:space:]]*$ ]]; then
        suggestions_ref+=("Run 'go fmt' to ensure consistent formatting")
    fi
}

#######################################
# Analyze Bash syntax
# Arguments:
#   $1 - Code content
#   $2 - Errors array reference
#   $3 - Warnings array reference
#   $4 - Suggestions array reference
#######################################
tool_analyze_code::analyze_bash_syntax() {
    local code="$1"
    local -n errors_ref=$2
    local -n warnings_ref=$3
    local -n suggestions_ref=$4
    
    # Check for bash-specific issues
    
    # Missing shebang
    if [[ ! "$code" =~ ^#!/ ]]; then
        warnings_ref+=("Missing shebang line (#!/bin/bash)")
    fi
    
    # Unquoted variables
    if [[ "$code" =~ \$[a-zA-Z_][a-zA-Z0-9_]*[^"] ]]; then
        suggestions_ref+=("Consider quoting variables to prevent word splitting: \"\$var\"")
    fi
    
    # Use of [[ instead of [
    if [[ "$code" =~ if[[:space:]]+\[ ]] && [[ ! "$code" =~ if[[:space:]]+\[\[ ]]; then
        suggestions_ref+=("Consider using [[ ]] instead of [ ] for better condition testing")
    fi
    
    # Set -e usage
    if [[ ! "$code" =~ set[[:space:]]+-e ]]; then
        suggestions_ref+=("Consider 'set -e' to exit on errors")
    fi
    
    # Function definition style
    if [[ "$code" =~ function[[:space:]]+[a-zA-Z_] ]] && [[ ! "$code" =~ \(\)[[:space:]]*\{ ]]; then
        suggestions_ref+=("Use 'function_name() {' style for function definitions")
    fi
}

#######################################
# Analyze generic syntax (fallback)
# Arguments:
#   $1 - Code content
#   $2 - Errors array reference
#   $3 - Warnings array reference
#   $4 - Suggestions array reference
#######################################
tool_analyze_code::analyze_generic_syntax() {
    local code="$1"
    local -n errors_ref=$2
    local -n warnings_ref=$3
    local -n suggestions_ref=$4
    
    # Generic checks that apply to most languages
    
    # Check for extremely long lines
    local line_num=1
    while IFS= read -r line; do
        if [[ ${#line} -gt 120 ]]; then
            warnings_ref+=("Line $line_num: Line length exceeds 120 characters (${#line})")
        fi
        ((line_num++))
    done <<< "$code"
    
    # Check for mixed tabs and spaces
    if [[ "$code" =~ $'\t' ]] && [[ "$code" =~ ^[[:space:]]*[^[:space:]\t] ]]; then
        warnings_ref+=("Mixed tabs and spaces detected")
    fi
    
    # Basic bracket matching
    local open_braces=$(echo "$code" | grep -o '{' | wc -l)
    local close_braces=$(echo "$code" | grep -o '}' | wc -l)
    if [[ $open_braces -ne $close_braces ]]; then
        errors_ref+=("Mismatched braces: $open_braces opening, $close_braces closing")
    fi
    
    local open_parens=$(echo "$code" | grep -o '(' | wc -l)
    local close_parens=$(echo "$code" | grep -o ')' | wc -l)
    if [[ $open_parens -ne $close_parens ]]; then
        errors_ref+=("Mismatched parentheses: $open_parens opening, $close_parens closing")
    fi
}

#######################################
# Analyze code style
# Arguments:
#   $1 - Code content
#   $2 - Language
# Returns:
#   Style analysis JSON
#######################################
tool_analyze_code::analyze_style() {
    local code="$1"
    local language="$2"
    
    local issues=()
    local score=100
    
    # Generic style checks
    local line_num=1
    local total_lines=0
    local long_lines=0
    local empty_lines=0
    
    while IFS= read -r line; do
        ((total_lines++))
        
        # Line length
        if [[ ${#line} -gt 100 ]]; then
            ((long_lines++))
            issues+=("Line $line_num: Line too long (${#line} > 100 characters)")
            ((score -= 2))
        fi
        
        # Empty lines
        if [[ "$line" =~ ^[[:space:]]*$ ]]; then
            ((empty_lines++))
        fi
        
        # Trailing whitespace
        if [[ "$line" =~ [[:space:]]$ ]]; then
            issues+=("Line $line_num: Trailing whitespace")
            ((score -= 1))
        fi
        
        ((line_num++))
    done <<< "$code"
    
    # Language-specific style checks
    case "$language" in
        python)
            tool_analyze_code::check_python_style "$code" issues score
            ;;
        javascript|typescript)
            tool_analyze_code::check_js_style "$code" issues score
            ;;
    esac
    
    # Calculate metrics
    local avg_line_length
    if [[ $total_lines -gt 0 ]]; then
        avg_line_length=$(echo "scale=1; ${#code} / $total_lines" | bc -l 2>/dev/null || echo "0")
    else
        avg_line_length="0"
    fi
    
    local issues_json
    issues_json=$(printf '%s\n' "${issues[@]}" | jq -R . | jq -s .)
    
    cat << EOF
{
  "success": true,
  "analysis_type": "style",
  "language": "$language",
  "style_score": $score,
  "issues": $issues_json,
  "metrics": {
    "total_lines": $total_lines,
    "long_lines": $long_lines,
    "empty_lines": $empty_lines,
    "average_line_length": $avg_line_length,
    "issue_count": ${#issues[@]}
  }
}
EOF
}

#######################################
# Analyze code complexity
# Arguments:
#   $1 - Code content
#   $2 - Language
# Returns:
#   Complexity analysis JSON
#######################################
tool_analyze_code::analyze_complexity() {
    local code="$1"
    local language="$2"
    
    # Calculate cyclomatic complexity (simplified)
    local complexity=1
    local functions=0
    local loops=0
    local conditions=0
    local nesting_level=0
    local max_nesting=0
    
    # Count complexity indicators
    complexity=$((complexity + $(echo "$code" | grep -c "if\|elif\|else\|for\|while\|case\|switch\|catch\|except")))
    loops=$(echo "$code" | grep -c "for\|while")
    conditions=$(echo "$code" | grep -c "if\|elif\|switch\|case")
    
    # Count functions
    case "$language" in
        python)
            functions=$(echo "$code" | grep -c "^[[:space:]]*def ")
            ;;
        javascript|typescript)
            functions=$(echo "$code" | grep -c "function\|=>[[:space:]]*{")
            ;;
        go)
            functions=$(echo "$code" | grep -c "^func ")
            ;;
        *)
            functions=$(echo "$code" | grep -c "function\|def\|func")
            ;;
    esac
    
    # Estimate nesting by brace/indentation depth
    while IFS= read -r line; do
        local line_nesting=0
        
        # Count opening braces
        line_nesting=$((line_nesting + $(echo "$line" | grep -o '{' | wc -l)))
        
        # Or count indentation depth for Python-like languages
        if [[ "$language" == "python" ]]; then
            local leading_spaces=${line%%[^[:space:]]*}
            line_nesting=$((${#leading_spaces} / 4))
        fi
        
        if [[ $line_nesting -gt $max_nesting ]]; then
            max_nesting=$line_nesting
        fi
    done <<< "$code"
    
    # Calculate overall complexity score
    local complexity_score=$((complexity + loops + (max_nesting * 2)))
    
    # Determine complexity rating
    local rating
    if [[ $complexity_score -lt 10 ]]; then
        rating="low"
    elif [[ $complexity_score -lt 20 ]]; then
        rating="medium"
    elif [[ $complexity_score -lt 40 ]]; then
        rating="high"
    else
        rating="very_high"
    fi
    
    cat << EOF
{
  "success": true,
  "analysis_type": "complexity",
  "language": "$language",
  "complexity_score": $complexity_score,
  "complexity_rating": "$rating",
  "metrics": {
    "cyclomatic_complexity": $complexity,
    "function_count": $functions,
    "loop_count": $loops,
    "condition_count": $conditions,
    "max_nesting_depth": $max_nesting,
    "lines_of_code": $(echo "$code" | wc -l)
  },
  "recommendations": $(tool_analyze_code::get_complexity_recommendations "$rating" "$complexity_score")
}
EOF
}

#######################################
# Get complexity recommendations
# Arguments:
#   $1 - Complexity rating
#   $2 - Complexity score
# Returns:
#   JSON array of recommendations
#######################################
tool_analyze_code::get_complexity_recommendations() {
    local rating="$1"
    local score="$2"
    
    case "$rating" in
        low)
            echo '["Code complexity is well-managed", "Consider adding comments for clarity"]'
            ;;
        medium)
            echo '["Consider breaking down larger functions", "Add unit tests for complex logic", "Document complex algorithms"]'
            ;;
        high)
            echo '["Refactor large functions into smaller ones", "Extract complex logic into separate functions", "Consider design patterns to reduce complexity", "Increase test coverage"]'
            ;;
        very_high)
            echo '["Urgent: Refactor code to reduce complexity", "Break down monolithic functions", "Consider architectural changes", "Implement comprehensive testing", "Add extensive documentation"]'
            ;;
        *)
            echo '["Review code structure"]'
            ;;
    esac
}

#######################################
# Analyze code security
# Arguments:
#   $1 - Code content
#   $2 - Language
# Returns:
#   Security analysis JSON
#######################################
tool_analyze_code::analyze_security() {
    local code="$1"
    local language="$2"
    
    local vulnerabilities=()
    local warnings=()
    local risk_level="low"
    
    # Generic security checks
    
    # Hardcoded passwords/secrets
    if [[ "$code" =~ (password|passwd|pwd)[[:space:]]*=[[:space:]]*[\"'][^\"']{3,} ]]; then
        vulnerabilities+=("Hardcoded password detected")
        risk_level="high"
    fi
    
    if [[ "$code" =~ (api[_-]?key|secret|token)[[:space:]]*=[[:space:]]*[\"'][^\"']{10,} ]]; then
        vulnerabilities+=("Hardcoded API key or token detected")
        risk_level="high"
    fi
    
    # SQL injection patterns
    if [[ "$code" =~ (execute|query|sql)[[:space:]]*\([^)]*\+[^)]* ]]; then
        vulnerabilities+=("Potential SQL injection: string concatenation in query")
        risk_level="high"
    fi
    
    # Command injection patterns
    if [[ "$code" =~ (system|exec|eval|shell)[[:space:]]*\([^)]*\+[^)]* ]]; then
        vulnerabilities+=("Potential command injection: string concatenation in system call")
        risk_level="high"
    fi
    
    # Language-specific security checks
    case "$language" in
        python)
            tool_analyze_code::check_python_security "$code" vulnerabilities warnings risk_level
            ;;
        javascript|typescript)
            tool_analyze_code::check_js_security "$code" vulnerabilities warnings risk_level
            ;;
        bash)
            tool_analyze_code::check_bash_security "$code" vulnerabilities warnings risk_level
            ;;
    esac
    
    # Build results
    local vulnerabilities_json warnings_json
    vulnerabilities_json=$(printf '%s\n' "${vulnerabilities[@]}" | jq -R . | jq -s .)
    warnings_json=$(printf '%s\n' "${warnings[@]}" | jq -R . | jq -s .)
    
    cat << EOF
{
  "success": true,
  "analysis_type": "security",
  "language": "$language",
  "risk_level": "$risk_level",
  "vulnerabilities": $vulnerabilities_json,
  "warnings": $warnings_json,
  "vulnerability_count": ${#vulnerabilities[@]},
  "warning_count": ${#warnings[@]}
}
EOF
}

#######################################
# Comprehensive analysis (all types)
# Arguments:
#   $1 - Code content
#   $2 - Language
# Returns:
#   Complete analysis JSON
#######################################
tool_analyze_code::analyze_comprehensive() {
    local code="$1"
    local language="$2"
    
    # Run all analysis types
    local syntax_result style_result complexity_result security_result
    syntax_result=$(tool_analyze_code::analyze_syntax "$code" "$language")
    style_result=$(tool_analyze_code::analyze_style "$code" "$language")
    complexity_result=$(tool_analyze_code::analyze_complexity "$code" "$language")
    security_result=$(tool_analyze_code::analyze_security "$code" "$language")
    
    # Combine results
    cat << EOF
{
  "success": true,
  "analysis_type": "comprehensive",
  "language": "$language",
  "syntax": $syntax_result,
  "style": $style_result,
  "complexity": $complexity_result,
  "security": $security_result,
  "summary": {
    "total_issues": $(echo "$syntax_result $style_result $security_result" | jq -s 'map(select(.error_count // .vulnerability_count // .issue_count // 0) | (.error_count // .vulnerability_count // .issue_count // 0)) | add // 0'),
    "risk_assessment": $(echo "$security_result" | jq -r '.risk_level'),
    "complexity_rating": $(echo "$complexity_result" | jq -r '.complexity_rating'),
    "style_score": $(echo "$style_result" | jq -r '.style_score // 0')
  }
}
EOF
}

#######################################
# Check Python-specific security issues
# Arguments:
#   $1 - Code content
#   $2 - Vulnerabilities array reference
#   $3 - Warnings array reference
#   $4 - Risk level reference
#######################################
tool_analyze_code::check_python_security() {
    local code="$1"
    local -n vuln_ref=$2
    local -n warn_ref=$3
    local -n risk_ref=$4
    
    # eval() usage
    if [[ "$code" =~ eval\( ]]; then
        vuln_ref+=("Use of eval() can lead to code injection")
        risk_ref="high"
    fi
    
    # exec() usage
    if [[ "$code" =~ exec\( ]]; then
        vuln_ref+=("Use of exec() can lead to code injection")
        risk_ref="high"
    fi
    
    # pickle usage with user input
    if [[ "$code" =~ pickle\.loads? ]]; then
        warn_ref+=("pickle.load() can be dangerous with untrusted input")
        if [[ "$risk_ref" == "low" ]]; then risk_ref="medium"; fi
    fi
    
    # subprocess with shell=True
    if [[ "$code" =~ subprocess.*shell[[:space:]]*=[[:space:]]*True ]]; then
        vuln_ref+=("subprocess with shell=True can lead to command injection")
        risk_ref="high"
    fi
    
    # URL opening without validation
    if [[ "$code" =~ urllib.*urlopen\(|requests\.get\( ]] && [[ ! "$code" =~ https:// ]]; then
        warn_ref+=("HTTP requests should use HTTPS and validate URLs")
        if [[ "$risk_ref" == "low" ]]; then risk_ref="medium"; fi
    fi
}

#######################################
# Check JavaScript security issues
# Arguments:
#   $1 - Code content
#   $2 - Vulnerabilities array reference
#   $3 - Warnings array reference
#   $4 - Risk level reference
#######################################
tool_analyze_code::check_js_security() {
    local code="$1"
    local -n vuln_ref=$2
    local -n warn_ref=$3
    local -n risk_ref=$4
    
    # eval() usage
    if [[ "$code" =~ eval\( ]]; then
        vuln_ref+=("Use of eval() can lead to code injection")
        risk_ref="high"
    fi
    
    # innerHTML without sanitization
    if [[ "$code" =~ innerHTML[[:space:]]*=[[:space:]]*[^;]*\+ ]]; then
        vuln_ref+=("Setting innerHTML with concatenation can lead to XSS")
        risk_ref="high"
    fi
    
    # document.write usage
    if [[ "$code" =~ document\.write\( ]]; then
        warn_ref+=("document.write() can be dangerous and is deprecated")
        if [[ "$risk_ref" == "low" ]]; then risk_ref="medium"; fi
    fi
    
    # Unsafe regular expressions
    if [[ "$code" =~ RegExp\(.*\+.*\) ]]; then
        warn_ref+=("Dynamic regex construction can lead to ReDoS attacks")
        if [[ "$risk_ref" == "low" ]]; then risk_ref="medium"; fi
    fi
}

#######################################
# Check Bash security issues
# Arguments:
#   $1 - Code content
#   $2 - Vulnerabilities array reference
#   $3 - Warnings array reference
#   $4 - Risk level reference
#######################################
tool_analyze_code::check_bash_security() {
    local code="$1"
    local -n vuln_ref=$2
    local -n warn_ref=$3
    local -n risk_ref=$4
    
    # eval usage
    if [[ "$code" =~ eval[[:space:]] ]]; then
        vuln_ref+=("Use of eval in bash can lead to code injection")
        risk_ref="high"
    fi
    
    # Unquoted variables in commands
    if [[ "$code" =~ (rm|cp|mv)[[:space:]]+.*\$[a-zA-Z_] ]] && [[ ! "$code" =~ \"\$[a-zA-Z_] ]]; then
        vuln_ref+=("Unquoted variables in file operations can lead to injection")
        risk_ref="high"
    fi
    
    # curl/wget without SSL verification
    if [[ "$code" =~ (curl|wget).*-k\|--insecure ]]; then
        warn_ref+=("Disabling SSL verification is insecure")
        if [[ "$risk_ref" == "low" ]]; then risk_ref="medium"; fi
    fi
    
    # Temporary file creation
    if [[ "$code" =~ /tmp/.*\$\$ ]] && [[ ! "$code" =~ mktemp ]]; then
        warn_ref+=("Use mktemp for secure temporary file creation")
        if [[ "$risk_ref" == "low" ]]; then risk_ref="medium"; fi
    fi
}

#######################################
# Validate analyze_code arguments
# Arguments:
#   $1 - Tool arguments (JSON)
# Returns:
#   0 if valid, 1 if invalid
#######################################
tool_analyze_code::validate() {
    local arguments="$1"
    
    # Check required fields
    if ! echo "$arguments" | jq -e '.code' &>/dev/null; then
        log::debug "analyze_code: missing code parameter"
        return 1
    fi
    
    # Validate code is not empty
    local code
    code=$(echo "$arguments" | jq -r '.code // ""')
    if [[ -z "$code" ]]; then
        log::debug "analyze_code: code parameter is empty"
        return 1
    fi
    
    # Validate analysis_type if provided
    local analysis_type
    analysis_type=$(echo "$arguments" | jq -r '.analysis_type // "syntax"')
    case "$analysis_type" in
        syntax|style|complexity|security|all)
            # Valid
            ;;
        *)
            log::debug "analyze_code: invalid analysis_type: $analysis_type"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Get tool information
# Returns:
#   Tool info JSON
#######################################
tool_analyze_code::info() {
    cat << 'EOF'
{
  "name": "analyze_code",
  "category": "code",
  "description": "Perform static code analysis for syntax, style, complexity, and security",
  "security_level": "low",
  "supports_contexts": ["sandbox", "local"],
  "analysis_types": [
    "syntax - Check for syntax errors and language-specific issues",
    "style - Analyze code style and formatting",
    "complexity - Calculate cyclomatic complexity and maintainability metrics",
    "security - Detect potential security vulnerabilities",
    "all - Comprehensive analysis including all above types"
  ],
  "supported_languages": [
    "python", "javascript", "typescript", "go", "java", "c", "cpp", "rust", "bash"
  ],
  "features": [
    "Language auto-detection",
    "Multi-metric analysis",
    "Security vulnerability detection",
    "Style and formatting checks",
    "Complexity scoring"
  ]
}
EOF
}

# Export functions
export -f tool_analyze_code::execute
export -f tool_analyze_code::validate