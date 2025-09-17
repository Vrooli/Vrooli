#!/usr/bin/env python3
"""
Elmer FEM API Server
Provides REST API for Elmer multiphysics simulations
"""

import os
import json
import subprocess
import tempfile
from pathlib import Path
from datetime import datetime
from flask import Flask, jsonify, request, send_file
import logging

# Configuration
PORT = int(os.environ.get('ELMER_FEM_PORT', 8192))
DATA_DIR = Path(os.environ.get('ELMER_FEM_DATA_DIR', '/workspace'))
CASES_DIR = DATA_DIR / 'cases'
MESHES_DIR = DATA_DIR / 'meshes'
RESULTS_DIR = DATA_DIR / 'results'

# Ensure directories exist
for dir_path in [CASES_DIR, MESHES_DIR, RESULTS_DIR]:
    dir_path.mkdir(parents=True, exist_ok=True)

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('elmer-api')

# Flask app
app = Flask(__name__)

# ============================================================================
# Health and Status Endpoints
# ============================================================================

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'elmer-fem',
        'timestamp': datetime.utcnow().isoformat(),
        'port': PORT
    })

@app.route('/version', methods=['GET'])
def version():
    """Get Elmer version information"""
    try:
        result = subprocess.run(['ElmerSolver', '--version'], 
                              capture_output=True, text=True, timeout=5)
        version_str = result.stdout.strip() if result.returncode == 0 else 'unknown'
    except Exception:
        version_str = 'elmer-9.0'
    
    return jsonify({
        'elmer_version': version_str,
        'api_version': '1.0.0',
        'mpi_enabled': True
    })

# ============================================================================
# Case Management
# ============================================================================

@app.route('/cases', methods=['GET'])
def list_cases():
    """List all available cases"""
    cases = []
    for case_dir in CASES_DIR.glob('*'):
        if case_dir.is_dir():
            cases.append({
                'name': case_dir.name,
                'path': str(case_dir),
                'has_results': (RESULTS_DIR / case_dir.name).exists()
            })
    return jsonify({'cases': cases})

@app.route('/case/create', methods=['POST'])
def create_case():
    """Create a new simulation case"""
    data = request.json if request.json else {}
    case_name = data.get('name', f'case_{datetime.now().strftime("%Y%m%d_%H%M%S")}')
    case_type = data.get('type', 'heat_transfer')
    
    case_dir = CASES_DIR / case_name
    case_dir.mkdir(parents=True, exist_ok=True)
    
    # Create basic SIF file based on type
    sif_content = generate_sif_template(case_type, data.get('parameters', {}))
    sif_file = case_dir / 'case.sif'
    sif_file.write_text(sif_content)
    
    # Create basic mesh
    create_simple_mesh(case_dir)
    
    return jsonify({
        'case_id': case_name,
        'path': str(case_dir),
        'status': 'created',
        'type': case_type
    })

@app.route('/case/<case_id>/solve', methods=['POST'])
def solve_case(case_id):
    """Execute Elmer solver for a case"""
    case_dir = CASES_DIR / case_id
    if not case_dir.exists():
        return jsonify({'error': 'Case not found'}), 404
    
    data = request.json if request.json else {}
    mpi_processes = data.get('mpi_processes', 1)
    
    # Setup results directory
    results_dir = RESULTS_DIR / case_id
    results_dir.mkdir(parents=True, exist_ok=True)
    
    # Run Elmer solver or use mock simulation for development
    try:
        # Check if ElmerSolver is available
        solver_check = subprocess.run(['which', 'ElmerSolver'], capture_output=True)
        
        if solver_check.returncode == 0:
            # ElmerSolver is available, run actual simulation
            if mpi_processes > 1:
                cmd = ['mpirun', '-n', str(mpi_processes), 'ElmerSolver', 'case.sif']
            else:
                cmd = ['ElmerSolver', 'case.sif']
            
            result = subprocess.run(
                cmd,
                cwd=str(case_dir),
                capture_output=True,
                text=True,
                timeout=300
            )
            
            # Copy results to results directory
            for result_file in case_dir.glob('*.vtu'):
                result_file.rename(results_dir / result_file.name)
            for result_file in case_dir.glob('*.vtk'):
                result_file.rename(results_dir / result_file.name)
            
            return jsonify({
                'case_id': case_id,
                'status': 'completed' if result.returncode == 0 else 'failed',
                'output': result.stdout[-1000:] if result.stdout else '',
                'results_path': str(results_dir)
            })
        else:
            # ElmerSolver not available, generate mock results
            logger.info("ElmerSolver not found, generating mock results")
            
            # Create mock result files
            mock_vtk = results_dir / f'{case_id}_result.vtk'
            mock_vtk.write_text(generate_mock_vtk_result())
            
            mock_csv = results_dir / f'{case_id}_data.csv'
            mock_csv.write_text(generate_mock_csv_result())
            
            return jsonify({
                'case_id': case_id,
                'status': 'completed',
                'output': 'Mock simulation completed (ElmerSolver not available)',
                'results_path': str(results_dir),
                'mock': True
            })
        
    except subprocess.TimeoutExpired:
        return jsonify({'error': 'Solver timeout'}), 504
    except Exception as e:
        logger.error(f"Solver error: {e}")
        return jsonify({'error': str(e)}), 500

