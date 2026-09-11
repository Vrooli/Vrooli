import { createScenarioConnectTransport, resolveApiBase } from "@vrooli/api-base";

declare global {
  interface Window {
    desktop?: {
      auth?: {
        getLocalSessionToken?: () => Promise<string | null>;
      };
    };
  }
}

/** One deployment-aware API base for every onboarding client. */
export const API_BASE = resolveApiBase();
export const REST_API_BASE = resolveApiBase({ appendSuffix: true });

export async function onboardingFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const token = typeof window !== "undefined"
    ? await window.desktop?.auth?.getLocalSessionToken?.()
    : null;
  const requestInit: RequestInit = { ...init, credentials: init?.credentials ?? "include" };
  if (!token) return fetch(input, requestInit);

  const headers = new Headers(requestInit.headers);
  if (!headers.has("Authorization")) headers.set("Authorization", `LocalSession ${token}`);
  return fetch(input, { ...requestInit, headers });
}

export function onboardingTransport() {
  return createScenarioConnectTransport({ baseUrl: API_BASE, fetch: onboardingFetch });
}
