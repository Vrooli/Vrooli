#!/bin/bash
# Example: Managing video and audio sources in OBS Studio

set -euo pipefail

echo "🎥 OBS Studio Source Management Example"
echo "======================================="
echo ""

# Check if OBS Studio is running
if ! vrooli resource obs-studio status | grep -q "Running: Yes"; then
    echo "Starting OBS Studio..."
    vrooli resource obs-studio manage start --wait
fi

echo "1. Listing available devices..."
echo ""
echo "Available cameras:"
vrooli resource obs-studio sources cameras
echo ""
echo "Available audio devices:"
vrooli resource obs-studio sources audio
echo ""

echo "2. Adding various source types..."
echo ""

# Screen capture
echo "Adding screen capture source..."
vrooli resource obs-studio sources add \
    --name "primary-screen" \
    --type screen \
    --settings '{"monitor": 0, "capture_cursor": true}'

# Window capture for specific application
echo "Adding window capture source..."
vrooli resource obs-studio sources add \
    --name "browser-window" \
    --type window \
    --settings '{"window": "Google Chrome", "capture_cursor": false}'

# Camera source
echo "Adding webcam source..."
vrooli resource obs-studio sources add \
    --name "face-cam" \
    --type camera \
    --settings '{"device_id": "default", "resolution": "1920x1080", "fps": 30}'

# Image overlay
echo "Adding logo overlay..."
vrooli resource obs-studio sources add \
    --name "logo" \
    --type image \
    --settings '{"file": "/path/to/logo.png", "unload_when_not_showing": true}'

# Browser source for web content
echo "Adding browser source..."
vrooli resource obs-studio sources add \
    --name "alerts" \
    --type browser \
    --settings '{"url": "https://streamlabs.com/alert-box", "width": 800, "height": 600}'

# Text source for titles
echo "Adding text overlay..."
vrooli resource obs-studio sources add \
    --name "title-text" \
    --type text \
    --settings '{
        "text": "Welcome to the Stream!",
        "font": {
            "face": "Arial",
            "size": 72,
            "style": "Bold"
        },
        "color": "#FFFFFF",
        "outline": true,
        "outline_color": "#000000",
        "outline_size": 3
    }'

# Media source for video playback
echo "Adding video intro..."
vrooli resource obs-studio sources add \
    --name "intro-video" \
    --type media \
    --settings '{
        "local_file": "/path/to/intro.mp4",
        "looping": false,
        "restart_on_activate": true,
        "hw_decode": true
    }'

echo ""
echo "3. Listing all configured sources..."
vrooli resource obs-studio sources list

echo ""
echo "4. Configuring source properties..."

# Update webcam settings
echo "Updating webcam to 60fps..."
vrooli resource obs-studio sources configure \
    --name "face-cam" \
    --property "fps" \
    --value "60"

# Update text content
echo "Updating text overlay..."
vrooli resource obs-studio sources configure \
    --name "title-text" \
    --property "text" \
    --value "Live Now - Tech Tutorial"

echo ""
echo "5. Managing source visibility..."

# Hide logo initially
echo "Hiding logo source..."
vrooli resource obs-studio sources visibility \
    --name "logo" \
    --visible false

# Show webcam
echo "Showing webcam source..."
vrooli resource obs-studio sources visibility \
    --name "face-cam" \
    --visible true

echo ""
echo "6. Creating a scene with multiple sources..."
cat > /tmp/tutorial-scene.json <<EOF
{
    "name": "Tutorial Scene",
    "sources": [
        {
            "name": "primary-screen",
            "visible": true,
            "position": {"x": 0, "y": 0},
            "scale": {"x": 1.0, "y": 1.0},
            "order": 0
        },
        {
            "name": "face-cam",
            "visible": true,
            "position": {"x": 1440, "y": 760},
            "scale": {"x": 0.25, "y": 0.25},
            "order": 1
        },
        {
            "name": "title-text",
            "visible": true,
            "position": {"x": 50, "y": 50},
            "order": 2
        },
        {
            "name": "logo",
            "visible": true,
            "position": {"x": 1700, "y": 50},
            "scale": {"x": 0.3, "y": 0.3},
            "order": 3
        }
    ]
}
EOF

vrooli resource obs-studio content add --file /tmp/tutorial-scene.json --name tutorial-scene
echo "✅ Tutorial scene created"

echo ""
echo "7. Previewing source configuration..."
echo "Webcam configuration:"
vrooli resource obs-studio sources preview --name "face-cam"

echo ""
echo "8. List sources in JSON format..."
vrooli resource obs-studio sources list --format json | jq '.'

echo ""
echo "============================================"
echo "✅ Source management example complete!"
echo ""
echo "Tips:"
echo "• Use 'sources cameras' to discover available camera devices"
echo "• Use 'sources audio' to discover audio input/output devices"
echo "• Configure each source with appropriate settings for your use case"
echo "• Manage visibility to show/hide sources dynamically"
echo "• Preview sources to check their configuration"
echo ""
echo "Common source types:"
echo "• camera - Video capture devices"
echo "• screen - Display/monitor capture"
echo "• window - Specific application window"
echo "• image - Static images (logos, overlays)"
echo "• media - Video/audio files"
echo "• browser - Web content via embedded browser"
echo "• text - Dynamic text overlays"

# Clean up
rm -f /tmp/tutorial-scene.json