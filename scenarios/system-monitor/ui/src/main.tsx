import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App.tsx'

type ProxyEndpoint = {
  path?: string;
  basePath?: string;
};

type AppMonitorProxyInfo = {
  path?: string;
  basePath?: string;
  primary?: ProxyEndpoint;
  endpoints?: ProxyEndpoint[];
};

declare global {
  interface Window {
    __systemMonitorBridgeInitialized?: boolean;
    __SYSTEM_MONITOR_BASE_PATH__?: string;
    __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo;
    __APP_MONITOR_PROXY_INDEX__?: AppMonitorProxyInfo;
  }
}

const normalizeBasename = (value?: string | null) => {
  if (!value) {
    return '';
  }

  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }

  // Treat dot-based base paths ('.', './', '/.') as equivalent to the root.
  if (/^\/?\.\/?$/.test(trimmed)) {
    return '';
  }

  const withLeading = trimmed.startsWith('/') ? trimmed : `/${trimmed}`;
  if (withLeading === '/') {
    return '/';
  }

  const normalized = withLeading.replace(/\/+$/, '');
  return normalized === '/.' ? '/' : normalized;
};

const pickProxyCandidate = (info?: AppMonitorProxyInfo): string | undefined => {
  if (!info) {
    return undefined;
  }

  const candidates: (string | undefined)[] = [];
  if (info.primary) {
    candidates.push(info.primary.path, info.primary.basePath);
  }

  if (Array.isArray(info.endpoints)) {
    for (const endpoint of info.endpoints) {
      candidates.push(endpoint?.path, endpoint?.basePath);
    }
  }

  candidates.push(info.path, info.basePath);

  for (const candidate of candidates) {
    const normalized = normalizeBasename(candidate);
    if (normalized) {
      return normalized;
    }
  }

  return undefined;
};

const resolveRouterBase = () => {
  const fallback = normalizeBasename(import.meta.env.BASE_URL || '/') || '/';

  if (typeof window === 'undefined') {
    return fallback;
  }

  const proxyInfo = window.__APP_MONITOR_PROXY_INFO__ ?? window.__APP_MONITOR_PROXY_INDEX__;
  const proxyBase = pickProxyCandidate(proxyInfo);
  if (proxyBase) {
    return proxyBase;
  }

  const pathname = window.location?.pathname;
  if (pathname && pathname.includes('/proxy/')) {
    const index = pathname.indexOf('/proxy/');
    if (index >= 0) {
      const inferred = normalizeBasename(pathname.slice(0, index + '/proxy'.length));
      if (inferred) {
        return inferred;
      }
    }
  }

  return fallback;
};

const routerBase = resolveRouterBase();

if (typeof window !== 'undefined') {
  window.__SYSTEM_MONITOR_BASE_PATH__ = routerBase;
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__systemMonitorBridgeInitialized) {
  let parentOrigin: string | undefined
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[SystemMonitor] Unable to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'system-monitor' })
  window.__systemMonitorBridgeInitialized = true
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter basename={routerBase}>
      <App />
    </BrowserRouter>
  </StrictMode>,
)
