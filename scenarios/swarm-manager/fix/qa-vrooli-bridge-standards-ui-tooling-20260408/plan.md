# Implementation Plan: Resolve vrooli-bridge standards — UI tooling and deps

## Required Reading

- `prompt-manager skill read scientific-debugging` — hypothesis-driven root cause analysis for fix items
- `prompt-manager skill read react-coherence` — UI architecture patterns and standards alignment
- `prompt-manager skill read interoperability-steer` — cross-scenario dependency and integration patterns

## Problem Statement

The GCT standards review (job `52410c6c`) reports 3 blocking violations and 7 warnings for vrooli-bridge. All relate to UI tooling and dependency configuration:

**Blocking (3):**
1. Missing `@vrooli/api-base` in `ui/package.json`
2. Missing `@vrooli/iframe-bridge` in `ui/package.json`
3. (Likely) Missing tsconfig or eslint config counted as blocking

**Warnings (7):**
- `ui/tsconfig.json` not found
- `ui/eslint.config.js` not found
- `npm install` in service.json instead of `pnpm`
- `package-lock.json` present (npm artifact)
- Unexpected `node_modules` presence (currently absent on disk, may be intermittent)
- Additional standards warnings TBD from full GCT output

### Critical Context: Vanilla JS UI

The vrooli-bridge UI is **vanilla JavaScript** — plain `app.js`, `index.html`, `styles.css` with a copy-based build (`cp -R src/* dist/`). It does **not** use React, TypeScript, or any bundler. This fundamentally affects the approach:

- Adding `@vrooli/api-base` and `@vrooli/iframe-bridge` requires deciding whether to keep vanilla JS or migrate to TypeScript/React
- tsconfig.json and eslint.config.js presume a TS/React stack — applying them to vanilla JS needs adaptation
- The reference configs from swarm-manager assume React + Vite + TypeScript

## Scope

### acceptance_allow
```
scenarios/vrooli-bridge/**
```

### Goals
- Reduce standards blockingViolations to 0
- Add required UI tooling config files
- Switch from npm to pnpm in service.json lifecycle
- Clean up npm artifacts (package-lock.json)

### Non-Goals
- Full React/TypeScript migration of the UI (unless chosen in workshop)
- API changes
- Functional feature additions

## Approach

<!-- TBD — depends on workshop decision d1 (vanilla JS vs migration) -->

### Option A: Minimal Compliance (Vanilla JS stays)
1. Add `@vrooli/api-base` and `@vrooli/iframe-bridge` as dependencies in `package.json`
2. Create a minimal `tsconfig.json` with `allowJs: true` for type-checking JS files
3. Create a minimal `eslint.config.js` adapted for JS (no TS parser needed for pure JS)
4. Update `service.json` to use `pnpm install` instead of `npm install`
5. Remove `package-lock.json`, add to `.gitignore` if needed

### Option B: TypeScript Migration
1. Rename `app.js` → `app.ts`, add type annotations
2. Add full tsconfig.json matching swarm-manager reference
3. Add full eslint.config.js matching swarm-manager reference
4. Add Vite or esbuild as bundler
5. Add `@vrooli/api-base` and `@vrooli/iframe-bridge` with proper imports
6. Update build script from `cp` to bundler
7. Switch to pnpm

## Phases

<!-- TBD — phasing depends on approach decision -->

## Testing & Verification

- Run `git-control-tower review-run vrooli-bridge --json` after changes
- Verify `blockingViolations = 0`
- Verify `warnings` reduced (target: ≤ 2)
- Verify `make start` still works for vrooli-bridge
- Verify UI still loads and functions correctly

## Risks

1. **Vanilla JS + TypeScript tooling mismatch**: tsconfig/eslint configs designed for TS/React may not apply cleanly to vanilla JS
2. **@vrooli packages may require TS**: If these packages export only TS types, they may not be usable from vanilla JS without a build step
3. **Build system change**: If migration is chosen, the `cp -R` build must be replaced, which could break the scenario lifecycle
4. **pnpm workspace interaction**: The `--ignore-workspace` flag may be needed to avoid monorepo workspace resolution issues

## Open Questions

- What do `@vrooli/api-base` and `@vrooli/iframe-bridge` actually export? Are they usable from vanilla JS?
- Does the GCT standards checker have a "vanilla JS" mode or does it always expect TS/React?
- Is there an existing pattern for non-React scenario UIs passing standards?
