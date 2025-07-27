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