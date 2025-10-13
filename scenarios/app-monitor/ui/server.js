const express = require('express');
const path = require('path');
const fs = require('fs');
const WebSocket = require('ws');
const http = require('http');
const axios = require('axios');
const net = require('net');
require('dotenv').config();

const app = express();

// Ensure required environment variables are set
if (!process.env.UI_PORT) {
    console.error('Error: UI_PORT environment variable is required');
    process.exit(1);
}
if (!process.env.API_PORT) {
    console.error('Error: API_PORT environment variable is required');
    process.exit(1);
}

const PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;
const server = http.createServer(app);
const wss = new WebSocket.Server({ noServer: true });

const LOOPBACK_HOST = '127.0.0.1';
const API_BASE = `http://${LOOPBACK_HOST}:${API_PORT}`;
const APP_PROXY_PREFIX = '/apps';
const APP_PROXY_SEGMENT = 'proxy';
const APP_PORTS_SEGMENT = 'ports';
const LOOPBACK_HOSTS = new Set([LOOPBACK_HOST, 'localhost', '::1', '[::1]', '0.0.0.0']);
const LOOPBACK_HOST_ARRAY = Array.from(LOOPBACK_HOSTS);
const PORT_LABEL_SANITIZE_REGEX = /[^a-z0-9]+/g;
const DEFAULT_PRIMARY_LABEL = 'ui';
const HOP_BY_HOP_HEADERS = new Set([
    'connection',
    'keep-alive',
    'proxy-authenticate',
    'proxy-authorization',
    'te',
    'trailers',
    'transfer-encoding',
    'upgrade',
    'sec-websocket-key',
    'sec-websocket-accept',
    'sec-websocket-version',
    'sec-websocket-protocol',
    'sec-websocket-extensions'
]);

const APP_CACHE_TTL_MS = 15_000;
const APP_PROXY_CACHE = new Map();
const APP_PROXY_INFLIGHT = new Map();

const PROXY_AFFINITY_TTL_MS = 120_000;
const PROXY_AFFINITY_MAX_ENTRIES = 5_000;
const DEFAULT_REF_KEY = '__NO_REF__';
// Map of request key -> Map of referer key -> affinity metadata
const proxyAffinity = new Map();
let proxyAffinityEntryCount = 0;

// Force HTTPS for browsers accessing via Cloudflare tunnel to avoid mixed-protocol issues
app.use((req, res, next) => {
    const forwardedProtoHeader = req.headers['x-forwarded-proto'];
    if (forwardedProtoHeader) {
        const forwardedProto = Array.isArray(forwardedProtoHeader)
            ? forwardedProtoHeader[0]
            : String(forwardedProtoHeader).split(',')[0].trim().toLowerCase();

        if (forwardedProto && forwardedProto !== 'https') {
            const host = req.headers.host;
            if (host && (req.method === 'GET' || req.method === 'HEAD')) {
                const redirectUrl = `https://${host}${req.originalUrl || ''}`;
                res.redirect(301, redirectUrl);
                return;
            }
        }
    }

    next();
});

const candidatePortKeys = [
    'ui_port',
    'ui',
    'app_port',
    'web_port',
    'client_port',
    'frontend_port',
    'preview_port',
    'vite_port',
    'http_port',
    'http',
    'port'
];

const ROOT_ASSET_PREFIXES = [
    '/@vite',
    '/@react-refresh',
    '/@fs/',
    '/src/',
    '/node_modules/.vite/',
    '/assets/'
];

const ROOT_ASSET_EXTENSIONS = new Set([
    '.js',
    '.mjs',
    '.ts',
    '.tsx',
    '.jsx',
    '.css',
    '.map',
    '.json',
    '.svg',
    '.png',
    '.jpg',
    '.jpeg',
    '.gif',
    '.ico',
    '.webp',
    '.woff',
    '.woff2',
    '.ttf',
    '.eot',
    '.wasm'
]);

const hasAssetExtension = (path) => {
    if (typeof path !== 'string') {
        return false;
    }
    const questionIndex = path.indexOf('?');
    const hashIndex = path.indexOf('#');
    let clean = path;
    if (questionIndex !== -1) {
        clean = clean.slice(0, questionIndex);
    }
    if (hashIndex !== -1) {
        clean = clean.slice(0, hashIndex);
    }
    const lastSlash = clean.lastIndexOf('/');
    const fileName = lastSlash === -1 ? clean : clean.slice(lastSlash + 1);
    const dotIndex = fileName.lastIndexOf('.');
    if (dotIndex === -1) {
        return false;
    }
    const ext = fileName.slice(dotIndex).toLowerCase();
    return ROOT_ASSET_EXTENSIONS.has(ext);
};

const startsWithAssetPrefix = (path) => {
    if (typeof path !== 'string') {
        return false;
    }
    return ROOT_ASSET_PREFIXES.some((prefix) => path.startsWith(prefix));
};

const isProxiableRootAssetRequest = (path) => {
    if (typeof path !== 'string') {
        return false;
    }
    if (path.startsWith(`${APP_PROXY_PREFIX}/`)) {
        return false;
    }
    if (path.startsWith('/api/') || path.startsWith('/ws') || path === '/api' || path === '/ws') {
        return false;
    }
    return startsWithAssetPrefix(path) || hasAssetExtension(path);
};

const parseNumericPort = (value) => {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) {
        return null;
    }
    if (!Number.isInteger(parsed) || parsed <= 0 || parsed > 65535) {
        return null;
    }
    return parsed;
};

const unwrapValue = (value, depth = 0) => {
    if (depth > 4 || value == null) {
        return value;
    }
    if (Array.isArray(value) && value.length > 0) {
        return unwrapValue(value[0], depth + 1);
    }
    if (typeof value === 'object') {
        if (Object.prototype.hasOwnProperty.call(value, 'value')) {
            return unwrapValue(value.value, depth + 1);
        }
        if (Object.prototype.hasOwnProperty.call(value, 'port')) {
            return unwrapValue(value.port, depth + 1);
        }
    }
    return value;
};

const parsePortField = (value) => {
    const unwrapped = unwrapValue(value);
    if (typeof unwrapped === 'number') {
        return parseNumericPort(unwrapped);
    }
    if (typeof unwrapped === 'string') {
        const trimmed = unwrapped.trim();
        if (!trimmed) {
            return null;
        }
        if (/^\d+$/.test(trimmed)) {
            return parseNumericPort(Number(trimmed));
        }
        try {
            const url = new URL(trimmed, trimmed.startsWith('http') ? undefined : 'http://placeholder');
            if (url.port) {
                return parseNumericPort(Number(url.port));
            }
        } catch (error) {
            const match = trimmed.match(/:(\d+)(?!.*:\d+)/);
            if (match) {
                return parseNumericPort(Number(match[1]));
            }
        }
    }
    return null;
};

const getPortFromMappings = (portMappings = {}, key) => {
    const normalizedKey = key.toLowerCase();
    for (const [label, value] of Object.entries(portMappings)) {
        if (label.toLowerCase() === normalizedKey) {
            const parsed = parsePortField(value);
            if (parsed !== null) {
                return parsed;
            }
        }
    }
    return null;
};

const getPortFromEnvironment = (environment = {}, key) => {
    const normalizedKey = key.toLowerCase();
    for (const [label, value] of Object.entries(environment)) {
        const normalizedLabel = label.toLowerCase();
        if (
            normalizedLabel === normalizedKey ||
            normalizedLabel.endsWith(`_${normalizedKey}`) ||
            normalizedLabel.endsWith(`-${normalizedKey}`) ||
            normalizedLabel.includes(normalizedKey)
        ) {
            const parsed = parsePortField(value);
            if (parsed !== null) {
                return parsed;
            }
        }
    }
    return null;
};

const findFallbackEnvironmentPort = (environment = {}) => {
    let fallback = null;
    for (const [label, value] of Object.entries(environment)) {
        const parsed = parsePortField(value);
        if (parsed === null) {
            continue;
        }
        const normalizedLabel = label.toLowerCase();
        if (normalizedLabel.includes('ui') && normalizedLabel.includes('port')) {
            return parsed;
        }
        if (fallback === null) {
            fallback = parsed;
        }
    }
    return fallback;
};

const computeAppUIPort = (app) => {
    if (!app || typeof app !== 'object') {
        return null;
    }

    const portMappings = app.port_mappings || {};
    const environment = app.environment || {};
    const config = app.config || {};

    const primaryPort = parsePortField(config.primary_port);
    if (primaryPort !== null) {
        return primaryPort;
    }

    if (typeof config.primary_port_label === 'string') {
        const labeled = getPortFromMappings(portMappings, config.primary_port_label);
        if (labeled !== null) {
            return labeled;
        }
    }

    for (const key of candidatePortKeys) {
        const fromMappings = getPortFromMappings(portMappings, key);
        if (fromMappings !== null) {
            return fromMappings;
        }
        const fromEnv = getPortFromEnvironment(environment, key);
        if (fromEnv !== null) {
            return fromEnv;
        }
    }

    for (const value of Object.values(portMappings)) {
        const parsed = parsePortField(value);
        if (parsed !== null) {
            return parsed;
        }
    }

    const envFallback = findFallbackEnvironmentPort(environment);
    if (envFallback !== null) {
        return envFallback;
    }

    const generalPort = parsePortField(app.port);
    if (generalPort !== null) {
        return generalPort;
    }

    return null;
};

