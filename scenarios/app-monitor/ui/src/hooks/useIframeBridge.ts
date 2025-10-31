import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type {
  BridgeCapability,
  BridgeLogEvent,
  BridgeNetworkEvent,
  BridgeLogStreamState,
  BridgeNetworkStreamState,
  BridgeLogLevel,
  BridgeScreenshotMode,
  BridgeScreenshotOptions,
  BridgeInspectHoverPayload,
  BridgeInspectResultPayload,
} from '@vrooli/iframe-bridge';
import { logger } from '@/services/logger';

type BridgeHelloMessage = {
  v: 1;
  t: 'HELLO';
  appId?: string;
  title?: string;
  caps?: BridgeCapability[];
  logs?: BridgeLogStreamState;
  network?: BridgeNetworkStreamState;
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
  mode?: BridgeScreenshotMode;
  clip?: { x: number; y: number; width: number; height: number };
  error?: string;
};

type BridgeLogEventMessage = {
  v: 1;
  t: 'LOG_EVENT';
  event: BridgeLogEvent;
};

type BridgeLogBatchMessage = {
  v: 1;
  t: 'LOG_BATCH';
  requestId: string;
  events: BridgeLogEvent[];
};

type BridgeLogStateMessage = {
  v: 1;
  t: 'LOG_STATE';
  state: BridgeLogStreamState;
};

type BridgeNetworkEventMessage = {
  v: 1;
  t: 'NETWORK_EVENT';
  event: BridgeNetworkEvent;
};

type BridgeNetworkBatchMessage = {
  v: 1;
  t: 'NETWORK_BATCH';
  requestId: string;
  events: BridgeNetworkEvent[];
};

type BridgeNetworkStateMessage = {
  v: 1;
  t: 'NETWORK_STATE';
  state: BridgeNetworkStreamState;
};

type BridgeInspectHoverMessage = {
  v: 1;
  t: 'INSPECT_HOVER';
  payload: BridgeInspectHoverPayload;
};

type BridgeInspectResultMessage = {
  v: 1;
  t: 'INSPECT_RESULT';
  payload: BridgeInspectResultPayload;
};

type BridgeInspectStateMessage = {
  v: 1;
  t: 'INSPECT_STATE';
  active: boolean;
  reason?: InspectLifecycleReason | 'start';
};

type BridgeInspectCancelMessage = {
  v: 1;
  t: 'INSPECT_CANCEL';
};

type BridgeInspectErrorMessage = {
  v: 1;
  t: 'INSPECT_ERROR';
  error: string;
};

type BridgeChildToParentMessage =
  | BridgeHelloMessage
  | BridgeReadyMessage
  | BridgeLocationMessage
  | BridgeErrorMessage
  | BridgePongMessage
  | BridgeScreenshotResultMessage
  | BridgeLogEventMessage
  | BridgeLogBatchMessage
  | BridgeLogStateMessage
  | BridgeNetworkEventMessage
  | BridgeNetworkBatchMessage
  | BridgeNetworkStateMessage
  | BridgeInspectHoverMessage
  | BridgeInspectResultMessage
  | BridgeInspectStateMessage
  | BridgeInspectCancelMessage
  | BridgeInspectErrorMessage;

type BridgeSnapshotRequestOptions = {
  since?: number;
  afterSeq?: number;
  limit?: number;
};

export type InspectLifecycleReason = 'start' | 'stop' | 'cancel' | 'complete';

export interface BridgeInspectState {
  supported: boolean;
  active: boolean;
  lastReason: InspectLifecycleReason | null;
  hover: BridgeInspectHoverPayload | null;
  result: BridgeInspectResultPayload | null;
  error: string | null;
}

const initialInspectState: BridgeInspectState = {
  supported: false,
  active: false,
  lastReason: null,
  hover: null,
  result: null,
  error: null,
};

