#!/bin/sh
set -eu

repository=${VROOLI_GITHUB_REPOSITORY:-Vrooli/Vrooli}
version=${VROOLI_VERSION:-latest}
if [ -n "${VROOLI_RELEASE_BASE_URL:-}" ]; then
    release_base=${VROOLI_RELEASE_BASE_URL%/}
elif [ "${version}" = latest ]; then
    release_base="https://github.com/${repository}/releases/latest/download"
else
    case "${version}" in v*) tag=${version} ;; *) tag=v${version} ;; esac
    release_base="https://github.com/${repository}/releases/download/${tag}"
fi

if ! command -v curl >/dev/null 2>&1; then
    printf '%s\n' 'curl is required to install Vrooli' >&2
    exit 1
fi
install_source=${VROOLI_INSTALL_SOURCE:-1}

run_bootstrap_privileged() {
    if [ "$(id -u)" = 0 ]; then
        "$@"
        return
    fi
    if command -v sudo >/dev/null 2>&1; then
        printf 'Vrooli needs administrator access to install release verification tools: %s\n' "$*"
        sudo "$@"
        return
    fi
    printf '%s\n' 'Release verification tools are missing and sudo is unavailable' >&2
    return 1
}

install_bootstrap_tools() {
    packages=$1
    printf 'Installing required release verification tools: %s\n' "${packages}"
    if command -v apt-get >/dev/null 2>&1; then
        run_bootstrap_privileged apt-get update
        # Package names are fixed values assembled above, not user input.
        # shellcheck disable=SC2086
        run_bootstrap_privileged apt-get install -y --no-install-recommends ${packages}
    elif command -v dnf >/dev/null 2>&1; then
        # shellcheck disable=SC2086
        run_bootstrap_privileged dnf install -y ${packages}
    elif command -v yum >/dev/null 2>&1; then
        # shellcheck disable=SC2086
        run_bootstrap_privileged yum install -y ${packages}
    elif command -v pacman >/dev/null 2>&1; then
        # shellcheck disable=SC2086
        run_bootstrap_privileged pacman -Sy --needed --noconfirm ${packages}
    elif command -v apk >/dev/null 2>&1; then
        # shellcheck disable=SC2086
        run_bootstrap_privileged apk add --no-cache ${packages}
    else
        printf 'Cannot install release verification tools (%s): no supported package manager found\n' "${packages}" >&2
        return 1
    fi
}

bootstrap_packages=
if ! command -v openssl >/dev/null 2>&1; then
    bootstrap_packages=openssl
fi
if [ "${install_source}" = 1 ] && ! command -v tar >/dev/null 2>&1; then
    bootstrap_packages="${bootstrap_packages}${bootstrap_packages:+ }tar"
fi
if [ -n "${bootstrap_packages}" ]; then
    install_bootstrap_tools "${bootstrap_packages}"
fi
if ! command -v openssl >/dev/null 2>&1; then
    printf '%s\n' 'OpenSSL is required to authenticate the Vrooli release manifest and automatic installation failed' >&2
    exit 1
fi
if [ "${install_source}" = 1 ] && ! command -v tar >/dev/null 2>&1; then
    printf '%s\n' 'tar is required to install the authenticated Vrooli source tree and automatic installation failed' >&2
    exit 1
fi

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/vrooli-install.XXXXXX")
cleanup() { rm -rf "${work_dir}"; }
trap cleanup EXIT HUP INT TERM

helper_asset=vrooli-install-lib.sh
manifest_asset=SHA256SUMS
signature_asset=SHA256SUMS.sig
curl -fsSL --retry 3 --retry-delay 1 -o "${work_dir}/${helper_asset}" "${release_base}/${helper_asset}"
curl -fsSL --retry 3 --retry-delay 1 -o "${work_dir}/${manifest_asset}" "${release_base}/${manifest_asset}"
curl -fsSL --retry 3 --retry-delay 1 -o "${work_dir}/${signature_asset}" "${release_base}/${signature_asset}"

if [ -n "${VROOLI_RELEASE_PUBLIC_KEY_FILE:-}" ]; then
    cp "${VROOLI_RELEASE_PUBLIC_KEY_FILE}" "${work_dir}/vrooli-release.pub"
