import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App'
import './index.css'

declare global {
  interface Window {
    __graphStudioBridgeInitialized?: boolean
  }
}

const BRIDGE_STATE_KEY = '__graphStudioBridgeInitialized' as const

if (typeof window !== 'undefined' && window.parent !== window && !window[BRIDGE_STATE_KEY]) {
  let parentOrigin: string | undefined
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[GraphStudio] Unable to determine parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'graph-studio' })
  window[BRIDGE_STATE_KEY] = true
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 3,
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
      staleTime: 5 * 60 * 1000, // 5 minutes
    },
  },
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>,
)
