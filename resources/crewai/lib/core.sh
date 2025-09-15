#!/bin/bash
set -euo pipefail

# CrewAI Core Functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CREWAI_LIB_DIR="${APP_ROOT}/resources/crewai/lib"
CREWAI_ROOT_DIR="${APP_ROOT}/resources/crewai"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Configuration - Get port from registry or environment
if [[ -z "${CREWAI_PORT:-}" ]]; then
    CREWAI_PORT=$("${APP_ROOT}/scripts/resources/port_registry.sh" crewai | grep -E "crewai\s+:" | awk '{print $3}')
    CREWAI_PORT="${CREWAI_PORT:-8084}"
fi

CREWAI_NAME="${CREWAI_NAME:-crewai}"
CREWAI_DATA_DIR="${CREWAI_DATA_DIR:-${HOME}/.crewai}"
CREWAI_WORKSPACE_DIR="${CREWAI_WORKSPACE_DIR:-${CREWAI_DATA_DIR}/workspace}"
CREWAI_CREWS_DIR="${CREWAI_CREWS_DIR:-${CREWAI_DATA_DIR}/crews}"
CREWAI_AGENTS_DIR="${CREWAI_AGENTS_DIR:-${CREWAI_DATA_DIR}/agents}"
CREWAI_PID_FILE="${CREWAI_PID_FILE:-${CREWAI_DATA_DIR}/crewai.pid}"
CREWAI_LOG_FILE="${CREWAI_LOG_FILE:-${CREWAI_DATA_DIR}/crewai.log}"
CREWAI_SERVER_FILE="${CREWAI_SERVER_FILE:-${CREWAI_DATA_DIR}/server.py}"
CREWAI_MOCK_MODE="${CREWAI_MOCK_MODE:-false}"  # Try to use Flask server with tools
CREWAI_VENV_DIR="${CREWAI_VENV_DIR:-${CREWAI_DATA_DIR}/venv}"

# Check if Python is available
check_python() {
    if command -v python3 &>/dev/null; then
        echo "python3"
    else
        log::error "Python 3 is required but not found"
        return 1
    fi
}

# Initialize directories
init_directories() {
    mkdir -p "${CREWAI_DATA_DIR}"
    mkdir -p "${CREWAI_WORKSPACE_DIR}"
    mkdir -p "${CREWAI_CREWS_DIR}"
    mkdir -p "${CREWAI_AGENTS_DIR}"
}

# Install CrewAI
install_crewai() {
    init_directories
    
    # Check environment variable for mode
    local mode="${CREWAI_MOCK_MODE}"
    
    log::info "Installing CrewAI with mode=${mode}"
    
    # Try to use Flask mode if requested
    # mode="true"  # Removed forced mock mode
    
    if [[ "${mode}" == "true" ]]; then
        log::info "Installing CrewAI in mock mode..."
        create_server_file
        log::success "CrewAI installed successfully (mock mode)"
    else
        log::info "Installing real CrewAI library..."
        
        # Check Python availability
        if ! command -v python3 &>/dev/null; then
            log::error "Python 3 is required but not installed"
            log::info "Falling back to mock mode"
            create_server_file
            log::success "CrewAI installed successfully (mock mode)"
            return 0
        fi
        
        # Create virtual environment
        if [[ ! -d "${CREWAI_VENV_DIR}" ]]; then
            log::info "Creating virtual environment..."
            python3 -m venv "${CREWAI_VENV_DIR}" || {
                log::warn "Failed to create virtual environment, using system Python"
            }
        fi
        
        # Install dependencies
        if [[ -f "${CREWAI_VENV_DIR}/bin/pip" ]]; then
            log::info "Installing CrewAI dependencies..."
            "${CREWAI_VENV_DIR}/bin/pip" install --upgrade pip &>/dev/null || true
            "${CREWAI_VENV_DIR}/bin/pip" install crewai crewai-tools flask flask-cors &>/dev/null || {
                log::warn "Failed to install CrewAI packages, but continuing with Flask support"
                "${CREWAI_VENV_DIR}/bin/pip" install flask flask-cors &>/dev/null || true
            }
        else
            # System pip fallback
            log::info "Installing Flask dependencies with system pip..."
            pip3 install flask flask-cors &>/dev/null || true
        fi
        
        # Create enhanced server file with real CrewAI
        create_real_server_file
        
        log::success "CrewAI installed (real mode with Flask API)"
    fi
}

