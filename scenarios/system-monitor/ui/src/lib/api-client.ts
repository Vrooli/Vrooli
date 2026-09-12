import { resolveApiBase, buildApiUrl as composeApiUrl } from '@vrooli/api-base';

// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Centralized API URL resolution            ║
// ║                                                              ║
// ║  resolveApiBase() auto-detects the correct API endpoint      ║
// ║  for localhost, tunnel, and proxy contexts.                  ║
// ║  buildApiUrl() normalizes paths against that base.           ║
// ║                                                              ║
// ║  DO NOT hardcode localhost URLs or construct API URLs         ║
// ║  manually anywhere else in the codebase.                     ║
// ╚══════════════════════════════════════════════════════════════╝

const API_BASE = resolveApiBase({ appendSuffix: true });
const API_ROOT = resolveApiBase();

/**
 * Build a full API URL from a path segment.
 * All API calls in the UI must use this function.
 */
export function buildUrl(path: string): string {
  return composeApiUrl(path, { baseUrl: API_BASE });
}

export function buildRootUrl(path: string): string {
  return composeApiUrl(path, { baseUrl: API_ROOT });
}
