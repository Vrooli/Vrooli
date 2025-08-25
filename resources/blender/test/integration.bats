#!/usr/bin/env bats

# Blender integration tests

# Test directory setup
setup() {
    load '../../../../../__test/helpers/bats-support/load'
    load '../../../../../__test/helpers/bats-assert/load'
    
    # Source the CLI
    BLENDER_CLI="${BATS_TEST_DIRNAME}/../cli.sh"
    
    # Test data from shared fixtures
    TEST_BLEND_FILE="${BATS_TEST_DIRNAME}/../../../../__test/fixtures/data/cube.blend"
    TEST_PYTHON_SCRIPT="${BATS_TEST_DIRNAME}/../../../../__test/fixtures/data/blender_test.py"
    TEST_OUTPUT_DIR="/tmp/blender-test-$$"
    mkdir -p "$TEST_OUTPUT_DIR"
}

teardown() {
    # Clean up test output
    rm -rf "$TEST_OUTPUT_DIR"
}

@test "blender: status returns healthy when running" {
    run bash "$BLENDER_CLI" status --format json
    assert_success
    
    # Parse JSON and check status
    echo "$output" | jq -e '.installed == true'
    echo "$output" | jq -e '.running == true'
    echo "$output" | jq -e '.health == "healthy"'
}

@test "blender: can render a simple scene" {
    # Create a simple Python script to test rendering
    cat > "$TEST_OUTPUT_DIR/test_render.py" << 'EOF'
import bpy
# Clear existing mesh objects
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete(use_global=False)
# Add a cube
bpy.ops.mesh.primitive_cube_add(location=(0, 0, 0))
# Add a light
bpy.ops.object.light_add(type='SUN', location=(0, 0, 5))
# Add a camera
bpy.ops.object.camera_add(location=(5, -5, 5))
camera = bpy.context.object
camera.rotation_euler = (1.1, 0, 0.785)
bpy.context.scene.camera = camera
# Set render output
bpy.context.scene.render.filepath = '/output/test_render.png'
bpy.context.scene.render.resolution_x = 320
bpy.context.scene.render.resolution_y = 240
# Render
bpy.ops.render.render(write_still=True)
EOF
    
    # Inject and run the script
    run bash "$BLENDER_CLI" inject "$TEST_OUTPUT_DIR/test_render.py"
    assert_success
    
    run bash "$BLENDER_CLI" run test_render.py
    assert_success
    
    # Check that output was created
    run bash "$BLENDER_CLI" export test_render.png "$TEST_OUTPUT_DIR/rendered.png"
    assert_success
    assert [ -f "$TEST_OUTPUT_DIR/rendered.png" ]
}

@test "blender: can run Python scripts" {
    # Create a test Python script
    cat > "$TEST_OUTPUT_DIR/test_script.py" << 'EOF'
import bpy
import json
# Get Blender version info
info = {
    "version": ".".join(map(str, bpy.app.version)),
    "version_string": bpy.app.version_string,
    "build_branch": bpy.app.build_branch,
}
# Write to output
with open('/output/version_info.json', 'w') as f:
    json.dump(info, f)
EOF
    
    # Inject and run
    run bash "$BLENDER_CLI" inject "$TEST_OUTPUT_DIR/test_script.py"
    assert_success
    
    run bash "$BLENDER_CLI" run test_script.py
    assert_success
    
    # Export and verify output
    run bash "$BLENDER_CLI" export version_info.json "$TEST_OUTPUT_DIR/version.json"
    assert_success
    assert [ -f "$TEST_OUTPUT_DIR/version.json" ]
    
    # Check JSON content
    run jq -e '.version' "$TEST_OUTPUT_DIR/version.json"
    assert_success
}

@test "blender: can list injected scripts" {
    # Inject a test script first
    cat > "$TEST_OUTPUT_DIR/list_test.py" << 'EOF'
print("Test script for listing")
EOF
    
    run bash "$BLENDER_CLI" inject "$TEST_OUTPUT_DIR/list_test.py"
    assert_success
    
    # List scripts
    run bash "$BLENDER_CLI" list
    assert_success
    assert_output --partial "list_test.py"
}

@test "blender: can export rendered outputs" {
    # Create a script that generates output
    cat > "$TEST_OUTPUT_DIR/export_test.py" << 'EOF'
import bpy
# Clear existing objects
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete(use_global=False)
# Create simple scene and render
bpy.ops.mesh.primitive_sphere_add(location=(0, 0, 0))
# Add camera and light
bpy.ops.object.camera_add(location=(3, -3, 3))
camera = bpy.context.object
camera.rotation_euler = (1.1, 0, 0.785)
bpy.context.scene.camera = camera
bpy.ops.object.light_add(type='SUN', location=(0, 0, 5))
# Set render settings
bpy.context.scene.render.filepath = '/output/sphere.png'
bpy.context.scene.render.resolution_x = 100
bpy.context.scene.render.resolution_y = 100
bpy.ops.render.render(write_still=True)
EOF
    
    run bash "$BLENDER_CLI" inject "$TEST_OUTPUT_DIR/export_test.py"
    assert_success
    
    run bash "$BLENDER_CLI" run export_test.py
    assert_success
    
    # Export the rendered image
    run bash "$BLENDER_CLI" export sphere.png "$TEST_OUTPUT_DIR/sphere_out.png"
    assert_success
    assert [ -f "$TEST_OUTPUT_DIR/sphere_out.png" ]
}

@test "blender: handles missing script gracefully" {
    run bash "$BLENDER_CLI" run nonexistent_script.py
    assert_failure
    assert_output --partial "not found"
}

@test "blender: validates Python syntax before injection" {
    # Create invalid Python
    cat > "$TEST_OUTPUT_DIR/invalid.py" << 'EOF'
import bpy
this is not valid python syntax
EOF
    
    run bash "$BLENDER_CLI" inject "$TEST_OUTPUT_DIR/invalid.py"
    # Should either fail or warn about syntax
    if [ "$status" -eq 0 ]; then
        run bash "$BLENDER_CLI" run invalid.py
        assert_failure
    else
        assert_output --partial "syntax"
    fi
}

@test "blender: supports batch processing" {
    # Create multiple test scripts
    for i in 1 2 3; do
        cat > "$TEST_OUTPUT_DIR/batch_$i.py" << EOF
import bpy
with open('/output/batch_$i.txt', 'w') as f:
    f.write('Processed batch $i')
EOF
        run bash "$BLENDER_CLI" inject "$TEST_OUTPUT_DIR/batch_$i.py"
        assert_success
    done
    
    # Run all scripts
    for i in 1 2 3; do
        run bash "$BLENDER_CLI" run "batch_$i.py"
        assert_success
    done
    
    # Verify outputs
    for i in 1 2 3; do
        run bash "$BLENDER_CLI" export "batch_$i.txt" "$TEST_OUTPUT_DIR/batch_$i.txt"
        assert_success
        assert [ -f "$TEST_OUTPUT_DIR/batch_$i.txt" ]
    done
}

@test "blender: CLI is registered with vrooli" {
    run which resource-blender
    assert_success
}

@test "blender: documentation exists and is complete" {
    assert [ -f "${BATS_TEST_DIRNAME}/../README.md" ]
    
    # Check for essential sections
    run grep -q "## Overview" "${BATS_TEST_DIRNAME}/../README.md"
    assert_success
    
    run grep -q "## Installation" "${BATS_TEST_DIRNAME}/../README.md"
    assert_success
    
    run grep -q "## Usage" "${BATS_TEST_DIRNAME}/../README.md"
    assert_success
}