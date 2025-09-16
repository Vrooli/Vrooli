#!/usr/bin/env python3
"""
FreeCAD API Server
Provides REST API for FreeCAD CAD operations
"""

import os
import sys
import json
import traceback
import subprocess
from flask import Flask, request, jsonify, send_file
from flask_cors import CORS
import tempfile
import uuid
from pathlib import Path

# Add FreeCAD to path
freecad_path = os.environ.get('PYTHONPATH', '/usr/lib/freecad/lib')
if freecad_path not in sys.path:
    sys.path.append(freecad_path)

try:
    import FreeCAD
    import Part
    import Mesh
    import Draft
    FREECAD_AVAILABLE = True
except ImportError:
    FREECAD_AVAILABLE = False
    print("Warning: FreeCAD not available, running in limited mode")

app = Flask(__name__)
CORS(app)

# Configuration
PORT = int(os.environ.get('PORT', 8080))
DATA_DIR = Path('/data')
PROJECTS_DIR = Path('/projects')
SCRIPTS_DIR = Path('/scripts')
EXPORTS_DIR = Path('/exports')

# Ensure directories exist
for directory in [DATA_DIR, PROJECTS_DIR, SCRIPTS_DIR, EXPORTS_DIR]:
    directory.mkdir(parents=True, exist_ok=True)

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'freecad',
        'freecad_available': FREECAD_AVAILABLE,
        'version': FreeCAD.Version() if FREECAD_AVAILABLE else None
    })

@app.route('/generate', methods=['POST'])
def generate():
    """Generate CAD model from Python script"""
    if not FREECAD_AVAILABLE:
        return jsonify({'error': 'FreeCAD not available'}), 503
    
    try:
        data = request.get_json()
        script_content = data.get('script', '')
        output_format = data.get('format', 'FCStd')
        
        if not script_content:
            return jsonify({'error': 'No script provided'}), 400
        
        # Create temporary script file
        script_id = str(uuid.uuid4())
        script_path = SCRIPTS_DIR / f'{script_id}.py'
        output_path = EXPORTS_DIR / f'{script_id}.{output_format.lower()}'
        
        # Write script
        script_path.write_text(script_content)
        
        # Execute script in FreeCAD context
        doc = FreeCAD.newDocument()
        
        # Create execution context with useful globals
        exec_globals = {
            'FreeCAD': FreeCAD,
            'Part': Part,
            'doc': doc,
            'Draft': Draft,
            '__file__': str(script_path),
            '__name__': '__main__'
        }
        
        # Execute the script
        exec(script_content, exec_globals)
        
        # Save the document
        if output_format.upper() == 'FCSTD':
            doc.saveAs(str(output_path))
        else:
            # Export to other formats
            if doc.Objects:
                if output_format.upper() in ['STEP', 'STP']:
                    Part.export(doc.Objects, str(output_path.with_suffix('.step')))
                elif output_format.upper() in ['IGES', 'IGS']:
                    Part.export(doc.Objects, str(output_path.with_suffix('.iges')))
                elif output_format.upper() == 'STL':
                    Mesh.export(doc.Objects, str(output_path.with_suffix('.stl')))
                elif output_format.upper() == 'OBJ':
                    Mesh.export(doc.Objects, str(output_path.with_suffix('.obj')))
                else:
                    return jsonify({'error': f'Unsupported format: {output_format}'}), 400
        
        FreeCAD.closeDocument(doc.Name)
        
        return jsonify({
            'success': True,
            'output': str(output_path),
            'id': script_id
        })
        
    except Exception as e:
        return jsonify({
            'error': str(e),
            'traceback': traceback.format_exc()
        }), 500

@app.route('/export', methods=['POST'])
def export():
    """Export existing model to different format"""
    if not FREECAD_AVAILABLE:
        return jsonify({'error': 'FreeCAD not available'}), 503
    
    try:
        data = request.get_json()
        input_file = data.get('input')
        output_format = data.get('format', 'STEP')
        
        if not input_file:
            return jsonify({'error': 'No input file specified'}), 400
        
        input_path = Path(input_file)
        if not input_path.exists():
            # Check in projects directory
            input_path = PROJECTS_DIR / input_file
            if not input_path.exists():
                return jsonify({'error': 'Input file not found'}), 404
        
        # Open document
        doc = FreeCAD.open(str(input_path))
        
        # Generate output path
        output_id = str(uuid.uuid4())
        output_path = EXPORTS_DIR / f'{output_id}.{output_format.lower()}'
        
        # Export based on format
        if doc.Objects:
            if output_format.upper() in ['STEP', 'STP']:
                Part.export(doc.Objects, str(output_path.with_suffix('.step')))
            elif output_format.upper() in ['IGES', 'IGS']:
                Part.export(doc.Objects, str(output_path.with_suffix('.iges')))
            elif output_format.upper() == 'STL':
                Mesh.export(doc.Objects, str(output_path.with_suffix('.stl')))
            elif output_format.upper() == 'OBJ':
                Mesh.export(doc.Objects, str(output_path.with_suffix('.obj')))
            elif output_format.upper() == 'DXF':
                import importDXF
                importDXF.export(doc.Objects, str(output_path.with_suffix('.dxf')))
            else:
                FreeCAD.closeDocument(doc.Name)
                return jsonify({'error': f'Unsupported format: {output_format}'}), 400
        
        FreeCAD.closeDocument(doc.Name)
        
        return jsonify({
            'success': True,
            'output': str(output_path),
            'id': output_id
        })
        
    except Exception as e:
        return jsonify({
            'error': str(e),
            'traceback': traceback.format_exc()
        }), 500