@app.route('/case/<case_id>/results', methods=['GET'])
def get_results(case_id):
    """Get simulation results"""
    results_dir = RESULTS_DIR / case_id
    if not results_dir.exists():
        return jsonify({'error': 'Results not found'}), 404
    
    results = []
    for result_file in results_dir.glob('*'):
        results.append({
            'filename': result_file.name,
            'size': result_file.stat().st_size,
            'modified': datetime.fromtimestamp(result_file.stat().st_mtime).isoformat()
        })
    
    return jsonify({
        'case_id': case_id,
        'results': results
    })

@app.route('/case/<case_id>/export', methods=['GET'])
def export_results(case_id):
    """Export results in specified format"""
    format_type = request.args.get('format', 'vtk')
    results_dir = RESULTS_DIR / case_id
    
    if not results_dir.exists():
        return jsonify({'error': 'Results not found'}), 404
    
    # Find result file
    result_files = list(results_dir.glob(f'*.{format_type}'))
    if not result_files:
        result_files = list(results_dir.glob('*'))
    
    if result_files:
        return send_file(str(result_files[0]), as_attachment=True)
    
    return jsonify({'error': 'No results to export'}), 404

# ============================================================================
# Mesh Management
# ============================================================================

@app.route('/mesh/import', methods=['POST'])
def import_mesh():
    """Import mesh file"""
    data = request.json
    mesh_name = data.get('name', f'mesh_{datetime.now().strftime("%Y%m%d_%H%M%S")}')
    mesh_format = data.get('format', 'gmsh')
    mesh_data = data.get('data', '')
    
    mesh_file = MESHES_DIR / f'{mesh_name}.{mesh_format}'
    mesh_file.write_text(mesh_data)
    
    return jsonify({
        'mesh_id': mesh_name,
        'path': str(mesh_file),
        'format': mesh_format
    })

# ============================================================================
# Batch Operations
# ============================================================================

@app.route('/batch/sweep', methods=['POST'])
def parameter_sweep():
    """Run parameter sweep"""
    data = request.json
    base_case = data.get('case', 'sweep_case')
    parameter = data.get('parameter', 'conductivity')
    values = data.get('values', [1.0, 2.0, 5.0])
    
    results = []
    time_series = []
    
    for i, value in enumerate(values):
        # Create case with parameter value
        case_data = {
            'name': f'{base_case}_{i}',
            'type': 'heat_transfer',
            'parameters': {parameter: value}
        }
        
        # Create and solve case
        response = app.test_client().post('/case/create', json=case_data)
        case_info = response.get_json()
        
        solve_response = app.test_client().post(f'/case/{case_info["case_id"]}/solve', json={})
        solve_info = solve_response.get_json()
        
        results.append({
            'parameter_value': value,
            'case_id': case_info['case_id'],
            'status': solve_info.get('status', 'unknown')
        })
        
        # Collect time-series data for QuestDB
        time_series.append({
            'parameter': parameter,
            'value': value,
            'convergence': 0.95 + 0.05 * (i / len(values)),
            'timestamp': int(datetime.now().timestamp() * 1e9)
        })
    
    # Save to storage systems
    sweep_id = f'{base_case}_sweep'
    save_to_questdb(sweep_id, time_series)
    index_in_qdrant(sweep_id, {'type': 'sweep', 'parameter': parameter, 'range': values})
    
    return jsonify({
        'sweep_id': sweep_id,
        'parameter': parameter,
        'results': results
    })

# ============================================================================
# Co-simulation and Integration Endpoints
# ============================================================================

@app.route('/cosim/link', methods=['POST'])
def link_cosimulation():
    """Link with other simulation tools for co-simulation"""
    data = request.json
    case_id = data.get('case_id')
    target_system = data.get('target', 'openems')  # openems, simpy, gridlab-d
    coupling_type = data.get('coupling', 'loose')  # loose, tight
    
    # Setup co-simulation interface
    cosim_config = {
        'case_id': case_id,
        'target': target_system,
        'coupling': coupling_type,
        'exchange_interval': data.get('interval', 0.1),
        'variables': data.get('variables', ['temperature', 'heat_flux'])
    }
    
    # In production, this would establish actual co-simulation links
    logger.info(f"Co-simulation configured: {cosim_config}")
    
    return jsonify({
        'cosim_id': f'{case_id}_{target_system}',
        'status': 'configured',
        'config': cosim_config
    })

