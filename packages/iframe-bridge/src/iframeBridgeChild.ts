export type BridgeCapability =
  | 'history'
  | 'hash'
  | 'title'
  | 'deeplink'
  | 'resize'
  | 'screenshot'
  | 'logs'
  | 'network';

export type BridgeLogLevel = 'log' | 'info' | 'warn' | 'error' | 'debug';

export interface BridgeLogEvent {
  seq: number;
  ts: number;
  level: BridgeLogLevel;
  args: unknown[];
  source: 'console' | 'runtime';
  message?: string;
  context?: Record<string, unknown>;
}

export interface BridgeLogStreamState {
  enabled: boolean;
  streaming: boolean;
  levels?: BridgeLogLevel[];
}

export type NetworkEventKind = 'fetch' | 'xhr';

export interface BridgeNetworkEvent {
  seq: number;
  ts: number;
  kind: NetworkEventKind;
  requestId: string;
  method: string;
  url: string;
  status?: number;
  ok?: boolean;
  durationMs?: number;
  error?: string;
  readyState?: number;
  responseType?: XMLHttpRequestResponseType;
}

export interface BridgeNetworkStreamState {
  enabled: boolean;
  streaming: boolean;
}

type LogCaptureOption = boolean | LogCaptureOptions | undefined;
type NetworkCaptureOption = boolean | NetworkCaptureOptions | undefined;

export interface LogCaptureOptions {
  enabled?: boolean;
  streaming?: boolean;
  levels?: BridgeLogLevel[];
  bufferSize?: number;
}

export interface NetworkCaptureOptions {
  enabled?: boolean;
  streaming?: boolean;
  bufferSize?: number;
}

export interface BridgeChildOptions {
  parentOrigin?: string;
  appId?: string;
  onNav?: (href: string) => void;
  captureLogs?: LogCaptureOption;
  captureNetwork?: NetworkCaptureOption;
}

export interface BridgeChildController {
  notify: () => void;
  dispose: () => void;
}

declare global {
  interface Window {
    __vrooliBridgeChildInstalled?: boolean;
    html2canvas?: Html2CanvasFn;
  }

  interface XMLHttpRequest {
    __vrooliBridgeMeta?: {
      method: string;
      url: string;
      requestId: string;
      startTime: number;
      completed?: boolean;
    };
  }
}

type Html2CanvasFn = (element: HTMLElement, options?: Record<string, unknown>) => Promise<HTMLCanvasElement>;

type PostFn = (payload: Record<string, unknown>) => void;

type SnapshotOptions = {
  since?: number;
  afterSeq?: number;
  limit?: number;
};

type LogSetCommand = {
  enable?: boolean;
  streaming?: boolean;
  levels?: BridgeLogLevel[];
  bufferSize?: number;
};

type NetworkSetCommand = {
  enable?: boolean;
  streaming?: boolean;
  bufferSize?: number;
};

interface RingBuffer<T> {
  push: (value: T) => void;
  values: () => T[];
  setLimit: (limit: number) => void;
  clear: () => void;
}

interface NormalizedLogOptions {
  supported: boolean;
  enabled: boolean;
  streaming: boolean;
  bufferSize: number;
  levels?: BridgeLogLevel[];
}

interface NormalizedNetworkOptions {
  supported: boolean;
  enabled: boolean;
  streaming: boolean;
  bufferSize: number;
}

interface LogCaptureHandle {
  supported: boolean;
  recordConsole: (level: BridgeLogLevel, args: unknown[]) => void;
  recordRuntimeError: (message: string, context?: Record<string, unknown>) => void;
  snapshot: (options?: SnapshotOptions) => BridgeLogEvent[];
  setConfig: (command: LogSetCommand) => BridgeLogStreamState;
  emitState: () => void;
  getState: () => BridgeLogStreamState;
  dispose: () => void;
}

interface NetworkCaptureHandle {
  supported: boolean;
  snapshot: (options?: SnapshotOptions) => BridgeNetworkEvent[];
  setConfig: (command: NetworkSetCommand) => BridgeNetworkStreamState;
  emitState: () => void;
  getState: () => BridgeNetworkStreamState;
  dispose: () => void;
}

