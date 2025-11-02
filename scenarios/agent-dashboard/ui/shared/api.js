// Shared API Module
// Handles all API interactions across pages

const PROXY_INFO_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__'];
const API_SUFFIX = '/api/v1';
const DEFAULT_API_PORT = '15000';
const LOOPBACK_HOST = '127.0.0.1';

const stripTrailingSlashes = (value = '') => value.replace(/\/+$/, '');
const ensureLeadingSlash = (value = '') => (value.startsWith('/') ? value : `/${value}`);

const ensureApiSuffix = (base) => {
    const sanitizedBase = stripTrailingSlashes(base || '');
    if (!sanitizedBase) {
        return `http://${LOOPBACK_HOST}:${DEFAULT_API_PORT}${API_SUFFIX}`;
    }

    const lower = sanitizedBase.toLowerCase();
    if (lower.endsWith(API_SUFFIX)) {
        return sanitizedBase;
    }

    if (lower.endsWith('/api')) {
        return `${sanitizedBase}${API_SUFFIX.slice(4)}`;
    }

    return `${sanitizedBase}${API_SUFFIX}`;
};

const collectProxyCandidates = (info, accumulator) => {
    if (!info) {
        return;
    }

    const pushCandidate = (candidate) => {
        if (!candidate) {
            return;
        }

        if (typeof candidate === 'string') {
            const trimmed = candidate.trim();
            if (trimmed.length > 0) {
                accumulator.push(trimmed);
            }
            return;
        }

        if (Array.isArray(candidate)) {
            candidate.forEach(pushCandidate);
            return;
        }

        if (typeof candidate === 'object') {
            const fields = ['apiBase', 'api', 'apiUrl', 'apiURL', 'baseUrl', 'baseURL', 'url', 'target', 'href', 'path'];
            fields.forEach((field) => pushCandidate(candidate[field]));
            if (Array.isArray(candidate.endpoints)) {
                candidate.endpoints.forEach(pushCandidate);
            }
            if (Array.isArray(candidate.ports)) {
                candidate.ports.forEach((entry) => {
                    if (!entry) {
                        return;
                    }
                    if (typeof entry === 'string') {
                        pushCandidate(entry);
                        return;
                    }
                    const { url, target, path, proxy, host, port } = entry;
                    pushCandidate(url || target || proxy || path);
                    if (host && port) {
                        pushCandidate(`${host}:${port}`);
                    }
                });
            }
            return;
        }
    };

    pushCandidate(info);
};

const normalizeCandidate = (candidate) => {
    if (typeof candidate !== 'string') {
        return undefined;
    }

    const trimmed = candidate.trim();
    if (!trimmed) {
        return undefined;
    }

    if (/^https?:\/\//i.test(trimmed)) {
        return stripTrailingSlashes(trimmed);
    }

    if (trimmed.startsWith('//')) {
        const protocol = typeof window !== 'undefined' && window.location?.protocol ? window.location.protocol : 'https:';
        return stripTrailingSlashes(`${protocol}${trimmed}`);
    }

    if (trimmed.startsWith('/')) {
        if (typeof window !== 'undefined' && window.location?.origin) {
            return stripTrailingSlashes(`${window.location.origin}${trimmed}`);
        }
        return stripTrailingSlashes(trimmed);
    }

    if (/^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)(:\d+)?/i.test(trimmed)) {
        return stripTrailingSlashes(`http://${trimmed}`);
    }

    if (typeof window !== 'undefined' && window.location?.origin) {
        return stripTrailingSlashes(`${window.location.origin}/${trimmed.replace(/^\/+/, '')}`);
    }

    return stripTrailingSlashes(`http://${LOOPBACK_HOST}:${DEFAULT_API_PORT}`);
};

const deriveProxyBaseFromLocation = () => {
    if (typeof window === 'undefined') {
        return undefined;
    }

    const { origin, pathname } = window.location || {};
    if (!origin || !pathname || !pathname.includes('/proxy')) {
        return undefined;
    }

    const segment = '/proxy';
    const index = pathname.indexOf(segment);
    if (index === -1) {
        return undefined;
    }

    return `${origin}${pathname.slice(0, index + segment.length)}`;
};

let cachedApiBase;

const applyCachedApiBase = (value) => {
    cachedApiBase = ensureApiSuffix(value);
    if (typeof window !== 'undefined') {
        window.AGENT_DASHBOARD_API_BASE = cachedApiBase;
    }
    return cachedApiBase;
};

