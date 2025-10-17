import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'demo', captureNetwork: { enabled: false } })
}