const DEFAULT_LOG_BUFFER_SIZE = 500;
const DEFAULT_NETWORK_BUFFER_SIZE = 250;
const MIN_BUFFER_SIZE = 50;
const SERIALIZE_MAX_DEPTH = 3;
const SERIALIZE_MAX_KEYS = 20;
const SERIALIZE_MAX_STRING = 10_000;
const LOG_LEVELS: BridgeLogLevel[] = ['log', 'info', 'warn', 'error', 'debug'];

const loadHtml2Canvas = (() => {
  let loader: Promise<Html2CanvasFn> | null = null;
  return (): Promise<Html2CanvasFn> => {
    if (typeof window !== 'undefined' && typeof window.html2canvas === 'function') {
      return Promise.resolve(window.html2canvas);
    }

    if (!loader) {
      loader = new Promise<Html2CanvasFn>((resolve, reject) => {
        const existing = document.querySelector<HTMLScriptElement>('script[data-html2canvas="true"]');
        if (existing) {
          existing.addEventListener('load', () => {
            if (typeof window.html2canvas === 'function') {
              resolve(window.html2canvas);
            } else {
              reject(new Error('html2canvas failed to initialize'));
            }
          });
          existing.addEventListener('error', () => reject(new Error('Failed to load html2canvas script')));
          return;
        }

        const script = document.createElement('script');
        script.src = 'https://cdn.jsdelivr.net/npm/html2canvas@1.4.1/dist/html2canvas.min.js';
        script.async = true;
        script.crossOrigin = 'anonymous';
        script.dataset.html2canvas = 'true';
        script.onload = () => {
          if (typeof window.html2canvas === 'function') {
            resolve(window.html2canvas);
          } else {
            reject(new Error('html2canvas failed to initialize'));
          }
        };
        script.onerror = () => reject(new Error('Failed to load html2canvas script'));
        document.head.appendChild(script);
      });
    }

    return loader;
  };
})();

const inferParentOrigin = (): string | null => {
  try {
    if (document.referrer) {
      const referrer = new URL(document.referrer);
      return referrer.origin;
    }
  } catch (error) {
    console.warn('[BridgeChild] Failed to parse document.referrer', error);
  }
  return null;
};

const buildLocationPayload = () => ({
  v: 1 as const,
  t: 'LOCATION' as const,
  href: window.location.href,
  path: `${window.location.pathname}${window.location.search}${window.location.hash}`,
  title: document.title,
  canGoBack: true,
  canGoFwd: true,
});

const performanceNow = (): number => {
  if (typeof performance !== 'undefined' && typeof performance.now === 'function') {
    return performance.now();
  }
  return Date.now();
};

const elapsedMs = (start: number): number => {
  const diff = performanceNow() - start;
  return diff < 0 ? 0 : diff;
};

const sanitizeBufferSize = (value: number | undefined, fallback: number): number => {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return Math.max(MIN_BUFFER_SIZE, fallback);
  }
  return Math.max(MIN_BUFFER_SIZE, Math.floor(value));
};

const createRingBuffer = <T>(limit: number): RingBuffer<T> => {
  let max = Math.max(MIN_BUFFER_SIZE, limit);
  let data: T[] = [];
  return {
    push: (value: T) => {
      data.push(value);
      if (data.length > max) {
        data.splice(0, data.length - max);
      }
    },
    values: () => data.slice(),
    setLimit: (next: number) => {
      max = Math.max(MIN_BUFFER_SIZE, next);
      if (data.length > max) {
        data.splice(0, data.length - max);
      }
    },
    clear: () => {
      data = [];
    },
  };
};

