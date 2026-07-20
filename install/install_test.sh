#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT
RELEASE="${TMP}/release"
TOOLS="${TMP}/tools"
mkdir -p "${RELEASE}" "${TOOLS}"

cp "${ROOT}/packages/cli-core/install/platform.sh" "${RELEASE}/vrooli-install-lib.sh"
cat > "${RELEASE}/vrooli_linux_amd64" <<'FIXTURE_BINARY'
#!/bin/sh
if [ "${1:-}" = setup ]; then
    printf '%s\n' "$0 $*" > "${VROOLI_SETUP_RECORD:?VROOLI_SETUP_RECORD is required for setup fixture}"
    exit 0
fi
printf '%s\n' 'vrooli fixture is runnable'
FIXTURE_BINARY
chmod 0755 "${RELEASE}/vrooli_linux_amd64"
printf '%s\n' 'fixture-fingerprint' > "${RELEASE}/vrooli_linux_amd64.fp"
mkdir -p "${TMP}/fixture-source/Vrooli"
printf '%s\n' 'module github.com/vrooli/vrooli' > "${TMP}/fixture-source/Vrooli/go.mod"
tar -C "${TMP}/fixture-source" -czf "${RELEASE}/vrooli-source.tar.gz" Vrooli
SOURCE_DIGEST=$(sha256sum "${RELEASE}/vrooli-source.tar.gz")
SOURCE_DIGEST=${SOURCE_DIGEST%% *}

openssl genpkey -quiet -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "${TMP}/private.pem"
openssl pkey -in "${TMP}/private.pem" -pubout -out "${TMP}/public.pem"
(
    cd "${RELEASE}"
    sha256sum vrooli-install-lib.sh vrooli_linux_amd64 vrooli_linux_amd64.fp vrooli-source.tar.gz > SHA256SUMS
    openssl dgst -sha256 -sign "${TMP}/private.pem" SHA256SUMS | openssl base64 -A > SHA256SUMS.sig
)

for tool in uname curl openssl mktemp rm sed sha256sum mkdir chmod mv cat cp tar gzip tr dirname; do
    ln -s "$(command -v "${tool}")" "${TOOLS}/${tool}"
done

INSTALL_DIR="${TMP}/installed"
env \
    PATH="${TOOLS}" \
    HOME="${TMP}/home" \
    VROOLI_RELEASE_BASE_URL="file://${RELEASE}" \
    VROOLI_RELEASE_PUBLIC_KEY_FILE="${TMP}/public.pem" \
    VROOLI_INSTALL_DIR="${INSTALL_DIR}" \
    /bin/sh "${ROOT}/install/install.sh"

test "$("${INSTALL_DIR}/vrooli")" = 'vrooli fixture is runnable'
test "$(cat "${INSTALL_DIR}/vrooli.fp")" = 'fixture-fingerprint'
test -f "${TMP}/home/.vrooli/src/${SOURCE_DIGEST}/Vrooli/go.mod"
test "$(cat "${TMP}/home/.vrooli/source-root")" = "${TMP}/home/.vrooli/src/${SOURCE_DIGEST}/Vrooli"

# A child process cannot update its parent's PATH. When setup is requested in
# the same installer process, it must therefore invoke the newly installed
# binary by its absolute path rather than relying on command discovery.
RUN_SETUP_DIR="${TMP}/run-setup-installed"
RUN_SETUP_RECORD="${TMP}/run-setup.record"
env \
    PATH="${TOOLS}" \
    HOME="${TMP}/run-setup-home" \
    VROOLI_RELEASE_BASE_URL="file://${RELEASE}" \
    VROOLI_RELEASE_PUBLIC_KEY_FILE="${TMP}/public.pem" \
    VROOLI_INSTALL_DIR="${RUN_SETUP_DIR}" \
    VROOLI_RUN_SETUP=1 \
    VROOLI_SETUP_RECORD="${RUN_SETUP_RECORD}" \
    /bin/sh "${ROOT}/install/install.sh"
test "$(cat "${RUN_SETUP_RECORD}")" = "${RUN_SETUP_DIR}/vrooli setup"

