import { StrictMode } from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App.tsx'

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'text-tools-react-ui' })
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
