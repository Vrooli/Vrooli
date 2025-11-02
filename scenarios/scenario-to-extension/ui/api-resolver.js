/**
 * API Resolution & Proxy Detection
 * Handles automatic detection of API endpoints via App Monitor proxy metadata
 * and fallback to loopback/localhost for development
 *
 * This file must be loaded before app.js as it provides:
 * - resolveApiConfig() - Main entry point
 * - applyApiResolution() - Apply resolved config
 * - startApiResolutionWatcher() - Background watcher for proxy detection
 */

// Constants (must match app.js)
const API_PATH = '/api/v1';
const DEFAULT_API_PORT = 3201;
const PROXY_INFO_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__'];
const PROXY_RESOLUTION_POLL_INTERVAL_MS = 750;
const PROXY_RESOLUTION_POLL_LIMIT = 40;
const LOOPBACK_SOURCE = 'loopback-fallback';
const API_RESOLUTION_EVENT = 'scenario-to-extension:api-resolution';

// ============================================
// Main API Resolution Logic
// ============================================

function resolveApiConfig() {
    const win = typeof window !== 'undefined' ? window : undefined;
    const loopbackRoot = buildLoopbackRoot(win);
    const loopbackBase = ensureApiPath(loopbackRoot);

    const resolution = {
        base: loopbackBase,
        root: stripApiSuffix(loopbackBase),
        source: LOOPBACK_SOURCE,
        notes: 'Using default loopback developer API base'
    };

    if (!win) {
        return resolution;
    }

    const proxyBootstrap = readProxyBootstrap(win);
    const proxyBase = resolveProxyBase(proxyBootstrap, win);
    if (proxyBase) {
        const ensured = ensureApiPath(proxyBase);
        return {
            base: ensured,
            root: stripApiSuffix(ensured),
            source: 'app-monitor-proxy',
            notes: 'Resolved via App Monitor proxy metadata'
        };
    }

    const derivedFromLocation = deriveProxyBaseFromLocation(win);
    if (derivedFromLocation) {
        const ensured = ensureApiPath(derivedFromLocation);
        return {
            base: ensured,
            root: stripApiSuffix(ensured),
            source: 'location-derived',
            notes: 'Derived from current location pathname'
        };
    }

    const { origin, hostname } = win.location || {};
    if (origin && hostname && !isLocalHostname(hostname)) {
        const ensured = ensureApiPath(origin);
        return {
            base: ensured,
            root: stripApiSuffix(ensured),
            source: 'same-origin',
            notes: 'Using current origin as API base'
        };
    }

    return resolution;
}

// ============================================
// Proxy Detection & Bootstrap
// ============================================

function buildLoopbackRoot(win, port = DEFAULT_API_PORT) {
    const protocol = win?.location?.protocol || 'http:';
    const normalizedPort = Number.isInteger(port) && port > 0 && port < 65536 ? port : DEFAULT_API_PORT;
    return `${protocol}//127.0.0.1:${normalizedPort}`;
}

function readProxyBootstrap(win) {
    if (!win) {
        return undefined;
    }

    for (const key of PROXY_INFO_KEYS) {
        if (Object.prototype.hasOwnProperty.call(win, key) && win[key]) {
            return win[key];
        }
    }

    return undefined;
}

function resolveProxyBase(proxyInfo, win) {
    if (!proxyInfo || typeof proxyInfo !== 'object') {
        return undefined;
    }

    const prioritized = [];
    const preferredEntry = selectPreferredProxyEntry(proxyInfo);
    if (preferredEntry?.path) {
        prioritized.push(preferredEntry.path);
    }
    if (preferredEntry?.port) {
        prioritized.push(preferredEntry.port);
    }

    const candidates = collectProxyCandidates(proxyInfo);
    const derived = deriveProxyBaseFromLocation(win);
    if (derived) {
        candidates.push(derived);
    }

    const ordered = prioritizeProxyCandidates([...prioritized, ...candidates]);
    const seen = new Set();

    for (const candidate of ordered) {
        const normalized = normalizeProxyCandidate(candidate, win);
        if (!normalized || seen.has(normalized)) {
            continue;
        }
        seen.add(normalized);
        return normalized;
    }

    return undefined;
}

