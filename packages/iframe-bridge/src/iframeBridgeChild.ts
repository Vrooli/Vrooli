export type BridgeCapability =
  | 'history'
  | 'hash'
  | 'title'
  | 'deeplink'
  | 'resize'
  | 'screenshot'
  | 'logs'
  | 'network'
  | 'inspect';

export type BridgeLogLevel = 'log' | 'info' | 'warn' | 'error' | 'debug';

export type BridgeScreenshotMode = 'viewport' | 'full-page' | 'clip';

export interface BridgeScreenshotOptions {
  scale?: number;
  mode?: BridgeScreenshotMode;
  clip?: Rect | null;
  selector?: string | null;
  backgroundColor?: string | null;
}

export interface BridgeInspectRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface BridgeInspectElementMeta {
  tag: string;
  id?: string;
  classes?: string[];
  selector?: string;
  role?: string;
  ariaLabel?: string;
  ariaDescription?: string;
  title?: string;
  text?: string;
  label?: string;
}

export interface BridgeInspectAncestorMeta extends BridgeInspectElementMeta {
  rect?: BridgeInspectRect;
  documentRect?: BridgeInspectRect;
  depth?: number;
}

export interface BridgeInspectHoverPayload {
  rect: BridgeInspectRect;
  documentRect: BridgeInspectRect;
  meta: BridgeInspectElementMeta;
  pointerType?: string;
  ancestors?: BridgeInspectAncestorMeta[];
  selectedAncestorIndex?: number;
}

export interface BridgeInspectResultPayload extends BridgeInspectHoverPayload {
  method: 'pointer' | 'keyboard';
}

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

interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

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

const clampClipToCanvas = (clip: Rect, canvasWidth: number, canvasHeight: number): Rect | null => {
  const x = Math.min(Math.max(Math.round(clip.x), 0), canvasWidth);
  const y = Math.min(Math.max(Math.round(clip.y), 0), canvasHeight);
  const remainingWidth = canvasWidth - x;
  const remainingHeight = canvasHeight - y;
  if (remainingWidth <= 0 || remainingHeight <= 0) {
    return null;
  }
  const width = Math.max(1, Math.min(Math.round(clip.width), remainingWidth));
  const height = Math.max(1, Math.min(Math.round(clip.height), remainingHeight));
  if (width <= 0 || height <= 0) {
    return null;
  }
  return { x, y, width, height };
};

const clampNumber = (value: number, min: number, max: number): number => {
  if (!Number.isFinite(value)) {
    return min;
  }
  if (value < min) {
    return min;
  }
  if (value > max) {
    return max;
  }
  return value;
};

const scaleRect = (rect: Rect, factor: number): Rect => {
  return {
    x: rect.x * factor,
    y: rect.y * factor,
    width: rect.width * factor,
    height: rect.height * factor,
  };
};

const sanitizeClipRect = (candidate: Rect | null | undefined): Rect | null => {
  if (!candidate) {
    return null;
  }
  const x = Number(candidate.x);
  const y = Number(candidate.y);
  const width = Number(candidate.width);
  const height = Number(candidate.height);
  if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(width) || !Number.isFinite(height)) {
    return null;
  }
  if (width <= 0 || height <= 0) {
    return null;
  }
  return {
    x,
    y,
    width,
    height,
  };
};

const findElementForSelector = (selector: string | null | undefined): HTMLElement | null => {
  if (!selector) {
    return null;
  }
  const trimmed = selector.trim();
  if (!trimmed) {
    return null;
  }
  try {
    const node = document.querySelector(trimmed);
    return node instanceof HTMLElement ? node : null;
  } catch (error) {
    console.warn('[BridgeChild] Invalid selector for screenshot capture', error);
    return null;
  }
};

const truncateText = (value: string | null | undefined, limit: number): string | undefined => {
  if (!value) {
    return undefined;
  }
  const normalized = value.replace(/\s+/g, ' ').trim();
  if (!normalized) {
    return undefined;
  }
  if (normalized.length <= limit) {
    return normalized;
  }
  return `${normalized.slice(0, Math.max(0, limit - 1))}…`;
};

const cssEscape = (value: string): string => {
  if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') {
    try {
      return CSS.escape(value);
    } catch {
      // fall through to manual escaping
    }
  }
  return value.replace(/([^_a-zA-Z0-9-])/g, match => `\\${match}`);
};

const buildCssSelector = (element: Element): string | undefined => {
  const parts: string[] = [];
  let current: Element | null = element;
  let depth = 0;
  const maxDepth = 6;
  while (current && depth < maxDepth) {
    const currentElement = current as Element;
    const tag = currentElement.tagName?.toLowerCase?.();
    if (!tag) {
      break;
    }

    if (currentElement.id) {
      parts.unshift(`${tag}#${cssEscape(currentElement.id)}`);
      break;
    }

    let part = tag;
    const classNames = Array.from(currentElement.classList?.values?.() ?? []).filter((name): name is string => Boolean(name)).slice(0, 3);
    if (classNames.length > 0) {
      part += classNames.map(name => `.${cssEscape(name)}`).join('');
    }

    const parent = currentElement.parentElement;
    if (parent) {
      const siblings = (Array.from(parent.children) as Element[]).filter(node => node.tagName === currentElement.tagName);
      if (siblings.length > 1) {
        const index = siblings.indexOf(currentElement);
        if (index >= 0) {
          part += `:nth-of-type(${index + 1})`;
        }
      }
    }

    parts.unshift(part);
    depth += 1;
    current = currentElement.parentElement;
    if (!current || current === document.documentElement) {
      break;
    }
  }

  if (parts.length === 0) {
    return undefined;
  }

  return parts.join(' > ');
};

