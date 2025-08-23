# LiteLLM Examples

This directory contains practical examples of using LiteLLM with Vrooli.

## Examples Overview

### 1. Basic Chat Example
**File**: `basic-chat.sh`
- Simple chat completion using LiteLLM proxy
- Demonstrates API authentication and model selection
- Shows error handling and response parsing

### 2. Multi-Provider Configuration
**File**: `multi-provider-config.yaml`
- Complete configuration with multiple AI providers
- Demonstrates routing strategies and fallback logic
- Includes budget controls and rate limiting

### 3. Claude Code Integration
**File**: `claude-code-setup.sh`
- Script to configure Claude Code with LiteLLM
- Environment variable setup
- Testing and validation

### 4. Cost-Optimized Routing
**File**: `cost-routing-config.yaml`
- Configuration optimized for cost-effective model routing
- Demonstrates routing cheap models first with expensive fallbacks
- Budget management settings

### 5. N8n Workflow Integration
**File**: `n8n-workflow.json`
- N8n workflow that uses LiteLLM for AI processing
- Shows HTTP request configuration
- Error handling and response processing

## Quick Start

1. **Install LiteLLM**:
   ```bash
   resource-litellm install --verbose
   ```

2. **Run Basic Example**:
   ```bash
   ./examples/basic-chat.sh
   ```

3. **Set up Claude Code Integration**:
   ```bash
   ./examples/claude-code-setup.sh
   ```

4. **Import Custom Configuration**:
   ```bash
   resource-litellm content add --type config --file examples/multi-provider-config.yaml --name multi-provider
   resource-litellm content execute --type config --name multi-provider
   ```

## Example Configurations

Each example includes:
- **Purpose**: What the example demonstrates
- **Prerequisites**: Required API keys and setup
- **Usage**: How to run the example
- **Expected Output**: What to expect
- **Troubleshooting**: Common issues and solutions

## Contributing Examples

To add new examples:
1. Create appropriately named files
2. Include clear documentation
3. Test with fresh LiteLLM installation
4. Update this README with example description