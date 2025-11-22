import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

let API_BASE_URL: string | null = null;

const getApiBaseUrl = () => {
  if (API_BASE_URL === null) {
    API_BASE_URL = resolveApiBase({
      appendSuffix: true,
    });
  }
  return API_BASE_URL;
};

export async function fetchHealth() {
  const res = await fetch(buildApiUrl("/health", { baseUrl: getApiBaseUrl() }), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json() as Promise<{ status: string; service: string; timestamp: string }>;
}
