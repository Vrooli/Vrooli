import { useCallback, useEffect, useRef, useState, useMemo } from 'react';

type BridgeCapability = 'history' | 'hash' | 'title' | 'deeplink' | 'resize' | 'screenshot' | 'logs' | 'network' | 'inspect';

interface BridgeState {
  isSupported: boolean;
  isReady: boolean;
  href: string;
  title?: string;
  canGoBack: boolean;
  canGoForward: boolean;
  caps: BridgeCapability[];
}

const initialBridgeState: BridgeState = {
  isSupported: false,
  isReady: false,
  href: '',
  title: undefined,
  canGoBack: false,
  canGoForward: false,
  caps: [],
};

interface UseIframeBridgeOptions {
  iframeRef: React.RefObject<HTMLIFrameElement>;
  previewUrl: string | null;
  onLocation?: (href: string, title?: string) => void;
}

export interface UseIframeBridgeReturn {
  state: BridgeState;
  childOrigin: string | null;
  sendNav: (command: 'BACK' | 'FWD' | 'GO', target?: string) => boolean;
  sendPing: () => boolean;
  resetState: () => void;
}

const deriveOrigin = (url: string | null): string | null => {
  if (!url || typeof window === 'undefined') {
    return null;
  }

  try {
    const resolved = new URL(url, window.location.href);
    return resolved.origin;
  } catch {
    return null;
  }
};

export const useIframeBridge = ({
  iframeRef,
  previewUrl,
  onLocation,
}: UseIframeBridgeOptions): UseIframeBridgeReturn => {
  const [state, setState] = useState<BridgeState>(initialBridgeState);
  const childOrigin = useMemo(() => deriveOrigin(previewUrl), [previewUrl]);
  const effectiveOriginRef = useRef<string | null>(null);
  const helloReceivedRef = useRef(false);
  const readyReceivedRef = useRef(false);

  const resetState = useCallback(() => {
    helloReceivedRef.current = false;
    readyReceivedRef.current = false;
    effectiveOriginRef.current = null;
    setState(initialBridgeState);
  }, []);

  useEffect(() => {
    resetState();
  }, [childOrigin, resetState, previewUrl]);

  useEffect(() => {
    effectiveOriginRef.current = childOrigin ?? null;
  }, [childOrigin]);

  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      const iframeWindow = iframeRef.current?.contentWindow;
      if (!iframeWindow || event.source !== iframeWindow) {
        return;
      }

      const message = event.data;
      if (!message || typeof message !== 'object' || message.v !== 1) {
        return;
      }

      if (event.origin) {
        effectiveOriginRef.current = event.origin;
      }

      switch (message.t) {
        case 'HELLO': {
          helloReceivedRef.current = true;
          const nextCaps = Array.isArray(message.caps) ? message.caps : [];
          setState(prev => ({
            ...prev,
            isSupported: true,
            caps: nextCaps,
            title: message.title ?? prev.title,
          }));
          break;
        }

        case 'READY': {
          readyReceivedRef.current = true;
          setState(prev => ({
            ...prev,
            isReady: true,
          }));
          break;
        }

        case 'LOCATION': {
          setState(prev => ({
            ...prev,
            href: message.href,
            title: message.title ?? prev.title,
            canGoBack: !!message.canGoBack,
            canGoForward: !!message.canGoFwd,
          }));
          onLocation?.(message.href, message.title);
          break;
        }

        default:
          break;
      }
    };

    window.addEventListener('message', handleMessage);
    return () => {
      window.removeEventListener('message', handleMessage);
    };
  }, [childOrigin, iframeRef, onLocation]);

  const postMessage = useCallback(
    (payload: { v: 1; t: string; [key: string]: unknown }) => {
      const iframeWindow = iframeRef.current?.contentWindow;
      if (!iframeWindow) {
        return false;
      }
      const targetOrigin = effectiveOriginRef.current ?? childOrigin;
      if (!targetOrigin) {
        return false;
      }
      try {
        iframeWindow.postMessage(payload, targetOrigin);
        return true;
      } catch {
        return false;
      }
    },
    [childOrigin, iframeRef],
  );

  const sendNav = useCallback(
    (command: 'BACK' | 'FWD' | 'GO', target?: string) => {
      if (command === 'GO') {
        return postMessage({ v: 1, t: 'NAV', cmd: 'GO', to: target });
      }
      if (command === 'BACK') {
        return postMessage({ v: 1, t: 'NAV', cmd: 'BACK' });
      }
      return postMessage({ v: 1, t: 'NAV', cmd: 'FWD' });
    },
    [postMessage],
  );

  const sendPing = useCallback(() => {
    return postMessage({ v: 1, t: 'PING', ts: Date.now() });
  }, [postMessage]);

  return {
    state: {
      ...state,
      isSupported: state.isSupported || helloReceivedRef.current,
      isReady: state.isReady || readyReceivedRef.current,
    },
    childOrigin,
    sendNav,
    sendPing,
    resetState,
  };
};

export default useIframeBridge;
