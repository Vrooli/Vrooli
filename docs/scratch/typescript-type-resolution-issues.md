# TypeScript Type Resolution Issues in @vrooli/shared

## Problem
Intermittent type resolution failures when importing from `@vrooli/shared` package. Types work temporarily after rebuild but break again randomly.

## Root Causes

### 1. **TypeScript Language Service Cache**
- VSCode's TypeScript server caches module resolution results
- When files change but the cache isn't invalidated, it serves stale type information
- This is why restarting VSCode often temporarily fixes the issue

### 2. **Missing/Incomplete Declaration Files**
- The build process uses SWC for JavaScript compilation and TypeScript only for declarations
- If the TypeScript declaration step fails silently, you get JS files without .d.ts files
- TypeScript can't find types without the .d.ts files

### 3. **Incremental Compilation Issues**
- TypeScript's `incremental: true` setting creates `.tsbuildinfo` files
- These can become corrupted or out of sync
- Causes partial or failed declaration generation

### 4. **Module Resolution Timing**
- In monorepos, TypeScript might try to resolve types before they're built
- Race condition between build processes and type checking

## Solutions

### Immediate Fixes
```bash
# 1. Clean and rebuild shared package
cd packages/shared
rm -rf dist .tsbuildinfo
pnpm run build

# 2. Restart TypeScript server in VSCode
# Cmd/Ctrl + Shift + P → "TypeScript: Restart TS Server"

# 3. Clear all caches
rm -rf node_modules/.cache
rm -rf packages/*/.tsbuildinfo
```

### Long-term Solutions

1. **Add a watch script for development**
```json
// In packages/shared/package.json
"scripts": {
  "dev": "tsc --watch --emitDeclarationOnly --outDir dist"
}
```

2. **Use project references properly**
```json
// In packages/server/tsconfig.json
{
  "references": [
    { "path": "../shared" }
  ]
}
```

3. **Add a postinstall script**
```json
// In root package.json
"scripts": {
  "postinstall": "pnpm -r run build"
}
```

4. **Consider using a build tool that handles this better**
- Turborepo
- Nx
- Rush

## Verification Commands
```bash
# Check if declarations exist
ls packages/shared/dist/**/*.d.ts

# Verify TypeScript can resolve types
cd packages/server
npx tsc --noEmit --listFiles | grep shared

# Check for build errors
cd packages/shared
npx tsc --noEmit
```

## Current Status
The issue is related to TypeScript's module resolution cache and incremental compilation. The types ARE being generated correctly, but TypeScript's language service doesn't always pick them up.