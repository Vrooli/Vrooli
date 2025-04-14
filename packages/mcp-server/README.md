# Vrooli MCP Server

A Model Context Protocol (MCP) server for the Vrooli project that provides tools and resources to LLMs in Cursor and other MCP clients.

## What is MCP?

Model Context Protocol (MCP) is an open protocol developed by Anthropic that standardizes how applications provide context to LLMs. Think of MCP like a USB-C port for AI applications - a standardized way to connect AI models to different data sources and tools.

## Features

- **Project Information**: Get basic or detailed information about the Vrooli project
- **Search Functionality**: Search for routines, users, and teams in the project
- **Project Documentation**: Access project README and other documentation
- **File Structure**: Get a hierarchical view of the project file structure

## Getting Started

### Prerequisites

- Node.js 16+
- Yarn

### Installation

```bash
# From the project root
cd packages/mcp-server
npm install
```

### Running the Server

```bash
# Development mode with auto-reload
npm run dev

# Production mode
npm start
```

The server will start on port 3100 by default. You can change this by setting the `MCP_SERVER_PORT` environment variable.

## Connecting to the Server

### In Cursor IDE

1. Open the Command Palette (Ctrl+Shift+P or Cmd+Shift+P)
2. Type "Connect to MCP Server" and select it
3. Enter `ws://localhost:3100` as the server URL
4. The Vrooli MCP Server should now be available to the LLM in Cursor

## Available Tools

- `project_info`: Get information about the Vrooli project
- `search_routines`: Search for routines, users, and teams in the project

## Available Resources

- `project_readme`: The README file for the Vrooli project
- `project_structure`: The directory structure of the Vrooli project

## Development

### Running Tests

```bash
npm test
```

### Adding New Tools

To add a new tool, create a new file in the `src/tools` directory and export an object with the following structure:

```javascript
export const myTool = {
  id: 'my_tool',
  name: 'My Tool',
  description: 'Description of my tool',
  parameters: {
    type: 'object',
    properties: {
      // Parameter definitions
    }
  },
  handler: async (params) => {
    // Tool implementation
    return result;
  }
};
```

Then register it in `src/index.js` by adding it to the `tools` array.

### Adding New Resources

To add a new resource, create a new file in the `src/resources` directory and export an object with the following structure:

```javascript
export const myResource = {
  id: 'my_resource',
  name: 'My Resource',
  description: 'Description of my resource',
  resource_type: 'text',
  parameters: {
    type: 'object',
    properties: {
      // Parameter definitions (optional)
    }
  },
  handler: async (params) => {
    // Resource implementation
    return {
      content: 'Resource content',
      metadata: {
        // Resource metadata
      }
    };
  }
};
```

Then register it in `src/index.js` by adding it to the `resources` array.

## License

This project is licensed under the AGPL-3.0 License - see the LICENSE file for details. 