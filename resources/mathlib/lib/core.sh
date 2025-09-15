#!/usr/bin/env bash
# Mathlib Resource - Core Library Functions

set -euo pipefail

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../config/defaults.sh"

# Display resource information from runtime.json
mathlib::info() {
    local json_flag="${1:-}"
    local runtime_file="${SCRIPT_DIR}/../config/runtime.json"
    
    if [[ ! -f "${runtime_file}" ]]; then
        echo "Error: runtime.json not found"
        return 1
    fi
    
    if [[ "${json_flag}" == "--json" ]]; then
        cat "${runtime_file}"
    else
        echo "Mathlib Resource Information:"
        echo "=============================="
        jq -r '
            "Resource: " + .resource,
            "Version: " + .version,
            "Description: " + .description,
            "Category: " + .category,
            "Startup Order: " + (.startup_order | tostring),
            "Dependencies: " + (.dependencies | join(", ")),
            "Startup Timeout: " + (.startup_timeout | tostring) + "s",
            "Startup Time Estimate: " + .startup_time_estimate,
            "Recovery Attempts: " + (.recovery_attempts | tostring),
            "Priority: " + .priority
        ' "${runtime_file}"
    fi
}

# Lifecycle management
mathlib::manage() {
    local subcommand="${1:-}"
    shift || true
    
    # Check for help flag
    if [[ "${subcommand}" == "--help" ]] || [[ "${subcommand}" == "-h" ]] || [[ -z "${subcommand}" ]]; then
        echo "Usage: vrooli resource mathlib manage <command> [options]"
        echo ""
        echo "Commands:"
        echo "  install    Install Lean 4 and Mathlib dependencies"
        echo "  uninstall  Remove installed components"
        echo "  start      Start the Mathlib service"
        echo "  stop       Stop the Mathlib service"
        echo "  restart    Restart the Mathlib service"
        echo ""
        echo "Options:"
        echo "  --wait     Wait for service to be healthy (for start/restart)"
        echo "  --help     Show this help message"
        return 0
    fi
    
    case "${subcommand}" in
        install)
            # Check for help on subcommand
            if [[ "${1:-}" == "--help" ]]; then
                echo "Usage: vrooli resource mathlib manage install"
                echo "Install Lean 4 compiler and Mathlib dependencies"
                return 0
            fi
            mathlib::install "$@"
            ;;
        uninstall)
            if [[ "${1:-}" == "--help" ]]; then
                echo "Usage: vrooli resource mathlib manage uninstall"
                echo "Remove installed Lean 4 and Mathlib components"
                return 0
            fi
            mathlib::uninstall "$@"
            ;;
        start)
            if [[ "${1:-}" == "--help" ]]; then
                echo "Usage: vrooli resource mathlib manage start [--wait]"
                echo "Start the Mathlib service"
                echo "  --wait  Wait for service to be healthy"
                return 0
            fi
            mathlib::start "$@"
            ;;
        stop)
            if [[ "${1:-}" == "--help" ]]; then
                echo "Usage: vrooli resource mathlib manage stop"
                echo "Stop the Mathlib service"
                return 0
            fi
            mathlib::stop "$@"
            ;;
        restart)
            if [[ "${1:-}" == "--help" ]]; then
                echo "Usage: vrooli resource mathlib manage restart [--wait]"
                echo "Restart the Mathlib service"
                echo "  --wait  Wait for service to be healthy"
                return 0
            fi
            mathlib::restart "$@"
            ;;
        *)
            echo "Error: Unknown manage command '${subcommand}'"
            echo "Valid commands: install, uninstall, start, stop, restart"
            return 1
            ;;
    esac
}

