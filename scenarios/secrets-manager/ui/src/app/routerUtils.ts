export function normalizeRouterBasename(baseUrl: string): string {
  const normalized = baseUrl.trim().replace(/\/+$/, "");
  return normalized === "" || normalized === "." ? "/" : normalized;
}
