declare module '@vrooli/api-base' {
  export interface ResolveApiBaseOptions {
    explicitUrl?: string | null
    defaultPort?: string
    appendSuffix?: boolean
  }

  export interface BuildApiUrlOptions extends ResolveApiBaseOptions {
    baseUrl?: string
  }

  export function resolveApiBase(options?: ResolveApiBaseOptions): string
  export function buildApiUrl(path: string, options?: BuildApiUrlOptions): string
}

declare module '@vrooli/iframe-bridge/child' {
  interface BridgeInitOptions {
    appId?: string
    parentOrigin?: string
    enableNetwork?: boolean
  }

  export function initIframeBridgeChild(options?: BridgeInitOptions): void
}
