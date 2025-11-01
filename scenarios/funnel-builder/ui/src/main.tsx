import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import toast, { Toaster, ToastBar } from 'react-hot-toast'
import { X } from 'lucide-react'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App'
import './index.css'
import type { AppMonitorProxyInfo } from './utils/apiClient'

declare global {
  interface Window {
    __funnelBuilderBridgeInitialized?: boolean
    __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo
    __APP_MONITOR_PROXY_INDEX__?: AppMonitorProxyInfo
    __FUNNEL_BUILDER_BASE_PATH__?: string
  }

  interface ImportMeta {
    readonly env: Record<string, string | undefined>
  }
}

const normalizeBasename = (value?: string | null) => {
  if (!value) {
    return ''
  }

  const trimmed = value.trim()
  if (!trimmed) {
    return ''
  }

  const withLeading = trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  if (withLeading === '/') {
    return '/'
  }

  return withLeading.replace(/\/+$/, '')
}

const resolveRouterBase = () => {
  const fallback = normalizeBasename(import.meta.env.BASE_URL || '/') || '/'

  if (typeof window === 'undefined') {
    return fallback
  }

  const info = window.__APP_MONITOR_PROXY_INFO__
  const candidates = [info?.primary?.path, info?.path, info?.primary?.basePath]

  for (const candidate of candidates) {
    const normalized = normalizeBasename(candidate)
    if (normalized) {
      return normalized
    }
  }

  const pathname = window.location?.pathname
  if (pathname && pathname.includes('/proxy/')) {
    const index = pathname.indexOf('/proxy/')
    if (index >= 0) {
      return normalizeBasename(pathname.slice(0, index + '/proxy'.length)) || fallback
    }
  }

  return fallback
}

const routerBase = resolveRouterBase()

if (typeof window !== 'undefined') {
  window.__FUNNEL_BUILDER_BASE_PATH__ = routerBase
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__funnelBuilderBridgeInitialized) {
  let parentOrigin: string | undefined
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[FunnelBuilder] Unable to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'funnel-builder' })
  window.__funnelBuilderBridgeInitialized = true
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter basename={routerBase}>
      <App />
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          style: {
            background: '#fff',
            color: '#363636',
            boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
          },
        }}
      >
        {(t) => (
          <ToastBar toast={t}>
            {({ icon, message }) => (
              <div className="flex w-full items-start gap-3">
                {icon}
                <div className="flex-1 text-sm text-gray-900">{message}</div>
                {t.type !== 'loading' && (
                  <button
                    type="button"
                    onClick={() => toast.dismiss(t.id)}
                    className="ml-2 inline-flex h-6 w-6 items-center justify-center rounded-full border border-transparent text-gray-500 transition hover:text-gray-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-1"
                    aria-label="Close notification"
                  >
                    <X className="h-3.5 w-3.5" aria-hidden="true" />
                  </button>
                )}
              </div>
            )}
          </ToastBar>
        )}
      </Toaster>
    </BrowserRouter>
  </React.StrictMode>
)