@app.route('/analyze', methods=['POST'])
def analyze():
    """Perform analysis on CAD model (placeholder for FEM)"""
    if not FREECAD_AVAILABLE:
        return jsonify({'error': 'FreeCAD not available'}), 503
    
    try:
        data = request.get_json()
        input_file = data.get('input')
        analysis_type = data.get('type', 'properties')
        
        if not input_file:
            return jsonify({'error': 'No input file specified'}), 400
        
        input_path = Path(input_file)
        if not input_path.exists():
            input_path = PROJECTS_DIR / input_file
            if not input_path.exists():
                return jsonify({'error': 'Input file not found'}), 404
        
        # Open document
        doc = FreeCAD.open(str(input_path))
        
        results = {'analysis_type': analysis_type}
        
        if analysis_type == 'properties':
            # Basic geometric properties
            for obj in doc.Objects:
                if hasattr(obj, 'Shape'):
                    shape = obj.Shape
                    results[obj.Name] = {
                        'volume': shape.Volume if hasattr(shape, 'Volume') else None,
                        'area': shape.Area if hasattr(shape, 'Area') else None,
                        'center_of_mass': list(shape.CenterOfMass) if hasattr(shape, 'CenterOfMass') else None,
                        'bounding_box': {
                            'min': [shape.BoundBox.XMin, shape.BoundBox.YMin, shape.BoundBox.ZMin],
                            'max': [shape.BoundBox.XMax, shape.BoundBox.YMax, shape.BoundBox.ZMax]
                        } if hasattr(shape, 'BoundBox') else None
                    }
        elif analysis_type == 'fem':
            # Placeholder for FEM analysis
            results['message'] = 'FEM analysis not yet implemented'
            results['status'] = 'pending'
        
        FreeCAD.closeDocument(doc.Name)
        
        return jsonify(results)
        
    except Exception as e:
        return jsonify({
            'error': str(e),
            'traceback': traceback.format_exc()
        }), 500

@app.route('/parametric/update', methods=['POST'])
def parametric_update():
    """Update parametric model parameters"""
    if not FREECAD_AVAILABLE:
        return jsonify({'error': 'FreeCAD not available'}), 503
    
    try:
        data = request.get_json()
        input_file = data.get('input')
        parameters = data.get('parameters', {})
        
        if not input_file:
            return jsonify({'error': 'No input file specified'}), 400
        
        input_path = Path(input_file)
        if not input_path.exists():
            input_path = PROJECTS_DIR / input_file
            if not input_path.exists():
                return jsonify({'error': 'Input file not found'}), 404
        
        # Open document
        doc = FreeCAD.open(str(input_path))
        
        # Update parameters
        for obj_name, params in parameters.items():
            if hasattr(doc, obj_name):
                obj = getattr(doc, obj_name)
                for param, value in params.items():
                    if hasattr(obj, param):
                        setattr(obj, param, value)
        
        # Recompute
        doc.recompute()
        
        # Save updated document
        output_id = str(uuid.uuid4())
        output_path = PROJECTS_DIR / f'{output_id}.FCStd'
        doc.saveAs(str(output_path))
        
        FreeCAD.closeDocument(doc.Name)
        
        return jsonify({
            'success': True,
            'output': str(output_path),
            'id': output_id
        })
        
    except Exception as e:
        return jsonify({
            'error': str(e),
            'traceback': traceback.format_exc()
        }), 500

@app.route('/download/<file_id>', methods=['GET'])
def download(file_id):
    """Download generated file"""
    # Try to find file in exports directory
    for file in EXPORTS_DIR.glob(f'{file_id}.*'):
        if file.exists():
            return send_file(str(file), as_attachment=True)
    
    # Try projects directory
    for file in PROJECTS_DIR.glob(f'{file_id}.*'):
        if file.exists():
            return send_file(str(file), as_attachment=True)
    
    return jsonify({'error': 'File not found'}), 404

@app.route('/list/scripts', methods=['GET'])
def list_scripts():
    """List available scripts"""
    scripts = [f.name for f in SCRIPTS_DIR.glob('*.py')]
    return jsonify({'scripts': scripts})

@app.route('/list/projects', methods=['GET'])
def list_projects():
    """List available projects"""
    projects = [f.name for f in PROJECTS_DIR.glob('*.FCStd')]
    return jsonify({'projects': projects})

@app.route('/list/exports', methods=['GET'])
def list_exports():
    """List exported files"""
    exports = [f.name for f in EXPORTS_DIR.glob('*')]
    return jsonify({'exports': exports})

if __name__ == '__main__':
    print(f"Starting FreeCAD API server on port {PORT}")
    print(f"FreeCAD available: {FREECAD_AVAILABLE}")
    if FREECAD_AVAILABLE:
        print(f"FreeCAD version: {FreeCAD.Version()}")
    app.run(host='0.0.0.0', port=PORT, debug=False)