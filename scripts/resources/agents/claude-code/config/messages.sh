#!/usr/bin/env bash
# Claude Code Messages and Help Text
# This file contains all user-facing messages and documentation

#######################################
# Display usage information
#######################################
claude_code::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Actions:"
    echo "  install      Install Claude Code CLI globally"
    echo "  uninstall    Remove Claude Code CLI"
    echo "  status       Check installation status"
    echo "  info         Display detailed information"
    echo "  run          Run Claude with a prompt"
    echo "  batch        Run Claude in batch mode"
    echo "  session      Manage Claude sessions"
    echo "  settings     View or update Claude settings"
    echo "  logs         View Claude logs"
    echo ""
    echo "MCP Actions:"
    echo "  register-mcp    Register Vrooli as MCP server with Claude Code"
    echo "  unregister-mcp  Remove Vrooli MCP server registration"
    echo "  mcp-status      Check Vrooli MCP registration status"
    echo "  mcp-test        Test MCP connection to Vrooli server"
    echo ""
    echo "Automation Actions:"
    echo "  parse-result    Parse Claude Code output into structured format"
    echo "  extract         Extract specific information from results"
    echo "  session-manage  Advanced session management operations"
    echo "  health-check    Check Claude Code health and capabilities"
    echo "  run-automation  Run Claude with automation-friendly output"
    echo "  batch-automation Enhanced batch processing with progress tracking"
    echo ""
    echo "Enhanced Batch Actions:"
    echo "  batch-simple    Clean batch execution with automatic session handling"
    echo "  batch-config    Configuration-based batch execution (JSON/YAML)"
    echo "  batch-multi     Execute multiple tasks in sequence"
    echo "  batch-parallel  Execute multiple tasks in parallel"
    echo ""
    echo "Error Handling Actions:"
    echo "  error-report    Generate detailed error report for debugging"
    echo "  error-validate  Validate error handling configuration"
    echo "  safe-execute    Execute with automatic retry and recovery"
    echo ""
    echo "Template System Actions:"
    echo "  template-list     List available prompt templates"
    echo "  template-load     Load and process a template"  
    echo "  template-run      Execute Claude with a template"
    echo "  template-create   Create a new template"
    echo "  template-info     Show template information"
    echo "  template-validate Validate template syntax"
    echo ""
    echo "Enhanced Session Management Actions:"
    echo "  session-extract      Extract comprehensive results from session output"
    echo "  session-analytics    Generate session usage analytics and reports"
    echo "  session-recover      Advanced session recovery with retry strategies"
    echo "  session-cleanup      Clean up old, failed, or large sessions"
    echo "  session-list-enhanced Enhanced session listing with filtering and sorting"
    echo ""
    echo "Sandbox Actions:"
    echo "  sandbox      Run Claude Code in isolated sandbox environment"
    echo ""
    echo "  help         Show this help message"
    echo
    echo "Examples:"
    echo "  $0 --action install                    # Install Claude Code"
    echo "  $0 --action status                     # Check installation status"
    echo "  $0 --action run --prompt \"Explain this code\"  # Run a prompt"
    echo "  $0 --action batch --prompt \"Write tests\" --max-turns 10"
    echo "  $0 --action session --session-id abc123  # Resume session"
    echo "  $0 --action settings                   # View current settings"
    echo "  $0 --action uninstall                  # Remove Claude Code"
    echo ""
    echo "MCP Examples:"
    echo "  $0 --action register-mcp               # Auto-register Vrooli MCP server"
    echo "  $0 --action register-mcp --scope user  # Register in user scope"
    echo "  $0 --action mcp-status --format json   # Get status in JSON format"
    echo "  $0 --action mcp-test                   # Test MCP connection"
    echo "  $0 --action unregister-mcp             # Remove MCP registration"
    echo ""
    echo "Automation Examples:"
    echo "  $0 --action health-check --check-type full --format json"
    echo "  $0 --action parse-result --input-file output.json --format automation"
    echo "  $0 --action extract --input-file output.json --extract-type files"
    echo "  $0 --action run-automation --prompt \"Fix bugs\" --allowed-tools \"Read,Edit\""
    echo "  $0 --action session-manage list        # List recent sessions"
    echo "  $0 --action batch-automation --prompt \"Refactor code\" --max-turns 100"
    echo ""
    echo "Enhanced Batch Examples:"
    echo "  $0 --action batch-simple --prompt \"Fix all bugs\" --max-turns 100"
    echo "  $0 --action batch-config --config-file workflow.json"
    echo "  $0 --action batch-parallel --workers 3 --prompt \"Analyze code\" --max-turns 50"
    echo ""
    echo "Error Handling Examples:"
    echo "  $0 --action error-report --exit-code 1 --error-context \"deploy script\""
    echo "  $0 --action error-validate                        # Check error handling config"
    echo "  $0 --action safe-execute --prompt \"Fix bugs\" --max-retries 5"
    echo ""
    echo "Template System Examples:"
    echo "  $0 --action template-list                        # List available templates"
    echo "  $0 --action template-run --template-name security-audit --template-vars \"target=src/auth/,scope=authentication\""
    echo "  $0 --action template-create --template-name my-template --template-description \"Custom code review\""
    echo "  $0 --action template-info --template-name code-review  # Show template details"
    echo ""
    echo "Enhanced Session Management Examples:"
    echo "  $0 --action session-extract --output-file session.json --format detailed"
    echo "  $0 --action session-analytics --analysis-period month --format json"
    echo "  $0 --action session-recover --session-id abc123 --recovery-strategy auto"
    echo "  $0 --action session-cleanup --cleanup-strategy old --threshold 7    # Clean sessions older than 7 days"
    echo "  $0 --action session-list-enhanced --filter completed --sort-by turns --limit 10"
    echo ""
    echo "Sandbox Examples:"
    echo "  $0 --action sandbox --sandbox-command setup          # Set up sandbox authentication"
    echo "  $0 --action sandbox                                  # Start interactive sandbox session"
    echo "  $0 --action sandbox --sandbox-command run --prompt \"Test code\"  # Run in sandbox"
    echo "  $0 --action sandbox --sandbox-command stop           # Stop sandbox container"
    echo
    echo "Requirements:"
    echo "  - Node.js version $MIN_NODE_VERSION or newer"
    echo "  - npm package manager"
    echo "  - Valid Claude Pro or Max subscription for full functionality"
}

