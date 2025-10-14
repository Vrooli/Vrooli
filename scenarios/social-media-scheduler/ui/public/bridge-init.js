import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined') {
  if (typeof window.initIframeBridgeChild !== 'function') {
    window.initIframeBridgeChild = initIframeBridgeChild
  }

  if (window.parent !== window && !window.__socialMediaSchedulerBridgeInitialized) {
    window.initIframeBridgeChild({ appId: 'social-media-scheduler-ui' })
    window.__socialMediaSchedulerBridgeInitialized = true
  }
}
