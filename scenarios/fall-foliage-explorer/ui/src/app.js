import { initIframeBridgeChild } from '/bridge/iframeBridgeChild.js';

// Fall Foliage Explorer - Interactive Application

const BRIDGE_STATE_KEY = '__fallFoliageBridgeInitialized';
const PROXY_INFO_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__'];

if (typeof window !== 'undefined' && window.parent !== window && !window[BRIDGE_STATE_KEY]) {
    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[FallFoliage] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'fall-foliage-explorer' });
    window[BRIDGE_STATE_KEY] = true;
}

// Configuration
const DEFAULT_API_PORT = 17175;
const LOOPBACK_HOST = '127.0.0.1';
const apiResolution = resolveApiBase();
const API_BASE = apiResolution.base;
const API_BASE_SOURCE = apiResolution.source;
const API_BASE_NOTES = apiResolution.notes;

function formatStatus(status) {
    if (typeof status !== 'string') {
        return 'Unknown';
    }
    return status
        .split('_')
        .filter(Boolean)
        .map(token => token.charAt(0).toUpperCase() + token.slice(1))
        .join(' ') || 'Unknown';
}

function resolveApiBase() {
    const win = typeof window !== 'undefined' ? window : undefined;
    const resolution = {
        base: buildLoopbackBase(DEFAULT_API_PORT, win),
        source: 'loopback-fallback',
        notes: 'Using default loopback developer API base'
    };

    if (!win) {
        return resolution;
    }

    const proxyBootstrap = readProxyBootstrap(win);
    const proxyBase = resolveProxyBase(proxyBootstrap, win);
    if (proxyBase) {
        resolution.base = proxyBase;
        resolution.source = 'app-monitor-proxy';
        resolution.notes = 'Resolved via App Monitor proxy metadata';
        return resolution;
    }

    const derivedFromPath = deriveProxyBaseFromLocation(win);
    if (derivedFromPath) {
        resolution.base = derivedFromPath;
        resolution.source = 'location-derived';
        resolution.notes = 'Derived from current location pathname';
        return resolution;
    }

    const { origin, hostname } = win.location || {};
    if (origin && hostname && !isLocalHostname(hostname)) {
        resolution.base = stripTrailingSlash(origin);
        resolution.source = 'same-origin';
        resolution.notes = 'Using current origin as API base';
        return resolution;
    }

    return resolution;
}

function buildLoopbackBase(port = DEFAULT_API_PORT, win) {
    const protocol = win?.location?.protocol || 'http:';
    return `${protocol}//${LOOPBACK_HOST}:${port}`;
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

    const candidates = collectProxyCandidates(proxyInfo);
    const derivedFromLocation = deriveProxyBaseFromLocation(win);
    if (derivedFromLocation) {
        candidates.unshift(derivedFromLocation);
    }
    for (const candidate of candidates) {
        const normalized = normalizeProxyCandidate(candidate, win);
        if (normalized) {
            return stripApiSuffix(normalized);
        }
    }

    return undefined;
}

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

function normalizeProxyCandidate(candidate, win) {
    if (typeof candidate !== 'string') {
        return undefined;
    }

    const trimmed = candidate.trim();
    if (!trimmed) {
        return undefined;
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

    if (/^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)(:\\d+)?(\/.*)?$/i.test(trimmed)) {
        return stripTrailingSlash(`http://${trimmed}`);
    }

    return undefined;
}

function deriveProxyBaseFromLocation(win) {
    if (!win || !win.location) {
        return undefined;
    }

    const { origin, pathname } = win.location;
    if (!origin || !pathname) {
        return undefined;
    }

    const proxyMatch = pathname.match(/^(.*?\/apps\/[^/]+)\/(?:proxy|ui)(?:\/|$)/i);
    if (proxyMatch && proxyMatch[1]) {
        return stripTrailingSlash(`${origin}${proxyMatch[1]}`);
    }

    return undefined;
}

function stripTrailingSlash(value) {
    return typeof value === 'string' ? value.replace(/\/+$/, '') : value;
}

function stripApiSuffix(value) {
    if (typeof value !== 'string') {
        return value;
    }
    const trimmed = stripTrailingSlash(value);
    return trimmed.replace(/\/api\/?$/i, '');
}

function isLocalHostname(hostname) {
    if (!hostname) {
        return false;
    }
    const normalized = hostname.toLowerCase();
    return normalized === 'localhost' ||
        normalized === '127.0.0.1' ||
        normalized === '0.0.0.0' ||
        normalized === '::1' ||
        normalized === '[::1]';
}

