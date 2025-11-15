import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

const API_BASE = resolveApiBase({ appendSuffix: true })

export const apiUrl = (path: string): string => buildApiUrl(path, { baseUrl: API_BASE })
