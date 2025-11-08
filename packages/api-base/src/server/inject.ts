/**
 * HTML injection utilities
 *
 * Functions for injecting proxy metadata and configuration into HTML documents.
 * Used by server middleware to provide runtime information to client code.
 */

import {
  PROXY_METADATA_SCRIPT_ID,
  SCENARIO_CONFIG_SCRIPT_ID,
  DEFAULT_PROXY_GLOBALS,
  DEFAULT_CONFIG_GLOBAL,
  LOOPBACK_HOSTS,
} from '../shared/constants.js'
import type { ProxyInfo, ProxyMetadataOptions, ScenarioConfig, PortEntry } from '../shared/types.js'
import { escapeForInlineScript } from '../shared/utils.js'

/**
 * Build proxy metadata payload
 *
 * Constructs the ProxyInfo object that will be injected into the client.
 *
 * @param options - Metadata options
 * @returns Proxy metadata object
 *
 * @example
 * ```typescript
 * const metadata = buildProxyMetadata({
 *   appId: 'scenario-auditor',
 *   hostScenario: 'app-monitor',
 *   targetScenario: 'scenario-auditor',
 *   ports: [uiPortEntry, apiPortEntry],
 *   primaryPort: uiPortEntry
 * })
 * ```
 */
export function buildProxyMetadata(options: ProxyMetadataOptions): ProxyInfo {
  const {
    appId,
    hostScenario,
    targetScenario,
    ports,
    primaryPort,
    loopbackHosts = Array.from(LOOPBACK_HOSTS),
  } = options

  return {
    appId,
    hostScenario,
    targetScenario,
    generatedAt: Date.now(),
    hosts: loopbackHosts,
    primary: primaryPort,
    ports: ports,
  }
}

/**
 * Build proxy index for fast lookups
 *
 * Creates an optimized index structure from proxy metadata for client-side lookups.
 * This is injected separately to avoid redundant data in the info payload.
 *
 * @internal
 */
function buildProxyIndex(info: ProxyInfo) {
  const aliasMap = new Map<string, PortEntry>()

  for (const entry of info.ports) {
    const aliases = entry.aliases || []
    // Add port number as alias
    aliases.push(String(entry.port))

    for (const alias of aliases) {
      const normalized = alias.trim().toLowerCase()
      if (normalized && !aliasMap.has(normalized)) {
        aliasMap.set(normalized, entry)
      }
    }
  }

  return {
    appId: info.appId,
    generatedAt: info.generatedAt,
    aliasMap,
    primary: info.primary,
    hosts: new Set(info.hosts.map(h => h.toLowerCase())),
  }
}

/**
 * Build proxy bootstrap script
 *
 * Creates an inline script that:
 * 1. Injects proxy metadata into window globals
 * 2. Builds an index for fast lookups
 * 3. Optionally patches fetch/XMLHttpRequest (if patchFetch=true)
 *
 * @internal
 */