# Create server file
create_server_file() {
    cat > "${CREWAI_SERVER_FILE}" << 'EOF'
#!/usr/bin/env python3
"""CrewAI Mock API Server for Vrooli"""

import os
import json
import uuid
from pathlib import Path
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime
import shutil
from urllib.parse import urlparse, parse_qs
import threading
import time

# Configuration
PORT = int(os.environ.get('CREWAI_PORT', 8084))

# Paths
CREWAI_DATA_DIR = Path.home() / ".crewai"
CREWS_DIR = CREWAI_DATA_DIR / "crews"
AGENTS_DIR = CREWAI_DATA_DIR / "agents"
WORKSPACE_DIR = CREWAI_DATA_DIR / "workspace"
TASKS_DIR = CREWAI_DATA_DIR / "tasks"

# Ensure directories exist
for dir_path in [CREWS_DIR, AGENTS_DIR, WORKSPACE_DIR, TASKS_DIR]:
    dir_path.mkdir(parents=True, exist_ok=True)

# Store loaded crews and agents
loaded_crews = {}
loaded_agents = {}
task_executions = {}

class CrewAIHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        """Handle GET requests"""
        parsed_path = urlparse(self.path)
        path = parsed_path.path
        
        if path == "/":
            self.send_json_response(200, {
                "name": "CrewAI Server",
                "version": "1.0.0-mock",
                "status": "running",
                "crews": len(loaded_crews),
                "agents": len(loaded_agents),
                "workspace": str(WORKSPACE_DIR),
                "capabilities": ["crews", "agents", "tasks", "inject", "execute"]
            })
        elif path == "/health":
            self.send_json_response(200, {
                "status": "healthy",
                "timestamp": datetime.utcnow().isoformat(),
                "crews_loaded": len(loaded_crews),
                "agents_loaded": len(loaded_agents),
                "active_tasks": len([t for t in task_executions.values() if t["status"] == "running"])
            })
        elif path == "/crews":
            crews = []
            for crew_file in CREWS_DIR.glob("*.py"):
                crew_name = crew_file.stem
                crew_data = loaded_crews.get(crew_name, {})
                crews.append({
                    "name": crew_name,
                    "loaded": crew_name in loaded_crews,
                    "path": str(crew_file),
                    "agents": crew_data.get("agents", []),
                    "tasks": crew_data.get("tasks", [])
                })
            # Also include JSON crews
            for crew_file in CREWS_DIR.glob("*.json"):
                crew_name = crew_file.stem
                try:
                    with open(crew_file) as f:
                        crew_data = json.load(f)
                    crews.append({
                        "name": crew_name,
                        "loaded": True,
                        "path": str(crew_file),
                        "agents": crew_data.get("agents", []),
                        "tasks": crew_data.get("tasks", [])
                    })
                except:
                    pass
            self.send_json_response(200, {"crews": crews})
        elif path == "/agents":
            agents = []
            for agent_file in AGENTS_DIR.glob("*.py"):
                agent_name = agent_file.stem
                agent_data = loaded_agents.get(agent_name, {})
                agents.append({
                    "name": agent_name,
                    "loaded": agent_name in loaded_agents,
                    "path": str(agent_file),
                    "role": agent_data.get("role", ""),
                    "goal": agent_data.get("goal", "")
                })
            # Also include JSON agents
            for agent_file in AGENTS_DIR.glob("*.json"):
                agent_name = agent_file.stem
                try:
                    with open(agent_file) as f:
                        agent_data = json.load(f)
                    agents.append({
                        "name": agent_name,
                        "loaded": True,
                        "path": str(agent_file),
                        "role": agent_data.get("role", ""),
                        "goal": agent_data.get("goal", "")
                    })
                except:
                    pass
            self.send_json_response(200, {"agents": agents})
        elif path == "/tasks":
            tasks = list(task_executions.values())
            self.send_json_response(200, {"tasks": tasks})
        elif path.startswith("/tasks/"):
            task_id = path.split("/")[-1]
            if task_id in task_executions:
                self.send_json_response(200, task_executions[task_id])
            else:
                self.send_json_response(404, {"error": "Task not found"})
        elif path.startswith("/crews/"):
            crew_name = path.split("/")[-1]
            crew_file = CREWS_DIR / f"{crew_name}.json"
            if crew_file.exists():
                with open(crew_file) as f:
                    crew_data = json.load(f)
                self.send_json_response(200, crew_data)
            else:
                self.send_json_response(404, {"error": "Crew not found"})
        elif path.startswith("/agents/"):
            agent_name = path.split("/")[-1]
            agent_file = AGENTS_DIR / f"{agent_name}.json"
            if agent_file.exists():
                with open(agent_file) as f:
                    agent_data = json.load(f)
                self.send_json_response(200, agent_data)
            else:
                self.send_json_response(404, {"error": "Agent not found"})
        else:
            self.send_json_response(404, {"error": "Not found"})
    
    def do_POST(self):
        """Handle POST requests"""
        parsed_path = urlparse(self.path)
        path = parsed_path.path
        
        content_length = int(self.headers.get('Content-Length', 0))
        if content_length > 0:
            post_data = self.rfile.read(content_length)
            try:
                data = json.loads(post_data)
            except:
                data = {}
        else:
            data = {}
        
        if path == "/inject":
            file_path = data.get('file_path')
            file_type = data.get('file_type')
            
            if not file_path or not file_type:
                self.send_json_response(400, {"error": "Missing file_path or file_type"})
                return
            
            source_path = Path(file_path)
            if not source_path.exists():
                self.send_json_response(404, {"error": f"File not found: {file_path}"})
                return
            
            if file_type == "crew":
                dest_dir = CREWS_DIR
            elif file_type == "agent":
                dest_dir = AGENTS_DIR
            else:
                self.send_json_response(400, {"error": "file_type must be 'crew' or 'agent'"})
                return
            
            dest_path = dest_dir / source_path.name
            
            try:
                shutil.copy2(source_path, dest_path)
                self.send_json_response(200, {
                    "status": "injected",
                    "type": file_type,
                    "name": source_path.stem,
                    "destination": str(dest_path)
                })
            except Exception as e:
                self.send_json_response(500, {"error": str(e)})
        
        elif path == "/crews":
            # Create a new crew
            crew_name = data.get("name", f"crew_{int(time.time())}")
            agents = data.get("agents", [])
            tasks = data.get("tasks", [])
            
            crew_data = {
                "name": crew_name,
                "agents": agents,
                "tasks": tasks,
                "created": datetime.utcnow().isoformat()
            }
            
            crew_file = CREWS_DIR / f"{crew_name}.json"
            with open(crew_file, "w") as f:
                json.dump(crew_data, f, indent=2)
            
            loaded_crews[crew_name] = crew_data
            
            self.send_json_response(201, {
                "status": "created",
                "crew": crew_data,
                "path": str(crew_file)
            })
        
        elif path == "/agents":
            # Create a new agent
            agent_name = data.get("name", f"agent_{int(time.time())}")
            role = data.get("role", "assistant")
            goal = data.get("goal", "help with tasks")
            backstory = data.get("backstory", "")
            
            agent_data = {
                "name": agent_name,
                "role": role,
                "goal": goal,
                "backstory": backstory,
                "created": datetime.utcnow().isoformat()
            }
            
            agent_file = AGENTS_DIR / f"{agent_name}.json"
            with open(agent_file, "w") as f:
                json.dump(agent_data, f, indent=2)
            
            loaded_agents[agent_name] = agent_data
            
            self.send_json_response(201, {
                "status": "created",
                "agent": agent_data,
                "path": str(agent_file)
            })
        
        elif path == "/execute":
            # Execute a crew (mock execution)
            crew_name = data.get("crew")
            input_data = data.get("input", {})
            
            if not crew_name:
                self.send_json_response(400, {"error": "Missing crew name"})
                return
            
            crew_file = CREWS_DIR / f"{crew_name}.json"
            if not crew_file.exists():
                self.send_json_response(404, {"error": "Crew not found"})
                return
            
            # Create mock task execution
            task_id = str(uuid.uuid4())
            task_data = {
                "id": task_id,
                "crew": crew_name,
                "input": input_data,
                "status": "running",
                "started": datetime.utcnow().isoformat(),
                "progress": 0,
                "result": None
            }
            
            task_executions[task_id] = task_data
            
            # Simulate async execution
            def mock_execute():
                for i in range(1, 11):
                    time.sleep(0.5)
                    task_executions[task_id]["progress"] = i * 10
                
                task_executions[task_id]["status"] = "completed"
                task_executions[task_id]["completed"] = datetime.utcnow().isoformat()
                task_executions[task_id]["result"] = {
                    "output": f"Mock execution of {crew_name} completed",
                    "data": input_data
                }
            
            thread = threading.Thread(target=mock_execute)
            thread.daemon = True
            thread.start()
            
            self.send_json_response(202, {
                "status": "started",
                "task_id": task_id,
                "message": f"Crew {crew_name} execution started"
            })
        
        elif path.startswith("/crews/") and path.endswith("/delete"):
            crew_name = path.split("/")[-2]
            crew_file = CREWS_DIR / f"{crew_name}.json"
            if crew_file.exists():
                crew_file.unlink()
                if crew_name in loaded_crews:
                    del loaded_crews[crew_name]
                self.send_json_response(200, {"status": "deleted", "crew": crew_name})
            else:
                self.send_json_response(404, {"error": "Crew not found"})
        
        elif path.startswith("/agents/") and path.endswith("/delete"):
            agent_name = path.split("/")[-2]
            agent_file = AGENTS_DIR / f"{agent_name}.json"
            if agent_file.exists():
                agent_file.unlink()
                if agent_name in loaded_agents:
                    del loaded_agents[agent_name]
                self.send_json_response(200, {"status": "deleted", "agent": agent_name})
            else:
                self.send_json_response(404, {"error": "Agent not found"})
        
        else:
            self.send_json_response(404, {"error": "Not found"})
    
    def do_DELETE(self):
        """Handle DELETE requests"""
        parsed_path = urlparse(self.path)
        path = parsed_path.path
        
        if path.startswith("/crews/"):
            crew_name = path.split("/")[-1]
            crew_file = CREWS_DIR / f"{crew_name}.json"
            if crew_file.exists():
                crew_file.unlink()
                if crew_name in loaded_crews:
                    del loaded_crews[crew_name]
                self.send_json_response(200, {"status": "deleted", "crew": crew_name})
            else:
                self.send_json_response(404, {"error": "Crew not found"})
        
        elif path.startswith("/agents/"):
            agent_name = path.split("/")[-1]
            agent_file = AGENTS_DIR / f"{agent_name}.json"
            if agent_file.exists():
                agent_file.unlink()
                if agent_name in loaded_agents:
                    del loaded_agents[agent_name]
                self.send_json_response(200, {"status": "deleted", "agent": agent_name})
            else:
                self.send_json_response(404, {"error": "Agent not found"})
        
        else:
            self.send_json_response(404, {"error": "Not found"})
    
    def send_json_response(self, status_code, data):
        """Send JSON response"""
        self.send_response(status_code)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())
    
    def log_message(self, format, *args):
        """Override to suppress default logging"""
        pass

