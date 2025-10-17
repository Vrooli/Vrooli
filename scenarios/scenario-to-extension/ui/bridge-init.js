import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && typeof window.initIframeBridgeChild !== 'function') {
  // Expose the shared bridge initializer for non-module entry scripts
  window.initIframeBridgeChild = initIframeBridgeChild
}
