import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import App from './App.tsx'
import './index.css'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

declare global {
  interface Window {
    __scenarioAuditorBridgeInitialized?: boolean
  }
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__scenarioAuditorBridgeInitialized) {
  let parentOrigin: string | undefined
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[ScenarioAuditor] Unable to parse parent origin from referrer', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'scenario-auditor' })
  window.__scenarioAuditorBridgeInitialized = true
}

const routerBasename = import.meta.env.BASE_URL || '/'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60,
      refetchOnWindowFocus: false,
      retry: 2,
    },
  },
})

const rootElement = document.getElementById('root')

if (!rootElement) {
  throw new Error('Scenario Auditor root element was not found')
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter basename={routerBasename}>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>,
)