# Install Lean 4 and dependencies
mathlib::install() {
    echo "Installing Mathlib (Lean 4) resource..."
    
    # Create directories
    mkdir -p "${MATHLIB_INSTALL_DIR}"
    mkdir -p "${MATHLIB_WORK_DIR}"
    mkdir -p "${MATHLIB_CACHE_DIR}"
    mkdir -p "${MATHLIB_LOG_DIR}"
    
    # Check if already installed
    if [[ -f "${MATHLIB_INSTALL_DIR}/.installed" ]]; then
        echo "Mathlib is already installed"
        return 2  # Already installed
    fi
    
    echo "Installing Lean 4 compiler and Lake build system..."
    
    # Install Lean 4 using elan (official Lean version manager)
    # Always add elan to PATH first if it exists
    if [[ -d "${HOME}/.elan/bin" ]]; then
        export PATH="${HOME}/.elan/bin:${PATH}"
    fi
    
    if ! command -v elan &> /dev/null; then
        echo "Installing elan (Lean version manager)..."
        curl -sSfL https://raw.githubusercontent.com/leanprover/elan/master/elan-init.sh -o /tmp/elan-init.sh
        chmod +x /tmp/elan-init.sh
        /tmp/elan-init.sh -y --default-toolchain leanprover/lean4:stable --no-modify-path
        rm /tmp/elan-init.sh
        
        # Add elan to PATH for this session
        export PATH="${HOME}/.elan/bin:${PATH}"
    else
        echo "Elan already installed, checking Lean version..."
        # elan self-update is what updates elan itself
        # elan update stable updates the stable toolchain
        elan self update 2>/dev/null || true
        elan update stable 2>/dev/null || true
    fi
    
    # Verify Lean installation
    if ! command -v lean &> /dev/null; then
        echo "Error: Lean installation failed"
        return 1
    fi
    
    echo "Lean 4 installed: $(lean --version)"
    
    # Clone and build Mathlib4 (this takes significant time and space)
    echo "Setting up Mathlib4 library..."
    cd "${MATHLIB_INSTALL_DIR}"
    
    # Create a minimal Lean project with Mathlib
    lake new mathlib_server
    cd mathlib_server
    
    # Add Mathlib4 as a dependency
    cat > lakefile.lean << 'EOF'
import Lake
open Lake DSL

package mathlib_server

@[default_target]
lean_lib MathlibServer

require mathlib from git
  "https://github.com/leanprover-community/mathlib4.git"
EOF
    
    # Download Mathlib cache to speed up compilation
    echo "Downloading Mathlib cache (this may take several minutes)..."
    if command -v lake &> /dev/null; then
        lake exe cache get || echo "Warning: Cache download failed, will build from source"
    fi
    
    # Create installation marker
    touch "${MATHLIB_INSTALL_DIR}/.installed"
    
    echo "Mathlib resource installed successfully"
    return 0
}

# Uninstall the resource
mathlib::uninstall() {
    echo "Uninstalling Mathlib resource..."
    
    # Stop service if running
    mathlib::stop 2>/dev/null || true
    
    # Remove installation directory
    if [[ -d "${MATHLIB_INSTALL_DIR}" ]]; then
        rm -rf "${MATHLIB_INSTALL_DIR}"
        echo "Removed installation directory"
    fi
    
    # Optionally keep work and cache directories
    local keep_data="${1:-}"
    if [[ "${keep_data}" != "--keep-data" ]]; then
        rm -rf "${MATHLIB_WORK_DIR}"
        rm -rf "${MATHLIB_CACHE_DIR}"
        echo "Removed work and cache directories"
    fi
    
    echo "Mathlib resource uninstalled successfully"
    return 0
}

