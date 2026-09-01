/**
 * @libraryId react-component-library:AuthClient
 * @displayName AuthClient
 * @description Shared authenticated API client boundary for paid surfaces
 * @version 1.0.0
 * @tags ["auth","monetization"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
export interface AuthClientOptions { baseURL: string; getToken?: () => string | undefined; fetchImpl?: typeof fetch; }
export class AuthClient {
  constructor(private readonly options: AuthClientOptions) {}
  async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers);
    const token = this.options.getToken?.();
    if (token && !headers.has("Authorization")) headers.set("Authorization", `Bearer ${token}`);
    const target = path.startsWith("/") ? path : new URL(path, this.options.baseURL);
    const response = await (this.options.fetchImpl ?? fetch)(target, { ...init, headers });
    if (!response.ok) throw new Error(`authenticated request failed: ${response.status}`);
    if (response.status === 204 || typeof response.json !== "function") return undefined as T;
    return response.json() as Promise<T>;
  }
}
export default AuthClient;