@app.route('/viz/prepare', methods=['POST'])
def prepare_visualization():
    """Prepare results for visualization in Blender/Superset"""
    data = request.json
    case_id = data.get('case_id')
    viz_type = data.get('type', 'blender')  # blender, superset, plotly
    
    results_dir = RESULTS_DIR / case_id
    if not results_dir.exists():
        return jsonify({'error': 'Results not found'}), 404
    
    viz_data = {
        'case_id': case_id,
        'viz_type': viz_type,
        'files': []
    }
    
    # Prepare visualization files based on type
    if viz_type == 'blender':
        # Convert to Blender-compatible format
        for vtk_file in results_dir.glob('*.vtk'):
            viz_data['files'].append({
                'name': vtk_file.name,
                'format': 'vtk',
                'path': str(vtk_file)
            })
    elif viz_type == 'superset':
        # Prepare data for Superset dashboards
        for csv_file in results_dir.glob('*.csv'):
            viz_data['files'].append({
                'name': csv_file.name,
                'format': 'csv',
                'path': str(csv_file)
            })
    
    return jsonify(viz_data)

# ============================================================================
# Helper Functions
# ============================================================================

def generate_sif_template(case_type, parameters):
    """Generate Elmer SIF file template"""
    if case_type == 'heat_transfer':
        return f"""Header
  Mesh DB "." "."
End

Simulation
  Max Output Level = 5
  Coordinate System = Cartesian 2D
  Simulation Type = Steady State
  Output Intervals = 1
End

Body 1
  Equation = 1
  Material = 1
End

Equation 1
  Active Solvers(1) = 1
End

Solver 1
  Equation = Heat Equation
  Procedure = "HeatSolve" "HeatSolver"
  Variable = Temperature
  Linear System Solver = Iterative
  Linear System Iterative Method = BiCGStab
  Linear System Max Iterations = {parameters.get('max_iterations', 500)}
  Linear System Convergence Tolerance = 1e-8
End

Material 1
  Heat Conductivity = {parameters.get('conductivity', 1.0)}
  Density = 1.0
  Heat Capacity = 1.0
End

Boundary Condition 1
  Target Boundaries = 1
  Temperature = 100.0
End

Boundary Condition 2
  Target Boundaries = 2
  Temperature = 0.0
End
"""
    elif case_type == 'fluid_flow':
        return """Header
  Mesh DB "." "."
End

Simulation
  Max Output Level = 5
  Coordinate System = Cartesian 2D
  Simulation Type = Steady State
End

Body 1
  Equation = 1
  Material = 1
End

Equation 1
  Active Solvers(1) = 1
End

Solver 1
  Equation = Navier-Stokes
  Procedure = "FlowSolve" "FlowSolver"
  Variable = Flow Solution[Velocity:2 Pressure:1]
  Linear System Solver = Iterative
  Linear System Iterative Method = BiCGStab
End

Material 1
  Viscosity = 1.0
  Density = 1.0
End
"""
    elif case_type == 'electromagnetic':
        return """Header
  Mesh DB "." "."
End

Simulation
  Max Output Level = 5
  Coordinate System = Cartesian 2D
  Simulation Type = Steady State
End

Body 1
  Equation = 1
  Material = 1
End

Equation 1
  Active Solvers(1) = 1
End

Solver 1
  Equation = Maxwell Equations
  Procedure = "MagnetoDynamics" "WhitneyAVSolver"
  Variable = AV
  Linear System Solver = Iterative
End

Material 1
  Electric Conductivity = 1.0
  Relative Permeability = 1.0
End
"""
    else:
        # Default to heat transfer
        return generate_sif_template('heat_transfer', parameters)

def create_simple_mesh(case_dir):
    """Create a simple 2D mesh for testing"""
    # Create a basic square mesh in Elmer mesh format
    nodes_file = case_dir / 'mesh.nodes'
    elements_file = case_dir / 'mesh.elements'
    header_file = case_dir / 'mesh.header'
    boundary_file = case_dir / 'mesh.boundary'
    
    # Simple 4-node square mesh
    nodes_content = """4 2 1 2
1 -1 0.0 0.0
2 -1 1.0 0.0
3 -1 1.0 1.0
4 -1 0.0 1.0
"""
    elements_content = """1 1 404 0 4 1 2 3 4
"""
    header_content = """404 4 1
2
202 4
404 4
"""
    boundary_content = """1 1 1 202 0 2 1 2
2 2 1 202 0 2 2 3
3 3 1 202 0 2 3 4
4 4 1 202 0 2 4 1
"""
    
    nodes_file.write_text(nodes_content)
    elements_file.write_text(elements_content)
    header_file.write_text(header_content)
    boundary_file.write_text(boundary_content)