function buildProxyBootstrapScript(info: ProxyInfo, options: {
  infoGlobalName?: string
  indexGlobalName?: string
  patchFetch?: boolean
} = {}): string {
  const {
    infoGlobalName = DEFAULT_PROXY_GLOBALS[0], // '__VROOLI_PROXY_INFO__'
    indexGlobalName = DEFAULT_PROXY_GLOBALS[1], // '__VROOLI_PROXY_INDEX__'
    patchFetch = false,
  } = options

  const serialized = escapeForInlineScript(JSON.stringify(info))

  let script = `;(() => {
  const INFO_KEY = ${JSON.stringify(infoGlobalName)};
  const INDEX_KEY = ${JSON.stringify(indexGlobalName)};
  const payload = ${serialized};

  if (typeof window === 'undefined') {
    return;
  }

  window[INFO_KEY] = payload;

  // Build index for fast lookups
  const buildIndex = (data) => {
    if (!data || !Array.isArray(data.ports)) {
      return null;
    }

    const aliasMap = new Map();
    for (const entry of data.ports) {
      if (!entry || typeof entry.port !== 'number') {
        continue;
      }

      const aliases = Array.isArray(entry.aliases) ? entry.aliases : [];
      aliases.push(String(entry.port));

      for (const alias of aliases) {
        if (typeof alias !== 'string') {
          continue;
        }
        const key = alias.trim().toLowerCase();
        if (key && !aliasMap.has(key)) {
          aliasMap.set(key, entry);
        }
      }
    }

    return {
      appId: data.appId,
      generatedAt: data.generatedAt,
      aliasMap,
      primary: data.primary,
      hosts: new Set(Array.isArray(data.hosts) ? data.hosts.map(h => h.toLowerCase()) : []),
    };
  };

  const getIndex = () => {
    const latest = window[INFO_KEY];
    if (!latest) {
      return null;
    }
    const cached = window[INDEX_KEY];
    if (cached && cached.generatedAt === latest.generatedAt) {
      return cached;
    }
    const index = buildIndex(latest);
    if (index) {
      window[INDEX_KEY] = index;
    }
    return index;
  };

  // Pre-build index
  getIndex();`

  if (patchFetch) {
    script += `

  // Patch fetch and XMLHttpRequest to rewrite localhost URLs and relative API requests
  const joinPath = (base, path) => {
    const trimmedBase = base.endsWith('/') ? base.slice(0, -1) : base;
    const trimmedPath = path.startsWith('/') ? path : '/' + path;
    return trimmedBase + trimmedPath;
  };

  const resolveTarget = (href, scheme) => {
    let url;
    try {
      url = new URL(href, window.location.href);
    } catch (error) {
      return null;
    }

    const index = getIndex();
    if (!index) {
      return null;
    }

    // Check if this is a localhost URL (absolute localhost request to a DIFFERENT port)
    const isLocalhost = index.hosts.has(url.hostname.toLowerCase());
    const isSamePort = url.port === window.location.port || (!url.port && !window.location.port);

    if (isLocalhost && !isSamePort) {
      // For localhost URLs to different ports, use port-based routing
      const portKey = url.port ? url.port : '';
      let entry = portKey ? index.aliasMap.get(portKey.toLowerCase()) : null;
      if (!entry && !portKey) {
        entry = index.primary;
      }
      if (!entry) {
        return null;
      }

      const basePath = entry.path || index.primary?.path;
      if (!basePath) {
        return null;
      }

      const protocol = scheme === 'ws'
        ? (window.location.protocol === 'https:' ? 'wss:' : 'ws:')
        : window.location.protocol;
      const host = window.location.host;
      const path = joinPath(basePath, url.pathname || '/');
      const search = url.search || '';
      const hash = scheme === 'ws' ? '' : (url.hash || '');
      return protocol + '//' + host + path + search + hash;
    }

    // Check if this is a same-origin request from proxied iframe
    // When child iframe at /apps/child/proxy/index.html makes fetch('/api/v1/data'),
    // it resolves to http://host/api/v1/data (NOT /apps/child/proxy/api/v1/data)
    // because fetch uses window.location, not <base> tag
    const isSameOrigin = url.hostname === window.location.hostname &&
                         url.port === window.location.port;

    if (isSameOrigin && index.primary?.path) {
      const proxyPath = index.primary.path;
      const currentPath = window.location.pathname;

      // Only rewrite if we're in a proxied context and request path doesn't already include proxy path
      if (currentPath.startsWith(proxyPath) && !url.pathname.startsWith(proxyPath)) {
        // Rewrite to go through proxy path
        const protocol = scheme === 'ws'
          ? (window.location.protocol === 'https:' ? 'wss:' : 'ws:')
          : window.location.protocol;
        const host = window.location.host;
        const path = joinPath(proxyPath, url.pathname || '/');
        const search = url.search || '';
        const hash = scheme === 'ws' ? '' : (url.hash || '');
        return protocol + '//' + host + path + search + hash;
      }
    }

    return null;
  };

  const ensurePatched = () => {
    if (window.__VROOLI_PROXY_SHIMMED__) {
      return;
    }
    window.__VROOLI_PROXY_SHIMMED__ = true;

    // Patch fetch
    const originalFetch = typeof window.fetch === 'function' ? window.fetch.bind(window) : null;
    if (originalFetch) {
      window.fetch = function(resource, init) {
        let rewritten = null;
        if (typeof resource === 'string' || resource instanceof URL) {
          rewritten = resolveTarget(resource.toString(), 'http');
          if (rewritten) {
            return originalFetch.call(this, rewritten, init);
          }
        } else if (resource instanceof Request) {
          rewritten = resolveTarget(resource.url, 'http');
          if (rewritten) {
            const cloned = resource.clone();
            const { method } = cloned;
            const headers = new Headers(cloned.headers);
            const bodyUsed = method && (method.toUpperCase() === 'GET' || method.toUpperCase() === 'HEAD');
            const initOptions = {
              method,
              headers,
              cache: cloned.cache,
              credentials: cloned.credentials,
              integrity: cloned.integrity,
              keepalive: cloned.keepalive,
              mode: cloned.mode,
              redirect: cloned.redirect,
              referrer: cloned.referrer && cloned.referrer !== 'about:client' ? cloned.referrer : undefined,
              referrerPolicy: cloned.referrerPolicy,
              signal: cloned.signal,
            };
            if (!bodyUsed) {
              initOptions.body = cloned.body;
            }
            if ('duplex' in cloned) {
              initOptions.duplex = cloned.duplex;
            }
            const rewrittenRequest = new Request(rewritten, initOptions);
            return originalFetch.call(this, rewrittenRequest, init);
          }
        }
        return originalFetch.call(this, resource, init);
      };
    }

    // Patch XMLHttpRequest
    if (typeof window.XMLHttpRequest === 'function') {
      const originalOpen = window.XMLHttpRequest.prototype.open;
      window.XMLHttpRequest.prototype.open = function(method, url, async, user, password) {
        let rewrittenUrl = url;
        if (typeof url === 'string') {
          const rewritten = resolveTarget(url, 'http');
          if (rewritten) {
            rewrittenUrl = rewritten;
          }
        }
        return originalOpen.call(this, method, rewrittenUrl, async, user, password);
      };
    }

    // Patch WebSocket
    if (typeof window.WebSocket === 'function') {
      const OriginalWebSocket = window.WebSocket;
      const PatchedWebSocket = function(url, protocols) {
        let rewrittenUrl = url;
        if (typeof url === 'string') {
          const rewritten = resolveTarget(url, 'ws');
          if (rewritten) {
            rewrittenUrl = rewritten;
          }
        }
        return new OriginalWebSocket(rewrittenUrl, protocols);
      };
      PatchedWebSocket.prototype = OriginalWebSocket.prototype;
      Object.setPrototypeOf(PatchedWebSocket, OriginalWebSocket);
      window.WebSocket = PatchedWebSocket;
    }
  };

  ensurePatched();`
  }

  script += `
})();`

  return script
}

