# lib/ - Infrastructure Utilities

This directory contains **infrastructure utilities** that are framework-agnostic and reusable across the application.

## What belongs here

- **API client** (`api/`) - HTTP request functions and API type definitions
- **Browser APIs** (`browser.ts`) - Clipboard, downloads, and other browser interactions
- **General utilities** (`utils.ts`, `cn.ts`) - Pure utility functions with no business logic
- **Error handling** (`error-utils.ts`) - Error parsing and formatting
- **URL/API resolution** (`api-base.ts`, `pipeline-utils.ts`) - URL construction and resolution

## What does NOT belong here

- **Business logic** - Use `services/` instead
- **React hooks** - Use `hooks/` instead
- **Domain types** - Use `domain/` instead
- **State management** - Use `store/` instead

## Key Principle

Code in `lib/` should be:

1. Pure functions with no side effects (except for necessary I/O like fetch)
2. Framework-agnostic (no React imports)
3. Reusable across different features
4. Well-tested in isolation
