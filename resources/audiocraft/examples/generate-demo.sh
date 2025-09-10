#!/usr/bin/env bash
################################################################################
# AudioCraft Demo Script
# Example usage of AudioCraft generation capabilities
################################################################################
set -euo pipefail

# Configuration
AUDIOCRAFT_PORT="${AUDIOCRAFT_PORT:-7862}"
OUTPUT_DIR="${OUTPUT_DIR:-./output}"

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "🎵 AudioCraft Generation Demo"
echo "============================"

# Function to generate music
generate_music() {
    local prompt="$1"
    local duration="$2"
    local output_file="$3"
    
    echo "Generating: $prompt (${duration}s)..."
    
    curl -X POST "http://localhost:${AUDIOCRAFT_PORT}/api/generate/music" \
        -H "Content-Type: application/json" \
        -d "{\"prompt\": \"$prompt\", \"duration\": $duration}" \
        -o "$output_file" \
        -s
    
    if [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
        echo "✅ Saved to: $output_file"
    else
        echo "❌ Generation failed"
    fi
}

# Function to generate sound effects
generate_sound() {
    local prompt="$1"
    local duration="$2"
    local output_file="$3"
    
    echo "Generating sound: $prompt (${duration}s)..."
    
    curl -X POST "http://localhost:${AUDIOCRAFT_PORT}/api/generate/sound" \
        -H "Content-Type: application/json" \
        -d "{\"prompt\": \"$prompt\", \"duration\": $duration}" \
        -o "$output_file" \
        -s
    
    if [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
        echo "✅ Saved to: $output_file"
    else
        echo "⚠️  Sound generation might not be available"
    fi
}

# Check if service is running
echo "Checking AudioCraft service..."
if curl -sf "http://localhost:${AUDIOCRAFT_PORT}/health" > /dev/null 2>&1; then
    echo "✅ Service is running"
else
    echo "❌ Service not running. Start with: resource-audiocraft manage start"
    exit 1
fi

# Generate various examples
echo ""
echo "Generating demo audio..."
echo "-----------------------"

# Music examples
generate_music "upbeat electronic dance music" 10 "$OUTPUT_DIR/edm.wav"
generate_music "peaceful piano melody with strings" 15 "$OUTPUT_DIR/piano.wav"
generate_music "epic orchestral movie soundtrack" 20 "$OUTPUT_DIR/orchestral.wav"
generate_music "jazz fusion with saxophone solo" 15 "$OUTPUT_DIR/jazz.wav"

# Sound effect examples
generate_sound "thunderstorm with heavy rain" 5 "$OUTPUT_DIR/storm.wav"
generate_sound "busy coffee shop ambience" 10 "$OUTPUT_DIR/coffeeshop.wav"
generate_sound "ocean waves on beach" 8 "$OUTPUT_DIR/ocean.wav"
generate_sound "forest birds chirping" 6 "$OUTPUT_DIR/forest.wav"

echo ""
echo "============================"
echo "✅ Demo complete! Check the $OUTPUT_DIR directory for generated audio files."
echo ""
echo "To play audio files (requires ffplay):"
echo "  ffplay $OUTPUT_DIR/edm.wav"