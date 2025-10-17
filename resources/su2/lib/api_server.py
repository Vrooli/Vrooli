#!/usr/bin/env python3
"""
SU2 API Server
Provides REST API for CFD simulation management
"""

import os
import json
import subprocess
import threading
import uuid
import time
from datetime import datetime
from pathlib import Path
from flask import Flask, request, jsonify
from flask_cors import CORS

app = Flask(__name__)
CORS(app)

# Configuration
PORT = int(os.environ.get('SU2_PORT', 9514))
MESH_DIR = Path('/workspace/meshes')
CONFIG_DIR = Path('/workspace/configs')
RESULTS_DIR = Path('/workspace/results')
CACHE_DIR = Path('/workspace/cache')

# Ensure directories exist
for dir_path in [MESH_DIR, CONFIG_DIR, RESULTS_DIR, CACHE_DIR]:
    dir_path.mkdir(parents=True, exist_ok=True)

# Simulation tracking
simulations = {}
simulation_lock = threading.Lock()

@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'su2',
        'version': '7.5.1',
        'timestamp': datetime.now().isoformat()
    })

@app.route('/api/status', methods=['GET'])
def api_status():
    """Get API status and capabilities"""
    return jsonify({
        'status': 'running',
        'capabilities': [
            'cfd_simulation',
            'mesh_processing',
            'adjoint_optimization',
            'multi_physics',
            'mpi_parallel'
        ],
        'available_solvers': [
            'EULER',
            'NAVIER_STOKES',
            'RANS',
            'HEAT',
            'ELASTICITY'
        ],
        'mpi_processes': int(os.environ.get('SU2_MPI_PROCESSES', 4))
    })

@app.route('/api/designs', methods=['GET'])
def list_designs():
    """List available meshes and configurations"""
    meshes = [f.name for f in MESH_DIR.glob('*.su2')]
    configs = [f.name for f in CONFIG_DIR.glob('*.cfg')]
    
    return jsonify({
        'meshes': meshes,
        'configs': configs,
        'count': {
            'meshes': len(meshes),
            'configs': len(configs)
        }
    })

@app.route('/api/designs/<name>', methods=['POST'])
def upload_design(name):
    """Upload mesh or configuration file"""
    if 'file' not in request.files:
        return jsonify({'error': 'No file provided'}), 400
    
    file = request.files['file']
    if file.filename == '':
        return jsonify({'error': 'No file selected'}), 400
    
    # Determine file type and destination
    if name.endswith('.su2') or name.endswith('.cgns'):
        dest = MESH_DIR / name
    elif name.endswith('.cfg'):
        dest = CONFIG_DIR / name
    else:
        return jsonify({'error': 'Unsupported file type'}), 400
    
    file.save(str(dest))
    
    return jsonify({
        'message': 'File uploaded successfully',
        'filename': name,
        'path': str(dest)
    })

@app.route('/api/designs/<name>', methods=['GET'])
def download_design(name):
    """Download mesh or configuration file"""
    # Try mesh directory first
    file_path = MESH_DIR / name
    if not file_path.exists():
        # Try config directory
        file_path = CONFIG_DIR / name
    
    if not file_path.exists():
        return jsonify({'error': 'File not found'}), 404
    
    with open(file_path, 'r') as f:
        content = f.read()
    
    return content, 200, {'Content-Type': 'text/plain'}

@app.route('/api/simulate', methods=['POST'])
def run_simulation():
    """Submit CFD simulation"""
    data = request.get_json()
    
    mesh = data.get('mesh')
    config = data.get('config')
    
    if not mesh or not config:
        return jsonify({'error': 'Mesh and config required'}), 400
    
    # Check if files exist
    mesh_path = MESH_DIR / mesh
    config_path = CONFIG_DIR / config
    
    if not mesh_path.exists():
        return jsonify({'error': f'Mesh not found: {mesh}'}), 404
    if not config_path.exists():
        return jsonify({'error': f'Config not found: {config}'}), 404
    
    # Generate simulation ID
    sim_id = str(uuid.uuid4())[:8]
    result_dir = RESULTS_DIR / sim_id
    result_dir.mkdir(parents=True, exist_ok=True)
    
    # Prepare simulation
    with simulation_lock:
        simulations[sim_id] = {
            'id': sim_id,
            'mesh': mesh,
            'config': config,
            'status': 'queued',
            'start_time': datetime.now().isoformat(),
            'result_dir': str(result_dir)
        }
    
    # Start simulation in background
    thread = threading.Thread(target=run_su2_simulation, args=(sim_id,))
    thread.start()
    
    return jsonify({
        'simulation_id': sim_id,
        'status': 'queued',
        'message': 'Simulation submitted successfully'
    })

