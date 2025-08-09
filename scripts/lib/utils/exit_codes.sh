#!/usr/bin/env bash
# Standard exit codes used throughout the Vrooli project
# These codes provide consistent error handling across all scripts

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/var.sh"

# Prevent multiple sourcing
if [[ -n "${_EXIT_CODES_SOURCED:-}" ]]; then
    return 0
fi
readonly _EXIT_CODES_SOURCED=1

# Standard exit codes
readonly EXIT_SUCCESS=0              # Successful execution
readonly EXIT_GENERAL_ERROR=1        # General/unknown error
readonly EXIT_MISUSE=2               # Misuse of shell command
readonly EXIT_INVALID_ARGUMENT=3     # Invalid argument provided
readonly EXIT_FILE_NOT_FOUND=4       # File or directory not found
readonly EXIT_PERMISSION_DENIED=5    # Permission denied
readonly EXIT_DEPENDENCY_ERROR=6     # Missing or failed dependency
readonly EXIT_NETWORK_ERROR=7        # Network-related error
readonly EXIT_CONFIGURATION_ERROR=8  # Configuration error
readonly EXIT_STATE_ERROR=9          # Invalid state or precondition
readonly EXIT_RESOURCE_ERROR=10      # Resource unavailable or exhausted

# Signal-based exit codes
readonly EXIT_TIMEOUT=124            # Command timed out
readonly EXIT_USER_INTERRUPT=130     # User interrupt (Ctrl+C/SIGINT)

# Application-specific exit codes (11-50)
readonly ERROR_BACKUP_DIRECTORY_CREATION=11    # Failed to create backup directory
readonly ERROR_MISSING_BACKUP_TOOLS=12         # Required backup tools not available
readonly ERROR_INSUFFICIENT_DISK_SPACE=13      # Not enough disk space for backup
readonly ERROR_BACKUP_PERMISSION_DENIED=14     # No permission for backup operations
readonly ERROR_DATABASE_BACKUP_FAILED=15       # Database backup operation failed
readonly ERROR_BACKUP_ENVIRONMENT=16           # Backup environment not properly set up
readonly ERROR_INVALID_SSH_KEY=17              # SSH key format is invalid
readonly ERROR_SSH_KEY_FILE_READ=18            # Cannot read SSH key file
readonly ERROR_SSH_FILE_MISSING=19             # SSH key file is missing

# Export all exit codes
export EXIT_SUCCESS EXIT_GENERAL_ERROR EXIT_MISUSE EXIT_INVALID_ARGUMENT
export EXIT_FILE_NOT_FOUND EXIT_PERMISSION_DENIED EXIT_DEPENDENCY_ERROR
export EXIT_NETWORK_ERROR EXIT_CONFIGURATION_ERROR EXIT_STATE_ERROR
export EXIT_RESOURCE_ERROR EXIT_TIMEOUT EXIT_USER_INTERRUPT
export ERROR_BACKUP_DIRECTORY_CREATION ERROR_MISSING_BACKUP_TOOLS ERROR_INSUFFICIENT_DISK_SPACE
export ERROR_BACKUP_PERMISSION_DENIED ERROR_DATABASE_BACKUP_FAILED ERROR_BACKUP_ENVIRONMENT
export ERROR_INVALID_SSH_KEY ERROR_SSH_KEY_FILE_READ ERROR_SSH_FILE_MISSING

# Helper functions
exit_with_code() {
    local code="${1:-$EXIT_GENERAL_ERROR}"
    exit "$code"
}

exit_success() {
    exit "$EXIT_SUCCESS"
}

exit_error() {
    local code="${1:-$EXIT_GENERAL_ERROR}"
    exit "$code"
}

# Describe an exit code
describe_exit_code() {
    local code="${1:-}"
    
    case "$code" in
        0|"$EXIT_SUCCESS")
            echo "Success"
            ;;
        1|"$EXIT_GENERAL_ERROR")
            echo "General error"
            ;;
        2|"$EXIT_MISUSE")
            echo "Misuse of shell command"
            ;;
        3|"$EXIT_INVALID_ARGUMENT")
            echo "Invalid argument"
            ;;
        4|"$EXIT_FILE_NOT_FOUND")
            echo "File not found"
            ;;
        5|"$EXIT_PERMISSION_DENIED")
            echo "Permission denied"
            ;;
        6|"$EXIT_DEPENDENCY_ERROR")
            echo "Dependency error"
            ;;
        7|"$EXIT_NETWORK_ERROR")
            echo "Network error"
            ;;
        8|"$EXIT_CONFIGURATION_ERROR")
            echo "Configuration error"
            ;;
        9|"$EXIT_STATE_ERROR")
            echo "State error"
            ;;
        10|"$EXIT_RESOURCE_ERROR")
            echo "Resource error"
            ;;
        124|"$EXIT_TIMEOUT")
            echo "Command timed out"
            ;;
        130|"$EXIT_USER_INTERRUPT")
            echo "User interrupt (SIGINT)"
            ;;
        11|"$ERROR_BACKUP_DIRECTORY_CREATION")
            echo "Failed to create backup directory"
            ;;
        12|"$ERROR_MISSING_BACKUP_TOOLS")
            echo "Required backup tools not available"
            ;;
        13|"$ERROR_INSUFFICIENT_DISK_SPACE")
            echo "Not enough disk space for backup"
            ;;
        14|"$ERROR_BACKUP_PERMISSION_DENIED")
            echo "No permission for backup operations"
            ;;
        15|"$ERROR_DATABASE_BACKUP_FAILED")
            echo "Database backup operation failed"
            ;;
        16|"$ERROR_BACKUP_ENVIRONMENT")
            echo "Backup environment not properly set up"
            ;;
        17|"$ERROR_INVALID_SSH_KEY")
            echo "SSH key format is invalid"
            ;;
        18|"$ERROR_SSH_KEY_FILE_READ")
            echo "Cannot read SSH key file"
            ;;
        19|"$ERROR_SSH_FILE_MISSING")
            echo "SSH key file is missing"
            ;;
        *)
            echo "Unknown exit code: $code"
            ;;
    esac
}
