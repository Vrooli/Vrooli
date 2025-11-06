import type { BridgeChildController } from '@vrooli/iframe-bridge/child';

export type BridgeController = BridgeChildController;

declare global {
  interface Window {
    __qrCodeGeneratorBridgeInitialized?: boolean;
    __qrCodeGeneratorBridgeController?: BridgeController | null;
    __APP_MONITOR_PROXY_INFO__?: unknown;
    __APP_MONITOR_PROXY_INDEX__?: unknown;
    ENV?: {
      API_URL?: string;
    };
  }
}

let bridgeController: BridgeController | null = null;

function hydrateCachedBridge(): void {
  if (typeof window === 'undefined') {
    return;
  }

  if (bridgeController !== null) {
    return;
  }

  if ('__qrCodeGeneratorBridgeController' in window) {
    bridgeController = window.__qrCodeGeneratorBridgeController ?? null;
  }
}

export function cacheBridgeController(controller: BridgeController | null) {
  bridgeController = controller;

  if (typeof window !== 'undefined') {
    window.__qrCodeGeneratorBridgeInitialized = Boolean(controller);
    window.__qrCodeGeneratorBridgeController = controller;
  }
}

export function ensureIframeBridge(): BridgeController | null {
  if (typeof window === 'undefined' || window.parent === window) {
    return null;
  }

  hydrateCachedBridge();

  if (bridgeController) {
    bridgeController.notify?.();
  }

  return bridgeController;
}
