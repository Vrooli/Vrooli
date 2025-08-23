#!/usr/bin/env python3
"""
Validate and update workflow metadata YAML file.
Helps prevent drift between actual workflow files and metadata.
"""

import os
import yaml
import json
from pathlib import Path
import sys
import requests
import time

def validate_workflow_file(file_path, platform):
    """Validate that workflow file is valid for its platform."""
    try:
        with open(file_path, 'r') as f:
            if platform in ['n8n', 'node-red', 'windmill', 'huginn', 'comfyui', 'integration']:
                # All platforms use JSON format
                data = json.load(f)
                
                # Platform-specific validation
                if platform == 'n8n':
                    return validate_n8n_workflow(data)
                elif platform == 'node-red':
                    return validate_node_red_workflow(data)
                elif platform == 'windmill':
                    return validate_windmill_workflow(data)
                elif platform == 'huginn':
                    return validate_huginn_workflow(data)
                elif platform == 'comfyui':
                    return validate_comfyui_workflow(data)
                elif platform == 'integration':
                    return validate_integration_workflow(data)
                
                return True, "Valid JSON format"
            else:
                return False, f"Unknown platform: {platform}"
                
    except json.JSONDecodeError as e:
        return False, f"Invalid JSON: {e}"
    except Exception as e:
        return False, f"Error reading file: {e}"

def validate_n8n_workflow(data):
    """Validate N8N workflow structure."""
    required_fields = ['name', 'nodes']
    
    for field in required_fields:
        if field not in data:
            return False, f"Missing required field: {field}"
    
    if not isinstance(data['nodes'], list):
        return False, "nodes must be a list"
    
    if len(data['nodes']) == 0:
        return False, "workflow must have at least one node"
    
    # Check for required node fields
    for i, node in enumerate(data['nodes']):
        if 'name' not in node:
            return False, f"Node {i} missing name field"
        if 'type' not in node:
            return False, f"Node {i} missing type field"
    
    return True, "Valid N8N workflow"

def validate_node_red_workflow(data):
    """Validate Node-RED flow structure."""
    if not isinstance(data, list):
        return False, "Node-RED flow must be a list of nodes"
    
    if len(data) == 0:
        return False, "flow must have at least one node"
    
    # Check for required node fields
    for i, node in enumerate(data):
        if 'id' not in node:
            return False, f"Node {i} missing id field"
        if 'type' not in node:
            return False, f"Node {i} missing type field"
    
    return True, "Valid Node-RED flow"

def validate_windmill_workflow(data):
    """Validate Windmill workflow structure."""
    # Basic validation - Windmill workflows can be quite flexible
    if not isinstance(data, dict):
        return False, "Windmill workflow must be a JSON object"
    
    return True, "Valid Windmill workflow"

def validate_huginn_workflow(data):
    """Validate Huginn agent structure."""
    if not isinstance(data, dict):
        return False, "Huginn scenario must be a JSON object"
    
    return True, "Valid Huginn scenario"

def validate_comfyui_workflow(data):
    """Validate ComfyUI workflow structure."""
    if not isinstance(data, dict):
        return False, "ComfyUI workflow must be a JSON object"
    
    return True, "Valid ComfyUI workflow"

def validate_integration_workflow(data):
    """Validate integration workflow structure."""
    if not isinstance(data, dict):
        return False, "Integration workflow must be a JSON object"
    
    # Integration workflows are cross-platform specifications
    if 'name' not in data:
        return False, "Integration workflow missing name field"
    
    return True, "Valid integration workflow"

def test_platform_availability(platform_config):
    """Test if a platform is available and responding."""
    port = platform_config.get('port')
    health_check = platform_config.get('health_check')
    
    # Special case for integration platform (cross-platform workflows)
    if port is None or port == 'null':
        return True, "Integration platform (no port required)"
    
    if not port:
        return False, "No port configured"
    
    # Test basic connectivity
    try:
        import socket
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(5)
        result = sock.connect_ex(('localhost', port))
        sock.close()
        
        if result != 0:
            return False, f"Port {port} not accessible"
            
    except Exception as e:
        return False, f"Connection test failed: {e}"
    
    # Test health endpoint if available
    if health_check and health_check != 'null':
        try:
            response = requests.get(f"http://localhost:{port}{health_check}", timeout=10)
            if response.status_code >= 400:
                return False, f"Health check failed with status {response.status_code}"
        except requests.RequestException as e:
            return False, f"Health check request failed: {e}"
    
    return True, "Platform available"