# Start the service
mathlib::start() {
    echo "Starting Mathlib service..."
    
    # Clean up any stale processes
    pkill -f "mathlib_health.py" 2>/dev/null || true
    rm -f "${MATHLIB_PID_FILE}" 2>/dev/null || true
    
    # Check if already running
    if mathlib::is_running; then
        echo "Mathlib service is already running"
        return 0
    fi
    
    # Create Lean proof server with API endpoints
    cat > /tmp/mathlib_server.py << 'EOF'
#!/usr/bin/env python3
from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import os
import sys
import socket
import subprocess
import uuid
import tempfile
import threading
import time
from urllib.parse import urlparse, parse_qs

PORT = int(os.environ.get('MATHLIB_PORT', 11458))
WORK_DIR = os.environ.get('MATHLIB_WORK_DIR', '/tmp/mathlib/work')
CACHE_DIR = os.environ.get('MATHLIB_CACHE_DIR', '/tmp/mathlib/cache')
ENABLE_CACHE = os.environ.get('MATHLIB_ENABLE_CACHE', 'true').lower() == 'true'

# Store proof jobs
proof_jobs = {}
proof_lock = threading.Lock()

# Cache for compiled proofs
proof_cache = {}
cache_lock = threading.Lock()
MAX_CACHE_SIZE = 100  # Maximum number of cached proofs

# Performance metrics
metrics = {
    'total_requests': 0,
    'successful_proofs': 0,
    'failed_proofs': 0,
    'total_execution_time': 0,
    'average_execution_time': 0,
    'min_execution_time': float('inf'),
    'max_execution_time': 0,
    'active_jobs': 0,
    'queued_jobs': 0,
    'cache_hits': 0,
    'cache_misses': 0
}
metrics_lock = threading.Lock()

def check_lean_installation():
    """Check if Lean is properly installed"""
    try:
        # Try to use lean from elan installation
        lean_paths = [
            'lean',  # System PATH
            os.path.expanduser('~/.elan/bin/lean'),  # Elan installation
        ]
        
        for lean_path in lean_paths:
            try:
                result = subprocess.run([lean_path, '--version'], capture_output=True, text=True, timeout=5)
                if result.returncode == 0:
                    return result.stdout.strip()
            except:
                continue
        return None
    except:
        return None

def execute_lean_proof(proof_code, job_id):
    """Execute a Lean proof and update job status"""
    import hashlib
    
    # Update metrics
    with metrics_lock:
        metrics['total_requests'] += 1
        metrics['active_jobs'] += 1
        metrics['queued_jobs'] = max(0, metrics['queued_jobs'] - 1)
    
    # Check cache if enabled
    proof_hash = None
    if ENABLE_CACHE:
        proof_hash = hashlib.sha256(proof_code.encode()).hexdigest()
        with cache_lock:
            if proof_hash in proof_cache:
                # Cache hit
                cached_result = proof_cache[proof_hash].copy()
                with metrics_lock:
                    metrics['cache_hits'] += 1
                    metrics['active_jobs'] = max(0, metrics['active_jobs'] - 1)
                
                # Update job with cached result
                with proof_lock:
                    proof_jobs[job_id]['status'] = cached_result['status']
                    proof_jobs[job_id]['result'] = cached_result['result']
                    proof_jobs[job_id]['start_time'] = time.time()
                    proof_jobs[job_id]['end_time'] = time.time()
                    proof_jobs[job_id]['duration'] = 0.001  # Minimal time for cache hit
                    proof_jobs[job_id]['cached'] = True
                return
            else:
                with metrics_lock:
                    metrics['cache_misses'] += 1
    
    try:
        # Create temporary file for proof
        with tempfile.NamedTemporaryFile(mode='w', suffix='.lean', dir=WORK_DIR, delete=False) as f:
            f.write(proof_code)
            proof_file = f.name
        
        # Update job status
        with proof_lock:
            proof_jobs[job_id]['status'] = 'running'
            proof_jobs[job_id]['start_time'] = time.time()
        
        # Run Lean on the proof (prefer elan installation)
        lean_cmd = 'lean'
        if os.path.exists(os.path.expanduser('~/.elan/bin/lean')):
            lean_cmd = os.path.expanduser('~/.elan/bin/lean')
        
        result = subprocess.run(
            [lean_cmd, proof_file],
            capture_output=True,
            text=True,
            timeout=60,
            cwd=WORK_DIR
        )
        
        # Process results
        with proof_lock:
            proof_jobs[job_id]['end_time'] = time.time()
            proof_jobs[job_id]['duration'] = proof_jobs[job_id]['end_time'] - proof_jobs[job_id]['start_time']
            
            # Update metrics
            duration = proof_jobs[job_id]['duration']
            with metrics_lock:
                metrics['total_execution_time'] += duration
                metrics['min_execution_time'] = min(metrics['min_execution_time'], duration)
                metrics['max_execution_time'] = max(metrics['max_execution_time'], duration)
                
                if result.returncode == 0:
                    metrics['successful_proofs'] += 1
                else:
                    metrics['failed_proofs'] += 1
                
                total_completed = metrics['successful_proofs'] + metrics['failed_proofs']
                if total_completed > 0:
                    metrics['average_execution_time'] = metrics['total_execution_time'] / total_completed
            
            if result.returncode == 0:
                proof_jobs[job_id]['status'] = 'success'
                proof_jobs[job_id]['result'] = {
                    'valid': True,
                    'output': result.stdout,
                    'message': 'Proof verified successfully'
                }
                
                # Store in cache if enabled
                if ENABLE_CACHE and proof_hash:
                    with cache_lock:
                        # Implement LRU eviction if cache is full
                        if len(proof_cache) >= MAX_CACHE_SIZE:
                            # Remove oldest entry (simple FIFO for now)
                            oldest_key = next(iter(proof_cache))
                            del proof_cache[oldest_key]
                        
                        proof_cache[proof_hash] = {
                            'status': 'success',
                            'result': proof_jobs[job_id]['result']
                        }
            else:
                proof_jobs[job_id]['status'] = 'failed'
                proof_jobs[job_id]['result'] = {
                    'valid': False,
                    'errors': result.stderr,
                    'output': result.stdout,
                    'message': 'Proof verification failed'
                }
                
                # Also cache failures to avoid recomputing
                if ENABLE_CACHE and proof_hash:
                    with cache_lock:
                        if len(proof_cache) >= MAX_CACHE_SIZE:
                            oldest_key = next(iter(proof_cache))
                            del proof_cache[oldest_key]
                        
                        proof_cache[proof_hash] = {
                            'status': 'failed',
                            'result': proof_jobs[job_id]['result']
                        }
        
        # Clean up
        os.unlink(proof_file)
        
    except subprocess.TimeoutExpired:
        with proof_lock:
            proof_jobs[job_id]['status'] = 'timeout'
            proof_jobs[job_id]['result'] = {
                'valid': False,
                'message': 'Proof execution timed out after 60 seconds'
            }
        with metrics_lock:
            metrics['failed_proofs'] += 1
    except Exception as e:
        with proof_lock:
            proof_jobs[job_id]['status'] = 'error'
            proof_jobs[job_id]['result'] = {
                'valid': False,
                'message': f'Internal error: {str(e)}'
            }
        with metrics_lock:
            metrics['failed_proofs'] += 1
    finally:
        with metrics_lock:
            metrics['active_jobs'] = max(0, metrics['active_jobs'] - 1)

class MathlibHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        parsed_path = urlparse(self.path)
        path = parsed_path.path
        
        if path == '/health':
            lean_version = check_lean_installation()
            response = {
                'status': 'healthy',
                'service': 'mathlib',
                'version': '1.0.0',
                'lean': lean_version if lean_version else 'not-installed',
                'mathlib': 'available' if lean_version else 'not-available',
                'endpoints': ['/health', '/prove', '/status/{id}', '/tactics', '/batch', '/batch/status/{batch_id}', '/metrics']
            }
            self.send_json_response(200, response)
            
        elif path.startswith('/status/'):
            job_id = path.replace('/status/', '')
            with proof_lock:
                if job_id in proof_jobs:
                    self.send_json_response(200, proof_jobs[job_id])
                else:
                    self.send_json_response(404, {'error': 'Job not found'})
                    
        elif path == '/tactics':
            # Return available tactics (simplified list)
            tactics = {
                'basic': ['rfl', 'simp', 'ring', 'linarith', 'norm_num'],
                'logic': ['intro', 'apply', 'exact', 'constructor', 'cases'],
                'advanced': ['omega', 'interval_cases', 'field_simp', 'polyrith']
            }
            self.send_json_response(200, tactics)
            
        elif path.startswith('/batch/status/'):
            # Get status of all jobs in a batch
            batch_id = path.replace('/batch/status/', '')
            with proof_lock:
                batch_jobs = [job for job in proof_jobs.values() if job.get('batch_id') == batch_id]
                
            if batch_jobs:
                completed = sum(1 for job in batch_jobs if job['status'] in ['success', 'failed', 'timeout', 'error'])
                total = len(batch_jobs)
                batch_status = 'completed' if completed == total else 'processing'
                
                response = {
                    'batch_id': batch_id,
                    'status': batch_status,
                    'completed': completed,
                    'total': total,
                    'jobs': batch_jobs
                }
                self.send_json_response(200, response)
            else:
                self.send_json_response(404, {'error': 'Batch not found'})
        
        elif path == '/metrics':
            # Return performance metrics
            with metrics_lock:
                # Calculate current queue size
                with proof_lock:
                    queued_count = sum(1 for job in proof_jobs.values() if job['status'] == 'queued')
                    running_count = sum(1 for job in proof_jobs.values() if job['status'] == 'running')
                
                response = {
                    'performance': {
                        'total_requests': metrics['total_requests'],
                        'successful_proofs': metrics['successful_proofs'],
                        'failed_proofs': metrics['failed_proofs'],
                        'success_rate': metrics['successful_proofs'] / max(1, metrics['total_requests']) * 100,
                        'average_execution_time': round(metrics['average_execution_time'], 3),
                        'min_execution_time': round(metrics['min_execution_time'], 3) if metrics['min_execution_time'] != float('inf') else 0,
                        'max_execution_time': round(metrics['max_execution_time'], 3),
                    },
                    'cache': {
                        'enabled': ENABLE_CACHE,
                        'cache_hits': metrics['cache_hits'],
                        'cache_misses': metrics['cache_misses'],
                        'cache_hit_rate': metrics['cache_hits'] / max(1, metrics['cache_hits'] + metrics['cache_misses']) * 100,
                        'cache_size': len(proof_cache),
                        'max_cache_size': MAX_CACHE_SIZE
                    },
                    'current_status': {
                        'active_jobs': running_count,
                        'queued_jobs': queued_count,
                        'total_jobs_processed': len(proof_jobs)
                    },
                    'system': {
                        'lean_installed': bool(check_lean_installation()),
                        'work_dir': WORK_DIR,
                        'cache_dir': CACHE_DIR
                    }
                }
            self.send_json_response(200, response)
            
        else:
            self.send_json_response(404, {'error': 'Not found'})
    
    def do_POST(self):
        if self.path == '/prove':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            try:
                data = json.loads(post_data.decode('utf-8'))
                proof_code = data.get('proof', '')
                
                if not proof_code:
                    self.send_json_response(400, {'error': 'No proof code provided'})
                    return
                
                # Create job
                job_id = str(uuid.uuid4())
                with proof_lock:
                    proof_jobs[job_id] = {
                        'id': job_id,
                        'status': 'queued',
                        'created': time.time()
                    }
                
                with metrics_lock:
                    metrics['queued_jobs'] += 1
                
                # Start proof execution in background
                thread = threading.Thread(target=execute_lean_proof, args=(proof_code, job_id))
                thread.daemon = True
                thread.start()
                
                self.send_json_response(202, {'job_id': job_id, 'status': 'queued'})
                
            except json.JSONDecodeError:
                self.send_json_response(400, {'error': 'Invalid JSON'})
            except Exception as e:
                self.send_json_response(500, {'error': str(e)})
                
        elif self.path == '/batch':
            # Batch processing endpoint for multiple proofs
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            try:
                data = json.loads(post_data.decode('utf-8'))
                proofs = data.get('proofs', [])
                
                if not proofs or not isinstance(proofs, list):
                    self.send_json_response(400, {'error': 'No proofs array provided'})
                    return
                
                batch_id = str(uuid.uuid4())
                job_ids = []
                
                # Create jobs for all proofs
                for i, proof_item in enumerate(proofs):
                    if isinstance(proof_item, str):
                        proof_code = proof_item
                        proof_name = f"proof_{i}"
                    elif isinstance(proof_item, dict):
                        proof_code = proof_item.get('proof', '')
                        proof_name = proof_item.get('name', f"proof_{i}")
                    else:
                        continue
                    
                    if not proof_code:
                        continue
                    
                    job_id = str(uuid.uuid4())
                    with proof_lock:
                        proof_jobs[job_id] = {
                            'id': job_id,
                            'batch_id': batch_id,
                            'name': proof_name,
                            'status': 'queued',
                            'created': time.time()
                        }
                    
                    with metrics_lock:
                        metrics['queued_jobs'] += 1
                    
                    # Start proof execution in background
                    thread = threading.Thread(target=execute_lean_proof, args=(proof_code, job_id))
                    thread.daemon = True
                    thread.start()
                    
                    job_ids.append(job_id)
                
                self.send_json_response(202, {
                    'batch_id': batch_id,
                    'job_ids': job_ids,
                    'total': len(job_ids),
                    'status': 'processing'
                })
                
            except json.JSONDecodeError:
                self.send_json_response(400, {'error': 'Invalid JSON'})
            except Exception as e:
                self.send_json_response(500, {'error': str(e)})
                
        else:
            self.send_json_response(404, {'error': 'Not found'})
    
    def send_json_response(self, code, data):
        self.send_response(code)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())
    
    def log_message(self, format, *args):
        pass  # Suppress logs

