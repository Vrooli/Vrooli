#!/bin/bash

# Transition Effects Workflow Example
# This example demonstrates scene transition management in OBS Studio

# Start OBS Studio if not running
vrooli resource obs-studio manage start --wait

echo "=== Transition Effects Management Example ==="

# 1. List all available transitions
echo -e "\n📋 Available Transitions:"
vrooli resource obs-studio transitions list

# 2. Show current transition settings
echo -e "\n⚙️ Current Transition:"
vrooli resource obs-studio transitions current

# 3. Test different transition types
echo -e "\n🎬 Testing Transition Types:"

# Cut transition (instant)
echo -e "\n  Testing: Cut (instant)"
vrooli resource obs-studio transitions set cut
vrooli resource obs-studio transitions preview "Scene 2"
sleep 2

# Fade transition (smooth)
echo -e "\n  Testing: Fade (500ms)"
vrooli resource obs-studio transitions set fade --duration 500
vrooli resource obs-studio transitions preview "Scene 1"
sleep 2

# Swipe transition (directional)
echo -e "\n  Testing: Swipe (left to right)"
vrooli resource obs-studio transitions set swipe --duration 750
vrooli resource obs-studio transitions configure --direction left
vrooli resource obs-studio transitions preview "Scene 2"
sleep 2

# Slide transition
echo -e "\n  Testing: Slide (top to bottom)"
vrooli resource obs-studio transitions set slide --duration 600
vrooli resource obs-studio transitions configure --direction down
vrooli resource obs-studio transitions preview "Scene 1"
sleep 2

# 4. Configure fade to color transition
echo -e "\n🎨 Configuring Fade to Color:"
vrooli resource obs-studio transitions set fade-color --duration 400
vrooli resource obs-studio transitions configure --color "#1a1a1a"
echo "  ✓ Fade to dark gray configured"

# 5. Set up stinger transition
echo -e "\n⚡ Setting Up Stinger Transition:"

# Configure stinger with video file
vrooli resource obs-studio transitions stinger \
    --video "/path/to/stinger.mp4" \
    --point 250 \
    --audio-monitoring \
    --audio-fade
echo "  ✓ Stinger transition configured"

# 6. Create custom transitions
echo -e "\n✨ Creating Custom Transitions:"

# Quick fade for fast switches
vrooli resource obs-studio transitions custom create \
    --name "Quick Fade" \
    --base fade
vrooli resource obs-studio transitions duration 150
echo "  ✓ Quick Fade (150ms) created"

# Slow fade for dramatic effect
vrooli resource obs-studio transitions custom create \
    --name "Dramatic Fade" \
    --base fade
vrooli resource obs-studio transitions duration 2000
echo "  ✓ Dramatic Fade (2000ms) created"

# Logo wipe transition
vrooli resource obs-studio transitions custom create \
    --name "Logo Wipe" \
    --base luma-wipe
vrooli resource obs-studio transitions configure \
    --luma-image "/path/to/logo-mask.png" \
    --softness 20
echo "  ✓ Logo Wipe created"

# 7. Configure luma wipe
echo -e "\n🌊 Configuring Luma Wipe:"
vrooli resource obs-studio transitions set luma-wipe --duration 800
vrooli resource obs-studio transitions configure \
    --luma-image "/path/to/gradient.png" \
    --softness 50 \
    --invert
echo "  ✓ Luma wipe configured with gradient"

# 8. Manage transition library
echo -e "\n📚 Managing Transition Library:"

# List saved transitions
vrooli resource obs-studio transitions library list

# Export custom transition
vrooli resource obs-studio transitions library export "Quick Fade" \
    "/tmp/quick-fade-transition.json"
echo "  ✓ Exported Quick Fade transition"

# Share transition (generates share link)
vrooli resource obs-studio transitions library share "Logo Wipe"
echo "  ✓ Shared Logo Wipe transition"

# 9. Test transition timing
echo -e "\n⏱️ Testing Transition Durations:"

durations=(100 250 500 1000 1500)
for duration in "${durations[@]}"; do
    echo "  Testing ${duration}ms fade"
    vrooli resource obs-studio transitions set fade
    vrooli resource obs-studio transitions duration "$duration"
    vrooli resource obs-studio transitions preview "Scene 2"
    sleep 2
done

# 10. Configure studio mode transitions
echo -e "\n🎚️ Studio Mode Configuration:"

# Set transition for studio mode
vrooli resource obs-studio transitions set fade --duration 300
echo "  ✓ Studio mode transition set"

# Configure switch point (when to switch during transition)
vrooli resource obs-studio transitions configure --switch-point 150
echo "  ✓ Switch point set to 50% (150ms of 300ms)"

# 11. Final configuration summary
echo -e "\n✅ Final Transition Configuration:"
vrooli resource obs-studio transitions current

echo -e "\n🎉 Transition effects configuration complete!"
echo "Your transitions are now configured with:"
echo "  • Multiple transition types tested"
echo "  • Custom transitions created"
echo "  • Stinger transition ready"
echo "  • Luma wipe effects configured"
echo "  • Transition library managed"