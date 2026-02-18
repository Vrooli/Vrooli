import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge'
import App from './App.tsx'

// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe bridge initialization              ║
// ║                                                              ║
// ║  Must run BEFORE React mount so that:                        ║
// ║  1. Storage shimming is in place before any component        ║
// ║     accesses localStorage/sessionStorage                     ║
// ║  2. The bridge message channel is ready for host commands    ║
// ║                                                              ║
// ║  The window.parent check ensures this is a no-op when        ║
// ║  running outside an iframe (localhost, tunnel).              ║
// ╚══════════════════════════════════════════════════════════════╝

declare global {
  interface Window {
    __systemMonitorBridgeInitialized?: boolean;
  }
}

if (
  typeof window !== 'undefined' &&
  window.parent !== window &&
  !window.__systemMonitorBridgeInitialized
) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Fall back to default origin when parsing fails.
  }

  initIframeBridgeChild({ parentOrigin, appId: 'system-monitor' });
  window.__systemMonitorBridgeInitialized = true;
}

const rootEl = document.getElementById('root');
if (!rootEl) throw new Error('Root element not found');
createRoot(rootEl).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
