# CLI Integration

## Core Principle
Every resource must provide a consistent CLI interface for automation and manual operation.

## Learning From Existing Resources
**IMPORTANT**: Before implementing CLI commands, study existing resources:
```bash
# Learn from any existing resource's CLI implementation
vrooli resource [existing-resource-name] help

# Example: Learn from ollama's CLI
vrooli resource ollama help
vrooli resource ollama status
vrooli resource ollama content list
```

## Required Commands
Every resource MUST implement these standard commands:
- `setup` - One-time installation and configuration
- `start/develop` - Start the resource
- `stop` - Stop the resource  
- `health` - Check health status (must return quickly)
- `test` - Run tests
- `help` - Show usage information

## Full Contract Reference
**CRITICAL**: The complete v2.0 CLI contract with all requirements is defined in:
```
/scripts/resources/contracts/v2.0/universal.yaml
```

This contract specifies:
- All required and optional commands
- Expected command behaviors and outputs
- Error handling patterns
- JSON output formats
- Port management requirements
- Health check specifications
- Logging standards

## Implementation Approach
1. Study existing resource CLIs using `vrooli resource [name] help`
2. Review the v2.0 contract at `/scripts/resources/contracts/v2.0/universal.yaml`
3. Implement required commands following the patterns you observe
4. Ensure health checks respond within 5 seconds
5. Support both human-readable and `--json` output formats
6. Use consistent exit codes (0=success, 1=error)

## Validation
Test your CLI implementation:
```bash
# Verify all required commands exist
vrooli resource [your-resource] help

# Test lifecycle
vrooli resource [your-resource] setup
vrooli resource [your-resource] start
vrooli resource [your-resource] health
vrooli resource [your-resource] stop

# Verify JSON output
vrooli resource [your-resource] health --json
```

Remember: Consistency across resources is more important than unique features. Follow the established patterns.