/// <reference types="vite/client" />

declare interface ImportMetaEnv {
  readonly VITE_API_PORT?: string
  readonly VITE_API_BASE_URL?: string
  readonly VITE_PROXY_API_ORIGIN?: string
  readonly VITE_PROXY_API_TARGET?: string
  readonly VITE_AUTH_UI_URL?: string
  readonly VITE_AUTH_URL?: string
  readonly VITE_AUTH_BASE_URL?: string
  readonly VITE_AUTHENTICATOR_URL?: string
  readonly VITE_SCENARIO_AUTHENTICATOR_URL?: string
  readonly VITE_SCENARIO_AUTH_URL?: string
  readonly VITE_AUTH_PORT?: string
}

declare interface ImportMeta {
  readonly env: ImportMetaEnv
}