if __name__ == "__main__":
    # Create sample data if none exists
    if not list(AGENTS_DIR.glob("*.json")):
        sample_agent = {
            "name": "researcher",
            "role": "Senior Research Analyst",
            "goal": "Gather and analyze information",
            "backstory": "Expert at finding and synthesizing information"
        }
        with open(AGENTS_DIR / "researcher.json", "w") as f:
            json.dump(sample_agent, f, indent=2)
    
    if not list(CREWS_DIR.glob("*.json")):
        sample_crew = {
            "name": "research_crew",
            "agents": ["researcher"],
            "tasks": ["gather_info", "analyze_data"],
            "description": "Crew for research and analysis tasks"
        }
        with open(CREWS_DIR / "research_crew.json", "w") as f:
            json.dump(sample_crew, f, indent=2)
    
    server = HTTPServer(('0.0.0.0', PORT), CrewAIHandler)
    print(f"CrewAI Mock Server running on port {PORT}")
    server.serve_forever()
EOF
    chmod +x "${CREWAI_SERVER_FILE}"
}

# Create real server file with actual CrewAI integration
create_real_server_file() {
    cat > "${CREWAI_SERVER_FILE}" << 'EOF'
#!/usr/bin/env python3
"""CrewAI Real API Server for Vrooli"""

import os
import json
import uuid
import sys
from pathlib import Path
from datetime import datetime
import threading
import time
from flask import Flask, request, jsonify
from flask_cors import CORS

# Try to import CrewAI - fallback to mock if not available
try:
    from crewai import Agent, Crew, Task, Process
    from crewai.tools import tool
    CREWAI_AVAILABLE = True
except ImportError:
    CREWAI_AVAILABLE = False
    print("Warning: CrewAI not installed, running in limited mode", file=sys.stderr)

# Try to import Qdrant for memory support
try:
    from qdrant_client import QdrantClient
    from qdrant_client.models import Distance, VectorParams, PointStruct
    QDRANT_AVAILABLE = True
except ImportError:
    QDRANT_AVAILABLE = False
    print("Warning: Qdrant not installed, memory features disabled", file=sys.stderr)

# Configuration
PORT = int(os.environ.get('CREWAI_PORT', 8084))

# Paths
CREWAI_DATA_DIR = Path.home() / ".crewai"
CREWS_DIR = CREWAI_DATA_DIR / "crews"
AGENTS_DIR = CREWAI_DATA_DIR / "agents"
WORKSPACE_DIR = CREWAI_DATA_DIR / "workspace"
TASKS_DIR = CREWAI_DATA_DIR / "tasks"

# Ensure directories exist
for dir_path in [CREWS_DIR, AGENTS_DIR, WORKSPACE_DIR, TASKS_DIR]:
    dir_path.mkdir(parents=True, exist_ok=True)

# Flask app
app = Flask(__name__)
CORS(app)

# Store loaded crews, agents, and tasks
loaded_crews = {}
loaded_agents = {}
task_executions = {}
crewai_agents = {}  # Store actual CrewAI Agent objects
crewai_crews = {}   # Store actual CrewAI Crew objects

# Initialize Qdrant client if available
qdrant_client = None
if QDRANT_AVAILABLE:
    try:
        qdrant_client = QdrantClient(host="localhost", port=6333)
        # Create collection for agent memories if it doesn't exist
        try:
            qdrant_client.get_collection("agent_memories")
        except:
            qdrant_client.create_collection(
                collection_name="agent_memories",
                vectors_config=VectorParams(size=384, distance=Distance.COSINE)
            )
    except Exception as e:
        print(f"Warning: Could not connect to Qdrant: {e}", file=sys.stderr)
        qdrant_client = None

# Tool definitions for agents
def get_tool_by_name(tool_name):
    """Get a tool by name"""
    if not CREWAI_AVAILABLE:
        return None
    
    # Define available tools
    available_tools = {
        "web_search": create_web_search_tool(),
        "file_reader": create_file_reader_tool(),
        "api_caller": create_api_caller_tool(),
        "database_query": create_database_query_tool(),
        "llm_query": create_llm_query_tool(),
        "memory_store": create_memory_store_tool(),
        "memory_retrieve": create_memory_retrieve_tool()
    }
    
    return available_tools.get(tool_name)

def create_web_search_tool():
    """Create a web search tool"""
    if not CREWAI_AVAILABLE:
        return None
    
    @tool("Web Search")
    def web_search(query: str) -> str:
        """Search the web for information"""
        import requests
        try:
            # This is a mock implementation - replace with actual search API
            return f"Search results for: {query} - [Mock results would appear here]"
        except Exception as e:
            return f"Error searching web: {str(e)}"
    
    return web_search

def create_file_reader_tool():
    """Create a file reader tool"""
    if not CREWAI_AVAILABLE:
        return None
    
    @tool("File Reader")
    def file_reader(file_path: str) -> str:
        """Read contents of a file"""
        try:
            file = Path(file_path)
            if file.exists():
                with open(file, 'r') as f:
                    return f.read()
            return f"File not found: {file_path}"
        except Exception as e:
            return f"Error reading file: {str(e)}"
    
    return file_reader

def create_api_caller_tool():
    """Create an API caller tool"""
    if not CREWAI_AVAILABLE:
        return None
    
    @tool("API Caller")
    def api_caller(url: str, method: str = "GET", data: dict = None) -> str:
        """Make API calls to external services"""
        import requests
        try:
            if method.upper() == "GET":
                response = requests.get(url, timeout=10)
            elif method.upper() == "POST":
                response = requests.post(url, json=data, timeout=10)
            else:
                return f"Unsupported method: {method}"
            
            return response.text
        except Exception as e:
            return f"Error calling API: {str(e)}"
    
    return api_caller

def create_database_query_tool():
    """Create a database query tool"""
    if not CREWAI_AVAILABLE:
        return None
    
    @tool("Database Query")
    def database_query(query: str, database: str = "postgres") -> str:
        """Query a database"""
        try:
            # Mock implementation - would connect to actual database
            return f"Query result for '{query}' on {database}: [Mock results]"
        except Exception as e:
            return f"Error querying database: {str(e)}"
    
    return database_query

def create_llm_query_tool():
    """Create an LLM query tool for Ollama integration"""
    if not CREWAI_AVAILABLE:
        return None
    
    @tool("LLM Query")
    def llm_query(prompt: str, model: str = "llama2") -> str:
        """Query a local LLM via Ollama"""
        import requests
        try:
            # Call Ollama API
            response = requests.post(
                "http://localhost:11434/api/generate",
                json={"model": model, "prompt": prompt, "stream": False},
                timeout=30
            )
            if response.status_code == 200:
                return response.json().get("response", "No response from LLM")
            return f"LLM error: {response.status_code}"
        except Exception as e:
            return f"Error querying LLM: {str(e)}"
    
    return llm_query

def create_memory_store_tool():
    """Create a memory storage tool using Qdrant"""
    if not CREWAI_AVAILABLE or not qdrant_client:
        return None
    
    @tool("Memory Store")
    def memory_store(key: str, value: str, metadata: dict = None) -> str:
        """Store information in long-term memory"""
        try:
            # Generate a simple embedding (in production, use a proper embedding model)
            import hashlib
            vector = [float(ord(c)) / 255.0 for c in hashlib.sha384(value.encode()).hexdigest()][:384]
            
            point = PointStruct(
                id=abs(hash(key)),
                vector=vector,
                payload={"key": key, "value": value, "metadata": metadata or {}}
            )
            
            qdrant_client.upsert(
                collection_name="agent_memories",
                points=[point]
            )
            
            return f"Stored memory with key: {key}"
        except Exception as e:
            return f"Error storing memory: {str(e)}"
    
    return memory_store

def create_memory_retrieve_tool():
    """Create a memory retrieval tool using Qdrant"""
    if not CREWAI_AVAILABLE or not qdrant_client:
        return None
    
    @tool("Memory Retrieve")
    def memory_retrieve(query: str, limit: int = 5) -> str:
        """Retrieve information from long-term memory"""
        try:
            # Generate query vector (in production, use same embedding model as storage)
            import hashlib
            vector = [float(ord(c)) / 255.0 for c in hashlib.sha384(query.encode()).hexdigest()][:384]
            
            results = qdrant_client.search(
                collection_name="agent_memories",
                query_vector=vector,
                limit=limit
            )
            
            if results:
                memories = []
                for result in results:
                    memories.append(f"Key: {result.payload.get('key')}, Value: {result.payload.get('value')}")
                return "\n".join(memories)
            return "No memories found"
        except Exception as e:
            return f"Error retrieving memory: {str(e)}"
    
    return memory_retrieve

# Helper functions
def load_agent_config(agent_name):
    """Load agent configuration from JSON file"""
    agent_file = AGENTS_DIR / f"{agent_name}.json"
    if agent_file.exists():
        with open(agent_file) as f:
            return json.load(f)
    return None

def load_crew_config(crew_name):
    """Load crew configuration from JSON file"""
    crew_file = CREWS_DIR / f"{crew_name}.json"
    if crew_file.exists():
        with open(crew_file) as f:
            return json.load(f)
    return None

def create_crewai_agent(agent_config):
    """Create a real CrewAI Agent from configuration"""
    if not CREWAI_AVAILABLE:
        return None
    
    # Get tools if specified
    tools = []
    for tool_name in agent_config.get("tools", []):
        tool = get_tool_by_name(tool_name)
        if tool:
            tools.append(tool)
    
    return Agent(
        role=agent_config.get("role", "Assistant"),
        goal=agent_config.get("goal", "Help with tasks"),
        backstory=agent_config.get("backstory", "An AI assistant"),
        verbose=True,
        allow_delegation=agent_config.get("allow_delegation", False),
        max_iter=agent_config.get("max_iter", 5),
        tools=tools
    )

def create_crewai_crew(crew_config, agents):
    """Create a real CrewAI Crew from configuration"""
    if not CREWAI_AVAILABLE:
        return None
    
    # Create tasks for the crew
    tasks = []
    for task_config in crew_config.get("tasks", []):
        task = Task(
            description=task_config.get("description", "Complete task"),
            expected_output=task_config.get("expected_output", "Task result"),
            agent=agents[0] if agents else None  # Assign to first agent for now
        )
        tasks.append(task)
    
    # Create the crew
    return Crew(
        agents=agents,
        tasks=tasks,
        process=Process.sequential,  # Use sequential process by default
        verbose=True
    )

# Routes
@app.route("/", methods=["GET"])
def root():
    """API root endpoint"""
    return jsonify({
        "name": "CrewAI Server",
        "version": "2.0.0" if CREWAI_AVAILABLE else "1.0.0-limited",
        "status": "running",
        "crewai_library": CREWAI_AVAILABLE,
        "crews": len(loaded_crews),
        "agents": len(loaded_agents),
        "workspace": str(WORKSPACE_DIR),
        "capabilities": ["crews", "agents", "tasks", "inject", "execute"]
    })

@app.route("/health", methods=["GET"])
def health():
    """Health check endpoint"""
    return jsonify({
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "crewai_available": CREWAI_AVAILABLE,
        "qdrant_available": qdrant_client is not None,
        "crews_loaded": len(loaded_crews),
        "agents_loaded": len(loaded_agents),
        "active_tasks": len([t for t in task_executions.values() if t.get("status") == "running"])
    })

@app.route("/tools", methods=["GET"])
def tools():
    """List available tools for agents"""
    tool_list = []
    if CREWAI_AVAILABLE:
        tool_list = [
            {"name": "web_search", "description": "Search the web for information"},
            {"name": "file_reader", "description": "Read contents of a file"},
            {"name": "api_caller", "description": "Make API calls to external services"},
            {"name": "database_query", "description": "Query a database"},
            {"name": "llm_query", "description": "Query a local LLM via Ollama"},
        ]
        if qdrant_client:
            tool_list.extend([
                {"name": "memory_store", "description": "Store information in long-term memory"},
                {"name": "memory_retrieve", "description": "Retrieve information from long-term memory"}
            ])
    return jsonify({"tools": tool_list, "available": CREWAI_AVAILABLE})

@app.route("/agents", methods=["GET", "POST"])
def agents():
    """List or create agents"""
    if request.method == "GET":
        agents_list = []
        for agent_file in AGENTS_DIR.glob("*.json"):
            agent_name = agent_file.stem
            agent_data = load_agent_config(agent_name)
            if agent_data:
                agents_list.append({
                    "name": agent_name,
                    "loaded": agent_name in crewai_agents if CREWAI_AVAILABLE else True,
                    "path": str(agent_file),
                    "role": agent_data.get("role", ""),
                    "goal": agent_data.get("goal", "")
                })
        return jsonify({"agents": agents_list})
    
    elif request.method == "POST":
        data = request.json or {}
        agent_name = data.get("name", f"agent_{int(time.time())}")
        
        agent_data = {
            "name": agent_name,
            "role": data.get("role", "Assistant"),
            "goal": data.get("goal", "Help with tasks"),
            "backstory": data.get("backstory", "An AI assistant"),
            "allow_delegation": data.get("allow_delegation", False),
            "max_iter": data.get("max_iter", 5),
            "tools": data.get("tools", []),  # Add tools support
            "created": datetime.utcnow().isoformat()
        }
        
        # Save configuration
        agent_file = AGENTS_DIR / f"{agent_name}.json"
        with open(agent_file, "w") as f:
            json.dump(agent_data, f, indent=2)
        
        loaded_agents[agent_name] = agent_data
        
        # Create real CrewAI agent if available
        if CREWAI_AVAILABLE:
            crewai_agents[agent_name] = create_crewai_agent(agent_data)
        
        return jsonify({
            "status": "created",
            "agent": agent_data,
            "path": str(agent_file),
            "crewai_initialized": agent_name in crewai_agents if CREWAI_AVAILABLE else False
        }), 201

@app.route("/agents/<agent_name>", methods=["GET", "DELETE"])
def agent_detail(agent_name):
    """Get or delete specific agent"""
    if request.method == "GET":
        agent_data = load_agent_config(agent_name)
        if agent_data:
            return jsonify(agent_data)
        return jsonify({"error": "Agent not found"}), 404
    
    elif request.method == "DELETE":
        agent_file = AGENTS_DIR / f"{agent_name}.json"
        if agent_file.exists():
            agent_file.unlink()
            if agent_name in loaded_agents:
                del loaded_agents[agent_name]
            if CREWAI_AVAILABLE and agent_name in crewai_agents:
                del crewai_agents[agent_name]
            return jsonify({"status": "deleted", "agent": agent_name})
        return jsonify({"error": "Agent not found"}), 404

@app.route("/crews", methods=["GET", "POST"])
def crews():
    """List or create crews"""
    if request.method == "GET":
        crews_list = []
        for crew_file in CREWS_DIR.glob("*.json"):
            crew_name = crew_file.stem
            crew_data = load_crew_config(crew_name)
            if crew_data:
                crews_list.append({
                    "name": crew_name,
                    "loaded": crew_name in crewai_crews if CREWAI_AVAILABLE else True,
                    "path": str(crew_file),
                    "agents": crew_data.get("agents", []),
                    "tasks": crew_data.get("tasks", [])
                })
        return jsonify({"crews": crews_list})
    
    elif request.method == "POST":
        data = request.json or {}
        crew_name = data.get("name", f"crew_{int(time.time())}")
        
        crew_data = {
            "name": crew_name,
            "agents": data.get("agents", []),
            "tasks": data.get("tasks", []),
            "process": data.get("process", "sequential"),
            "created": datetime.utcnow().isoformat()
        }
        
        # Save configuration
        crew_file = CREWS_DIR / f"{crew_name}.json"
        with open(crew_file, "w") as f:
            json.dump(crew_data, f, indent=2)
        
        loaded_crews[crew_name] = crew_data
        
        # Create real CrewAI crew if available
        if CREWAI_AVAILABLE:
            # Load agents for the crew
            agents = []
            for agent_name in crew_data.get("agents", []):
                if agent_name not in crewai_agents:
                    agent_config = load_agent_config(agent_name)
                    if agent_config:
                        crewai_agents[agent_name] = create_crewai_agent(agent_config)
                if agent_name in crewai_agents:
                    agents.append(crewai_agents[agent_name])
            
            if agents:
                crewai_crews[crew_name] = create_crewai_crew(crew_data, agents)
        
        return jsonify({
            "status": "created",
            "crew": crew_data,
            "path": str(crew_file),
            "crewai_initialized": crew_name in crewai_crews if CREWAI_AVAILABLE else False
        }), 201

@app.route("/crews/<crew_name>", methods=["GET", "DELETE"])
def crew_detail(crew_name):
    """Get or delete specific crew"""
    if request.method == "GET":
        crew_data = load_crew_config(crew_name)
        if crew_data:
            return jsonify(crew_data)
        return jsonify({"error": "Crew not found"}), 404
    
    elif request.method == "DELETE":
        crew_file = CREWS_DIR / f"{crew_name}.json"
        if crew_file.exists():
            crew_file.unlink()
            if crew_name in loaded_crews:
                del loaded_crews[crew_name]
            if CREWAI_AVAILABLE and crew_name in crewai_crews:
                del crewai_crews[crew_name]
            return jsonify({"status": "deleted", "crew": crew_name})
        return jsonify({"error": "Crew not found"}), 404

@app.route("/execute", methods=["POST"])
def execute():
    """Execute a crew with input"""
    data = request.json or {}
    crew_name = data.get("crew")
    input_data = data.get("input", {})
    
    if not crew_name:
        return jsonify({"error": "Missing crew name"}), 400
    
    crew_config = load_crew_config(crew_name)
    if not crew_config:
        return jsonify({"error": "Crew not found"}), 404
    
    # Create task execution record
    task_id = str(uuid.uuid4())
    task_data = {
        "id": task_id,
        "crew": crew_name,
        "input": input_data,
        "status": "running",
        "started": datetime.utcnow().isoformat(),
        "progress": 0,
        "result": None
    }
    
    task_executions[task_id] = task_data
    
    # Execute crew
    def execute_crew():
        try:
            if CREWAI_AVAILABLE and crew_name in crewai_crews:
                # Real CrewAI execution
                crew = crewai_crews[crew_name]
                result = crew.kickoff(inputs=input_data)
                
                task_executions[task_id]["status"] = "completed"
                task_executions[task_id]["completed"] = datetime.utcnow().isoformat()
                task_executions[task_id]["result"] = {
                    "output": str(result),
                    "data": input_data
                }
            else:
                # Mock execution
                for i in range(1, 11):
                    time.sleep(0.5)
                    task_executions[task_id]["progress"] = i * 10
                
                task_executions[task_id]["status"] = "completed"
                task_executions[task_id]["completed"] = datetime.utcnow().isoformat()
                task_executions[task_id]["result"] = {
                    "output": f"Mock execution of {crew_name} completed",
                    "data": input_data,
                    "note": "Install CrewAI library for real execution"
                }
        except Exception as e:
            task_executions[task_id]["status"] = "failed"
            task_executions[task_id]["error"] = str(e)
            task_executions[task_id]["completed"] = datetime.utcnow().isoformat()
    
    thread = threading.Thread(target=execute_crew)
    thread.daemon = True
    thread.start()
    
    return jsonify({
        "status": "started",
        "task_id": task_id,
        "message": f"Crew {crew_name} execution started",
        "using_crewai": CREWAI_AVAILABLE and crew_name in crewai_crews
    }), 202

@app.route("/tasks", methods=["GET"])
def tasks():
    """List all task executions"""
    return jsonify({"tasks": list(task_executions.values())})

@app.route("/tasks/<task_id>", methods=["GET"])
def task_detail(task_id):
    """Get specific task execution status"""
    if task_id in task_executions:
        return jsonify(task_executions[task_id])
    return jsonify({"error": "Task not found"}), 404

@app.route("/inject", methods=["POST"])
def inject():
    """Inject crew or agent files"""
    data = request.json or {}
    file_path = data.get("file_path")
    file_type = data.get("file_type")
    
    if not file_path or not file_type:
        return jsonify({"error": "Missing file_path or file_type"}), 400
    
    source_path = Path(file_path)
    if not source_path.exists():
        return jsonify({"error": f"File not found: {file_path}"}), 404
    
    if file_type == "crew":
        dest_dir = CREWS_DIR
    elif file_type == "agent":
        dest_dir = AGENTS_DIR
    else:
        return jsonify({"error": "file_type must be 'crew' or 'agent'"}), 400
    
    dest_path = dest_dir / source_path.name
    
    try:
        import shutil
        shutil.copy2(source_path, dest_path)
        
        # Load the configuration if it's JSON
        if source_path.suffix == ".json":
            with open(dest_path) as f:
                config = json.load(f)
            
            if file_type == "agent" and CREWAI_AVAILABLE:
                agent_name = source_path.stem
                crewai_agents[agent_name] = create_crewai_agent(config)
            elif file_type == "crew" and CREWAI_AVAILABLE:
                crew_name = source_path.stem
                # Load agents for the crew
                agents = []
                for agent_name in config.get("agents", []):
                    if agent_name in crewai_agents:
                        agents.append(crewai_agents[agent_name])
                if agents:
                    crewai_crews[crew_name] = create_crewai_crew(config, agents)
        
        return jsonify({
            "status": "injected",
            "type": file_type,
            "name": source_path.stem,
            "destination": str(dest_path),
            "crewai_loaded": CREWAI_AVAILABLE
        })
    except Exception as e:
        return jsonify({"error": str(e)}), 500

if __name__ == "__main__":
    # Create sample data if none exists
    if not list(AGENTS_DIR.glob("*.json")):
        sample_agent = {
            "name": "researcher",
            "role": "Senior Research Analyst",
            "goal": "Gather and analyze information",
            "backstory": "Expert at finding and synthesizing information",
            "allow_delegation": False,
            "max_iter": 5
        }
        with open(AGENTS_DIR / "researcher.json", "w") as f:
            json.dump(sample_agent, f, indent=2)
    
    if not list(CREWS_DIR.glob("*.json")):
        sample_crew = {
            "name": "research_crew",
            "agents": ["researcher"],
            "tasks": [
                {
                    "description": "Research the given topic thoroughly",
                    "expected_output": "Comprehensive research report"
                }
            ],
            "process": "sequential",
            "description": "Crew for research and analysis tasks"
        }
        with open(CREWS_DIR / "research_crew.json", "w") as f:
            json.dump(sample_crew, f, indent=2)
    
    print(f"CrewAI Server running on port {PORT} (CrewAI {'enabled' if CREWAI_AVAILABLE else 'not available'})")
    app.run(host="0.0.0.0", port=PORT, debug=False)
EOF
    chmod +x "${CREWAI_SERVER_FILE}"
}

