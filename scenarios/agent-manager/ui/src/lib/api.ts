import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true });

export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json() as Promise<{ status: string; service: string; timestamp: string }>;
}

// --- Path Validation ---

export interface PathValidationResult {
  path: string;
  projectRoot?: string;
  valid: boolean;
  exists?: boolean;
  isDirectory?: boolean;
  withinProjectRoot?: boolean;
  error?: string;
}

export async function validatePath(
  path: string,
  projectRoot?: string
): Promise<PathValidationResult> {
  const params = new URLSearchParams({ path });
  if (projectRoot) {
    params.set("projectRoot", projectRoot);
  }
  const url = buildApiUrl(`/validate-path?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`Path validation failed: ${res.status}`);
  }

  return res.json() as Promise<PathValidationResult>;
}
