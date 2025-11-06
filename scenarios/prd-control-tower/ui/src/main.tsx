import { StrictMode } from 'react'
import ReactDOM from 'react-dom/client'
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import { ConfirmDialogProvider } from './utils/confirmDialog'
import Catalog from './pages/Catalog'
import PRDViewer from './pages/PRDViewer'
import Drafts from './pages/Drafts'
import './index.css'

declare global {
  interface Window {
    __prdControlTowerBridgeInitialized?: boolean
  }
}

const BRIDGE_FLAG = '__prdControlTowerBridgeInitialized'

if (typeof window !== 'undefined' && window.parent !== window) {
  const globalWindow = window as Window & Record<string, unknown>
  if (!globalWindow[BRIDGE_FLAG]) {
    let parentOrigin: string | undefined
    try {
      if (document.referrer) {
        parentOrigin = new URL(document.referrer).origin
      }
    } catch {
      // Unable to determine parent origin; iframe bridge will use fallback behavior
    }

    try {
      initIframeBridgeChild({
        parentOrigin,
        appId: 'prd-control-tower',
        captureLogs: { enabled: true },
        captureNetwork: { enabled: true },
      })
      globalWindow[BRIDGE_FLAG] = true
    } catch (error) {
      console.warn('[prd-control-tower] Unable to initialize iframe bridge', error)
    }
  }
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ConfirmDialogProvider>
      <HashRouter>
        <Routes>
          <Route path="/" element={<Catalog />} />
          <Route path="/prd/:type/:name" element={<PRDViewer />} />
          <Route path="/drafts" element={<Drafts />} />
          <Route path="/draft/:entityType/:entityName" element={<Drafts />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </HashRouter>
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          style: {
            background: '#fff',
            color: '#1a202c',
            padding: '16px',
            borderRadius: '8px',
            boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
          },
          success: {
            iconTheme: {
              primary: '#10b981',
              secondary: '#fff',
            },
          },
          error: {
            iconTheme: {
              primary: '#ef4444',
              secondary: '#fff',
            },
          },
        }}
      />
    </ConfirmDialogProvider>
  </StrictMode>,
)
