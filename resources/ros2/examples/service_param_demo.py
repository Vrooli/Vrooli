#!/usr/bin/env python3
"""
ROS2 Service and Parameter Demo
Demonstrates the new service client/server and parameter management features
"""

import sys
import time
import json
import requests
from typing import Dict, Any

# Configuration
API_HOST = "localhost"
API_PORT = 11501
BASE_URL = f"http://{API_HOST}:{API_PORT}"

def print_header(text: str):
    """Print a formatted header"""
    print(f"\n{'='*60}")
    print(f"  {text}")
    print(f"{'='*60}\n")

def api_call(method: str, endpoint: str, data: Dict[str, Any] = None) -> Dict[str, Any]:
    """Make an API call and return the response"""
    url = f"{BASE_URL}{endpoint}"
    try:
        if method == "GET":
            response = requests.get(url, timeout=5)
        elif method == "POST":
            response = requests.post(url, json=data, timeout=5)
        elif method == "PUT":
            response = requests.put(url, json=data, timeout=5)
        elif method == "DELETE":
            response = requests.delete(url, timeout=5)
        else:
            raise ValueError(f"Unknown method: {method}")
        
        return response.json()
    except Exception as e:
        return {"error": str(e)}

def demo_services():
    """Demonstrate service creation and calling"""
    print_header("SERVICE DEMONSTRATION")
    
    # 1. List existing services
    print("1. Listing existing services...")
    services = api_call("GET", "/services")
    print(f"   Found {services.get('count', 0)} services")
    
    # 2. Create a new service
    print("\n2. Creating a new service 'robot_status'...")
    result = api_call("POST", "/services/create", {
        "name": "robot_status",
        "type": "std_srvs/srv/Trigger"
    })
    print(f"   {result.get('message', 'Service created')}")
    
    # 3. Call the service
    print("\n3. Calling the 'robot_status' service...")
    response = api_call("POST", "/services/robot_status/call", {})
    print(f"   Response: {response.get('response', {})}")
    
    # 4. Create another service with custom handler
    print("\n4. Creating 'compute_trajectory' service...")
    result = api_call("POST", "/services/create", {
        "name": "compute_trajectory",
        "type": "geometry_msgs/srv/GetPlan",
        "handler": "trajectory_planner"
    })
    print(f"   {result.get('message', 'Service created')}")
    
    # 5. List all services again
    print("\n5. Listing all services...")
    services = api_call("GET", "/services")
    print(f"   Total services: {services.get('count', 0)}")
    if 'services' in services:
        for service_name, service_type in services['services'].items():
            print(f"   - {service_name}: {service_type}")

def demo_parameters():
    """Demonstrate parameter management"""
    print_header("PARAMETER DEMONSTRATION")
    
    # 1. Set parameters for navigation node
    print("1. Setting parameters for 'navigation_controller'...")
    params = {
        "max_velocity": 2.0,
        "min_velocity": 0.1,
        "acceleration_limit": 1.5,
        "obstacle_distance": 0.5,
        "goal_tolerance": 0.1,
        "use_dynamic_reconfigure": True
    }
    result = api_call("PUT", "/params/navigation_controller", params)
    print(f"   Set {len(result.get('parameters_set', []))} parameters")
    
    # 2. Get parameters back
    print("\n2. Retrieving parameters for 'navigation_controller'...")
    params_response = api_call("GET", "/params/navigation_controller")
    retrieved_params = params_response.get('parameters', {})
    for param, value in retrieved_params.items():
        print(f"   - {param}: {value}")
    
    # 3. Set parameters for sensor fusion
    print("\n3. Setting parameters for 'sensor_fusion'...")
    sensor_params = {
        "lidar_weight": 0.7,
        "camera_weight": 0.3,
        "imu_weight": 0.5,
        "fusion_rate": 30,
        "outlier_rejection": True
    }
    result = api_call("PUT", "/params/sensor_fusion", sensor_params)
    print(f"   Set {len(result.get('parameters_set', []))} parameters")
    
    # 4. List all parameters across nodes
    print("\n4. Listing all parameters across all nodes...")
    all_params = api_call("GET", "/params/list")
    print(f"   Total nodes with parameters: {all_params.get('total_nodes', 0)}")
    for node, params in all_params.get('parameters', {}).items():
        print(f"   - {node}: {len(params)} parameters")
    
    # 5. Save parameters to persistent storage
    print("\n5. Saving all parameters to persistent storage...")
    result = api_call("POST", "/params/save")
    print(f"   {result.get('message', 'Parameters saved')}")
    
    # 6. Simulate loading after restart
    print("\n6. Simulating parameter recovery after restart...")
    print("   (In real scenario, this would restore parameters after ROS2 restart)")
    result = api_call("POST", "/params/load")
    print(f"   {result.get('message', 'Parameters loaded')}")

