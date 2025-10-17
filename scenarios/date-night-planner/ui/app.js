import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_FLAG = '__dateNightPlannerBridgeInitialized';

function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window.parent === window || window[BRIDGE_FLAG]) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[DateNightPlanner] Unable to parse parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'date-night-planner' });
    window[BRIDGE_FLAG] = true;
}

bootstrapIframeBridge();

// Date Night Planner - Main Application JavaScript

const DEFAULT_API_PORT = 8080;
const PROXY_PATH_FALLBACK = '/proxy';
const LOOPBACK_HOST_FALLBACK = ['127', '0', '0', '1'].join('.');

function getCurrentLocation() {
    if (typeof window === 'undefined' || !window.location) {
        return undefined;
    }
    const { protocol, hostname, port } = window.location;
    return {
        protocol: typeof protocol === 'string' && protocol ? protocol.replace(/:$/, '') : undefined,
        hostname: typeof hostname === 'string' && hostname ? hostname : undefined,
        port: typeof port === 'string' && port ? port : undefined
    };
}

function resolveLoopbackBase(port = DEFAULT_API_PORT) {
    const info = getCurrentLocation();
    const protocol = info?.protocol || 'http';
    const hostname = info?.hostname || (typeof process !== 'undefined' && process?.env?.API_HOST) || LOOPBACK_HOST_FALLBACK;
    const preferredPort = Number.isFinite(port) && port > 0 ? String(port) : undefined;
    const finalPort = preferredPort || info?.port || '';
    const authority = finalPort ? `${hostname}:${finalPort}` : hostname;
    return `${protocol}://${authority}`;
}

function stripTrailingSlash(value) {
    if (typeof value !== 'string' || value.length === 0) {
        return '';
    }
    if (/^https?:\/\/$/i.test(value)) {
        return value;
    }
    return value.replace(/\/+$/, '');
}

function ensureLeadingSlash(value) {
    if (typeof value !== 'string' || value.length === 0) {
        return '';
    }
    return value.startsWith('/') ? value : `/${value}`;
}

function joinUrl(base, segment) {
    const normalizedBase = stripTrailingSlash(base || '');
    const normalizedSegment = ensureLeadingSlash(segment || '');
    if (!normalizedBase) {
        return normalizedSegment;
    }
    if (!normalizedSegment) {
        return normalizedBase;
    }
    return `${normalizedBase}${normalizedSegment}`;
}

function normalizeCandidate(candidate) {
    if (typeof candidate !== 'string') {
        return '';
    }
    const trimmed = candidate.trim();
    if (!trimmed) {
        return '';
    }

    if (/^https?:\/\//i.test(trimmed)) {
        return stripTrailingSlash(trimmed);
    }

    if (trimmed.startsWith('//')) {
        if (typeof window !== 'undefined' && window.location) {
            return stripTrailingSlash(`${window.location.protocol}${trimmed}`);
        }
        return stripTrailingSlash(`https:${trimmed}`);
    }

    if (trimmed.startsWith('/')) {
        return stripTrailingSlash(trimmed);
    }

    if (typeof window !== 'undefined' && window.location) {
        return stripTrailingSlash(joinUrl(window.location.origin, trimmed));
    }

    return '';
}

function pickProxyCandidate(info) {
    if (!info || typeof info !== 'object') {
        return undefined;
    }

    const collectEndpoint = (endpoint) => {
        if (!endpoint) {
            return undefined;
        }
        return endpoint.url || endpoint.path || endpoint.target;
    };

    if (Array.isArray(info.ports)) {
        const ranked = [];
        info.ports.forEach((entry) => {
            if (!entry) {
                return;
            }
            const aliases = Array.isArray(entry.aliases)
                ? entry.aliases.map((alias) => String(alias).toLowerCase())
                : [];
            const label = typeof entry.label === 'string' ? entry.label.toLowerCase() : '';
            const isApiPort = aliases.some((alias) => alias.includes('api')) || label.includes('api');
            const priority = isApiPort ? 0 : 1;
            [entry.url, entry.path, entry.target]
                .filter((value) => typeof value === 'string' && value.trim())
                .forEach((value) => ranked.push({ value, priority }));
            if (Array.isArray(entry.routes)) {
                entry.routes
                    .filter((route) => typeof route === 'string' && route.trim())
                    .forEach((route) => ranked.push({ value: route, priority }));
            }
        });
        ranked.sort((a, b) => a.priority - b.priority);
        const match = ranked.find((item) => item.value && item.value.trim().length > 0);
        if (match) {
            return match.value;
        }
    }

    const primaryCandidate = collectEndpoint(info.primary);
    if (primaryCandidate) {
        return primaryCandidate;
    }

    if (Array.isArray(info.endpoints)) {
        for (const endpoint of info.endpoints) {
            const candidate = collectEndpoint(endpoint);
            if (candidate) {
                return candidate;
            }
        }
    }

    if (typeof info.apiBase === 'string' && info.apiBase.trim()) {
        return info.apiBase;
    }

    if (typeof info.baseUrl === 'string' && info.baseUrl.trim()) {
        return info.baseUrl;
    }

    return undefined;
}