def validate_metadata(yaml_file='workflows.yaml'):
    """Validate that metadata matches actual workflow files."""
    base_dir = Path(__file__).parent
    
    with open(yaml_file, 'r') as f:
        data = yaml.safe_load(f)
    
    errors = []
    warnings = []
    
    # Get platform configurations
    platforms = data.get('platforms', {})
    
    # Test platform availability
    print("Testing platform availability...")
    for platform_name, platform_config in platforms.items():
        available, message = test_platform_availability(platform_config)
        if available:
            print(f"  ✅ {platform_name}: {message}")
        else:
            print(f"  ❌ {platform_name}: {message}")
            warnings.append(f"Platform {platform_name} not available: {message}")
    
    # Flatten all workflow entries
    all_workflows = []
    workflows_data = data.get('workflows', {})
    
    for platform_workflows in workflows_data.values():
        if isinstance(platform_workflows, list):
            all_workflows.extend(platform_workflows)
    
    # Check each workflow file
    for workflow_meta in all_workflows:
        path = workflow_meta.get('path')
        platform = workflow_meta.get('platform')
        
        if not path:
            errors.append("Workflow entry missing 'path' field")
            continue
            
        if not platform:
            errors.append(f"{path}: missing 'platform' field")
            continue
        
        full_path = base_dir / path
        
        # Check if file exists
        if not full_path.exists():
            errors.append(f"Workflow file not found: {path}")
            continue
        
        # Validate workflow file structure
        is_valid, message = validate_workflow_file(full_path, platform)
        if not is_valid:
            errors.append(f"{path}: {message}")
        
        # Check metadata consistency
        expected_duration = workflow_meta.get('expectedDuration', 0)
        if expected_duration <= 0:
            warnings.append(f"{path}: expectedDuration should be positive")
        
        complexity = workflow_meta.get('complexity')
        if complexity not in ['basic', 'intermediate', 'advanced']:
            warnings.append(f"{path}: complexity should be basic/intermediate/advanced")
        
        tags = workflow_meta.get('tags', [])
        if not tags:
            warnings.append(f"{path}: no tags specified")
        
        integrations = workflow_meta.get('integration', [])
        if not integrations:
            warnings.append(f"{path}: no integrations specified")
    
    # Check for orphaned workflow files
    yaml_paths = {w.get('path') for w in all_workflows if w.get('path')}
    
    for workflow_file in base_dir.rglob('*.json'):
        if workflow_file.name in ['workflows.yaml', 'validate_metadata.py', 'test_helpers.sh']:
            continue
            
        rel_path = workflow_file.relative_to(base_dir)
        if str(rel_path) not in yaml_paths:
            warnings.append(f"Workflow file without metadata: {rel_path}")
    
    # Validate test suite references
    test_suites = data.get('testSuites', {})
    for suite_name, suite_workflows in test_suites.items():
        for workflow_path in suite_workflows:
            if workflow_path not in yaml_paths:
                errors.append(f"Test suite '{suite_name}' references unknown workflow: {workflow_path}")
    
    # Validate integration expectations
    integration_expectations = data.get('integrationExpectations', {})
    for integration_name, expectations in integration_expectations.items():
        endpoint = expectations.get('endpoint')
        if endpoint and endpoint.startswith('http://localhost:'):
            # Extract port and test basic connectivity
            try:
                port = int(endpoint.split(':')[2].split('/')[0])
                import socket
                sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                sock.settimeout(2)
                result = sock.connect_ex(('localhost', port))
                sock.close()
                
                if result != 0:
                    warnings.append(f"Integration '{integration_name}' endpoint not reachable: {endpoint}")
            except (ValueError, IndexError):
                warnings.append(f"Integration '{integration_name}' has invalid endpoint format: {endpoint}")
            except Exception:
                pass  # Skip connection test errors
    
    return errors, warnings

def main():
    """Run validation and report results."""
    print("Validating workflow metadata...")
    
    try:
        errors, warnings = validate_metadata()
    except FileNotFoundError:
        print("ERROR: workflows.yaml not found")
        sys.exit(1)
    except yaml.YAMLError as e:
        print(f"ERROR: Invalid YAML format: {e}")
        sys.exit(1)
    
    if errors:
        print(f"\n❌ Found {len(errors)} errors:")
        for error in errors:
            print(f"  - {error}")
    
    if warnings:
        print(f"\n⚠️  Found {len(warnings)} warnings:")
        for warning in warnings:
            print(f"  - {warning}")
    
    if not errors and not warnings:
        print("✅ All workflow metadata is valid!")
    
    # Exit with error code if there are errors
    sys.exit(1 if errors else 0)

if __name__ == '__main__':
    main()