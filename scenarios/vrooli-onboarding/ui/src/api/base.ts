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

async function onboardingFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const token = typeof window !== "undefined"
    ? await window.desktop?.auth?.getLocalSessionToken?.()
    : null;
  if (!token) return fetch(input, init);

  const headers = new Headers(init?.headers);
  if (!headers.has("Authorization")) headers.set("Authorization", `LocalSession ${token}`);
  return fetch(input, { ...init, headers });
}

export function onboardingTransport() {
  return createScenarioConnectTransport({ baseUrl: API_BASE, fetch: onboardingFetch });
}
