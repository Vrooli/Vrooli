import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const isBrowser = typeof window !== 'undefined';

const iframeParentOrigin = (() => {
    if (!isBrowser) {
        return undefined;
    }
    try {
        return document.referrer ? new URL(document.referrer).origin : undefined;
    } catch (error) {
        return undefined;
    }
})();

if (isBrowser && window.parent !== window) {
    initIframeBridgeChild({ appId: 'knowledge-observatory', parentOrigin: iframeParentOrigin });
}

const API_BASE = resolveApiBase();

function buildApiUrl(path) {
    return joinUrl(API_BASE, path);
}

function buildWebSocketUrl(path) {
    const httpUrl = buildApiUrl(path);
    if (httpUrl.startsWith('https://')) {
        return `wss://${httpUrl.slice('https://'.length)}`;
    }
    if (httpUrl.startsWith('http://')) {
        return `ws://${httpUrl.slice('http://'.length)}`;
    }
    if (httpUrl.startsWith('//')) {
        return `ws${httpUrl.slice(2)}`;
    }
    return httpUrl.replace(/^http/i, 'ws');
}

function resolveApiBase() {
    if (!isBrowser) {
        return '/api';
    }

    const proxyCandidate = resolveProxyCandidate();
    if (proxyCandidate) {
        return joinUrl(proxyCandidate, '/api');
    }

    const envUrl = typeof window.ENV?.API_URL === 'string' ? window.ENV.API_URL.trim() : '';
    if (envUrl) {
        return joinUrl(envUrl, '/api');
    }

    const protocol = window.location.protocol === 'https:' ? 'https' : 'http';
    const host = window.location.hostname || 'localhost';
    const fallbackPort = resolveFallbackPort(host);
    const origin = `${protocol}://${host}${fallbackPort ? `:${fallbackPort}` : ''}`;

    return joinUrl(origin, '/api');
}

function resolveProxyCandidate() {
    const info = isBrowser ? window.__APP_MONITOR_PROXY_INFO__ : undefined;
    if (!info) {
        return undefined;
    }

    const candidate = pickProxyEndpoint(info);
    if (!candidate) {
        return undefined;
    }

    if (/^https?:\/\//i.test(candidate)) {
        return stripTrailingSlash(candidate);
    }

    return stripTrailingSlash(joinUrl(window.location.origin, candidate));
}

function pickProxyEndpoint(info) {
    const candidates = [];

    const append = (endpoint) => {
        if (!endpoint) {
            return;
        }
        if (typeof endpoint === 'string') {
            const trimmed = endpoint.trim();
            if (trimmed) {
                candidates.push(trimmed);
            }
            return;
        }
        if (typeof endpoint.url === 'string') {
            const url = endpoint.url.trim();
            if (url) {
                candidates.push(url);
            }
        }
        if (typeof endpoint.path === 'string') {
            const path = endpoint.path.trim();
            if (path) {
                candidates.push(path);
            }
        }
        if (typeof endpoint.target === 'string') {
            const target = endpoint.target.trim();
            if (target) {
                candidates.push(target);
            }
        }
    };

    append(info.primary);
    if (Array.isArray(info.endpoints)) {
        info.endpoints.forEach(append);
    }

    return candidates.find(Boolean);
}

function resolveFallbackPort(hostname) {
    const envValue = window.ENV?.API_PORT;
    if (typeof envValue === 'string' && envValue.trim()) {
        return envValue.trim();
    }
    if (typeof envValue === 'number' && Number.isFinite(envValue)) {
        return String(envValue);
    }
    if (typeof window.location.port === 'string' && window.location.port.trim()) {
        return window.location.port.trim();
    }
    if (hostname === 'localhost' || hostname === '127.0.0.1') {
        return '20270';
    }
    return '';
}

function stripTrailingSlash(value) {
    if (!value) {
        return '';
    }
    if (value.endsWith('/') && !/^https?:\/\/$/i.test(value)) {
        return value.replace(/\/+$/, '');
    }
    return value;
}

function ensureLeadingSlash(value) {
    if (!value) {
        return '/';
    }
    return value.startsWith('/') ? value : `/${value}`;
}

function joinUrl(base, segment) {
    const normalizedBase = stripTrailingSlash(base);
    const normalizedSegment = ensureLeadingSlash(segment);
    if (!normalizedBase) {
        return normalizedSegment;
    }
    return `${normalizedBase}${normalizedSegment}`;
}

let currentTab = 'dashboard';
let selectedCollection = 'all';
let searchResults = [];
let healthData = null;
let graphData = null;
let metricsData = null;

async function fetchHealth() {
    try {
        const response = await fetch(buildApiUrl('/v1/knowledge/health'));
        const data = await response.json();
        healthData = data;
        updateHealthDisplay(data);
    } catch (error) {
        console.error('Failed to fetch health data:', error);
    }
}

