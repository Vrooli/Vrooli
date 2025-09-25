export type BridgeCapability = 'history' | 'hash' | 'title' | 'deeplink' | 'resize' | 'screenshot';

type Html2CanvasFn = (element: HTMLElement, options?: Record<string, unknown>) => Promise<HTMLCanvasElement>;

export interface BridgeChildOptions {
  parentOrigin?: string;
  appId?: string;
  onNav?: (href: string) => void;
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
}

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

  const post = (payload: Record<string, unknown>) => {
    try {
      window.parent.postMessage(payload, resolvedOrigin);
    } catch (error) {
      console.warn('[BridgeChild] postMessage failed', error);
    }
  };

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
    } else if (message.t === 'PING' && typeof message.ts === 'number') {
      post({ v: 1, t: 'PONG', ts: message.ts });
    } else if (message.t === 'CAPTURE' && message.cmd === 'SCREENSHOT' && typeof message.id === 'string') {
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

  post({ v: 1, t: 'HELLO', appId: options.appId, title: document.title, caps });

  interceptHistory();
  const observer = setupObservers();

  queueMicrotask(() => post({ v: 1, t: 'READY' }));

  return {
    notify,
    dispose: () => {
      observer.disconnect();
      window.removeEventListener('message', handleMessage);
      window.removeEventListener('popstate', notify);
      window.removeEventListener('hashchange', notify);
      window.__vrooliBridgeChildInstalled = false;
    },
  };
}
