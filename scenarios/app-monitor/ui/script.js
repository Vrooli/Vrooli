// App Monitor Dashboard - Matrix Theme
// Real-time monitoring and control for Vrooli scenarios

const API_BASE = `${window.location.protocol}//${window.location.host}`;

let ws = null;
let currentView = 'apps';
let viewMode = 'card'; // 'card' or 'list'
let apps = [];
let resources = [];
let selectedApp = null;
let currentLogsApp = null;
let currentLogsType = 'both';

// Router functionality
class Router {
    constructor() {
        this.routes = {};
        this.currentRoute = null;
        
        // Listen for popstate events (browser back/forward)
        window.addEventListener('popstate', () => this.handleRoute());
    }
    
    register(path, handler) {
        this.routes[path] = handler;
    }
    
    navigate(path) {
        window.history.pushState(null, '', path);
        this.handleRoute();
    }
    
    handleRoute() {
        const fullPath = window.location.pathname;
        
        // Handle both root and subdirectory deployments
        // If served from subdirectory, extract the app path
        let routePath = fullPath;
        
        // Check if we're served from a subdirectory (e.g., /app-monitor/apps)
        const baseMatch = fullPath.match(/^(\/[^\/]+)?(\/apps|\/metrics|\/logs|\/resources|\/terminal)/);
        if (baseMatch) {
            routePath = baseMatch[2]; // Get just the route part
        }
        
        // Default to /apps if at root
        if (routePath === '/' || routePath === '' || !routePath.match(/^\/(apps|metrics|logs|resources|terminal)/)) {
            routePath = '/apps';
            // Only update URL if we're actually at root
            if (fullPath === '/' || fullPath === '') {
                window.history.replaceState(null, '', routePath);
            }
        }
        
        // Update page title based on route
        const titles = {
            '/apps': 'Applications',
            '/applications': 'Applications',
            '/metrics': 'Performance Metrics',
            '/logs': 'Application Logs',
            '/resources': 'Resources',
            '/terminal': 'Terminal'
        };
        
        // Find matching route
        const handler = this.routes[routePath];
        if (handler) {
            this.currentRoute = routePath;
            document.title = `${titles[routePath] || 'Dashboard'} - VROOLI Monitor`;
            handler();
            
            // Update active menu item
            document.querySelectorAll('.menu-item').forEach(item => {
                const viewName = item.dataset.view;
                const routeName = routePath.substring(1); // Remove leading slash
                if (viewName === routeName || (viewName === 'apps' && routeName === 'applications')) {
                    item.classList.add('active');
                } else {
                    item.classList.remove('active');
                }
            });
        } else {
            // Check if it's an app logs route
            const appLogsMatch = routePath.match(/^\/logs\/(.+)$/);
            if (appLogsMatch) {
                const appId = decodeURIComponent(appLogsMatch[1]);
                this.currentRoute = '/logs';
                document.title = `Logs: ${appId} - VROOLI Monitor`;
                switchView('logs');
                // Set app selector to this app
                const appSelector = document.getElementById('app-selector');
                appSelector.value = appId;
                currentLogsApp = appId;
                loadAppLogs(appId);
                
                // Update menu
                document.querySelectorAll('.menu-item').forEach(item => {
                    item.classList.toggle('active', item.dataset.view === 'logs');
                });
            } else {
                // Route not found, redirect to apps
                this.navigate('/apps');
            }
        }
    }
}

const router = new Router();

// Initialize dashboard
document.addEventListener('DOMContentLoaded', async () => {
    initializeWebSocket();
    initializeEventListeners();
    setupRoutes();
    await loadInitialData();  // Wait for initial data to load
    startUptimeCounter();     // Now start the counter with actual uptime
    initializeMatrixEffect();
    
    // Handle initial route
    router.handleRoute();
});

// WebSocket connection for real-time updates
function initializeWebSocket() {
    const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;
    
    try {
        ws = new WebSocket(wsUrl);
        
        ws.onopen = () => {
            console.log('WebSocket connected');
            updateSystemStatus('ONLINE');
            addLog('info', 'WebSocket connection established');
        };
        
        ws.onmessage = (event) => {
            handleWebSocketMessage(JSON.parse(event.data));
        };
        
        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            addLog('error', 'WebSocket connection error');
        };
        
        ws.onclose = () => {
            console.log('WebSocket disconnected');
            updateSystemStatus('OFFLINE');
            addLog('warning', 'WebSocket connection closed - attempting reconnect...');
            setTimeout(initializeWebSocket, 5000);
        };
    } catch (error) {
        console.error('Failed to initialize WebSocket:', error);
        addLog('error', `WebSocket initialization failed: ${error.message}`);
    }
}

// Handle incoming WebSocket messages
function handleWebSocketMessage(data) {
    switch (data.type) {
        case 'app_update':
            updateApp(data.payload);
            break;
        case 'metric_update':
            updateMetrics(data.payload);
            break;
        case 'log_entry':
            addLog(data.payload.level, data.payload.message, data.payload.timestamp);
            break;
        case 'resource_update':
            updateResource(data.payload);
            break;
        default:
            console.log('Unknown message type:', data.type);
    }
}