function buildApiUrl(path, searchParams) {
    const base = stripTrailingSlash(API_BASE);
    const normalizedPath = path.startsWith('/') ? path : `/${path}`;
    let url = base;

    if (base.toLowerCase().endsWith('/api') && normalizedPath.startsWith('/api')) {
        url += normalizedPath.slice(4);
    } else {
        url += normalizedPath;
    }

    if (searchParams && Object.keys(searchParams).length > 0) {
        const query = new URLSearchParams(searchParams).toString();
        if (query) {
            url += `?${query}`;
        }
    }

    return url;
}
let map = null;
let markers = [];
let currentView = 'map';
let regionsData = [];  // Will be populated from API
let regionsMeta = {
    source: 'pending',
    description: 'Dataset has not been loaded yet.',
    lastFetched: null,
    rowCount: 0,
    usingFallback: false,
    message: '',
    apiBase: API_BASE,
    apiBaseSource: API_BASE_SOURCE,
    apiBaseNotes: API_BASE_NOTES
};

// Initialize application
document.addEventListener('DOMContentLoaded', () => {
    initializeMap();
    setupEventListeners();
    loadRegions();
    updateTimeSlider();
    renderDataTransparencyPanel();
});

// Initialize Leaflet map
function initializeMap() {
    map = L.map('map').setView([42.3601, -71.0589], 6);
    
    // Add autumn-themed tile layer
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors',
        opacity: 0.8
    }).addTo(map);
    
    // Add foliage overlay
    addFoliageMarkers();
}

// Add markers for foliage regions
async function addFoliageMarkers() {
    // Use real data from API if available, otherwise use empty array
    const regions = regionsData.length > 0 ? regionsData : [];

    // Fetch current foliage status for each region
    for (const region of regions) {
        let status = typeof region.current_status === 'string'
            ? region.current_status
            : 'not_started';
        let intensity = Number.isFinite(region.color_intensity)
            ? region.color_intensity
            : Number.isFinite(Number(region.color_intensity))
                ? Number(region.color_intensity)
                : 0;

        try {
            const foliageResponse = await fetch(buildApiUrl('/api/foliage', { region_id: region.id }));

            if (foliageResponse.ok) {
                try {
                    const foliageData = await foliageResponse.json();
                    status = foliageData.data?.peak_status || status;
                    const apiIntensity = Number(foliageData.data?.color_intensity);
                    if (Number.isFinite(apiIntensity)) {
                        intensity = apiIntensity;
                    }
                } catch (parseErr) {
                    console.warn(`Failed to parse foliage data for region ${region.id}`, parseErr);
                }
            } else {
                console.warn(`Foliage API responded with ${foliageResponse.status} for region ${region.id}`);
            }
        } catch (err) {
            console.error(`Failed to load foliage data for region ${region.id}:`, err);
        }

        const color = getColorForStatus(status);
        const marker = L.circleMarker([region.latitude, region.longitude], {
            radius: 10 + intensity,
            fillColor: color,
            color: '#4a3c2a',
            weight: 2,
            opacity: 1,
            fillOpacity: 0.7
        }).addTo(map);

        marker.bindPopup(`
            <div class="popup-content">
                <h3>${region.name}</h3>
                <p>${region.state}</p>
                <p><strong>Status:</strong> ${formatStatus(status)}</p>
                <p><strong>Color Intensity:</strong> ${intensity}/10</p>
                <button onclick="showRegionDetails(${region.id})">View Details</button>
            </div>
        `);

        // Store status and intensity on marker for later updates
        marker.regionData = { id: region.id, status, intensity };
        markers.push(marker);
    }
}

// Get color based on foliage status
function getColorForStatus(status) {
    const colors = {
        'not_started': '#2d5a2d',
        'progressing': '#8b7355',
        'near_peak': '#ff8c00',
        'peak': '#dc143c',
        'past_peak': '#8b4513'
    };
    return colors[status] || '#8b7355';
}

// Setup event listeners
function setupEventListeners() {
    // Navigation buttons
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            switchView(e.target.dataset.view);
        });
    });
    
    // Time slider
    const slider = document.getElementById('time-slider');
    slider.addEventListener('input', updateTimeSlider);
    
    // Trip planner
    document.querySelector('.save-trip-btn')?.addEventListener('click', saveTripPlan);

    document.getElementById('refresh-data-btn')?.addEventListener('click', () => {
        loadRegions(true);
    });

    const downloadLink = document.getElementById('download-regions-json');
    if (downloadLink) {
        downloadLink.href = buildApiUrl('/api/regions');
    }
}

