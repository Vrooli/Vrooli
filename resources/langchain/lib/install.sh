#!/usr/bin/env bash
# LangChain Installation Functions
# Handle installation of LangChain framework

# Source required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
LANGCHAIN_INSTALL_DIR="${APP_ROOT}/resources/langchain/lib"
# shellcheck disable=SC1091
source "${LANGCHAIN_INSTALL_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_INSTALL_DIR}/../../../../lib/utils/log.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_INSTALL_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${LANGCHAIN_INSTALL_DIR}/core.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v langchain::export_config &>/dev/null; then
    langchain::export_config 2>/dev/null || true
fi

#######################################
# Install LangChain framework
#######################################
langchain::install() {
    log::header "Installing LangChain Framework"
    
    # Check if already installed
    if langchain::is_installed && langchain::check_packages; then
        log::success "LangChain is already installed"
        
        # Show current status
        if command -v langchain::status &>/dev/null; then
            langchain::status
        fi
        return 0
    fi
    
    # Initialize directories
    log::info "Initializing directories..."
    langchain::init_directories
    
    # Check Python availability
    if ! command -v python3 &>/dev/null; then
        log::error "Python 3 is required but not installed"
        log::info "Please install Python 3.8+ first"
        return 1
    fi
    
    # Check Python version
    local python_version
    python_version=$(python3 --version | cut -d' ' -f2 | cut -d'.' -f1,2)
    local required_version="3.8"
    
    if [[ "$(echo -e "$python_version\n$required_version" | sort -V | head -n1)" != "$required_version" ]]; then
        log::error "Python $required_version or higher is required (found $python_version)"
        return 1
    fi
    
    # Create virtual environment
    log::info "Creating Python virtual environment..."
    if ! langchain::create_venv; then
        log::error "Failed to create virtual environment"
        return 1
    fi
    
    # Install packages
    log::info "Installing LangChain packages..."
    if ! langchain::install_packages; then
        log::error "Failed to install LangChain packages"
        return 1
    fi
    
    # Create example chains and agents
    langchain::create_examples
    
    # Verify installation
    if langchain::is_healthy; then
        log::success "LangChain installation completed successfully"
        
        # Show status
        if command -v langchain::status &>/dev/null; then
            echo ""
            langchain::status
        fi
        
        return 0
    else
        log::error "LangChain installation completed but health check failed"
        return 1
    fi
}

#######################################
# Uninstall LangChain framework
#######################################
langchain::uninstall() {
    local force="${1:-false}"
    
    if [[ "$force" != "--force" ]]; then
        log::warn "This will remove LangChain and all associated data"
        log::warn "Use --force to confirm"
        return 1
    fi
    
    log::header "Uninstalling LangChain"
    
    # Stop any running processes
    if langchain::is_running; then
        log::info "Stopping LangChain processes..."
        pkill -f "langchain" 2>/dev/null || true
    fi
    
    # Remove virtual environment
    if [[ -d "$LANGCHAIN_VENV_DIR" ]]; then
        log::info "Removing virtual environment..."
        rm -rf "$LANGCHAIN_VENV_DIR"
    fi
    
    # Optional: Remove data directories
    log::info "Keeping data directories at $LANGCHAIN_DATA_DIR"
    log::info "Remove manually if no longer needed"
    
    log::success "LangChain uninstalled"
    return 0
}

