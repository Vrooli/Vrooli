import { i18n } from "./i18n";
import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1";
import { BaseStyles } from "@vrooli/react-component-library/BaseStyles/1";
import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import { initSpatialNav } from '@vrooli/iframe-bridge/spatial'
import { installChunkReloadGuard } from '@vrooli/api-base'
import App from './App.tsx'
import { onProfilerRender } from './lib/profiler'
import './styles/globals.css'

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard()

declare global {
  interface Window {
    __promptManagerBridgeInitialized?: boolean
  }
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__promptManagerBridgeInitialized) {
  let parentOrigin: string | undefined
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[PromptManager] Unable to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'prompt-manager' })
  window.__promptManagerBridgeInitialized = true
}

// INTEROP-CRITICAL: Enables keyboard/gamepad spatial navigation for embedded Vrooli surfaces.
initSpatialNav()

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      retry: 3,
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
    },
  },
})

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Root element not found')
}
ReactDOM.createRoot(rootElement).render(
    // vrooli:library-strings-provider start
    <LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>
      <BaseStyles />
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <React.Profiler id="App" onRender={onProfilerRender}>
        <App />
      </React.Profiler>
    </QueryClientProvider>
  </React.StrictMode>,

    </LibraryStringsProvider>
    // vrooli:library-strings-provider end
)