/**
 * Inject proxy metadata into HTML
 *
 * Injects a script tag containing proxy metadata into the HTML <head> section.
 * The client can then read this metadata to determine correct API URLs.
 *
 * @param html - HTML document
 * @param metadata - Proxy metadata to inject
 * @param options - Injection options
 * @returns Modified HTML with injected script
 *
 * @example
 * ```typescript
 * const metadata = buildProxyMetadata({ ... })
 * const modifiedHtml = injectProxyMetadata(html, metadata, {
 *   patchFetch: true // Auto-rewrite fetch/XHR requests
 * })
 * ```
 */
export function injectProxyMetadata(
  html: string,
  metadata: ProxyInfo,
  options: {
    infoGlobalName?: string
    indexGlobalName?: string
    patchFetch?: boolean
  } = {}
): string {
  const script = buildProxyBootstrapScript(metadata, options)

  // Inject into <head> if present, otherwise at start of <body>
  if (html.includes('<head>')) {
    return html.replace(
      '<head>',
      `<head>\n<script id="${PROXY_METADATA_SCRIPT_ID}">${script}</script>`
    )
  } else if (html.includes('<body>')) {
    return html.replace(
      '<body>',
      `<body>\n<script id="${PROXY_METADATA_SCRIPT_ID}">${script}</script>`
    )
  }

  // Fallback: prepend to entire document
  return `<script id="${PROXY_METADATA_SCRIPT_ID}">${script}</script>\n${html}`
}

