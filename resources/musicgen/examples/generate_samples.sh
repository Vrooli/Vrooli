#!/bin/bash
# Example: Generate sample music tracks

set -euo pipefail

# Source the CLI
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/musicgen/examples"
source "${APP_ROOT}/resources/musicgen/cli.sh"

# Generate various music samples
echo "Generating sample music tracks..."

# 1. Upbeat electronic
echo "1. Generating upbeat electronic music..."
musicgen::generate "upbeat electronic dance music with synth leads" 15

# 2. Calm piano
echo "2. Generating calm piano music..."
musicgen::generate "peaceful solo piano melody in minor key" 20

# 3. Orchestral
echo "3. Generating orchestral music..."
musicgen::generate "epic orchestral cinematic theme with strings and brass" 30

# 4. Jazz
echo "4. Generating jazz music..."
musicgen::generate "smooth jazz with saxophone and double bass" 25

# 5. Ambient
echo "5. Generating ambient music..."
musicgen::generate "ambient space music with ethereal pads" 20

echo "Sample generation complete!"
echo "Check outputs in: ${MUSICGEN_OUTPUT_DIR}"