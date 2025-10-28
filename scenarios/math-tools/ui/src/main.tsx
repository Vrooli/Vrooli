import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App'
import './styles/tailwind.css'

declare global {
  interface Window {
    __mathToolsBridgeInitialized?: boolean
  }
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__mathToolsBridgeInitialized) {
  let parentOrigin: string | undefined

  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[math-tools-ui] Failed to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'math-tools' })
  window.__mathToolsBridgeInitialized = true
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>,
)