function resolveProxyMetadataBase() {
    if (typeof window === 'undefined') {
        return undefined;
    }
    const info = window.__APP_MONITOR_PROXY_INFO__;
    const candidate = pickProxyCandidate(info);
    if (!candidate) {
        return undefined;
    }
    return normalizeCandidate(candidate);
}

function resolveApiBase(...preferredCandidates) {
    const metadataBase = resolveProxyMetadataBase();
    if (metadataBase) {
        return metadataBase;
    }

    for (const candidate of preferredCandidates) {
        const normalized = normalizeCandidate(candidate);
        if (normalized) {
            return normalized;
        }
    }

    if (typeof window !== 'undefined' && window.location) {
        const origin = window.location.origin || '';
        const normalized = normalizeCandidate(origin ? `${origin}${PROXY_PATH_FALLBACK}` : PROXY_PATH_FALLBACK);
        if (normalized) {
            return normalized;
        }
    }

    return resolveLoopbackBase(DEFAULT_API_PORT);
}

let config = {
    apiUrl: resolveApiBase(resolveLoopbackBase(DEFAULT_API_PORT)),
    proxyApiUrl: PROXY_PATH_FALLBACK,
    version: '1.0.0'
};

// Load configuration
async function loadConfig() {
    try {
        const response = await fetch('/config');
        if (response.ok) {
            const remoteConfig = await response.json();
            const directCandidate = typeof remoteConfig.apiUrl === 'string' ? remoteConfig.apiUrl : '';
            const proxyUrlCandidate = typeof remoteConfig.proxyApiUrl === 'string' ? remoteConfig.proxyApiUrl : '';
            const proxyPathCandidate = typeof remoteConfig.proxyApiPath === 'string' ? remoteConfig.proxyApiPath : '';

            const resolvedApiUrl = resolveApiBase(proxyUrlCandidate, proxyPathCandidate, directCandidate, config.apiUrl);

            config = {
                ...config,
                ...remoteConfig,
                apiUrl: resolvedApiUrl,
                proxyApiUrl: proxyPathCandidate || proxyUrlCandidate || config.proxyApiUrl,
                directApiUrl: directCandidate || config.directApiUrl,
                resolvedApiUrl
            };
        }
    } catch (error) {
        console.error('Failed to load config:', error);
        config.apiUrl = resolveApiBase(config.apiUrl, resolveLoopbackBase(DEFAULT_API_PORT));
    }
}

// Tab navigation
function initTabs() {
    const tabs = document.querySelectorAll('.nav-tab');
    const contents = document.querySelectorAll('.tab-content');
    
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const targetTab = tab.dataset.tab;
            
            // Update active tab
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            
            // Show corresponding content
            contents.forEach(content => {
                if (content.id === `${targetTab}-tab`) {
                    content.classList.add('active');
                } else {
                    content.classList.remove('active');
                }
            });
        });
    });
}

// Status monitoring
async function updateStatus() {
    const indicators = {
        api: document.getElementById('api-status'),
        db: document.getElementById('db-status'),
        wf: document.getElementById('wf-status')
    };
    
    try {
        // Check API health
        const apiResponse = await fetch(`${config.apiUrl}/health`);
        indicators.api.className = apiResponse.ok ? 'status-indicator online' : 'status-indicator offline';
        
        // Check database health
        const dbResponse = await fetch(`${config.apiUrl}/health/database`);
        indicators.db.className = dbResponse.ok ? 'status-indicator online' : 'status-indicator degraded';
        
        // Check workflow health
        const wfResponse = await fetch(`${config.apiUrl}/health/workflows`);
        indicators.wf.className = wfResponse.ok ? 'status-indicator online' : 'status-indicator degraded';
    } catch (error) {
        // If API is completely unreachable
        indicators.api.className = 'status-indicator offline';
        indicators.db.className = 'status-indicator offline';
        indicators.wf.className = 'status-indicator offline';
    }
}

// Budget slider update
function initBudgetSlider() {
    const budgetInput = document.getElementById('budget');
    const budgetDisplay = document.getElementById('budget-value');
    
    if (budgetInput && budgetDisplay) {
        budgetInput.addEventListener('input', () => {
            budgetDisplay.textContent = budgetInput.value;
        });
    }
}