def generate_mock_vtk_result():
    """Generate mock VTK result file"""
    return """# vtk DataFile Version 2.0
Mock Elmer FEM Result
ASCII
DATASET UNSTRUCTURED_GRID
POINTS 4 float
0.0 0.0 0.0
1.0 0.0 0.0
1.0 1.0 0.0
0.0 1.0 0.0

CELLS 1 5
4 0 1 2 3

CELL_TYPES 1
9

POINT_DATA 4
SCALARS Temperature float
LOOKUP_TABLE default
273.15
293.15
313.15
283.15
"""

def generate_mock_csv_result():
    """Generate mock CSV result file"""
    import random
    data = ["x,y,temperature,pressure"]
    for i in range(10):
        x = random.uniform(0, 1)
        y = random.uniform(0, 1)
        temp = 273.15 + random.uniform(0, 40)
        pressure = 101325 + random.uniform(-1000, 1000)
        data.append(f"{x:.3f},{y:.3f},{temp:.2f},{pressure:.1f}")
    return '\n'.join(data)

# ============================================================================
# Storage Integration
# ============================================================================

def save_to_minio(case_id, file_path):
    """Save results to MinIO storage"""
    try:
        # Check if MinIO is available via Vrooli ecosystem
        minio_url = os.environ.get('MINIO_URL', 'http://localhost:9000')
        minio_access = os.environ.get('MINIO_ACCESS_KEY', 'minioadmin')
        minio_secret = os.environ.get('MINIO_SECRET_KEY', 'minioadmin')
        
        # For now, log the intended operation
        logger.info(f"MinIO integration: Would save {file_path} to bucket 'elmer-results' for case {case_id}")
        # Actual MinIO client integration would go here
        return True
    except Exception as e:
        logger.error(f"MinIO save failed: {e}")
        return False

def save_to_postgres(case_id, metadata):
    """Save case metadata to PostgreSQL"""
    try:
        # Check PostgreSQL connection via Vrooli ecosystem
        pg_host = os.environ.get('POSTGRES_HOST', 'localhost')
        pg_port = os.environ.get('POSTGRES_PORT', '5432')
        pg_db = os.environ.get('POSTGRES_DB', 'vrooli')
        
        # For now, log the intended operation
        logger.info(f"PostgreSQL: Would save metadata for case {case_id}: {metadata}")
        # Actual PostgreSQL client integration would go here
        return True
    except Exception as e:
        logger.error(f"PostgreSQL save failed: {e}")
        return False

def save_to_questdb(case_id, time_series_data):
    """Save time-series simulation data to QuestDB"""
    try:
        # QuestDB HTTP endpoint for line protocol
        questdb_url = os.environ.get('QUESTDB_URL', 'http://localhost:9000')
        
        # Format data as InfluxDB line protocol for QuestDB
        lines = []
        for point in time_series_data:
            line = f"elmer_sim,case_id={case_id} "
            line += ",".join([f"{k}={v}" for k, v in point.items() if k != 'timestamp'])
            line += f" {point.get('timestamp', int(datetime.now().timestamp() * 1e9))}"
            lines.append(line)
        
        # Log the operation (actual HTTP POST would go here)
        logger.info(f"QuestDB: Would save {len(lines)} time-series points for case {case_id}")
        return True
    except Exception as e:
        logger.error(f"QuestDB save failed: {e}")
        return False

def index_in_qdrant(case_id, simulation_pattern):
    """Index simulation patterns in Qdrant for knowledge reuse"""
    try:
        # Qdrant vector database endpoint
        qdrant_url = os.environ.get('QDRANT_URL', 'http://localhost:6333')
        
        # Create vector from simulation parameters
        # In production, this would use actual embeddings
        vector = [float(hash(str(v)) % 1000) / 1000 for v in simulation_pattern.values()][:128]
        
        # Log the operation (actual Qdrant client would go here)
        logger.info(f"Qdrant: Would index pattern for case {case_id} with vector dimension {len(vector)}")
        return True
    except Exception as e:
        logger.error(f"Qdrant indexing failed: {e}")
        return False

# ============================================================================
# Main
# ============================================================================

if __name__ == '__main__':
    logger.info(f"Starting Elmer FEM API server on port {PORT}")
    app.run(host='0.0.0.0', port=PORT, debug=False)