type BridgeParentToChildMessage =
  | { v: 1; t: 'NAV'; cmd: 'GO'; to?: string }
  | { v: 1; t: 'NAV'; cmd: 'BACK' }
  | { v: 1; t: 'NAV'; cmd: 'FWD' }
  | { v: 1; t: 'PING'; ts: number }
  | { v: 1; t: 'CAPTURE'; cmd: 'SCREENSHOT'; id: string; options?: BridgeScreenshotOptions }
  | { v: 1; t: 'LOGS'; cmd: 'PULL'; requestId: string; options?: BridgeSnapshotRequestOptions }
  | { v: 1; t: 'LOGS'; cmd: 'SET'; enable?: boolean; streaming?: boolean; levels?: BridgeLogLevel[]; bufferSize?: number }
  | { v: 1; t: 'NETWORK'; cmd: 'PULL'; requestId: string; options?: BridgeSnapshotRequestOptions }
  | { v: 1; t: 'NETWORK'; cmd: 'SET'; enable?: boolean; streaming?: boolean; bufferSize?: number }
  | { v: 1; t: 'INSPECT'; cmd: 'START' }
  | { v: 1; t: 'INSPECT'; cmd: 'STOP' };

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
  requestScreenshot: (options?: BridgeScreenshotOptions) => Promise<{
    data: string;
    width: number;
    height: number;
    note?: string;
    mode?: BridgeScreenshotMode;
    clip?: { x: number; y: number; width: number; height: number };
  }>;
  logState: BridgeLogStreamState | null;
  networkState: BridgeNetworkStreamState | null;
  subscribeLogs: (listener: (event: BridgeLogEvent) => void) => () => void;
  getRecentLogs: () => BridgeLogEvent[];
  requestLogBatch: (options?: BridgeSnapshotRequestOptions) => Promise<BridgeLogEvent[]>;
  configureLogs: (config: { enable?: boolean; streaming?: boolean; levels?: BridgeLogLevel[]; bufferSize?: number }) => boolean;
  subscribeNetwork: (listener: (event: BridgeNetworkEvent) => void) => () => void;
  getRecentNetworkEvents: () => BridgeNetworkEvent[];
  requestNetworkBatch: (options?: BridgeSnapshotRequestOptions) => Promise<BridgeNetworkEvent[]>;
  configureNetwork: (config: { enable?: boolean; streaming?: boolean; bufferSize?: number }) => boolean;
  inspectState: BridgeInspectState;
  startInspect: () => boolean;
  stopInspect: () => boolean;
}

const deriveOrigin = (url: string | null): string | null => {
  if (!url || typeof window === 'undefined') {
    return null;
  }

  try {
    const resolved = new URL(url, window.location.href);
    return resolved.origin;
  } catch (error) {
    logger.warn('[Bridge] Failed to derive origin from URL', { url, error });
    return null;
  }
};

const LOG_BUFFER_LIMIT = 500;
const NETWORK_BUFFER_LIMIT = 300;
const LOG_REQUEST_TIMEOUT_MS = 5_000;
const NETWORK_REQUEST_TIMEOUT_MS = 5_000;

const generateRequestId = (prefix: string): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
};

