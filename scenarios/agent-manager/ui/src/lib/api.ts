import { createClient } from "@connectrpc/connect";
import { resolveApiBase, buildApiUrl, createScenarioConnectTransport } from "@vrooli/api-base";
import { AgentManagerService } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

// One scenario-owned resolver keeps REST and Connect clients on the same
// proxy-aware base URL without each feature re-implementing environment logic.
export function resolveAgentManagerApiBase(appendSuffix = false): string {
  return resolveApiBase({ appendSuffix });
}

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveAgentManagerApiBase(true);

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
  const res = await fetchWithDeadline(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`Path validation failed: ${res.status}`);
  }

  return res.json() as Promise<PathValidationResult>;
}

/** Every request the UI makes gives up after this long; a hung API reads as an error, not a spinner. */
export const REQUEST_DEADLINE_MS = 30_000;

/** Per-call options giving a Connect call the same deadline `fetchWithDeadline` gives a REST call. */
export const CALL_DEADLINE = { timeoutMs: REQUEST_DEADLINE_MS };

/**
 * The one typed client for Agent Manager's own procedures. `fetch` is resolved
 * per call so a test that stubs the global sees every request.
 */
export const agentManager = createClient(AgentManagerService, createScenarioConnectTransport({
  baseUrl: resolveAgentManagerApiBase(),
  fetch: (input, init) => globalThis.fetch(input, init),
}));

/** fetch that aborts at the shared deadline (and when the caller's own signal aborts). */
export async function fetchWithDeadline(url: string, init: RequestInit = {}): Promise<Response> {
  const deadline = AbortSignal.timeout(REQUEST_DEADLINE_MS);
  const signal = init.signal ? AbortSignal.any([init.signal, deadline]) : deadline;
  try {
    return await fetch(url, { ...init, signal });
  } catch (error) {
    if (deadline.aborted && !init.signal?.aborted) {
      throw new Error(`Request timed out after ${REQUEST_DEADLINE_MS / 1000} s`);
    }
    throw error;
  }
}