const collectElementMeta = (element: Element): BridgeInspectElementMeta => {
  const tag = element.tagName?.toLowerCase?.() ?? 'unknown';
  const id = element.id?.trim() || undefined;
  const classList = Array.from(element.classList || []).filter(Boolean);
  const classes = classList.length > 0 ? classList.slice(0, 5) : undefined;
  const selector = buildCssSelector(element);
  const role = element.getAttribute?.('role')?.trim() || undefined;
  const ariaLabel = element.getAttribute?.('aria-label')?.trim() || undefined;
  const ariaDescription = element.getAttribute?.('aria-description')?.trim() || undefined;
  const title = element.getAttribute?.('title')?.trim() || undefined;
  const text = truncateText(element.textContent, 180);

  const label = (() => {
    if (ariaLabel) {
      return ariaLabel;
    }
    if (title) {
      return title;
    }
    if (text) {
      return text;
    }
    if (id) {
      return `#${id}`;
    }
    if (classes && classes.length > 0) {
      return `${tag}.${classes[0]}`;
    }
    return tag;
  })();

  return {
    tag,
    id,
    classes,
    selector,
    role,
    ariaLabel,
    ariaDescription,
    title,
    text,
    label,
  };
};

const MAX_INSPECT_ANCESTOR_DEPTH = 12;

type InspectAncestryEntry = {
  element: Element;
  rect: BridgeInspectRect;
  documentRect: BridgeInspectRect;
  meta: BridgeInspectAncestorMeta;
  depth: number;
};

const collectElementAncestry = (element: Element): InspectAncestryEntry[] => {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return [];
  }

  const entries: InspectAncestryEntry[] = [];
  const seen = new Set<Element>();
  let current: Element | null = element;
  let depth = 0;

  while (current && depth < MAX_INSPECT_ANCESTOR_DEPTH) {
    if (seen.has(current)) {
      break;
    }
    seen.add(current);

    if (typeof current.getBoundingClientRect !== 'function') {
      current = current.parentElement;
      depth += 1;
      continue;
    }

    const rect = current.getBoundingClientRect();
    if (!rect || rect.width <= 0 || rect.height <= 0) {
      current = current.parentElement;
      depth += 1;
      continue;
    }

    const viewportRect: BridgeInspectRect = {
      x: rect.x,
      y: rect.y,
      width: rect.width,
      height: rect.height,
    };

    const documentRect: BridgeInspectRect = {
      x: rect.x + window.scrollX,
      y: rect.y + window.scrollY,
      width: rect.width,
      height: rect.height,
    };

    const baseMeta = collectElementMeta(current);
    const meta: BridgeInspectAncestorMeta = {
      ...baseMeta,
      rect: viewportRect,
      documentRect,
      depth,
    };

    entries.push({
      element: current,
      rect: viewportRect,
      documentRect,
      meta,
      depth,
    });

    current = current.parentElement;
    depth += 1;
  }

  return entries;
};

const isTransparentColor = (value: string | null | undefined): boolean => {
  if (!value) {
    return true;
  }
  const normalized = value.trim().toLowerCase();
  if (!normalized) {
    return true;
  }
  if (normalized === 'transparent' || normalized === 'inherit') {
    return true;
  }
  if (normalized.startsWith('rgba(') || normalized.startsWith('hsla(') || normalized.startsWith('rgb(') || normalized.startsWith('hsl(')) {
    if (normalized.includes('/')) {
      return /\/\s*0(?:\)|$)/.test(normalized);
    }
    return normalized.endsWith(',0)') || normalized.endsWith(', 0)');
  }
  if (normalized.startsWith('#')) {
    if (normalized.length === 5) {
      return normalized.endsWith('0');
    }
    if (normalized.length === 9) {
      return normalized.endsWith('00');
    }
  }
  return false;
};

const resolveBackgroundColorForHierarchy = (element?: Element | null): string | null => {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return null;
  }
  const candidates: Element[] = [];
  const seen = new Set<Element>();

  if (element) {
    let current: Element | null = element;
    while (current) {
      candidates.push(current);
      current = current.parentElement;
    }
  }

  const docCandidates: Array<Element | null | undefined> = [document.body, document.documentElement];
  for (const docCandidate of docCandidates) {
    if (docCandidate) {
      candidates.push(docCandidate);
    }
  }

  for (const candidate of candidates) {
    if (seen.has(candidate)) {
      continue;
    }
    seen.add(candidate);
    try {
      const style = window.getComputedStyle(candidate);
      if (!style) {
        continue;
      }
      const color = style.backgroundColor;
      if (color && !isTransparentColor(color)) {
        return color;
      }
    } catch (error) {
      console.warn('[BridgeChild] Failed to resolve background color', error);
    }
  }
  return null;
};

const normalizeBackgroundColorOption = (value: string | null | undefined): string | null | undefined => {
  if (value === null) {
    return null;
  }
  if (typeof value !== 'string') {
    return undefined;
  }
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }
  if (isTransparentColor(trimmed)) {
    return null;
  }
  return trimmed;
};

type InspectStopReason = 'stop' | 'cancel' | 'complete';

interface InspectController {
  supported: boolean;
  start: () => boolean;
  stop: (reason?: InspectStopReason) => void;
  dispose: () => void;
  isActive: () => boolean;
  setTargetIndex: (index: number) => void;
  shiftTarget: (delta: number) => void;
}

