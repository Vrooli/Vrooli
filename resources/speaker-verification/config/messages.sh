#!/usr/bin/env bash
# Speaker Verification User-Facing Messages

#######################################
# Export all user-facing messages
# Idempotent - safe to call multiple times
#######################################
messages::export_messages() {
    # Guard against multiple calls
    if [[ -n "${SPEAKER_VERIFICATION_MESSAGES_EXPORTED:-}" ]]; then
        return 0
    fi
    readonly SPEAKER_VERIFICATION_MESSAGES_EXPORTED="true"

    # Success messages
    [[ -z "${MSG_INSTALL_SUCCESS:-}" ]] && readonly MSG_INSTALL_SUCCESS="Speaker verification installed successfully"
    [[ -z "${MSG_START_SUCCESS:-}" ]] && readonly MSG_START_SUCCESS="Speaker verification service started"
    [[ -z "${MSG_STOP_SUCCESS:-}" ]] && readonly MSG_STOP_SUCCESS="Speaker verification service stopped"
    [[ -z "${MSG_ENROLL_SUCCESS:-}" ]] && readonly MSG_ENROLL_SUCCESS="Speaker profile enrolled successfully"
    [[ -z "${MSG_VERIFY_MATCH:-}" ]] && readonly MSG_VERIFY_MATCH="Speaker verified - match"
    [[ -z "${MSG_VERIFY_NO_MATCH:-}" ]] && readonly MSG_VERIFY_NO_MATCH="Speaker verification - no match"

    # Error messages
    [[ -z "${MSG_DOCKER_NOT_FOUND:-}" ]] && readonly MSG_DOCKER_NOT_FOUND="Docker is required but not installed or not running"
    [[ -z "${MSG_CONTAINER_NOT_RUNNING:-}" ]] && readonly MSG_CONTAINER_NOT_RUNNING="Speaker verification container is not running"
    [[ -z "${MSG_API_UNREACHABLE:-}" ]] && readonly MSG_API_UNREACHABLE="Speaker verification API is not reachable"
    [[ -z "${MSG_PROFILE_NOT_FOUND:-}" ]] && readonly MSG_PROFILE_NOT_FOUND="Speaker profile not found"
    [[ -z "${MSG_AUDIO_TOO_SHORT:-}" ]] && readonly MSG_AUDIO_TOO_SHORT="Audio file is too short for processing"
    [[ -z "${MSG_AUDIO_INVALID:-}" ]] && readonly MSG_AUDIO_INVALID="Audio file is invalid or unreadable"

    # Info messages
    [[ -z "${MSG_INSTALLING:-}" ]] && readonly MSG_INSTALLING="Installing speaker verification service..."
    [[ -z "${MSG_STARTING:-}" ]] && readonly MSG_STARTING="Starting speaker verification service..."
    [[ -z "${MSG_STOPPING:-}" ]] && readonly MSG_STOPPING="Stopping speaker verification service..."
    [[ -z "${MSG_WAITING_HEALTH:-}" ]] && readonly MSG_WAITING_HEALTH="Waiting for speaker verification to become healthy..."
    [[ -z "${MSG_MODEL_LOADING:-}" ]] && readonly MSG_MODEL_LOADING="Loading TitaNet model (this may take a minute)..."

    # Warning messages
    [[ -z "${MSG_GPU_NOT_AVAILABLE:-}" ]] && readonly MSG_GPU_NOT_AVAILABLE="GPU not available, falling back to CPU"
    [[ -z "${MSG_THRESHOLD_WARNING:-}" ]] && readonly MSG_THRESHOLD_WARNING="Custom threshold in use - verify calibration"

    # Export all
    export MSG_INSTALL_SUCCESS MSG_START_SUCCESS MSG_STOP_SUCCESS
    export MSG_ENROLL_SUCCESS MSG_VERIFY_MATCH MSG_VERIFY_NO_MATCH
    export MSG_DOCKER_NOT_FOUND MSG_CONTAINER_NOT_RUNNING MSG_API_UNREACHABLE
    export MSG_PROFILE_NOT_FOUND MSG_AUDIO_TOO_SHORT MSG_AUDIO_INVALID
    export MSG_INSTALLING MSG_STARTING MSG_STOPPING MSG_WAITING_HEALTH MSG_MODEL_LOADING
    export MSG_GPU_NOT_AVAILABLE MSG_THRESHOLD_WARNING
}

export -f messages::export_messages
