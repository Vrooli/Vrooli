# Vite Configuration Example

This guide shows the recommended Vite configuration for Vrooli scenarios using `@vrooli/api-base`.

## Complete Example

```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  // ✅ CRITICAL: Use relative paths for universal deployment
  base: './',

  plugins: [react()],

  // Development server (only used in rare dev cases)
  server: {
    port: 3000,
    // Usually not needed - scenarios use production bundles
  },

  // Build configuration
  build: {
    outDir: 'dist',
    // Vite automatically handles asset hashing and optimization
  },
})
```

## Why `base: './'` is Required

### The Problem

By default, Vite builds with **absolute paths**:

```html
<!-- ❌ Default Vite output -->
<script type="module" src="/assets/index-abc123.js"></script>
<link rel="stylesheet" href="/assets/index-xyz789.css">
```

These paths work on localhost (`http://localhost:3000/assets/...`) but **fail in other contexts**:

- **Direct Tunnel**: `https://my-scenario.itsagitime.com/assets/...` ✅ Works
- **Proxied**: `https://app-monitor.itsagitime.com/apps/my-scenario/proxy/assets/...` ❌ Fails (tries to load from `/assets/` instead of `/apps/my-scenario/proxy/assets/`)

### The Solution

Setting `base: './'` generates **relative paths**:

```html
<!-- ✅ With base: './' -->
<script type="module" src="./assets/index-abc123.js"></script>
<link rel="stylesheet" href="./assets/index-xyz789.css">
```

Relative paths work everywhere:
- **Localhost**: `http://localhost:3000/./assets/...` → `http://localhost:3000/assets/...` ✅
- **Direct Tunnel**: `https://my-scenario.itsagitime.com/./assets/...` → `https://my-scenario.itsagitime.com/assets/...` ✅
- **Proxied**: `https://host.com/apps/scenario/proxy/./assets/...` → `https://host.com/apps/scenario/proxy/assets/...` ✅

## Common Scenarios

### React with TypeScript

```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  base: './',
  plugins: [react()],
  resolve: {
    alias: {
      '@': '/src',
    },
  },
})
```

### With Path Aliases

```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  base: './',
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@components': path.resolve(__dirname, './src/components'),
      '@utils': path.resolve(__dirname, './src/utils'),
    },
  },
})
```

### With Environment Variables

```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  base: './',
  plugins: [react()],
  define: {
    // Make env vars available to your app
    __APP_VERSION__: JSON.stringify(process.env.npm_package_version),
  },
})
```

### With API Proxy (Dev Mode Only)

**Note**: In production, Vrooli scenarios serve production bundles and use the server proxy. This dev proxy is rarely needed.

```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  base: './',
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.API_PORT || 8080}`,
        changeOrigin: true,
      },
    },
  },
})
```

## Testing Your Build

After building, verify the HTML uses relative paths:

```bash
# Build
pnpm build

# Check the output
cat dist/index.html
```

**Expected output:**

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>My App</title>
    <!-- ✅ Relative paths -->
    <script type="module" crossorigin src="./assets/index-abc123.js"></script>
    <link rel="stylesheet" crossorigin href="./assets/index-xyz789.css">
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
```

**If you see absolute paths (`/assets/...`)**, you forgot `base: './'`!

## Troubleshooting

### Assets still use absolute paths after adding `base: './'`

**Solution**: Rebuild your app:

```bash
cd ui
rm -rf dist
pnpm build
```

### Images/fonts not loading

**Problem**: Asset imports using absolute paths

**Solution**: Use relative imports or Vite's import syntax:

```typescript
// ❌ Don't use absolute paths
import logo from '/assets/logo.png'

// ✅ Use relative imports
import logo from './assets/logo.png'
import logo from '@/assets/logo.png'  // with alias

// ✅ Or use Vite's public directory
// Put files in public/, reference as /logo.png
// Vite handles these correctly
```

### Different base path for different environments

**Not recommended**, but if you need it:

```typescript
// vite.config.ts
import { defineConfig } from 'vite'

export default defineConfig({
  base: process.env.NODE_ENV === 'production' ? './' : '/',
  // ... rest of config
})
```

For Vrooli scenarios, **always use `base: './'`** for consistency.

## Related Documentation

- [Quick Start Guide](../guides/quick-start.md) - Complete setup walkthrough
- [Deployment Contexts](../concepts/deployment-contexts.md) - Understanding localhost/tunnel/proxy
- [Server API](../api/server.md) - Server configuration options

## Summary

**One line change for universal deployment:**

```typescript
export default defineConfig({
  base: './',  // ← Add this
  // ... rest of config
})
```

That's it! Your scenario now works everywhere. 🎉