# The binary freshness fingerprint deliberately covers only inputs that affect
# the vrooli binary. A release can therefore update other authenticated source
# content without changing that sidecar. Prove the source cache follows the
# source archive digest instead of silently retaining the older tree.
printf '%s\n' 'source revision two' > "${TMP}/fixture-source/Vrooli/source-revision"
tar -C "${TMP}/fixture-source" -czf "${RELEASE}/vrooli-source.tar.gz" Vrooli
SOURCE_DIGEST_V2=$(sha256sum "${RELEASE}/vrooli-source.tar.gz")
SOURCE_DIGEST_V2=${SOURCE_DIGEST_V2%% *}
(
    cd "${RELEASE}"
    sha256sum vrooli-install-lib.sh vrooli_linux_amd64 vrooli_linux_amd64.fp vrooli-source.tar.gz > SHA256SUMS
    openssl dgst -sha256 -sign "${TMP}/private.pem" SHA256SUMS | openssl base64 -A > SHA256SUMS.sig
)
env \
    PATH="${TOOLS}" \
    HOME="${TMP}/home" \
    VROOLI_RELEASE_BASE_URL="file://${RELEASE}" \
    VROOLI_RELEASE_PUBLIC_KEY_FILE="${TMP}/public.pem" \
    VROOLI_INSTALL_DIR="${INSTALL_DIR}" \
    /bin/sh "${ROOT}/install/install.sh"
test "${SOURCE_DIGEST_V2}" != "${SOURCE_DIGEST}"
test -f "${TMP}/home/.vrooli/src/${SOURCE_DIGEST_V2}/Vrooli/source-revision"
test "$(cat "${TMP}/home/.vrooli/source-root")" = "${TMP}/home/.vrooli/src/${SOURCE_DIGEST_V2}/Vrooli"

# The fresh-host contract is shell + curl. Prove the installer can obtain its
# manifest verifier through the native package manager when OpenSSL is absent.
rm -f "${TOOLS}/openssl"
cat > "${TOOLS}/id" <<'FIXTURE_ID'
#!/bin/sh
printf '%s\n' 0
FIXTURE_ID
cat > "${TOOLS}/apt-get" <<FIXTURE_APT
#!/bin/sh
if [ "\${1:-}" = update ]; then
    exit 0
fi
"$(command -v ln)" -sf "$(command -v openssl)" "${TOOLS}/openssl"
FIXTURE_APT
chmod 0755 "${TOOLS}/id" "${TOOLS}/apt-get"
BOOTSTRAP_DIR="${TMP}/bootstrap-install"
env \
    PATH="${TOOLS}" \
    HOME="${TMP}/bootstrap-home" \
    VROOLI_RELEASE_BASE_URL="file://${RELEASE}" \
    VROOLI_RELEASE_PUBLIC_KEY_FILE="${TMP}/public.pem" \
    VROOLI_INSTALL_DIR="${BOOTSTRAP_DIR}" \
    /bin/sh "${ROOT}/install/install.sh" >"${TMP}/bootstrap.out"
test "$("${BOOTSTRAP_DIR}/vrooli")" = 'vrooli fixture is runnable'
grep -q 'Installing required release verification tools: openssl' "${TMP}/bootstrap.out"
rm -f "${TOOLS}/id" "${TOOLS}/apt-get"

printf '%s\n' 'tampered' >> "${RELEASE}/vrooli_linux_amd64"
TAMPERED_DIR="${TMP}/tampered-install"
if env \
    PATH="${TOOLS}" \
    HOME="${TMP}/home" \
    VROOLI_RELEASE_BASE_URL="file://${RELEASE}" \
    VROOLI_RELEASE_PUBLIC_KEY_FILE="${TMP}/public.pem" \
    VROOLI_INSTALL_DIR="${TAMPERED_DIR}" \
    /bin/sh "${ROOT}/install/install.sh" >"${TMP}/tamper.out" 2>&1; then
    printf '%s\n' 'tampered artifact was accepted' >&2
    exit 1
fi
test ! -e "${TAMPERED_DIR}/vrooli"
grep -q 'checksum mismatch' "${TMP}/tamper.out"

printf '%s\n' 'not-a-valid-signature' > "${RELEASE}/SHA256SUMS.sig"
BAD_SIGNATURE_DIR="${TMP}/bad-signature-install"
if env \
    PATH="${TOOLS}" \
    HOME="${TMP}/home" \
    VROOLI_RELEASE_BASE_URL="file://${RELEASE}" \
    VROOLI_RELEASE_PUBLIC_KEY_FILE="${TMP}/public.pem" \
    VROOLI_INSTALL_DIR="${BAD_SIGNATURE_DIR}" \
    /bin/sh "${ROOT}/install/install.sh" >"${TMP}/signature.out" 2>&1; then
    printf '%s\n' 'invalid signature was accepted' >&2
    exit 1
fi
test ! -e "${BAD_SIGNATURE_DIR}/vrooli"
grep -q 'signature verification failed' "${TMP}/signature.out"

printf '%s\n' 'installer tests passed'
