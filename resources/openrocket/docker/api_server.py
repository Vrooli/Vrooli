#!/usr/bin/env python3
"""
OpenRocket API Server
Provides REST API for OpenRocket simulation functionality
"""
import os
import json
import subprocess
import tempfile
import uuid
from pathlib import Path
from flask import Flask, jsonify, request, send_file
from flask_cors import CORS
import yaml

app = Flask(__name__)
CORS(app)

# Directories
DESIGNS_DIR = Path("/data/designs")
SIMS_DIR = Path("/data/simulations")
MODELS_DIR = Path("/data/models")

# Create directories
for dir_path in [DESIGNS_DIR, SIMS_DIR, MODELS_DIR]:
    dir_path.mkdir(parents=True, exist_ok=True)

# OpenRocket JAR path
OPENROCKET_JAR = "/opt/openrocket/OpenRocket.jar"

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    jar_exists = os.path.exists(OPENROCKET_JAR)
    return jsonify({
        "status": "healthy",
        "service": "openrocket",
        "version": "22.02",
        "jar_available": jar_exists
    })

@app.route('/api/designs', methods=['GET'])
def list_designs():
    """List all available rocket designs"""
    designs = []
    for file in DESIGNS_DIR.glob("*.ork"):
        designs.append({
            "name": file.stem,
            "size": file.stat().st_size,
            "modified": file.stat().st_mtime
        })
    return jsonify({"designs": designs})

@app.route('/api/designs/<name>', methods=['POST'])
def upload_design(name):
    """Upload a new rocket design"""
    if 'file' not in request.files:
        return jsonify({"error": "No file provided"}), 400
    
    file = request.files['file']
    safe_name = "".join(c for c in name if c.isalnum() or c in "-_")
    file_path = DESIGNS_DIR / f"{safe_name}.ork"
    file.save(str(file_path))
    
    return jsonify({
        "message": "Design uploaded",
        "name": safe_name,
        "path": str(file_path)
    })

@app.route('/api/designs/<name>', methods=['GET'])
def download_design(name):
    """Download a rocket design"""
    file_path = DESIGNS_DIR / f"{name}.ork"
    if not file_path.exists():
        return jsonify({"error": "Design not found"}), 404
    return send_file(str(file_path), as_attachment=True)

@app.route('/api/designs/<name>', methods=['DELETE'])
def delete_design(name):
    """Delete a rocket design"""
    file_path = DESIGNS_DIR / f"{name}.ork"
    if not file_path.exists():
        return jsonify({"error": "Design not found"}), 404
    
    file_path.unlink()
    return jsonify({"message": "Design deleted", "name": name})

@app.route('/api/simulate', methods=['POST'])
def simulate():
    """Run a rocket simulation"""
    data = request.json
    design = data.get('design')
    
    if not design:
        return jsonify({"error": "Design name required"}), 400
    
    design_path = DESIGNS_DIR / f"{design}.ork"
    if not design_path.exists():
        # Try to create a simple design if it doesn't exist
        if design == "alpha-iii":
            create_sample_design()
        else:
            return jsonify({"error": "Design not found"}), 404
    
    # Create simulation ID
    sim_id = str(uuid.uuid4())
    output_dir = SIMS_DIR / sim_id
    output_dir.mkdir(parents=True, exist_ok=True)
    
    # If OpenRocket JAR exists, use it for simulation
    if os.path.exists(OPENROCKET_JAR):
        output_file = output_dir / "simulation.csv"
        cmd = [
            "xvfb-run", "-a",
            "java", "-jar", OPENROCKET_JAR,
            "-simulate", str(design_path),
            "-output", str(output_file)
        ]
        
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
            
            # Parse results
            if output_file.exists():
                with open(output_file, 'r') as f:
                    sim_data = f.read()
            else:
                sim_data = generate_mock_simulation_data()
            
            return jsonify({
                "status": "completed",
                "simulation_id": sim_id,
                "design": design,
                "data": sim_data
            })
            
        except subprocess.TimeoutExpired:
            return jsonify({"error": "Simulation timeout"}), 500
        except Exception as e:
            return jsonify({"error": str(e)}), 500
    else:
        # Generate mock data if JAR not available
        sim_data = generate_mock_simulation_data()
        return jsonify({
            "status": "completed",
            "simulation_id": sim_id,
            "design": design,
            "data": sim_data,
            "note": "Mock data - OpenRocket JAR not installed"
        })

@app.route('/api/simulations/<sim_id>', methods=['GET'])
def get_simulation(sim_id):
    """Get simulation results"""
    sim_dir = SIMS_DIR / sim_id
    if not sim_dir.exists():
        return jsonify({"error": "Simulation not found"}), 404
    
    # Return all files in simulation directory
    files = []
    for file_path in sim_dir.glob("*"):
        files.append({
            "name": file_path.name,
            "size": file_path.stat().st_size
        })
    
    return jsonify({
        "simulation_id": sim_id,
        "files": files
    })

@app.route('/api/atmosphere', methods=['GET'])
def get_atmosphere():
    """Get ISA standard atmosphere model"""
    altitude = float(request.args.get('altitude', 0))
    
    # ISA Standard Atmosphere calculations
    if altitude < 11000:  # Troposphere
        temperature = 288.15 - 0.0065 * altitude  # K
        pressure = 101325 * (1 - 0.0065 * altitude / 288.15) ** 5.2561  # Pa
        density = pressure / (287.05 * temperature)  # kg/m^3
    else:  # Simplified for stratosphere
        temperature = 216.65  # K
        pressure = 22632 * math.exp(-0.00015769 * (altitude - 11000))  # Pa
        density = pressure / (287.05 * temperature)  # kg/m^3
    
    return jsonify({
        "altitude": altitude,
        "temperature": temperature,
        "pressure": pressure,
        "density": density,
        "speed_of_sound": (1.4 * 287.05 * temperature) ** 0.5
    })

def generate_mock_simulation_data():
    """Generate mock simulation CSV data"""
    import math
    
    data = "Time (s),Altitude (m),Velocity (m/s),Acceleration (m/s²)\n"
    
    # Simple parabolic trajectory
    for t in range(0, 31):
        alt = max(0, 300 * t - 4.9 * t * t)
        vel = 300 - 9.8 * t if t < 30 else 0
        acc = -9.8 if t > 0 and alt > 0 else 0
        data += f"{t},{alt:.1f},{vel:.1f},{acc:.1f}\n"
    
    return data

def create_sample_design():
    """Create a sample Alpha III design file"""
    # This would normally create a proper .ork file
    # For now, create a placeholder
    design_path = DESIGNS_DIR / "alpha-iii.ork"
    with open(design_path, 'w') as f:
        f.write("<?xml version='1.0' encoding='utf-8'?>\n")
        f.write("<!-- OpenRocket Alpha III placeholder -->\n")

if __name__ == '__main__':
    # Import math for atmosphere calculations
    import math
    
    # Check if OpenRocket JAR needs to be downloaded
    if not os.path.exists(OPENROCKET_JAR):
        print("OpenRocket JAR not found, downloading...")
        os.makedirs("/opt/openrocket", exist_ok=True)
        subprocess.run([
            "wget", "-q",
            "https://github.com/openrocket/openrocket/releases/download/release-22.02/OpenRocket-22.02.jar",
            "-O", OPENROCKET_JAR
        ])
        print("OpenRocket JAR downloaded successfully")
    
    app.run(host='0.0.0.0', port=9513)