// Switch between views
function switchView(view) {
    // Update nav buttons
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.view === view);
    });
    
    // Update panels
    document.querySelectorAll('.view-panel').forEach(panel => {
        panel.classList.remove('active');
    });
    document.getElementById(`${view}-view`).classList.add('active');
    
    currentView = view;
    
    // Load view-specific content
    switch(view) {
        case 'map':
            setTimeout(() => map.invalidateSize(), 100);
            break;
        case 'timeline':
            loadTimeline();
            break;
        case 'regions':
            loadRegionsGrid();
            break;
        case 'planner':
            loadPlannerRegions();
            break;
        case 'gallery':
            initializeGallery();
            break;
        case 'data':
            renderDataTransparencyPanel();
            break;
    }
}

// Update time slider display
function updateTimeSlider() {
    const slider = document.getElementById('time-slider');
    const value = parseInt(slider.value);
    
    // Calculate date based on slider position
    const startDate = new Date(2025, 8, 1); // September 1
    const currentDate = new Date(startDate);
    currentDate.setDate(startDate.getDate() + value);
    
    const dateDisplay = document.getElementById('selected-date');
    dateDisplay.textContent = currentDate.toLocaleDateString('en-US', { 
        month: 'long', 
        day: 'numeric', 
        year: 'numeric' 
    });
    
    // Update map markers based on date
    updateMarkersForDate(currentDate);
}

// Update markers based on selected date
function updateMarkersForDate(date) {
    // Simulate changing foliage based on date
    const dayOfYear = Math.floor((date - new Date(date.getFullYear(), 0, 0)) / 86400000);

    markers.forEach((marker) => {
        if (!marker.regionData) return;

        // Use typical peak week from region data if available
        const peakDay = 270 + (marker.regionData.id * 5); // Stagger based on region ID
        const dayDiff = Math.abs(dayOfYear - peakDay);

        let status, intensity;
        if (dayDiff > 30) {
            status = 'not_started';
            intensity = 1;
        } else if (dayDiff > 20) {
            status = 'progressing';
            intensity = 4;
        } else if (dayDiff > 10) {
            status = 'near_peak';
            intensity = 7;
        } else if (dayDiff <= 10) {
            status = 'peak';
            intensity = 10;
        }

        const color = getColorForStatus(status);
        marker.setStyle({ fillColor: color });
        marker.setRadius(10 + intensity);
    });
}

// Load timeline view
function loadTimeline() {
    const grid = document.getElementById('timeline-grid');
    grid.innerHTML = '';
    
    const weeks = ['Sep Week 1', 'Sep Week 2', 'Sep Week 3', 'Sep Week 4',
                   'Oct Week 1', 'Oct Week 2', 'Oct Week 3', 'Oct Week 4'];
    
    weeks.forEach((week, index) => {
        const card = document.createElement('div');
        card.className = 'timeline-card';
        card.innerHTML = `
            <h4>${week}</h4>
            <div style="margin-top: 1rem;">
                ${getRegionsForWeek(index).map(r => `
                    <div style="padding: 0.25rem; margin: 0.25rem 0; background: ${getColorForStatus(r.status)}20; border-radius: 5px;">
                        ${r.name}
                    </div>
                `).join('')}
            </div>
        `;
        grid.appendChild(card);
    });
}

// Get regions for a specific week
function getRegionsForWeek(weekIndex) {
    // Use real regions data if available
    if (regionsData.length === 0) return [];

    // Simulate which regions peak in which weeks based on typical_peak_week
    return regionsData.filter(region => {
        const peakWeek = region.typical_peak_week || 40;
        const weekNumber = 35 + weekIndex; // Sep Week 1 = week 35
        return Math.abs(peakWeek - weekNumber) <= 1;
    });
}

// Load regions grid
async function loadRegionsGrid() {
    const grid = document.getElementById('regions-grid');
    grid.innerHTML = '';

    for (const region of regionsData) {
        try {
            // Fetch current foliage status
            const foliageResponse = await fetch(buildApiUrl('/api/foliage', { region_id: region.id }));
            const foliageData = await foliageResponse.json();

            const status = foliageData.data?.peak_status || 'not_started';
            const intensity = foliageData.data?.color_intensity || 0;

            const card = document.createElement('div');
            card.className = 'region-card';
            card.innerHTML = `
                <div class="region-image"></div>
                <div class="region-info">
                    <div class="region-name">${region.name}</div>
                    <div class="region-state">${region.state}</div>
                    <span class="region-status status-${status}">${formatStatus(status)}</span>
                    <div style="margin-top: 1rem;">
                        <div>Color Intensity: ${intensity}/10</div>
                    </div>
                </div>
            `;
            card.addEventListener('click', () => showRegionDetails(region.id));
            grid.appendChild(card);
        } catch (err) {
            console.error(`Failed to load region ${region.id}:`, err);
        }
    }
}