const normalizePortLabel = (label) => {
    if (typeof label !== 'string') {
        return null;
    }
    const trimmed = label.trim();
    if (!trimmed) {
        return null;
    }
    return trimmed.toLowerCase();
};

const slugifyPortLabel = (label, port) => {
    const normalized = normalizePortLabel(label);
    if (normalized && normalized !== String(port)) {
        const slug = normalized.replace(PORT_LABEL_SANITIZE_REGEX, '-').replace(/^-+|-+$/g, '');
        if (slug) {
            return slug;
        }
    }
    return `port-${port}`;
};

const createPortEntry = ({
    appId,
    port,
    label,
    source,
    priority = 0,
    kind = null,
}) => {
    if (typeof port !== 'number' || !Number.isInteger(port) || port <= 0 || port > 65535) {
        throw new Error(`Invalid port for ${appId}: ${port}`);
    }

    const normalizedLabel = normalizePortLabel(label);
    const entry = {
        appId,
        port,
        label: typeof label === 'string' && label.trim() ? label.trim() : null,
        normalizedLabel,
        slug: slugifyPortLabel(label, port),
        source: source || 'unknown',
        priority: Number.isFinite(priority) ? priority : 0,
        kind,
        isPrimary: false,
        path: null,
        aliases: [],
    };

    return entry;
};

const buildPortAliasIndex = (entries) => {
    const aliasIndex = new Map();
    for (const entry of entries) {
        const aliases = new Set();
        if (entry.label) {
            aliases.add(entry.label);
        }
        if (entry.normalizedLabel) {
            aliases.add(entry.normalizedLabel);
        }
        if (entry.slug) {
            aliases.add(entry.slug);
        }
        aliases.add(String(entry.port));
        if (entry.isPrimary) {
            aliases.add(DEFAULT_PRIMARY_LABEL);
            aliases.add('primary');
            aliases.add('default');
        }

        entry.aliases = Array.from(aliases);

        for (const alias of entry.aliases) {
            const normalized = normalizePortLabel(alias);
            if (!normalized) {
                continue;
            }
            if (!aliasIndex.has(normalized)) {
                aliasIndex.set(normalized, entry);
            }
        }
    }

    return aliasIndex;
};

const buildAppProxyState = (appId, metadata) => {
    const entries = [];
    const entryKeySet = new Set();
    const portIndex = new Map();

    const registerEntry = (draftEntry) => {
        const key = `${draftEntry.port}:${draftEntry.slug}`;
        if (entryKeySet.has(key)) {
            return entries.find((entry) => entry.port === draftEntry.port && entry.slug === draftEntry.slug);
        }

        entries.push(draftEntry);
        entryKeySet.add(key);

        if (!portIndex.has(draftEntry.port)) {
            portIndex.set(draftEntry.port, []);
        }
        portIndex.get(draftEntry.port).push(draftEntry);
        return draftEntry;
    };

    const portMappings = metadata?.port_mappings || {};
    const environment = metadata?.environment || {};
    const config = metadata?.config || {};

    for (const [label, value] of Object.entries(portMappings)) {
        const port = parsePortField(value);
        if (port === null) {
            continue;
        }
        registerEntry(
            createPortEntry({
                appId,
                port,
                label,
                source: 'port_mappings',
                priority: 30,
            })
        );
    }

    for (const [label, value] of Object.entries(environment)) {
        const normalizedLabel = normalizePortLabel(label);
        if (!normalizedLabel || (!normalizedLabel.includes('port') && !normalizedLabel.includes('url'))) {
            continue;
        }
        const port = parsePortField(value);
        if (port === null) {
            continue;
        }
        registerEntry(
            createPortEntry({
                appId,
                port,
                label,
                source: 'environment',
                priority: normalizedLabel.includes('ui') ? 15 : 10,
            })
        );
    }

    const configPrimaryPort = parsePortField(config.primary_port);
    if (configPrimaryPort !== null) {
        registerEntry(
            createPortEntry({
                appId,
                port: configPrimaryPort,
                label: normalizePortLabel(config.primary_port_label) || 'primary',
                source: 'config.primary_port',
                priority: 80,
            })
        );
    }

    if (typeof config.primary_port_label === 'string') {
        const labeledPort =
            getPortFromMappings(portMappings, config.primary_port_label) ??
            getPortFromEnvironment(environment, config.primary_port_label);
        if (labeledPort !== null) {
            registerEntry(
                createPortEntry({
                    appId,
                    port: labeledPort,
                    label: config.primary_port_label,
                    source: 'config.primary_port_label',
                    priority: 70,
                })
            );
        }
    }

    const fallbackPort = parsePortField(metadata?.port);
    if (fallbackPort !== null) {
        registerEntry(
            createPortEntry({
                appId,
                port: fallbackPort,
                label: 'port',
                source: 'metadata.port',
                priority: 5,
            })
        );
    }

    const primaryPort = computeAppUIPort(metadata);
    if (primaryPort === null) {
        throw new Error(`App ${appId} does not expose a preview port`);
    }

    const existingPrimaryCandidates = portIndex.get(primaryPort) || [];
    let primaryEntry = existingPrimaryCandidates.sort((a, b) => (b.priority ?? 0) - (a.priority ?? 0))[0] ?? null;

    if (!primaryEntry) {
        primaryEntry = registerEntry(
            createPortEntry({
                appId,
                port: primaryPort,
                label: DEFAULT_PRIMARY_LABEL,
                source: 'heuristic',
                priority: 90,
            })
        );
    }

    primaryEntry.isPrimary = true;
    const encodedAppId = encodeURIComponent(appId);
    const defaultPath = `${APP_PROXY_PREFIX}/${encodedAppId}/${APP_PROXY_SEGMENT}`;
    primaryEntry.path = defaultPath;

    const slugUsage = new Map();
    for (const entry of entries) {
        if (entry === primaryEntry) {
            continue;
        }

        let slug = entry.slug;
        const usageKey = `${entry.port}:${slug}`;
        let suffix = 1;
        while (slugUsage.has(usageKey)) {
            suffix += 1;
            slug = `${entry.slug}-${suffix}`;
        }
        slugUsage.set(usageKey, true);
        entry.slug = slug;
        entry.path = `${APP_PROXY_PREFIX}/${encodedAppId}/${APP_PORTS_SEGMENT}/${encodeURIComponent(slug)}/${APP_PROXY_SEGMENT}`;
    }

    const aliasIndex = buildPortAliasIndex(entries);

    return {
        appId,
        fetchedAt: Date.now(),
        entries,
        primaryEntry,
        aliasIndex,
        metadata,
    };
};

const selectPortEntry = (state, portKey) => {
    if (!state) {
        return null;
    }
    if (portKey == null || (typeof portKey === 'string' && portKey.trim() === '')) {
        return state.primaryEntry;
    }

    const normalizedKey = normalizePortLabel(portKey);
    if (normalizedKey && state.aliasIndex.has(normalizedKey)) {
        return state.aliasIndex.get(normalizedKey);
    }

    const numericPort = parseNumericPort(portKey);
    if (numericPort !== null) {
        const match = state.entries.find((entry) => entry.port === numericPort);
        if (match) {
            return match;
        }
    }

    return null;
};

const buildProxyBootstrapPayload = (state) => {
    const payloadEntries = state.entries.map((entry) => ({
        port: entry.port,
        label: entry.label,
        slug: entry.slug,
        path: entry.path,
        aliases: entry.aliases,
        source: entry.source,
        isPrimary: entry.isPrimary,
    }));

    return {
        appId: state.appId,
        generatedAt: Date.now(),
        hosts: LOOPBACK_HOST_ARRAY,
        primary: {
            port: state.primaryEntry.port,
            label: state.primaryEntry.label,
            slug: state.primaryEntry.slug,
            path: state.primaryEntry.path,
            aliases: state.primaryEntry.aliases,
        },
        ports: payloadEntries,
    };
};