// Initialize event listeners
// Setup routes
function setupRoutes() {
    router.register('/apps', () => switchView('apps'));
    router.register('/applications', () => switchView('apps')); // Alias
    router.register('/metrics', () => switchView('metrics'));
    router.register('/logs', () => switchView('logs'));
    router.register('/resources', () => switchView('resources'));
    router.register('/terminal', () => switchView('terminal'));
}

function initializeEventListeners() {
    // Navigation menu - now uses router
    document.querySelectorAll('.menu-item').forEach(item => {
        item.addEventListener('click', () => {
            const view = item.dataset.view;
            router.navigate('/' + view);
        });
    });
    
    // Quick actions
    document.getElementById('restart-all').addEventListener('click', restartAllApps);
    document.getElementById('stop-all').addEventListener('click', stopAllApps);
    document.getElementById('health-check').addEventListener('click', performHealthCheck);
    
    // App controls
    document.getElementById('refresh-apps').addEventListener('click', loadApps);
    document.getElementById('app-search').addEventListener('input', filterApps);
    document.getElementById('toggle-view').addEventListener('click', toggleViewMode);
    
    // Logs view controls
    document.getElementById('clear-logs').addEventListener('click', () => {
        document.getElementById('logs-output').innerHTML = '';
    });
    
    // Populate app selector
    window.updateAppSelector = function() {
        const appSelector = document.getElementById('app-selector');
        const currentValue = appSelector.value;
        appSelector.innerHTML = '<option value="">SELECT APP...</option>';
        apps.forEach(app => {
            const option = document.createElement('option');
            option.value = app.id;
            option.textContent = `${app.name} (${app.status})`;
            appSelector.appendChild(option);
        });
        // Restore previous selection if it still exists
        if (currentValue && apps.find(a => a.id === currentValue)) {
            appSelector.value = currentValue;
        }
    };
    
    // App selector change handler
    document.getElementById('app-selector').addEventListener('change', (e) => {
        currentLogsApp = e.target.value;
        loadAppLogs(currentLogsApp);
    });
    
    // Log type change handler  
    document.getElementById('log-type').addEventListener('change', (e) => {
        currentLogsType = e.target.value;
        if (currentLogsApp) {
            loadAppLogs(currentLogsApp);
        }
    });
    
    // Refresh logs button
    document.getElementById('refresh-logs').addEventListener('click', () => {
        if (currentLogsApp) {
            loadAppLogs(currentLogsApp);
        }
    });
    
    // Log controls
    document.getElementById('log-level').addEventListener('change', filterLogs);
    
    // Terminal input
    document.getElementById('terminal-input').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            executeCommand(e.target.value);
            e.target.value = '';
        }
    });
    
    // Modal controls
    document.getElementById('close-modal').addEventListener('click', closeModal);
    document.getElementById('modal-start').addEventListener('click', () => controlApp('start'));
    document.getElementById('modal-stop').addEventListener('click', () => controlApp('stop'));
    document.getElementById('modal-restart').addEventListener('click', () => controlApp('restart'));
    document.getElementById('modal-logs').addEventListener('click', () => viewAppLogs());
}

// Load initial data
async function loadInitialData() {
    await loadSystemInfo();
    await loadApps();
    await loadResources();
    await loadMetrics();
}

// Load apps from API
async function loadApps() {
    try {
        const response = await fetch(`${API_BASE}/api/apps`);
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        
        apps = await response.json();
        renderApps();
        updateAppCount(apps.length);
        updateAppSelector(); // Update the app selector in logs view
        addLog('info', `Loaded ${apps.length} applications`);
    } catch (error) {
        console.error('Failed to load apps:', error);
        addLog('error', `Failed to load applications: ${error.message}`);
        apps = [];
        renderApps();
        updateAppCount(0);
        updateAppSelector();
    }
}

// Load resources from API
async function loadResources() {
    try {
        const response = await fetch(`${API_BASE}/api/resources`);
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        
        resources = await response.json();
        renderResources();
    } catch (error) {
        console.error('Failed to load resources:', error);
        addLog('error', `Failed to load resources: ${error.message}`);
        resources = [];
        renderResources();
    }
}

// Load metrics
async function loadMetrics() {
    try {
        const response = await fetch(`${API_BASE}/api/metrics`);
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        
        const metrics = await response.json();
        updateMetrics(metrics);
    } catch (error) {
        console.error('Failed to load metrics:', error);
        addLog('error', `Failed to load metrics: ${error.message}`);
        // Show zero metrics on error
        updateMetrics({ cpu: 0, memory: 0, network: 0, disk: 0 });
    }
}

// Toggle between card and list view
function toggleViewMode() {
    viewMode = viewMode === 'card' ? 'list' : 'card';
    const container = document.getElementById('apps-container');
    const toggleBtn = document.getElementById('toggle-view');
    
    if (viewMode === 'list') {
        container.classList.remove('apps-grid');
        container.classList.add('apps-list');
        toggleBtn.innerHTML = '☰'; // List icon
        toggleBtn.title = 'Switch to Card View';
    } else {
        container.classList.remove('apps-list');
        container.classList.add('apps-grid');
        toggleBtn.innerHTML = '⊞'; // Grid icon
        toggleBtn.title = 'Switch to List View';
    }
    
    renderApps();
}