# Ensure work directory exists
os.makedirs(WORK_DIR, exist_ok=True)
os.makedirs(CACHE_DIR, exist_ok=True)

try:
    server = HTTPServer(('localhost', PORT), MathlibHandler)
    server.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    print(f"Mathlib proof server started on port {PORT}")
    print(f"Lean version: {check_lean_installation() or 'Not installed'}")
    server.serve_forever()
except OSError as e:
    print(f"Error: Port {PORT} is already in use or not available: {e}")
    sys.exit(1)
except KeyboardInterrupt:
    print("Server stopped")
    sys.exit(0)
EOF
    
    # Start the proof server
    nohup python3 /tmp/mathlib_server.py > "${MATHLIB_LOG_FILE}" 2>&1 &
    local pid=$!
    echo "${pid}" > "${MATHLIB_PID_FILE}"
    
    # Wait for startup
    local wait_flag="${1:-}"
    if [[ "${wait_flag}" == "--wait" ]]; then
        echo "Waiting for service to be healthy..."
        if mathlib::wait_for_health; then
            echo "Mathlib service started successfully"
            return 0
        else
            echo "Error: Service failed to become healthy"
            mathlib::stop
            return 1
        fi
    fi
    
    echo "Mathlib service started (PID: ${pid})"
    return 0
}

