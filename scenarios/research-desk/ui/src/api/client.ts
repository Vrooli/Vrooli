import { createClient, type Client } from "@connectrpc/connect";
import type { DescService } from "@bufbuild/protobuf";
import { buildApiUrl, createScenarioConnectTransport, resolveApiBase } from "@vrooli/api-base";

/**
 * The one transport every domain client shares. Connect calls go to the UI
 * server, which forwards `/vrooli.research_desk.v1.*` to the API; plain REST
 * (`/api/v1/...`) is forwarded by the template proxy. Both stay on the page's
 * own origin, so the desk works unchanged through the tunnel hostname.
 */
// fetch is looked up per call so tests can observe the exact request on the wire.
const transport = createScenarioConnectTransport({ baseUrl: resolveApiBase(), fetch: (input, init) => globalThis.fetch(input, init) });
const REST_BASE = resolveApiBase({ appendSuffix: true });

export function serviceClient<T extends DescService>(service: T): Client<T> {
  return createClient(service, transport);
}

export function restUrl(path: string): string {
  return buildApiUrl(path, { baseUrl: REST_BASE });
}

/**
 * The URL of a research-home file on the desk's read-only file route, or null
 * when the path is outside the home (session records, skill files): those are
 * shown as text the operator can copy, never as a link that would fail.
 */
export function homeFileUrl(path: string, researchHome: string | undefined): string | null {
  const rel = homeRelative(path, researchHome);
  return rel === null ? null : restUrl(`/home/${rel.split("/").map(encodeURIComponent).join("/")}`);
}

export function homeRelative(path: string, researchHome: string | undefined): string | null {
  if (!researchHome || !path.startsWith(`${researchHome}/`)) return null;
  const rel = path.slice(researchHome.length + 1);
  return rel.split("/").some((part) => part === ".." || part.startsWith(".")) ? null : rel;
}

/** An `/api/v1/...` path returned by the API, resolved against this page's API base. */
export function apiPathUrl(path: string): string | null {
  if (!path) return null;
  return path.startsWith("/api/v1/") ? restUrl(path.slice("/api/v1".length)) : null;
}
