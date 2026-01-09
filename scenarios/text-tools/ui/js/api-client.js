// API Client Module

const PROXY_INFO_KEY = '__APP_MONITOR_PROXY_INFO__';

const trimTrailingSlashes = (value = '') => value.replace(/\/+$/, '');
const ensureLeadingSlash = (value = '') => (value.startsWith('/') ? value : `/${value}`);

const resolveProxyCandidate = (info) => {
    const pushCandidate = (list, candidate) => {
        if (!candidate) {
            return;
        }

        if (typeof candidate === 'string') {
            list.push(candidate);
            return;
        }

        if (typeof candidate === 'object') {
            pushCandidate(list, candidate.url || candidate.path || candidate.target || candidate.href);
        }
    };

    const candidates = [];

    if (Array.isArray(info?.endpoints)) {
        info.endpoints.forEach((endpoint) => pushCandidate(candidates, endpoint));
    }

    pushCandidate(candidates, info?.api);
    pushCandidate(candidates, info?.apiUrl || info?.apiURL);
    pushCandidate(candidates, info?.baseUrl || info?.baseURL);
    pushCandidate(candidates, info?.url);
    pushCandidate(candidates, info?.target);

    if (typeof info === 'string') {
        pushCandidate(candidates, info);
    }

    for (const rawCandidate of candidates) {
        if (typeof rawCandidate !== 'string') {
            continue;
        }

        const candidate = rawCandidate.trim();
        if (!candidate) {
            continue;
        }

        if (/^https?:\/\//i.test(candidate)) {
            return trimTrailingSlashes(candidate);
        }

        if (candidate.startsWith('/') && typeof window !== 'undefined' && window.location?.origin) {
            return trimTrailingSlashes(`${window.location.origin}${candidate}`);
        }
    }

    return undefined;
};

const resolveTextToolsApiOrigin = () => {
    if (typeof window === 'undefined') {
        return undefined;
    }

    const proxyInfo = window[PROXY_INFO_KEY];
    const proxyBase = resolveProxyCandidate(proxyInfo);
    if (proxyBase) {
        return proxyBase;
    }

    const port = window.API_PORT;
    if (port) {
        const protocol = window.location?.protocol === 'https:' ? 'https:' : 'http:';
        return trimTrailingSlashes(`${protocol}//localhost:${port}`);
    }

    if (window.location?.origin) {
        return trimTrailingSlashes(window.location.origin);
    }

    return undefined;
};

const joinApiPath = (origin, path) => {
    const safeOrigin = trimTrailingSlashes(origin || '');
    if (!safeOrigin) {
        return undefined;
    }
    const safePath = ensureLeadingSlash(path || '');
    return `${safeOrigin}${safePath}`;
};

if (typeof window !== 'undefined') {
    window.resolveTextToolsApiOrigin = resolveTextToolsApiOrigin;
}

class APIClient {
    constructor(baseURL) {
        const origin = resolveTextToolsApiOrigin();
        if (!origin) {
            throw new Error('Unable to resolve Text Tools API origin. Ensure the API is available.');
        }

        this.origin = origin;
        this.baseURL = baseURL || joinApiPath(origin, '/api/v1/text');
    }

    async request(endpoint, options = {}) {
        const url = `${this.baseURL}${endpoint}`;
        const config = {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
        };

        try {
            const response = await fetch(url, config);
            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.message || `HTTP error! status: ${response.status}`);
            }

            return data;
        } catch (error) {
            console.error('API request failed:', error);
            throw error;
        }
    }

    // Diff API
    async diff(text1, text2, options = {}) {
        return this.request('/diff', {
            method: 'POST',
            body: JSON.stringify({ text1, text2, options }),
        });
    }

    // Search API
    async search(text, pattern, options = {}) {
        return this.request('/search', {
            method: 'POST',
            body: JSON.stringify({ text, pattern, options }),
        });
    }

    async searchText(text, pattern, options = {}) {
        return this.search(text, pattern, options);
    }

    // Transform API
    async transform(text, transformations) {
        return this.request('/transform', {
            method: 'POST',
            body: JSON.stringify({ text, transformations }),
        });
    }

    async transformText(text, transformations) {
        return this.transform(text, transformations);
    }

    // Extract API
    async extract(source, format, options = {}) {
        return this.request('/extract', {
            method: 'POST',
            body: JSON.stringify({ source, format, options }),
        });
    }

    async extractText(source, options = {}) {
        return this.extract(source, options.format, options);
    }

    // Analyze API
    async analyze(text, analyses, options = {}) {
        return this.request('/analyze', {
            method: 'POST',
            body: JSON.stringify({ text, analyses, options }),
        });
    }

    async analyzeText(text, analyses, options = {}) {
        return this.analyze(text, analyses, options);
    }

    // Pipeline API (v2 endpoint)
    async processPipeline(input, steps) {
        // Use v2 endpoint for pipeline processing
        const url = joinApiPath(this.origin, '/api/v2/text/pipeline');

        try {
            const response = await fetch(url, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ input, steps }),
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || `HTTP error! status: ${response.status}`);
            }

            return data;
        } catch (error) {
            console.error('Pipeline API request failed:', error);
            throw error;
        }
    }

    // Health check
    async checkHealth() {
        const url = joinApiPath(this.origin, '/health');
        if (!url) {
            throw new Error('Unable to resolve health endpoint');
        }

        const response = await fetch(url);
        return response.json();
    }
}

// Export for use in other modules
window.APIClient = APIClient;