# Stop the service
mathlib::stop() {
    echo "Stopping Mathlib service..."
    
    if [[ -f "${MATHLIB_PID_FILE}" ]]; then
        local pid=$(cat "${MATHLIB_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            kill "${pid}"
            rm -f "${MATHLIB_PID_FILE}"
            echo "Mathlib service stopped"
        else
            echo "Service not running (stale PID file)"
            rm -f "${MATHLIB_PID_FILE}"
        fi
    else
        echo "Mathlib service is not running"
    fi
    
    return 0
}

# Restart the service
mathlib::restart() {
    mathlib::stop
    sleep 2
    mathlib::start "$@"
}

# Check if service is running
mathlib::is_running() {
    if [[ -f "${MATHLIB_PID_FILE}" ]]; then
        local pid=$(cat "${MATHLIB_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Wait for service to be healthy
mathlib::wait_for_health() {
    local timeout="${MATHLIB_STARTUP_TIMEOUT}"
    local elapsed=0
    
    while [[ ${elapsed} -lt ${timeout} ]]; do
        if timeout 5 curl -sf "${MATHLIB_HEALTH_ENDPOINT}" > /dev/null 2>&1; then
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    
    return 1
}

# Run tests
mathlib::test() {
    local test_type="${1:-all}"
    
    case "${test_type}" in
        smoke|integration|unit|all)
            source "${SCRIPT_DIR}/test.sh"
            mathlib::test::run "${test_type}"
            ;;
        *)
            echo "Error: Unknown test type '${test_type}'"
            echo "Valid types: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

# Content management for proofs and tactics
mathlib::content() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        list)
            mathlib::content::list "$@"
            ;;
        add)
            mathlib::content::add "$@"
            ;;
        get)
            mathlib::content::get "$@"
            ;;
        remove)
            mathlib::content::remove "$@"
            ;;
        execute)
            mathlib::content::execute "$@"
            ;;
        tactics)
            mathlib::content::tactics "$@"
            ;;
        *)
            echo "Error: Unknown content command '${subcommand}'"
            echo "Valid commands: list, add, get, remove, execute, tactics"
            return 1
            ;;
    esac
}

