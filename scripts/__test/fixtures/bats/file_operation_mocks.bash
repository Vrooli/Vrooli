#!/bin/bash
# File Operation Mocks
# Provides consistent file operation mocking to prevent real filesystem access

#######################################
# Setup file operation mocks
# Globals: None
# Arguments: None  
# Returns: 0 on success
#######################################
setup_file_operation_mocks() {
    # Prevent duplicate setup
    if [[ "${FILE_MOCKS_LOADED:-}" == "true" ]]; then
        return 0
    fi
    export FILE_MOCKS_LOADED="true"
    
    # Mock file tests
    test() {
        case "$*" in
            *"-f"*) return 0 ;;  # File exists
            *"-d"*) return 0 ;;  # Directory exists
            *"-e"*) return 0 ;;  # Path exists
            *"-r"*) return 0 ;;  # Readable
            *"-w"*) return 0 ;;  # Writable
            *"-x"*) return 0 ;;  # Executable
            *"-s"*) return 0 ;;  # Non-empty file
            *) return 1 ;;
        esac
    }
    
    # Mock directory operations
    mkdir() {
        return 0
    }
    
    mktemp() {
        echo "/tmp/mock_temp_$$"
    }
    
    # Mock file operations
    touch() {
        return 0
    }
    
    chmod() {
        return 0
    }
    
    chown() {
        return 0
    }
    
    # Mock file content operations
    cat() {
        case "$*" in
            */package.json*) echo '{"name": "test-package", "version": "1.0.0"}' ;;
            */.env*) echo "TEST_VAR=test_value" ;;
            */Dockerfile*) echo "FROM node:latest" ;;
            *) echo "mock file content" ;;
        esac
        return 0
    }
    
    # Mock grep to prevent file searches
    grep() {
        case "$*" in
            *"^ok"*|*"^not ok"*) echo "ok 1 test" ;;
            *"Command not found"*) return 1 ;;
            *) echo "mock grep result" ;;
        esac
        return 0
    }
    
    # Mock find to prevent directory traversal
    find() {
        case "$*" in
            *".bats"*) echo "test.bats" ;;
            *".sh"*) echo "script.sh" ;;
            *) echo "mock_file" ;;
        esac
        return 0
    }
    
    # Mock ls
    ls() {
        echo "file1 file2 file3"
        return 0
    }
    
    # Export all mocked functions
    export -f test mkdir mktemp touch chmod chown cat grep find ls
    
    return 0
}

#######################################
# Cleanup file operation mocks
# Globals: FILE_MOCKS_LOADED
# Arguments: None
# Returns: 0 on success
#######################################
cleanup_file_operation_mocks() {
    unset FILE_MOCKS_LOADED
    return 0
}

# Auto-setup if sourced
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    setup_file_operation_mocks
fi