# Start CrewAI service
start_crewai() {
    if is_running; then
        log::info "CrewAI is already running"
        return 0
    fi
    
    init_directories
    
    # Use configured mode (defaults to Flask/real mode)
    local mode="${CREWAI_MOCK_MODE}"
    
    if [[ ! -f "${CREWAI_SERVER_FILE}" ]]; then
        if [[ "${mode}" == "true" ]]; then
            create_server_file
        else
            create_real_server_file
        fi
    fi
    
    if [[ "${mode}" == "true" ]]; then
        log::info "Starting CrewAI service (mock mode)..."
        # Start server directly with python3
        nohup python3 "${CREWAI_SERVER_FILE}" > "${CREWAI_LOG_FILE}" 2>&1 &
    else
        log::info "Starting CrewAI service with Flask server..."
        # Check if Flask is available
        if python3 -c "import flask" 2>/dev/null; then
            log::info "Flask is available, starting Flask server..."
            nohup python3 "${CREWAI_SERVER_FILE}" > "${CREWAI_LOG_FILE}" 2>&1 &
        elif [[ -f "${CREWAI_VENV_DIR}/bin/python" ]]; then
            log::info "Using virtual environment..."
            nohup "${CREWAI_VENV_DIR}/bin/python" "${CREWAI_SERVER_FILE}" > "${CREWAI_LOG_FILE}" 2>&1 &
        else
            log::warn "Flask not available, falling back to mock mode"
            CREWAI_MOCK_MODE="true"
            create_server_file
            nohup python3 "${CREWAI_SERVER_FILE}" > "${CREWAI_LOG_FILE}" 2>&1 &
        fi
    fi
    local pid=$!
    echo $pid > "${CREWAI_PID_FILE}"
    
    # Register the server process as an agent
    local agent_id
    agent_id=$(agents::generate_id)
    agents::register "$agent_id" "$pid" "crewai_server python3 ${CREWAI_SERVER_FILE}" || log::debug "Failed to register server agent"
    
    sleep 2
    
    if is_running; then
        log::success "CrewAI started on port ${CREWAI_PORT}"
    else
        log::error "Failed to start CrewAI"
        return 1
    fi
}