/**
 * Inject scenario configuration into HTML
 *
 * Injects a script tag containing runtime configuration into the HTML.
 * This allows the client to access configuration without fetching /config.
 *
 * @param html - HTML document
 * @param config - Scenario configuration to inject
 * @param options - Injection options
 * @returns Modified HTML with injected config
 *
 * @example
 * ```typescript
 * const config = {
 *   apiUrl: 'http://localhost:8080/api/v1',
 *   wsUrl: 'ws://localhost:8080/ws',
 *   apiPort: '8080',
 *   uiPort: '3000'
 * }
 * const modifiedHtml = injectScenarioConfig(html, config)
 * ```
 */
export function injectScenarioConfig(
  html: string,
  config: ScenarioConfig,
  options: {
    configGlobalName?: string
  } = {}
): string {
  const {
    configGlobalName = DEFAULT_CONFIG_GLOBAL,
  } = options

  const serialized = escapeForInlineScript(JSON.stringify(config))
  const script = `;(() => {
  if (typeof window !== 'undefined') {
    window[${JSON.stringify(configGlobalName)}] = ${serialized};
  }
})();`

  // Inject into <head> if present, otherwise at start of <body>
  if (html.includes('<head>')) {
    return html.replace(
      '<head>',
      `<head>\n<script id="${SCENARIO_CONFIG_SCRIPT_ID}">${script}</script>`
    )
  } else if (html.includes('<body>')) {
    return html.replace(
      '<body>',
      `<body>\n<script id="${SCENARIO_CONFIG_SCRIPT_ID}">${script}</script>`
    )
  }

  // Fallback: prepend to entire document
  return `<script id="${SCENARIO_CONFIG_SCRIPT_ID}">${script}</script>\n${html}`
}

/**
 * Inject base tag into HTML
 *
 * Injects a `<base href="...">` tag into HTML to control how relative URLs are resolved.
 * This is critical for scenarios that act as hosts (like app-monitor) to ensure their
 * own assets load correctly even when accessed via nested URLs.
 *
 * @param html - HTML content
 * @param basePath - Base path to inject (e.g., "/" for root, or "/apps/scenario/proxy/" for proxied)
 * @param options - Injection options
 * @returns Modified HTML with base tag
 *
 * @example
 * ```typescript
 * // For a host scenario accessed at any URL, inject "/" to ensure assets load from root
 * const html = '<html><head></head><body>...</body></html>'
 * const modified = injectBaseTag(html, '/')
 * // Result: <html><head><base href="/"></head><body>...</body></html>
 *
 * // For a proxied scenario, inject the proxy path
 * const modified = injectBaseTag(html, '/apps/scenario-auditor/proxy/')
 * ```
 */
export function injectBaseTag(
  html: string,
  basePath: string,
  options: {
    /** Skip injection if base tag already exists */
    skipIfExists?: boolean
    /** Data attribute to add to base tag for identification */
    dataAttribute?: string
  } = {}
): string {
  const {
    skipIfExists = true,
    dataAttribute = 'data-vrooli-base',
  } = options

  // Check if base tag already exists
  if (skipIfExists && /<base\s+[^>]*>/i.test(html)) {
    return html
  }

  // Ensure trailing slash for base href
  const normalizedBase = basePath.endsWith('/') ? basePath : `${basePath}/`
  const baseTag = `<base ${dataAttribute}="injected" href="${normalizedBase}">`

  // Try to inject into <head>
  if (html.includes('<head>')) {
    return html.replace('<head>', `<head>\n${baseTag}`)
  }

  // Try to inject at start of <html>
  if (html.includes('<html>')) {
    return html.replace('<html>', `<html>\n${baseTag}`)
  }

  // Fallback: prepend to entire document
  return `${baseTag}\n${html}`
}
