#!/usr/bin/env bash
# LangChain Injection Functions
# Handle injection of chains, agents, and workflows into LangChain

# Source required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LANGCHAIN_INJECT_DIR="${APP_ROOT}/resources/langchain/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/langchain/config/defaults.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_INJECT_DIR}/core.sh"

# Ensure configuration is exported
if command -v langchain::export_config &>/dev/null; then
    langchain::export_config 2>/dev/null || true
fi

#######################################
# Inject a chain or agent file
#######################################
langchain::inject_file() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-langchain inject <file>"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Determine file type
    local filename
    filename=$(basename "$file")
    local ext="${filename##*.}"
    
    case "$ext" in
        py)
            langchain::inject_python "$file"
            ;;
        json)
            langchain::inject_config "$file"
            ;;
        *)
            log::error "Unsupported file type: $ext"
            log::info "Supported types: .py (Python scripts), .json (configurations)"
            return 1
            ;;
    esac
}

#######################################
# Inject a Python chain or agent
#######################################
langchain::inject_python() {
    local source_file="$1"
    local filename
    filename=$(basename "$source_file")
    
    if ! langchain::is_installed; then
        log::error "LangChain is not installed"
        return 1
    fi
    
    log::info "Injecting Python file: $filename"
    
    # Determine target directory based on content
    local target_dir="$LANGCHAIN_WORKSPACE_DIR"
    
    if grep -q "class.*Agent\|from langchain.agents" "$source_file" 2>/dev/null; then
        target_dir="$LANGCHAIN_AGENTS_DIR"
        log::info "Detected agent - installing to agents directory"
    elif grep -q "class.*Chain\|from langchain.chains" "$source_file" 2>/dev/null; then
        target_dir="$LANGCHAIN_CHAINS_DIR"
        log::info "Detected chain - installing to chains directory"
    else
        log::info "Installing to workspace directory"
    fi
    
    # Copy file to target directory
    cp "$source_file" "$target_dir/$filename"
    chmod +x "$target_dir/$filename"
    
    # Validate Python syntax
    if "${LANGCHAIN_VENV_DIR}/bin/python" -m py_compile "$target_dir/$filename" 2>/dev/null; then
        log::success "Python file injected successfully: $target_dir/$filename"
        
        # Try to import and list available functions/classes
        local imports
        imports=$("${LANGCHAIN_VENV_DIR}/bin/python" -c "
import sys
sys.path.insert(0, '$target_dir')
try:
    module = __import__('${filename%.py}')
    items = [item for item in dir(module) if not item.startswith('_')]
    print('Available items:', ', '.join(items))
except Exception as e:
    print(f'Import check: {e}')
" 2>&1)
        
        [[ -n "$imports" ]] && log::info "$imports"
        
        return 0
    else
        log::error "Python syntax validation failed"
        rm "$target_dir/$filename"
        return 1
    fi
}

#######################################
# Inject a configuration file
#######################################
langchain::inject_config() {
    local config_file="$1"
    local filename
    filename=$(basename "$config_file")
    
    log::info "Injecting configuration: $filename"
    
    # Parse JSON configuration
    if ! jq empty "$config_file" 2>/dev/null; then
        log::error "Invalid JSON file"
        return 1
    fi
    
    # Extract configuration type
    local config_type
    config_type=$(jq -r '.type // "unknown"' "$config_file")
    
    case "$config_type" in
        "chain")
            langchain::inject_chain_config "$config_file"
            ;;
        "agent")
            langchain::inject_agent_config "$config_file"
            ;;
        "workflow")
            langchain::inject_workflow_config "$config_file"
            ;;
        *)
            log::warn "Unknown configuration type: $config_type"
            log::info "Copying to workspace directory"
            cp "$config_file" "$LANGCHAIN_WORKSPACE_DIR/$filename"
            log::success "Configuration saved to workspace"
            ;;
    esac
}

#######################################
# Main inject function - entry point for CLI
#######################################
langchain::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "Usage: resource-langchain inject <file>"
        return 1
    fi
    
    langchain::inject_file "$file"
}

#######################################
# Inject a chain configuration
#######################################
langchain::inject_chain_config() {
    local config_file="$1"
    local name
    name=$(jq -r '.name // "unnamed_chain"' "$config_file")
    
    log::info "Creating chain: $name"
    
    # Generate Python code from configuration
    local python_file="${LANGCHAIN_CHAINS_DIR}/${name}.py"
    
    cat > "$python_file" << 'EOF'
#!/usr/bin/env python3
"""Auto-generated chain from configuration"""

import json
import sys
from pathlib import Path

# Load configuration
config_path = Path(__file__).parent / f"{Path(__file__).stem}.json"
with open(config_path) as f:
    config = json.load(f)

# Import required modules
from langchain.chains import LLMChain
from langchain.prompts import PromptTemplate

# Setup based on configuration
def create_chain():
    # LLM setup
    llm_config = config.get('llm', {})
    llm_type = llm_config.get('type', 'ollama')
    
    if llm_type == 'ollama':
        from langchain.llms import Ollama
        llm = Ollama(
            model=llm_config.get('model', 'llama3.2:3b'),
            base_url=llm_config.get('base_url', 'http://localhost:11434')
        )
    else:
        raise ValueError(f"Unsupported LLM type: {llm_type}")
    
    # Prompt setup
    prompt_config = config.get('prompt', {})
    prompt = PromptTemplate(
        input_variables=prompt_config.get('variables', ['input']),
        template=prompt_config.get('template', '{input}')
    )
    
    # Create chain
    return LLMChain(llm=llm, prompt=prompt)

if __name__ == "__main__":
    chain = create_chain()
    if len(sys.argv) > 1:
        result = chain.run(" ".join(sys.argv[1:]))
        print(result)
    else:
        print(f"Chain '{config.get('name', 'unnamed')}' loaded. Pass arguments to run.")
EOF
    
    # Save configuration alongside Python file
    cp "$config_file" "${LANGCHAIN_CHAINS_DIR}/${name}.json"
    chmod +x "$python_file"
    
    log::success "Chain configuration injected: $name"
    return 0
}

#######################################
# Inject an agent configuration
#######################################
langchain::inject_agent_config() {
    local config_file="$1"
    local name
    name=$(jq -r '.name // "unnamed_agent"' "$config_file")
    
    log::info "Creating agent: $name"
    
    # Similar to chain but for agents
    local python_file="${LANGCHAIN_AGENTS_DIR}/${name}.py"
    cp "$config_file" "${LANGCHAIN_AGENTS_DIR}/${name}.json"
    
    # Generate basic agent wrapper
    cat > "$python_file" << 'EOF'
#!/usr/bin/env python3
"""Auto-generated agent from configuration"""

import json
import sys
from pathlib import Path

# Load configuration
config_path = Path(__file__).parent / f"{Path(__file__).stem}.json"
with open(config_path) as f:
    config = json.load(f)

print(f"Agent '{config.get('name', 'unnamed')}' configuration loaded")
# TODO: Implement agent creation based on configuration
EOF
    
    chmod +x "$python_file"
    log::success "Agent configuration injected: $name"
    return 0
}

#######################################
# Inject a workflow configuration
#######################################
langchain::inject_workflow_config() {
    local config_file="$1"
    local name
    name=$(jq -r '.name // "unnamed_workflow"' "$config_file")
    
    log::info "Creating workflow: $name"
    cp "$config_file" "${LANGCHAIN_WORKSPACE_DIR}/${name}_workflow.json"
    
    log::success "Workflow configuration saved: $name"
    return 0
}

# Export functions
export -f langchain::inject_file
export -f langchain::inject_python
export -f langchain::inject_config