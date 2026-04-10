/**
 * Shared base utilities for all API modules.
 *
 * Provides the API base URL constants, a type-safe JSON response helper,
 * the ApiErrorBody shape, and the resolveAttachmentUrl utility.
 */
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

export const API_BASE = resolveApiBase({ appendSuffix: true });

// Base URL without the /api/v1 suffix for resolving attachment paths
const ORIGIN_BASE = resolveApiBase({ appendSuffix: false });

export { buildApiUrl };

/**
 * Type-safe wrapper around Response.json().
 * Casts the untyped Promise<any> from fetch to the expected type,
 * eliminating @typescript-eslint/no-unsafe-return warnings at each call-site.
 */
export function jsonResponse<T>(res: Response): Promise<T> {
  return res.json() as Promise<T>;
}

/** Shape of error bodies returned by the API (used in error-handling blocks). */
export interface ApiErrorBody {
  error?: {
    message?: string;
    code?: string;
    recovery?: string;
    details?: { user_message?: string };
  };
}

/**
 * Resolve an attachment URL to work in proxy contexts.
 * The API returns paths like "/api/v1/uploads/..." which need to be
 * resolved relative to the current origin/proxy base.
 */
export function resolveAttachmentUrl(url: string | undefined): string | undefined {
  if (!url) return undefined;
  // If already absolute URL, return as-is
  if (url.startsWith("http://") || url.startsWith("https://") || url.startsWith("data:")) {
    return url;
  }
  // Resolve relative path against the origin base
  return `${ORIGIN_BASE}${url}`;
}