// ============================================
// Proxy Candidate Collection & Scoring
// ============================================

function collectProxyCandidates(source) {
    const queue = [source];
    const seen = new Set();
    const candidates = [];
    const keys = [
        'apiBase',
        'apiUrl',
        'api',
        'url',
        'baseUrl',
        'publicUrl',
        'proxyUrl',
        'origin',
        'target',
        'path',
        'proxyPath',
        'paths',
        'endpoint',
        'endpoints',
        'service',
        'services',
        'host',
        'hosts',
        'port',
        'ports',
        'httpProxyBase',
        'http_proxy_base'
    ];

    while (queue.length > 0) {
        const value = queue.shift();
        if (!value) {
            continue;
        }

        if (typeof value === 'string') {
            candidates.push(value);
            continue;
        }

        if (Array.isArray(value)) {
            queue.push(...value);
            continue;
        }

        if (typeof value === 'object') {
            if (seen.has(value)) {
                continue;
            }
            seen.add(value);
            keys.forEach((key) => {
                if (Object.prototype.hasOwnProperty.call(value, key)) {
                    queue.push(value[key]);
                }
            });
        }
    }

    return candidates;
}

function selectPreferredProxyEntry(proxyInfo) {
    const ports = Array.isArray(proxyInfo.ports) ? proxyInfo.ports : [];
    if (ports.length === 0) {
        return undefined;
    }

    const scored = ports
        .map((entry) => ({ entry, score: scoreProxyPortEntry(entry) }))
        .filter(({ score }) => Number.isFinite(score) && score > 0)
        .sort((a, b) => b.score - a.score);

    return scored.length > 0 ? scored[0].entry : undefined;
}

function scoreProxyPortEntry(entry) {
    if (!entry || typeof entry !== 'object') {
        return Number.NEGATIVE_INFINITY;
    }

    let score = 0;
    const textParts = [];

    if (typeof entry.label === 'string') {
        textParts.push(entry.label);
    }
    if (typeof entry.slug === 'string') {
        textParts.push(entry.slug);
    }
    if (Array.isArray(entry.aliases)) {
        entry.aliases.forEach((alias) => {
            if (typeof alias === 'string') {
                textParts.push(alias);
            }
        });
    }
    if (typeof entry.source === 'string') {
        textParts.push(entry.source);
    }
    if (typeof entry.kind === 'string') {
        textParts.push(entry.kind);
    }

    const normalizedText = textParts.join(' ').toLowerCase();
    if (/\bapi\b/.test(normalizedText)) {
        score += 90;
    } else if (normalizedText.includes('api')) {
        score += 60;
    }
    if (normalizedText.includes('backend')) {
        score += 30;
    }
    if (normalizedText.includes('server')) {
        score += 20;
    }
    if (normalizedText.includes('service')) {
        score += 12;
    }
    if (normalizedText.includes('core')) {
        score += 8;
    }
    if (normalizedText.includes('ui')) {
        score -= 60;
    }
    if (normalizedText.includes('front')) {
        score -= 40;
    }
    if (normalizedText.includes('client')) {
        score -= 30;
    }
    if (normalizedText.includes('asset') || normalizedText.includes('static')) {
        score -= 25;
    }
    if (normalizedText.includes('primary')) {
        score -= 15;
    }

    if (Number.isFinite(entry.port)) {
        if (entry.port >= 15000 && entry.port < 25000) {
            score += 12;
        }
        if (entry.port >= 30000 && entry.port < 40000) {
            score -= 8;
        }
        if (entry.port === DEFAULT_API_PORT) {
            score += 6;
        }
    }

    if (entry.isPrimary) {
        score -= 10;
    }

    return score;
}

function prioritizeProxyCandidates(candidates) {
    const scored = candidates.map((candidate) => ({ candidate, score: scoreProxyCandidate(candidate) }));
    scored.sort((a, b) => b.score - a.score);

    const seen = new Set();
    const ordered = [];

    for (const { candidate } of scored) {
        const key = typeof candidate === 'string' ? candidate : JSON.stringify(candidate);
        if (!key || seen.has(key)) {
            continue;
        }
        seen.add(key);
        ordered.push(candidate);
    }

    return ordered;
}