@app.route('/api/results/<sim_id>', methods=['GET'])
def get_results(sim_id):
    """Get simulation results"""
    with simulation_lock:
        if sim_id not in simulations:
            return jsonify({'error': 'Simulation not found'}), 404
        
        sim_info = simulations[sim_id]
    
    result_dir = Path(sim_info['result_dir'])
    
    # List result files
    result_files = []
    if result_dir.exists():
        for file in result_dir.iterdir():
            result_files.append({
                'name': file.name,
                'size': file.stat().st_size,
                'modified': datetime.fromtimestamp(file.stat().st_mtime).isoformat()
            })
    
    return jsonify({
        'simulation': sim_info,
        'files': result_files
    })

@app.route('/api/results/<sim_id>/convergence', methods=['GET'])
def get_convergence(sim_id):
    """Get convergence history"""
    with simulation_lock:
        if sim_id not in simulations:
            return jsonify({'error': 'Simulation not found'}), 404
        
        sim_info = simulations[sim_id]
    
    result_dir = Path(sim_info['result_dir'])
    history_file = result_dir / 'history.csv'
    
    if not history_file.exists():
        return jsonify({'error': 'Convergence history not available'}), 404
    
    # Read convergence data
    convergence_data = []
    with open(history_file, 'r') as f:
        lines = f.readlines()
        if len(lines) > 1:
            headers = lines[0].strip().split(',')
            for line in lines[1:]:
                values = line.strip().split(',')
                convergence_data.append(dict(zip(headers, values)))
    
    return jsonify({
        'simulation_id': sim_id,
        'convergence': convergence_data[:100]  # Limit to last 100 iterations
    })

def run_su2_simulation(sim_id):
    """Execute SU2 simulation"""
    with simulation_lock:
        sim_info = simulations[sim_id]
        sim_info['status'] = 'running'
    
    mesh_path = MESH_DIR / sim_info['mesh']
    config_path = CONFIG_DIR / sim_info['config']
    result_dir = Path(sim_info['result_dir'])
    
    try:
        # Copy config to result directory with modifications
        result_config = result_dir / 'simulation.cfg'
        with open(config_path, 'r') as f:
            config_content = f.read()
        
        # Update mesh path in config
        config_content = config_content.replace(
            f"MESH_FILENAME= {sim_info['mesh']}",
            f"MESH_FILENAME= {mesh_path}"
        )
        
        with open(result_config, 'w') as f:
            f.write(config_content)
        
        # Determine if MPI should be used
        mpi_procs = int(os.environ.get('SU2_MPI_PROCESSES', 4))
        
        if mpi_procs > 1:
            # Run with MPI
            cmd = [
                'mpirun', '-np', str(mpi_procs),
                'SU2_CFD', str(result_config)
            ]
        else:
            # Run single process
            cmd = ['SU2_CFD', str(result_config)]
        
        # Execute simulation
        process = subprocess.Popen(
            cmd,
            cwd=str(result_dir),
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        stdout, stderr = process.communicate()
        
        # Save output logs
        with open(result_dir / 'stdout.log', 'w') as f:
            f.write(stdout)
        with open(result_dir / 'stderr.log', 'w') as f:
            f.write(stderr)
        
        # Update status
        with simulation_lock:
            if process.returncode == 0:
                sim_info['status'] = 'completed'
                sim_info['end_time'] = datetime.now().isoformat()
            else:
                sim_info['status'] = 'failed'
                sim_info['error'] = 'Simulation failed with non-zero exit code'
                sim_info['end_time'] = datetime.now().isoformat()
    
    except Exception as e:
        with simulation_lock:
            sim_info['status'] = 'failed'
            sim_info['error'] = str(e)
            sim_info['end_time'] = datetime.now().isoformat()

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=PORT, debug=False)