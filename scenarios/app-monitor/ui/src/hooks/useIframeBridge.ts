import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { BridgeCapability } from '@vrooli/iframe-bridge';

type BridgeHelloMessage = {
  v: 1;
  t: 'HELLO';
  appId?: string;
  title?: string;
  caps?: BridgeCapability[];
};

type BridgeReadyMessage = {
  v: 1;
  t: 'READY';
};

type BridgeLocationMessage = {
  v: 1;
  t: 'LOCATION';
  href: string;
  path?: string;
  title?: string;
  canGoBack?: boolean;
  canGoFwd?: boolean;
};

type BridgeErrorMessage = {
  v: 1;
  t: 'ERROR';
  code: string;
  detail?: string;
};

type BridgePongMessage = {
  v: 1;
  t: 'PONG';
  ts: number;
};

type BridgeScreenshotResultMessage = {
  v: 1;
  t: 'SCREENSHOT_RESULT';
  id: string;
  ok: boolean;
  data?: string;
  width?: number;
  height?: number;
  note?: string;
  error?: string;
};

type BridgeChildToParentMessage =
  | BridgeHelloMessage
  | BridgeReadyMessage
  | BridgeLocationMessage
  | BridgeErrorMessage
  | BridgePongMessage
  | BridgeScreenshotResultMessage;

type BridgeParentToChildMessage =
  | { v: 1; t: 'NAV'; cmd: 'GO'; to?: string }
  | { v: 1; t: 'NAV'; cmd: 'BACK' }
  | { v: 1; t: 'NAV'; cmd: 'FWD' }
  | { v: 1; t: 'PING'; ts: number }
  | { v: 1; t: 'CAPTURE'; cmd: 'SCREENSHOT'; id: string; options?: { scale?: number } };

export interface BridgeComplianceResult {
  ok: boolean;
  failures: string[];
  checkedAt: number;
}

interface BridgeState {
  isSupported: boolean;
  isReady: boolean;
  href: string;
  title?: string;
  canGoBack: boolean;
  canGoForward: boolean;
  caps: BridgeCapability[];
  lastError?: { code: string; detail?: string };
  lastHelloAt?: number;
  lastReadyAt?: number;
  lastLocationAt?: number;
}

const initialBridgeState: BridgeState = {
  isSupported: false,
  isReady: false,
  href: '',
  title: undefined,
  canGoBack: false,
  canGoForward: false,
  caps: [],
  lastError: undefined,
  lastHelloAt: undefined,
  lastReadyAt: undefined,
  lastLocationAt: undefined,
};

interface UseIframeBridgeOptions {
  iframeRef: React.RefObject<HTMLIFrameElement>;
  previewUrl: string | null;
  onLocation?: (message: BridgeLocationMessage) => void;
}

export interface UseIframeBridgeReturn {
  state: BridgeState;
  childOrigin: string | null;
  sendNav: (command: 'BACK' | 'FWD' | 'GO', target?: string) => boolean;
  sendPing: () => boolean;
  runComplianceCheck: () => Promise<BridgeComplianceResult>;
  resetState: () => void;
  requestScreenshot: (options?: { scale?: number }) => Promise<{
    data: string;
    width: number;
    height: number;
    note?: string;
  }>;
}

const deriveOrigin = (url: string | null): string | null => {
  if (!url || typeof window === 'undefined') {
    return null;
  }

  try {
    const resolved = new URL(url, window.location.href);
    return resolved.origin;
  } catch (error) {
    console.warn('[Bridge] Failed to derive origin from URL', url, error);
    return null;
  }
};