function updateHealthDisplay(data) {
    document.getElementById('system-status').textContent = data.status;
    document.getElementById('total-entries').textContent = data.total_entries.toLocaleString();
    document.getElementById('collection-count').textContent = data.collections.length;
    document.getElementById('last-update').textContent = new Date().toLocaleTimeString();
    
    const statusIndicator = document.querySelector('.status-indicator');
    statusIndicator.className = `status-indicator ${data.overall_health}`;
    
    if (data.collections && data.collections.length > 0) {
        const avgMetrics = calculateAverageMetrics(data.collections);
        updateMetricDisplays(avgMetrics);
    }
    
    updateActivityFeed(data);
}

function calculateAverageMetrics(collections) {
    const total = collections.reduce((acc, col) => {
        acc.coherence += col.quality.coherence_score;
        acc.freshness += col.quality.freshness_score;
        acc.redundancy += col.quality.redundancy_score;
        acc.coverage += col.quality.coverage_score;
        return acc;
    }, { coherence: 0, freshness: 0, redundancy: 0, coverage: 0 });
    
    const count = collections.length;
    return {
        coherence: total.coherence / count,
        freshness: total.freshness / count,
        redundancy: total.redundancy / count,
        coverage: total.coverage / count
    };
}

function updateMetricDisplays(metrics) {
    updateMetric('coherence', metrics.coherence);
    updateMetric('freshness', metrics.freshness);
    updateMetric('redundancy', metrics.redundancy);
    updateMetric('coverage', metrics.coverage);
    
    document.getElementById('health-timestamp').textContent = new Date().toLocaleTimeString();
}

function updateMetric(name, value) {
    const scoreEl = document.getElementById(`${name}-score`);
    const barEl = document.getElementById(`${name}-bar`);
    
    if (scoreEl) scoreEl.textContent = value.toFixed(2);
    if (barEl) barEl.style.width = `${value * 100}%`;
    
    if (value < 0.6) {
        barEl.style.background = '#ff0041';
    } else if (value < 0.8) {
        barEl.style.background = '#ffa500';
    } else {
        barEl.style.background = '#00ff41';
    }
}

function updateActivityFeed(data) {
    const feed = document.getElementById('activity-feed');
    const activities = [
        `System health check completed - ${data.overall_health}`,
        `${data.total_entries} knowledge entries indexed`,
        `${data.collections.length} collections monitored`,
        `Last semantic search: ${new Date().toLocaleTimeString()}`
    ];
    
    feed.innerHTML = activities.map(activity => 
        `<div style="padding: 0.5rem; border-bottom: 1px solid #00ff4133;">${activity}</div>`
    ).join('');
}

async function performSearch() {
    const query = document.getElementById('searchInput').value;
    if (!query) return;
    
    switchTab('search');
    const resultsContainer = document.getElementById('search-results');
    resultsContainer.innerHTML = '<div class="loading"><div class="loading-spinner"></div></div>';
    
    try {
        const body = {
            query: query,
            limit: 20,
            threshold: 0.5
        };
        
        if (selectedCollection !== 'all') {
            body.collection = selectedCollection;
        }
        
        const response = await fetch(buildApiUrl('/v1/knowledge/search'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
        
        const data = await response.json();
        searchResults = data.results;
        displaySearchResults(data);
    } catch (error) {
        console.error('Search failed:', error);
        resultsContainer.innerHTML = '<p style="color: #ff0041;">Search failed. Please try again.</p>';
    }
}

function displaySearchResults(data) {
    const container = document.getElementById('search-results');
    document.getElementById('search-count').textContent = `${data.count} results (${data.query_time_ms}ms)`;
    
    if (!data.results || data.results.length === 0) {
        container.innerHTML = '<p style="opacity: 0.6;">No results found.</p>';
        return;
    }
    
    container.innerHTML = data.results.map(result => `
        <div class="result-item" onclick="showResultDetail('${result.id}')">
            <div>
                <span class="result-score">${result.score.toFixed(3)}</span>
                <strong>${result.metadata?.source || 'Unknown Source'}</strong>
            </div>
            <div class="result-content">${result.content}</div>
            <div class="result-metadata">
                ${result.metadata?.timestamp ? `Added: ${new Date(result.metadata.timestamp).toLocaleDateString()}` : ''}
                ${result.metadata?.scenario ? `| Scenario: ${result.metadata.scenario}` : ''}
            </div>
        </div>
    `).join('');
}

function showResultDetail(id) {
    const result = searchResults.find(r => r.id === id);
    if (result) {
        console.log('Result detail:', result);
    }
}

async function fetchGraph() {
    try {
        const response = await fetch(buildApiUrl('/v1/knowledge/graph'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                depth: 3,
                max_nodes: 100
            })
        });
        
        const data = await response.json();
        graphData = data;
        renderGraph(data);
    } catch (error) {
        console.error('Failed to fetch graph data:', error);
    }
}