else
cat > "${work_dir}/vrooli-release.pub" <<'VROOLI_RELEASE_PUBLIC_KEY'
-----BEGIN PUBLIC KEY-----
MIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAzTHpnCxVTVn7wSr/ZCpl
J5U07JXI5BLwpXoz2HVljpD4Rxv3A++PQ3E6Yje0C/F7AKH6+KAToETwPxiFxOQZ
VuU9ZYJXDMOIxEvOqu0yBKaQC2o2sv+JTmoWCCmxf4ADk2JQ/zMvHZpy9tO0Z1Tg
1dGORGV9UaKd7l5Lm7Aam0VtJ6NDXuzwbTb+Ag+98xiM88MJX1RcJVdM5auwExKg
qmHsBiFIL48jRCaJYMDtk5uB7UkFw9QsqbG4PM2J7lGnf5UgyIcM7KXH5HU009rq
apqFy9dt6BlAS4xByEpOB0uf7v79zdNqRiKPhC0Zu3rT4mfslhz9TQj2q1C/zfOH
XflKh/DY9qvhlD6rXbRCrIf31PP1ZVzu4sryQxxrPEAfNsdns4mmmSSGwVkKIn+/
dOCvbCEAy83Mxt5XN66Yn94rKO/8Z2V3qer7789Fq39fJC4fkbngFkLqpG9HxSl5
xOWW4L6Cfs7uecjvPUcxOqWkcpFR3PTM9gcGdqonbagLAgMBAAE=
-----END PUBLIC KEY-----
VROOLI_RELEASE_PUBLIC_KEY
fi

openssl base64 -d -A -in "${work_dir}/${signature_asset}" -out "${work_dir}/manifest.sig"
if ! openssl dgst -sha256 -verify "${work_dir}/vrooli-release.pub" -signature "${work_dir}/manifest.sig" "${work_dir}/${manifest_asset}" >/dev/null 2>&1; then
    printf '%s\n' 'Vrooli release signature verification failed; nothing was installed' >&2
    exit 1
fi

