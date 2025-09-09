# Scenario to MCP

**Transform Vrooli scenarios into AI-accessible tools via Model Context Protocol**

## 🎯 What is This?

Scenario to MCP adds Model Context Protocol (MCP) support to any Vrooli scenario, making them discoverable and callable by external AI agents like Claude, ChatGPT, and other MCP-compatible tools. This transforms isolated scenarios into interconnected AI tools that can be composed by any MCP client.

## 🚀 Quick Start

```bash
# Setup the scenario
cd scenarios/scenario-to-mcp
../../scripts/manage.sh setup --yes yes

# Start the services
vrooli scenario run scenario-to-mcp

# Use the CLI
scenario-to-mcp list                    # See all scenarios
scenario-to-mcp add my-scenario --auto  # Add MCP to a scenario
scenario-to-mcp registry                # View MCP registry
```

## 🎨 Dashboard

Access the visual dashboard at `http://localhost:3291` to:
- View all scenarios and their MCP status
- One-click MCP addition with Claude-code agent spawning
- Monitor MCP endpoint health
- Test MCP tools directly from the UI

## 🛠️ CLI Commands

### List Scenarios
```bash
scenario-to-mcp list [--filter <status>] [--json]
```
Shows all scenarios with their MCP status, confidence level, and allocated ports.

### Add MCP Support
```bash
scenario-to-mcp add <scenario> [--auto] [--template <name>]
```
Spawns a Claude-code agent to add MCP support to a scenario.

### Test MCP Endpoint
```bash
scenario-to-mcp test <scenario> [--tool <name>]
```
Tests MCP endpoint functionality and specific tools.

### View Registry
```bash
scenario-to-mcp registry [--format json|yaml|table]
```
Shows the MCP service discovery registry.

### Find Candidates
```bash
scenario-to-mcp candidates
```
Lists scenarios that are good candidates for MCP addition, ranked by score.

## 🏗️ Architecture

### Components

1. **MCP Detector** (`lib/detector.js`)
   - Scans scenarios for existing MCP implementations
   - Analyzes confidence levels (high/medium/low)
   - Identifies MCP candidates

2. **Code Generator** (`lib/code-generator.js`)
   - Generates MCP server implementations
   - Creates tool manifests from APIs/CLIs
   - Produces handler code for all capabilities

3. **React Dashboard** (`ui/`)
   - Grid view of all scenarios
   - Real-time MCP status monitoring
   - Agent spawning interface

4. **Go API Server** (`api/`)
   - Manages MCP registry
   - Handles agent sessions
   - Provides health monitoring

5. **CLI** (`cli/scenario-to-mcp`)
   - Full command-line interface
   - Direct integration with detector and generator
   - Registry management

### MCP Implementation Pattern

When MCP is added to a scenario, it creates:
```
scenarios/<target-scenario>/
├── mcp/
│   ├── server.js       # MCP server exposing capabilities
│   ├── manifest.json   # Tool definitions and schema
│   ├── handlers/       # Modular tool implementations
│   └── README.md       # Usage documentation
```

### Port Allocation

- Each scenario gets a unique MCP port (4000-5000 range)
- Ports are allocated based on scenario name hash
- Registry tracks all active endpoints

## 🔄 How It Works

1. **Detection Phase**
   - Scans all scenarios in `/scenarios`
   - Checks for existing `mcp/` directories
   - Analyzes APIs and CLIs for exposable capabilities

2. **Generation Phase**
   - Claude-code agent analyzes scenario structure
   - Generates MCP server using templates
   - Creates tool handlers for APIs and CLIs
   - Sets up manifest with proper schemas

3. **Registration Phase**
   - MCP server starts on allocated port
   - Registers with central registry
   - Becomes discoverable by AI agents

4. **Usage Phase**
   - External AI agents discover via registry
   - Connect to MCP endpoints
   - Execute tools with proper parameters
   - Receive structured responses

## 📊 Database Schema

The PostgreSQL schema (`initialization/storage/postgres/schema.sql`) includes:
- `mcp.endpoints` - Tracks all MCP-enabled scenarios
- `mcp.tools` - Cached tool definitions from manifests
- `mcp.tool_usage` - Analytics on tool invocations
- `mcp.agent_sessions` - Claude-code agent sessions
- `mcp.registry` - View for service discovery

## 🎯 Value Proposition

### For Developers
- One-click MCP addition to any scenario
- Automatic tool generation from existing code
- No manual MCP implementation needed

### For AI Agents
- Discover all Vrooli capabilities dynamically
- Execute scenario functions via standard protocol
- Chain multiple scenarios for complex tasks

### For Vrooli
- Every scenario becomes a reusable AI tool
- Exponential value multiplication
- Standard interoperability with AI ecosystem

## 🔗 Integration with Claude

To use MCP-enabled scenarios with Claude, add to your MCP configuration:

```json
{
  "mcpServers": {
    "vrooli-registry": {
      "command": "curl",
      "args": ["http://localhost:3292/api/v1/mcp/registry"]
    }
  }
}
```

Individual scenarios can also be added directly:
```json
{
  "mcpServers": {
    "my-scenario": {
      "command": "node",
      "args": ["/path/to/scenarios/my-scenario/mcp/server.js"]
    }
  }
}
```

## 🧪 Testing

```bash
# Run all tests
vrooli scenario test scenario-to-mcp

# Test MCP detection
node lib/detector.js scan

# Test code generation
node lib/code-generator.js generate test-scenario

# Test specific MCP endpoint
scenario-to-mcp test my-scenario --tool api_get_status
```

## 🚨 Troubleshooting

### MCP Server Won't Start
- Check port allocation: `scenario-to-mcp list`
- Verify scenario has API or CLI: `scenario-to-mcp candidates`
- Check logs: `docker logs scenario-to-mcp-api`

### Registry Not Updating
- Ensure PostgreSQL is running: `vrooli resource postgres status`
- Check API health: `curl http://localhost:3290/api/v1/health`
- Verify registry service: `curl http://localhost:3292/api/v1/mcp/registry`

### Claude Can't Connect
- Ensure MCP server is running: `scenario-to-mcp status`
- Check firewall/network settings
- Verify MCP configuration path

## 📚 Resources

- [Model Context Protocol Spec](https://github.com/anthropics/model-context-protocol)
- [MCP SDK Documentation](https://github.com/modelcontextprotocol/sdk)
- [Vrooli Scenarios Guide](../../docs/scenarios/README.md)

## 🎨 UI Style

The dashboard follows a technical, Matrix-inspired aesthetic:
- Dark background with green accent colors
- Monospace typography throughout
- Subtle animations and glow effects
- Clean grid layout for scenario cards
- Real-time status indicators

---

**Remember**: Every scenario you make MCP-enabled becomes a permanent tool in Vrooli's intelligence arsenal, accessible by any AI agent forever. This is how Vrooli achieves compound intelligence growth.