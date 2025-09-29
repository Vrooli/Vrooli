import React from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App'
import './index.css'

if (typeof window !== 'undefined' && window.parent !== window && !window.__financialCalculatorsBridgeInitialized) {
  let parentOrigin
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[FinancialCalculators] Unable to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'financial-calculators-hub' })
  window.__financialCalculatorsBridgeInitialized = true
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
