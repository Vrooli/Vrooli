#!/bin/bash

# Advanced Audio Mixer Workflow Example
# This example demonstrates comprehensive audio control in OBS Studio

# Start OBS Studio if not running
vrooli resource obs-studio manage start --wait

echo "=== Advanced Audio Mixer Control Example ==="

# 1. Show current audio mixer status
echo -e "\n📊 Current Audio Status:"
vrooli resource obs-studio audio status

# 2. List all audio sources
echo -e "\n📋 Available Audio Sources:"
vrooli resource obs-studio audio list

# 3. Configure microphone with professional settings
echo -e "\n🎤 Configuring Microphone:"

# Set optimal volume
vrooli resource obs-studio audio volume Microphone --level 85
echo "  ✓ Volume set to 85%"

# Enable noise suppression
vrooli resource obs-studio audio noise Microphone enable high
echo "  ✓ Noise suppression enabled (high)"

# Add compression for consistent levels
vrooli resource obs-studio audio compressor Microphone \
    --threshold -20 \
    --ratio 4:1 \
    --attack 10 \
    --release 100
echo "  ✓ Compressor configured"

# Apply EQ for voice clarity
vrooli resource obs-studio audio eq Microphone --preset voice
echo "  ✓ EQ optimized for voice"

# 4. Configure desktop audio
echo -e "\n🔊 Configuring Desktop Audio:"

# Lower desktop audio to avoid overpowering mic
vrooli resource obs-studio audio volume "Desktop Audio" --level 60
echo "  ✓ Volume set to 60%"

# Set up monitoring
vrooli resource obs-studio audio monitor "Desktop Audio" --mode monitor-and-output
echo "  ✓ Monitoring enabled"

# 5. Set up auto-ducking for background music
echo -e "\n🎵 Configuring Auto-Ducking:"

# Enable ducking to lower music when speaking
vrooli resource obs-studio audio ducking enable \
    --threshold -30 \
    --ratio 3:1
echo "  ✓ Auto-ducking enabled"

# 6. Configure audio sync for capture card
echo -e "\n⏱️ Configuring Audio Sync:"

# Add 50ms offset to sync with video
vrooli resource obs-studio audio sync "Capture Card Audio" --offset 50
echo "  ✓ Audio sync adjusted (+50ms)"

# 7. Create audio filter chain
echo -e "\n🔧 Adding Audio Filters:"

# Add multiple filters to microphone
vrooli resource obs-studio audio filter Microphone add --type noise-suppression
vrooli resource obs-studio audio filter Microphone add --type compressor
vrooli resource obs-studio audio filter Microphone add --type limiter
echo "  ✓ Filter chain applied"

# 8. Test different monitoring modes
echo -e "\n🎧 Testing Monitoring Modes:"

# Cycle through monitoring modes
for mode in none monitor-only monitor-and-output; do
    echo "  Testing mode: $mode"
    vrooli resource obs-studio audio monitor Microphone --mode "$mode"
    sleep 1
done

# 9. Balance stereo sources
echo -e "\n⚖️ Adjusting Stereo Balance:"

# Center all sources
vrooli resource obs-studio audio balance Microphone --balance 0.0
vrooli resource obs-studio audio balance "Desktop Audio" --balance 0.0
echo "  ✓ Stereo balance centered"

# 10. Final status check
echo -e "\n✅ Final Audio Configuration:"
vrooli resource obs-studio audio status

echo -e "\n🎉 Audio mixer configuration complete!"
echo "Your audio is now professionally configured with:"
echo "  • Noise suppression and compression on microphone"
echo "  • Auto-ducking for background music"
echo "  • Proper monitoring setup"
echo "  • Optimized EQ and filters"