const serializeBridgeValue = (value: unknown, depth = 0, seen?: WeakSet<object>): unknown => {
  if (value === null || typeof value === 'number' || typeof value === 'boolean') {
    return value;
  }

  if (typeof value === 'undefined') {
    return undefined;
  }

  if (typeof value === 'string') {
    return value.length > SERIALIZE_MAX_STRING ? `${value.slice(0, SERIALIZE_MAX_STRING)}…` : value;
  }

  if (typeof value === 'bigint') {
    return value.toString();
  }

  if (typeof value === 'symbol') {
    return value.toString();
  }

  if (typeof value === 'function') {
    return `[Function ${value.name || 'anonymous'}]`;
  }

  if (value instanceof Error) {
    return {
      name: value.name,
      message: value.message,
      stack: value.stack,
    };
  }

  if (value instanceof Date) {
    return value.toISOString();
  }

  if (typeof URL !== 'undefined' && value instanceof URL) {
    return value.toString();
  }

  if (typeof Node !== 'undefined' && value instanceof Node) {
    return `[DOM ${value.nodeName}]`;
  }

  if (Array.isArray(value)) {
    if (depth >= SERIALIZE_MAX_DEPTH) {
      return `[Array(${value.length})]`;
    }
    const seenSet = seen ?? new WeakSet<object>();
    const limited = value.slice(0, SERIALIZE_MAX_KEYS).map(item => serializeBridgeValue(item, depth + 1, seenSet));
    if (value.length > SERIALIZE_MAX_KEYS) {
      limited.push(`…${value.length - SERIALIZE_MAX_KEYS} more`);
    }
    return limited;
  }

  if (value && typeof value === 'object') {
    const obj = value as Record<string, unknown>;
    const seenSet = seen ?? new WeakSet<object>();
    if (seenSet.has(obj)) {
      return '[Circular]';
    }
    seenSet.add(obj);
    if (depth >= SERIALIZE_MAX_DEPTH) {
      return '[Object]';
    }
    const output: Record<string, unknown> = {};
    let count = 0;
    for (const [key, val] of Object.entries(obj)) {
      output[key] = serializeBridgeValue(val, depth + 1, seenSet);
      count += 1;
      if (count >= SERIALIZE_MAX_KEYS) {
        output.__truncated__ = true;
        break;
      }
    }
    return output;
  }

  return value;
};

const describeError = (error: unknown): string => {
  if (error instanceof Error) {
    return error.message || error.name || 'Unknown error';
  }
  if (typeof error === 'string') {
    return error;
  }
  try {
    return JSON.stringify(error);
  } catch {
    return String(error);
  }
};

const createRequestId = (prefix: string): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
};

const normalizeLogOptions = (option: LogCaptureOption): NormalizedLogOptions => {
  if (option === false) {
    return { supported: false, enabled: false, streaming: false, bufferSize: DEFAULT_LOG_BUFFER_SIZE };
  }

  if (option === true || option === undefined) {
    return { supported: true, enabled: true, streaming: false, bufferSize: DEFAULT_LOG_BUFFER_SIZE };
  }

  const enabled = option.enabled ?? true;
  const streaming = enabled ? option.streaming ?? false : false;
  const bufferSize = sanitizeBufferSize(option.bufferSize, DEFAULT_LOG_BUFFER_SIZE);

  let levels: BridgeLogLevel[] | undefined;
  if (Array.isArray(option.levels) && option.levels.length > 0) {
    levels = option.levels.filter((level): level is BridgeLogLevel => LOG_LEVELS.includes(level as BridgeLogLevel));
  }

  return { supported: true, enabled, streaming, bufferSize, levels };
};

const normalizeNetworkOptions = (option: NetworkCaptureOption): NormalizedNetworkOptions => {
  if (option === false) {
    return { supported: false, enabled: false, streaming: false, bufferSize: DEFAULT_NETWORK_BUFFER_SIZE };
  }

  if (option === true || option === undefined) {
    return { supported: true, enabled: true, streaming: false, bufferSize: DEFAULT_NETWORK_BUFFER_SIZE };
  }

  const enabled = option.enabled ?? true;
  const streaming = enabled ? option.streaming ?? false : false;
  const bufferSize = sanitizeBufferSize(option.bufferSize, DEFAULT_NETWORK_BUFFER_SIZE);

  return { supported: true, enabled, streaming, bufferSize };
};

