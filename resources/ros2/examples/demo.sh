#!/usr/bin/env bash

# ROS2 Resource - Interactive Demo

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"

echo "==================================="
echo "     ROS2 Resource Demo"
echo "==================================="
echo ""

# Check if ROS2 is running
if ! curl -sf "http://localhost:${ROS2_PORT}/health" &>/dev/null; then
    echo "Starting ROS2 services..."
    vrooli resource ros2 manage start --wait || {
        echo "Failed to start ROS2. Please run: vrooli resource ros2 manage install"
        exit 1
    }
fi

echo "1. Health Check"
echo "---------------"
curl -s "http://localhost:${ROS2_PORT}/health" | python3 -m json.tool
echo ""

echo "2. Launching Demo Nodes"
echo "-----------------------"
# Launch talker node
response=$(curl -sf -X POST "http://localhost:${ROS2_PORT}/nodes/launch?node_name=demo_talker&package=demo_nodes")
echo "Talker: $response"

# Launch listener node  
response=$(curl -sf -X POST "http://localhost:${ROS2_PORT}/nodes/launch?node_name=demo_listener&package=demo_nodes")
echo "Listener: $response"
echo ""

echo "3. Active Nodes"
echo "---------------"
curl -s "http://localhost:${ROS2_PORT}/nodes" | python3 -m json.tool
echo ""

echo "4. Publishing Message"
echo "--------------------"
message='{"data": "Hello from ROS2 Demo!", "timestamp": "'$(date -Iseconds)'"}'
response=$(curl -sf -X POST -H "Content-Type: application/json" \
    -d "${message}" \
    "http://localhost:${ROS2_PORT}/topics/demo_topic/publish")
echo "Published: $response"
echo ""

echo "5. Active Topics"
echo "----------------"
curl -s "http://localhost:${ROS2_PORT}/topics" | python3 -m json.tool
echo ""

echo "6. DDS Statistics"
echo "-----------------"
if command -v dds_get_stats &>/dev/null; then
    source "${SCRIPT_DIR}/lib/dds.sh"
    dds_get_stats json | python3 -m json.tool
else
    echo "DDS statistics not available"
fi
echo ""

echo "==================================="
echo "Demo complete! ROS2 is operational."
echo "==================================="
echo ""
echo "Try these commands:"
echo "  vrooli resource ros2 status"
echo "  vrooli resource ros2 content list-nodes"
echo "  vrooli resource ros2 content list-topics"
echo "  vrooli resource ros2 logs"