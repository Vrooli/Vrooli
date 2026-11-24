// Shared SmartNotes and ZenNotes browser API and icon runtime.
window.NotesRuntime = (() => {
const PROXY_INFO_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__'];
const DEFAULT_API_SUFFIX = '/api';
const ICON_DEFAULTS = { size: 20, strokeWidth: 1.8 };
let lucideReady = Boolean(window.lucide);

function stripTrailingSlash(value = '') {
    return typeof value === 'string' ? value.replace(/\/+$/u, '') : '';
}

function ensureLeadingSlash(value = '') {
    if (typeof value !== 'string' || !value) {
        return '/';
    }
    return value.startsWith('/') ? value : `/${value}`;
}

function normalizePort(value) {
    if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
        return String(value);
    }
    if (typeof value === 'string') {
        const trimmed = value.trim();
        return /^\d+$/.test(trimmed) ? trimmed : undefined;
    }
    return undefined;
}

function pickProxyEndpoint(info) {
    if (!info) {
        return undefined;
    }

    const candidates = [];
    const enqueue = (candidate) => {
        if (!candidate) {
            return;
        }
        if (typeof candidate === 'string') {
            const trimmed = candidate.trim();
            if (trimmed) {
                candidates.push(trimmed);
            }
            return;
        }
        if (typeof candidate === 'object') {
            enqueue(candidate.url || candidate.href || candidate.target || candidate.path);
            if (Array.isArray(candidate.endpoints)) {
                candidate.endpoints.forEach(enqueue);
            }
        }
    };

    enqueue(info);

    if (typeof info === 'object') {
        enqueue(info.apiBase || info.api_path || info.apiPath);
        enqueue(info.api);
        enqueue(info.baseUrl || info.baseURL);
        enqueue(info.apiUrl || info.apiURL);
        enqueue(info.url);
        enqueue(info.target);
        if (Array.isArray(info.endpoints)) {
            info.endpoints.forEach(enqueue);
        }
    }

    return candidates.find(Boolean);
}

function resolveProxyEndpoint() {
    if (typeof window === 'undefined') {
        return undefined;
    }

    for (const key of PROXY_INFO_KEYS) {
        const info = window[key];
        if (!info) {
            continue;
        }

        const candidate = pickProxyEndpoint(info);
        if (candidate) {
            return candidate;
        }

        const ports = typeof info === 'object' ? info.ports : undefined;
        const rawPort = ports && (ports.api?.port ?? ports.api);
        const normalizedPort = normalizePort(rawPort);
        if (normalizedPort) {
            const protocol = window.location?.protocol === 'https:' ? 'https:' : 'http:';
            const hostname = window.location?.hostname || 'localhost';
            return `${protocol}//${hostname}:${normalizedPort}`;
        }
    }

    return undefined;
}

function ensureApiSuffix(value) {
    const trimmed = stripTrailingSlash((value || '').trim());
    if (!trimmed) {
        return DEFAULT_API_SUFFIX;
    }
    if (trimmed.endsWith(DEFAULT_API_SUFFIX)) {
        return trimmed;
    }
    if (/^https?:\/\//i.test(trimmed)) {
        return `${trimmed}${DEFAULT_API_SUFFIX}`;
    }
    return ensureLeadingSlash(trimmed === '/' ? DEFAULT_API_SUFFIX : `${trimmed}${DEFAULT_API_SUFFIX}`);
}

function resolveApiBase() {
    if (typeof window === 'undefined') {
        return DEFAULT_API_SUFFIX;
    }

    const proxyBase = resolveProxyEndpoint();
    if (proxyBase) {
        if (/^https?:\/\//i.test(proxyBase)) {
            return ensureApiSuffix(proxyBase);
        }
        const origin = window.location?.origin;
        if (origin) {
            return ensureApiSuffix(`${stripTrailingSlash(origin)}${ensureLeadingSlash(proxyBase)}`);
        }
        return ensureApiSuffix(proxyBase);
    }

    const envApiUrl = typeof window.API_URL === 'string' ? window.API_URL.trim() : '';
    if (envApiUrl) {
        return ensureApiSuffix(envApiUrl);
    }

    const apiPort = normalizePort(window.API_PORT);
    if (apiPort) {
        const protocol = window.location?.protocol === 'https:' ? 'https:' : 'http:';
        const host = window.location?.hostname || 'localhost';
        return ensureApiSuffix(`${protocol}//${host}:${apiPort}`);
    }

    const origin = window.location?.origin;
    if (origin) {
        return ensureApiSuffix(origin);
    }

    return DEFAULT_API_SUFFIX;
}

const API_BASE = resolveApiBase();

function joinApiUrl(path = '/') {
    const base = stripTrailingSlash(API_BASE);
    const suffix = ensureLeadingSlash(path);
    return `${base}${suffix}`;
}

function apiFetch(path, options = {}) {
    const target = joinApiUrl(path);
    return fetch(target, options);
}

async function apiFetchJson(path, options = {}) {
    const response = await apiFetch(path, options);
    const raw = await response.text();

    if (!response.ok) {
        const message = raw || `Request failed (${response.status})`;
        throw new Error(message);
    }

    if (!raw) {
        return null;
    }

    try {
        return JSON.parse(raw);
    } catch (error) {
        throw new Error(`Failed to parse response JSON: ${error instanceof Error ? error.message : String(error)}`);
    }
}

function buildJsonRequest(method, payload, headers = {}) {
    const options = {
        method,
        headers: {
            'Content-Type': 'application/json',
            ...headers,
        },
    };

    if (payload !== undefined) {
        options.body = JSON.stringify(payload);
    }

    return options;
}

function applyIcon(element, iconName, overrides = {}) {
    if (!element) {
        return;
    }

    const resolvedIcon = iconName || element.dataset.icon;
    if (!resolvedIcon) {
        element.textContent = overrides.fallback || '';
        return;
    }

    element.dataset.icon = resolvedIcon;

    const lucideApi = window.lucide;
    const iconNode = lucideApi?.icons?.[resolvedIcon];
    if (lucideReady && iconNode && typeof lucideApi?.createElement === 'function') {
        const size = Number(overrides.size ?? element.dataset.iconSize ?? ICON_DEFAULTS.size);
        const strokeWidth = Number(overrides.strokeWidth ?? element.dataset.iconStroke ?? ICON_DEFAULTS.strokeWidth);
        const svg = lucideApi.createElement(iconNode, { name: resolvedIcon, size, strokeWidth });
        svg.setAttribute('aria-hidden', 'true');
        element.replaceChildren(svg);
    } else if (!element.childElementCount && !element.textContent.trim()) {
        element.textContent = overrides.fallback || '';
    }
}

function refreshStaticIcons(root = document) {
    if (!root) {
        return;
    }
    const elements = root.querySelectorAll('[data-icon]');
    elements.forEach((node) => applyIcon(node, node.dataset.icon));
}

window.addEventListener('load', () => {
    if (window.lucide) {
        lucideReady = true;
        refreshStaticIcons();
    }
});

return Object.freeze({ API_BASE, apiFetch, apiFetchJson, buildJsonRequest, applyIcon, refreshStaticIcons });
})();