expected_helper=
while read -r digest filename _rest; do
    filename=${filename#\*}
    if [ "${filename}" = "${helper_asset}" ]; then expected_helper=${digest}; break; fi
done < "${work_dir}/${manifest_asset}"
if [ -z "${expected_helper}" ]; then
    printf '%s\n' 'Vrooli release manifest does not authenticate the installer helper' >&2
    exit 1
fi
actual_helper=$(openssl dgst -sha256 "${work_dir}/${helper_asset}" | sed 's/^.*= //')
if [ "${actual_helper}" != "${expected_helper}" ]; then
    printf '%s\n' 'Vrooli installer helper checksum mismatch; nothing was installed' >&2
    exit 1
fi

# shellcheck source=../packages/cli-core/install/platform.sh
. "${work_dir}/${helper_asset}"
os_name=$(vrooli_detect_os)
arch_name=$(vrooli_detect_arch)
suffix=
if [ "${os_name}" = windows ]; then suffix=.exe; fi
asset="vrooli_${os_name}_${arch_name}${suffix}"
sidecar_asset="${asset}.fp"
source_asset=vrooli-source.tar.gz
vrooli_download "${release_base}/${asset}" "${work_dir}/${asset}"
vrooli_download "${release_base}/${sidecar_asset}" "${work_dir}/${sidecar_asset}"
if [ "${install_source}" = 1 ]; then
    vrooli_download "${release_base}/${source_asset}" "${work_dir}/${source_asset}"
fi
vrooli_verify_signature "${work_dir}/${manifest_asset}" "${work_dir}/${signature_asset}" "${work_dir}/vrooli-release.pub"
vrooli_verify_checksum "${work_dir}/${manifest_asset}" "${asset}" "${work_dir}/${asset}"
vrooli_verify_checksum "${work_dir}/${manifest_asset}" "${sidecar_asset}" "${work_dir}/${sidecar_asset}"
if [ "${install_source}" = 1 ]; then
    vrooli_verify_checksum "${work_dir}/${manifest_asset}" "${source_asset}" "${work_dir}/${source_asset}"
fi

# Managed-service servers are release assets in their own right. Install the
# target-specific files only after the release manifest has been authenticated
# and each file's digest verified. The compact index is generated by the
# release packager; strict field validation prevents it from choosing paths
# outside the per-user artifact store.
resource_index_asset=resource-artifacts-v1.txt
if resource_index_digest=$(vrooli_expected_sha256 "${work_dir}/${manifest_asset}" "${resource_index_asset}" 2>/dev/null); then
    vrooli_download "${release_base}/${resource_index_asset}" "${work_dir}/${resource_index_asset}"
    vrooli_verify_checksum "${work_dir}/${manifest_asset}" "${resource_index_asset}" "${work_dir}/${resource_index_asset}"
    artifact_root=${VROOLI_RESOURCE_ARTIFACT_DIR:-${HOME}/.vrooli/artifacts}
    while IFS="$(printf '\t')" read -r resource_name resource_version resource_os resource_arch resource_asset; do
        [ -n "${resource_name}" ] || continue
        case "${resource_name}" in ""|*[!A-Za-z0-9._-]*|*..*)
            printf '%s\n' 'resource artifact index contains unsafe fields' >&2
            exit 1
            ;;
        esac
        case "${resource_version}" in ""|*[!A-Za-z0-9._-]*|*..*)
            printf '%s\n' 'resource artifact index contains unsafe fields' >&2
            exit 1
            ;;
        esac
        case "${resource_os}" in ""|*[!A-Za-z0-9_-]*|*..*)
            printf '%s\n' 'resource artifact index contains unsafe fields' >&2
            exit 1
            ;;
        esac
        case "${resource_arch}" in ""|*[!A-Za-z0-9_-]*|*..*)
            printf '%s\n' 'resource artifact index contains unsafe fields' >&2
            exit 1
            ;;
        esac
        case "${resource_asset}" in ""|*[!A-Za-z0-9._-]*|*..*|*/*|*\\*)
                printf '%s\n' 'resource artifact index contains unsafe fields' >&2
                exit 1
                ;;
        esac
        [ "${resource_os}" = "${os_name}" ] || continue
        [ "${resource_arch}" = "${arch_name}" ] || continue
        vrooli_download "${release_base}/${resource_asset}" "${work_dir}/${resource_asset}"
        vrooli_verify_checksum "${work_dir}/${manifest_asset}" "${resource_asset}" "${work_dir}/${resource_asset}"
        resource_destination="${artifact_root}/${resource_name}/${resource_version}/${resource_asset}"
        mkdir -p "$(dirname "${resource_destination}")"
        chmod 0755 "${work_dir}/${resource_asset}"
        mv "${work_dir}/${resource_asset}" "${resource_destination}"
        printf 'Installed authenticated %s service artifact to %s\n' "${resource_name}" "${resource_destination}"
    done < "${work_dir}/${resource_index_asset}"
fi

source_root=
if [ "${install_source}" = 1 ]; then
    source_key=
    while read -r digest filename _rest; do
        filename=${filename#\*}
        if [ "${filename}" = "${source_asset}" ]; then source_key=${digest}; break; fi
    done < "${work_dir}/${manifest_asset}"
    if [ "${#source_key}" -ne 64 ]; then
        printf '%s\n' 'Vrooli release manifest has no valid source archive digest' >&2
        exit 1
    fi
    case "${source_key}" in
        *[!A-Fa-f0-9]*)
            printf '%s\n' 'Vrooli source archive digest cannot safely name the installed source tree' >&2
            exit 1
            ;;
    esac
    source_extract="${work_dir}/source-extract"
    mkdir -p "${source_extract}"
    tar -xzf "${work_dir}/${source_asset}" -C "${source_extract}"
    if [ ! -f "${source_extract}/Vrooli/go.mod" ]; then
        printf '%s\n' 'Authenticated Vrooli source archive has no Vrooli/go.mod root' >&2
        exit 1
    fi
    if [ -n "${VROOLI_SOURCE_DIR:-}" ]; then
        source_root=${VROOLI_SOURCE_DIR}
        if [ -e "${source_root}" ]; then
            printf 'Refusing to replace existing VROOLI_SOURCE_DIR: %s\n' "${source_root}" >&2
            exit 1
        fi
        mkdir -p "$(dirname "${source_root}")"
        mv "${source_extract}/Vrooli" "${source_root}"
    else
        source_parent="${HOME}/.vrooli/src/${source_key}"
        source_root="${source_parent}/Vrooli"
        if [ ! -f "${source_root}/go.mod" ]; then
            source_stage="${source_parent}.new.$$"
            rm -rf "${source_stage}"
            mkdir -p "${source_stage}"
            mv "${source_extract}/Vrooli" "${source_stage}/Vrooli"
            rm -rf "${source_parent}"
            mv "${source_stage}" "${source_parent}"
        fi
    fi
fi

install_dir=${VROOLI_INSTALL_DIR:-$(vrooli_default_install_dir)}
mkdir -p "${install_dir}"
chmod 0755 "${work_dir}/${asset}"
mv "${work_dir}/${sidecar_asset}" "${install_dir}/vrooli${suffix}.fp"
mv "${work_dir}/${asset}" "${install_dir}/vrooli${suffix}"

if [ -n "${source_root}" ]; then
    pointer_dir="${HOME}/.vrooli"
    mkdir -p "${pointer_dir}"
    pointer_tmp="${pointer_dir}/source-root.tmp.$$"
    printf '%s\n' "${source_root}" > "${pointer_tmp}"
    mv "${pointer_tmp}" "${pointer_dir}/source-root"
fi

printf 'Installed authenticated Vrooli CLI to %s\n' "${install_dir}/vrooli${suffix}"
if [ -n "${source_root}" ]; then
    printf 'Installed authenticated Vrooli source to %s\n' "${source_root}"
fi
case ":${PATH:-}:" in
    *:"${install_dir}":*) ;;
    *) printf 'Add %s to PATH, then run: vrooli setup\n' "${install_dir}" ;;
esac
if [ "${VROOLI_RUN_SETUP:-0}" = 1 ]; then
    if [ -z "${source_root}" ]; then
        printf '%s\n' 'VROOLI_RUN_SETUP=1 requires source installation (do not set VROOLI_INSTALL_SOURCE=0)' >&2
        exit 1
    fi
    VROOLI_SOURCE_ROOT="${source_root}" "${install_dir}/vrooli${suffix}" setup
fi