// Render apps grid or list
function renderApps() {
    const container = document.getElementById('apps-container');
    container.innerHTML = '';
    
    if (viewMode === 'list') {
        // Render list view
        const table = createAppsList();
        container.appendChild(table);
    } else {
        // Render card view
        apps.forEach(app => {
            const card = createAppCard(app);
            container.appendChild(card);
        });
    }
}

// Create app card element
function createAppCard(app) {
    const card = document.createElement('div');
    card.className = 'app-card';
    
    // Extract ports from port_mappings
    const ports = app.port_mappings || app.portMappings || {};
    const portEntries = Object.entries(ports);
    
    // Find UI port if available
    let uiPort = null;
    if (ports.ui) {
        uiPort = ports.ui;
    } else if (ports.UI) {
        uiPort = ports.UI;
    } else if (ports.web) {
        uiPort = ports.web;
    }
    
    // Create ports display
    let portsHtml = '';
    if (portEntries.length > 0) {
        portsHtml = portEntries.slice(0, 3).map(([name, port]) => `
            <div class="app-metric">
                <div class="metric-label">${name.toUpperCase()}</div>
                <div class="metric-data port-number">${port}</div>
            </div>
        `).join('');
        
        // Add indicator if there are more ports
        if (portEntries.length > 3) {
            portsHtml += `
                <div class="app-metric">
                    <div class="metric-label">MORE</div>
                    <div class="metric-data">+${portEntries.length - 3}</div>
                </div>
            `;
        }
    } else {
        portsHtml = `
            <div class="app-metric">
                <div class="metric-label">PORTS</div>
                <div class="metric-data">No ports</div>
            </div>
        `;
    }
    
    // Add click handler to entire card if it has a UI port
    if (uiPort) {
        card.style.cursor = 'pointer';
        card.onclick = (e) => {
            // Don't open if clicking on action buttons
            if (!e.target.classList.contains('app-btn')) {
                window.open(`http://localhost:${uiPort}`, '_blank');
            }
        };
    }
    
    card.innerHTML = `
        <div class="app-header">
            <span class="app-name">${app.name}</span>
            <span class="app-status ${app.status}">${app.status.toUpperCase()}</span>
        </div>
        <div class="app-metrics">
            ${portsHtml}
        </div>
        <div class="app-actions">
            <button class="app-btn" data-app-id="${app.id}" data-action="start">START</button>
            <button class="app-btn" data-app-id="${app.id}" data-action="stop">STOP</button>
            <button class="app-btn" data-app-id="${app.id}" data-action="details">DETAILS</button>
        </div>
    `;
    
    // Add event listeners to buttons
    const buttons = card.querySelectorAll('.app-btn');
    buttons.forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            const appId = btn.dataset.appId;
            const action = btn.dataset.action;
            
            if (action === 'details') {
                showAppDetails(appId);
            } else {
                // Add loading indicator
                btn.disabled = true;
                btn.textContent = '...';
                controlAppDirect(appId, action).finally(() => {
                    btn.disabled = false;
                    btn.textContent = action.toUpperCase();
                });
            }
        });
    });
    
    return card;
}

// Create apps list view
function createAppsList() {
    const table = document.createElement('div');
    table.className = 'apps-table';
    
    // Create header
    const header = document.createElement('div');
    header.className = 'table-header';
    header.innerHTML = `
        <div class="table-row">
            <div class="table-cell">NAME</div>
            <div class="table-cell">STATUS</div>
            <div class="table-cell">PORTS</div>
            <div class="table-cell">ACTIONS</div>
        </div>
    `;
    table.appendChild(header);
    
    // Create rows
    const tbody = document.createElement('div');
    tbody.className = 'table-body';
    
    apps.forEach(app => {
        const row = createAppListRow(app);
        tbody.appendChild(row);
    });
    
    table.appendChild(tbody);
    return table;
}

