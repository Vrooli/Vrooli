import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window !== window.parent) {
  initIframeBridgeChild({ appId: 'demo', captureLogs: { enabled: true } })
}
