# Vrooli MCP Server

A Model Context Protocol (MCP) server for the Vrooli project that provides tools and resources to LLMs in Cursor and other MCP clients.

## What is MCP?

Model Context Protocol (MCP) is an open protocol, originally developed by Anthropic and now community-maintained, that standardizes how applications provide context to LLMs. Think of MCP like a USB-C port for AI applications - a standardized way to connect AI models to different data sources and tools.

## Features (Example)

*(Update this section with actual implemented tools/resources)*

- **Example Resource**: Provides a basic example resource.

## Getting Started

### Prerequisites

- Node.js 18+ (or version supporting top-level await)
- npm (comes with Node.js)

### Installation

```bash
# From the project root
cd packages/mcp-server
npm install
```

### Running the Server

This server can run in two modes:

-   **SSE (Server-Sent Events) Mode (Default):** Runs an HTTP server on port 3100, communicating via SSE. This is typically used for remote connections (e.g., connecting from Cursor manually).
-   **STDIO Mode:** Uses standard input/output for communication, typically launched and managed by a local client application using the MCP SDK.

You can select the mode using the `--mode` flag. If no flag is provided, it defaults to **SSE mode**.

```bash
# Run in SSE mode (default)
npx ts-node src/index.ts
# OR explicitly:
npx ts-node src/index.ts --mode=sse

# Run in STDIO mode
npx ts-node src/index.ts --mode=stdio
```

When running in SSE mode, the server will listen on `http://localhost:3100`.

## Connecting to the Server

### In Cursor IDE (SSE Mode)

1.  Ensure the server is running in SSE mode (`npx ts-node src/index.ts --mode=sse`).
2.  Open the Command Palette (Ctrl+Shift+P or Cmd+Shift+P).
3.  Type "Connect to MCP Server" and select it.
4.  Enter `http://localhost:3100/sse` as the server URL (use `http`, not `ws`).
5.  The Vrooli MCP Server should now be available to the LLM in Cursor.

### STDIO Mode

Connections in STDIO mode are typically handled automatically by the client application that launches the server process. Manual connection is usually not required.

## Available Tools & Resources

*(Update this section based on implementation)*

Currently, the server provides basic example resources/tools depending on the mode.

-   **SSE Mode:** The `initialize` response currently sends an empty `offerings` array. Tools/resources need to be added to the POST `/` handler.
-   **STDIO Mode:** Includes an example resource (`example://resource-stdio`).

## Development

### Running Tests

*(Assumes test setup exists)*

```bash
npm test
```

### Adding New Tools (Example for STDIO Mode)

To add a new tool (example structure, adapt as needed), you might create a definition and register it within the `index.ts` file inside the `if (mode === 'stdio')` block, potentially adjusting the `sdkCapabilities` and using the SDK's methods if available, or by populating the `offerings` array in the `initialize` response for SSE mode.

```typescript
// Example Tool Definition (Conceptual)
const myToolDefinition = {
  // MCP Tool Schema fields
  // ...
};

// Registration might involve:
// 1. Adding to offerings in initialize response (SSE)
// 2. Using an SDK method like server.addTool() (STDIO, if available)
```

### Adding New Resources (Example for STDIO Mode)

Similar to tools, resource definitions follow the MCP specification. Register them in the appropriate mode's logic (`offerings` array for SSE, potentially using SDK methods for STDIO).

```typescript
// Example Resource Definition (Conceptual)
const myResourceDefinition = {
  // MCP Resource Schema fields
  uri: "vrooli://my-resource",
  name: "My Vrooli Resource",
  // ...
};

// Handler for reading the resource
mcpServer.setRequestHandler(ReadResourceRequestSchema, async (request) => {
  if (request.params.uri === myResourceDefinition.uri) {
    // Logic to fetch and return resource content
    return { contents: [{ uri: request.params.uri, text: "Resource content" }] };
  }
  // Handle other resource URIs or throw error
});
```

Refer to the [MCP Specification](https://spec.modelcontextprotocol.io/) for details on tool and resource schemas.

## License

This project is licensed under the AGPL-3.0 License - see the LICENSE file for details. 