// Create app list row
function createAppListRow(app) {
    const row = document.createElement('div');
    row.className = 'table-row app-row';
    
    // Extract ports
    const ports = app.port_mappings || app.portMappings || {};
    const portEntries = Object.entries(ports);
    
    // Find UI port
    let uiPort = null;
    if (ports.ui) uiPort = ports.ui;
    else if (ports.UI) uiPort = ports.UI;
    else if (ports.web) uiPort = ports.web;
    
    // Format ports display
    let portsDisplay = 'No ports';
    if (portEntries.length > 0) {
        portsDisplay = portEntries.map(([name, port]) => 
            `<span class="port-tag">${name}: ${port}</span>`
        ).join(' ');
    }
    
    // Add click handler if has UI port
    if (uiPort) {
        row.style.cursor = 'pointer';
        row.onclick = (e) => {
            if (!e.target.classList.contains('app-btn')) {
                window.open(`http://localhost:${uiPort}`, '_blank');
            }
        };
    }
    
    row.innerHTML = `
        <div class="table-cell app-name-cell">
            <span class="app-name">${app.name}</span>
        </div>
        <div class="table-cell status-cell">
            <span class="app-status ${app.status}">${app.status.toUpperCase()}</span>
        </div>
        <div class="table-cell ports-cell">
            ${portsDisplay}
        </div>
        <div class="table-cell actions-cell">
            <button class="app-btn small" data-app-id="${app.id}" data-action="start">▶</button>
            <button class="app-btn small" data-app-id="${app.id}" data-action="stop">■</button>
            <button class="app-btn small" data-app-id="${app.id}" data-action="restart">⟳</button>
            <button class="app-btn small" data-app-id="${app.id}" data-action="details">ℹ</button>
        </div>
    `;
    
    // Add event listeners to buttons
    const buttons = row.querySelectorAll('.app-btn');
    buttons.forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            const appId = btn.dataset.appId;
            const action = btn.dataset.action;
            
            if (action === 'details') {
                showAppDetails(appId);
            } else if (action === 'restart') {
                btn.disabled = true;
                btn.textContent = '...';
                controlAppDirect(appId, 'restart').finally(() => {
                    btn.disabled = false;
                    btn.textContent = '⟳';
                });
            } else {
                btn.disabled = true;
                btn.textContent = '...';
                controlAppDirect(appId, action).finally(() => {
                    btn.disabled = false;
                    btn.textContent = action === 'start' ? '▶' : '■';
                });
            }
        });
    });
    
    return row;
}

// Render resources
function renderResources() {
    const container = document.getElementById('resources-container');
    container.innerHTML = '';
    
    resources.forEach(resource => {
        const card = createResourceCard(resource);
        container.appendChild(card);
    });
}

// Create resource card
function createResourceCard(resource) {
    const card = document.createElement('div');
    card.className = 'resource-card';
    card.innerHTML = `
        <div class="resource-icon">${getResourceIcon(resource.type)}</div>
        <div class="resource-name">${resource.name}</div>
        <div class="resource-status ${resource.status}">${resource.status}</div>
    `;
    return card;
}

// Get resource icon based on type
function getResourceIcon(type) {
    const icons = {
        'postgres': '🗄️',
        'postgresql': '🗄️',
        'redis': '⚡',
        'n8n': '🔄',
        'ollama': '🤖',
        'windmill': '🌀',
        'node-red': '🔴',
        'nodered': '🔴',
        'qdrant': '🔍',
        'minio': '📦',
        'docker': '🐳',
        'kubernetes': '☸️',
        'mongodb': '🍃',
        'mysql': '🐬',
        'elasticsearch': '🔎',
        'kafka': '📨',
        'rabbitmq': '🐰',
        'nginx': '🌐',
        'apache': '🪶',
        'jenkins': '🏗️',
        'gitlab': '🦊',
        'github': '🐙',
        'grafana': '📊',
        'prometheus': '📈',
        'vault': '🔐',
        'consul': '🔗',
        'terraform': '🏗️',
        'ansible': '⚙️',
        'traefik': '🚦',
        'portainer': '🚢',
        'keycloak': '🔑',
        'jupyterhub': '📓',
        'airflow': '🌬️',
        'superset': '📊',
        'metabase': '📊',
        'hasura': '⚡',
        'strapi': '🚀',
        'supabase': '⚡',
        'appsmith': '🛠️',
        'budibase': '🏗️',
        'nocodb': '📊',
        'baserow': '📊',
        'directus': '🎯',
        'pocketbase': '📦',
        'umami': '📊',
        'plausible': '📊',
        'matomo': '📊',
        'sentry': '🚨',
        'datadog': '🐕',
        'newrelic': '👁️',
        'splunk': '🔍',
        'elastic': '🔍',
        'logstash': '📝',
        'kibana': '📊',
        'fluentd': '💧',
        'vector': '➡️',
        'temporal': '⏰',
        'camunda': '⚙️',
        'activiti': '⚙️',
        'prefect': '🌊',
        'dagster': '⛰️',
        'dbt': '🔧',
        'spark': '✨',
        'flink': '⚡',
        'beam': '📡',
        'nifi': '🌊',
        'streamsets': '🌊',
        'airbyte': '🔄',
        'fivetran': '🔄',
        'stitch': '🔄',
        'segment': '📊',
        'rudderstack': '📊',
        'snowplow': '❄️',
        'mixpanel': '📊',
        'amplitude': '📊',
        'heap': '📊',
        'posthog': '🦔',
        'growthbook': '📊',
        'flagsmith': '🚩',
        'launchdarkly': '🚀',
        'unleash': '🚀',
        'flipt': '🔄',
        'statsig': '📊',
        'split': '🔀',
        'optimizely': '🎯'
    };
    return icons[type.toLowerCase()] || '📊';
}