def demo_integrated_workflow():
    """Demonstrate integrated service and parameter workflow"""
    print_header("INTEGRATED WORKFLOW: Robot Configuration")
    
    # 1. Create robot configuration service
    print("1. Creating 'configure_robot' service...")
    result = api_call("POST", "/services/create", {
        "name": "configure_robot",
        "type": "std_srvs/srv/SetBool"
    })
    print(f"   {result.get('message', 'Service created')}")
    
    # 2. Set robot configuration parameters
    print("\n2. Setting robot configuration parameters...")
    config_params = {
        "robot_name": "Vrooli-Bot-1",
        "autonomous_mode": True,
        "safety_level": 2,
        "battery_threshold": 20.0,
        "mission_type": "exploration"
    }
    result = api_call("PUT", "/params/robot_config", config_params)
    print(f"   Configured {len(result.get('parameters_set', []))} parameters")
    
    # 3. Call configuration service to apply settings
    print("\n3. Calling configuration service to apply settings...")
    response = api_call("POST", "/services/configure_robot/call", {
        "data": True  # Enable configuration
    })
    print(f"   Configuration applied: {response.get('success', False)}")
    
    # 4. Verify configuration
    print("\n4. Verifying robot configuration...")
    params = api_call("GET", "/params/robot_config")
    print("   Current configuration:")
    for param, value in params.get('parameters', {}).items():
        print(f"   - {param}: {value}")
    
    # 5. Save configuration
    print("\n5. Saving configuration for persistence...")
    result = api_call("POST", "/params/save")
    print(f"   {result.get('message', 'Configuration saved')}")

def main():
    """Main demo function"""
    print_header("ROS2 SERVICE & PARAMETER DEMO")
    print("This demo showcases the enhanced ROS2 service and parameter functionality")
    print("Make sure ROS2 is running: vrooli resource ros2 develop")
    
    # Check health first
    print("\nChecking ROS2 API health...")
    health = api_call("GET", "/health")
    if "error" in health:
        print(f"❌ Error: Cannot connect to ROS2 API at {BASE_URL}")
        print("   Make sure ROS2 is running: vrooli resource ros2 develop")
        sys.exit(1)
    
    print(f"✅ ROS2 API is healthy (status: {health.get('status', 'unknown')})")
    
    # Run demonstrations
    try:
        demo_services()
        time.sleep(1)
        
        demo_parameters()
        time.sleep(1)
        
        demo_integrated_workflow()
        
        print_header("DEMO COMPLETE")
        print("✅ Successfully demonstrated service and parameter functionality!")
        print("\nThe ROS2 resource now provides:")
        print("  • Service server creation and management")
        print("  • Service client calls with request/response")
        print("  • Parameter setting and retrieval per node")
        print("  • Parameter persistence across restarts")
        print("  • Integrated configuration workflows")
        
    except KeyboardInterrupt:
        print("\n\nDemo interrupted by user")
    except Exception as e:
        print(f"\n❌ Error during demo: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()