const resolveAgentDashboardApiBase = (forceRefresh = false) => {
    if (!forceRefresh && cachedApiBase) {
        return cachedApiBase;
    }

    const candidates = [];
    if (typeof window !== 'undefined') {
        for (const key of PROXY_INFO_KEYS) {
            if (typeof window[key] !== 'undefined') {
                collectProxyCandidates(window[key], candidates);
            }
        }
    }

    for (const candidate of candidates) {
        const normalized = normalizeCandidate(candidate);
        if (normalized) {
            return applyCachedApiBase(normalized);
        }
    }

    const proxyBase = deriveProxyBaseFromLocation();
    if (proxyBase) {
        return applyCachedApiBase(proxyBase);
    }

    if (typeof window !== 'undefined' && window.location?.origin) {
        return applyCachedApiBase(window.location.origin);
    }

    const apiPort = typeof window !== 'undefined' && window.API_PORT ? String(window.API_PORT) : DEFAULT_API_PORT;
    const protocol = typeof window !== 'undefined' && window.location?.protocol === 'https:' ? 'https' : 'http';
    return applyCachedApiBase(`${protocol}://${LOOPBACK_HOST}:${apiPort}`);
};

const buildAgentDashboardApiUrl = (path = '') => {
    const base = resolveAgentDashboardApiBase();
    if (!path) {
        return base;
    }

    if (path.startsWith('?')) {
        return `${base}${path}`;
    }

    const safePath = ensureLeadingSlash(path);
    return `${stripTrailingSlashes(base)}${safePath}`;
};

class AgentAPI {
    constructor() {
        this.baseUrl = resolveAgentDashboardApiBase();
    }

    buildUrl(path = '') {
        return buildAgentDashboardApiUrl(path);
    }

    async fetchAgents() {
        try {
            const response = await fetch(this.buildUrl('/agents'));
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            const data = await response.json();
            return data.agents || [];
        } catch (error) {
            console.error('Failed to fetch agents:', error);
            throw error;
        }
    }

    async fetchAgentDetails(agentId) {
        try {
            const response = await fetch(this.buildUrl(`/agents/${encodeURIComponent(agentId)}`));
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`Failed to fetch agent ${agentId}:`, error);
            throw error;
        }
    }

    async fetchAgentLogs(agentId, lines = 50) {
        try {
            const url = this.buildUrl(`/agents/${encodeURIComponent(agentId)}/logs?lines=${lines}`);
            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            const data = await response.json();
            return data.data || data;
        } catch (error) {
            console.error(`Failed to fetch logs for agent ${agentId}:`, error);
            throw error;
        }
    }

    async fetchAgentMetrics(agentId) {
        try {
            const response = await fetch(this.buildUrl(`/agents/${encodeURIComponent(agentId)}/metrics`));
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`Failed to fetch metrics for agent ${agentId}:`, error);
            throw error;
        }
    }

    async stopAgent(agentId) {
        try {
            const response = await fetch(this.buildUrl(`/agents/${encodeURIComponent(agentId)}/stop`), {
                method: 'POST'
            });
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`Failed to stop agent ${agentId}:`, error);
            throw error;
        }
    }

    async startAgent(agentId) {
        try {
            const response = await fetch(this.buildUrl(`/agents/${encodeURIComponent(agentId)}/start`), {
                method: 'POST'
            });
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`Failed to start agent ${agentId}:`, error);
            throw error;
        }
    }

    async triggerScan() {
        try {
            const response = await fetch(this.buildUrl('/scan'), {
                method: 'POST'
            });
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error('Failed to trigger scan:', error);
            throw error;
        }
    }

    async orchestrate(mode = 'auto') {
        try {
            const response = await fetch(this.buildUrl('/orchestrate'), {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ mode })
            });
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error('Failed to orchestrate:', error);
            throw error;
        }
    }
}

// Create global helpers and instances
window.resolveAgentDashboardApiBase = resolveAgentDashboardApiBase;
window.buildAgentDashboardApiUrl = buildAgentDashboardApiUrl;
window.agentAPI = new AgentAPI();

// Agent actions for button handlers
window.agentActions = {
    async stop(agentId) {
        try {
            const result = await window.agentAPI.stopAgent(agentId);
            if (result.success) {
                window.location.reload();
            }
        } catch (error) {
            alert(`Failed to stop agent: ${error.message}`);
        }
    },
    
    async start(agentId) {
        try {
            const result = await window.agentAPI.startAgent(agentId);
            if (result.success) {
                window.location.reload();
            }
        } catch (error) {
            alert(`Failed to start agent: ${error.message}`);
        }
    }
};

// Update agent count in header (used by all pages)
function updateAgentCount(count) {
    const agentCountElement = document.getElementById('agentCount');
    if (agentCountElement) {
        agentCountElement.textContent = `${count} AGENT${count !== 1 ? 'S' : ''} ACTIVE`;
        
        const statusIcon = agentCountElement.parentElement.querySelector('.status-icon');
        if (statusIcon) {
            statusIcon.classList.remove('active', 'warning', 'error');
            statusIcon.classList.add(count > 0 ? 'active' : 'warning');
        }
    }
}

// Make it available globally
window.updateAgentCount = updateAgentCount;

// Auto-update agent count on all pages
document.addEventListener('DOMContentLoaded', async () => {
    try {
        const agents = await window.agentAPI.fetchAgents();
        updateAgentCount(agents.length);
    } catch (error) {
        console.error('Failed to update agent count:', error);
    }
});