// Switch view
function switchView(view) {
    // Update panels
    document.querySelectorAll('.view-panel').forEach(panel => {
        panel.classList.remove('active');
    });
    document.getElementById(`${view}-view`).classList.add('active');
    
    // Update menu (only if not called by router)
    if (!router.currentRoute || router.currentRoute !== '/' + view) {
        document.querySelectorAll('.menu-item').forEach(item => {
            item.classList.remove('active');
        });
        const menuItem = document.querySelector(`[data-view="${view}"]`);
        if (menuItem) {
            menuItem.classList.add('active');
        }
    }
    
    currentView = view;
    
    // Load view-specific data
    switch (view) {
        case 'metrics':
            initializeCharts();
            // Refresh metrics every 5 seconds when on metrics view
            if (window.metricsInterval) {
                clearInterval(window.metricsInterval);
            }
            window.metricsInterval = setInterval(() => {
                if (currentView === 'metrics') {
                    initializeCharts();
                } else {
                    clearInterval(window.metricsInterval);
                }
            }, 5000);
            break;
        case 'terminal':
            document.getElementById('terminal-input').focus();
            break;
        default:
            // Clear metrics interval if switching away from metrics
            if (window.metricsInterval) {
                clearInterval(window.metricsInterval);
            }
            break;
    }
}

// Filter apps
function filterApps() {
    const searchTerm = document.getElementById('app-search').value.toLowerCase();
    
    if (viewMode === 'list') {
        // Filter list rows
        const rows = document.querySelectorAll('.app-row');
        rows.forEach(row => {
            const appName = row.querySelector('.app-name').textContent.toLowerCase();
            row.style.display = appName.includes(searchTerm) ? 'grid' : 'none';
        });
    } else {
        // Filter cards
        const cards = document.querySelectorAll('.app-card');
        cards.forEach(card => {
            const appName = card.querySelector('.app-name').textContent.toLowerCase();
            card.style.display = appName.includes(searchTerm) ? 'block' : 'none';
        });
    }
}

// Control app directly  
window.controlAppDirect = async function(appId, action) {
    try {
        const response = await fetch(`${API_BASE}/api/apps/${appId}/${action}`, {
            method: 'POST'
        });
        
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        
        const result = await response.json();
        addLog('success', `App ${appId} ${action} successful`);
        
        // Wait a moment for the action to take effect
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        loadApps(); // Reload apps to show updated status
        return result;
    } catch (error) {
        console.error(`Failed to ${action} app:`, error);
        addLog('error', `Failed to ${action} app: ${error.message}`);
        throw error;
    }
};

// Show app details
window.showAppDetails = function(appId) {
    const app = apps.find(a => a.id === appId);
    if (!app) return;
    
    selectedApp = app;
    
    // Extract ports
    const ports = app.port_mappings || app.portMappings || {};
    const portEntries = Object.entries(ports);
    
    // Create ports display
    let portsDisplay = 'None';
    if (portEntries.length > 0) {
        portsDisplay = portEntries.map(([name, port]) => 
            `<span style="display: inline-block; margin-right: 10px;">${name}: <strong>${port}</strong></span>`
        ).join('');
    }
    
    // Calculate uptime if we have created_at
    let uptime = '0h';
    if (app.created_at) {
        const startTime = new Date(app.created_at);
        const now = new Date();
        const diff = now - startTime;
        const hours = Math.floor(diff / 3600000);
        const minutes = Math.floor((diff % 3600000) / 60000);
        uptime = hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;
    }
    
    document.getElementById('modal-app-name').textContent = app.name.toUpperCase();
    document.getElementById('modal-body').innerHTML = `
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
                <strong>Status:</strong> <span class="${app.status}">${app.status.toUpperCase()}</span>
            </div>
            <div>
                <strong>Uptime:</strong> ${uptime}
            </div>
            <div style="grid-column: 1 / -1;">
                <strong>Available Ports:</strong><br/>
                ${portsDisplay}
            </div>
            <div>
                <strong>CPU Usage:</strong> ${app.cpu || 'N/A'}
            </div>
            <div>
                <strong>Memory:</strong> ${app.memory || 'N/A'}
            </div>
            <div style="grid-column: 1 / -1;">
                <strong>Path:</strong> ${app.path || 'scenarios/' + app.name}
            </div>
            <div style="grid-column: 1 / -1;">
                <strong>Description:</strong> ${app.description || 'No description available'}
            </div>
        </div>
    `;
    
    document.getElementById('app-modal').classList.add('active');
};

// Close modal
function closeModal() {
    document.getElementById('app-modal').classList.remove('active');
    selectedApp = null;
}

// Control app from modal
async function controlApp(action) {
    if (!selectedApp) return;
    
    await controlAppDirect(selectedApp.id, action);
    closeModal();
}

// View app logs
async function viewAppLogs() {
    if (!selectedApp) return;
    
    closeModal();
    
    // Navigate to logs page with app ID in URL
    router.navigate(`/logs/${encodeURIComponent(selectedApp.id)}`);
}