#######################################
# Display detailed information about Claude Code
#######################################
claude_code::display_info() {
    echo "Claude Code is Anthropic's official CLI for Claude."
    echo
    echo "Key Features:"
    echo "  • Deep codebase awareness"
    echo "  • Direct file editing and command execution"
    echo "  • Claude Opus 4 model integration"
    echo "  • Composable and scriptable interface"
    echo "  • Automatic updates"
    echo
    echo "Requirements:"
    echo "  • Node.js $MIN_NODE_VERSION or newer"
    echo "  • npm package manager"
    echo "  • Claude Pro or Max subscription"
    echo
    echo "Subscription Plans:"
    echo "  • Pro (\$20/month): Light coding on small repositories"
    echo "  • Max (\$100/month): Heavy usage, ~50-200 prompts per 5 hours"
    echo
    echo "Usage:"
    echo "  1. Install: npm install -g $CLAUDE_PACKAGE"
    echo "  2. Navigate to project: cd /your/project"
    echo "  3. Start Claude: claude"
    echo
    echo "Documentation: https://docs.anthropic.com/claude-code"
    echo "Support: https://support.anthropic.com"
}

#######################################
# Next steps message after successful MCP registration
#######################################
claude_code::mcp_next_steps() {
    echo
    log::header "🎯 Next Steps"
    log::info "1. Start Claude Code: claude"
    log::info "2. Use Vrooli tools with @vrooli prefix:"
    log::info "   • @vrooli send_message to send chat messages"
    log::info "   • @vrooli run_routine to execute routines"
    log::info "   • @vrooli spawn_swarm to start AI swarms"
    log::info "3. Type '@' in Claude Code to see available resources"
}

#######################################
# Next steps message after installation
#######################################
claude_code::install_next_steps() {
    echo
    log::header "🎯 Next Steps"
    log::info "1. Navigate to your project directory: cd /path/to/project"
    log::info "2. Start Claude Code: claude"
    log::info "3. Login with your Claude credentials when prompted"
    log::info "4. Use 'claude doctor' to verify your installation"
}