// Load regions
async function loadRegions(forceRefresh = false) {
    try {
        const response = await fetch(buildApiUrl('/api/regions'));
        const data = await response.json();
        const normalizedRegions = normalizeRegionsResponse(data);

        if (normalizedRegions.length > 0) {
            const meta = extractRegionsMeta(data, normalizedRegions);
            regionsData = decorateRegions(normalizedRegions, meta);
            updateRegionsMeta({
                source: meta.source,
                description: meta.description,
                rowCount: regionsData.length,
                usingFallback: meta.usingFallback,
                lastFetched: new Date().toISOString(),
                message: data?.message || ''
            });

            console.log('Regions loaded from API:', regionsData);

            if (map) {
                clearMarkers();
                addFoliageMarkers();
            }
            renderDataTransparencyPanel();
            return;
        }

        console.warn('Regions API returned no data, using fallback dataset.');
    } catch (err) {
        console.error('Failed to load regions from API:', err);
        updateRegionsMeta({
            source: 'api-error',
            description: 'The API request failed, falling back to the packaged dataset.',
            usingFallback: true,
            message: err?.message || 'Network error',
            lastFetched: new Date().toISOString()
        });
    }

    // Fallback to sample data for demo
    regionsData = [
        { id: 1, name: 'White Mountains', state: 'New Hampshire', latitude: 44.2700, longitude: -71.3034, current_status: 'near_peak', color_intensity: 7, data_source: 'ui-fallback' },
        { id: 2, name: 'Green Mountains', state: 'Vermont', latitude: 43.9207, longitude: -72.8986, current_status: 'peak', color_intensity: 9, data_source: 'ui-fallback' },
        { id: 3, name: 'Adirondacks', state: 'New York', latitude: 44.1127, longitude: -74.0524, current_status: 'progressing', color_intensity: 5, data_source: 'ui-fallback' },
        { id: 4, name: 'Great Smoky Mountains', state: 'Tennessee', latitude: 35.6532, longitude: -83.5070, current_status: 'near_peak', color_intensity: 8, data_source: 'ui-fallback' },
        { id: 5, name: 'Blue Ridge Parkway', state: 'Virginia', latitude: 37.5615, longitude: -79.3553, current_status: 'peak', color_intensity: 10, data_source: 'ui-fallback' },
        { id: 6, name: 'Berkshires', state: 'Massachusetts', latitude: 42.3604, longitude: -73.2290, current_status: 'progressing', color_intensity: 6, data_source: 'ui-fallback' },
    ];

    updateRegionsMeta({
        source: 'ui-fallback',
        description: 'Static dataset bundled with the UI (no live database connection).',
        rowCount: regionsData.length,
        usingFallback: true,
        lastFetched: new Date().toISOString()
    });

    if (map) {
        clearMarkers();
        addFoliageMarkers();
    }

    renderDataTransparencyPanel();
}

function normalizeRegionsResponse(payload) {
    if (!payload) {
        return [];
    }

    if (Array.isArray(payload.regions)) {
        return payload.regions;
    }

    if (Array.isArray(payload.data?.regions)) {
        return payload.data.regions;
    }

    if (Array.isArray(payload.data)) {
        return payload.data;
    }

    return [];
}

function extractRegionsMeta(payload, normalizedRegions) {
    const defaultDescription = 'Live data provided by the Fall Foliage Explorer API.';
    const meta = payload?.data?.meta;

    if (meta && typeof meta === 'object') {
        return {
            source: meta.source || 'live-api',
            description: meta.source_description || defaultDescription,
            usingFallback: Boolean(meta.using_fallback),
        };
    }

    if (payload?.message) {
        const message = String(payload.message).toLowerCase();
        if (message.includes('sample')) {
            return {
                source: 'api-sample-dataset',
                description: payload.message,
                usingFallback: true,
            };
        }
    }

    return {
        source: 'live-api',
        description: defaultDescription,
        usingFallback: false,
    };
}

function decorateRegions(regions, meta) {
    return regions.map((region, index) => ({
        ...region,
        data_source: region.data_source || meta?.source || 'live-api',
        current_status: region.current_status || inferDefaultStatus(index),
    }));
}

function inferDefaultStatus(index) {
    const statuses = ['progressing', 'near_peak', 'peak'];
    return statuses[index % statuses.length];
}

function updateRegionsMeta(partial = {}) {
    regionsMeta = {
        ...regionsMeta,
        ...partial,
        apiBase: API_BASE,
        apiBaseSource: API_BASE_SOURCE,
        apiBaseNotes: API_BASE_NOTES,
        lastFetched: partial.lastFetched || regionsMeta.lastFetched,
    };

    renderDataTransparencyPanel();
}

