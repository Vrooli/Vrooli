# Getting Started with Vrooli

Vrooli is a resource orchestration platform that runs scenarios directly to create business applications. This guide will get you up and running in 15 minutes.

## Quick Start (5 Minutes)

### 1. Setup Environment
```bash
# Clone and navigate to Vrooli
git clone https://github.com/Vrooli/Vrooli.git
cd Vrooli

# Run initial setup
vrooli setup
```

### 2. Start Core Resources
```bash
# Start essential resources (PostgreSQL, Redis, Ollama)
vrooli resource start-all

# Verify resources are running
vrooli resource status
```

### 3. Run Your First Scenario
```bash
# List available scenarios
vrooli scenario list

# Run a simple scenario
vrooli scenario run research-assistant

# Test scenario integration
vrooli scenario test research-assistant
```

You now have Vrooli running with a working scenario!

## Core Concepts (5 Minutes)

### Resources
**Resources** are the foundational services that provide capabilities:
- **AI**: Ollama (local LLM), Whisper (speech-to-text), ComfyUI (image generation)
- **Storage**: PostgreSQL (database), Redis (cache), Qdrant (vector database)
- **Automation**: N8n (workflows), Windmill (code-first automation)
- **Agents**: Agent-S2 (screen automation), Browserless (web automation)

**Management**: `vrooli resource <command>` or direct `resource-<name> <command>`

### Scenarios
**Scenarios** are complete business applications that orchestrate multiple resources:
- **Purpose**: Deliver specific business value ($10K-50K revenue potential)
- **Examples**: Research Assistant, Invoice Generator, Customer Service Portal
- **Architecture**: Combine resources to create emergent business capabilities
- **Deployment**: Run directly without conversion to standalone apps

**Management**: `vrooli scenario <command>`

### The Key Relationship
**Scenarios leverage resources** to create business applications through orchestration. Resources provide the raw capabilities, scenarios provide the business logic.

## Common Commands

```bash
# System Management
vrooli setup                    # Initial setup
vrooli develop                  # Start development environment
vrooli build                    # Build the system  
vrooli status                   # Show system health
vrooli stop                     # Stop all components

# Resource Management
vrooli resource list            # List available resources
vrooli resource start-all       # Start all enabled resources
vrooli resource status          # Show resource health
vrooli resource stop-all        # Stop all resources
resource-<name> start           # Start specific resource
resource-<name> logs            # View resource logs

# Scenario Operations
vrooli scenario list            # List available scenarios
vrooli scenario run <name>      # Run a scenario
vrooli scenario test <name>     # Test scenario integration
```

## Learning Path (Choose Your Track)

### 🚀 **Quick User** - Run Existing Scenarios
**Perfect if**: You want to use Vrooli scenarios immediately
**Time**: 5 minutes
```bash
# You're done! Continue with:
vrooli scenario list                    # Explore available scenarios
vrooli scenario run <name>              # Try different scenarios
```

### 🏗️ **Scenario Creator** - Build Business Applications  
**Perfect if**: You want to create new scenarios for business value
**Time**: 30-45 minutes
**Next**: [Comprehensive Scenario Guide](scenarios/getting-started.md)

### ⚙️ **Platform Developer** - Contribute to Vrooli Core
**Perfect if**: You want to develop resources, core features, or platform improvements
**Time**: 60+ minutes  
**Next**: [Development Environment Setup](devops/development-environment.md)

## Quick Reference

### Core Commands
```bash
# System Management
vrooli setup                    # Initial setup
vrooli develop                  # Start development environment
vrooli build                    # Build the system  
vrooli status                   # Show system health
vrooli stop                     # Stop all components

# Resource Management  
vrooli resource start-all       # Start all enabled resources
vrooli resource status          # Show resource health
resource-<name> start           # Start specific resource

# Scenario Operations
vrooli scenario list            # List available scenarios
vrooli scenario run <name>      # Run a scenario
vrooli scenario test <name>     # Test scenario integration
```

### Get Help
- **Issues with scenarios**: [Scenario Troubleshooting](devops/troubleshooting.md#scenario-issues)
- **Resource problems**: [Resource Troubleshooting](devops/troubleshooting.md#resource-issues)  
- **Contributing**: [Contributing Guide](CONTRIBUTING.md)

## Success Checklist

You're ready to build with Vrooli when you can:
- [ ] Start and stop resources reliably
- [ ] Run existing scenarios successfully  
- [ ] Create a basic scenario from template
- [ ] Understand the resource + scenario architecture
- [ ] Know where to find detailed documentation

**Welcome to Vrooli!** You're now ready to orchestrate resources into profitable business applications.