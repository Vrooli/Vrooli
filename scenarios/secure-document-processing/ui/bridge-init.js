import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

const bridgeWindow = typeof window !== 'undefined' ? window : undefined

if (bridgeWindow && typeof bridgeWindow.initIframeBridgeChild !== 'function') {
  bridgeWindow.initIframeBridgeChild = initIframeBridgeChild
}