# Stop CrewAI service
stop_crewai() {
    if [[ -f "${CREWAI_PID_FILE}" ]]; then
        local pid=$(cat "${CREWAI_PID_FILE}")
        if kill -0 "$pid" 2>/dev/null; then
            log::info "Stopping CrewAI..."
            
            # Clean up any agents with this PID
            if [[ -f "${APP_ROOT}/.vrooli/crewai-agents.json" ]]; then
                local agent_ids
                agent_ids=$(jq -r --arg pid "$pid" '.agents | to_entries[] | select(.value.pid == ($pid | tonumber)) | .key' "${APP_ROOT}/.vrooli/crewai-agents.json" 2>/dev/null || echo "")
                if [[ -n "$agent_ids" ]]; then
                    while IFS= read -r agent_id; do
                        [[ -n "$agent_id" ]] && agents::unregister "$agent_id" 2>/dev/null || true
                    done <<< "$agent_ids"
                fi
            fi
            
            kill "$pid"
            rm -f "${CREWAI_PID_FILE}"
            log::success "CrewAI stopped"
        else
            rm -f "${CREWAI_PID_FILE}"
            log::info "CrewAI was not running"
        fi
    else
        log::info "CrewAI is not running"
    fi
    
    # Clean up any remaining dead agents
    agents::cleanup &>/dev/null || true
}