// Get date suggestions
async function getSuggestions(event) {
    event.preventDefault();
    
    const coupleId = document.getElementById('couple-id').value;
    const dateType = document.getElementById('date-type').value;
    const budget = document.getElementById('budget').value;
    const preferredDate = document.getElementById('preferred-date').value;
    const weatherPref = document.querySelector('input[name="weather"]:checked')?.value || 'flexible';
    
    if (!coupleId) {
        alert('Please enter a couple ID');
        return;
    }
    
    const requestBody = {
        couple_id: coupleId,
        ...(dateType && { date_type: dateType }),
        ...(budget && { budget_max: parseFloat(budget) }),
        ...(preferredDate && { preferred_date: preferredDate }),
        weather_preference: weatherPref
    };
    
    try {
        const response = await fetch(`${config.apiUrl}/api/v1/dates/suggest`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestBody)
        });
        
        if (!response.ok) {
            throw new Error('Failed to get suggestions');
        }
        
        const data = await response.json();
        displaySuggestions(data.suggestions || []);
    } catch (error) {
        console.error('Error getting suggestions:', error);
        alert('Failed to get date suggestions. Please check if the API is running.');
    }
}

// Display suggestions
function displaySuggestions(suggestions) {
    const resultsContainer = document.getElementById('suggestions-results');
    const suggestionsList = document.getElementById('suggestions-list');
    
    if (suggestions.length === 0) {
        suggestionsList.innerHTML = '<p class="empty-state">No suggestions found. Try adjusting your criteria.</p>';
    } else {
        suggestionsList.innerHTML = suggestions.map((suggestion, index) => `
            <div class="suggestion-card" data-suggestion-id="${index}">
                <h3 class="suggestion-title">${suggestion.title || 'Date Suggestion'}</h3>
                <p class="suggestion-description">${suggestion.description || 'A wonderful date experience'}</p>
                <div class="suggestion-meta">
                    <span class="suggestion-cost">💰 $${suggestion.estimated_cost || 0}</span>
                    <span class="suggestion-duration">⏱️ ${suggestion.estimated_duration || '2 hours'}</span>
                </div>
                ${suggestion.confidence_score ? `
                    <div style="margin-top: 12px;">
                        <div style="background: #E6E6FA; border-radius: 8px; height: 8px; overflow: hidden;">
                            <div style="background: linear-gradient(90deg, #FF69B4, #9370DB); height: 100%; width: ${suggestion.confidence_score * 100}%;"></div>
                        </div>
                        <small style="color: #7A7A7A;">Confidence: ${Math.round(suggestion.confidence_score * 100)}%</small>
                    </div>
                ` : ''}
                <button class="btn btn-primary" style="margin-top: 16px; width: 100%;" onclick="planDate('${index}')">
                    Plan This Date 💝
                </button>
            </div>
        `).join('');
    }
    
    resultsContainer.style.display = 'block';
}

// Plan a date
async function planDate(suggestionIndex) {
    const coupleId = document.getElementById('couple-id').value;
    const preferredDate = document.getElementById('preferred-date').value || new Date().toISOString().split('T')[0];
    
    // Get the suggestion data (in a real app, we'd store this properly)
    const suggestionCard = document.querySelector(`[data-suggestion-id="${suggestionIndex}"]`);
    const title = suggestionCard.querySelector('.suggestion-title').textContent;
    const description = suggestionCard.querySelector('.suggestion-description').textContent;
    
    const requestBody = {
        couple_id: coupleId,
        selected_suggestion: {
            title: title,
            description: description,
            activities: [],
            estimated_cost: 100,
            estimated_duration: "2 hours"
        },
        planned_date: `${preferredDate}T19:00:00Z`
    };
    
    try {
        const response = await fetch(`${config.apiUrl}/api/v1/dates/plan`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestBody)
        });
        
        if (!response.ok) {
            throw new Error('Failed to create date plan');
        }
        
        const data = await response.json();
        alert(`✅ Date plan created successfully!\n\nDate: ${preferredDate}\nTitle: ${title}`);
        
        // Switch to plans tab
        document.querySelector('[data-tab="plan"]').click();
        loadPlans();
    } catch (error) {
        console.error('Error creating date plan:', error);
        alert('Failed to create date plan. Please try again.');
    }
}

// Load existing plans
async function loadPlans() {
    const plansList = document.getElementById('plans-list');
    
    // In a real implementation, we'd fetch plans from the API
    // For now, show a placeholder
    plansList.innerHTML = `
        <div class="suggestion-card">
            <h3 class="suggestion-title">Your Next Date</h3>
            <p class="suggestion-description">Check back here to see your planned dates!</p>
            <div class="suggestion-meta">
                <span>📅 Coming Soon</span>
            </div>
        </div>
    `;
}

// Initialize the application
document.addEventListener('DOMContentLoaded', async () => {
    await loadConfig();
    initTabs();
    initBudgetSlider();
    
    // Set up form submission
    const suggestionForm = document.getElementById('suggestion-form');
    if (suggestionForm) {
        suggestionForm.addEventListener('submit', getSuggestions);
    }
    
    // Set default date to tomorrow
    const dateInput = document.getElementById('preferred-date');
    if (dateInput) {
        const tomorrow = new Date();
        tomorrow.setDate(tomorrow.getDate() + 1);
        dateInput.value = tomorrow.toISOString().split('T')[0];
    }
    
    // Start status monitoring
    updateStatus();
    setInterval(updateStatus, 30000); // Update every 30 seconds
});
