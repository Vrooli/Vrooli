import { resolveApiBase, buildApiUrl as composeApiUrl } from '@vrooli/api-base'

const DEFAULT_API_PORT = (import.meta.env.VITE_API_PORT as string | undefined)?.trim() || '8080'

const API_BASE_PREFIX = resolveApiBase({
  explicitUrl: import.meta.env.VITE_API_BASE_URL as string | undefined,
  defaultPort: DEFAULT_API_PORT,
  appendSuffix: false,
})

export const buildApiUrl = (path: string): string => composeApiUrl(path, { baseUrl: API_BASE_PREFIX })

export const getApiBasePrefix = () => API_BASE_PREFIX