const createInspectController = (post: PostFn): InspectController => {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return {
      supported: false,
      start: () => false,
      stop: () => undefined,
      dispose: () => undefined,
      isActive: () => false,
      setTargetIndex: () => undefined,
      shiftTarget: () => undefined,
    };
  }

  let active = false;
  let overlay: HTMLDivElement | null = null;
  let labelNode: HTMLDivElement | null = null;
  let lastHoverElement: Element | null = null;
  let lastRect: BridgeInspectRect | null = null;
  let lastPointerType: string | undefined;
  let lastHoverStack: InspectAncestryEntry[] = [];
  let selectedAncestorIndex = 0;
  let previousCursor: string | null = null;
  let previousTouchAction: string | null = null;
  let previousRootOverflow: string | null = null;
  let previousBodyOverflow: string | null = null;
  let activePointerId: number | null = null;
  let activePointerCaptureTarget: Element | null = null;
  let touchMoveGuardRef: ((event: TouchEvent) => void) | null = null;

  const ensureOverlayNodes = () => {
    if (!overlay) {
      overlay = document.createElement('div');
      overlay.setAttribute('data-bridge-inspector-overlay', 'true');
      overlay.style.position = 'fixed';
      overlay.style.zIndex = '2147483647';
      overlay.style.pointerEvents = 'none';
      overlay.style.border = '2px solid rgba(59, 130, 246, 0.85)';
      overlay.style.background = 'rgba(59, 130, 246, 0.2)';
      overlay.style.borderRadius = '4px';
      overlay.style.boxShadow = '0 0 0 1px rgba(59, 130, 246, 0.6)';
      overlay.style.transition = 'transform 0.08s ease, width 0.08s ease, height 0.08s ease, left 0.08s ease, top 0.08s ease';
      overlay.style.display = 'none';
    }

    if (!labelNode) {
      labelNode = document.createElement('div');
      labelNode.setAttribute('data-bridge-inspector-overlay', 'true');
      labelNode.style.position = 'fixed';
      labelNode.style.zIndex = '2147483647';
      labelNode.style.pointerEvents = 'none';
      labelNode.style.background = 'rgba(17, 24, 39, 0.92)';
      labelNode.style.color = '#f9fafb';
      labelNode.style.fontFamily = 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif';
      labelNode.style.fontSize = '12px';
      labelNode.style.lineHeight = '1.35';
      labelNode.style.padding = '4px 6px';
      labelNode.style.borderRadius = '4px';
      labelNode.style.boxShadow = '0 2px 4px rgba(15, 23, 42, 0.35)';
      labelNode.style.maxWidth = '280px';
      labelNode.style.whiteSpace = 'nowrap';
      labelNode.style.textOverflow = 'ellipsis';
      labelNode.style.overflow = 'hidden';
      labelNode.style.display = 'none';
    }
  };

  const removeOverlayNodes = () => {
    if (overlay?.parentElement) {
      overlay.parentElement.removeChild(overlay);
    }
    if (labelNode?.parentElement) {
      labelNode.parentElement.removeChild(labelNode);
    }
    overlay = null;
    labelNode = null;
  };

  const hideOverlay = () => {
    if (overlay) {
      overlay.style.display = 'none';
    }
    if (labelNode) {
      labelNode.style.display = 'none';
    }
  };

  const restoreCursor = () => {
    const body = document.body;
    if (!body) {
      return;
    }
    if (previousCursor === null) {
      body.style.removeProperty('cursor');
    } else {
      body.style.cursor = previousCursor;
    }
    previousCursor = null;
  };

  const applyCursor = () => {
    const body = document.body;
    if (!body) {
      return;
    }
    if (previousCursor === null) {
      previousCursor = body.style.cursor || '';
    }
    body.style.cursor = 'crosshair';
  };

  const restoreTouchAction = () => {
    const root = document.documentElement;
    if (!root) {
      return;
    }
    if (previousTouchAction === null) {
      return;
    }
    if (previousTouchAction === '') {
      root.style.removeProperty('touch-action');
    } else {
      root.style.touchAction = previousTouchAction;
    }
    previousTouchAction = null;
  };

  const applyTouchAction = () => {
    const root = document.documentElement;
    if (!root) {
      return;
    }
    if (previousTouchAction === null) {
      previousTouchAction = root.style.touchAction || '';
    }
    root.style.touchAction = 'none';
  };

  const applyScrollLock = () => {
    const root = document.documentElement;
    const body = document.body;
    if (root && previousRootOverflow === null) {
      previousRootOverflow = root.style.overflow || '';
      root.style.overflow = 'hidden';
    }
    if (body && previousBodyOverflow === null) {
      previousBodyOverflow = body.style.overflow || '';
      body.style.overflow = 'hidden';
    }
  };

  const restoreScrollLock = () => {
    const root = document.documentElement;
    const body = document.body;
    if (root && previousRootOverflow !== null) {
      if (previousRootOverflow === '') {
        root.style.removeProperty('overflow');
      } else {
        root.style.overflow = previousRootOverflow;
      }
      previousRootOverflow = null;
    }
    if (body && previousBodyOverflow !== null) {
      if (previousBodyOverflow === '') {
        body.style.removeProperty('overflow');
      } else {
        body.style.overflow = previousBodyOverflow;
      }
      previousBodyOverflow = null;
    }
  };

  const installTouchMoveGuard = () => {
    if (touchMoveGuardRef || typeof window === 'undefined') {
      return;
    }
    const handler = (event: TouchEvent) => {
      if (!active) {
        return;
      }
      if (event.touches.length === 0) {
        return;
      }
      try {
        event.preventDefault();
      } catch (error) {
        console.debug('[BridgeChild] touch gesture preventDefault failed', error);
      }
    };
    window.addEventListener('touchmove', handler, { passive: false, capture: true });
    window.addEventListener('touchstart', handler, { passive: false, capture: true });
    touchMoveGuardRef = handler;
  };

  const removeTouchMoveGuard = () => {
    if (!touchMoveGuardRef || typeof window === 'undefined') {
      return;
    }
    window.removeEventListener('touchmove', touchMoveGuardRef, true);
    window.removeEventListener('touchstart', touchMoveGuardRef, true);
    touchMoveGuardRef = null;
  };

  const resetPointerTracking = () => {
    if (activePointerCaptureTarget && activePointerId !== null && typeof (activePointerCaptureTarget as any).releasePointerCapture === 'function') {
      try {
        (activePointerCaptureTarget as any).releasePointerCapture(activePointerId);
      } catch (error) {
        console.warn('[BridgeChild] Failed to release pointer capture', error);
      }
    }
    activePointerCaptureTarget = null;
    activePointerId = null;
    restoreTouchAction();
    restoreScrollLock();
  };

  const isOverlayElement = (node: EventTarget | null | undefined): boolean => {
    if (!node || !(node instanceof Element)) {
      return false;
    }
    return node.hasAttribute('data-bridge-inspector-overlay');
  };

  const findInspectableElement = (target: EventTarget | null | undefined): Element | null => {
    if (!target) {
      return null;
    }
    if (target instanceof Element && !isOverlayElement(target)) {
      return target;
    }
    if (typeof (target as any)?.composedPath === 'function') {
      const path = (target as any).composedPath() as EventTarget[];
      for (const entry of path) {
        if (entry instanceof Element && !isOverlayElement(entry)) {
          return entry;
        }
      }
    }
    return null;
  };

  const resolveElementFromPoint = (clientX: number, clientY: number): Element | null => {
    if (typeof document.elementFromPoint !== 'function') {
      return null;
    }
    const candidate = document.elementFromPoint(clientX, clientY);
    if (!candidate || !(candidate instanceof Element)) {
      return null;
    }
    if (!isOverlayElement(candidate)) {
      return candidate;
    }
    let parent: Element | null = candidate.parentElement;
    while (parent) {
      if (!isOverlayElement(parent)) {
        return parent;
      }
      parent = parent.parentElement;
    }
    return null;
  };

  const shouldDisableScrollForInspect = () => {
    if (typeof window === 'undefined') {
      return false;
    }
    try {
      if (typeof navigator !== 'undefined' && typeof navigator.maxTouchPoints === 'number' && navigator.maxTouchPoints > 0) {
        return true;
      }
      if (typeof window.matchMedia === 'function') {
        const mediaQuery = window.matchMedia('(pointer: coarse)');
        if (mediaQuery?.matches) {
          return true;
        }
      }
    } catch (error) {
      console.debug('[BridgeChild] Unable to evaluate pointer characteristics', error);
    }
    return false;
  };

  const ensureOverlayAttachment = () => {
    ensureOverlayNodes();
    if (overlay && !overlay.parentElement) {
      document.body.appendChild(overlay);
    }
    if (labelNode && !labelNode.parentElement) {
      document.body.appendChild(labelNode);
    }
  };

  const clampSelectedAncestorIndex = (index: number): number => {
    if (lastHoverStack.length === 0) {
      return 0;
    }
    return Math.min(Math.max(index, 0), lastHoverStack.length - 1);
  };

  const buildHoverPayloadFromSelection = (pointerType?: string): BridgeInspectHoverPayload | null => {
    if (lastHoverStack.length === 0) {
      return null;
    }
    selectedAncestorIndex = clampSelectedAncestorIndex(selectedAncestorIndex);
    const target = lastHoverStack[selectedAncestorIndex];
    const ancestors = lastHoverStack.map(entry => entry.meta);
    return {
      rect: target.rect,
      documentRect: target.documentRect,
      meta: target.meta,
      pointerType,
      ancestors,
      selectedAncestorIndex,
    };
  };

  const positionOverlay = (payload: BridgeInspectHoverPayload) => {
    if (!overlay || !labelNode) {
      return;
    }

    overlay.style.display = 'block';
    overlay.style.left = `${Math.round(payload.rect.x)}px`;
    overlay.style.top = `${Math.round(payload.rect.y)}px`;
    overlay.style.width = `${Math.max(1, Math.round(payload.rect.width))}px`;
    overlay.style.height = `${Math.max(1, Math.round(payload.rect.height))}px`;

    labelNode.textContent = payload.meta.label ?? payload.meta.selector ?? payload.meta.tag;
    labelNode.style.display = 'block';
    const margin = 8;
    let labelLeft = payload.rect.x;
    let labelTop = payload.rect.y - (labelNode.offsetHeight || 0) - margin;

    if (labelTop < margin) {
      labelTop = payload.rect.y + payload.rect.height + margin;
    }
    if (labelTop + (labelNode.offsetHeight || 0) > window.innerHeight - margin) {
      labelTop = Math.max(margin, window.innerHeight - (labelNode.offsetHeight || 0) - margin);
    }

    if (labelNode.offsetWidth > window.innerWidth) {
      labelNode.style.maxWidth = `${window.innerWidth - margin * 2}px`;
    }

    const overflowRight = labelLeft + (labelNode.offsetWidth || 0) - (window.innerWidth - margin);
    if (overflowRight > 0) {
      labelLeft = Math.max(margin, labelLeft - overflowRight);
    }

    if (labelLeft < margin) {
      labelLeft = margin;
    }

    labelNode.style.left = `${Math.round(labelLeft)}px`;
    labelNode.style.top = `${Math.round(labelTop)}px`;
  };

  const emitHover = (payload: BridgeInspectHoverPayload) => {
    post({ v: 1, t: 'INSPECT_HOVER', payload });
  };

  const emitCancel = () => {
    post({ v: 1, t: 'INSPECT_CANCEL' });
  };

  const emitState = (next: boolean, reason?: InspectStopReason | 'start') => {
    post({ v: 1, t: 'INSPECT_STATE', active: next, reason });
  };

  const emitResult = (payload: BridgeInspectResultPayload) => {
    post({ v: 1, t: 'INSPECT_RESULT', payload });
  };

  const emitHoverFromSelection = (pointerType?: string) => {
    const pointerValue = typeof pointerType !== 'undefined' ? pointerType : lastPointerType;
    const payload = buildHoverPayloadFromSelection(pointerValue);
    if (!payload) {
      hideOverlay();
      lastHoverElement = null;
      lastRect = null;
      return;
    }

    const rectChanged = !lastRect
      || Math.abs(payload.rect.x - (lastRect?.x ?? 0)) > 0.5
      || Math.abs(payload.rect.y - (lastRect?.y ?? 0)) > 0.5
      || Math.abs(payload.rect.width - (lastRect?.width ?? 0)) > 0.5
      || Math.abs(payload.rect.height - (lastRect?.height ?? 0)) > 0.5;
    const nextElement = lastHoverStack[selectedAncestorIndex]?.element ?? null;
    const elementChanged = nextElement !== lastHoverElement;
    const pointerChanged = typeof pointerType !== 'undefined' && pointerType !== lastPointerType;

    if (!rectChanged && !elementChanged && !pointerChanged) {
      return;
    }

    ensureOverlayAttachment();
    positionOverlay(payload);
    emitHover(payload);

    lastHoverElement = nextElement;
    lastRect = payload.rect;
    lastPointerType = pointerValue;
  };

  const setSelectedAncestor = (index: number, pointerType?: string) => {
    if (lastHoverStack.length === 0) {
      return;
    }
    const clamped = clampSelectedAncestorIndex(index);
    if (clamped === selectedAncestorIndex && lastHoverElement === lastHoverStack[clamped]?.element) {
      return;
    }
    selectedAncestorIndex = clamped;
    emitHoverFromSelection(pointerType);
  };

  const shiftSelectedAncestor = (delta: number, pointerType?: string) => {
    if (!Number.isFinite(delta) || delta === 0) {
      return;
    }
    setSelectedAncestor(selectedAncestorIndex + delta, pointerType);
  };

  const updateHover = (element: Element, pointerType?: string) => {
    const stack = collectElementAncestry(element);
    if (stack.length === 0) {
      hideOverlay();
      lastHoverStack = [];
      lastHoverElement = null;
      lastRect = null;
      lastPointerType = pointerType ?? lastPointerType;
      return;
    }

    const previousBase = lastHoverStack.length > 0 ? lastHoverStack[0]?.element ?? null : null;
    lastHoverStack = stack;
    if (!previousBase || previousBase !== stack[0]?.element) {
      selectedAncestorIndex = 0;
    } else {
      selectedAncestorIndex = clampSelectedAncestorIndex(selectedAncestorIndex);
    }

    emitHoverFromSelection(pointerType);
  };

  const resetHover = () => {
    hideOverlay();
    lastHoverElement = null;
    lastRect = null;
    lastPointerType = undefined;
    lastHoverStack = [];
    selectedAncestorIndex = 0;
  };

  const finalizeSelection = (method: 'pointer' | 'keyboard') => {
    const payload = buildHoverPayloadFromSelection(lastPointerType);
    if (!payload) {
      return;
    }
    emitResult({ ...payload, method });
    stop('complete');
  };

  const isCoarsePointer = (pointerType: string | undefined) => {
    return pointerType === 'touch' || pointerType === 'pen';
  };

  const handlePointerMove = (event: PointerEvent) => {
    if (!active) {
      return;
    }

    if (isCoarsePointer(event.pointerType)) {
      if (activePointerId !== null && event.pointerId !== activePointerId) {
        return;
      }
      if (activePointerId === null) {
        // Ignore passive touch moves until we have an active touch pointer.
        return;
      }
    }

    let element: Element | null = null;
    if (isCoarsePointer(event.pointerType)) {
      element = resolveElementFromPoint(event.clientX, event.clientY);
    }
    if (!element) {
      element = findInspectableElement(event.target ?? event.currentTarget ?? null);
    }
    if (!element) {
      resetHover();
      return;
    }
    updateHover(element, event.pointerType);
  };

  const handlePointerDown = (event: PointerEvent) => {
    if (!active) {
      return;
    }
    const coarse = isCoarsePointer(event.pointerType);
    let element: Element | null = null;
    if (coarse) {
      element = resolveElementFromPoint(event.clientX, event.clientY);
    }
    if (!element) {
      element = findInspectableElement(event.target ?? event.currentTarget ?? null);
    }
    if (!element) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    if (typeof (event as any).stopImmediatePropagation === 'function') {
      (event as any).stopImmediatePropagation();
    }
    if (coarse) {
      if (activePointerId !== null && event.pointerId !== activePointerId) {
        return;
      }
      updateHover(element, event.pointerType);
      activePointerId = event.pointerId;
      applyTouchAction();
      let captureTarget: Element | null = null;
      const candidateTargets: Array<Element | null> = [element, event.target instanceof Element ? event.target : null];
      for (const candidate of candidateTargets) {
        if (candidate && typeof (candidate as any).setPointerCapture === 'function') {
          captureTarget = candidate;
          break;
        }
      }
      if (captureTarget) {
        try {
          (captureTarget as any).setPointerCapture(event.pointerId);
          activePointerCaptureTarget = captureTarget;
        } catch (error) {
          console.warn('[BridgeChild] Failed to set pointer capture', error);
          activePointerCaptureTarget = null;
        }
      } else {
        activePointerCaptureTarget = null;
      }
      return;
    }

    updateHover(element, event.pointerType);
    finalizeSelection('pointer');
  };

  const handlePointerUp = (event: PointerEvent) => {
    if (!active || !isCoarsePointer(event.pointerType)) {
      return;
    }
    if (activePointerId === null || event.pointerId !== activePointerId) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    if (typeof (event as any).stopImmediatePropagation === 'function') {
      (event as any).stopImmediatePropagation();
    }

    let element = resolveElementFromPoint(event.clientX, event.clientY);
    if (!element) {
      element = findInspectableElement(event.target ?? event.currentTarget ?? lastHoverElement);
    }
    if (element) {
      updateHover(element, event.pointerType);
      finalizeSelection('pointer');
      return;
    }

    emitCancel();
    stop('cancel');
  };

  const handlePointerCancel = (event: PointerEvent) => {
    if (!active) {
      return;
    }
    if (activePointerId === null || event.pointerId !== activePointerId) {
      return;
    }
    emitCancel();
    stop('cancel');
  };

  const handleClick = (event: MouseEvent) => {
    if (!active) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    if (typeof (event as any).stopImmediatePropagation === 'function') {
      (event as any).stopImmediatePropagation();
    }
  };

  const handleKeyDown = (event: KeyboardEvent) => {
    if (!active) {
      return;
    }
    if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      emitCancel();
      stop('cancel');
      return;
    }

    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      event.stopPropagation();
      finalizeSelection('keyboard');
    }
  };

  const handleScrollOrResize = () => {
    if (!active || !lastHoverElement) {
      return;
    }
    updateHover(lastHoverElement, lastPointerType);
  };

  const start = () => {
    if (active) {
      return true;
    }
    if (!document.body) {
      return false;
    }
    active = true;
    activePointerId = null;
    activePointerCaptureTarget = null;
    if (shouldDisableScrollForInspect()) {
      applyTouchAction();
      applyScrollLock();
      installTouchMoveGuard();
    }
    resetHover();
    ensureOverlayNodes();
    if (overlay && !overlay.parentElement) {
      document.body.appendChild(overlay);
    }
    if (labelNode && !labelNode.parentElement) {
      document.body.appendChild(labelNode);
    }
    applyCursor();
    window.addEventListener('pointermove', handlePointerMove, true);
    window.addEventListener('pointerdown', handlePointerDown, true);
    window.addEventListener('pointerup', handlePointerUp, true);
    window.addEventListener('pointercancel', handlePointerCancel, true);
    window.addEventListener('click', handleClick, true);
    window.addEventListener('keydown', handleKeyDown, true);
    window.addEventListener('scroll', handleScrollOrResize, true);
    window.addEventListener('resize', handleScrollOrResize, true);
    emitState(true, 'start');
    return true;
  };

  const stop = (reason: InspectStopReason = 'stop') => {
    if (!active) {
      return;
    }
    active = false;
    window.removeEventListener('pointermove', handlePointerMove, true);
    window.removeEventListener('pointerdown', handlePointerDown, true);
    window.removeEventListener('pointerup', handlePointerUp, true);
    window.removeEventListener('pointercancel', handlePointerCancel, true);
    window.removeEventListener('click', handleClick, true);
    window.removeEventListener('keydown', handleKeyDown, true);
    window.removeEventListener('scroll', handleScrollOrResize, true);
    window.removeEventListener('resize', handleScrollOrResize, true);
    restoreCursor();
    resetPointerTracking();
    resetHover();
    removeOverlayNodes();
    removeTouchMoveGuard();
    emitState(false, reason);
  };

  const dispose = () => {
    stop('stop');
    removeOverlayNodes();
  };

  return {
    supported: true,
    start,
    stop,
    dispose,
    isActive: () => active,
    setTargetIndex: (index: number) => {
      if (!active) {
        return;
      }
      setSelectedAncestor(index);
    },
    shiftTarget: (delta: number) => {
      if (!active) {
        return;
      }
      shiftSelectedAncestor(delta);
    },
  };
};

