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
const LOOPBACK_HOSTS = new Set([LOOPBACK_HOST, 'localhost', '::1', '[::1]', '0.0.0.0']);
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
const APP_PORT_CACHE = new Map();
const APP_PORT_INFLIGHT = new Map();

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

const fetchAppMetadata = async (appId) => {
    const url = `${API_BASE}/api/v1/apps/${encodeURIComponent(appId)}`;
    try {
        const response = await axios.get(url, { timeout: 5000 });
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

const resolveAppPreviewTarget = async (appId) => {
    const cached = APP_PORT_CACHE.get(appId);
    if (cached && Date.now() - cached.fetchedAt < APP_CACHE_TTL_MS) {
        return cached;
    }

    if (APP_PORT_INFLIGHT.has(appId)) {
        return APP_PORT_INFLIGHT.get(appId);
    }

    const lookupPromise = (async () => {
        const metadata = await fetchAppMetadata(appId);
        const port = computeAppUIPort(metadata);
        if (port === null) {
            throw new Error(`App ${appId} does not expose a preview port`);
        }
        const result = { port, fetchedAt: Date.now() };
        APP_PORT_CACHE.set(appId, result);
        return result;
    })().finally(() => {
        APP_PORT_INFLIGHT.delete(appId);
    });

    APP_PORT_INFLIGHT.set(appId, lookupPromise);
    return lookupPromise;
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

    const pattern = new RegExp(`^${APP_PROXY_PREFIX}/[^/]+/${APP_PROXY_SEGMENT}`);
    const stripped = fullPath.replace(pattern, '');
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

const extractAppIdFromProxyPath = (path) => {
    if (typeof path !== 'string') {
        return null;
    }
    const match = path.match(new RegExp(`^${APP_PROXY_PREFIX}/([^/]+)/${APP_PROXY_SEGMENT}`));
    if (!match) {
        return null;
    }
    try {
        return decodeURIComponent(match[1]);
    } catch (error) {
        return match[1];
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

const resolveAppIdFromRequest = (req) => {
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

    const refererHeader = req.headers.referer || req.headers.referrer;
    let refererParts = null;
    let refererKey = null;
    if (typeof refererHeader === 'string' && refererHeader.length > 0) {
        refererParts = parseForAffinity(refererHeader, hostHeader);
        if (refererParts) {
            refererKey = buildAffinityKey(refererParts);
            const refererAppId = extractAppIdFromProxyPath(refererParts.path);
            if (refererAppId) {
                return refererAppId;
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
        return direct;
    }

    if (refererKey) {
        return resolveAppIdForKey(refererKey, null);
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

const proxyWebSocketUpgrade = async ({ req, socket, head, appId, targetPath }) => {
    let target;
    try {
        target = await resolveAppPreviewTarget(appId);
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
            APP_PORT_CACHE.delete(appId);
        }
        socket.destroy();
    });

    socket.on('error', () => {
        upstream.destroy();
    });
};

const proxyHttpRequest = async ({ req, res, appId, targetPath }) => {
    const { port } = await resolveAppPreviewTarget(appId);
    const sanitizedHeaders = sanitizeRequestHeaders(req.headers, port, req);
    const requestPath = targetPath && targetPath.length > 0 ? targetPath : '/';

    console.log(`[APP PROXY] ${req.method} ${req.originalUrl} -> http://${LOOPBACK_HOST}:${port}${requestPath}`);

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
                res.status(proxyRes.statusCode ?? 502);
                for (const [headerKey, headerValue] of Object.entries(rewrittenHeaders)) {
                    if (typeof headerValue === 'undefined') {
                        continue;
                    }
                    res.setHeader(headerKey, headerValue);
                }
                proxyRes.pipe(res);
                proxyRes.on('end', resolve);
            }
        );

        proxyReq.on('error', (err) => {
            console.error('[APP PROXY] error:', err.message);
            if (err.code === 'ECONNREFUSED' || err.code === 'ECONNRESET') {
                APP_PORT_CACHE.delete(appId);
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

app.use(`${APP_PROXY_PREFIX}/:appId/${APP_PROXY_SEGMENT}`, async (req, res, next) => {
    try {
        const targetPath = stripProxyPrefix(req.originalUrl);
        await proxyHttpRequest({
            req,
            res,
            appId: req.params.appId,
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
    const affinityAppId = resolveAppIdFromRequest(req);
    if (affinityAppId) {
        try {
            await proxyHttpRequest({
                req,
                res,
                appId: affinityAppId,
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

    const referer = req.headers.referer || req.headers.referrer;
    if (!referer || typeof referer !== 'string') {
        return next();
    }

    const fetchSite = req.headers['sec-fetch-site'];
    if (fetchSite && fetchSite !== 'same-origin' && fetchSite !== 'same-site' && fetchSite !== 'none') {
        return next();
    }

    const hostHeader = normalizeHost(req.headers.host || '');
    const refererDetails = parseForAffinity(referer, hostHeader);
    if (!refererDetails) {
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

    if (!refererDetails.path.startsWith(`${APP_PROXY_PREFIX}/`) || !refererDetails.path.includes(`/${APP_PROXY_SEGMENT}`)) {
        return next();
    }

    if (refererDetails.host && hostHeader && refererDetails.host !== hostHeader) {
        return next();
    }

    const appId = extractAppIdFromProxyPath(refererDetails.path);
    if (!appId) {
        return next();
    }

    try {
        await proxyHttpRequest({
            req,
            res,
            appId,
            targetPath,
        });
        return;
    } catch (error) {
        console.error('[APP PROXY] Referer-based proxy failed:', error.message);
        if (!res.headersSent) {
            res.status(502).json({ error: 'Application preview unavailable', details: error.message, appId });
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
        const match = parsed.pathname.match(new RegExp(`^${APP_PROXY_PREFIX}/([^/]+)/${APP_PROXY_SEGMENT}(.*)?$`));
        if (!match) {
            abortWebSocketUpgrade(socket, 400, 'Bad Request');
            return;
        }

        const appId = decodeURIComponent(match[1]);
        const strippedPath = stripProxyPrefix(parsed.pathname);
        const targetPath = `${strippedPath}${parsed.search || ''}`;

        await proxyWebSocketUpgrade({ req, socket, head, appId, targetPath });
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