// Load logs for a specific app
async function loadAppLogs(appId) {
    if (!appId) {
        document.getElementById('logs-output').innerHTML = '<div class="log-entry info">Select an app to view logs</div>';
        return;
    }
    
    // Clear existing logs
    document.getElementById('logs-output').innerHTML = '';
    
    const logType = document.getElementById('log-type').value;
    currentLogsType = logType;
    
    // Show loading message
    addLog('info', `Loading logs for ${appId}...`);
    
    // Load lifecycle logs if needed
    if (logType === 'both' || logType === 'lifecycle') {
        try {
            const response = await fetch(`${API_BASE}/api/apps/${appId}/logs/lifecycle`);
            if (response.ok) {
                const lifecycleLogs = await response.text();
                if (lifecycleLogs && lifecycleLogs.trim() && lifecycleLogs !== 'No lifecycle logs available' && !lifecycleLogs.includes('not found')) {
                    if (logType === 'both') {
                        addLog('info', '=== LIFECYCLE LOGS ===');
                    }
                    lifecycleLogs.split('\n').forEach(line => {
                        if (line.trim()) {
                            // Try to detect log level from line content
                            let level = 'log';
                            const lowerLine = line.toLowerCase();
                            if (lowerLine.includes('error') || lowerLine.includes('failed')) {
                                level = 'error';
                            } else if (lowerLine.includes('warn')) {
                                level = 'warning';
                            } else if (lowerLine.includes('info')) {
                                level = 'info';
                            } else if (lowerLine.includes('debug')) {
                                level = 'debug';
                            } else if (lowerLine.includes('success') || lowerLine.includes('started')) {
                                level = 'success';
                            }
                            addLog(level, line);
                        }
                    });
                } else if (logType === 'lifecycle') {
                    addLog('warning', 'No lifecycle logs available for this app');
                }
            } else {
                if (logType === 'lifecycle') {
                    addLog('warning', `Lifecycle logs not available (${response.status})`);
                }
            }
        } catch (error) {
            console.error('Failed to load lifecycle logs:', error);
            if (logType === 'lifecycle') {
                addLog('error', `Failed to load lifecycle logs: ${error.message}`);
            }
        }
    }
    
    // Load background task logs if needed
    if (logType === 'both' || logType === 'background') {
        try {
            const response = await fetch(`${API_BASE}/api/apps/${appId}/logs/background`);
            if (response.ok) {
                const backgroundLogs = await response.json();
                if (backgroundLogs && backgroundLogs.length > 0) {
                    // Filter out the generic "no logs found" message if we have actual logs
                    const realLogs = backgroundLogs.filter(log => 
                        !log.message.includes('No background process logs found')
                    );
                    
                    if (realLogs.length > 0) {
                        if (logType === 'both') {
                            addLog('info', '=== BACKGROUND PROCESS LOGS ===');
                        }
                        realLogs.forEach(log => {
                            addLog(log.level || 'log', log.message);
                        });
                    } else if (logType === 'background') {
                        addLog('info', 'No background process logs available for this app');
                    }
                } else if (logType === 'background') {
                    addLog('info', 'No background process logs available for this app');
                }
            } else {
                if (logType === 'background') {
                    addLog('warning', `Background logs not available (${response.status})`);
                }
            }
        } catch (error) {
            console.error('Failed to load background logs:', error);
            if (logType === 'background') {
                addLog('error', `Failed to load background logs: ${error.message}`);
            }
        }
    }
}

// Load app logs
async function loadAppLogs(appId) {
    try {
        const response = await fetch(`${API_BASE}/api/apps/${appId}/logs`);
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        
        const logs = await response.json();
        const container = document.getElementById('logs-output');
        container.innerHTML = '';
        
        logs.forEach(log => {
            addLog(log.level, log.message, log.timestamp);
        });
    } catch (error) {
        console.error('Failed to load app logs:', error);
        addLog('error', `Failed to load logs: ${error.message}`);
    }
}

// Add log entry
function addLog(level, message, timestamp = null) {
    const container = document.getElementById('logs-output');
    const entry = document.createElement('div');
    entry.className = `log-entry ${level}`;
    
    const time = timestamp ? new Date(timestamp).toLocaleTimeString() : new Date().toLocaleTimeString();
    
    entry.innerHTML = `
        <span class="log-timestamp">[${time}]</span>
        <span class="log-level">[${level.toUpperCase()}]</span>
        ${message}
    `;
    
    container.appendChild(entry);
    container.scrollTop = container.scrollHeight;
}

// Filter logs
function filterLogs() {
    const level = document.getElementById('log-level').value;
    const entries = document.querySelectorAll('.log-entry');
    
    entries.forEach(entry => {
        if (level === 'all' || entry.classList.contains(level)) {
            entry.style.display = 'block';
        } else {
            entry.style.display = 'none';
        }
    });
}

// Clear logs
function clearLogs() {
    document.getElementById('logs-output').innerHTML = '';
    addLog('info', 'Logs cleared');
}

// Execute terminal command
async function executeCommand(command) {
    const output = document.getElementById('terminal-output');
    
    // Add command to output
    const commandLine = document.createElement('div');
    commandLine.className = 'terminal-line';
    commandLine.textContent = `vrooli@matrix:~$ ${command}`;
    output.appendChild(commandLine);
    
    // Process command
    try {
        const response = await processCommand(command);
        
        const responseLine = document.createElement('div');
        responseLine.className = 'terminal-line success';
        responseLine.textContent = response;
        output.appendChild(responseLine);
    } catch (error) {
        const errorLine = document.createElement('div');
        errorLine.className = 'terminal-line error';
        errorLine.textContent = `Error: ${error.message}`;
        output.appendChild(errorLine);
    }
    
    output.scrollTop = output.scrollHeight;
}

