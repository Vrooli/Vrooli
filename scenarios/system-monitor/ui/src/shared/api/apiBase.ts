import { resolveApiBase, buildApiUrl as composeApiUrl } from '@vrooli/api-base'

// Standard @vrooli/api-base usage: resolve API origin once and append /api/v1 so
// every request flows through the UI proxy no matter the deployment context.
const API_BASE_PREFIX = resolveApiBase({ appendSuffix: true })

export const buildApiUrl = (path: string): string => composeApiUrl(path, { baseUrl: API_BASE_PREFIX })

export const getApiBasePrefix = () => API_BASE_PREFIX
