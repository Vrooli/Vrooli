declare module '@vrooli/iframe-bridge/child' {
  interface BridgeController {
    notify: () => void;
    destroy?: () => void;
  }

  interface InitOptions {
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

  export function initIframeBridgeChild(options: InitOptions): BridgeController;
}

interface Window {
  __qrCodeGeneratorBridgeInitialized?: boolean;
  __APP_MONITOR_PROXY_INFO__?: unknown;
  __APP_MONITOR_PROXY_INDEX__?: unknown;
  ENV?: {
    API_URL?: string;
  };
}
