#!/bin/bash

# Generate routine-reference.json from staged routines
# This script scans all routine JSON files in the staged directory
# and creates a reference file for agent validation

echo "Generating routine reference from staged routines..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STAGED_DIR="$SCRIPT_DIR/staged"
OUTPUT_FILE="$SCRIPT_DIR/routine-reference.json"

# Check if staged directory exists
if [ ! -d "$STAGED_DIR" ]; then
    echo "Error: Staged directory not found at $STAGED_DIR"
    exit 1
fi

# Check if Python is available
if ! command -v python3 >/dev/null 2>&1; then
    echo "Error: Python 3 is required"
    exit 1
fi

# Pass directory paths as environment variables
export SCRIPT_DIR STAGED_DIR OUTPUT_FILE

# Use Python to process all files
python3 << 'EOF'
import json
import os
import glob
from datetime import datetime
from collections import defaultdict

# Get paths from environment
script_dir = os.environ['SCRIPT_DIR']
staged_dir = os.environ['STAGED_DIR']
output_file = os.environ['OUTPUT_FILE']

print(f"Processing routines in: {staged_dir}")

# Initialize data structure
data = {
    "metadata": {
        "generatedOn": datetime.utcnow().isoformat() + "Z",
        "totalRoutines": 0,
        "version": "1.0"
    },
    "routines": [],
    "byCategory": defaultdict(list),
    "byType": defaultdict(list),
    "byName": {}
}

routine_count = 0

# Find all JSON files in staged directory
for root, dirs, files in os.walk(staged_dir):
    for file in files:
        if not file.endswith(".json"):
            continue
        
        # Skip special files
        if file.startswith("_") or file.startswith("tags-") or file == "all-tags-for-import.json":
            continue
            
        filepath = os.path.join(root, file)
        relative_path = os.path.relpath(filepath, script_dir)
        
        try:
            with open(filepath, 'r') as f:
                routine_data = json.load(f)
            
            # Check if this is a routine file
            if routine_data.get("resourceType") != "Routine":
                continue
            
            # Extract routine information
            routine_id = routine_data.get("id", "")
            public_id = routine_data.get("publicId", "")
            
            # Get first version data
            versions = routine_data.get("versions", [])
            if not versions:
                continue
                
            version = versions[0]
            version_id = version.get("id", "")
            version_label = version.get("versionLabel", "")
            version_notes = version.get("versionNotes", "")
            is_automatable = version.get("isAutomatable", False)
            
            # Extract name from translations or derive from publicId
            name = ""
            translations = routine_data.get("translations", [])
            if translations and translations[0].get("name"):
                name = translations[0]["name"]
            else:
                # Derive from publicId (remove version suffix and convert to title case)
                name = public_id.replace("-v1", "").replace("-", " ").title()
            
            # Get category from directory structure
            category_path = os.path.relpath(root, staged_dir)
            category = category_path if category_path != "." else "uncategorized"
            
            # Get tags
            tags = [tag.get("tag", "") for tag in routine_data.get("tags", []) if tag.get("tag")]
            
            # Create routine entry
            routine_entry = {
                "id": routine_id,
                "publicId": public_id,
                "name": name,
                "category": category,
                "versionId": version_id,
                "versionLabel": version_label,
                "description": version_notes,
                "isAutomatable": is_automatable,
                "tags": tags,
                "filePath": relative_path
            }
            
            data["routines"].append(routine_entry)
            
            # Build indices
            category_entry = {
                "id": routine_id,
                "name": name,
                "publicId": public_id
            }
            
            data["byCategory"][category].append(category_entry)
            
            for tag in tags:
                data["byType"][tag].append(category_entry)
            
            if name:
                data["byName"][name] = {
                    "id": routine_id,
                    "publicId": public_id,
                    "versionId": version_id,
                    "isAutomatable": is_automatable
                }
            
            routine_count += 1
            print(f"  Processed: {name} ({public_id})")
            
        except Exception as e:
            print(f"  Error processing {filepath}: {e}")
            continue

# Convert defaultdicts to regular dicts
data["byCategory"] = dict(data["byCategory"])
data["byType"] = dict(data["byType"])

# Update total count
data["metadata"]["totalRoutines"] = routine_count

# Write output file
with open(output_file, 'w') as f:
    json.dump(data, f, indent=2, sort_keys=True)

print(f"\nGenerated routine reference with {routine_count} routines")
print(f"Output saved to: {output_file}")

# Also copy to agent directory
agent_ref_file = os.path.join(os.path.dirname(script_dir), "agent", "routine-reference.json")
try:
    import shutil
    shutil.copy2(output_file, agent_ref_file)
    print(f"Copied to agent directory: {agent_ref_file}")
except Exception as e:
    print(f"Could not copy to agent directory: {e}")
EOF

# Update the agent routine index if the script exists
if [ -f "$SCRIPT_DIR/../agent/create-routine-index.sh" ]; then
    echo "Updating agent routine index..."
    (cd "$SCRIPT_DIR/../agent" && bash create-routine-index.sh)
fi