// Process terminal command
async function processCommand(command) {
    const [cmd, ...args] = command.trim().split(' ');
    
    switch (cmd) {
        case 'help':
            return `Available commands:
  list apps     - List all applications
  list resources - List all resources  
  start <app>   - Start an application
  stop <app>    - Stop an application
  restart <app> - Restart an application
  status        - Show system status
  clear         - Clear terminal`;
            
        case 'list':
            if (args[0] === 'apps') {
                return apps.map(a => `${a.name} (${a.status})`).join('\n');
            } else if (args[0] === 'resources') {
                return resources.map(r => `${r.name} (${r.status})`).join('\n');
            }
            break;
            
        case 'start':
        case 'stop':
        case 'restart':
            if (args[0]) {
                const app = apps.find(a => a.name === args[0]);
                if (app) {
                    await controlAppDirect(app.id, cmd);
                    return `${cmd} command sent to ${args[0]}`;
                }
                return `App not found: ${args[0]}`;
            }
            return `Usage: ${cmd} <app-name>`;
            
        case 'status':
            return `System Status: ${document.getElementById('system-status').textContent}
Apps Running: ${apps.filter(a => a.status === 'running').length}/${apps.length}
Resources Online: ${resources.filter(r => r.status === 'online').length}/${resources.length}`;
            
        case 'clear':
            document.getElementById('terminal-output').innerHTML = '';
            return '';
            
        default:
            return `Command not found: ${cmd}. Type 'help' for available commands.`;
    }
    
    return 'Invalid command syntax';
}

// Quick actions
async function restartAllApps() {
    if (!confirm('Restart all applications?')) return;
    
    addLog('info', 'Restarting all applications...');
    
    for (const app of apps) {
        await controlAppDirect(app.id, 'restart');
    }
    
    addLog('success', 'All applications restarted');
}

async function stopAllApps() {
    if (!confirm('Stop all applications?')) return;
    
    addLog('info', 'Stopping all applications...');
    
    for (const app of apps) {
        await controlAppDirect(app.id, 'stop');
    }
    
    addLog('success', 'All applications stopped');
}

async function performHealthCheck() {
    addLog('info', 'Performing system health check...');
    
    try {
        const response = await fetch(`${API_BASE}/api/health-check`);
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        
        const results = await response.json();
        
        results.forEach(result => {
            addLog(result.healthy ? 'success' : 'error', 
                   `${result.name}: ${result.message}`);
        });
    } catch (error) {
        console.error('Health check failed:', error);
        addLog('error', `Health check failed: ${error.message}`);
    }
}

// Update functions
function updateSystemStatus(status) {
    document.getElementById('system-status').textContent = status;
}

function updateAppCount(count) {
    document.getElementById('app-count').textContent = count;
}

function updateApp(appData) {
    const index = apps.findIndex(a => a.id === appData.id);
    if (index !== -1) {
        apps[index] = { ...apps[index], ...appData };
        renderApps();
    }
}

function updateResource(resourceData) {
    const index = resources.findIndex(r => r.id === resourceData.id);
    if (index !== -1) {
        resources[index] = { ...resources[index], ...resourceData };
        renderResources();
    }
}

function updateMetrics(metrics) {
    document.getElementById('cpu-value').textContent = `${metrics.cpu || 0}%`;
    document.getElementById('memory-value').textContent = `${metrics.memory || 0} MB`;
    document.getElementById('network-value').textContent = `${metrics.network || 0} KB/s`;
    document.getElementById('disk-value').textContent = `${metrics.disk || 0}%`;
    
    // Update charts if in metrics view
    if (currentView === 'metrics') {
        updateCharts(metrics);
    }
}

// Initialize charts with real metrics
async function initializeCharts() {
    try {
        const response = await fetch(`${API_BASE}/api/system/metrics`);
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        
        const metrics = await response.json();
        
        // Draw charts with real data
        drawChart('cpu-chart', metrics.cpu || 0);
        drawChart('memory-chart', metrics.memory || 0);
        drawChart('network-chart', Math.min(metrics.network || 0, 100)); // Cap network at 100
        drawChart('disk-chart', metrics.disk || 0);
        
        // Update the value displays
        document.getElementById('cpu-value').textContent = `${Math.round(metrics.cpu || 0)}%`;
        document.getElementById('memory-value').textContent = `${Math.round(metrics.memory || 0)}%`;
        document.getElementById('network-value').textContent = `${Math.round(metrics.network || 0)} MB`;
        document.getElementById('disk-value').textContent = `${Math.round(metrics.disk || 0)}%`;
    } catch (error) {
        console.error('Failed to load system metrics:', error);
        // Draw empty charts on error
        drawChart('cpu-chart', 0);
        drawChart('memory-chart', 0);
        drawChart('network-chart', 0);
        drawChart('disk-chart', 0);
    }
}

