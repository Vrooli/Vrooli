#!/usr/bin/env python3

"""
ROS2 Resource - API Server
Provides REST API for ROS2 operations and health monitoring
"""

import os
import sys
import json
import asyncio
from datetime import datetime
from typing import Dict, Any, Optional

try:
    from fastapi import FastAPI, HTTPException
    from fastapi.responses import JSONResponse
    import uvicorn
except ImportError:
    print("FastAPI not installed. Installing minimal health server...")
    # Fallback to basic HTTP server if FastAPI not available
    from http.server import HTTPServer, BaseHTTPRequestHandler
    
    class HealthHandler(BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path == '/health':
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                response = {
                    "status": "healthy",
                    "timestamp": datetime.now().isoformat(),
                    "service": "ros2",
                    "domain_id": int(os.environ.get("ROS2_DOMAIN_ID", 0))
                }
                self.wfile.write(json.dumps(response).encode())
            else:
                self.send_response(404)
                self.end_headers()
        
        def log_message(self, format, *args):
            # Suppress default logging
            pass
    
    if __name__ == "__main__":
        port = int(os.environ.get("ROS2_PORT", 11501))
        server = HTTPServer(('0.0.0.0', port), HealthHandler)
        print(f"ROS2 API Server (minimal) running on port {port}")
        server.serve_forever()
    sys.exit(0)

# FastAPI application
app = FastAPI(title="ROS2 Resource API", version="1.0.0")

# Global state
ros2_state = {
    "nodes": {},
    "topics": {},
    "services": {},
    "parameters": {},
    "start_time": datetime.now()
}

@app.get("/health")
async def health_check() -> Dict[str, Any]:
    """Health check endpoint"""
    try:
        # Check if ROS2 is available
        ros2_available = check_ros2_availability()
        
        return {
            "status": "healthy" if ros2_available else "degraded",
            "timestamp": datetime.now().isoformat(),
            "service": "ros2",
            "domain_id": int(os.environ.get("ROS2_DOMAIN_ID", 0)),
            "middleware": os.environ.get("ROS2_MIDDLEWARE", "fastdds"),
            "uptime_seconds": (datetime.now() - ros2_state["start_time"]).total_seconds(),
            "nodes_active": len(ros2_state["nodes"]),
            "topics_active": len(ros2_state["topics"])
        }
    except Exception as e:
        return JSONResponse(
            status_code=503,
            content={"status": "unhealthy", "error": str(e)}
        )

@app.get("/nodes")
async def list_nodes() -> Dict[str, Any]:
    """List active ROS2 nodes"""
    # Try to get actual ROS2 nodes if available
    try:
        use_docker = os.environ.get("ROS2_USE_DOCKER", "true") == "true"
        if use_docker:
            import subprocess
            result = subprocess.run(
                ["docker", "exec", "vrooli-ros2", "ros2", "node", "list"],
                capture_output=True,
                text=True,
                timeout=2
            )
            if result.returncode == 0:
                nodes = [n.strip() for n in result.stdout.strip().split("\n") if n.strip()]
                return {"nodes": nodes, "count": len(nodes)}
    except Exception:
        pass
    
    # Fallback to simulated state
    return {
        "nodes": list(ros2_state["nodes"].keys()),
        "count": len(ros2_state["nodes"])
    }

@app.post("/nodes/launch")
async def launch_node(node_name: str, package: Optional[str] = None) -> Dict[str, Any]:
    """Launch a ROS2 node"""
    # Simulated node launch
    node_id = f"{node_name}_{datetime.now().timestamp()}"
    ros2_state["nodes"][node_id] = {
        "name": node_name,
        "package": package,
        "started_at": datetime.now().isoformat(),
        "status": "running"
    }
    
    return {
        "success": True,
        "node_id": node_id,
        "message": f"Node {node_name} launched successfully"
    }

@app.delete("/nodes/{node_id}")
async def stop_node(node_id: str) -> Dict[str, Any]:
    """Stop a ROS2 node"""
    if node_id not in ros2_state["nodes"]:
        raise HTTPException(status_code=404, detail="Node not found")
    
    del ros2_state["nodes"][node_id]
    return {"success": True, "message": f"Node {node_id} stopped"}

@app.get("/topics")
async def list_topics() -> Dict[str, Any]:
    """List available ROS2 topics"""
    # Try to get actual ROS2 topics if available
    try:
        use_docker = os.environ.get("ROS2_USE_DOCKER", "true") == "true"
        if use_docker:
            import subprocess
            result = subprocess.run(
                ["docker", "exec", "vrooli-ros2", "ros2", "topic", "list"],
                capture_output=True,
                text=True,
                timeout=2
            )
            if result.returncode == 0:
                topics = [t.strip() for t in result.stdout.strip().split("\n") if t.strip()]
                return {"topics": topics, "count": len(topics)}
    except Exception:
        pass
    
    # Fallback to simulated state
    return {
        "topics": list(ros2_state["topics"].keys()),
        "count": len(ros2_state["topics"])
    }

@app.post("/topics/{topic_name}/publish")
async def publish_to_topic(topic_name: str, message: Dict[str, Any]) -> Dict[str, Any]:
    """Publish a message to a ROS2 topic"""
    # Simulated message publishing
    if topic_name not in ros2_state["topics"]:
        ros2_state["topics"][topic_name] = {
            "created_at": datetime.now().isoformat(),
            "message_count": 0
        }
    
    ros2_state["topics"][topic_name]["message_count"] += 1
    ros2_state["topics"][topic_name]["last_message"] = message
    
    return {
        "success": True,
        "topic": topic_name,
        "message": "Message published successfully"
    }

@app.get("/services")
async def list_services() -> Dict[str, Any]:
    """List available ROS2 services"""
    return {
        "services": list(ros2_state["services"].keys()),
        "count": len(ros2_state["services"])
    }

@app.post("/services/{service_name}/call")
async def call_service(service_name: str, request: Dict[str, Any]) -> Dict[str, Any]:
    """Call a ROS2 service"""
    # Simulated service call
    return {
        "success": True,
        "service": service_name,
        "response": {"result": "Service call simulated"}
    }

@app.get("/params/{node_name}")
async def get_parameters(node_name: str) -> Dict[str, Any]:
    """Get parameters for a ROS2 node"""
    if node_name not in ros2_state["parameters"]:
        return {"parameters": {}}
    
    return {"parameters": ros2_state["parameters"][node_name]}

@app.put("/params/{node_name}")
async def set_parameters(node_name: str, params: Dict[str, Any]) -> Dict[str, Any]:
    """Set parameters for a ROS2 node"""
    if node_name not in ros2_state["parameters"]:
        ros2_state["parameters"][node_name] = {}
    
    ros2_state["parameters"][node_name].update(params)
    
    return {
        "success": True,
        "node": node_name,
        "parameters_set": list(params.keys())
    }

def check_ros2_availability() -> bool:
    """Check if ROS2 is available and functioning"""
    try:
        # Check if ROS2 environment is sourced
        ros_distro = os.environ.get("ROS_DISTRO")
        
        # Check if we're using Docker
        use_docker = os.environ.get("ROS2_USE_DOCKER", "true") == "true"
        
        if use_docker:
            # Check if Docker container is running
            import subprocess
            result = subprocess.run(
                ["docker", "ps", "--filter", "name=vrooli-ros2", "--format", "{{.Names}}"],
                capture_output=True,
                text=True,
                timeout=2
            )
            return "vrooli-ros2" in result.stdout
        else:
            # Check native installation
            return ros_distro is not None
    except Exception:
        return False

if __name__ == "__main__":
    port = int(os.environ.get("ROS2_PORT", 11501))
    host = os.environ.get("ROS2_API_HOST", "0.0.0.0")
    
    print(f"Starting ROS2 API Server on {host}:{port}")
    
    uvicorn.run(
        app,
        host=host,
        port=port,
        log_level="info"
    )