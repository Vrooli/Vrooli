#\!/bin/bash

# Injection functions for AutoGPT

autogpt_inject() {
    local file="${1:-}"
    
    if [ -z "$file" ]; then
        echo "[ERROR]   No file specified for injection"
        echo "Usage: resource-autogpt inject <file>"
        return 1
    fi
    
    if [ \! -f "$file" ]; then
        echo "[ERROR]   File not found: $file"
        return 1
    fi
    
    local filename=$(basename "$file")
    local extension="${filename##*.}"
    
    echo "[INFO]    Injecting $filename into AutoGPT..."
    
    case "$extension" in
        yaml|yml)
            # Agent configuration
            cp "$file" "$AUTOGPT_AGENTS_DIR/"
            echo "[SUCCESS] Agent configuration injected: $filename"
            ;;
        py)
            # Python tool/plugin
            cp "$file" "$AUTOGPT_TOOLS_DIR/"
            echo "[SUCCESS] Tool/plugin injected: $filename"
            ;;
        *)
            echo "[ERROR]   Unsupported file type: .$extension"
            echo "[INFO]    Supported types: .yaml, .yml (agents), .py (tools)"
            return 1
            ;;
    esac
    
    # If container is running, reload configurations
    if autogpt_container_running; then
        echo "[INFO]    Reloading AutoGPT configurations..."
        docker exec "$AUTOGPT_CONTAINER_NAME" pkill -HUP python 2>/dev/null || true
    fi
    
    return 0
}

# Create an agent
autogpt_create_agent() {
    local name="${1:-}"
    local goal="${2:-}"
    local model="${3:-auto}"
    
    if [ -z "$name" ] || [ -z "$goal" ]; then
        echo "[ERROR]   Agent name and goal are required"
        echo "Usage: resource-autogpt create-agent <name> <goal> [model]"
        return 1
    fi
    
    echo "[INFO]    Creating agent: $name"
    
    cat > "$AUTOGPT_AGENTS_DIR/${name}.yaml" << YAML
name: $name
role: "Assistant"
goals:
  - "$goal"
constraints:
  - "Work within the workspace directory"
  - "Minimize costs and API calls"
  - "Provide clear progress updates"
best_practices:
  - "Break down complex tasks into smaller steps"
  - "Verify results before proceeding"
  - "Document findings and decisions"
model: $model
YAML
    
    echo "[SUCCESS] Agent created: $name"
    echo "[INFO]    Configuration saved to: $AUTOGPT_AGENTS_DIR/${name}.yaml"
    
    return 0
}

# List agents
autogpt_list_agents() {
    echo "[HEADER]  Available AutoGPT Agents"
    echo ""
    
    if [ \! -d "$AUTOGPT_AGENTS_DIR" ] || [ -z "$(ls -A "$AUTOGPT_AGENTS_DIR" 2>/dev/null)" ]; then
        echo "[INFO]    No agents found"
        return 0
    fi
    
    for agent_file in "$AUTOGPT_AGENTS_DIR"/*.yaml "$AUTOGPT_AGENTS_DIR"/*.yml; do
        [ -f "$agent_file" ] || continue
        local agent_name=$(basename "$agent_file" .yaml)
        agent_name=$(basename "$agent_name" .yml)
        local goal=$(grep -m1 "^  - " "$agent_file" 2>/dev/null | sed 's/^  - "\?\(.*\)"\?$/\1/')
        echo "  • $agent_name: ${goal:-No goal specified}"
    done
    
    echo ""
    return 0
}

# Run an agent
autogpt_run_agent() {
    local agent_name="${1:-}"
    
    if [ -z "$agent_name" ]; then
        echo "[ERROR]   Agent name required"
        echo "Usage: resource-autogpt run-agent <name>"
        return 1
    fi
    
    local agent_file="$AUTOGPT_AGENTS_DIR/${agent_name}.yaml"
    if [ \! -f "$agent_file" ]; then
        agent_file="$AUTOGPT_AGENTS_DIR/${agent_name}.yml"
    fi
    
    if [ \! -f "$agent_file" ]; then
        echo "[ERROR]   Agent not found: $agent_name"
        return 1
    fi
    
    if \! autogpt_container_running; then
        echo "[ERROR]   AutoGPT is not running. Start it first with: resource-autogpt start"
        return 1
    fi
    
    echo "[INFO]    Running agent: $agent_name"
    
    # Execute agent in container
    docker exec -it "$AUTOGPT_CONTAINER_NAME" python -m autogpt.agent \
        --config "/app/agents/${agent_name}.yaml" \
        --workspace "/workspace" \
        --continuous
    
    return $?
}