function renderGraph(data) {
    const canvas = document.getElementById('knowledge-graph');
    const ctx = canvas.getContext('2d');
    
    canvas.width = canvas.offsetWidth;
    canvas.height = canvas.offsetHeight;
    
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.strokeStyle = '#00ff41';
    ctx.fillStyle = '#00ff41';
    
    const centerX = canvas.width / 2;
    const centerY = canvas.height / 2;
    const radius = Math.min(canvas.width, canvas.height) / 3;
    
    if (!data.nodes || data.nodes.length === 0) {
        ctx.font = '16px monospace';
        ctx.textAlign = 'center';
        ctx.fillText('No graph data available', centerX, centerY);
        return;
    }
    
    const nodePositions = {};
    data.nodes.forEach((node, i) => {
        const angle = (i / data.nodes.length) * Math.PI * 2;
        const x = centerX + Math.cos(angle) * radius;
        const y = centerY + Math.sin(angle) * radius;
        nodePositions[node.id] = { x, y };
    });
    
    ctx.strokeStyle = '#00ff4133';
    ctx.lineWidth = 1;
    data.edges.forEach(edge => {
        const start = nodePositions[edge.source];
        const end = nodePositions[edge.target];
        if (start && end) {
            ctx.beginPath();
            ctx.moveTo(start.x, start.y);
            ctx.lineTo(end.x, end.y);
            ctx.stroke();
            
            ctx.globalAlpha = edge.weight;
            ctx.strokeStyle = '#00ff41';
            ctx.stroke();
            ctx.globalAlpha = 1;
            ctx.strokeStyle = '#00ff4133';
        }
    });
    
    ctx.fillStyle = '#00ff41';
    data.nodes.forEach(node => {
        const pos = nodePositions[node.id];
        if (pos) {
            ctx.beginPath();
            ctx.arc(pos.x, pos.y, 5, 0, Math.PI * 2);
            ctx.fill();
            
            ctx.font = '10px monospace';
            ctx.textAlign = 'center';
            ctx.fillText(node.label || node.id, pos.x, pos.y - 10);
        }
    });
}

async function fetchMetrics() {
    try {
        const response = await fetch(buildApiUrl('/v1/knowledge/metrics'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({})
        });
        
        const data = await response.json();
        metricsData = data;
        displayMetrics(data);
    } catch (error) {
        console.error('Failed to fetch metrics:', error);
    }
}

function displayMetrics(data) {
    const container = document.getElementById('collection-metrics');
    
    if (!data.metrics) {
        container.innerHTML = '<p style="opacity: 0.6;">No metrics data available.</p>';
        return;
    }
    
    const metricsHTML = Object.entries(data.metrics).map(([collection, metrics]) => `
        <div style="margin-bottom: 1.5rem;">
            <h3 style="margin-bottom: 0.5rem; color: #00ff41;">${collection}</h3>
            <div class="metric-grid">
                <div style="padding: 0.5rem;">Coherence: ${metrics.coherence_score.toFixed(2)}</div>
                <div style="padding: 0.5rem;">Freshness: ${metrics.freshness_score.toFixed(2)}</div>
                <div style="padding: 0.5rem;">Redundancy: ${metrics.redundancy_score.toFixed(2)}</div>
                <div style="padding: 0.5rem;">Coverage: ${metrics.coverage_score.toFixed(2)}</div>
            </div>
        </div>
    `).join('');
    
    container.innerHTML = metricsHTML;
    
    if (data.alerts && data.alerts.length > 0) {
        const alertsContainer = document.getElementById('alerts-container');
        alertsContainer.innerHTML = data.alerts.map(alert => `
            <div class="alert-box ${alert.level}">
                <span>${alert.level === 'critical' ? '🚨' : '⚠️'}</span>
                <span>${alert.message}</span>
            </div>
        `).join('');
    }
}

function switchTab(tab) {
    currentTab = tab;
    
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('[id$="-view"]').forEach(v => v.style.display = 'none');
    
    const activeTab = document.querySelector(`.tab:nth-child(${['dashboard', 'search', 'graph', 'metrics'].indexOf(tab) + 1})`);
    if (activeTab) activeTab.classList.add('active');
    
    const view = document.getElementById(`${tab}-view`);
    if (view) view.style.display = 'grid';
    
    if (tab === 'graph' && !graphData) {
        fetchGraph();
    } else if (tab === 'metrics' && !metricsData) {
        fetchMetrics();
    }
}

function refreshGraph() {
    fetchGraph();
}

document.addEventListener('DOMContentLoaded', () => {
    fetchHealth();
    setInterval(fetchHealth, 10000);
    
    document.getElementById('searchInput').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') performSearch();
    });
    
    document.querySelectorAll('.filter-item').forEach(filter => {
        filter.addEventListener('click', () => {
            document.querySelectorAll('.filter-item').forEach(f => f.classList.remove('active'));
            filter.classList.add('active');
            selectedCollection = filter.dataset.collection;
        });
    });
    
    const ws = new WebSocket(buildWebSocketUrl('/v1/knowledge/stream'));
    
    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        updateHealthDisplay(data);
    };
    
    ws.onerror = (error) => {
        console.log('WebSocket connection failed, falling back to polling');
    };
});
