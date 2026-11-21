#!/bin/sh

# Shared, POSIX installer primitives. Keep this file free of side effects: the
# source-build installer and the public prebuilt installer both source it.

vrooli_detect_os() {
    case "$(uname -s 2>/dev/null)" in
        Linux) printf '%s\n' linux ;;
        Darwin) printf '%s\n' darwin ;;
        MINGW*|MSYS*|CYGWIN*) printf '%s\n' windows ;;
        *) printf 'unsupported operating system: %s\n' "$(uname -s 2>/dev/null || printf unknown)" >&2; return 1 ;;
    esac
}

vrooli_detect_arch() {
    case "$(uname -m 2>/dev/null)" in
        x86_64|amd64) printf '%s\n' amd64 ;;
        arm64|aarch64) printf '%s\n' arm64 ;;
        *) printf 'unsupported architecture: %s\n' "$(uname -m 2>/dev/null || printf unknown)" >&2; return 1 ;;
    esac
}

vrooli_default_install_dir() {
    if [ -z "${HOME:-}" ]; then
        printf '%s\n' 'HOME is required to resolve the Vrooli install directory' >&2
        return 1
    fi
    printf '%s\n' "${HOME}/.vrooli/bin"
}

vrooli_download() {
    vrooli_platform_source_url=$1
    vrooli_platform_destination=$2
    curl -fsSL --retry 3 --retry-delay 1 --connect-timeout 15 -o "${vrooli_platform_destination}" "${vrooli_platform_source_url}"
}

vrooli_sha256() {
    vrooli_platform_target=$1
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "${vrooli_platform_target}" | { read -r vrooli_platform_digest _rest; printf '%s\n' "${vrooli_platform_digest}"; }
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "${vrooli_platform_target}" | { read -r vrooli_platform_digest _rest; printf '%s\n' "${vrooli_platform_digest}"; }
    elif command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 "${vrooli_platform_target}" | sed 's/^.*= //'
    else
        printf '%s\n' 'checksum verification requires sha256sum, shasum, or openssl' >&2
        return 1
    fi
}

vrooli_expected_sha256() {
    vrooli_platform_manifest=$1
    vrooli_platform_asset=$2
    while read -r vrooli_platform_digest vrooli_platform_filename _rest; do
        vrooli_platform_filename=${vrooli_platform_filename#\*}
        if [ "${vrooli_platform_filename}" = "${vrooli_platform_asset}" ]; then
            printf '%s\n' "${vrooli_platform_digest}"
            return 0
        fi
    done < "${vrooli_platform_manifest}"
    printf 'release checksum manifest does not contain %s\n' "${vrooli_platform_asset}" >&2
    return 1
}

vrooli_verify_checksum() {
    vrooli_platform_verify_manifest=$1
    vrooli_platform_verify_asset=$2
    vrooli_platform_verify_path=$3
    vrooli_platform_expected=$(vrooli_expected_sha256 "${vrooli_platform_verify_manifest}" "${vrooli_platform_verify_asset}") || return 1
    vrooli_platform_actual=$(vrooli_sha256 "${vrooli_platform_verify_path}") || return 1
    if [ "${vrooli_platform_actual}" != "${vrooli_platform_expected}" ]; then
        printf 'checksum mismatch for %s\nexpected: %s\nactual:   %s\n' "${vrooli_platform_verify_asset}" "${vrooli_platform_expected}" "${vrooli_platform_actual}" >&2
        return 1
    fi
}

vrooli_verify_signature() {
    vrooli_platform_signature_manifest=$1
    vrooli_platform_signature_base64=$2
    vrooli_platform_public_key=$3
    if ! command -v openssl >/dev/null 2>&1; then
        printf '%s\n' 'release signature verification requires openssl' >&2
        return 1
    fi
    vrooli_platform_decoded_signature="${vrooli_platform_signature_base64}.decoded.$$"
    if ! openssl base64 -d -A -in "${vrooli_platform_signature_base64}" -out "${vrooli_platform_decoded_signature}"; then
		rm -f "${vrooli_platform_decoded_signature}"
        printf '%s\n' 'release signature is not valid base64' >&2
        return 1
    fi
    if ! openssl dgst -sha256 -verify "${vrooli_platform_public_key}" -signature "${vrooli_platform_decoded_signature}" "${vrooli_platform_signature_manifest}" >/dev/null 2>&1; then
		rm -f "${vrooli_platform_decoded_signature}"
        printf '%s\n' 'release signature verification failed' >&2
        return 1
    fi
    rm -f "${vrooli_platform_decoded_signature}"
}
