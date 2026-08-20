#!/usr/bin/env bash
set -euo pipefail

# Build one target-native sherpa adapter tree, create the same deterministic
# tree digest used by binaryfetch, and optionally sign the stage with the
# managed Vrooli release authority. This script intentionally refuses to
# cross-build cgo: a successful Go compile is not evidence of a runnable
# target artifact when the sherpa/ONNX shared libraries are target-specific.

TARGET=${TARGET:-}
ROOT=${ROOT:-}
OUT=${OUT:-}
VERSION=${VERSION:-1.13.2-vrooli.1}
SIGN=${SIGN:-1}
PUBLICATION_URL=${PUBLICATION_URL:-}
RESOURCE_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

fail() {
    printf 'sherpa native release: %s\n' "$1" >&2
    exit 1
}

[ -n "$TARGET" ] || fail 'TARGET is required'
[ -n "$ROOT" ] || fail 'ROOT is required'
[ -n "$OUT" ] || fail 'OUT is required'
[ -d "$ROOT/include" ] || fail "runtime root has no include directory: $ROOT"
[ -d "$ROOT/lib" ] || fail "runtime root has no lib directory: $ROOT"

if [ -n "$PUBLICATION_URL" ]; then
    case "$PUBLICATION_URL" in
        https://*) ;;
        *) fail 'PUBLICATION_URL must be an absolute HTTPS base URL' ;;
    esac
    case "$PUBLICATION_URL" in
        *[!A-Za-z0-9:/._~%+-]*) fail 'PUBLICATION_URL contains characters unsafe for JSON publication metadata' ;;
    esac
    PUBLICATION_URL=${PUBLICATION_URL%/}
else
    PUBLICATION_URL='<publish-artifact-url>'
fi

case "$TARGET" in
    linux-amd64|linux-arm64|macos-arm64|windows-amd64) ;;
    *) fail "unsupported target $TARGET" ;;
esac

mkdir -p "$OUT"
if [ -n "$(find "$OUT" -mindepth 1 -maxdepth 1 -print -quit)" ]; then
    fail "release stage must be empty: $OUT"
fi

artifact_name="sherpa-onnx-server_${TARGET}"
artifact="$OUT/$artifact_name"
printf 'sherpa native release: building %s on the matching host\n' "$TARGET"
make -C "$RESOURCE_ROOT" "bundle-native-$TARGET" ROOT="$ROOT" OUT="$artifact"

[ -f "$artifact/server/sherpa-onnx-server" ] || fail "bundle entrypoint is missing: $artifact/server/sherpa-onnx-server"
if [ "$TARGET" != windows-amd64 ] && [ ! -x "$artifact/server/sherpa-onnx-server" ]; then
    fail "bundle entrypoint is not executable: $artifact/server/sherpa-onnx-server"
fi

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        fail 'sha256sum or shasum is required'
    fi
}

