#!/bin/bash

# Docker Image Manager for Judge0 Direct Executor
# Manages Docker images for various programming languages

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Simple logging
log() {
    local level="$1"
    shift
    echo "[$(date +'%H:%M:%S')] [$level] $*" >&2
}

# Language to Docker image mapping
declare -A LANGUAGE_IMAGES=(
    ["python3"]="python:3.11-slim"
    ["python"]="python:3.11-slim"
    ["javascript"]="node:18-slim"
    ["node"]="node:18-slim"
    ["java"]="openjdk:17-slim"
    ["cpp"]="gcc:12"
    ["c++"]="gcc:12"
    ["c"]="gcc:12"
    ["go"]="golang:1.21-alpine"
    ["rust"]="rust:1.75-slim"
    ["ruby"]="ruby:3.2-slim"
    ["php"]="php:8.2-cli"
    ["csharp"]="mcr.microsoft.com/dotnet/sdk:7.0"
    ["typescript"]="node:18-slim"
    ["kotlin"]="zenika/kotlin:1.9"
    ["swift"]="swift:5.9-slim"
    ["r"]="r-base:4.3.2"
    ["perl"]="perl:5.38-slim"
    ["lua"]="nickblah/lua:5.4-alpine"
    ["haskell"]="haskell:9.6-slim"
    ["scala"]="hseeberger/scala-sbt:11.0.20_1.9.6_3.3.1"
    ["clojure"]="clojure:temurin-21-tools-deps"
    ["elixir"]="elixir:1.15-slim"
    ["erlang"]="erlang:26-slim"
    ["julia"]="julia:1.9-slim"
    ["fortran"]="gcc:12"
    ["cobol"]="esolang/cobol"
    ["pascal"]="nacyot/pascal:latest"
    ["dart"]="dart:3.2-sdk"
    ["nim"]="nimlang/nim:2.0.0-alpine"
    ["crystal"]="crystallang/crystal:1.10-alpine"
    ["zig"]="euantorano/zig:0.11.0"
)

# Check if Docker is available
check_docker() {
    if ! command -v docker &>/dev/null; then
        log error "Docker is not installed or not in PATH"
        return 1
    fi
    
    if ! docker info &>/dev/null; then
        log error "Docker daemon is not running or not accessible"
        return 1
    fi
    
    return 0
}

# Check if an image is available locally
is_image_available() {
    local image="$1"
    docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${image}$"
}

# Pull a Docker image
pull_image() {
    local image="$1"
    log info "Pulling Docker image: $image"
    
    if docker pull "$image"; then
        log success "Successfully pulled: $image"
        return 0
    else
        log error "Failed to pull: $image"
        return 1
    fi
}

# Install images for a specific language
install_language() {
    local language="$1"
    
    if [[ ! "${LANGUAGE_IMAGES[$language]+isset}" ]]; then
        log error "Unknown language: $language"
        echo "Available languages:"
        for lang in "${!LANGUAGE_IMAGES[@]}"; do
            echo "  - $lang"
        done | sort
        return 1
    fi
    
    local image="${LANGUAGE_IMAGES[$language]}"
    
    if is_image_available "$image"; then
        log info "Image already available: $image"
        return 0
    else
        pull_image "$image"
        return $?
    fi
}