export const useIframeBridge = ({ iframeRef, previewUrl, onLocation }: UseIframeBridgeOptions): UseIframeBridgeReturn => {
  const [state, setState] = useState<BridgeState>(initialBridgeState);
  const [logState, setLogState] = useState<BridgeLogStreamState | null>(null);
  const [networkState, setNetworkState] = useState<BridgeNetworkStreamState | null>(null);
  const [inspectState, setInspectState] = useState<BridgeInspectState>(initialInspectState);
  const childOrigin = useMemo(() => deriveOrigin(previewUrl), [previewUrl]);
  const lastHrefRef = useRef<string>('');
  const helloReceivedRef = useRef(false);
  const readyReceivedRef = useRef(false);
  const effectiveOriginRef = useRef<string | null>(null);
  const pendingScreenshotRequestsRef = useRef(new Map<string, {
    resolve: (value: {
      data: string;
      width: number;
      height: number;
      note?: string;
      mode?: BridgeScreenshotMode;
      clip?: { x: number; y: number; width: number; height: number };
    }) => void;
    reject: (error: Error) => void;
    timeoutHandle: number;
  }>());
  const pendingLogRequestsRef = useRef(new Map<string, {
    resolve: (events: BridgeLogEvent[]) => void;
    reject: (error: Error) => void;
    timeoutHandle: number;
  }>());
  const pendingNetworkRequestsRef = useRef(new Map<string, {
    resolve: (events: BridgeNetworkEvent[]) => void;
    reject: (error: Error) => void;
    timeoutHandle: number;
  }>());
  const logBufferRef = useRef<BridgeLogEvent[]>([]);
  const networkBufferRef = useRef<BridgeNetworkEvent[]>([]);
  const logListenersRef = useRef(new Set<(event: BridgeLogEvent) => void>());
  const networkListenersRef = useRef(new Set<(event: BridgeNetworkEvent) => void>());
  const supportsLogsRef = useRef(false);
  const supportsNetworkRef = useRef(false);
  const capsRef = useRef<BridgeCapability[]>([]);

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
    pendingLogRequestsRef.current.forEach(({ reject, timeoutHandle }) => {
      window.clearTimeout(timeoutHandle);
      reject(new Error('bridge-reset'));
    });
    pendingLogRequestsRef.current.clear();
    pendingNetworkRequestsRef.current.forEach(({ reject, timeoutHandle }) => {
      window.clearTimeout(timeoutHandle);
      reject(new Error('bridge-reset'));
    });
    pendingNetworkRequestsRef.current.clear();
    logBufferRef.current = [];
    networkBufferRef.current = [];
    supportsLogsRef.current = false;
    supportsNetworkRef.current = false;
    capsRef.current = [];
    setLogState(null);
    setNetworkState(null);
    setState(initialBridgeState);
    setInspectState(initialInspectState);
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
        logger.warn('[Bridge] Message origin differs from expected', { expected: childOrigin, received: event.origin });
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
          const nextCaps = Array.isArray(message.caps) ? message.caps : capsRef.current;
          capsRef.current = nextCaps;
          supportsLogsRef.current = nextCaps.includes('logs');
          supportsNetworkRef.current = nextCaps.includes('network');
          const supportsInspect = nextCaps.includes('inspect');
          if (supportsLogsRef.current) {
            setLogState(message.logs ?? { enabled: false, streaming: false });
          } else {
            setLogState(null);
          }
          if (supportsNetworkRef.current) {
            setNetworkState(message.network ?? { enabled: false, streaming: false });
          } else {
            setNetworkState(null);
          }
          setInspectState({
            ...initialInspectState,
            supported: supportsInspect,
          });
          setState(prev => ({
            ...prev,
            isSupported: true,
            caps: nextCaps,
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
            pending.resolve({
              data: message.data,
              width: message.width,
              height: message.height,
              note: message.note,
              mode: message.mode,
              clip: message.clip,
            });
          } else {
            pending.reject(new Error(message.error || 'screenshot-failed'));
          }
          break;
        }

        case 'LOG_EVENT': {
          if (!message.event) {
            break;
          }
          logBufferRef.current.push(message.event);
          if (logBufferRef.current.length > LOG_BUFFER_LIMIT) {
            logBufferRef.current.splice(0, logBufferRef.current.length - LOG_BUFFER_LIMIT);
          }
          logListenersRef.current.forEach(listener => {
            try {
              listener(message.event);
            } catch (error) {
              logger.warn('[Bridge] Log listener failed', error);
            }
          });
          break;
        }

        case 'LOG_BATCH': {
          const pending = pendingLogRequestsRef.current.get(message.requestId);
          if (!pending) {
            break;
          }
          pendingLogRequestsRef.current.delete(message.requestId);
          window.clearTimeout(pending.timeoutHandle);
          pending.resolve(Array.isArray(message.events) ? message.events : []);
          break;
        }

        case 'LOG_STATE': {
          setLogState(message.state);
          break;
        }

        case 'NETWORK_EVENT': {
          if (!message.event) {
            break;
          }
          networkBufferRef.current.push(message.event);
          if (networkBufferRef.current.length > NETWORK_BUFFER_LIMIT) {
            networkBufferRef.current.splice(0, networkBufferRef.current.length - NETWORK_BUFFER_LIMIT);
          }
          networkListenersRef.current.forEach(listener => {
            try {
              listener(message.event);
            } catch (error) {
              logger.warn('[Bridge] Network listener failed', error);
            }
          });
          break;
        }

        case 'NETWORK_BATCH': {
          const pending = pendingNetworkRequestsRef.current.get(message.requestId);
          if (!pending) {
            break;
          }
          pendingNetworkRequestsRef.current.delete(message.requestId);
          window.clearTimeout(pending.timeoutHandle);
          pending.resolve(Array.isArray(message.events) ? message.events : []);
          break;
        }

        case 'NETWORK_STATE': {
          setNetworkState(message.state);
          break;
        }

        case 'INSPECT_STATE': {
          setInspectState(prev => ({
            ...prev,
            supported: prev.supported || capsRef.current.includes('inspect'),
            active: message.active,
            lastReason: message.reason && message.reason !== 'start'
              ? message.reason as InspectLifecycleReason
              : (message.active ? 'start' : prev.lastReason),
            error: message.active ? null : prev.error,
            hover: message.active ? null : prev.hover,
          }));
          break;
        }

        case 'INSPECT_HOVER': {
          setInspectState(prev => ({
            ...prev,
            supported: prev.supported || capsRef.current.includes('inspect'),
            hover: message.payload,
          }));
          break;
        }

        case 'INSPECT_RESULT': {
          setInspectState(prev => ({
            ...prev,
            supported: prev.supported || capsRef.current.includes('inspect'),
            active: false,
            hover: message.payload,
            result: message.payload,
            lastReason: 'complete',
            error: null,
          }));
          break;
        }

        case 'INSPECT_CANCEL': {
          setInspectState(prev => ({
            ...prev,
            supported: prev.supported || capsRef.current.includes('inspect'),
            active: false,
            lastReason: 'cancel',
            hover: null,
          }));
          break;
        }

        case 'INSPECT_ERROR': {
          setInspectState(prev => ({
            ...prev,
            supported: prev.supported || capsRef.current.includes('inspect'),
            error: message.error,
            active: false,
            hover: null,
          }));
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
        logger.warn('[Bridge] Unable to determine target origin for message', { type: payload.t });
        return false;
      }
      try {
        iframeWindow.postMessage(payload, targetOrigin);
        return true;
      } catch (error) {
        logger.warn('[Bridge] postMessage failed', error);
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

  const startInspect = useCallback(() => {
    if (!capsRef.current.includes('inspect')) {
      return false;
    }
    return postMessage({ v: 1, t: 'INSPECT', cmd: 'START' });
  }, [postMessage]);

  const stopInspect = useCallback(() => {
    if (!capsRef.current.includes('inspect')) {
      return false;
    }
    return postMessage({ v: 1, t: 'INSPECT', cmd: 'STOP' });
  }, [postMessage]);

  const requestScreenshot = useCallback(
    (options?: BridgeScreenshotOptions) => {
      return new Promise<{
        data: string;
        width: number;
        height: number;
        note?: string;
        mode?: BridgeScreenshotMode;
        clip?: { x: number; y: number; width: number; height: number };
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
        const payloadOptions: BridgeScreenshotOptions = { mode: 'viewport', ...(options ?? {}) };
        const sent = postMessage({ v: 1, t: 'CAPTURE', cmd: 'SCREENSHOT', id, options: payloadOptions });
        if (!sent) {
          window.clearTimeout(timeoutHandle);
          pendingScreenshotRequestsRef.current.delete(id);
          reject(new Error('screenshot-unsupported'));
        }
      });
    },
    [iframeRef, postMessage],
  );

  const subscribeLogs = useCallback((listener: (event: BridgeLogEvent) => void) => {
    logListenersRef.current.add(listener);
    return () => {
      logListenersRef.current.delete(listener);
    };
  }, []);

  const getRecentLogs = useCallback(() => {
    return logBufferRef.current.slice();
  }, []);

  const requestLogBatch = useCallback((options?: BridgeSnapshotRequestOptions) => {
    return new Promise<BridgeLogEvent[]>((resolve, reject) => {
      if (!supportsLogsRef.current) {
        reject(new Error('logs-unsupported'));
        return;
      }
      const requestId = generateRequestId('logs');
      const timeoutHandle = window.setTimeout(() => {
        const pending = pendingLogRequestsRef.current.get(requestId);
        if (pending) {
          pendingLogRequestsRef.current.delete(requestId);
          pending.reject(new Error('logs-request-timeout'));
        }
      }, LOG_REQUEST_TIMEOUT_MS);

      pendingLogRequestsRef.current.set(requestId, { resolve, reject, timeoutHandle });
      const sent = postMessage({ v: 1, t: 'LOGS', cmd: 'PULL', requestId, options });
      if (!sent) {
        window.clearTimeout(timeoutHandle);
        pendingLogRequestsRef.current.delete(requestId);
        reject(new Error('logs-request-failed'));
      }
    });
  }, [postMessage]);

  const configureLogs = useCallback((config: { enable?: boolean; streaming?: boolean; levels?: BridgeLogLevel[]; bufferSize?: number }) => {
    if (!supportsLogsRef.current) {
      return false;
    }
    return postMessage({
      v: 1,
      t: 'LOGS',
      cmd: 'SET',
      enable: config.enable,
      streaming: config.streaming,
      levels: config.levels,
      bufferSize: config.bufferSize,
    });
  }, [postMessage]);

  const subscribeNetwork = useCallback((listener: (event: BridgeNetworkEvent) => void) => {
    networkListenersRef.current.add(listener);
    return () => {
      networkListenersRef.current.delete(listener);
    };
  }, []);

  const getRecentNetworkEvents = useCallback(() => {
    return networkBufferRef.current.slice();
  }, []);

  const requestNetworkBatch = useCallback((options?: BridgeSnapshotRequestOptions) => {
    return new Promise<BridgeNetworkEvent[]>((resolve, reject) => {
      if (!supportsNetworkRef.current) {
        reject(new Error('network-unsupported'));
        return;
      }
      const requestId = generateRequestId('network');
      const timeoutHandle = window.setTimeout(() => {
        const pending = pendingNetworkRequestsRef.current.get(requestId);
        if (pending) {
          pendingNetworkRequestsRef.current.delete(requestId);
          pending.reject(new Error('network-request-timeout'));
        }
      }, NETWORK_REQUEST_TIMEOUT_MS);

      pendingNetworkRequestsRef.current.set(requestId, { resolve, reject, timeoutHandle });
      const sent = postMessage({ v: 1, t: 'NETWORK', cmd: 'PULL', requestId, options });
      if (!sent) {
        window.clearTimeout(timeoutHandle);
        pendingNetworkRequestsRef.current.delete(requestId);
        reject(new Error('network-request-failed'));
      }
    });
  }, [postMessage]);

  const configureNetwork = useCallback((config: { enable?: boolean; streaming?: boolean; bufferSize?: number }) => {
    if (!supportsNetworkRef.current) {
      return false;
    }
    return postMessage({
      v: 1,
      t: 'NETWORK',
      cmd: 'SET',
      enable: config.enable,
      streaming: config.streaming,
      bufferSize: config.bufferSize,
    });
  }, [postMessage]);

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

    if (helloReceivedRef.current && !supportsLogsRef.current) {
      recordFailure('CAP_LOGS');
    }

    if (helloReceivedRef.current && !supportsNetworkRef.current) {
      recordFailure('CAP_NETWORK');
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
    logState,
    networkState,
    subscribeLogs,
    getRecentLogs,
    requestLogBatch,
    configureLogs,
    subscribeNetwork,
    getRecentNetworkEvents,
    requestNetworkBatch,
    configureNetwork,
    inspectState,
    startInspect,
    stopInspect,
  };
};

export default useIframeBridge;
