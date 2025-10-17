import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined') {
  if (typeof window.initIframeBridgeChild !== 'function') {
    window.initIframeBridgeChild = initIframeBridgeChild
  }

  if (window.parent !== window && !window.__seoOptimizerBridgeInitialized) {
    window.initIframeBridgeChild({ appId: 'seo-optimizer-ui' })
    window.__seoOptimizerBridgeInitialized = true
  }
}