function renderDataTransparencyPanel() {
    const statusEl = document.getElementById('data-source-status');
    if (!statusEl) {
        return;
    }

    const statusClass = regionsMeta.usingFallback
        ? 'status-fallback'
        : regionsMeta.source === 'api-error'
            ? 'status-error'
            : 'status-live';

    statusEl.textContent = regionsMeta.usingFallback
        ? 'Fallback Dataset'
        : regionsMeta.source === 'api-error'
            ? 'API Error'
            : 'Live Dataset';
    statusEl.className = `data-source-pill ${statusClass}`;

    const descriptionEl = document.getElementById('data-source-description');
    if (descriptionEl) {
        descriptionEl.textContent = regionsMeta.description || 'Dataset details unavailable.';
    }

    const metaEl = document.getElementById('data-source-meta');
    if (metaEl) {
        const timestamp = regionsMeta.lastFetched
            ? new Date(regionsMeta.lastFetched).toLocaleString()
            : '—';
        const cohort = regionsMeta.rowCount === 1 ? 'region' : 'regions';
        metaEl.textContent = `${regionsMeta.rowCount} ${cohort} • Updated ${timestamp}`;
    }

    const messageEl = document.getElementById('data-source-message');
    if (messageEl) {
        messageEl.textContent = regionsMeta.message || '';
    }

    const apiBaseEl = document.getElementById('api-base-display');
    if (apiBaseEl) {
        apiBaseEl.textContent = regionsMeta.apiBase || API_BASE;
    }

    const apiSourceEl = document.getElementById('api-base-source');
    if (apiSourceEl) {
        apiSourceEl.textContent = formatApiSource(regionsMeta.apiBaseSource);
    }

    populateRegionsTable();
    populateEndpointList();
}

function formatApiSource(source) {
    switch (source) {
        case 'app-monitor-proxy':
            return 'App Monitor Proxy';
        case 'location-derived':
            return 'Derived from Location';
        case 'same-origin':
            return 'Same Origin';
        case 'loopback-fallback':
            return 'Loopback (Local Dev)';
        default:
            return source ? source.replace(/[-_]/g, ' ') : 'Unknown';
    }
}

