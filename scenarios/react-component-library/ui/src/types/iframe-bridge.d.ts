interface BridgeController {
  notify: () => void;
  destroy?: () => void;
}

interface BridgeInitOptions {
  parentOrigin?: string;
  appId: string;
  captureLogs?: {
    enabled: boolean;
    streaming?: boolean;
    bufferSize?: number;
    levels?: string[];
  };
  captureNetwork?: {
    enabled: boolean;
    streaming?: boolean;
    bufferSize?: number;
  };
}

declare module '@vrooli/iframe-bridge' {
  export { BridgeController, BridgeInitOptions };
  export function initIframeBridgeChild(options: BridgeInitOptions): BridgeController;
}

declare module '@vrooli/iframe-bridge/child' {
  export { BridgeController, BridgeInitOptions };
  export function initIframeBridgeChild(options: BridgeInitOptions): BridgeController;
}

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INFO__?: unknown;
    __APP_MONITOR_PROXY_INDEX__?: unknown;
  }
}