const getViewportRect = (): Rect => {
  const viewport = window.visualViewport;
  if (viewport) {
    const width = viewport.width > 0 ? viewport.width : window.innerWidth;
    const height = viewport.height > 0 ? viewport.height : window.innerHeight;
    return {
      x: Math.max(0, viewport.pageLeft),
      y: Math.max(0, viewport.pageTop),
      width: Math.max(1, width || document.documentElement.clientWidth || document.body.clientWidth || 0),
      height: Math.max(1, height || document.documentElement.clientHeight || document.body.clientHeight || 0),
    };
  }
  const width = Math.max(1, window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth || 0);
  const height = Math.max(1, window.innerHeight || document.documentElement.clientHeight || document.body.clientHeight || 0);
  return {
    x: Math.max(0, window.scrollX),
    y: Math.max(0, window.scrollY),
    width,
    height,
  };
};

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

  const inspectController = createInspectController(post);
  if (inspectController.supported) {
    caps.push('inspect');
  }

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

  const handleSpaHooksDiagnostic = (payload: { id: string; token?: string }) => {
    const originalHref = window.location.href;
    const originalState = history.state;
    const hashSuffix = typeof payload.token === 'string' && payload.token.length > 0
      ? payload.token
      : `bridge-diag-${Date.now().toString(16)}`;
    const diagHash = hashSuffix.startsWith('#') ? hashSuffix : `#${hashSuffix}`;

    try {
      const diagUrl = new URL(originalHref);
      diagUrl.hash = diagHash;
      history.replaceState(originalState ?? {}, '', diagUrl.href);
      history.replaceState(originalState ?? {}, '', originalHref);
      post({ v: 1, t: 'DIAG_RESULT', kind: 'SPA_HOOKS', id: payload.id, ok: true });
    } catch (error) {
      try {
        history.replaceState(originalState ?? {}, '', originalHref);
      } catch (restoreError) {
        console.warn('[BridgeChild] Failed to restore history during diagnostic', restoreError);
      }
      post({
        v: 1,
        t: 'DIAG_RESULT',
        kind: 'SPA_HOOKS',
        id: payload.id,
        ok: false,
        error: (error as Error)?.message ?? String(error),
      });
    }
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

    if (message.t === 'DIAG') {
      if (message.kind === 'SPA_HOOKS' && typeof message.id === 'string') {
        handleSpaHooksDiagnostic({ id: message.id, token: typeof message.token === 'string' ? message.token : undefined });
      }
      return;
    }

    if (message.t === 'PING' && typeof message.ts === 'number') {
      post({ v: 1, t: 'PONG', ts: message.ts });
      return;
    }

    if (message.t === 'INSPECT' && inspectController.supported) {
      if (message.cmd === 'START') {
        const started = inspectController.start();
        if (!started) {
          post({ v: 1, t: 'INSPECT_ERROR', error: 'UNAVAILABLE' });
        }
      } else if (message.cmd === 'STOP') {
        inspectController.stop('stop');
      } else if (message.cmd === 'SET_TARGET' && typeof message.index === 'number') {
        inspectController.setTargetIndex(message.index);
      } else if (message.cmd === 'SHIFT_TARGET' && typeof message.delta === 'number') {
        inspectController.shiftTarget(message.delta);
      }
      return;
    }

    if (message.t === 'CAPTURE' && message.cmd === 'SCREENSHOT' && typeof message.id === 'string') {
      const capture = async () => {
        try {
          const html2canvas = await loadHtml2Canvas();
          const requestOptions = (typeof message.options === 'object' && message.options)
            ? (message.options as BridgeScreenshotOptions)
            : {};
          const scale = typeof requestOptions.scale === 'number' && Number.isFinite(requestOptions.scale) && requestOptions.scale > 0
            ? requestOptions.scale
            : window.devicePixelRatio || 1;
          const selector = typeof requestOptions.selector === 'string' ? requestOptions.selector.trim() : '';
          const element = selector ? findElementForSelector(selector) : null;
          const backgroundColorOption = normalizeBackgroundColorOption(requestOptions.backgroundColor);
          const inferredBackground = backgroundColorOption !== undefined
            ? backgroundColorOption
            : resolveBackgroundColorForHierarchy(element);
          let requestedClip = sanitizeClipRect(requestOptions.clip ?? null);
          const canCaptureElement = Boolean(element) && requestOptions.mode !== 'full-page';
          if (canCaptureElement && element && typeof element.getBoundingClientRect === 'function') {
            const rect = element.getBoundingClientRect();
            if (rect.width > 0 && rect.height > 0) {
              try {
                const elementCanvas = await html2canvas(element, {
                  scale,
                  logging: false,
                  useCORS: true,
                  backgroundColor: inferredBackground ?? null,
                });
                const elementDataUrl = elementCanvas.toDataURL('image/png');
                const elementBase64 = elementDataUrl.replace(/^data:image\/png;base64,/, '');
                const clipRect: Rect = {
                  x: rect.x + window.scrollX,
                  y: rect.y + window.scrollY,
                  width: rect.width,
                  height: rect.height,
                };
                post({
                  v: 1,
                  t: 'SCREENSHOT_RESULT',
                  id: message.id,
                  ok: true,
                  data: elementBase64,
                  width: elementCanvas.width,
                  height: elementCanvas.height,
                  note: 'Captured selected element region.',
                  mode: 'clip',
                  clip: clipRect,
                });
                return;
              } catch (error) {
                console.warn('[BridgeChild] Element screenshot capture failed; falling back to clip capture', error);
              }
            }
          }
          if (!requestedClip && element && typeof element.getBoundingClientRect === 'function') {
            const rect = element.getBoundingClientRect();
            if (rect.width > 0 && rect.height > 0) {
              requestedClip = {
                x: rect.x + window.scrollX,
                y: rect.y + window.scrollY,
                width: rect.width,
                height: rect.height,
              };
            }
          }
          let captureMode: BridgeScreenshotMode = requestOptions.mode === 'full-page' ? 'full-page' : 'viewport';
          const target = document.documentElement as HTMLElement;
          const captureOptions: Record<string, unknown> = {
            scale,
            logging: false,
            useCORS: true,
            backgroundColor: inferredBackground ?? null,
          };

          const currentViewport = getViewportRect();
          const viewportWidth = Math.max(1, Math.round(currentViewport.width));
          const viewportHeight = Math.max(1, Math.round(currentViewport.height));
          const docWidth = Math.max(
            document.documentElement?.scrollWidth ?? 0,
            document.body?.scrollWidth ?? 0,
            viewportWidth,
          );
          const docHeight = Math.max(
            document.documentElement?.scrollHeight ?? 0,
            document.body?.scrollHeight ?? 0,
            viewportHeight,
          );

          let viewportRect: Rect | null = null;
          let cropContext: {
            offsetX: number;
            offsetY: number;
            width: number;
            height: number;
            scrollX: number;
            scrollY: number;
          } | null = null;

          if (requestedClip) {
            captureMode = 'clip';
            const maxScrollX = Math.max(0, docWidth - viewportWidth);
            const maxScrollY = Math.max(0, docHeight - viewportHeight);
            const desiredScrollX = requestedClip.x + requestedClip.width - viewportWidth;
            const desiredScrollY = requestedClip.y + requestedClip.height - viewportHeight;
            const scrollX = clampNumber(desiredScrollX, 0, maxScrollX);
            const scrollY = clampNumber(desiredScrollY, 0, maxScrollY);
            const offsetX = Math.max(0, requestedClip.x - scrollX);
            const offsetY = Math.max(0, requestedClip.y - scrollY);
            const visibleWidth = Math.min(
              Math.max(1, Math.round(requestedClip.width)),
              Math.max(1, viewportWidth - offsetX),
            );
            const visibleHeight = Math.min(
              Math.max(1, Math.round(requestedClip.height)),
              Math.max(1, viewportHeight - offsetY),
            );

            cropContext = {
              offsetX,
              offsetY,
              width: visibleWidth,
              height: visibleHeight,
              scrollX,
              scrollY,
            };

            captureOptions.scrollX = scrollX;
            captureOptions.scrollY = scrollY;
            captureOptions.windowWidth = viewportWidth;
            captureOptions.windowHeight = viewportHeight;
            captureOptions.x = 0;
            captureOptions.y = 0;
            captureOptions.width = viewportWidth;
            captureOptions.height = viewportHeight;
          } else if (captureMode === 'viewport') {
            viewportRect = currentViewport;
            captureOptions.scrollX = currentViewport.x;
            captureOptions.scrollY = currentViewport.y;
            captureOptions.windowWidth = viewportWidth;
            captureOptions.windowHeight = viewportHeight;
            captureOptions.x = 0;
            captureOptions.y = 0;
            captureOptions.width = viewportWidth;
            captureOptions.height = viewportHeight;
          }

          const canvas = await html2canvas(target, captureOptions);

          let resultCanvas = canvas;
          let note: string | undefined;
          let clipMetadata: Rect | undefined;

          if (captureMode === 'clip' && requestedClip) {
            const fallbackScrollX = typeof captureOptions.scrollX === 'number' ? captureOptions.scrollX : 0;
            const fallbackScrollY = typeof captureOptions.scrollY === 'number' ? captureOptions.scrollY : 0;
            const context = cropContext ?? {
              offsetX: Math.max(0, requestedClip.x - fallbackScrollX),
              offsetY: Math.max(0, requestedClip.y - fallbackScrollY),
              width: Math.max(1, Math.round(requestedClip.width)),
              height: Math.max(1, Math.round(requestedClip.height)),
              scrollX: fallbackScrollX,
              scrollY: fallbackScrollY,
            };

            const relativeClip: Rect = {
              x: context.offsetX,
              y: context.offsetY,
              width: Math.max(1, context.width),
              height: Math.max(1, context.height),
            };
            const scaledClip = scaleRect(relativeClip, scale);
            const clampedClip = clampClipToCanvas(scaledClip, canvas.width, canvas.height);

            clipMetadata = {
              x: requestedClip.x,
              y: requestedClip.y,
              width: relativeClip.width,
              height: relativeClip.height,
            };

            const clippedPartially =
              Math.round(relativeClip.width) < Math.round(requestedClip.width)
              || Math.round(relativeClip.height) < Math.round(requestedClip.height);
            note = clippedPartially
              ? 'Captured the visible portion of the selected region.'
              : 'Captured selected element region.';

            if (clampedClip) {
              const requiresCrop =
                clampedClip.x !== 0
                || clampedClip.y !== 0
                || clampedClip.width !== canvas.width
                || clampedClip.height !== canvas.height;

              if (requiresCrop) {
                const clipCanvas = document.createElement('canvas');
                clipCanvas.width = clampedClip.width;
                clipCanvas.height = clampedClip.height;
                const context2d = clipCanvas.getContext('2d');
                if (context2d) {
                  context2d.drawImage(
                    canvas,
                    clampedClip.x,
                    clampedClip.y,
                    clampedClip.width,
                    clampedClip.height,
                    0,
                    0,
                    clampedClip.width,
                    clampedClip.height,
                  );
                  resultCanvas = clipCanvas;
                }
              }
            } else {
              clipMetadata = undefined;
              captureMode = 'full-page';
              note = 'Unable to capture requested clip; captured the full page instead.';
            }
          } else if (captureMode === 'viewport') {
            const effectiveViewport = viewportRect ?? currentViewport;
            clipMetadata = {
              x: effectiveViewport.x,
              y: effectiveViewport.y,
              width: effectiveViewport.width,
              height: effectiveViewport.height,
            };
            note = 'Captured the currently visible viewport.';
          }

          const dataUrl = resultCanvas.toDataURL('image/png');
          const base64 = dataUrl.replace(/^data:image\/png;base64,/, '');
          post({
            v: 1,
            t: 'SCREENSHOT_RESULT',
            id: message.id,
            ok: true,
            data: base64,
            width: resultCanvas.width,
            height: resultCanvas.height,
            note,
            mode: captureMode,
            clip: clipMetadata,
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
      if (inspectController.supported) {
        inspectController.dispose();
      }
    },
  };
}
