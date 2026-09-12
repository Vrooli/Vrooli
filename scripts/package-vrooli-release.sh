#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${VROOLI_RELEASE_OUT:-${ROOT}/dist/vrooli-release}"
VERSION="${VROOLI_VERSION:-}"
SIGNING_KEY="${VROOLI_RELEASE_SIGNING_KEY_FILE:-}"

PREBUILT="${VROOLI_RELEASE_PREBUILT_DIR:-}"

usage() {
    printf '%s\n' 'Usage: package-vrooli-release.sh --version <vX.Y.Z> --signing-key <private.pem> [--out <directory>] [--prebuilt <directory>]'
    printf '%s\n' '  --prebuilt  stage CLI binaries built on another runner (darwin must be built on macOS for a working Keychain)'
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --version) VERSION=${2:?missing version}; shift 2 ;;
        --signing-key) SIGNING_KEY=${2:?missing signing key}; shift 2 ;;
        --out) OUT=${2:?missing output directory}; shift 2 ;;
        --prebuilt) PREBUILT=${2:?missing prebuilt directory}; shift 2 ;;
        --help|-h) usage; exit 0 ;;
        *) printf 'unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "${VERSION}" || -z "${SIGNING_KEY}" ]]; then
    usage >&2
    exit 2
fi
if [[ ! -f "${SIGNING_KEY}" ]]; then
    printf 'release signing key does not exist: %s\n' "${SIGNING_KEY}" >&2
    exit 1
fi
if ! command -v openssl >/dev/null 2>&1; then
    printf '%s\n' 'OpenSSL is required to sign release artifacts' >&2
    exit 1
fi
if ! command -v tar >/dev/null 2>&1; then
    printf '%s\n' 'tar is required to package the release source tree' >&2
    exit 1
fi

rm -rf "${OUT}"
mkdir -p "${OUT}"

# Darwin binaries must be produced on a macOS host: the Keychain adapter needs
# cgo and the macOS SDK, and a cgo-free darwin build ships with no credential
# backend at all. The release workflow builds them in a macos job and hands them
# over here; --skip-existing then builds only the targets still missing.
if [[ -n "${PREBUILT}" ]]; then
    if [[ ! -d "${PREBUILT}" ]]; then
        printf 'prebuilt directory does not exist: %s\n' "${PREBUILT}" >&2
        exit 1
    fi
    staged=0
    for path in "${PREBUILT}"/vrooli_*; do
        [[ -e "${path}" ]] || continue
        cp "${path}" "${OUT}/"
        staged=$((staged + 1))
    done
    if [[ "${staged}" -eq 0 ]]; then
        printf 'prebuilt directory %s contains no vrooli_* binaries\n' "${PREBUILT}" >&2
        exit 1
    fi
    printf 'Adopted %s prebuilt release files from %s\n' "${staged}" "${PREBUILT}"
fi
go run "${ROOT}/cmd/vrooli-dist" --root "${ROOT}" --all --skip-existing --out-dir "${OUT}" --version "${VERSION}"

# Prebuilt binaries bypass the build-time check inside BuildDistribution, so the
# guarantee is re-asserted here against whatever actually landed in the release.
for path in "${OUT}"/vrooli_darwin_*; do
    [[ -f "${path}" ]] || continue
    case "${path}" in *.fp) continue ;; esac
    if ! go version -m "${path}" | grep -q 'CGO_ENABLED=1'; then
        printf '%s was built without cgo, so it has no macOS Keychain adapter; build darwin targets on a macOS runner\n' "$(basename "${path}")" >&2
        exit 1
    fi
done
# Resource artifacts are built by the native release tool, once, for every
# manifest that declares a prebuilt distribution. They are included in
# SHA256SUMS and the release signature below; deployed desktops never run a
# resource source build.
go run "${ROOT}/cmd/vrooli-dist" --root "${ROOT}" --resource-artifacts --out-dir "${OUT}"
cp "${ROOT}/packages/cli-core/install/platform.sh" "${OUT}/vrooli-install-lib.sh"
cp "${ROOT}/packages/cli-core/install/Platform.ps1" "${OUT}/vrooli-install-lib.ps1"
cp "${ROOT}/install/install.sh" "${OUT}/install.sh"
cp "${ROOT}/install/install.ps1" "${OUT}/install.ps1"
cp "${ROOT}/install/vrooli-release.pub" "${OUT}/vrooli-release.pub"

# Package the exact visible working tree (tracked + untracked non-ignored) under
# one Vrooli/ prefix. Tagged CI runs are clean; local dry-runs may be dirty, and
# the archive must still match the binary fingerprint they just built.
source_list_raw="${OUT}/.source-files.raw"
source_list="${OUT}/.source-files"
git -C "${ROOT}" ls-files -z -c -o --exclude-standard > "${source_list_raw}"
: > "${source_list}"
while IFS= read -r -d '' rel; do
    if [[ -e "${ROOT}/${rel}" || -L "${ROOT}/${rel}" ]]; then
        printf '%s\0' "${rel}" >> "${source_list}"
    fi
done < "${source_list_raw}"
if tar --help 2>&1 | grep -q -- '--transform'; then
    tar -C "${ROOT}" --ignore-failed-read --null --files-from="${source_list}" --transform='s,^,Vrooli/,' -czf "${OUT}/vrooli-source.tar.gz"
else
    # BSD tar (macOS) spells GNU tar's --transform as -s.
    tar -C "${ROOT}" --null --files-from="${source_list}" -s ',^,Vrooli/,' -czf "${OUT}/vrooli-source.tar.gz"
fi
rm -f "${source_list_raw}" "${source_list}"

derived_public_key="${OUT}/.derived-release-public.pem"
openssl pkey -in "${SIGNING_KEY}" -pubout -out "${derived_public_key}" >/dev/null 2>&1
if ! cmp -s "${derived_public_key}" "${ROOT}/install/vrooli-release.pub"; then
    printf '%s\n' 'release signing key does not match install/vrooli-release.pub' >&2
    exit 1
fi
rm -f "${derived_public_key}"

# shellcheck source=../packages/cli-core/install/platform.sh
source "${ROOT}/packages/cli-core/install/platform.sh"
manifest="${OUT}/SHA256SUMS"
: > "${manifest}"
for path in "${OUT}"/*; do
    name=$(basename "${path}")
    [[ "${name}" == SHA256SUMS || "${name}" == SHA256SUMS.sig ]] && continue
    printf '%s  %s\n' "$(vrooli_sha256 "${path}")" "${name}" >> "${manifest}"
done

openssl dgst -sha256 -sign "${SIGNING_KEY}" "${manifest}" | openssl base64 -A > "${OUT}/SHA256SUMS.sig"
vrooli_verify_signature "${manifest}" "${OUT}/SHA256SUMS.sig" "${ROOT}/install/vrooli-release.pub"

expected_targets=$(go run "${ROOT}/cmd/vrooli-dist" --matrix-json | tr -cd '{' | wc -c)
actual_targets=$(find "${OUT}" -maxdepth 1 -type f \( -name 'vrooli_linux_*' -o -name 'vrooli_darwin_*' -o -name 'vrooli_windows_*.exe' \) ! -name '*.fp' | wc -l)
if [[ "${actual_targets}" -ne "${expected_targets}" ]]; then
    printf 'release contains %s platform binaries, want %s\n' "${actual_targets}" "${expected_targets}" >&2
    exit 1
fi
printf 'Packaged %s signed Vrooli release assets in %s\n' "${actual_targets}" "${OUT}"