function scoreProxyCandidate(candidate) {
    if (candidate === null || candidate === undefined) {
        return Number.NEGATIVE_INFINITY;
    }

    if (typeof candidate === 'number') {
        return Number.isFinite(candidate) && candidate > 0 ? 40 : Number.NEGATIVE_INFINITY;
    }

    if (typeof candidate !== 'string') {
        return Number.NEGATIVE_INFINITY;
    }

    const trimmed = candidate.trim();
    if (!trimmed) {
        return Number.NEGATIVE_INFINITY;
    }

    const normalized = trimmed.toLowerCase();
    let score = 0;

    if (/^https?:\/\//.test(normalized)) {
        score += 18;
    }
    if (normalized.startsWith('/')) {
        score += 10;
    }
    if (normalized.includes('proxy')) {
        score += 8;
    }
    if (/(^|\b)api(\b|$)/.test(normalized)) {
        score += 90;
    } else if (normalized.includes('api')) {
        score += 60;
    }
    if (normalized.includes('backend')) {
        score += 30;
    }
    if (normalized.includes('server')) {
        score += 18;
    }
    if (normalized.includes('service')) {
        score += 12;
    }
    if (normalized.includes('core')) {
        score += 6;
    }
    if (normalized.includes('ui')) {
        score -= 55;
    }
    if (normalized.includes('front')) {
        score -= 35;
    }
    if (normalized.includes('client')) {
        score -= 30;
    }
    if (normalized.includes('asset') || normalized.includes('static')) {
        score -= 25;
    }
    if (normalized.includes('primary')) {
        score -= 15;
    }
    if (/\d{2,}/.test(normalized)) {
        score += 6;
    }

    return score;
}

// ============================================
// Candidate Normalization
// ============================================

function normalizeProxyCandidate(candidate, win) {
    if (candidate === null || candidate === undefined) {
        return undefined;
    }

    if (typeof candidate === 'number') {
        const port = Math.round(candidate);
        if (port > 0 && port < 65536) {
            return stripTrailingSlash(buildLoopbackRoot(win, port));
        }
        return undefined;
    }

    if (typeof candidate !== 'string') {
        return undefined;
    }

    const trimmed = candidate.trim();
    if (!trimmed) {
        return undefined;
    }

    if (/^\d+$/.test(trimmed)) {
        const numericPort = Number.parseInt(trimmed, 10);
        if (numericPort > 0 && numericPort < 65536) {
            return stripTrailingSlash(buildLoopbackRoot(win, numericPort));
        }
    }

    if (/^https?:\/\//i.test(trimmed)) {
        return stripTrailingSlash(trimmed);
    }

    if (trimmed.startsWith('//')) {
        const protocol = win?.location?.protocol || 'https:';
        return stripTrailingSlash(`${protocol}${trimmed}`);
    }

    if (trimmed.startsWith('/')) {
        const origin = win?.location?.origin;
        if (origin) {
            return stripTrailingSlash(`${origin}${trimmed}`);
        }
        return stripTrailingSlash(trimmed);
    }

    if (/^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)/i.test(trimmed)) {
        return stripTrailingSlash(`http://${trimmed}`);
    }

    const origin = win?.location?.origin;
    if (origin) {
        return stripTrailingSlash(`${origin}/${trimmed.replace(/^\/+/, '')}`);
    }

    return stripTrailingSlash(`/${trimmed.replace(/^\/+/, '')}`);
}

function deriveProxyBaseFromLocation(win) {
    if (!win?.location) {
        return undefined;
    }

    const { origin, pathname } = win.location;
    if (!origin || !pathname) {
        return undefined;
    }

    if (!/\/proxy(\b|\/)/i.test(pathname)) {
        return undefined;
    }

    return `${origin}${pathname}`.replace(/\/+$/, '');
}

// ============================================
// String Utilities
// ============================================

function stripTrailingSlash(value) {
    if (typeof value !== 'string') {
        return value;
    }
    return value.replace(/\/+$/, '');
}

