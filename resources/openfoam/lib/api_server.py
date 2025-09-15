#!/usr/bin/env python3
"""OpenFOAM API Server - REST interface for CFD simulations"""

from flask import Flask, jsonify, request
import subprocess
import os
import sys
import json
from pathlib import Path
import threading
import time

app = Flask(__name__)

# Configuration
CASES_DIR = os.environ.get('OPENFOAM_CASES_DIR', '/cases')
RESULTS_DIR = os.environ.get('OPENFOAM_RESULTS_DIR', '/results')
PORT = int(os.environ.get('OPENFOAM_PORT', '8090'))

# Ensure directories exist
Path(CASES_DIR).mkdir(parents=True, exist_ok=True)
Path(RESULTS_DIR).mkdir(parents=True, exist_ok=True)

def run_openfoam_command(command, cwd=None):
    """Run OpenFOAM command with proper environment"""
    bash_cmd = f"source /opt/openfoam11/etc/bashrc && {command}"
    try:
        result = subprocess.run(
            ['bash', '-c', bash_cmd],
            capture_output=True,
            text=True,
            cwd=cwd,
            timeout=60
        )
        return result.returncode == 0, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return False, "", "Command timed out"
    except Exception as e:
        return False, "", str(e)

@app.route('/health')
def health():
    """Health check endpoint"""
    success, version, _ = run_openfoam_command('foamVersion')
    return jsonify({
        'status': 'healthy',
        'service': 'openfoam',
        'version': version.strip() if success else 'v11',
        'timestamp': int(time.time())
    })

@app.route('/api/status')
def status():
    """Get resource status"""
    cases = []
    try:
        cases = [d for d in os.listdir(CASES_DIR) if os.path.isdir(os.path.join(CASES_DIR, d))]
    except:
        pass
    
    return jsonify({
        'status': 'running',
        'cases': cases,
        'cases_count': len(cases),
        'paths': {
            'cases': CASES_DIR,
            'results': RESULTS_DIR
        }
    })

@app.route('/api/case/create', methods=['POST'])
def create_case():
    """Create new simulation case"""
    data = request.json or {}
    case_name = data.get('name', 'cavity')
    case_type = data.get('type', 'fluid/cavity')
    
    case_path = os.path.join(CASES_DIR, case_name)
    if os.path.exists(case_path):
        return jsonify({'error': f'Case {case_name} already exists'}), 400
    
    # Copy tutorial case
    cmd = f"cp -r $FOAM_TUTORIALS/{case_type} {case_path}"
    success, stdout, stderr = run_openfoam_command(cmd)
    
    if not success:
        # Try alternative tutorial
        cmd = f"cp -r $FOAM_TUTORIALS/fluid/pitzDaily {case_path}"
        success, stdout, stderr = run_openfoam_command(cmd)
    
    if success:
        return jsonify({
            'message': f'Case {case_name} created successfully',
            'path': case_path
        })
    else:
        return jsonify({'error': 'Failed to create case', 'details': stderr}), 500

@app.route('/api/mesh/generate', methods=['POST'])
def generate_mesh():
    """Generate mesh for a case"""
    data = request.json or {}
    case_name = data.get('case', 'cavity')
    mesh_type = data.get('type', 'blockMesh')
    
    case_path = os.path.join(CASES_DIR, case_name)
    if not os.path.exists(case_path):
        return jsonify({'error': f'Case {case_name} not found'}), 404
    
    success, stdout, stderr = run_openfoam_command(mesh_type, cwd=case_path)
    
    if success:
        return jsonify({
            'message': f'Mesh generated for {case_name}',
            'type': mesh_type,
            'output': stdout
        })
    else:
        return jsonify({'error': 'Mesh generation failed', 'details': stderr}), 500

@app.route('/api/solver/run', methods=['POST'])
def run_solver():
    """Run solver for a case"""
    data = request.json or {}
    case_name = data.get('case', 'cavity')
    solver = data.get('solver', 'simpleFoam')
    
    case_path = os.path.join(CASES_DIR, case_name)
    if not os.path.exists(case_path):
        return jsonify({'error': f'Case {case_name} not found'}), 404
    
    # Run solver in background for long simulations
    def run_solver_async():
        success, stdout, stderr = run_openfoam_command(solver, cwd=case_path)
        # Save results
        result_file = os.path.join(RESULTS_DIR, f"{case_name}_result.json")
        with open(result_file, 'w') as f:
            json.dump({
                'success': success,
                'solver': solver,
                'output': stdout,
                'errors': stderr
            }, f)
    
    thread = threading.Thread(target=run_solver_async)
    thread.start()
    
    return jsonify({
        'message': f'Solver {solver} started for {case_name}',
        'status': 'running'
    })