const setupLogCapture = (post: PostFn, options: NormalizedLogOptions): LogCaptureHandle | null => {
  if (!options.supported) {
    return null;
  }

  const buffer = createRingBuffer<BridgeLogEvent>(options.bufferSize);
  let bufferLimit = options.bufferSize;
  let seq = 0;
  let enabled = options.enabled;
  let streaming = options.streaming;
  let levelFilter = options.levels ? new Set(options.levels) : undefined;

  const shouldRecordLevel = (level: BridgeLogLevel): boolean => {
    if (!enabled) {
      return false;
    }
    if (!levelFilter || levelFilter.size === 0) {
      return true;
    }
    return levelFilter.has(level);
  };

  const recordEvent = (
    level: BridgeLogLevel,
    args: unknown[],
    source: BridgeLogEvent['source'],
    message?: string,
    context?: Record<string, unknown>,
  ) => {
    if (!shouldRecordLevel(level)) {
      return;
    }
    const event: BridgeLogEvent = {
      seq: ++seq,
      ts: Date.now(),
      level,
      args: args.map(arg => serializeBridgeValue(arg)),
      source,
      message,
      context,
    };
    buffer.push(event);
    if (streaming) {
      post({ v: 1, t: 'LOG_EVENT', event });
    }
  };

  const originalConsole: Partial<Record<BridgeLogLevel, (...args: unknown[]) => void>> = {};

  const consoleAny = console as Record<BridgeLogLevel, ((...args: unknown[]) => void) | undefined>;

  LOG_LEVELS.forEach(level => {
    const existing = consoleAny[level];
    if (typeof existing !== 'function') {
      return;
    }
    originalConsole[level] = existing.bind(console);
    consoleAny[level] = (...args: unknown[]) => {
      try {
        recordEvent(level, args, 'console');
      } catch (error) {
        const fallback = originalConsole.warn ?? originalConsole.log;
        fallback?.('[BridgeChild] Failed to record console event', error);
      }
      originalConsole[level]?.(...args);
    };
  });

  const handleWindowError = (event: ErrorEvent) => {
    recordEvent('error', [event.message], 'runtime', event.message, {
      filename: event.filename,
      lineno: event.lineno,
      colno: event.colno,
      error: event.error ? serializeBridgeValue(event.error) : undefined,
    });
  };

  const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
    recordEvent('error', [event.reason], 'runtime', undefined, {
      unhandledRejection: true,
      reason: serializeBridgeValue(event.reason),
    });
  };

  window.addEventListener('error', handleWindowError);
  window.addEventListener('unhandledrejection', handleUnhandledRejection);

  const snapshot = (options?: SnapshotOptions): BridgeLogEvent[] => {
    let events = buffer.values();
    if (typeof options?.afterSeq === 'number') {
      events = events.filter(event => event.seq > options.afterSeq!);
    }
    if (typeof options?.since === 'number') {
      events = events.filter(event => event.ts >= options.since!);
    }
    if (typeof options?.limit === 'number' && options.limit > 0 && events.length > options.limit) {
      events = events.slice(events.length - options.limit);
    }
    return events;
  };

  const getState = (): BridgeLogStreamState => ({
    enabled,
    streaming,
    levels: levelFilter && levelFilter.size > 0 ? Array.from(levelFilter) : undefined,
  });

  const setConfig = (command: LogSetCommand): BridgeLogStreamState => {
    if (typeof command.enable === 'boolean') {
      enabled = command.enable;
    }
    if (typeof command.streaming === 'boolean') {
      streaming = command.streaming;
    }
    if (typeof command.bufferSize === 'number') {
      bufferLimit = sanitizeBufferSize(command.bufferSize, bufferLimit);
      buffer.setLimit(bufferLimit);
    }
    if (Array.isArray(command.levels)) {
      if (command.levels.length === 0) {
        levelFilter = undefined;
      } else {
        const filtered = command.levels.filter((level): level is BridgeLogLevel => LOG_LEVELS.includes(level as BridgeLogLevel));
        levelFilter = filtered.length > 0 ? new Set(filtered) : undefined;
      }
    }
    return getState();
  };

  const emitState = () => {
    post({ v: 1, t: 'LOG_STATE', state: getState() });
  };

  const dispose = () => {
    LOG_LEVELS.forEach(level => {
      if (originalConsole[level]) {
        consoleAny[level] = originalConsole[level]!;
      }
    });
    window.removeEventListener('error', handleWindowError);
    window.removeEventListener('unhandledrejection', handleUnhandledRejection);
    buffer.clear();
  };

  return {
    supported: true,
    recordConsole: (level, args) => recordEvent(level, args, 'console'),
    recordRuntimeError: (message, context) => recordEvent('error', [message], 'runtime', message, context),
    snapshot,
    setConfig,
    emitState,
    getState,
    dispose,
  };
};

