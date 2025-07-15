# UI Build Performance Optimization Task

## Problem Statement
The UI build process is extremely slow, taking 5-10 minutes to complete. This impacts developer productivity and CI/CD pipeline efficiency.

## Current Metrics
- **Build time**: 5-10 minutes
- **Modules processed**: 4444
- **Warnings**: 
  - Large chunks (>1000kb) 
  - Dynamic import conflicts
  - Node.js module externalization issues

## Root Causes
1. **Large number of dependencies** being processed (4444 modules)
2. **Inefficient chunk splitting** - some chunks exceed 1MB
3. **Mixed import patterns** - both dynamic and static imports for same modules
4. **No build caching** configured

## Suggested Solutions

### 1. Optimize Vite Configuration
- Implement manual chunk splitting strategy
- Configure better tree shaking
- Enable build caching

### 2. Reduce Dependencies
- Audit and remove unused packages
- Use lighter alternatives where possible
- Lazy load heavy libraries

### 3. Fix Import Patterns
- Choose either static or dynamic imports consistently
- Prefer dynamic imports for large features
- Fix the following files with mixed imports:
  - `moment-timezone/index.js`
  - `ChatCrud.tsx`
  - `BotUpsert.tsx`
  - `ProView.tsx`

### 4. Configure Build Optimizations
```javascript
// vite.config.js suggestions
export default {
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor': ['react', 'react-dom'],
          'mui': ['@mui/material'],
          // ... other strategic chunks
        }
      }
    },
    // Increase chunk size warning limit temporarily
    chunkSizeWarningLimit: 2000
  }
}
```

## Implementation Steps
1. Run dependency analysis: `pnpm list --depth=0`
2. Identify unused dependencies
3. Update Vite configuration
4. Fix mixed import patterns
5. Implement build caching
6. Measure improvement

## Success Criteria
- Build time reduced to under 3 minutes
- No chunk size warnings
- Clean build output without externalization warnings

## Priority
Medium-High (impacts developer productivity significantly)