@app.route('/api/results/<case_name>')
def get_results(case_name):
    """Get simulation results"""
    result_file = os.path.join(RESULTS_DIR, f"{case_name}_result.json")
    
    if os.path.exists(result_file):
        with open(result_file, 'r') as f:
            result = json.load(f)
        return jsonify(result)
    else:
        # Check if case exists
        case_path = os.path.join(CASES_DIR, case_name)
        if not os.path.exists(case_path):
            return jsonify({'error': f'Case {case_name} not found'}), 404
        
        # Check for VTK files
        vtk_path = os.path.join(case_path, 'VTK')
        has_vtk = os.path.exists(vtk_path)
        
        return jsonify({
            'case': case_name,
            'status': 'pending',
            'has_vtk': has_vtk,
            'path': case_path
        })

@app.route('/api/export/<format>', methods=['POST'])
def export_results(format):
    """Export results in specified format"""
    data = request.json or {}
    case_name = data.get('case', 'cavity')
    
    case_path = os.path.join(CASES_DIR, case_name)
    if not os.path.exists(case_path):
        return jsonify({'error': f'Case {case_name} not found'}), 404
    
    if format == 'vtk':
        success, stdout, stderr = run_openfoam_command('foamToVTK', cwd=case_path)
        if success:
            return jsonify({
                'message': f'Results exported to VTK format',
                'path': os.path.join(case_path, 'VTK')
            })
        else:
            return jsonify({'error': 'VTK export failed', 'details': stderr}), 500
    else:
        return jsonify({'error': f'Unsupported format: {format}'}), 400

@app.route('/api/paraview/prepare', methods=['POST'])
def prepare_paraview():
    """Prepare case for ParaView visualization"""
    data = request.json or {}
    case_name = data.get('case', 'cavity')
    
    case_path = os.path.join(CASES_DIR, case_name)
    if not os.path.exists(case_path):
        return jsonify({'error': f'Case {case_name} not found'}), 404
    
    # Create ParaView state file
    pvsm_content = f"""
# ParaView State File for {case_name}
from paraview.simple import *
import os

case_path = '{case_path}'
vtk_path = os.path.join(case_path, 'VTK')

# Load OpenFOAM case or VTK files
if os.path.exists(os.path.join(case_path, 'system', 'controlDict')):
    reader = OpenFOAMReader(FileName=os.path.join(case_path, '{case_name}.OpenFOAM'))
elif os.path.exists(vtk_path):
    reader = LegacyVTKReader(FileNames=[os.path.join(vtk_path, f) for f in os.listdir(vtk_path) if f.endswith('.vtk')])
else:
    print('No suitable data found')
    
# Create render view
renderView = GetActiveViewOrCreate('RenderView')
display = Show(reader, renderView)
display.Representation = 'Surface'

# Color by velocity magnitude if available
ColorBy(display, ('CELLS', 'U'))

# Reset camera
renderView.ResetCamera()

# Save screenshot
SaveScreenshot(os.path.join(case_path, 'preview.png'), renderView, ImageResolution=[1024, 768])
"""
    
    state_file = os.path.join(case_path, f'{case_name}_paraview.py')
    try:
        # Write state file inside container
        success, _, _ = run_openfoam_command(f"cat > {state_file} << 'EOF'\n{pvsm_content}\nEOF")
        
        return jsonify({
            'message': 'ParaView state prepared',
            'state_file': state_file,
            'vtk_path': os.path.join(case_path, 'VTK'),
            'info': 'Use pvpython or ParaView to load the state file'
        })
    except Exception as e:
        return jsonify({'error': 'Failed to prepare ParaView state', 'details': str(e)}), 500

@app.route('/api/case/list')
def list_cases():
    """List all cases"""
    try:
        cases = []
        for case_name in os.listdir(CASES_DIR):
            case_path = os.path.join(CASES_DIR, case_name)
            if os.path.isdir(case_path):
                cases.append({
                    'name': case_name,
                    'path': case_path,
                    'has_mesh': os.path.exists(os.path.join(case_path, 'constant/polyMesh')),
                    'has_results': os.path.exists(os.path.join(case_path, 'VTK'))
                })
        return jsonify({'cases': cases})
    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    # Use port from environment or default
    actual_port = PORT
    # Try to get the actual mapped port if running in Docker
    try:
        import socket
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.bind(('', 0))
        available_port = s.getsockname()[1]
        s.close()
        # If default port is in use, find another
        try:
            test_sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            test_sock.bind(('', PORT))
            test_sock.close()
        except:
            actual_port = available_port
    except:
        pass
    
    print(f"Starting OpenFOAM API server on port {actual_port}")
    app.run(host='0.0.0.0', port=actual_port, debug=False)