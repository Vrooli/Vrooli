#!/usr/bin/env bash
# Speaker Verification - CLI Profile and Content Commands

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

#######################################
# Content: enroll a speaker profile
# Arguments: --profile <id> --file <audio_file> [--name <display_name>]
#######################################
speaker_verification::content::enroll() {
    local profile_id="" audio_file="" display_name=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --profile) profile_id="$2"; shift 2 ;;
            --file)    audio_file="$2"; shift 2 ;;
            --name)    display_name="$2"; shift 2 ;;
            *)
                log::error "Unknown argument: $1"
                echo "Usage: resource-speaker-verification content enroll --profile <id> --file <audio.wav> [--name <display_name>]"
                return 1
                ;;
        esac
    done

    if [[ -z "$profile_id" ]] || [[ -z "$audio_file" ]]; then
        echo "Usage: resource-speaker-verification content enroll --profile <id> --file <audio.wav> [--name <display_name>]"
        return 1
    fi

    if [[ ! -f "$audio_file" ]]; then
        log::error "Audio file not found: $audio_file"
        return 1
    fi

    log::info "Enrolling profile '${profile_id}' from ${audio_file}..."

    local result
    if result=$(speaker_verification::api::enroll "$profile_id" "$audio_file" "${display_name:-$profile_id}"); then
        echo "$result" | jq . 2>/dev/null || echo "$result"
        log::success "$MSG_ENROLL_SUCCESS"
        return 0
    else
        log::error "Enrollment failed"
        return 1
    fi
}
export -f speaker_verification::content::enroll

#######################################
# Content: verify speaker against profile
# Arguments: --profile <id> --file <audio_file> [--threshold <float>]
#######################################
speaker_verification::content::verify() {
    local profile_id="" audio_file="" threshold=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --profile)   profile_id="$2"; shift 2 ;;
            --file)      audio_file="$2"; shift 2 ;;
            --threshold) threshold="$2"; shift 2 ;;
            *)
                log::error "Unknown argument: $1"
                echo "Usage: resource-speaker-verification content verify --profile <id> --file <audio.wav> [--threshold <float>]"
                return 1
                ;;
        esac
    done

    if [[ -z "$profile_id" ]] || [[ -z "$audio_file" ]]; then
        echo "Usage: resource-speaker-verification content verify --profile <id> --file <audio.wav> [--threshold <float>]"
        return 1
    fi

    if [[ ! -f "$audio_file" ]]; then
        log::error "Audio file not found: $audio_file"
        return 1
    fi

    log::info "Verifying audio against profile '${profile_id}'..."

    local result
    if result=$(speaker_verification::api::verify "$profile_id" "$audio_file" "$threshold"); then
        echo "$result" | jq . 2>/dev/null || echo "$result"

        local matched
        matched=$(echo "$result" | jq -r '.matched // false' 2>/dev/null)
        if [[ "$matched" == "true" ]]; then
            log::success "$MSG_VERIFY_MATCH"
        else
            log::warn "$MSG_VERIFY_NO_MATCH"
        fi
        return 0
    else
        log::error "Verification failed"
        return 1
    fi
}
export -f speaker_verification::content::verify

#######################################
# Content: manage profiles (list/get/remove)
# Arguments: list | get --profile <id> | remove --profile <id>
#######################################
speaker_verification::content::profiles() {
    local subcommand="${1:-list}"
    shift || true

    case "$subcommand" in
        list)
            local result
            if result=$(speaker_verification::api::list_profiles); then
                echo "$result" | jq . 2>/dev/null || echo "$result"
            else
                return 1
            fi
            ;;
        get)
            local profile_id=""
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --profile) profile_id="$2"; shift 2 ;;
                    *) profile_id="$1"; shift ;;
                esac
            done
            if [[ -z "$profile_id" ]]; then
                echo "Usage: resource-speaker-verification content profiles get --profile <id>"
                return 1
            fi
            local result
            if result=$(speaker_verification::api::get_profile "$profile_id"); then
                echo "$result" | jq . 2>/dev/null || echo "$result"
            else
                return 1
            fi
            ;;
        remove)
            local profile_id=""
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --profile) profile_id="$2"; shift 2 ;;
                    *) profile_id="$1"; shift ;;
                esac
            done
            if [[ -z "$profile_id" ]]; then
                echo "Usage: resource-speaker-verification content profiles remove --profile <id>"
                return 1
            fi
            if speaker_verification::api::delete_profile "$profile_id"; then
                log::success "Profile '${profile_id}' removed"
            else
                return 1
            fi
            ;;
        help|--help|-h)
            echo "Usage: resource-speaker-verification content profiles <subcommand>"
            echo
            echo "Subcommands:"
            echo "  list                     List all profiles"
            echo "  get --profile <id>       Show profile details"
            echo "  remove --profile <id>    Remove a profile"
            ;;
        *)
            log::error "Unknown profiles subcommand: $subcommand"
            echo "Use 'resource-speaker-verification content profiles help' for available commands"
            return 1
            ;;
    esac
}
export -f speaker_verification::content::profiles

#######################################
# Content: show service info
#######################################
speaker_verification::content::info() {
    local result
    if result=$(speaker_verification::api::info); then
        echo "$result" | jq . 2>/dev/null || echo "$result"
    else
        return 1
    fi
}
export -f speaker_verification::content::info
