import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export function buildUrl(path: string): string {
  return buildApiUrl(path, { baseUrl: API_BASE });
}

export async function throwIfNotOk(response: Response): Promise<void> {
  if (response.ok) return;
  let detail = "";
  try {
    const body = await response.text();
    if (body) detail = `: ${body}`;
  } catch {
    // Ignore body read failures.
  }
  throw new Error(`HTTP ${response.status} ${response.statusText}${detail}`);
}
