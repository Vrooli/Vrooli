# @vrooli/api-base Documentation Hub

Welcome to the comprehensive documentation for `@vrooli/api-base` - the universal API connectivity solution for Vrooli scenarios.

## 📚 Documentation Structure

This documentation follows a **hub-and-spokes** model, organized by purpose:

### 🚀 [Guides](./guides/) - Learn By Doing
Step-by-step instructions for common tasks:
- **[Quick Start](./guides/quick-start.md)** - Get running in 5 minutes
- **[Client Usage](./api/client.md)** - Using the client-side API in your UI
- **[Server Usage](./api/server.md)** - Setting up your scenario server
- **[Proxy Setup](./guides/host-scenario-pattern.md)** - Hosting embedded scenarios
- **[Testing Guide](../TESTING.md)** - Testing scenarios in all contexts
- **[Migration Guide](./guides/migration.md)** - Migrating from existing patterns

### 💡 [Concepts](./concepts/) - Understanding How It Works
Deep dives into core ideas:
- **[Deployment Contexts](./concepts/deployment-contexts.md)** - The three deployment environments
- **[Proxy Resolution](./concepts/proxy-resolution.md)** - How API URLs are resolved
- **[Runtime Configuration](./concepts/runtime-config.md)** - Dynamic configuration in production
- **[WebSocket Support](./concepts/websocket-support.md)** - WS/WSS endpoint resolution

### 📖 [API Reference](./api/) - Complete Technical Details
Comprehensive function documentation:
- **[Client API](./api/client.md)** - resolveApiBase, buildApiUrl, etc.
- **[Server API](./api/server.md)** - createScenarioServer, proxyToApi, etc.
- **[Types](./api/types.md)** - TypeScript interfaces and types

### 💻 [Examples](./examples/) - Real-World Use Cases
Complete, working examples:
- **[Vite Configuration](./examples/vite-config.md)** - Required Vite setup for universal deployment
- **[Basic Scenario](./examples/basic-scenario.md)** - Standard scenario setup
- **[Embedded Scenario](./examples/embedded-scenario.md)** - Being hosted by another app
- **[Custom Proxy](./examples/custom-proxy.md)** - Creating your own proxy patterns
- **[Advanced Patterns](./examples/advanced-patterns.md)** - Complex use cases

## 🎯 Quick Navigation

### I want to...

**Get started quickly**
→ [Quick Start Guide](./guides/quick-start.md)

**Understand why this package exists**
→ [Deployment Contexts](./concepts/deployment-contexts.md)

**Set up my scenario's UI**
→ [Client Usage Guide](./api/client.md)

**Set up my scenario's server**
→ [Server Usage Guide](./api/server.md)

**Migrate from old patterns**
→ [Migration Guide](./guides/migration.md)

**Host another scenario in an iframe**
→ [Proxy Setup Guide](./guides/host-scenario-pattern.md)

**Look up a specific function**
→ [Client API Reference](./api/client.md) or [Server API Reference](./api/server.md)

**See complete examples**
→ [Examples Directory](./examples/)

**Test my scenario in all contexts**
→ [Testing Guide](../TESTING.md)

## 🔑 Key Concepts

### The Problem

Vrooli scenarios must work in three different deployment contexts:

1. **Localhost** - Development environment (`http://localhost:3000`)
2. **Direct Tunnel** - Standalone deployment (`https://my-scenario.example.com`)
3. **Proxied/Embedded** - Inside another scenario (`https://host.com/apps/my-scenario/proxy/`)

Each context requires different API URL resolution strategies. Writing this logic manually for every scenario is error-prone and inconsistent.

### The Solution

`@vrooli/api-base` provides:
- **Automatic detection** of deployment context
- **Universal resolution** that works everywhere
- **Zero configuration** for standard cases
- **Full customization** when needed
- **Server utilities** for hosting and proxying

## 📦 Package Structure

```
@vrooli/api-base
├── /            # Client-side exports (resolveApiBase, buildApiUrl, etc.)
├── /server      # Server-side exports (createScenarioServer, proxyToApi, etc.)
└── /types       # TypeScript type definitions
```

## 🌟 Features at a Glance

- ✅ **Universal** - Works with any domain, any proxy pattern
- ✅ **Smart Resolution** - Automatically detects deployment context
- ✅ **WebSocket Support** - First-class WS/WSS support
- ✅ **Runtime Config** - Dynamic configuration for production bundles
- ✅ **Server Template** - Complete Express server setup
- ✅ **Zero Dependencies** - No runtime dependencies
- ✅ **Fully Tested** - 166+ unit tests
- ✅ **TypeScript** - Complete type safety
- ✅ **Backwards Compatible** - Supports existing conventions

## 🚦 Getting Started

```typescript
// Client (UI)
import { resolveApiBase } from '@vrooli/api-base'

const API_BASE = resolveApiBase({ appendSuffix: true })
fetch(`${API_BASE}/health`)
```

```typescript
// Server (Express)
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist'
})

app.listen(process.env.UI_PORT)
```

**That's it!** Your scenario now works in all three deployment contexts automatically.

## 📚 Learn More

- **New to @vrooli/api-base?** → Start with the [Quick Start Guide](./guides/quick-start.md)
- **Migrating existing code?** → Read the [Migration Guide](./guides/migration.md)
- **Need complete reference?** → Check the [API Documentation](./api/)
- **Want to understand deeply?** → Explore the [Concepts](./concepts/)

## 🤝 Contributing

Found a bug or have a suggestion? The source code is in `packages/api-base/src/`.

See [IMPLEMENTATION_PLAN.md](../IMPLEMENTATION_PLAN.md) for architecture details.

## 📄 License

MIT