# Check if running
is_running() {
    if [[ -f "${CREWAI_PID_FILE}" ]]; then
        local pid=$(cat "${CREWAI_PID_FILE}")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        else
            rm -f "${CREWAI_PID_FILE}"
        fi
    fi
    
    # Check if process is listening on port
    if ss -tlnp 2>/dev/null | grep -q ":${CREWAI_PORT}"; then
        return 0
    fi
    
    return 1
}

# List crews
list_crews() {
    if [[ -d "${CREWAI_CREWS_DIR}" ]]; then
        log::header "Available Crews"
        local count=0
        for crew_file in "${CREWAI_CREWS_DIR}"/*.py; do
            if [[ -f "$crew_file" ]]; then
                basename "$crew_file" .py
                ((count++))
            fi
        done
        if [[ $count -eq 0 ]]; then
            log::info "No crews found"
        fi
    else
        log::info "Crews directory not initialized"
    fi
}

# List agents
list_agents() {
    if [[ -d "${CREWAI_AGENTS_DIR}" ]]; then
        log::header "Available Agents"
        local count=0
        for agent_file in "${CREWAI_AGENTS_DIR}"/*.py; do
            if [[ -f "$agent_file" ]]; then
                basename "$agent_file" .py
                ((count++))
            fi
        done
        if [[ $count -eq 0 ]]; then
            log::info "No agents found"
        fi
    else
        log::info "Agents directory not initialized"
    fi
}

# Get health status
get_health() {
    if is_running; then
        curl -s "http://localhost:${CREWAI_PORT}/health" 2>/dev/null || echo "{}"
    else
        echo "{}"
    fi
}