#######################################
# Create example chains and agents
#######################################
langchain::create_examples() {
    log::info "Creating example chains and agents..."
    
    # Create a simple chain example
    cat > "${LANGCHAIN_CHAINS_DIR}/simple_chain.py" << 'EOF'
#!/usr/bin/env python3
"""Simple LangChain example chain"""

from langchain.chains import LLMChain
from langchain.prompts import PromptTemplate
from langchain.llms import Ollama

# Configure LLM (using Ollama as default)
llm = Ollama(model="llama3.2:3b", base_url="http://localhost:11434")

# Create prompt template
prompt = PromptTemplate(
    input_variables=["topic"],
    template="Tell me an interesting fact about {topic}."
)

# Create chain
chain = LLMChain(llm=llm, prompt=prompt)

if __name__ == "__main__":
    import sys
    topic = sys.argv[1] if len(sys.argv) > 1 else "artificial intelligence"
    result = chain.run(topic)
    print(result)
EOF
    
    # Create a simple agent example
    cat > "${LANGCHAIN_AGENTS_DIR}/simple_agent.py" << 'EOF'
#!/usr/bin/env python3
"""Simple LangChain agent with tools"""

from langchain.agents import initialize_agent, Tool
from langchain.agents import AgentType
from langchain.llms import Ollama

# Configure LLM
llm = Ollama(model="llama3.2:3b", base_url="http://localhost:11434")

# Define tools
def get_word_length(word: str) -> str:
    """Returns the length of a word."""
    return str(len(word))

tools = [
    Tool(
        name="Word Length",
        func=get_word_length,
        description="Useful for getting the length of a word"
    )
]

# Initialize agent
agent = initialize_agent(
    tools, 
    llm, 
    agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,
    verbose=True
)

if __name__ == "__main__":
    import sys
    query = " ".join(sys.argv[1:]) if len(sys.argv) > 1 else "How many letters are in the word 'artificial'?"
    result = agent.run(query)
    print(result)
EOF
    
    chmod +x "${LANGCHAIN_CHAINS_DIR}/simple_chain.py"
    chmod +x "${LANGCHAIN_AGENTS_DIR}/simple_agent.py"
    
    log::success "Example chains and agents created"
}

#######################################
# Start LangChain service (if applicable)
#######################################
langchain::start() {
    if ! langchain::is_installed; then
        log::error "LangChain is not installed"
        log::info "Run: resource-langchain install"
        return 1
    fi
    
    if langchain::is_running; then
        log::info "LangChain is already running"
        return 0
    fi
    
    # For now, LangChain is a framework library, not a service
    # Future: Could start an API server or agent runner
    log::info "LangChain is a framework library - no service to start"
    log::info "Use langchain::run_script to execute chains and agents"
    
    return 0
}

#######################################
# Stop LangChain service
#######################################
langchain::test() {
    log::header "Testing LangChain Connectivity"
    
    if ! langchain::is_installed; then
        log::error "LangChain is not installed"
        return 1
    fi
    
    log::info "Testing Python environment..."
    if ! langchain::execute_python "print('Python environment OK')"; then
        log::error "Python environment test failed"
        return 1
    fi
    
    log::info "Testing LangChain import..."
    if ! langchain::execute_python "import langchain; print(f'LangChain version: {langchain.__version__}')"; then
        log::error "LangChain import test failed"
        return 1
    fi
    
    log::info "Testing integrations..."
    
    # Test Redis connection if enabled
    if [[ "$LANGCHAIN_CACHE_TYPE" == "redis" ]] && [[ -n "$LANGCHAIN_REDIS_URL" ]]; then
        log::info "Testing Redis connection..."
        if langchain::execute_python "import redis; r = redis.from_url('$LANGCHAIN_REDIS_URL'); r.ping()"; then
            log::success "Redis connection OK"
        else
            log::warn "Redis connection failed (non-critical)"
        fi
    fi
    
    # Test Ollama connection if enabled
    if [[ "$LANGCHAIN_ENABLE_OLLAMA" == "true" ]]; then
        log::info "Testing Ollama connection..."
        if curl -sf "http://localhost:11434/api/tags" >/dev/null 2>&1; then
            log::success "Ollama connection OK"
        else
            log::warn "Ollama not responding (check if resource is running)"
        fi
    fi
    
    # Test OpenRouter if enabled
    if [[ "$LANGCHAIN_ENABLE_OPENROUTER" == "true" ]]; then
        log::info "Testing OpenRouter availability..."
        if [[ -n "${OPENROUTER_API_KEY:-}" ]]; then
            log::success "OpenRouter API key configured"
        else
            log::warn "OpenRouter API key not found in environment"
        fi
    fi
    
    log::success "LangChain connectivity test completed"
    return 0
}

langchain::stop() {
    if ! langchain::is_running; then
        log::info "LangChain is not running"
        return 0
    fi
    
    log::info "Stopping LangChain processes..."
    pkill -f "langchain" 2>/dev/null || true
    
    log::success "LangChain processes stopped"
    return 0
}

# Export functions
export -f langchain::install
export -f langchain::uninstall
export -f langchain::start
export -f langchain::stop