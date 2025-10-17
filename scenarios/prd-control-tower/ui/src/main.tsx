import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import Catalog from './pages/Catalog'
import PRDViewer from './pages/PRDViewer'
import Drafts from './pages/Drafts'
import './index.css'

declare global {
  interface Window {
    __prdControlTowerBridgeInitialized?: boolean
  }
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__prdControlTowerBridgeInitialized) {
  let parentOrigin: string | undefined
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[PRDControlTower] Unable to determine parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'prd-control-tower' })
  window.__prdControlTowerBridgeInitialized = true
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Catalog />} />
        <Route path="/prd/:type/:name" element={<PRDViewer />} />
        <Route path="/drafts" element={<Drafts />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>,
)