function populateRegionsTable() {
    const tableBody = document.getElementById('regions-data-table-body');
    if (!tableBody) {
        return;
    }

    tableBody.innerHTML = '';

    if (!regionsData || regionsData.length === 0) {
        const row = document.createElement('tr');
        const cell = document.createElement('td');
        cell.colSpan = 6;
        cell.style.textAlign = 'center';
        cell.textContent = 'No dataset available yet.';
        row.appendChild(cell);
        tableBody.appendChild(row);
        return;
    }

    const maxRows = 8;
    regionsData.slice(0, maxRows).forEach((region) => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${region.id}</td>
            <td>${region.name}</td>
            <td>${region.state || region.country || ''}</td>
            <td>${Number(region.latitude).toFixed(2)}</td>
            <td>${Number(region.longitude).toFixed(2)}</td>
            <td>${formatSourceLabel(region.data_source)}</td>
        `;
        tableBody.appendChild(row);
    });

    if (regionsData.length > maxRows) {
        const row = document.createElement('tr');
        const cell = document.createElement('td');
        cell.colSpan = 6;
        cell.style.textAlign = 'center';
        cell.textContent = `+ ${regionsData.length - maxRows} more regions…`;
        row.appendChild(cell);
        tableBody.appendChild(row);
    }
}

function populateEndpointList() {
    const listEl = document.getElementById('endpoint-list');
    if (!listEl) {
        return;
    }

    const endpoints = [
        {
            name: 'GET /api/regions',
            path: '/api/regions',
            note: 'Returns regions with source metadata'
        },
        {
            name: 'GET /api/foliage?region_id=1',
            path: '/api/foliage',
            params: { region_id: 1 },
            note: 'Latest foliage snapshot for a region'
        },
        {
            name: 'POST /api/reports',
            path: '/api/reports',
            note: 'Submit user-generated foliage reports'
        }
    ];

    listEl.innerHTML = '';

    endpoints.forEach((endpoint) => {
        const item = document.createElement('li');
        const url = endpoint.params
            ? buildApiUrl(endpoint.path, endpoint.params)
            : buildApiUrl(endpoint.path);
        item.innerHTML = `
            <span class="endpoint-name">${endpoint.name}</span>
            <span class="endpoint-url">${url}</span>
            <span class="endpoint-note">${endpoint.note}</span>
        `;
        listEl.appendChild(item);
    });
}

function formatSourceLabel(source) {
    if (!source) {
        return 'unknown';
    }

    if (source.includes('postgres')) {
        return 'postgres';
    }

    if (source.includes('sample') || source.includes('fallback')) {
        return 'sample dataset';
    }

    return source;
}

// Clear existing markers
function clearMarkers() {
    markers.forEach(marker => map.removeLayer(marker));
    markers = [];
}

// Load planner regions
function loadPlannerRegions() {
    const selector = document.getElementById('region-selector');
    selector.innerHTML = '';

    regionsData.forEach(region => {
        const checkbox = document.createElement('div');
        checkbox.className = 'region-checkbox';
        checkbox.innerHTML = `
            <label>
                <input type="checkbox" value="${region.id}">
                ${region.name}, ${region.state}
            </label>
        `;
        selector.appendChild(checkbox);
    });
}

// Save trip plan
function saveTripPlan() {
    const name = document.getElementById('trip-name').value;
    const startDate = document.getElementById('start-date').value;
    const endDate = document.getElementById('end-date').value;
    
    const selectedRegions = Array.from(document.querySelectorAll('#region-selector input:checked'))
        .map(cb => cb.value);
    
    const trip = {
        name,
        startDate,
        endDate,
        regions: selectedRegions,
        timestamp: new Date().toISOString()
    };
    
    // Save to localStorage
    const trips = JSON.parse(localStorage.getItem('foliageTrips') || '[]');
    trips.push(trip);
    localStorage.setItem('foliageTrips', JSON.stringify(trips));
    
    alert('Trip saved successfully!');
    displaySavedTrips();
}

// Display saved trips
function displaySavedTrips() {
    const trips = JSON.parse(localStorage.getItem('foliageTrips') || '[]');
    const list = document.getElementById('saved-trips-list');
    
    list.innerHTML = trips.map(trip => `
        <div style="background: white; padding: 1rem; margin: 0.5rem 0; border-radius: 8px;">
            <strong>${trip.name}</strong><br>
            ${trip.startDate} - ${trip.endDate}<br>
            ${trip.regions.length} regions selected
        </div>
    `).join('');
}

// Show region details
async function showRegionDetails(regionId) {
    const region = regionsData.find(r => r.id === regionId);
    if (!region) return;

    const panel = document.getElementById('info-panel');
    const content = document.getElementById('info-content');

    try {
        // Fetch current foliage status
        const foliageResponse = await fetch(buildApiUrl('/api/foliage', { region_id: regionId }));
        const foliageData = await foliageResponse.json();

        const status = foliageData.data?.peak_status || 'not_started';
        const intensity = foliageData.data?.color_intensity || 0;

        content.innerHTML = `
            <div style="margin-top: 1rem;">
                <p><strong>Location:</strong> ${region.name}, ${region.state}</p>
                <p><strong>Current Status:</strong> ${formatStatus(status)}</p>
                <p><strong>Color Intensity:</strong> ${intensity}/10</p>
                <p><strong>Coordinates:</strong> ${region.latitude.toFixed(4)}, ${region.longitude.toFixed(4)}</p>
                <p><strong>Elevation:</strong> ${region.elevation_meters || 'N/A'} meters</p>

                <div style="margin-top: 1.5rem;">
                    <h4>Forecast</h4>
                    <p>Peak Expected: Week ${region.typical_peak_week || 40} of year</p>
                    <p>Confidence: 85%</p>
                </div>

                <div style="margin-top: 1.5rem;">
                    <h4>Weather Conditions</h4>
                    <p>Temperature: 52°F / 11°C</p>
                    <p>Recent Rainfall: Moderate</p>
                    <p>Night Temps: Optimal for color</p>
                </div>
            </div>
        `;

        document.getElementById('info-title').textContent = region.name;
        panel.classList.add('open');
    } catch (err) {
        console.error(`Failed to show region details for ${regionId}:`, err);
    }
}

// Close info panel
function closeInfoPanel() {
    document.getElementById('info-panel').classList.remove('open');
}

// Photo Gallery Functions
let currentGalleryPhotos = [];

// Initialize gallery when view is activated
async function initializeGallery() {
    // Populate region filters
    const regionFilter = document.getElementById('gallery-region-filter');
    const uploadRegion = document.getElementById('upload-region');

    // Clear existing options (except "All Regions" for filter)
    uploadRegion.innerHTML = '<option value="">Select a region</option>';

    // Add regions to both selects
    regionsData.forEach(region => {
        const filterOption = document.createElement('option');
        filterOption.value = region.id;
        filterOption.textContent = region.name;
        regionFilter.appendChild(filterOption);

        const uploadOption = document.createElement('option');
        uploadOption.value = region.id;
        uploadOption.textContent = region.name;
        uploadRegion.appendChild(uploadOption);
    });

    // Setup form submission
    const form = document.getElementById('photo-upload-form');
    form.addEventListener('submit', handlePhotoSubmit);

    // Setup filter listeners
    regionFilter.addEventListener('change', filterGallery);
    document.getElementById('gallery-date-filter').addEventListener('change', filterGallery);

    // Load initial photos
    loadGalleryPhotos();
}

// Load photos from API
async function loadGalleryPhotos(regionId = null) {
    try {
        const response = await fetch(buildApiUrl('/api/reports', regionId ? { region_id: regionId } : undefined));
        const result = await response.json();

        if (result.status === 'success') {
            currentGalleryPhotos = result.data || [];
            // If no regionId filter, load from all regions
            if (!regionId && regionsData.length > 0) {
                const allPhotos = [];
                for (const region of regionsData) {
                    const regionResponse = await fetch(buildApiUrl('/api/reports', { region_id: region.id }));
                    const regionResult = await regionResponse.json();
                    if (regionResult.status === 'success' && regionResult.data) {
                        allPhotos.push(...regionResult.data);
                    }
                }
                currentGalleryPhotos = allPhotos;
            }
            displayGalleryPhotos(currentGalleryPhotos);
        }
    } catch (error) {
        console.error('Failed to load gallery photos:', error);
        displayGalleryPhotos([]);
    }
}

// Display photos in the gallery
function displayGalleryPhotos(photos) {
    const grid = document.getElementById('photo-grid');

    if (!photos || photos.length === 0) {
        grid.innerHTML = '<p class="no-photos">No photos available. Share the first photo!</p>';
        return;
    }

    grid.innerHTML = photos
        .filter(photo => photo.photo_url) // Only show reports with photos
        .map(photo => {
            const region = regionsData.find(r => r.id === photo.region_id);
            const regionName = region ? region.name : 'Unknown Region';
            const date = photo.report_date || 'Date unknown';

            return `
                <div class="photo-card">
                    <div class="photo-image-container">
                        <img src="${photo.photo_url}" alt="${regionName} foliage"
                             onerror="this.src='data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%22300%22 height=%22200%22%3E%3Crect fill=%22%23ddd%22 width=%22300%22 height=%22200%22/%3E%3Ctext x=%2250%25%22 y=%2250%25%22 dominant-baseline=%22middle%22 text-anchor=%22middle%22 fill=%22%23999%22%3EImage not available%3C/text%3E%3C/svg%3E'">
                    </div>
                    <div class="photo-info">
                        <h4>${regionName}</h4>
                        <p class="photo-status">
                            <span class="status-badge status-${photo.foliage_status}">${formatStatus(photo.foliage_status)}</span>
                        </p>
                        <p class="photo-date">${formatDate(date)}</p>
                        ${photo.description ? `<p class="photo-description">${photo.description}</p>` : ''}
                    </div>
                </div>
            `;
        })
        .join('');
}

// Filter gallery based on selected filters
function filterGallery() {
    const regionFilter = document.getElementById('gallery-region-filter').value;
    const dateFilter = document.getElementById('gallery-date-filter').value;

    let filtered = currentGalleryPhotos;

    if (regionFilter) {
        filtered = filtered.filter(photo => photo.region_id === parseInt(regionFilter));
    }

    if (dateFilter) {
        filtered = filtered.filter(photo => photo.report_date === dateFilter);
    }

    displayGalleryPhotos(filtered);
}

// Handle photo submission
async function handlePhotoSubmit(e) {
    e.preventDefault();

    const regionId = parseInt(document.getElementById('upload-region').value);
    const status = document.getElementById('upload-status').value;
    const photoUrl = document.getElementById('upload-photo-url').value;
    const description = document.getElementById('upload-description').value;

    if (!regionId || !status || !photoUrl) {
        alert('Please fill in all required fields');
        return;
    }

    try {
        const response = await fetch(buildApiUrl('/api/reports'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                region_id: regionId,
                foliage_status: status,
                photo_url: photoUrl,
                description: description
            })
        });

        const result = await response.json();

        if (result.status === 'success') {
            alert('Photo shared successfully!');
            // Clear form
            e.target.reset();
            // Reload gallery
            loadGalleryPhotos();
        } else {
            alert('Failed to share photo: ' + (result.error || 'Unknown error'));
        }
    } catch (error) {
        console.error('Failed to submit photo:', error);
        alert('Failed to share photo. Please try again.');
    }
}

// Format date for display
function formatDate(dateStr) {
    if (!dateStr) return 'Date unknown';
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric'
    });
}

// Export Functions

// Export predictions as CSV
async function exportPredictionsCSV() {
    try {
        const predictions = [];

        for (const region of regionsData) {
            // Fetch foliage data for each region
            const foliageResponse = await fetch(buildApiUrl('/api/foliage', { region_id: region.id }));
            const foliageData = await foliageResponse.json();

            predictions.push({
                region: region.name,
                state: region.state,
                latitude: region.latitude,
                longitude: region.longitude,
                elevation: region.elevation_meters || 'N/A',
                current_status: foliageData.data?.peak_status || 'unknown',
                color_intensity: foliageData.data?.color_intensity || 0,
                typical_peak_week: region.typical_peak_week || 'N/A'
            });
        }

        // Convert to CSV
        const headers = ['Region', 'State', 'Latitude', 'Longitude', 'Elevation (m)', 'Current Status', 'Color Intensity', 'Typical Peak Week'];
        const csvRows = [headers.join(',')];

        predictions.forEach(pred => {
            const row = [
                `"${pred.region}"`,
                `"${pred.state}"`,
                pred.latitude,
                pred.longitude,
                pred.elevation,
                `"${pred.current_status}"`,
                pred.color_intensity,
                pred.typical_peak_week
            ];
            csvRows.push(row.join(','));
        });

        const csvContent = csvRows.join('\n');
        downloadFile(csvContent, 'foliage-predictions.csv', 'text/csv');

    } catch (error) {
        console.error('Failed to export predictions:', error);
        alert('Failed to export predictions. Please try again.');
    }
}

// Export predictions as JSON
async function exportPredictionsJSON() {
    try {
        const predictions = [];

        for (const region of regionsData) {
            // Fetch foliage data for each region
            const foliageResponse = await fetch(buildApiUrl('/api/foliage', { region_id: region.id }));
            const foliageData = await foliageResponse.json();

            predictions.push({
                region: region.name,
                state: region.state,
                coordinates: {
                    latitude: region.latitude,
                    longitude: region.longitude
                },
                elevation_meters: region.elevation_meters || null,
                current_status: foliageData.data?.peak_status || 'unknown',
                color_intensity: foliageData.data?.color_intensity || 0,
                typical_peak_week: region.typical_peak_week || null,
                exported_at: new Date().toISOString()
            });
        }

        const jsonContent = JSON.stringify({
            export_date: new Date().toISOString(),
            predictions: predictions
        }, null, 2);

        downloadFile(jsonContent, 'foliage-predictions.json', 'application/json');

    } catch (error) {
        console.error('Failed to export predictions:', error);
        alert('Failed to export predictions. Please try again.');
    }
}

// Export trip plans as CSV
async function exportTripsCSV() {
    try {
        const tripsResponse = await fetch(buildApiUrl('/api/trips'));
        const tripsData = await tripsResponse.json();

        if (!tripsData.trips || tripsData.trips.length === 0) {
            alert('No trip plans to export');
            return;
        }

        const headers = ['Trip Name', 'Start Date', 'End Date', 'Regions', 'Created'];
        const csvRows = [headers.join(',')];

        tripsData.trips.forEach(trip => {
            const regionNames = trip.region_ids?.map(id => {
                const region = regionsData.find(r => r.id === id);
                return region ? region.name : `Region ${id}`;
            }).join('; ') || '';

            const row = [
                `"${trip.name}"`,
                trip.start_date,
                trip.end_date,
                `"${regionNames}"`,
                new Date(trip.created_at).toLocaleDateString()
            ];
            csvRows.push(row.join(','));
        });

        const csvContent = csvRows.join('\n');
        downloadFile(csvContent, 'foliage-trips.csv', 'text/csv');

    } catch (error) {
        console.error('Failed to export trips:', error);
        alert('Failed to export trips. Please try again.');
    }
}

// Export trip plans as JSON
async function exportTripsJSON() {
    try {
        const tripsResponse = await fetch(buildApiUrl('/api/trips'));
        const tripsData = await tripsResponse.json();

        if (!tripsData.trips || tripsData.trips.length === 0) {
            alert('No trip plans to export');
            return;
        }

        // Enhance trips with region names
        const enhancedTrips = tripsData.trips.map(trip => ({
            ...trip,
            regions: trip.region_ids?.map(id => {
                const region = regionsData.find(r => r.id === id);
                return region ? {
                    id: region.id,
                    name: region.name,
                    state: region.state,
                    coordinates: {
                        latitude: region.latitude,
                        longitude: region.longitude
                    }
                } : { id: id, name: `Unknown Region ${id}` };
            }) || []
        }));

        const jsonContent = JSON.stringify({
            export_date: new Date().toISOString(),
            trips: enhancedTrips
        }, null, 2);

        downloadFile(jsonContent, 'foliage-trips.json', 'application/json');

    } catch (error) {
        console.error('Failed to export trips:', error);
        alert('Failed to export trips. Please try again.');
    }
}

// Utility function to download file
function downloadFile(content, filename, mimeType) {
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
}

Object.assign(window, {
    showRegionDetails,
    exportPredictionsCSV,
    exportPredictionsJSON,
    exportTripsCSV,
    exportTripsJSON,
    closeInfoPanel,
});