const setupNetworkCapture = (post: PostFn, options: NormalizedNetworkOptions): NetworkCaptureHandle | null => {
  if (!options.supported) {
    return null;
  }

  const buffer = createRingBuffer<BridgeNetworkEvent>(options.bufferSize);
  let bufferLimit = options.bufferSize;
  let seq = 0;
  let enabled = options.enabled;
  let streaming = options.streaming;

  const recordEvent = (event: Omit<BridgeNetworkEvent, 'seq' | 'ts'> & { ts?: number }) => {
    if (!enabled) {
      return;
    }
    const entry: BridgeNetworkEvent = {
      ...event,
      seq: ++seq,
      ts: event.ts ?? Date.now(),
    };
    buffer.push(entry);
    if (streaming) {
      post({ v: 1, t: 'NETWORK_EVENT', event: entry });
    }
  };

  const snapshot = (options?: SnapshotOptions): BridgeNetworkEvent[] => {
    let events = buffer.values();
    if (typeof options?.afterSeq === 'number') {
      events = events.filter(event => event.seq > options.afterSeq!);
    }
    if (typeof options?.since === 'number') {
      events = events.filter(event => event.ts >= options.since!);
    }
    if (typeof options?.limit === 'number' && options.limit > 0 && events.length > options.limit) {
      events = events.slice(events.length - options.limit);
    }
    return events;
  };

  const getState = (): BridgeNetworkStreamState => ({ enabled, streaming });

  const setConfig = (command: NetworkSetCommand): BridgeNetworkStreamState => {
    if (typeof command.enable === 'boolean') {
      enabled = command.enable;
    }
    if (typeof command.streaming === 'boolean') {
      streaming = command.streaming;
    }
    if (typeof command.bufferSize === 'number') {
      bufferLimit = sanitizeBufferSize(command.bufferSize, bufferLimit);
      buffer.setLimit(bufferLimit);
    }
    return getState();
  };

  const emitState = () => {
    post({ v: 1, t: 'NETWORK_STATE', state: getState() });
  };

  const originalFetch = typeof window.fetch === 'function' ? window.fetch.bind(window) : undefined;

  if (originalFetch) {
    window.fetch = async (...args: Parameters<typeof window.fetch>) => {
      const requestId = createRequestId('fetch');
      const { method, url } = (() => {
        try {
          if (args[0] instanceof Request) {
            return { method: args[0].method || 'GET', url: args[0].url };
          }
          const init = args[1] ?? {};
          const method = typeof init.method === 'string' ? init.method : 'GET';
          return { method, url: typeof args[0] === 'string' ? args[0] : String(args[0]) };
        } catch {
          return { method: 'GET', url: 'unknown' };
        }
      })();
      const upperMethod = method?.toUpperCase?.() || 'GET';
      const start = performanceNow();
      try {
        const response = await originalFetch(...args);
        recordEvent({
          kind: 'fetch',
          requestId,
          method: upperMethod,
          url,
          status: response.status,
          ok: response.ok,
          durationMs: Math.round(elapsedMs(start)),
        });
        return response;
      } catch (error) {
        recordEvent({
          kind: 'fetch',
          requestId,
          method: upperMethod,
          url,
          ok: false,
          error: describeError(error),
          durationMs: Math.round(elapsedMs(start)),
        });
        throw error;
      }
    };
  }

  const originalXHROpen = XMLHttpRequest.prototype.open;
  const originalXHRSend = XMLHttpRequest.prototype.send;

  XMLHttpRequest.prototype.open = function patchedOpen(
    this: XMLHttpRequest,
    method: string,
    url: string | URL,
    async?: boolean,
    username?: string | null,
    password?: string | null,
  ) {
    this.__vrooliBridgeMeta = {
      method: typeof method === 'string' ? method.toUpperCase() : 'GET',
      url: typeof url === 'string' ? url : url?.toString() ?? 'unknown',
      requestId: createRequestId('xhr'),
      startTime: 0,
    };
    const normalizedAsync = typeof async === 'boolean' ? async : true;
    return originalXHROpen.call(this, method, url, normalizedAsync, username, password);
  };

  XMLHttpRequest.prototype.send = function patchedSend(this: XMLHttpRequest, body?: Document | XMLHttpRequestBodyInit | null) {
    if (!this.__vrooliBridgeMeta) {
      this.__vrooliBridgeMeta = {
        method: 'GET',
        url: 'unknown',
        requestId: createRequestId('xhr'),
        startTime: 0,
      };
    }
    const meta = this.__vrooliBridgeMeta;
    meta.startTime = performanceNow();
    meta.completed = false;

    const finalize = (error?: string) => {
      if (meta.completed) {
        return;
      }
      meta.completed = true;
      if (error) {
        recordEvent({
          kind: 'xhr',
          requestId: meta.requestId,
          method: meta.method,
          url: meta.url,
          ok: false,
          error,
          durationMs: Math.round(elapsedMs(meta.startTime)),
          readyState: this.readyState,
          responseType: this.responseType,
        });
      } else {
        const status = this.status;
        recordEvent({
          kind: 'xhr',
          requestId: meta.requestId,
          method: meta.method,
          url: meta.url,
          status,
          ok: status >= 200 && status < 400,
          durationMs: Math.round(elapsedMs(meta.startTime)),
          readyState: this.readyState,
          responseType: this.responseType,
        });
      }
    };

    this.addEventListener('loadend', () => finalize(), { once: true });
    this.addEventListener('error', () => finalize('error'), { once: true });
    this.addEventListener('abort', () => finalize('aborted'), { once: true });
    this.addEventListener('timeout', () => finalize('timeout'), { once: true });

    return originalXHRSend.apply(this, [body]);
  };

  const dispose = () => {
    buffer.clear();
    if (originalFetch) {
      window.fetch = originalFetch;
    }
    XMLHttpRequest.prototype.open = originalXHROpen;
    XMLHttpRequest.prototype.send = originalXHRSend;
  };

  return {
    supported: true,
    snapshot,
    setConfig,
    emitState,
    getState,
    dispose,
  };
};