tree_digest() {
    local tree="$1"
    local entries
    local digest_input
    entries=$(mktemp "${TMPDIR:-/tmp}/sherpa-tree-entries.XXXXXX")
    digest_input=$(mktemp "${TMPDIR:-/tmp}/sherpa-tree-input.XXXXXX")

    if [ -n "$(find "$tree" -type l -print -quit)" ]; then
        fail 'release bundle contains a symlink; package the resolved runtime files explicitly'
    fi
    if [ -n "$(find "$tree" ! -type f ! -type d -print -quit)" ]; then
        fail 'release bundle contains a non-regular, non-directory entry'
    fi

    while IFS= read -r -d '' path; do
        rel=${path#"$tree"/}
        printf '%s\t%s\n' "$rel" "$(sha256_file "$path")" >> "$entries"
    done < <(find "$tree" -type f -print0)

    : > "$digest_input"
    while IFS=$'\t' read -r rel digest; do
        [ -n "$rel" ] || continue
        printf '%s\0%s\n' "$rel" "$digest" >> "$digest_input"
    done < <(LC_ALL=C sort "$entries")
    digest=$(sha256_file "$digest_input")
    rm -f "$entries" "$digest_input"
    printf '%s\n' "$digest"
}

digest=$(tree_digest "$artifact")
provenance="sherpa-onnx-v1.13.2 upstream runtime; Vrooli adapter $VERSION; target-native build"
os=${TARGET%%-*}
arch=${TARGET#*-}
[ "$os" = macos ] && os=macos

# Managed acquisition consumes an archive, while the signed release manifest
# authenticates the extracted executable tree. Emit both from the same stage so
# publication cannot accidentally pair an archive with a different tree. Keep
# both tar metadata and gzip metadata deterministic: gzip's default header
# includes the current time, which otherwise changes the archive checksum even
# when the target tree is byte-for-byte identical.
archive="$OUT/$artifact_name.tar.gz"
archive_tar=$(mktemp "${TMPDIR:-/tmp}/sherpa-archive.XXXXXX.tar")
archive_list=''
cleanup_archive() {
    rm -f "$archive_tar"
    [ -z "$archive_list" ] || rm -f "$archive_list"
}
trap cleanup_archive EXIT
tar_help=$(tar --help 2>&1 || true)
if printf '%s\n' "$tar_help" | grep -q -- '--sort='; then
    tar --sort=name --mtime='UTC 1970-01-01' --owner=0 --group=0 --numeric-owner \
        -cf "$archive_tar" -C "$OUT" "$artifact_name"
elif printf '%s\n' "$tar_help" | grep -q -- '--null' \
    && printf '%s\n' "$tar_help" | grep -q -- '--mtime' \
    && printf '%s\n' "$tar_help" | grep -q -- '--uid' \
    && printf '%s\n' "$tar_help" | grep -q -- '--gid'; then
    # BSD tar does not provide GNU tar's --sort option. Feed it a sorted,
    # NUL-delimited complete tree and normalize the metadata it does expose.
    # If a host tar cannot provide these controls, fail closed below instead
    # of publishing an archive whose checksum changes between builds.
    archive_list=$(mktemp "${TMPDIR:-/tmp}/sherpa-archive-list.XXXXXX")
    (cd "$OUT" && find "$artifact_name" -print0 | LC_ALL=C sort -z > "$archive_list")
    tar --null --mtime='UTC 1970-01-01' --uid=0 --gid=0 --uname=root --gname=root \
        -cf "$archive_tar" -C "$OUT" -T "$archive_list"
else
    fail 'tar lacks deterministic sort or metadata controls required for release archives'
fi
gzip -n -c "$archive_tar" > "$archive"
archive_sha256=$(sha256_file "$archive")

cat > "$OUT/release-manifest.json" <<EOF
{"schema_version":"v1","artifacts":[{"name":"$artifact_name","sha256":"$digest","role":"managed-service","os":"$os","arch":"$arch","upstream_provenance":"$provenance"}]}
EOF

cat > "$OUT/publication-target.json" <<EOF
{"schema_version":"v1","resource":"sherpa-onnx","target":{"os":"$os","arch":"$arch"},"archive":"$artifact_name.tar.gz","archive_sha256":"$archive_sha256","artifact_sha256":"$digest","layout":"dir","entry_path":"server/sherpa-onnx-server","bin_path":"server/sherpa-onnx-server","archive_format":"tar.gz","publication_url":"$PUBLICATION_URL/$artifact_name.tar.gz"}
EOF

if [ "$SIGN" = 1 ]; then
    command -v vrooli >/dev/null 2>&1 || fail 'SIGN=1 requires the vrooli control-plane binary'
    vrooli release-authority sign --stage "$OUT"
else
    printf 'sherpa native release: unsigned stage (SIGN=0), digest=%s\n' "$digest"
fi

printf 'sherpa native release: staged %s\n' "$artifact"
printf 'sherpa native release: tree digest %s\n' "$digest"
printf 'sherpa native release: archive %s (sha256 %s)\n' "$archive" "$archive_sha256"
