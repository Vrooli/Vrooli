#!/usr/bin/env bash
set -euo pipefail

BUILD_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${BUILD_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${BUILD_DIR}/../utils/var.sh"

# Zips a folder into a tarball
zip::folder() {
    local folder="$1"
    local outfile="$2"
    log::info "Zipping contents of folder $folder into $outfile..."
    tar -czf "$outfile" -C "$folder" .
    log::success "Zipped contents of folder $folder into $outfile"
}

# Unzips a tarball into a folder
zip::unzip() {
    local infile="$1"
    local outdir="$2"
    log::info "Unzipping $infile into $outdir..."
    tar -xzf "$infile" -C "$outdir"
    log::success "Unzipped $infile into $outdir"
}

zip::compress() {
    local infile="$1"
    local outfile="$2"
    log::info "Compressing $infile into $outfile..."
    gzip -f "$infile"
    log::success "Compressed $infile into $outfile"
}

zip::decompress() {
    local infile="$1"
    local outdir="$2"
    log::info "Decompressing $infile into $outdir..."
    tar -xzf "$infile" -C "$outdir" --strip-components=1
    log::success "Decompressed $infile into $outdir"
}

zip::build_artifacts() {
    local outdir="$1"
    log::info "Copy build artifacts to $outdir..."
  
    mkdir -p "$outdir"
    # Copy (not zip - that happens later) each package's dist folder
    for pkg in "${var_PACKAGES_DIR}/"*; do
        if [ -d "$pkg/dist" ]; then
            log::info "Copying $(basename "$pkg") distribution..."
            cp -r "$pkg/dist" "$outdir/$(basename "$pkg")-dist"
        fi
    done
}

zip::undo_build_artifacts() {
    local artifacts_dir="$1"
    for pkg in "${var_PACKAGES_DIR}/"*; do
        # Remove existing dist folder
        rm -rf "$pkg/dist"
        # Copy dist folder from artifacts directory
        if [ -d "$artifacts_dir/$(basename "$pkg")-dist" ]; then
            log::info "Copying $(basename "$pkg") distribution..."
            cp -r "$artifacts_dir/$(basename "$pkg")-dist" "$pkg/dist"
        fi
    done
}

zip::copy_project_files() {
    local folder="$1"
    local outdir="$2"
    log::info "Copying project files from $folder to $outdir..."

    mkdir -p "$outdir"
    cp "${folder}/package.json" "$outdir"
    cp "${folder}/pnpm-lock.yaml" "$outdir"
    cp "${folder}/pnpm-workspace.yaml" "$outdir"
}

zip::copy_project() {
    local outdir="$1"
    mkdir -p "$outdir"
    log::header "Preparing project (minus Docker/Kubernetes files) for deployment..."
    zip::copy_project_files "$var_ROOT_DIR" "$outdir"
    zip::build_artifacts "$outdir"
    log::success "Project artifacts have been copies to $outdir"
}

zip::artifacts() {
    local folder="$1"
    local outdir="$2"
    log::info "Creating compressed build artifact bundle ${outdir}/artifacts.zip.gz..."
    mkdir -p "${outdir}"
    zip::folder "${folder}" "${outdir}/artifacts.zip.gz"
    log::success "Created tarball: ${outdir}/artifacts.zip.gz"
}

# Decompresses the artifacts.zip.gz file into the given directory
zip::unzip_artifacts() {
    local infile="$1"
    local outdir="$2"
    log::info "Decompressing $infile into $outdir..."
    tar -xzf "$infile" -C "$outdir" --strip-components=1
    log::success "Decompressed $infile into $outdir"
}

zip::load_artifacts() {
    local infile="$1"
    log::info "Loading artifacts from $infile into $var_ROOT_DIR..."
    zip::copy_project_files "$infile" "$var_ROOT_DIR"
    zip::undo_build_artifacts "$infile"
    log::success "Loaded artifacts from $infile into $var_ROOT_DIR"
}