export const useIframeBridge = ({ iframeRef, previewUrl, onLocation }: UseIframeBridgeOptions): UseIframeBridgeReturn => {
  const [state, setState] = useState<BridgeState>(initialBridgeState);
  const childOrigin = useMemo(() => deriveOrigin(previewUrl), [previewUrl]);
  const lastHrefRef = useRef<string>('');
  const helloReceivedRef = useRef(false);
  const readyReceivedRef = useRef(false);
  const effectiveOriginRef = useRef<string | null>(null);
  const pendingScreenshotRequestsRef = useRef(new Map<string, {
    resolve: (value: { data: string; width: number; height: number; note?: string }) => void;
    reject: (error: Error) => void;
    timeoutHandle: number;
  }>());

  const resetState = useCallback(() => {
    helloReceivedRef.current = false;
    readyReceivedRef.current = false;
    lastHrefRef.current = '';
    effectiveOriginRef.current = null;
    pendingScreenshotRequestsRef.current.forEach(({ reject, timeoutHandle }) => {
      window.clearTimeout(timeoutHandle);
      reject(new Error('bridge-reset'));
    });
    pendingScreenshotRequestsRef.current.clear();
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

      if (childOrigin && event.origin !== childOrigin) {
        console.warn('[Bridge] Message origin differs from expected', { expected: childOrigin, received: event.origin });
      }

      const message = event.data as BridgeChildToParentMessage | null | undefined;
      if (!message || typeof message !== 'object' || message.v !== 1) {
        return;
      }

      if (event.origin) {
        effectiveOriginRef.current = event.origin;
      }

      switch (message.t) {
        case 'HELLO': {
          helloReceivedRef.current = true;
          setState(prev => ({
            ...prev,
            isSupported: true,
            caps: Array.isArray(message.caps) ? message.caps : prev.caps,
            title: message.title ?? prev.title,
            lastHelloAt: Date.now(),
          }));
          break;
        }

        case 'READY': {
          readyReceivedRef.current = true;
          setState(prev => ({
            ...prev,
            isReady: true,
            lastReadyAt: Date.now(),
          }));
          break;
        }

        case 'LOCATION': {
          lastHrefRef.current = message.href;
          setState(prev => ({
            ...prev,
            href: message.href,
            title: message.title ?? prev.title,
            canGoBack: !!message.canGoBack,
            canGoForward: !!message.canGoFwd,
            lastLocationAt: Date.now(),
          }));
          onLocation?.(message);
          break;
        }

        case 'ERROR': {
          setState(prev => ({
            ...prev,
            lastError: { code: message.code, detail: message.detail },
          }));
          break;
        }

        case 'PONG': {
          // We could track latency here if needed later.
          break;
        }

        case 'SCREENSHOT_RESULT': {
          if (!message.id) {
            break;
          }
          const pending = pendingScreenshotRequestsRef.current.get(message.id);
          if (!pending) {
            break;
          }
          pendingScreenshotRequestsRef.current.delete(message.id);
          window.clearTimeout(pending.timeoutHandle);
          if (message.ok && typeof message.data === 'string' && typeof message.width === 'number' && typeof message.height === 'number') {
            pending.resolve({ data: message.data, width: message.width, height: message.height, note: message.note });
          } else {
            pending.reject(new Error(message.error || 'screenshot-failed'));
          }
          break;
        }

        default: {
          break;
        }
      }
    };

    window.addEventListener('message', handleMessage);
    return () => {
      window.removeEventListener('message', handleMessage);
    };
  }, [childOrigin, iframeRef, onLocation]);

  const postMessage = useCallback(
    (payload: BridgeParentToChildMessage) => {
      const iframeWindow = iframeRef.current?.contentWindow;
      if (!iframeWindow) {
        return false;
      }
      const targetOrigin = effectiveOriginRef.current ?? childOrigin;
      if (!targetOrigin) {
        console.warn('[Bridge] Unable to determine target origin for message', payload.t);
        return false;
      }
      try {
        iframeWindow.postMessage(payload, targetOrigin);
        return true;
      } catch (error) {
        console.warn('[Bridge] postMessage failed', error);
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

  const requestScreenshot = useCallback(
    (options?: { scale?: number }) => {
      return new Promise<{
        data: string;
        width: number;
        height: number;
        note?: string;
      }>((resolve, reject) => {
        const iframeWindow = iframeRef.current?.contentWindow;
        if (!iframeWindow) {
          reject(new Error('missing-iframe'));
          return;
        }

        const id = typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
          ? crypto.randomUUID()
          : `shot-${Date.now()}-${Math.random().toString(16).slice(2)}`;

        const timeoutHandle = window.setTimeout(() => {
          const pending = pendingScreenshotRequestsRef.current.get(id);
          if (pending) {
            pendingScreenshotRequestsRef.current.delete(id);
            pending.reject(new Error('screenshot-timeout'));
          }
        }, 7000);

        pendingScreenshotRequestsRef.current.set(id, { resolve, reject, timeoutHandle });

        const sent = postMessage({ v: 1, t: 'CAPTURE', cmd: 'SCREENSHOT', id, options });
        if (!sent) {
          window.clearTimeout(timeoutHandle);
          pendingScreenshotRequestsRef.current.delete(id);
          reject(new Error('screenshot-unsupported'));
        }
      });
    },
    [iframeRef, postMessage],
  );

  const waitForMessage = useCallback(
    (predicate: (message: BridgeChildToParentMessage) => boolean, timeoutMs: number) => {
      return new Promise<void>((resolve, reject) => {
        const iframeWindow = iframeRef.current?.contentWindow;
        if (!iframeWindow) {
          reject(new Error('missing-iframe'));
          return;
        }

        const expectedOrigin = effectiveOriginRef.current ?? childOrigin;

        const timer = window.setTimeout(() => {
          cleanup();
          reject(new Error('timeout'));
        }, timeoutMs);

        const listener = (event: MessageEvent) => {
          if (event.source !== iframeWindow) {
            return;
          }
          if (expectedOrigin && event.origin !== expectedOrigin) {
            return;
          }
          const payload = event.data as BridgeChildToParentMessage | null | undefined;
          if (!payload || typeof payload !== 'object' || payload.v !== 1) {
            return;
          }
          if (predicate(payload)) {
            cleanup();
            resolve();
          }
        };

        const cleanup = () => {
          window.clearTimeout(timer);
          window.removeEventListener('message', listener);
        };

        window.addEventListener('message', listener);
      });
    },
    [childOrigin, iframeRef],
  );

  const runComplianceCheck = useCallback(async (): Promise<BridgeComplianceResult> => {
    const failures: string[] = [];

    if (!childOrigin || !iframeRef.current) {
      return { ok: false, failures: ['NO_IFRAME'], checkedAt: Date.now() };
    }

    const originalHref = lastHrefRef.current;

    const recordFailure = (label: string) => {
      if (!failures.includes(label)) {
        failures.push(label);
      }
    };

    if (!helloReceivedRef.current) {
      try {
        await waitForMessage(message => message.t === 'HELLO', 1000);
      } catch (error) {
        recordFailure('HELLO');
      }
    }

    if (!readyReceivedRef.current) {
      try {
        await waitForMessage(message => message.t === 'READY', 3000);
      } catch (error) {
        recordFailure('READY');
      }
    }

    const randomHash = `#bridge-test-${Math.random().toString(16).slice(2)}`;
    const navIssued = sendNav('GO', randomHash);
    if (navIssued) {
      try {
        await waitForMessage(
          message => message.t === 'LOCATION' && typeof message.href === 'string' && message.href.includes(randomHash),
          700,
        );
      } catch (error) {
        recordFailure('SPA hooks');
      }
    } else {
      recordFailure('SPA hooks');
    }

    const backIssued = sendNav('BACK');
    if (backIssued) {
      try {
        await waitForMessage(message => message.t === 'LOCATION', 700);
      } catch (error) {
        recordFailure('BACK/FWD');
      }
    } else {
      recordFailure('BACK/FWD');
    }

    if (originalHref) {
      sendNav('GO', originalHref);
    }

    return { ok: failures.length === 0, failures, checkedAt: Date.now() };
  }, [childOrigin, iframeRef, sendNav, waitForMessage]);

  return {
    state: {
      ...state,
      isSupported: state.isSupported || helloReceivedRef.current,
      isReady: state.isReady || readyReceivedRef.current,
      caps: state.caps,
      lastError: state.lastError,
    },
    childOrigin,
    sendNav,
    sendPing,
    runComplianceCheck,
    resetState,
    requestScreenshot,
  };
};

export default useIframeBridge;