export function initIframeBridgeChild(options: BridgeChildOptions = {}): BridgeChildController {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return {
      notify: () => undefined,
      dispose: () => undefined,
    };
  }

  if (window.parent === window) {
    return {
      notify: () => undefined,
      dispose: () => undefined,
    };
  }

  if (window.__vrooliBridgeChildInstalled) {
    return {
      notify: () => undefined,
      dispose: () => undefined,
    };
  }

  const caps: BridgeCapability[] = ['history', 'hash', 'title', 'deeplink', 'screenshot'];
  let resolvedOrigin = options.parentOrigin ?? inferParentOrigin() ?? '*';

  const post: PostFn = payload => {
    try {
      window.parent.postMessage(payload, resolvedOrigin);
    } catch (error) {
      console.warn('[BridgeChild] postMessage failed', error);
    }
  };

  const logCapture = setupLogCapture(post, normalizeLogOptions(options.captureLogs));
  if (logCapture?.supported) {
    caps.push('logs');
  }

  const networkCapture = setupNetworkCapture(post, normalizeNetworkOptions(options.captureNetwork));
  if (networkCapture?.supported) {
    caps.push('network');
  }

  const notify = () => {
    const payload = buildLocationPayload();
    post(payload);
    options.onNav?.(payload.href);
  };

  const handleMessage = (event: MessageEvent) => {
    if (resolvedOrigin !== '*' && event.origin !== resolvedOrigin) {
      return;
    }
    if (resolvedOrigin === '*' && event.origin) {
      resolvedOrigin = event.origin;
    }

    const message = event.data;
    if (!message || typeof message !== 'object' || message.v !== 1) {
      return;
    }

    if (message.t === 'NAV') {
      try {
        if (message.cmd === 'BACK') {
          history.back();
        } else if (message.cmd === 'FWD') {
          history.forward();
        } else if (message.cmd === 'GO' && typeof message.to === 'string') {
          const resolved = new URL(message.to, window.location.href);
          if (resolved.origin !== window.location.origin) {
            window.location.assign(resolved.href);
            return;
          }
          history.pushState({}, '', `${resolved.pathname}${resolved.search}${resolved.hash}`);
          window.dispatchEvent(new PopStateEvent('popstate', { state: history.state }));
        }
        notify();
      } catch (error) {
        post({ v: 1, t: 'ERROR', code: 'NAV_FAILED', detail: String((error as Error)?.message ?? error) });
      }
      return;
    }

    if (message.t === 'PING' && typeof message.ts === 'number') {
      post({ v: 1, t: 'PONG', ts: message.ts });
      return;
    }

    if (message.t === 'CAPTURE' && message.cmd === 'SCREENSHOT' && typeof message.id === 'string') {
      const capture = async () => {
        try {
          const html2canvas = await loadHtml2Canvas();
          const target = document.documentElement as HTMLElement;
          const scale = typeof message.options === 'object' && typeof message.options?.scale === 'number'
            ? message.options.scale
            : window.devicePixelRatio || 1;
          const canvas = await html2canvas(target, {
            scale,
            logging: false,
            useCORS: true,
          });
          const dataUrl = canvas.toDataURL('image/png');
          const base64 = dataUrl.replace(/^data:image\/png;base64,/, '');
          post({
            v: 1,
            t: 'SCREENSHOT_RESULT',
            id: message.id,
            ok: true,
            data: base64,
            width: canvas.width,
            height: canvas.height,
          });
        } catch (error) {
          post({
            v: 1,
            t: 'SCREENSHOT_RESULT',
            id: message.id,
            ok: false,
            error: (error as Error)?.message ?? String(error),
          });
        }
      };
      void capture();
      return;
    }

    if (message.t === 'LOGS') {
      if (!logCapture) {
        return;
      }
      if (message.cmd === 'PULL' && typeof message.requestId === 'string') {
        const events = logCapture.snapshot(message.options);
        post({ v: 1, t: 'LOG_BATCH', requestId: message.requestId, events });
      } else if (message.cmd === 'SET') {
        const state = logCapture.setConfig(message);
        post({ v: 1, t: 'LOG_STATE', state });
      }
      return;
    }

    if (message.t === 'NETWORK') {
      if (!networkCapture) {
        return;
      }
      if (message.cmd === 'PULL' && typeof message.requestId === 'string') {
        const events = networkCapture.snapshot(message.options);
        post({ v: 1, t: 'NETWORK_BATCH', requestId: message.requestId, events });
      } else if (message.cmd === 'SET') {
        const state = networkCapture.setConfig(message);
        post({ v: 1, t: 'NETWORK_STATE', state });
      }
      return;
    }
  };

  const interceptHistory = () => {
    const originalPush = history.pushState;
    const originalReplace = history.replaceState;

    history.pushState = function pushState(...args) {
      originalPush.apply(history, args as any);
      notify();
    };

    history.replaceState = function replaceState(...args) {
      originalReplace.apply(history, args as any);
      notify();
    };
  };

  const setupObservers = () => {
    window.addEventListener('message', handleMessage);
    window.addEventListener('popstate', notify);
    window.addEventListener('hashchange', notify);

    if (document.readyState === 'complete') {
      notify();
    } else {
      window.addEventListener('load', notify, { once: true });
    }

    const titleElement = document.querySelector('title') || document.head;
    const observer = new MutationObserver(() => notify());
    observer.observe(titleElement, { childList: true, subtree: true });
    return observer;
  };

  window.__vrooliBridgeChildInstalled = true;

  post({
    v: 1,
    t: 'HELLO',
    appId: options.appId,
    title: document.title,
    caps,
    logs: logCapture ? logCapture.getState() : undefined,
    network: networkCapture ? networkCapture.getState() : undefined,
  });

  interceptHistory();
  const observer = setupObservers();

  queueMicrotask(() => {
    post({ v: 1, t: 'READY' });
    logCapture?.emitState();
    networkCapture?.emitState();
  });

  return {
    notify,
    dispose: () => {
      observer.disconnect();
      window.removeEventListener('message', handleMessage);
      window.removeEventListener('popstate', notify);
      window.removeEventListener('hashchange', notify);
      window.__vrooliBridgeChildInstalled = false;
      logCapture?.dispose();
      networkCapture?.dispose();
    },
  };
}
