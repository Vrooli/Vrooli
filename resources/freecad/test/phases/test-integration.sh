#!/usr/bin/env bash
################################################################################
# FreeCAD Resource - Integration Tests
# 
# Tests FreeCAD functionality and Python API integration
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and configuration
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/lib/core.sh"

log::info "Starting FreeCAD integration tests..."

# Test 1: Python API availability
log::info "Test 1: Testing Python API availability..."
test_script="/tmp/freecad_api_test_${RANDOM}.py"
cat > "$test_script" <<'EOF'
import sys
try:
    import FreeCAD
    print("FreeCAD imported successfully")
    print(f"Version: {FreeCAD.Version()}")
    sys.exit(0)
except ImportError as e:
    print(f"Failed to import FreeCAD: {e}")
    sys.exit(1)
EOF

if docker exec "${FREECAD_CONTAINER_NAME}" python3 < "$test_script"; then
    log::info "✓ Python API is available"
    rm "$test_script"
else
    log::error "✗ Python API is not available"
    rm "$test_script"
    exit 1
fi

# Test 2: Create parametric model
log::info "Test 2: Testing parametric modeling..."
model_script="/tmp/freecad_model_test_${RANDOM}.py"
cat > "$model_script" <<'EOF'
import FreeCAD
import Part

# Create a new document
doc = FreeCAD.newDocument("TestModel")

# Create a parametric box
box = doc.addObject("Part::Box", "TestBox")
box.Length = 100
box.Width = 50
box.Height = 25

# Recompute
doc.recompute()

# Verify creation
if len(doc.Objects) > 0:
    print("Model created successfully")
    print(f"Objects in document: {len(doc.Objects)}")
else:
    print("Failed to create model")
    exit(1)
EOF

if docker exec "${FREECAD_CONTAINER_NAME}" freecadcmd < "$model_script" 2>/dev/null | grep -q "Model created successfully"; then
    log::info "✓ Parametric modeling works"
    rm "$model_script"
else
    log::error "✗ Parametric modeling failed"
    rm "$model_script"
    exit 1
fi

# Test 3: Boolean operations
log::info "Test 3: Testing boolean operations..."
bool_script="/tmp/freecad_bool_test_${RANDOM}.py"
cat > "$bool_script" <<'EOF'
import FreeCAD
import Part

doc = FreeCAD.newDocument("BooleanTest")

# Create two shapes
box = doc.addObject("Part::Box", "Box")
cylinder = doc.addObject("Part::Cylinder", "Cylinder")

# Set properties
box.Length = 100
box.Width = 100
box.Height = 50
cylinder.Radius = 25
cylinder.Height = 100

# Create boolean cut
cut = doc.addObject("Part::Cut", "Cut")
cut.Base = box
cut.Tool = cylinder

doc.recompute()
print("Boolean operation completed")
EOF

if docker exec "${FREECAD_CONTAINER_NAME}" freecadcmd < "$bool_script" 2>/dev/null | grep -q "Boolean operation completed"; then
    log::info "✓ Boolean operations work"
    rm "$bool_script"
else
    log::error "✗ Boolean operations failed"
    rm "$bool_script"
    exit 1
fi

# Test 4: Export functionality
log::info "Test 4: Testing export functionality..."
export_script="/tmp/freecad_export_test_${RANDOM}.py"
cat > "$export_script" <<'EOF'
import FreeCAD
import Part
import os

doc = FreeCAD.newDocument("ExportTest")
box = doc.addObject("Part::Box", "ExportBox")
box.Length = 50
box.Width = 50
box.Height = 50
doc.recompute()

# Test STEP export
output_file = "/tmp/test_export.step"
Part.export([box], output_file)

if os.path.exists(output_file):
    print("Export successful")
    os.remove(output_file)
else:
    print("Export failed")
    exit(1)
EOF

if docker exec "${FREECAD_CONTAINER_NAME}" python3 < "$export_script" 2>/dev/null | grep -q "Export successful"; then
    log::info "✓ Export functionality works"
    rm "$export_script"
else
    log::error "✗ Export functionality failed"
    rm "$export_script"
    exit 1
fi

# Test 5: Content management
log::info "Test 5: Testing content management..."

# Add a test script
test_content="/tmp/test_content_${RANDOM}.py"
echo "# Test FreeCAD script" > "$test_content"
echo "import FreeCAD" >> "$test_content"
echo "print('Content management test')" >> "$test_content"

freecad::content::add "$test_content"
if [[ -f "${FREECAD_SCRIPTS_DIR}/$(basename "$test_content")" ]]; then
    log::info "✓ Content add works"
    
    # Test content list
    if freecad::content::list | grep -q "$(basename "$test_content")"; then
        log::info "✓ Content list works"
    else
        log::error "✗ Content list failed"
        exit 1
    fi
    
    # Clean up
    freecad::content::remove "$(basename "$test_content")"
    rm "$test_content"
else
    log::error "✗ Content add failed"
    rm "$test_content"
    exit 1
fi

log::info "All integration tests passed successfully"
exit 0