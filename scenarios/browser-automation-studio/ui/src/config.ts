import { resolveApiBase, resolveWsBase } from '@vrooli/api-base'
import type { Config } from './types/config'

declare global {
  interface Window {
    __BAS_EXPORT_API_BASE__?: string
  }
}

const normalizeForUrlParsing = (value: string): string => {
  if (!value) {
    return value
  }
  if (value.startsWith('ws://')) {
    return value.replace('ws://', 'http://')
  }
  if (value.startsWith('wss://')) {
    return value.replace('wss://', 'https://')
  }
  return value
}

const safeParsePort = (value: string): string => {
  if (!value) {
    return ''
  }
  try {
    const base = typeof window !== 'undefined' && window.location?.origin
      ? window.location.origin
      : 'http://127.0.0.1'
    const parsed = new URL(normalizeForUrlParsing(value), base)
    return parsed.port || ''
  } catch {
    return ''
  }
}

const getWindowPort = (): string => {
  if (typeof window === 'undefined' || !window.location) {
    return ''
  }
  return window.location.port || ''
}

const getExportApiOverride = (): string | undefined => {
  if (typeof window === 'undefined') {
    return undefined
  }
  const candidate = window.__BAS_EXPORT_API_BASE__
  if (typeof candidate !== 'string') {
    return undefined
  }
  const trimmed = candidate.trim()
  return trimmed.length > 0 ? trimmed : undefined
}

const API_OVERRIDE = getExportApiOverride()

const API_URL = resolveApiBase({
  appendSuffix: true,
  explicitUrl: API_OVERRIDE,
})

const WS_ROOT = resolveWsBase({
  appendSuffix: true,
  apiSuffix: '/ws',
  explicitUrl: API_OVERRIDE,
})

const runtimeConfig: Config = {
  API_URL,
  WS_URL: WS_ROOT,
  API_PORT: safeParsePort(API_URL),
  UI_PORT: getWindowPort(),
  WS_PORT: safeParsePort(WS_ROOT),
}

export async function getConfig(): Promise<Config> {
  return runtimeConfig
}

export const getCachedConfig = (): Config => runtimeConfig
export const config = (): Config => runtimeConfig
export const getApiBase = (): string => runtimeConfig.API_URL
export const getWsBase = (): string => runtimeConfig.WS_URL
export const API_BASE = runtimeConfig.API_URL
export const WS_BASE = runtimeConfig.WS_URL