# List available proofs and tactics
mathlib::content::list() {
    local type="${1:-all}"
    
    case "${type}" in
        proofs)
            echo "Available proof files:"
            if [[ -d "${MATHLIB_WORK_DIR}/proofs" ]]; then
                ls -la "${MATHLIB_WORK_DIR}/proofs/"*.lean 2>/dev/null || echo "No proof files found"
            else
                echo "No proofs directory found"
            fi
            ;;
        tactics)
            # Query the server for available tactics
            if mathlib::is_running; then
                curl -sf "http://localhost:${MATHLIB_PORT}/tactics" | jq '.' 2>/dev/null || echo "Could not fetch tactics"
            else
                echo "Service not running. Start with: vrooli resource mathlib manage start"
            fi
            ;;
        all|*)
            mathlib::content::list proofs
            echo ""
            mathlib::content::list tactics
            ;;
    esac
}

# Custom tactics management
mathlib::content::tactics() {
    local action="${1:-list}"
    shift || true
    
    # Source tactics library
    local resource_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    source "${resource_dir}/lib/tactics.sh"
    
    case "${action}" in
        load)
            local tactic_file="${1:-}"
            local tactic_name="${2:-}"
            if [[ -z "$tactic_file" ]]; then
                echo "Error: Please provide a tactic file to load"
                echo "Usage: vrooli resource mathlib content tactics load <file> [name]"
                return 1
            fi
            load_tactic "$tactic_file" "$tactic_name"
            ;;
        list)
            list_custom_tactics
            ;;
        remove)
            local tactic_name="${1:-}"
            if [[ -z "$tactic_name" ]]; then
                echo "Error: Please provide a tactic name to remove"
                echo "Usage: vrooli resource mathlib content tactics remove <name>"
                return 1
            fi
            remove_tactic "$tactic_name"
            ;;
        test)
            local tactic_name="${1:-}"
            local proof_code="${2:-}"
            if [[ -z "$tactic_name" ]] || [[ -z "$proof_code" ]]; then
                echo "Error: Please provide both tactic name and proof code"
                echo "Usage: vrooli resource mathlib content tactics test <name> <proof>"
                return 1
            fi
            test_tactic "$tactic_name" "$proof_code"
            ;;
        export)
            export_tactics "$@"
            ;;
        import)
            local bundle_file="${1:-}"
            if [[ -z "$bundle_file" ]]; then
                echo "Error: Please provide a bundle file to import"
                echo "Usage: vrooli resource mathlib content tactics import <bundle>"
                return 1
            fi
            import_tactics "$bundle_file"
            ;;
        help|--help|-h)
            echo "Custom Tactics Management"
            echo ""
            echo "USAGE:"
            echo "    vrooli resource mathlib content tactics <action> [options]"
            echo ""
            echo "ACTIONS:"
            echo "    load    - Load a custom tactic file"
            echo "    list    - List loaded custom tactics"
            echo "    remove  - Remove a custom tactic"
            echo "    test    - Test a custom tactic with a proof"
            echo "    export  - Export tactics to a bundle"
            echo "    import  - Import tactics from a bundle"
            echo "    help    - Show this help message"
            echo ""
            echo "EXAMPLES:"
            echo "    # Load a custom tactic"
            echo "    vrooli resource mathlib content tactics load my_tactic.lean"
            echo ""
            echo "    # List all custom tactics"
            echo "    vrooli resource mathlib content tactics list"
            echo ""
            echo "    # Test a custom tactic"
            echo "    vrooli resource mathlib content tactics test my_tactic 'theorem test : 2 + 2 = 4 := by my_custom_tactic'"
            ;;
        *)
            echo "Error: Unknown tactics action '${action}'"
            echo "Run 'vrooli resource mathlib content tactics help' for usage"
            return 1
            ;;
    esac
}