const escapeForInlineScript = (value) => value.replace(/<\//g, '\\u003C/');

const buildProxyBootstrapScript = (state) => {
    const payload = buildProxyBootstrapPayload(state);
    const serialized = escapeForInlineScript(JSON.stringify(payload));

    return `;(() => {\n` +
        `  const INFO_KEY = '__APP_MONITOR_PROXY_INFO__';\n` +
        `  const INDEX_KEY = '__APP_MONITOR_PROXY_INDEX__';\n` +
        `  const payload = ${serialized};\n` +
        `  if (typeof window === 'undefined') {\n` +
        `    return;\n` +
        `  }\n` +
        `  window[INFO_KEY] = payload;\n` +
        `  const buildIndex = (data) => {\n` +
        `    if (!data || !Array.isArray(data.ports)) {\n` +
        `      return null;\n` +
        `    }\n` +
        `    const aliasMap = new Map();\n` +
        `    for (const entry of data.ports) {\n` +
        `      if (!entry || typeof entry.port !== 'number') {\n` +
        `        continue;\n` +
        `      }\n` +
        `      const aliases = Array.isArray(entry.aliases) ? entry.aliases : [];\n` +
        `      aliases.push(String(entry.port));\n` +
        `      for (const alias of aliases) {\n` +
        `        if (typeof alias !== 'string') {\n` +
        `          continue;\n` +
        `        }\n` +
        `        const key = alias.trim().toLowerCase();\n` +
        `        if (key && !aliasMap.has(key)) {\n` +
        `          aliasMap.set(key, entry);\n` +
        `        }\n` +
        `      }\n` +
        `    }\n` +
        `    return {\n` +
        `      appId: data.appId,\n` +
        `      generatedAt: data.generatedAt,\n` +
        `      aliasMap,\n` +
        `      primary: data.primary,\n` +
        `      hosts: new Set(Array.isArray(data.hosts) ? data.hosts.map((host) => host.toLowerCase()) : []),\n` +
        `    };\n` +
        `  };\n` +
        `  const getIndex = () => {\n` +
        `    const latest = window[INFO_KEY];\n` +
        `    if (!latest) {\n` +
        `      return null;\n` +
        `    }\n` +
        `    const existing = window[INDEX_KEY];\n` +
        `    if (!existing || existing.appId !== latest.appId || existing.generatedAt !== latest.generatedAt) {\n` +
        `      const nextIndex = buildIndex(latest);\n` +
        `      window[INDEX_KEY] = nextIndex;\n` +
        `      return nextIndex;\n` +
        `    }\n` +
        `    return existing;\n` +
        `  };\n` +
        `  const joinPath = (basePath, nextPath) => {\n` +
        `    const trimmedBase = typeof basePath === 'string' ? basePath.replace(/\\/$/, '') : '';\n` +
        `    const normalizedNext = typeof nextPath === 'string' ? nextPath : '';\n` +
        `    if (!normalizedNext || normalizedNext === '/') {\n` +
        `      return trimmedBase + '/';\n` +
        `    }\n` +
        `    return trimmedBase + (normalizedNext.startsWith('/') ? normalizedNext : '/' + normalizedNext);\n` +
        `  };\n` +
        `  const resolveTarget = (href, scheme) => {\n` +
        `    if (!href) {\n` +
        `      return null;\n` +
        `    }\n` +
        `    let url;\n` +
        `    try {\n` +
        `      url = new URL(href, window.location.href);\n` +
        `    } catch (error) {\n` +
        `      return null;\n` +
        `    }\n` +
        `    const index = getIndex();\n` +
        `    if (!index || !index.hosts.has(url.hostname.toLowerCase())) {\n` +
        `      return null;\n` +
        `    }\n` +
        `    const portKey = url.port ? url.port : '';\n` +
        `    let entry = portKey ? index.aliasMap.get(portKey.toLowerCase()) : null;\n` +
        `    if (!entry && !portKey) {\n` +
        `      entry = index.primary;\n` +
        `    }\n` +
        `    if (!entry) {\n` +
        `      return null;\n` +
        `    }\n` +
        `    const basePath = entry.path || index.primary?.path;\n` +
        `    if (!basePath) {\n` +
        `      return null;\n` +
        `    }\n` +
        `    const protocol = scheme === 'ws'\n` +
        `      ? (window.location.protocol === 'https:' ? 'wss:' : 'ws:')\n` +
        `      : window.location.protocol;\n` +
        `    const host = window.location.host;\n` +
        `    const path = joinPath(basePath, url.pathname || '/');\n` +
        `    const search = url.search || '';\n` +
        `    const hash = scheme === 'ws' ? '' : (url.hash || '');\n` +
        `    return protocol + '//' + host + path + search + hash;\n` +
        `  };\n` +
        `  const ensurePatched = () => {\n` +
        `    if (window.__APP_MONITOR_PROXY_SHIMMED__) {\n` +
        `      return;\n` +
        `    }\n` +
        `    window.__APP_MONITOR_PROXY_SHIMMED__ = true;\n` +
        `    const originalFetch = typeof window.fetch === 'function' ? window.fetch.bind(window) : null;\n` +
        `    if (originalFetch) {\n` +
        `      window.fetch = function(resource, init) {\n` +
        `        let rewritten = null;\n` +
        `        if (typeof resource === 'string' || resource instanceof URL) {\n` +
        `          rewritten = resolveTarget(resource.toString(), 'http');\n` +
        `          if (rewritten) {\n` +
        `            return originalFetch.call(this, rewritten, init);\n` +
        `          }\n` +
        `        } else if (resource instanceof Request) {\n` +
        `          rewritten = resolveTarget(resource.url, 'http');\n` +
        `          if (rewritten) {\n` +
        `            const cloned = resource.clone();\n` +
        `            const { method } = cloned;\n` +
        `            const headers = new Headers(cloned.headers);\n` +
        `            const bodyUsed = method && (method.toUpperCase() === 'GET' || method.toUpperCase() === 'HEAD');\n` +
        `            const initOptions = {\n` +
        `              method,\n` +
        `              headers,\n` +
        `              cache: cloned.cache,\n` +
        `              credentials: cloned.credentials,\n` +
        `              integrity: cloned.integrity,\n` +
        `              keepalive: cloned.keepalive,\n` +
        `              mode: cloned.mode,\n` +
        `              redirect: cloned.redirect,\n` +
        `              referrer: cloned.referrer && cloned.referrer !== 'about:client' ? cloned.referrer : undefined,\n` +
        `              referrerPolicy: cloned.referrerPolicy,\n` +
        `              signal: cloned.signal,\n` +
        `            };\n` +
        `            if (!bodyUsed) {\n` +
        `              initOptions.body = cloned.body;\n` +
        `            }\n` +
        `            if ('duplex' in cloned) {\n` +
        `              initOptions.duplex = cloned.duplex;\n` +
        `            }\n` +
        `            const rewrittenRequest = new Request(rewritten, initOptions);\n` +
        `            return originalFetch.call(this, rewrittenRequest, init);\n` +
        `          }\n` +
        `        }\n` +
        `        return originalFetch.call(this, resource, init);\n` +
        `      };\n` +
        `    }\n` +
        `    if (typeof window.Request === 'function') {\n` +
        `      const OriginalRequest = window.Request;\n` +
        `      const PatchedRequest = function(input, init) {\n` +
        `        let updatedInput = input;\n` +
        `        if (typeof input === 'string' || input instanceof URL) {\n` +
        `          const rewritten = resolveTarget(input.toString(), 'http');\n` +
        `          if (rewritten) {\n` +
        `            updatedInput = rewritten;\n` +
        `          }\n` +
        `        }\n` +
        `        return new OriginalRequest(updatedInput, init);\n` +
        `      };\n` +
        `      PatchedRequest.prototype = OriginalRequest.prototype;\n` +
        `      Object.setPrototypeOf(PatchedRequest, OriginalRequest);\n` +
        `      window.Request = PatchedRequest;\n` +
        `    }\n` +
        `    if (typeof window.XMLHttpRequest === 'function') {\n` +
        `      const originalOpen = window.XMLHttpRequest.prototype.open;\n` +
        `      window.XMLHttpRequest.prototype.open = function(method, url, async, user, password) {\n` +
        `        let rewrittenUrl = url;\n` +
        `        if (typeof url === 'string') {\n` +
        `          const rewritten = resolveTarget(url, 'http');\n` +
        `          if (rewritten) {\n` +
        `            rewrittenUrl = rewritten;\n` +
        `          }\n` +
        `        }\n` +
        `        return originalOpen.call(this, method, rewrittenUrl, async, user, password);\n` +
        `      };\n` +
        `    }\n` +
        `    if (typeof window.WebSocket === 'function') {\n` +
        `      const OriginalWebSocket = window.WebSocket;\n` +
        `      const PatchedWebSocket = function(target, protocols) {\n` +
        `        let url = target;\n` +
        `        if (typeof target === 'string' || target instanceof URL) {\n` +
        `          const rewritten = resolveTarget(target.toString(), 'ws');\n` +
        `          if (rewritten) {\n` +
        `            url = rewritten;\n` +
        `          }\n` +
        `        }\n` +
        `        if (protocols === undefined) {\n` +
        `          return new OriginalWebSocket(url);\n` +
        `        }\n` +
        `        return new OriginalWebSocket(url, protocols);\n` +
        `      };\n` +
        `      Object.setPrototypeOf(PatchedWebSocket, OriginalWebSocket);\n` +
        `      PatchedWebSocket.prototype = OriginalWebSocket.prototype;\n` +
        `      window.WebSocket = PatchedWebSocket;\n` +
        `    }\n` +
        `    if (typeof window.EventSource === 'function') {\n` +
        `      const OriginalEventSource = window.EventSource;\n` +
        `      const PatchedEventSource = function(url, config) {\n` +
        `        let rewrittenUrl = url;\n` +
        `        if (typeof url === 'string' || url instanceof URL) {\n` +
        `          const rewritten = resolveTarget(url.toString(), 'http');\n` +
        `          if (rewritten) {\n` +
        `            rewrittenUrl = rewritten;\n` +
        `          }\n` +
        `        }\n` +
        `        return new OriginalEventSource(rewrittenUrl, config);\n` +
        `      };\n` +
        `      PatchedEventSource.prototype = OriginalEventSource.prototype;\n` +
        `      Object.setPrototypeOf(PatchedEventSource, OriginalEventSource);\n` +
        `      window.EventSource = PatchedEventSource;\n` +
        `    }\n` +
        `    if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function') {\n` +
        `      const originalSendBeacon = navigator.sendBeacon.bind(navigator);\n` +
        `      navigator.sendBeacon = function(url, data) {\n` +
        `        const rewritten = resolveTarget(url, 'http');\n` +
        `        const targetUrl = rewritten || url;\n` +
        `        return originalSendBeacon(targetUrl, data);\n` +
        `      };\n` +
        `    }\n` +
        `  };\n` +
        `  ensurePatched();\n` +
        `})();`;
};

const fetchAppMetadata = async (appId) => {
    const url = `${API_BASE}/api/v1/apps/${encodeURIComponent(appId)}`;
    try {
        const response = await axios.get(url, { timeout: 10000 });
        if (response.data && typeof response.data === 'object') {
            if (response.data.success === false) {
                throw new Error(response.data.error || 'App lookup failed');
            }
            return response.data.data ?? response.data;
        }
        return response.data;
    } catch (error) {
        const status = error.response?.status;
        if (status === 404) {
            throw new Error(`App ${appId} not found`);
        }
        throw new Error(`Failed to fetch app metadata: ${error.message}`);
    }
};

const resolveAppProxyState = async (appId) => {
    const cached = APP_PROXY_CACHE.get(appId);
    if (cached && Date.now() - cached.fetchedAt < APP_CACHE_TTL_MS) {
        return cached;
    }

    if (APP_PROXY_INFLIGHT.has(appId)) {
        return APP_PROXY_INFLIGHT.get(appId);
    }

    const lookupPromise = (async () => {
        const metadata = await fetchAppMetadata(appId);
        const state = buildAppProxyState(appId, metadata);
        APP_PROXY_CACHE.set(appId, state);
        return state;
    })().finally(() => {
        APP_PROXY_INFLIGHT.delete(appId);
    });

    APP_PROXY_INFLIGHT.set(appId, lookupPromise);
    return lookupPromise;
};

const resolveAppProxyTarget = async (appId, portKey = null) => {
    const state = await resolveAppProxyState(appId);
    const entry = selectPortEntry(state, portKey);
    if (!entry) {
        throw new Error(portKey ? `App ${appId} does not expose a port matching "${portKey}"` : `App ${appId} does not expose a preview port`);
    }
    return {
        port: entry.port,
        entry,
        state,
    };
};

const sanitizeRequestHeaders = (headers = {}, port, req) => {
    const sanitized = {};
    for (const [key, value] of Object.entries(headers)) {
        if (HOP_BY_HOP_HEADERS.has(key.toLowerCase())) {
            continue;
        }
        sanitized[key] = value;
    }

    sanitized.host = `${LOOPBACK_HOST}:${port}`;
    sanitized.connection = 'close';
    const originalHost = headers.host || req?.headers?.host || `${LOOPBACK_HOST}:${PORT}`;
    sanitized['x-forwarded-host'] = originalHost;
    const protoHeader = headers['x-forwarded-proto'] || (req?.socket?.encrypted ? 'https' : 'http');
    sanitized['x-forwarded-proto'] = protoHeader;
    const remoteAddr = headers['x-forwarded-for'] || req?.socket?.remoteAddress || LOOPBACK_HOST;
    sanitized['x-forwarded-for'] = remoteAddr;
    sanitized['accept-encoding'] = 'identity';
    return sanitized;
};

const rewriteContentSecurityPolicy = (value) => {
    if (!value || typeof value !== 'string') {
        return "frame-ancestors 'self'";
    }
    const directives = value.split(';').map(part => part.trim()).filter(Boolean);
    const filtered = directives.filter(directive => !/^frame-ancestors\b/i.test(directive));
    filtered.push("frame-ancestors 'self'");
    return filtered.join('; ');
};

const rewriteLocationHeader = (location, appId, port) => {
    if (!location || typeof location !== 'string') {
        return location;
    }
    const appPrefix = `${APP_PROXY_PREFIX}/${encodeURIComponent(appId)}/${APP_PROXY_SEGMENT}`;
    try {
        const absolute = new URL(location, `http://${LOOPBACK_HOST}:${port}`);
        if (LOOPBACK_HOSTS.has(absolute.hostname) && Number(absolute.port || port) === Number(port)) {
            const path = absolute.pathname.startsWith('/') ? absolute.pathname : `/${absolute.pathname}`;
            return `${appPrefix}${path}${absolute.search || ''}${absolute.hash || ''}`;
        }
    } catch (error) {
        // If not a valid URL, fall back below
    }
    if (location.startsWith('/')) {
        return `${appPrefix}${location}`;
    }
    return location;
};

const rewriteSetCookieHeaders = (setCookie, appId) => {
    if (!setCookie) {
        return undefined;
    }
    const appPrefix = `${APP_PROXY_PREFIX}/${encodeURIComponent(appId)}/${APP_PROXY_SEGMENT}`;
    const ensureArray = Array.isArray(setCookie) ? setCookie : [setCookie];
    return ensureArray.map(cookie => {
        let rewritten = cookie.replace(/;\s*Domain=[^;]+/gi, '');
        if (/;\s*Path=/i.test(rewritten)) {
            rewritten = rewritten.replace(/;\s*Path=[^;]*/i, `; Path=${appPrefix}/`);
        } else {
            rewritten = `${rewritten}; Path=${appPrefix}/`;
        }
        return rewritten;
    });
};

const rewriteResponseHeaders = (headers, appId, port) => {
    const rewritten = {};
    let hasContentSecurityPolicy = false;

    for (const [key, value] of Object.entries(headers || {})) {
        const lower = key.toLowerCase();
        if (lower === 'x-frame-options') {
            continue;
        }
        if (lower === 'content-security-policy') {
            rewritten[key] = rewriteContentSecurityPolicy(value);
            hasContentSecurityPolicy = true;
            continue;
        }
        if (lower === 'location') {
            rewritten[key] = rewriteLocationHeader(value, appId, port);
            continue;
        }
        if (lower === 'set-cookie') {
            const cookies = rewriteSetCookieHeaders(value, appId);
            if (cookies) {
                rewritten[key] = cookies;
            }
            continue;
        }
        rewritten[key] = value;
    }

    if (!hasContentSecurityPolicy) {
        rewritten['Content-Security-Policy'] = "frame-ancestors 'self'";
    }

    return rewritten;
};

const stripProxyPrefix = (fullPath) => {
    if (!fullPath || typeof fullPath !== 'string') {
        return '/';
    }

    const portPattern = new RegExp(`^${APP_PROXY_PREFIX}/[^/]+/${APP_PORTS_SEGMENT}/[^/]+/${APP_PROXY_SEGMENT}`);
    const basePattern = new RegExp(`^${APP_PROXY_PREFIX}/[^/]+/${APP_PROXY_SEGMENT}`);

    let stripped = fullPath.replace(portPattern, '');
    if (stripped === fullPath) {
        stripped = fullPath.replace(basePattern, '');
    }
    if (!stripped) {
        return '/';
    }
    return stripped.startsWith('/') ? stripped : `/${stripped}`;
};

const normalizeHost = (host) => {
    if (typeof host !== 'string') {
        return '';
    }
    return host.trim().toLowerCase();
};

const ensureLeadingSlash = (value = '/') => {
    if (typeof value !== 'string') {
        return '/';
    }
    if (!value) {
        return '/';
    }
    return value.startsWith('/') ? value : `/${value}`;
};

const parseForAffinity = (value, fallbackHost = '') => {
    const normalizedFallbackHost = normalizeHost(fallbackHost);
    if (typeof value !== 'string' || value.length === 0) {
        return {
            host: normalizedFallbackHost,
            path: '/',
            search: '',
        };
    }

    try {
        const base = normalizedFallbackHost ? `http://${normalizedFallbackHost}` : 'http://placeholder';
        const parsed = new URL(value, base);
        return {
            host: normalizeHost(parsed.host || normalizedFallbackHost),
            path: parsed.pathname || '/',
            search: parsed.search || '',
        };
    } catch (error) {
        const questionIndex = value.indexOf('?');
        const pathPart = questionIndex === -1 ? value : value.slice(0, questionIndex);
        const searchPart = questionIndex === -1 ? '' : value.slice(questionIndex);
        return {
            host: normalizedFallbackHost,
            path: ensureLeadingSlash(pathPart || value),
            search: typeof searchPart === 'string' ? searchPart : '',
        };
    }
};

const extractProxyContextFromPath = (path) => {
    if (typeof path !== 'string') {
        return null;
    }
    const portMatch = path.match(new RegExp(`^${APP_PROXY_PREFIX}/([^/]+)/${APP_PORTS_SEGMENT}/([^/]+)/${APP_PROXY_SEGMENT}`));
    if (portMatch) {
        let appId = portMatch[1];
        let portKey = portMatch[2];
        try {
            appId = decodeURIComponent(appId);
        } catch (error) {
            // noop
        }
        try {
            portKey = decodeURIComponent(portKey);
        } catch (error) {
            // noop
        }
        return { appId, portKey };
    }
    const baseMatch = path.match(new RegExp(`^${APP_PROXY_PREFIX}/([^/]+)/${APP_PROXY_SEGMENT}`));
    const match = baseMatch;
    if (!match) {
        return null;
    }
    try {
        return { appId: decodeURIComponent(match[1]), portKey: null };
    } catch (error) {
        return { appId: match[1], portKey: null };
    }
};

const buildAffinityKey = ({ host, path, search }) => {
    const normalizedHost = normalizeHost(host);
    const normalizedPath = path || '/';
    return `${normalizedHost}||${normalizedPath}||${search || ''}`;
};

const normalizeRefererKey = (value, fallbackHost = '') => {
    if (typeof value !== 'string' || value.length === 0) {
        return null;
    }
    const details = parseForAffinity(value, fallbackHost);
    if (!details.host && (!details.path || details.path === '/')) {
        return null;
    }
    return buildAffinityKey(details);
};

const cleanupProxyAffinity = () => {
    if (proxyAffinityEntryCount === 0) {
        return;
    }

    const now = Date.now();

    for (const [key, bucket] of proxyAffinity.entries()) {
        for (const [refKey, entry] of Array.from(bucket.entries())) {
            if (entry.expiresAt <= now) {
                bucket.delete(refKey);
                proxyAffinityEntryCount -= 1;
            }
        }

        if (bucket.size === 0) {
            proxyAffinity.delete(key);
        }
    }

    if (proxyAffinityEntryCount <= PROXY_AFFINITY_MAX_ENTRIES) {
        return;
    }

    const entries = [];
    for (const [key, bucket] of proxyAffinity.entries()) {
        for (const [refKey, entry] of bucket.entries()) {
            entries.push({ key, refKey, entry });
        }
    }

    entries.sort((a, b) => a.entry.lastSeen - b.entry.lastSeen);
    const excess = proxyAffinityEntryCount - PROXY_AFFINITY_MAX_ENTRIES;

    for (let index = 0; index < excess && index < entries.length; index += 1) {
        const { key, refKey } = entries[index];
        const bucket = proxyAffinity.get(key);
        if (!bucket) {
            continue;
        }
        if (bucket.delete(refKey)) {
            proxyAffinityEntryCount -= 1;
        }
        if (bucket.size === 0) {
            proxyAffinity.delete(key);
        }
    }
};

const registerProxyAffinityEntry = ({ host, path, search, refererKey, appId }) => {
    const normalizedHost = normalizeHost(host);
    if (!normalizedHost || !path) {
        return;
    }

    const key = buildAffinityKey({ host: normalizedHost, path, search });
    const normalizedRefKey = refererKey ?? DEFAULT_REF_KEY;
    const now = Date.now();

    let bucket = proxyAffinity.get(key);
    if (!bucket) {
        bucket = new Map();
        proxyAffinity.set(key, bucket);
    }

    const existing = bucket.get(normalizedRefKey);
    if (!existing) {
        proxyAffinityEntryCount += 1;
    }

    bucket.set(normalizedRefKey, {
        appId,
        expiresAt: now + PROXY_AFFINITY_TTL_MS,
        lastSeen: now,
    });

    cleanupProxyAffinity();
};

const recordProxyAffinityForRequest = ({ req, appId, targetPath }) => {
    if (!req || !appId) {
        return;
    }

    const hostHeader = normalizeHost(req.headers.host || '');
    if (!hostHeader) {
        return;
    }

    const requestParts = parseForAffinity(req.originalUrl, hostHeader);
    if (!requestParts || !requestParts.path) {
        return;
    }

    const refererKey = normalizeRefererKey(req.headers.referer || req.headers.referrer, hostHeader);

    const requestKey = buildAffinityKey({
        host: hostHeader,
        path: requestParts.path,
        search: requestParts.search,
    });

    registerProxyAffinityEntry({
        host: hostHeader,
        path: requestParts.path,
        search: requestParts.search,
        refererKey,
        appId,
    });

    if (targetPath && typeof targetPath === 'string') {
        const targetParts = parseForAffinity(targetPath, hostHeader);
        if (
            targetParts &&
            targetParts.path &&
            (targetParts.path !== requestParts.path || targetParts.search !== requestParts.search)
        ) {
            const usesProxyPrefix =
                requestParts.path.startsWith(`${APP_PROXY_PREFIX}/`) &&
                requestParts.path.includes(`/${APP_PROXY_SEGMENT}`);
            const targetRefererKey = usesProxyPrefix ? requestKey : refererKey;
            registerProxyAffinityEntry({
                host: hostHeader,
                path: targetParts.path,
                search: targetParts.search,
                refererKey: targetRefererKey ?? refererKey,
                appId,
            });
        }
    }
};

const resolveAppIdForKey = (key, refererKey, visited = new Set()) => {
    cleanupProxyAffinity();

    if (!key || visited.has(key)) {
        return null;
    }

    visited.add(key);

    const bucket = proxyAffinity.get(key);
    if (!bucket) {
        return null;
    }

    const now = Date.now();
    const normalizedRefKey = refererKey ?? DEFAULT_REF_KEY;
    const direct = bucket.get(normalizedRefKey);
    if (direct && direct.expiresAt > now) {
        return direct.appId;
    }

    let freshest = null;

    for (const [storedRefKey, entry] of bucket.entries()) {
        if (entry.expiresAt <= now) {
            continue;
        }

        if (refererKey && storedRefKey === refererKey) {
            return entry.appId;
        }

        if (storedRefKey !== DEFAULT_REF_KEY && !visited.has(storedRefKey)) {
            const resolved = resolveAppIdForKey(storedRefKey, null, visited);
            if (resolved && resolved === entry.appId) {
                return entry.appId;
            }
        }

        if (!freshest || entry.lastSeen > freshest.lastSeen) {
            freshest = entry;
        }
    }

    return freshest ? freshest.appId : null;
};

const resolveProxyContextFromRequest = (req) => {
    if (!req) {
        return null;
    }

    const hostHeader = normalizeHost(req.headers.host || '');
    if (!hostHeader) {
        return null;
    }

    const requestParts = parseForAffinity(req.originalUrl, hostHeader);
    if (!requestParts || !requestParts.path) {
        return null;
    }

    const requestContext = extractProxyContextFromPath(requestParts.path);
    if (requestContext?.appId) {
        return requestContext;
    }

    const refererHeader = req.headers.referer || req.headers.referrer;
    let refererParts = null;
    let refererKey = null;
    let refererContext = null;
    if (typeof refererHeader === 'string' && refererHeader.length > 0) {
        refererParts = parseForAffinity(refererHeader, hostHeader);
        if (refererParts) {
            refererKey = buildAffinityKey(refererParts);
            refererContext = extractProxyContextFromPath(refererParts.path);
            if (refererContext?.appId) {
                return refererContext;
            }
        }
    }

    const requestKey = buildAffinityKey({
        host: hostHeader,
        path: requestParts.path,
        search: requestParts.search,
    });

    const direct = resolveAppIdForKey(requestKey, refererKey);
    if (direct) {
        return { appId: direct, portKey: refererContext?.portKey ?? requestContext?.portKey ?? null };
    }

    if (refererKey) {
        const refererAppId = resolveAppIdForKey(refererKey, null);
        if (refererAppId) {
            return { appId: refererAppId, portKey: refererContext?.portKey ?? null };
        }
    }

    return null;
};

const sanitizeWebSocketHeaders = (headers = {}, port, req) => {
    const sanitized = {};
    for (const [key, value] of Object.entries(headers)) {
        if (key.toLowerCase() === 'host') {
            continue;
        }
        sanitized[key] = value;
    }

    sanitized.host = `${LOOPBACK_HOST}:${port}`;
    sanitized.connection = 'Upgrade';
    sanitized.upgrade = 'websocket';
    const originalHost = headers.host || req?.headers?.host || `${LOOPBACK_HOST}:${PORT}`;
    sanitized['x-forwarded-host'] = originalHost;
    const protoHeader = headers['x-forwarded-proto'] || (req?.socket?.encrypted ? 'https' : 'http');
    sanitized['x-forwarded-proto'] = protoHeader;
    sanitized['x-forwarded-for'] = headers['x-forwarded-for'] || req?.socket?.remoteAddress || LOOPBACK_HOST;

    if (typeof headers.origin === 'string') {
        try {
            const originUrl = new URL(headers.origin);
            originUrl.host = `${LOOPBACK_HOST}:${port}`;
            sanitized.origin = originUrl.toString();
        } catch (error) {
            sanitized.origin = headers.origin;
        }
    }

    return sanitized;
};

const abortWebSocketUpgrade = (socket, statusCode, message) => {
    try {
        socket.write(`HTTP/1.1 ${statusCode} ${message}\r\nConnection: close\r\n\r\n`);
    } catch (error) {
        // ignore write errors during abort
    }
    socket.destroy();
};

const proxyWebSocketUpgrade = async ({ req, socket, head, appId, portKey, targetPath }) => {
    let target;
    try {
        target = await resolveAppProxyTarget(appId, portKey);
    } catch (error) {
        console.error('[APP PROXY][WS] Failed to resolve app:', error.message);
        abortWebSocketUpgrade(socket, 502, 'Bad Gateway');
        return;
    }

    const headers = sanitizeWebSocketHeaders(req.headers, target.port, req);
    const requestPath = targetPath && targetPath.length > 0 ? targetPath : '/';
    const requestLine = `${req.method} ${requestPath} HTTP/${req.httpVersion}\r\n`;

    const upstream = net.connect(target.port, LOOPBACK_HOST);

    upstream.on('connect', () => {
        upstream.write(requestLine);
        for (const [headerKey, headerValue] of Object.entries(headers)) {
            if (Array.isArray(headerValue)) {
                headerValue.forEach((entry) => {
                    upstream.write(`${headerKey}: ${entry}\r\n`);
                });
            } else {
                upstream.write(`${headerKey}: ${headerValue}\r\n`);
            }
        }
        upstream.write('\r\n');
        if (head && head.length > 0) {
            upstream.write(head);
        }

        upstream.pipe(socket);
        socket.pipe(upstream);
    });

    upstream.on('error', (error) => {
        console.error('[APP PROXY][WS] Upstream error:', error.message);
        if (error.code === 'ECONNREFUSED' || error.code === 'ECONNRESET') {
            APP_PROXY_CACHE.delete(appId);
        }
        socket.destroy();
    });

    socket.on('error', () => {
        upstream.destroy();
    });
};

const proxyHttpRequest = async ({ req, res, appId, portKey, targetPath }) => {
    let target;
    try {
        target = await resolveAppProxyTarget(appId, portKey);
    } catch (error) {
        console.error('[APP PROXY] Failed to resolve target:', error.message);
        res.status(502).json({
            error: 'Application preview unavailable',
            details: error.message,
            appId,
            port: portKey ?? undefined,
        });
        return;
    }

    const { port, state, entry } = target;
    const sanitizedHeaders = sanitizeRequestHeaders(req.headers, port, req);
    sanitizedHeaders['accept-encoding'] = 'identity';
    const requestPath = targetPath && targetPath.length > 0 ? targetPath : '/';

    const portLabel = entry?.label || entry?.slug || String(entry?.port);
    console.log(
        `[APP PROXY] ${req.method} ${req.originalUrl} -> http://${LOOPBACK_HOST}:${port}${requestPath}` +
            (portLabel ? ` [${portLabel}]` : '')
    );

    return new Promise((resolve) => {
        const proxyReq = http.request(
            {
                hostname: LOOPBACK_HOST,
                port,
                method: req.method,
                path: requestPath,
                headers: sanitizedHeaders,
            },
            (proxyRes) => {
                recordProxyAffinityForRequest({ req, appId, targetPath: requestPath });
                const rewrittenHeaders = rewriteResponseHeaders(proxyRes.headers, appId, port);
                const contentTypeHeader = proxyRes.headers['content-type'] || proxyRes.headers['Content-Type'];
                const shouldInjectBootstrap =
                    typeof contentTypeHeader === 'string' && /text\/html/i.test(contentTypeHeader);

                if (shouldInjectBootstrap) {
                    delete rewrittenHeaders['content-length'];
                    delete rewrittenHeaders['Content-Length'];
                }

                res.status(proxyRes.statusCode ?? 502);
                for (const [headerKey, headerValue] of Object.entries(rewrittenHeaders)) {
                    if (typeof headerValue === 'undefined') {
                        continue;
                    }
                    res.setHeader(headerKey, headerValue);
                }

                if (!shouldInjectBootstrap) {
                    proxyRes.pipe(res);
                    proxyRes.on('end', resolve);
                    return;
                }

                const chunks = [];
                proxyRes.on('data', (chunk) => {
                    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
                });
                proxyRes.on('end', () => {
                    try {
                        const bodyBuffer = Buffer.concat(chunks);
                        const body = bodyBuffer.toString('utf8');
                        const script = buildProxyBootstrapScript(state);
                        const scriptTag = `<script data-app-monitor="proxy-bootstrap">${script}</script>`;

                        let injected = body;
                        if (body.includes('</head>')) {
                            injected = body.replace('</head>', `${scriptTag}</head>`);
                        } else if (body.includes('<body')) {
                            const bodyIndex = body.indexOf('<body');
                            const closeIndex = body.indexOf('>', bodyIndex);
                            if (closeIndex !== -1) {
                                injected = `${body.slice(0, closeIndex + 1)}${scriptTag}${body.slice(closeIndex + 1)}`;
                            } else {
                                injected = `${scriptTag}${body}`;
                            }
                        } else {
                            injected = `${scriptTag}${body}`;
                        }

                        res.send(injected);
                    } catch (error) {
                        console.error('[APP PROXY] Failed to inject bootstrap script:', error.message);
                        res.send(Buffer.concat(chunks));
                    }
                    resolve();
                });
            }
        );

        proxyReq.on('error', (err) => {
            console.error('[APP PROXY] error:', err.message);
            if (err.code === 'ECONNREFUSED' || err.code === 'ECONNRESET') {
                APP_PROXY_CACHE.delete(appId);
            }
            if (!res.headersSent) {
                res.status(502).json({
                    error: 'Preview target unavailable',
                    details: err.message,
                    appId,
                });
            } else {
                res.end();
            }
            resolve();
        });

        const method = (req.method || 'GET').toUpperCase();
        if (method !== 'GET' && method !== 'HEAD') {
            req.pipe(proxyReq);
        } else {
            proxyReq.end();
        }
    });
};

const proxyStaticAssetRequest = async ({ req, res, appId, portKey }) => {
    let target;
    try {
        target = await resolveAppProxyTarget(appId, portKey ?? null);
    } catch (error) {
        return false;
    }

    const { port } = target;
    const sanitizedHeaders = sanitizeRequestHeaders(req.headers, port, req);
    sanitizedHeaders['accept-encoding'] = 'identity';

    return await new Promise((resolve) => {
        const upstreamReq = http.request(
            {
                hostname: LOOPBACK_HOST,
                port,
                method: req.method,
                path: req.originalUrl,
                headers: sanitizedHeaders,
            },
            (upstreamRes) => {
                const contentTypeHeader = upstreamRes.headers['content-type'] || upstreamRes.headers['Content-Type'];
                if (typeof contentTypeHeader === 'string' && /text\/html/i.test(contentTypeHeader)) {
                    upstreamRes.resume();
                    upstreamRes.on('end', () => resolve(false));
                    return;
                }

                recordProxyAffinityForRequest({ req, appId, targetPath: req.originalUrl });
                const rewrittenHeaders = rewriteResponseHeaders(upstreamRes.headers, appId, port);

                res.status(upstreamRes.statusCode ?? 200);
                for (const [headerKey, headerValue] of Object.entries(rewrittenHeaders)) {
                    if (typeof headerValue === 'undefined') {
                        continue;
                    }
                    res.setHeader(headerKey, headerValue);
                }

                upstreamRes.pipe(res);
                upstreamRes.on('end', () => resolve(true));
            }
        );

        upstreamReq.on('error', (error) => {
            if (error.code === 'ECONNREFUSED' || error.code === 'ECONNRESET') {
                APP_PROXY_CACHE.delete(appId);
            }
            resolve(false);
        });

        const method = (req.method || 'GET').toUpperCase();
        if (method === 'GET' || method === 'HEAD') {
            upstreamReq.end();
        } else {
            req.pipe(upstreamReq);
        }
    });
};

app.use(async (req, res, next) => {
    if (req.method !== 'GET' && req.method !== 'HEAD') {
        return next();
    }

    const path = req.path || req.originalUrl || '';
    if (!isProxiableRootAssetRequest(path)) {
        return next();
    }

    const context = resolveProxyContextFromRequest(req);
    if (!context?.appId) {
        return next();
    }

    try {
        const served = await proxyStaticAssetRequest({
            req,
            res,
            appId: context.appId,
            portKey: context.portKey ?? null,
        });
        if (served) {
            return;
        }
    } catch (error) {
        console.error('[APP PROXY] Root asset proxy failed:', error.message);
    }

    return next();
});

app.use(`${APP_PROXY_PREFIX}/:appId/${APP_PORTS_SEGMENT}/:portKey/${APP_PROXY_SEGMENT}`, async (req, res, next) => {
    try {
        const targetPath = stripProxyPrefix(req.originalUrl);
        await proxyHttpRequest({
            req,
            res,
            appId: req.params.appId,
            portKey: req.params.portKey,
            targetPath,
        });
        return;
    } catch (error) {
        console.error('[APP PROXY] Failed to proxy request:', error.message);
        if (!res.headersSent) {
            res.status(502).json({ error: 'Application preview unavailable', details: error.message });
        }
    }
    if (!res.headersSent) {
        next();
    }
});

app.get(`${APP_PROXY_PREFIX}/:appId/_proxy/metadata`, async (req, res) => {
    try {
        const state = await resolveAppProxyState(req.params.appId);
        res.json({
            success: true,
            data: buildProxyBootstrapPayload(state),
        });
    } catch (error) {
        res.status(404).json({
            success: false,
            error: error.message,
        });
    }
});

app.use(`${APP_PROXY_PREFIX}/:appId/${APP_PROXY_SEGMENT}`, async (req, res, next) => {
    try {
        const targetPath = stripProxyPrefix(req.originalUrl);
        await proxyHttpRequest({
            req,
            res,
            appId: req.params.appId,
            portKey: null,
            targetPath,
        });
        return;
    } catch (error) {
        console.error('[APP PROXY] Failed to proxy request:', error.message);
        if (!res.headersSent) {
            res.status(502).json({ error: 'Application preview unavailable', details: error.message });
        }
    }
    if (!res.headersSent) {
        next();
    }
});

app.use(async (req, res, next) => {
    if (req.originalUrl.startsWith(`${APP_PROXY_PREFIX}/`) && req.originalUrl.includes(`/${APP_PROXY_SEGMENT}`)) {
        return next();
    }

    const targetPath = stripProxyPrefix(req.originalUrl);
    const hostHeader = normalizeHost(req.headers.host || '');

    const refererHeader = req.headers.referer || req.headers.referrer;
    const fetchSite = req.headers['sec-fetch-site'];
    const refererDetails =
        typeof refererHeader === 'string' && refererHeader.length > 0
            ? parseForAffinity(refererHeader, hostHeader)
            : null;
    const refererContext = refererDetails ? extractProxyContextFromPath(refererDetails.path) : null;

    const affinityContext = resolveProxyContextFromRequest(req);
    if (affinityContext?.appId) {
        try {
            await proxyHttpRequest({
                req,
                res,
                appId: affinityContext.appId,
                portKey: affinityContext.portKey ?? refererContext?.portKey ?? null,
                targetPath,
            });
            return;
        } catch (error) {
            console.error('[APP PROXY] Affinity proxy failed:', error.message);
            if (res.headersSent) {
                return;
            }
        }
    }

    if (
        !refererDetails ||
        !refererContext ||
        (fetchSite && fetchSite !== 'same-origin' && fetchSite !== 'same-site' && fetchSite !== 'none')
    ) {
        return next();
    }

    if (refererDetails.host && hostHeader && refererDetails.host !== hostHeader) {
        return next();
    }

    const refererKey = buildAffinityKey(refererDetails);
    const refererAffinityAppId = resolveAppIdForKey(refererKey, null);
    if (refererAffinityAppId) {
        try {
            await proxyHttpRequest({
                req,
                res,
                appId: refererAffinityAppId,
                portKey: refererContext.portKey ?? null,
                targetPath,
            });
            return;
        } catch (error) {
            console.error('[APP PROXY] Referer-affinity proxy failed:', error.message);
            if (res.headersSent) {
                return;
            }
        }
    }

    if (!refererContext.appId) {
        return next();
    }

    try {
        await proxyHttpRequest({
            req,
            res,
            appId: refererContext.appId,
            portKey: refererContext.portKey ?? null,
            targetPath,
        });
        return;
    } catch (error) {
        console.error('[APP PROXY] Referer-based proxy failed:', error.message);
        if (!res.headersSent) {
            res.status(502).json({
                error: 'Application preview unavailable',
                details: error.message,
                appId: refererContext.appId,
            });
        }
    }
    if (!res.headersSent) {
        next();
    }
});

// Manual proxy function for API calls
function proxyToApi(req, res, apiPath) {
    const options = {
        hostname: LOOPBACK_HOST,
        port: API_PORT,
        path: apiPath || req.url,
        method: req.method,
        headers: {
            ...req.headers,
            host: `${LOOPBACK_HOST}:${API_PORT}`
        }
    };

    console.log(`[PROXY] ${req.method} ${req.url} -> http://${LOOPBACK_HOST}:${API_PORT}${options.path}`);

    const proxyReq = http.request(options, (proxyRes) => {
        res.status(proxyRes.statusCode);
        Object.keys(proxyRes.headers).forEach(key => {
            res.setHeader(key, proxyRes.headers[key]);
        });
        proxyRes.pipe(res);
    });

    proxyReq.on('error', (err) => {
        console.error('Proxy error:', err.message);
        res.status(502).json({
            error: 'API server unavailable',
            details: err.message,
            target: `http://${LOOPBACK_HOST}:${API_PORT}${options.path}`
        });
    });

    if (req.method !== 'GET' && req.method !== 'HEAD') {
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
}

// API endpoints proxy - support both versioned and unversioned for backward compatibility
app.use('/api/v1', (req, res) => {
    const fullApiPath = '/api/v1' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, fullApiPath);
});

// Legacy API proxy (redirect to v1)
app.use('/api', (req, res) => {
    // Skip if already versioned
    if (req.url.startsWith('/health')) {
        const fullApiPath = '/api' + req.url;
        proxyToApi(req, res, fullApiPath);
        return;
    }

    if (req.url.startsWith('/v1')) {
        const fullApiPath = '/api' + req.url;
        proxyToApi(req, res, fullApiPath);
    } else {
        // Redirect to v1
        const fullApiPath = '/api/v1' + (req.url.startsWith('/') ? req.url : '/' + req.url);
        proxyToApi(req, res, fullApiPath);
    }
});

// WebSocket handling
wss.on('connection', (ws) => {
    console.log('New WebSocket client connected');

    // Add error handling to prevent crashes
    ws.on('error', (error) => {
        console.error('WebSocket error:', error.message);
        // Don't crash the server on WebSocket errors
    });

    // Send initial connection message (wrapped in try-catch)
    try {
        ws.send(JSON.stringify({
            type: 'connection',
            payload: { status: 'connected' }
        }));
    } catch (error) {
        console.error('Failed to send initial message:', error.message);
        return;
    }

    // Real-time metric updates removed - UI now uses initial API fetch for accurate data
    // Mock setInterval was sending memory values up to 2048, causing >100% displays

    // Handle client messages
    ws.on('message', (message) => {
        try {
            const data = JSON.parse(message);
            console.log('Received:', data);

            // Handle different message types
            switch (data.type) {
                case 'subscribe':
                    // Subscribe to specific app updates
                    console.log(`Client subscribed to: ${data.appId}`);
                    break;
                case 'command':
                    // Execute command and send response
                    handleCommand(data.command, ws);
                    break;
            }
        } catch (error) {
            console.error('WebSocket message error:', error);
        }
    });

    ws.on('close', () => {
        console.log('WebSocket client disconnected');
        // No intervals to clear - metrics come from API calls
    });

    ws.on('error', (error) => {
        console.error('WebSocket error:', error);
    });
});

server.on('upgrade', async (req, socket, head) => {
    const hostHeader = req.headers.host || `localhost:${PORT}`;
    let parsed;
    try {
        parsed = new URL(req.url, `http://${hostHeader}`);
    } catch (error) {
        socket.destroy();
        return;
    }

    if (parsed.pathname === '/ws') {
        wss.handleUpgrade(req, socket, head, (ws) => {
            wss.emit('connection', ws, req);
        });
        return;
    }

    if (parsed.pathname.startsWith(`${APP_PROXY_PREFIX}/`) && parsed.pathname.includes(`/${APP_PROXY_SEGMENT}`)) {
        const context = extractProxyContextFromPath(parsed.pathname);
        if (!context?.appId) {
            abortWebSocketUpgrade(socket, 400, 'Bad Request');
            return;
        }

        const strippedPath = stripProxyPrefix(parsed.pathname);
        const targetPath = `${strippedPath}${parsed.search || ''}`;

        await proxyWebSocketUpgrade({
            req,
            socket,
            head,
            appId: context.appId,
            portKey: context.portKey ?? null,
            targetPath,
        });
        return;
    }

    socket.destroy();
});

// Handle commands from WebSocket
async function handleCommand(command, ws) {
    try {
        // Process command and send response
        const result = await processCommand(command);
        ws.send(JSON.stringify({
            type: 'command_response',
            payload: result
        }));
    } catch (error) {
        ws.send(JSON.stringify({
            type: 'error',
            payload: { message: error.message }
        }));
    }
}

async function processCommand(command) {
    // This would integrate with the actual system
    // For now, return mock responses
    const [cmd, ...args] = command.split(' ');

    switch (cmd) {
        case 'status':
            return { status: 'System operational', apps: 5, resources: 8 };
        case 'list':
            if (args[0] === 'apps') {
                return { apps: ['app1', 'app2', 'app3'] };
            }
            break;
        default:
            throw new Error(`Unknown command: ${cmd}`);
    }
}

// WebSocket broadcasts removed - all real-time data comes from actual system events
// Mock broadcasts were causing fake metric data (memory > 100%) to be sent to clients

// Set up periodic broadcasts
// Note: Real-time updates should be triggered by actual events from the system,
// not by periodic intervals with mock data
// setInterval(broadcastAppUpdate, 10000);  // DISABLED - should use real events
// setInterval(broadcastLogEntry, 15000);   // DISABLED - should use real events

// Health check endpoint (before static files)
app.get('/health', async (req, res) => {
    const healthResponse = {
        status: 'healthy',
        service: 'app-monitor-ui',
        timestamp: new Date().toISOString(),
        readiness: true,
        api_connectivity: {
            connected: false,
            api_url: `http://${LOOPBACK_HOST}:${API_PORT}`,
            last_check: new Date().toISOString(),
            error: null,
            latency_ms: null
        }
    };
    
    // Test API connectivity
    if (API_PORT) {
        const startTime = Date.now();
        
        try {
            await new Promise((resolve, reject) => {
                const options = {
                    hostname: LOOPBACK_HOST,
                    port: API_PORT,
                    path: '/health',
                    method: 'GET',
                    timeout: 5000,
                    headers: {
                        'Accept': 'application/json'
                    }
                };
                
                const req = http.request(options, (res) => {
                    const endTime = Date.now();
                    healthResponse.api_connectivity.latency_ms = endTime - startTime;
                    
                    if (res.statusCode >= 200 && res.statusCode < 300) {
                        healthResponse.api_connectivity.connected = true;
                        healthResponse.api_connectivity.error = null;
                    } else {
                        healthResponse.api_connectivity.connected = false;
                        healthResponse.api_connectivity.error = {
                            code: `HTTP_${res.statusCode}`,
                            message: `API returned status ${res.statusCode}: ${res.statusMessage}`,
                            category: 'network',
                            retryable: res.statusCode >= 500 && res.statusCode < 600
                        };
                        healthResponse.status = 'degraded';
                    }
                    resolve();
                });
                
                req.on('error', (error) => {
                    const endTime = Date.now();
                    healthResponse.api_connectivity.latency_ms = endTime - startTime;
                    healthResponse.api_connectivity.connected = false;
                    
                    let errorCode = 'CONNECTION_FAILED';
                    let category = 'network';
                    let retryable = true;
                    
                    if (error.code === 'ECONNREFUSED') {
                        errorCode = 'CONNECTION_REFUSED';
                    } else if (error.code === 'ENOTFOUND') {
                        errorCode = 'HOST_NOT_FOUND';
                        category = 'configuration';
                    } else if (error.code === 'ETIMEOUT') {
                        errorCode = 'TIMEOUT';
                    }
                    
                    healthResponse.api_connectivity.error = {
                        code: errorCode,
                        message: `Failed to connect to API: ${error.message}`,
                        category: category,
                        retryable: retryable,
                        details: {
                            error_code: error.code
                        }
                    };
                    healthResponse.status = 'unhealthy';
                    resolve();
                });
                
                req.on('timeout', () => {
                    const endTime = Date.now();
                    healthResponse.api_connectivity.latency_ms = endTime - startTime;
                    healthResponse.api_connectivity.connected = false;
                    healthResponse.api_connectivity.error = {
                        code: 'TIMEOUT',
                        message: 'API health check timed out after 5 seconds',
                        category: 'network',
                        retryable: true
                    };
                    healthResponse.status = 'unhealthy';
                    req.destroy();
                    resolve();
                });
                
                req.end();
            });
        } catch (error) {
            const endTime = Date.now();
            healthResponse.api_connectivity.latency_ms = endTime - startTime;
            healthResponse.api_connectivity.connected = false;
            healthResponse.api_connectivity.error = {
                code: 'UNEXPECTED_ERROR',
                message: `Unexpected error: ${error.message}`,
                category: 'internal',
                retryable: true
            };
            healthResponse.status = 'unhealthy';
        }
    } else {
        // No API_PORT configured
        healthResponse.api_connectivity.connected = false;
        healthResponse.api_connectivity.error = {
            code: 'MISSING_CONFIG',
            message: 'API_PORT environment variable not configured',
            category: 'configuration',
            retryable: false
        };
        healthResponse.status = 'degraded';
    }
    
    // Check WebSocket server health
    healthResponse.websocket = {
        connected: true,
        active_connections: wss.clients.size,
        error: null
    };
    
    // Verify WebSocket server is actually working
    if (!wss || typeof wss.clients === 'undefined') {
        healthResponse.websocket.connected = false;
        healthResponse.websocket.error = {
            code: 'WEBSOCKET_SERVER_DOWN',
            message: 'WebSocket server not initialized',
            category: 'internal',
            retryable: false
        };
        if (healthResponse.status === 'healthy') {
            healthResponse.status = 'degraded';
        }
    }
    
    // Check static file availability (in production)
    if (process.env.NODE_ENV === 'production') {
        const staticPath = path.join(__dirname, 'dist');
        const indexPath = path.join(staticPath, 'index.html');
        
        healthResponse.static_files = {
            available: false,
            error: null
        };
        
        try {
            if (fs.existsSync(indexPath)) {
                healthResponse.static_files.available = true;
            } else {
                healthResponse.static_files.error = {
                    code: 'STATIC_FILES_MISSING',
                    message: 'Production build files not found',
                    category: 'resource',
                    retryable: false
                };
                if (healthResponse.status === 'healthy') {
                    healthResponse.status = 'degraded';
                }
            }
        } catch (error) {
            healthResponse.static_files.error = {
                code: 'STATIC_FILES_CHECK_FAILED',
                message: `Cannot check static files: ${error.message}`,
                category: 'resource',
                retryable: false
            };
        }
    }
    
    // Add server metrics
    healthResponse.metrics = {
        uptime_seconds: process.uptime(),
        memory_usage_mb: process.memoryUsage().heapUsed / 1024 / 1024,
        websocket_clients: wss.clients.size
    };
    
    res.json(healthResponse);
});

// In production, serve built React files
if (process.env.NODE_ENV === 'production') {
    const staticPath = path.join(__dirname, 'dist');
    app.use(express.static(staticPath));

    // Catch all routes for client-side routing in production
    app.get('*', (req, res) => {
        // Skip API routes
        if (req.path.startsWith('/api')) {
            return res.status(404).json({ error: 'Not found' });
        }
        res.sendFile(path.join(staticPath, 'index.html'));
    });
} else {
    // In development, show helpful message
    app.get('/', (req, res) => {
        const vitePort = process.env.VITE_PORT;
        res.send(`
            <html>
                <head>
                    <title>App Monitor - Express Server</title>
                    <style>
                        body {
                            background: #0a0a0a;
                            color: #00ff41;
                            font-family: 'Courier New', monospace;
                            padding: 2rem;
                            text-align: center;
                        }
                        h1 { color: #39ff14; }
                        a {
                            color: #00ffff;
                            font-size: 1.2rem;
                        }
                        .info {
                            margin: 2rem;
                            padding: 1rem;
                            border: 1px solid #00ff41;
                        }
                    </style>
                </head>
                <body>
                    <h1>🚀 App Monitor Express Server</h1>
                    <div class="info">
                        <p>This is the Express backend server (port ${PORT})</p>
                        <p>It handles API proxying and WebSocket connections only.</p>
                        <br>
                        <p><strong>To access the React UI, go to:</strong></p>
                        <h2><a href="http://localhost:${vitePort}">http://localhost:${vitePort}</a></h2>
                        <br>
                        <p>If Vite is not running, start it with: <code>npm run dev</code></p>
                    </div>
                    <div class="info">
                        <p>Available endpoints on this server:</p>
                        <ul style="text-align: left; display: inline-block;">
                            <li>/api/* - Proxied to Go API server</li>
                            <li>/ws - WebSocket endpoint</li>
                            <li>/health - Health check</li>
                        </ul>
                    </div>
                </body>
            </html>
        `);
    });

    // Catch all other routes in development with helpful message
    app.get('*', (req, res) => {
        if (req.path.startsWith('/api')) {
            return res.status(404).json({ error: 'API endpoint not found' });
        }
        const vitePort = process.env.VITE_PORT;
        res.status(404).send(`
            <html>
                <head>
                    <title>404 - Wrong Server</title>
                    <style>
                        body {
                            background: #0a0a0a;
                            color: #ff0040;
                            font-family: 'Courier New', monospace;
                            padding: 2rem;
                            text-align: center;
                        }
                        a { color: #00ffff; }
                    </style>
                </head>
                <body>
                    <h1>❌ 404 - Wrong Server</h1>
                    <p>You're accessing the Express backend server.</p>
                    <p>The React UI is running on the Vite dev server:</p>
                    <h2><a href="http://localhost:${vitePort}${req.path}">http://localhost:${vitePort}${req.path}</a></h2>
                </body>
            </html>
        `);
    });
}

// Start server
server.listen(PORT, () => {
    console.log(`
╔════════════════════════════════════════════╗
║     VROOLI APP MONITOR - MATRIX UI        ║
║                                            ║
║  UI Server running on port ${PORT}           ║
║  WebSocket server active                   ║
║  API proxy to port ${API_PORT}                ║
║                                            ║
║  Access dashboard at:                      ║
║  http://localhost:${PORT}                     ║
╚════════════════════════════════════════════╝
    
🔍 DIAGNOSTIC INFO:
Working Directory: ${process.cwd()}
Script Path: ${__filename}
Process ID: ${process.pid}
Node Version: ${process.version}
Arguments: ${process.argv.join(' ')}
Environment: NODE_ENV=${process.env.NODE_ENV || 'undefined'}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('SIGTERM received, closing server...');
    wss.clients.forEach((client) => {
        client.close();
    });
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('\nSIGINT received, closing server...');
    wss.clients.forEach((client) => {
        client.close();
    });
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});
