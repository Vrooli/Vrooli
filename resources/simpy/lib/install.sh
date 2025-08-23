#!/usr/bin/env bash
# SimPy installation module

# Get script directory
SIMPY_INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
# shellcheck disable=SC1091
source "${SIMPY_INSTALL_DIR}/core.sh"

#######################################
# Install SimPy
#######################################
simpy::install() {
    log::header "Installing SimPy"
    
    # Initialize environment
    simpy::init
    
    # Check if already installed
    if simpy::is_installed; then
        log::info "SimPy is already installed"
        return 0
    fi
    
    # Install Python if not available
    if ! command -v python3 &>/dev/null; then
        log::error "Python 3 is required but not found"
        return 1
    fi
    
    # Install SimPy and dependencies using system pip
    log::info "Installing SimPy and dependencies..."
    if ! python3 -c "import simpy" 2>/dev/null; then
        pip3 install --user --break-system-packages simpy >/dev/null 2>&1 || {
            log::error "Failed to install SimPy. You may need to install pip3 or use sudo"
            return 1
        }
    fi
    
    # Install other core packages
    log::info "Installing core dependencies..."
    for package in numpy pandas matplotlib scipy networkx; do
        if ! python3 -c "import $package" 2>/dev/null; then
            pip3 install --user --break-system-packages "$package" >/dev/null 2>&1 || true
        fi
    done
    
    # Try to install extra packages (non-critical)
    log::info "Installing visualization and analysis tools (optional)..."
    for package in plotly seaborn; do
        if ! python3 -c "import $package" 2>/dev/null; then
            pip3 install --user --break-system-packages "$package" >/dev/null 2>&1 || true
        fi
    done
    
    # Create service script
    log::info "Creating SimPy service..."
    cat > "$SIMPY_DATA_DIR/simpy-service.py" << 'EOF'
#!/usr/bin/env python3
"""SimPy Service - REST API for simulation execution"""

import json
import os
import sys
import time
import traceback
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs
import subprocess
import tempfile
import simpy
import numpy as np

PORT = int(os.environ.get('SIMPY_PORT', 9510))
HOST = os.environ.get('SIMPY_HOST', 'localhost')
RESULTS_DIR = os.environ.get('SIMPY_RESULTS_DIR', '/tmp/simpy-results')

class SimPyHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        """Handle GET requests"""
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                'status': 'healthy',
                'version': simpy.__version__,
                'timestamp': time.time()
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif parsed_path.path == '/version':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                'simpy': simpy.__version__,
                'numpy': np.__version__,
                'python': sys.version
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif parsed_path.path == '/examples':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            examples_dir = os.environ.get('SIMPY_EXAMPLES_DIR', '')
            examples = []
            if os.path.exists(examples_dir):
                for file in os.listdir(examples_dir):
                    if file.endswith('.py'):
                        examples.append(file[:-3])
            self.wfile.write(json.dumps(examples).encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        """Handle POST requests"""
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == '/simulate':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            try:
                data = json.loads(post_data.decode())
                code = data.get('code', '')
                
                # Create temporary file for simulation
                with tempfile.NamedTemporaryFile(mode='w', suffix='.py', delete=False) as f:
                    f.write(code)
                    temp_file = f.name
                
                # Run simulation
                result = subprocess.run(
                    [sys.executable, temp_file],
                    capture_output=True,
                    text=True,
                    timeout=60
                )
                
                # Clean up
                os.unlink(temp_file)
                
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                response = {
                    'success': result.returncode == 0,
                    'output': result.stdout,
                    'error': result.stderr
                }
                self.wfile.write(json.dumps(response).encode())
                
            except Exception as e:
                self.send_response(500)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                response = {
                    'success': False,
                    'error': str(e)
                }
                self.wfile.write(json.dumps(response).encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        """Suppress default logging"""
        pass

def main():
    """Start SimPy service"""
    print(f"Starting SimPy service on {HOST}:{PORT}")
    server = HTTPServer((HOST, PORT), SimPyHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down SimPy service")
        server.shutdown()

if __name__ == '__main__':
    main()
EOF
    
    chmod +x "$SIMPY_DATA_DIR/simpy-service.py"
    
    # Install resource CLI
    log::info "Installing SimPy CLI..."
    "${SIMPY_INSTALL_DIR}/../../../../lib/resources/install-resource-cli.sh" "execution/simpy"
    
    # Create examples
    simpy::create_examples
    
    log::success "SimPy installation complete"
    return 0
}

#######################################
# Create example simulations
#######################################
simpy::create_examples() {
    log::info "Creating example simulations..."
    
    # Example 1: Simple machine shop
    cat > "$SIMPY_EXAMPLES_DIR/machine_shop.py" << 'EOF'
"""Simple machine shop simulation example"""
import simpy
import random
import json

def machine(env, name, repair_time):
    """A machine that occasionally breaks down"""
    while True:
        # Work for a random time
        work_time = random.expovariate(1/10.0)
        yield env.timeout(work_time)
        
        # Machine breaks down
        print(f'{env.now:.2f}: {name} breaks down')
        
        # Repair time
        yield env.timeout(repair_time)
        print(f'{env.now:.2f}: {name} repaired')

def run_simulation():
    """Run the machine shop simulation"""
    random.seed(42)
    env = simpy.Environment()
    
    # Create machines
    for i in range(3):
        env.process(machine(env, f'Machine {i}', 5))
    
    # Run simulation
    env.run(until=50)
    
    return {'simulation': 'complete', 'time': 50}

if __name__ == '__main__':
    result = run_simulation()
    print(json.dumps(result))
EOF
    
    # Example 2: Bank queue simulation
    cat > "$SIMPY_EXAMPLES_DIR/bank_queue.py" << 'EOF'
"""Bank queue simulation example"""
import simpy
import random
import json

class Bank:
    def __init__(self, env, num_tellers):
        self.env = env
        self.teller = simpy.Resource(env, num_tellers)
        self.customers_served = 0
        self.wait_times = []
    
    def serve_customer(self, customer_id):
        """Serve a customer"""
        arrival_time = self.env.now
        
        with self.teller.request() as request:
            yield request
            wait_time = self.env.now - arrival_time
            self.wait_times.append(wait_time)
            
            # Service time
            service_time = random.expovariate(1/3.0)
            yield self.env.timeout(service_time)
            
            self.customers_served += 1
            print(f'Customer {customer_id} served at {self.env.now:.2f} (waited {wait_time:.2f})')

def customer_arrivals(env, bank):
    """Generate customer arrivals"""
    customer_id = 0
    while True:
        # Time between arrivals
        yield env.timeout(random.expovariate(1/2.0))
        customer_id += 1
        env.process(bank.serve_customer(customer_id))

def run_simulation():
    """Run bank simulation"""
    random.seed(42)
    env = simpy.Environment()
    
    bank = Bank(env, num_tellers=2)
    env.process(customer_arrivals(env, bank))
    
    env.run(until=50)
    
    avg_wait = sum(bank.wait_times) / len(bank.wait_times) if bank.wait_times else 0
    
    return {
        'customers_served': bank.customers_served,
        'average_wait_time': avg_wait,
        'simulation_time': 50
    }

if __name__ == '__main__':
    result = run_simulation()
    print(json.dumps(result))
EOF
    
    log::success "Examples created"
}