function drawChart(canvasId, value) {
    const canvas = document.getElementById(canvasId);
    if (!canvas) return;
    
    const ctx = canvas.getContext('2d');
    const width = canvas.width;
    const height = canvas.height;
    
    // Clear canvas
    ctx.clearRect(0, 0, width, height);
    
    // Draw simple bar chart
    ctx.fillStyle = '#00ff41';
    ctx.fillRect(0, height - (height * value / 100), width, height * value / 100);
    
    // Draw grid lines
    ctx.strokeStyle = '#008f11';
    ctx.lineWidth = 1;
    for (let i = 0; i < 5; i++) {
        const y = height * i / 4;
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(width, y);
        ctx.stroke();
    }
}

function updateCharts(metrics) {
    drawChart('cpu-chart', metrics.cpu || 0);
    drawChart('memory-chart', metrics.memory || 0);
    drawChart('network-chart', Math.min(metrics.network || 0, 100));
    drawChart('disk-chart', metrics.disk || 0);
    
    // Update the value displays
    document.getElementById('cpu-value').textContent = `${Math.round(metrics.cpu || 0)}%`;
    document.getElementById('memory-value').textContent = `${Math.round(metrics.memory || 0)}%`;
    document.getElementById('network-value').textContent = `${Math.round(metrics.network || 0)} MB`;
    document.getElementById('disk-value').textContent = `${Math.round(metrics.disk || 0)}%`;
}

// Global variable to store initial uptime
let orchestratorUptimeSeconds = 0;

// Load system info including real uptime
async function loadSystemInfo() {
    try {
        const response = await fetch(`${API_BASE}/api/system/info`);
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        
        const systemInfo = await response.json();
        console.log('System info loaded:', systemInfo); // Debug log
        
        if (systemInfo.orchestrator_running) {
            orchestratorUptimeSeconds = systemInfo.uptime_seconds || 0;
            console.log('Orchestrator uptime seconds set to:', orchestratorUptimeSeconds); // Debug log
            updateSystemStatus('ONLINE');
            addLog('info', `Orchestrator running - PID: ${systemInfo.orchestrator_pid}, Uptime: ${systemInfo.uptime}`);
            
            // Immediately display the actual uptime
            const days = Math.floor(orchestratorUptimeSeconds / 86400);
            const hours = Math.floor((orchestratorUptimeSeconds % 86400) / 3600);
            const minutes = Math.floor((orchestratorUptimeSeconds % 3600) / 60);
            const secs = orchestratorUptimeSeconds % 60;
            
            let uptime;
            if (days > 0) {
                uptime = `${days}d ${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
            } else {
                uptime = `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
            }
            
            document.getElementById('uptime').textContent = uptime;
        } else {
            orchestratorUptimeSeconds = 0;
            updateSystemStatus('OFFLINE');
            addLog('warning', 'Orchestrator not running');
        }
    } catch (error) {
        console.error('Failed to load system info:', error);
        orchestratorUptimeSeconds = 0;
        addLog('error', `Failed to load system info: ${error.message}`);
    }
}

// Uptime counter using real orchestrator uptime
function startUptimeCounter() {
    let baseSeconds = orchestratorUptimeSeconds;
    let startTime = Date.now();
    
    console.log('Starting uptime counter with base seconds:', baseSeconds); // Debug log
    
    setInterval(() => {
        const elapsed = Math.floor((Date.now() - startTime) / 1000);
        const totalSeconds = baseSeconds + elapsed;
        
        const days = Math.floor(totalSeconds / 86400);
        const hours = Math.floor((totalSeconds % 86400) / 3600);
        const minutes = Math.floor((totalSeconds % 3600) / 60);
        const secs = totalSeconds % 60;
        
        let uptime;
        if (days > 0) {
            uptime = `${days}d ${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
        } else {
            uptime = `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
        }
        
        document.getElementById('uptime').textContent = uptime;
    }, 1000);
}

// Matrix rain effect
function initializeMatrixEffect() {
    const canvas = document.createElement('canvas');
    canvas.style.position = 'fixed';
    canvas.style.top = '0';
    canvas.style.left = '0';
    canvas.style.width = '100%';
    canvas.style.height = '100%';
    canvas.style.pointerEvents = 'none';
    canvas.style.opacity = '0.05';
    canvas.style.zIndex = '-1';
    document.querySelector('.matrix-rain').appendChild(canvas);
    
    const ctx = canvas.getContext('2d');
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
    
    const columns = Math.floor(canvas.width / 20);
    const drops = Array(columns).fill(1);
    
    function draw() {
        ctx.fillStyle = 'rgba(0, 0, 0, 0.05)';
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        
        ctx.fillStyle = '#00ff41';
        ctx.font = '15px monospace';
        
        for (let i = 0; i < drops.length; i++) {
            const text = String.fromCharCode(Math.random() * 128);
            ctx.fillText(text, i * 20, drops[i] * 20);
            
            if (drops[i] * 20 > canvas.height && Math.random() > 0.975) {
                drops[i] = 0;
            }
            drops[i]++;
        }
    }
    
    setInterval(draw, 33);
    
    window.addEventListener('resize', () => {
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
    });
}