# Install all language images
install_all() {
    local languages=("$@")
    
    # If no languages specified, install common ones
    if [[ ${#languages[@]} -eq 0 ]]; then
        languages=(
            "python3"
            "javascript"
            "java"
            "cpp"
            "go"
            "rust"
            "ruby"
            "php"
        )
        log info "Installing common language images..."
    fi
    
    local success=0
    local failed=0
    
    for lang in "${languages[@]}"; do
        if install_language "$lang"; then
            ((success++))
        else
            ((failed++))
        fi
    done
    
    echo ""
    echo "Installation Summary:"
    echo "  ✅ Successful: $success"
    echo "  ❌ Failed: $failed"
    
    return $( [[ $failed -eq 0 ]] && echo 0 || echo 1 )
}

# List available languages and their images
list_languages() {
    echo "Available Languages and Docker Images:"
    echo "======================================"
    
    for lang in $(printf '%s\n' "${!LANGUAGE_IMAGES[@]}" | sort); do
        local image="${LANGUAGE_IMAGES[$lang]}"
        local status="❌ Not installed"
        
        if is_image_available "$image"; then
            status="✅ Installed"
        fi
        
        printf "  %-15s %-40s %s\n" "$lang" "$image" "$status"
    done
}

# Check status of all language images
check_status() {
    local installed=0
    local not_installed=0
    
    echo "Docker Image Status for Judge0 Languages"
    echo "========================================="
    
    for lang in "${!LANGUAGE_IMAGES[@]}"; do
        local image="${LANGUAGE_IMAGES[$lang]}"
        if is_image_available "$image" 2>/dev/null; then
            ((installed++)) || true
        else
            ((not_installed++)) || true
        fi
    done
    
    echo "  📊 Total languages: ${#LANGUAGE_IMAGES[@]}"
    echo "  ✅ Installed: $installed"
    echo "  ❌ Not installed: $not_installed"
    echo ""
    
    if [[ $not_installed -gt 0 ]]; then
        echo "To install missing images, run:"
        echo "  $0 install all"
    else
        echo "All language images are installed!"
    fi
}

# Clean up unused Docker images
cleanup_images() {
    log info "Cleaning up unused Docker images..."
    
    # Get list of required images
    local required_images=()
    for image in "${LANGUAGE_IMAGES[@]}"; do
        required_images+=("$image")
    done
    
    # Clean unused images
    docker image prune -f
    
    log success "Cleanup complete"
}

# Test execution with installed images
test_languages() {
    local languages=("$@")
    
    if [[ ${#languages[@]} -eq 0 ]]; then
        languages=("python3" "javascript" "ruby")
    fi
    
    echo "Testing Language Execution"
    echo "=========================="
    
    for lang in "${languages[@]}"; do
        echo ""
        echo "Testing $lang:"
        
        if [[ ! "${LANGUAGE_IMAGES[$lang]+isset}" ]]; then
            echo "  ❌ Unknown language"
            continue
        fi
        
        local image="${LANGUAGE_IMAGES[$lang]}"
        
        if ! is_image_available "$image"; then
            echo "  ❌ Image not installed: $image"
            echo "     Run: $0 install $lang"
            continue
        fi
        
        # Test simple code execution
        local test_code=""
        case "$lang" in
            python|python3)
                test_code='print("Hello from Python")'
                ;;
            javascript|node)
                test_code='console.log("Hello from JavaScript")'
                ;;
            ruby)
                test_code='puts "Hello from Ruby"'
                ;;
            java)
                test_code='public class Test { public static void main(String[] args) { System.out.println("Hello from Java"); } }'
                ;;
            *)
                echo "  ⚠️  No test code defined for $lang"
                continue
                ;;
        esac
        
        # Try to execute using direct executor if available
        if [[ -f "$SCRIPT_DIR/direct-executor.sh" ]]; then
            local result=$("$SCRIPT_DIR/direct-executor.sh" execute "$lang" "$test_code" 2>/dev/null || echo '{"status": "error"}')
            
            if [[ "$result" =~ \"status\":\ *\"accepted\" ]]; then
                echo "  ✅ Execution successful"
                local output=$(echo "$result" | jq -r '.stdout' 2>/dev/null || echo "")
                [[ -n "$output" ]] && echo "     Output: $output"
            else
                echo "  ❌ Execution failed"
            fi
        else
            echo "  ⚠️  Direct executor not available for testing"
        fi
    done
    
    echo ""
    echo "=========================="
}

# Main execution
main() {
    # Check Docker availability first
    if ! check_docker; then
        exit 1
    fi
    
    case "${1:-help}" in
        install)
            shift
            if [[ "${1:-}" == "all" ]]; then
                shift
                install_all "$@"
            else
                install_language "$@"
            fi
            ;;
        list)
            list_languages
            ;;
        status)
            check_status
            ;;
        test)
            shift
            test_languages "$@"
            ;;
        cleanup)
            cleanup_images
            ;;
        help|*)
            echo "Judge0 Docker Image Manager"
            echo ""
            echo "Usage: $0 {install|list|status|test|cleanup|help} [options]"
            echo ""
            echo "Commands:"
            echo "  install <language>       Install Docker image for a language"
            echo "  install all [langs...]   Install all or specified language images"
            echo "  list                     List all languages and their images"
            echo "  status                   Check installation status"
            echo "  test [languages...]      Test language execution"
            echo "  cleanup                  Clean up unused Docker images"
            echo "  help                     Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 install python3       # Install Python image"
            echo "  $0 install all           # Install common language images"
            echo "  $0 test python3 javascript  # Test specific languages"
            exit 0
            ;;
    esac
}

main "$@"