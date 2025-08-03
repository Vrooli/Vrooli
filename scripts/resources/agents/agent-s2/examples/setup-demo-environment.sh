#!/usr/bin/env bash
# Setup Demo Environment for Agent S2
# Launches common GUI applications for testing and demonstration

API_URL="http://localhost:4113"

echo "==================================="
echo "Agent S2 Demo Environment Setup"
echo "==================================="

# Check if Agent S2 is running
echo -n "Checking Agent S2 health... "
if curl -s -f "$API_URL/health" > /dev/null 2>&1; then
    echo "✅ Healthy"
else
    echo "❌ Not running"
    echo "Start Agent S2 with: ./manage.sh --action start"
    exit 1
fi

echo
echo "🚀 Launching demo applications in Agent S2..."

# Launch terminal
echo "📱 Starting terminal..."
docker exec agent-s2 sh -c 'DISPLAY=:99 xterm -geometry 80x24+50+50 -title "Agent S2 Demo Terminal" -e "echo \"Welcome to Agent S2 Demo!\"; echo \"This terminal is running inside the virtual display.\"; echo \"You can interact with it using Agent S2 automation.\"; bash" &' 2>/dev/null

# Launch text editor
echo "📝 Starting text editor..."
docker exec agent-s2 sh -c 'echo "Welcome to Agent S2 Demo!

This text editor is running inside the Agent S2 virtual display.
You can use Agent S2 to:
- Take screenshots of this content
- Click on buttons and menus
- Type text automatically
- Control the mouse cursor

Try the automation examples!" > /tmp/demo.txt'

docker exec agent-s2 sh -c 'DISPLAY=:99 mousepad /tmp/demo.txt &' 2>/dev/null

# Wait for applications to start
echo "⏳ Waiting for applications to initialize..."
sleep 3

# Take a demo screenshot
echo "📸 Taking demonstration screenshot..."
RESPONSE=$(curl -s -X POST "$API_URL/screenshot?format=png&quality=95")

if [ $? -eq 0 ]; then
    echo "✅ Screenshot captured successfully"
    
    # Save base64 data to file (if jq is available)
    if command -v jq > /dev/null 2>&1; then
        IMAGE_DATA=$(echo "$RESPONSE" | jq -r '.data // empty' | sed 's/^data:image\/[^;]*;base64,//')
        if [ -n "$IMAGE_DATA" ]; then
            # Ensure output directory exists
            mkdir -p ../data/test-outputs/screenshots
            echo "$IMAGE_DATA" | base64 -d > ../data/test-outputs/screenshots/demo-environment.png
            echo "   Saved to: ../data/test-outputs/screenshots/demo-environment.png"
        fi
    fi
else
    echo "❌ Screenshot failed"
fi

echo
echo "🎯 Demo environment ready!"
echo "   Applications running:"
echo "   • Terminal (xterm) - for command-line demos"
echo "   • Text Editor (mousepad) - for text editing demos"
echo
echo "   VNC Access:"
echo "   • URL: vnc://localhost:5900"
echo "   • Password: agents2vnc"
echo
echo "   Try the automation examples:"
echo "   • ./screenshot-demo.sh"
echo "   • python3 basic-automation.py"
echo "   • python3 core/screenshot.py"
echo
echo "✨ Agent S2 is ready for automation demonstrations!"