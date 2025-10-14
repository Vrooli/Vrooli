import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined') {
  if (typeof window.initIframeBridgeChild !== 'function') {
    window.initIframeBridgeChild = initIframeBridgeChild
  }

  if (window.parent !== window && !window.__swarmManagerBridgeInitialized) {
    window.initIframeBridgeChild({ appId: 'swarm-manager-ui' })
    window.__swarmManagerBridgeInitialized = true
  }
}