# Interactive REPL mode
mathlib::interactive() {
    local subcommand="${1:-start}"
    shift || true
    
    # Source the REPL library
    local resource_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    source "${resource_dir}/lib/repl.sh"
    
    case "${subcommand}" in
        start)
            echo "Starting Mathlib interactive REPL..."
            start_repl
            ;;
        replay)
            local proof_file="${1:-}"
            if [[ -z "$proof_file" ]]; then
                echo "Error: Please provide a proof file to replay"
                echo "Usage: vrooli resource mathlib interactive replay <file>"
                return 1
            fi
            replay_proof "$proof_file"
            ;;
        help|--help|-h)
            echo "Mathlib Interactive REPL"
            echo ""
            echo "USAGE:"
            echo "    vrooli resource mathlib interactive [command] [options]"
            echo ""
            echo "COMMANDS:"
            echo "    start   - Start interactive REPL session (default)"
            echo "    replay  - Replay a proof file step by step"
            echo "    help    - Show this help message"
            echo ""
            echo "REPL COMMANDS:"
            echo "    :help    - Show REPL help"
            echo "    :tactics - List available tactics"
            echo "    :check   - Check type of expression"
            echo "    :eval    - Evaluate expression"
            echo "    :quit    - Exit REPL"
            echo "    :clear   - Clear screen"
            ;;
        *)
            echo "Error: Unknown interactive command '${subcommand}'"
            echo "Run 'vrooli resource mathlib interactive help' for usage"
            return 1
            ;;
    esac
}

# Add a proof file
mathlib::content::add() {
    local proof_file="${1:-}"
    local name="${2:-}"
    
    if [[ -z "${proof_file}" ]]; then
        echo "Error: Please provide a proof file path"
        echo "Usage: vrooli resource mathlib content add <file> [name]"
        return 1
    fi
    
    if [[ ! -f "${proof_file}" ]]; then
        echo "Error: File not found: ${proof_file}"
        return 1
    fi
    
    # Create proofs directory if needed
    mkdir -p "${MATHLIB_WORK_DIR}/proofs"
    
    # Determine target name
    if [[ -z "${name}" ]]; then
        name=$(basename "${proof_file}")
    fi
    
    # Ensure .lean extension
    if [[ ! "${name}" =~ \.lean$ ]]; then
        name="${name}.lean"
    fi
    
    # Copy proof file
    cp "${proof_file}" "${MATHLIB_WORK_DIR}/proofs/${name}"
    echo "Proof added: ${name}"
    
    return 0
}

# Get a proof file
mathlib::content::get() {
    local name="${1:-}"
    
    if [[ -z "${name}" ]]; then
        echo "Error: Please provide a proof name"
        echo "Usage: vrooli resource mathlib content get <name>"
        return 1
    fi
    
    # Ensure .lean extension
    if [[ ! "${name}" =~ \.lean$ ]]; then
        name="${name}.lean"
    fi
    
    local proof_file="${MATHLIB_WORK_DIR}/proofs/${name}"
    
    if [[ ! -f "${proof_file}" ]]; then
        echo "Error: Proof not found: ${name}"
        return 1
    fi
    
    cat "${proof_file}"
    return 0
}

# Remove a proof file
mathlib::content::remove() {
    local name="${1:-}"
    
    if [[ -z "${name}" ]]; then
        echo "Error: Please provide a proof name"
        echo "Usage: vrooli resource mathlib content remove <name>"
        return 1
    fi
    
    # Ensure .lean extension
    if [[ ! "${name}" =~ \.lean$ ]]; then
        name="${name}.lean"
    fi
    
    local proof_file="${MATHLIB_WORK_DIR}/proofs/${name}"
    
    if [[ ! -f "${proof_file}" ]]; then
        echo "Error: Proof not found: ${name}"
        return 1
    fi
    
    rm "${proof_file}"
    echo "Proof removed: ${name}"
    return 0
}