function ensureApiPath(base) {
    if (typeof base !== 'string' || !base) {
        return API_PATH;
    }

    const normalized = stripTrailingSlash(base);
    if (normalized.endsWith(API_PATH)) {
        return normalized;
    }

    if (normalized.endsWith('/api')) {
        return `${normalized}${API_PATH.slice(4)}`;
    }

    return `${normalized}${API_PATH.startsWith('/') ? '' : '/'}${API_PATH}`;
}

function stripApiSuffix(value) {
    if (typeof value !== 'string') {
        return value;
    }

    const normalized = stripTrailingSlash(value);
    if (normalized.endsWith(API_PATH)) {
        return normalized.slice(0, -API_PATH.length) || normalized;
    }

    return normalized;
}

function isLocalHostname(value) {
    if (typeof value !== 'string') {
        return false;
    }

    return /^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)/i.test(value);
}

// ============================================
// Resolution Application (interfaces with app.js)
// ============================================

// Note: applyApiResolution and startApiResolutionWatcher are defined here
// but depend on variables from app.js (CONFIG, apiResolution, proxyWatcherStarted)
// They will be called from app.js after initialization

function applyApiResolution(resolution, { force = false, silent = false } = {}) {
    if (!resolution || typeof resolution.base !== 'string') {
        return;
    }

    // These variables come from app.js global scope
    const previous = window.__currentApiResolution;
    const changed =
        force ||
        !previous ||
        previous.base !== resolution.base ||
        previous.root !== resolution.root ||
        previous.source !== resolution.source;

    window.__currentApiResolution = resolution;

    if (typeof window.CONFIG !== 'undefined') {
        window.CONFIG.API_BASE = resolution.base;
        window.CONFIG.API_ROOT = resolution.root;
        window.CONFIG.API_SOURCE = resolution.source;
        window.CONFIG.API_NOTES = resolution.notes;
    }

    if (typeof window !== 'undefined') {
        window.__scenarioToExtensionApi = resolution;

        if (changed && !silent) {
            try {
                window.dispatchEvent(new CustomEvent(API_RESOLUTION_EVENT, { detail: resolution }));
            } catch (error) {
                if (window.CONFIG?.DEBUG) {
                    console.warn('[APIResolver] Failed to broadcast API resolution change', error);
                }
            }
        }
    }

    if (changed && typeof window.syncApiEndpointField === 'function') {
        window.syncApiEndpointField(previous?.root);
    }

    if (changed && typeof window.applyApiContext === 'function') {
        window.applyApiContext();
    }
}

function startApiResolutionWatcher() {
    if (typeof window === 'undefined' || window.__proxyWatcherStarted) {
        return;
    }

    window.__proxyWatcherStarted = true;

    const limit = Math.max(1, PROXY_RESOLUTION_POLL_LIMIT);
    const intervalMs = Math.max(100, PROXY_RESOLUTION_POLL_INTERVAL_MS);
    let attempts = 0;
    let timerId;

    const refresh = (options = {}) => {
        const next = resolveApiConfig();
        applyApiResolution(next, options);
        return next;
    };

    const stop = () => {
        if (timerId) {
            clearInterval(timerId);
            timerId = undefined;
        }
    };

    const initial = refresh({ force: true, silent: true });
    if (!initial || initial.source !== LOOPBACK_SOURCE) {
        return;
    }

    timerId = setInterval(() => {
        attempts += 1;
        const next = refresh({ silent: true });
        const proxyDetected = next && next.source && next.source !== LOOPBACK_SOURCE;
        if (proxyDetected || attempts >= limit) {
            stop();
        }
    }, intervalMs);

    window.addEventListener('message', (event) => {
        if (!event || typeof event.data !== 'object' || event.data === null) {
            return;
        }

        const payload = event.data;
        if (payload.__APP_MONITOR_PROXY_INFO__ || payload.type === 'APP_MONITOR_PROXY_INFO') {
            const next = refresh();
            if (next && next.source && next.source !== LOOPBACK_SOURCE) {
                stop();
            }
        }
    });
}

// Export functions to global scope for app.js to use
if (typeof window !== 'undefined') {
    window.resolveApiConfig = resolveApiConfig;
    window.applyApiResolution = applyApiResolution;
    window.startApiResolutionWatcher = startApiResolutionWatcher;
}
