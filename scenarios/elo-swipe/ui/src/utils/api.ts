const runtimeApiUrl = ((): string => {
  if (typeof window !== 'undefined') {
    const fromWindow = (window as unknown as { __API_URL__?: string }).__API_URL__;
    if (fromWindow) {
      return fromWindow;
    }
  }

  if (typeof __API_URL__ !== 'undefined') {
    return __API_URL__;
  }

  const port = import.meta.env.VITE_API_PORT || import.meta.env.API_PORT || '30400';
  return `http://localhost:${port}/api/v1`;
})();

export const API_BASE_URL = runtimeApiUrl.replace(/\/$/, '');

export const apiFetch = async <T>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
    ...init,
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed: ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
};

declare const __API_URL__: string | undefined;

declare global {
  interface ImportMetaEnv {
    readonly VITE_API_PORT?: string;
    readonly VITE_API_URL?: string;
    readonly API_PORT?: string;
  }
}