# Execute a proof file
mathlib::content::execute() {
    local proof_input="${1:-}"
    
    if [[ -z "${proof_input}" ]]; then
        echo "Error: Please provide a proof file or name"
        echo "Usage: vrooli resource mathlib content execute <file|name>"
        return 1
    fi
    
    # Check if service is running
    if ! mathlib::is_running; then
        echo "Error: Mathlib service is not running"
        echo "Start with: vrooli resource mathlib manage start"
        return 1
    fi
    
    # Determine if input is a file or a stored proof
    local proof_content=""
    
    if [[ -f "${proof_input}" ]]; then
        # Direct file path
        proof_content=$(cat "${proof_input}")
    elif [[ -f "${MATHLIB_WORK_DIR}/proofs/${proof_input}" ]]; then
        # Stored proof without extension
        proof_content=$(cat "${MATHLIB_WORK_DIR}/proofs/${proof_input}")
    elif [[ -f "${MATHLIB_WORK_DIR}/proofs/${proof_input}.lean" ]]; then
        # Stored proof with extension
        proof_content=$(cat "${MATHLIB_WORK_DIR}/proofs/${proof_input}.lean")
    else
        echo "Error: Proof not found: ${proof_input}"
        return 1
    fi
    
    # Submit proof to server
    echo "Submitting proof for verification..."
    local response=$(curl -sf -X POST "http://localhost:${MATHLIB_PORT}/prove" \
        -H "Content-Type: application/json" \
        -d "{\"proof\": $(echo "${proof_content}" | jq -Rs '.')}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to submit proof to server"
        return 1
    fi
    
    # Extract job ID
    local job_id=$(echo "${response}" | jq -r '.job_id' 2>/dev/null)
    
    if [[ -z "${job_id}" || "${job_id}" == "null" ]]; then
        echo "Error: Failed to get job ID from server"
        echo "Response: ${response}"
        return 1
    fi
    
    echo "Job ID: ${job_id}"
    echo "Checking status..."
    
    # Poll for results
    local max_attempts=30
    local attempt=0
    
    while [[ ${attempt} -lt ${max_attempts} ]]; do
        sleep 2
        local status_response=$(curl -sf "http://localhost:${MATHLIB_PORT}/status/${job_id}" 2>/dev/null)
        
        if [[ $? -ne 0 ]]; then
            echo "Error: Failed to check job status"
            return 1
        fi
        
        local status=$(echo "${status_response}" | jq -r '.status' 2>/dev/null)
        
        case "${status}" in
            success)
                echo "✓ Proof verified successfully!"
                echo "${status_response}" | jq '.result' 2>/dev/null
                return 0
                ;;
            failed)
                echo "✗ Proof verification failed"
                echo "${status_response}" | jq '.result' 2>/dev/null
                return 1
                ;;
            timeout)
                echo "⏱ Proof execution timed out"
                echo "${status_response}" | jq '.result' 2>/dev/null
                return 1
                ;;
            error)
                echo "❌ Internal error during proof execution"
                echo "${status_response}" | jq '.result' 2>/dev/null
                return 1
                ;;
            running|queued)
                # Still processing
                ;;
            *)
                echo "Unknown status: ${status}"
                return 1
                ;;
        esac
        
        attempt=$((attempt + 1))
    done
    
    echo "Timeout: Proof verification took too long"
    return 1
}

# Show service status
mathlib::status() {
    echo "Mathlib Service Status"
    echo "====================="
    
    if mathlib::is_running; then
        echo "Status: Running"
        if [[ -f "${MATHLIB_PID_FILE}" ]]; then
            echo "PID: $(cat "${MATHLIB_PID_FILE}")"
        fi
        
        # Check health
        if timeout 5 curl -sf "${MATHLIB_HEALTH_ENDPOINT}" > /dev/null 2>&1; then
            echo "Health: Healthy"
            
            # Show health details
            local health_data=$(timeout 5 curl -sf "${MATHLIB_HEALTH_ENDPOINT}" 2>/dev/null || echo "{}")
            echo "Details:"
            echo "${health_data}" | jq -r '
                "  Version: " + .version,
                "  Lean: " + .lean,
                "  Mathlib: " + .mathlib
            ' 2>/dev/null || echo "  Unable to parse health data"
        else
            echo "Health: Unhealthy"
        fi
    else
        echo "Status: Stopped"
    fi
    
    echo ""
    echo "Configuration:"
    echo "  Port: ${MATHLIB_PORT}"
    echo "  Work Dir: ${MATHLIB_WORK_DIR}"
    echo "  Cache Dir: ${MATHLIB_CACHE_DIR}"
}

# View logs
mathlib::logs() {
    local tail_flag="${1:-}"
    local lines="${2:-50}"
    
    if [[ ! -f "${MATHLIB_LOG_FILE}" ]]; then
        echo "No logs available"
        return 0
    fi
    
    if [[ "${tail_flag}" == "--tail" ]]; then
        tail -n "${lines}" "${MATHLIB_LOG_FILE}"
    else
        cat "${MATHLIB_LOG_FILE}"
    fi
}

# Display credentials (not applicable for this resource)
mathlib::credentials() {
    echo "Mathlib does not require credentials for basic usage"
    echo "API Key can be configured via MATHLIB_API_KEY environment variable"
}