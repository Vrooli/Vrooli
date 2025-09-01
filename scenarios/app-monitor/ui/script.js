// App Monitor Dashboard - Matrix Theme
// Real-time monitoring and control for Vrooli generated applications

const API_BASE = `${window.location.protocol}//${window.location.host}`;

let ws = null;
let currentView = 'apps';
let apps = [];
let resources = [];
let selectedApp = null;

// Initialize dashboard
document.addEventListener('DOMContentLoaded', () => {
    initializeWebSocket();
    initializeEventListeners();
    loadInitialData();
    startUptimeCounter();
    initializeMatrixEffect();
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
function initializeEventListeners() {
    // Navigation menu
    document.querySelectorAll('.menu-item').forEach(item => {
        item.addEventListener('click', () => {
            const view = item.dataset.view;
            switchView(view);
        });
    });
    
    // Quick actions
    document.getElementById('restart-all').addEventListener('click', restartAllApps);
    document.getElementById('stop-all').addEventListener('click', stopAllApps);
    document.getElementById('health-check').addEventListener('click', performHealthCheck);
    
    // App controls
    document.getElementById('refresh-apps').addEventListener('click', loadApps);
    document.getElementById('app-search').addEventListener('input', filterApps);
    
    // Log controls
    document.getElementById('log-level').addEventListener('change', filterLogs);
    document.getElementById('clear-logs').addEventListener('click', clearLogs);
    
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
        addLog('info', `Loaded ${apps.length} applications`);
    } catch (error) {
        console.error('Failed to load apps:', error);
        addLog('error', `Failed to load applications: ${error.message}`);
        // Use mock data for demonstration
        apps = getMockApps();
        renderApps();
        updateAppCount(apps.length);
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
        // Use mock data for demonstration
        resources = getMockResources();
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
        // Use mock data for demonstration
        updateMetrics(getMockMetrics());
    }
}

// Render apps grid
function renderApps() {
    const container = document.getElementById('apps-container');
    container.innerHTML = '';
    
    apps.forEach(app => {
        const card = createAppCard(app);
        container.appendChild(card);
    });
}

// Create app card element
function createAppCard(app) {
    const card = document.createElement('div');
    card.className = 'app-card';
    card.innerHTML = `
        <div class="app-header">
            <span class="app-name">${app.name}</span>
            <span class="app-status ${app.status}">${app.status}</span>
        </div>
        <div class="app-metrics">
            <div class="app-metric">
                <div class="metric-label">CPU</div>
                <div class="metric-data">${app.cpu || '0'}%</div>
            </div>
            <div class="app-metric">
                <div class="metric-label">Memory</div>
                <div class="metric-data">${app.memory || '0'} MB</div>
            </div>
            <div class="app-metric">
                <div class="metric-label">Uptime</div>
                <div class="metric-data">${app.uptime || '0h'}</div>
            </div>
            <div class="app-metric">
                <div class="metric-label">Port</div>
                <div class="metric-data">${app.port || 'N/A'}</div>
            </div>
        </div>
        <div class="app-actions">
            <button class="app-btn" onclick="controlAppDirect('${app.id}', 'start')">START</button>
            <button class="app-btn" onclick="controlAppDirect('${app.id}', 'stop')">STOP</button>
            <button class="app-btn" onclick="showAppDetails('${app.id}')">DETAILS</button>
        </div>
    `;
    return card;
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
        'redis': '⚡',
        'n8n': '🔄',
        'ollama': '🤖',
        'windmill': '🌀',
        'node-red': '🔴',
        'qdrant': '🔍',
        'minio': '📦'
    };
    return icons[type] || '📊';
}

// Switch view
function switchView(view) {
    // Update menu
    document.querySelectorAll('.menu-item').forEach(item => {
        item.classList.remove('active');
    });
    document.querySelector(`[data-view="${view}"]`).classList.add('active');
    
    // Update panels
    document.querySelectorAll('.view-panel').forEach(panel => {
        panel.classList.remove('active');
    });
    document.getElementById(`${view}-view`).classList.add('active');
    
    currentView = view;
    
    // Load view-specific data
    switch (view) {
        case 'metrics':
            initializeCharts();
            break;
        case 'terminal':
            document.getElementById('terminal-input').focus();
            break;
    }
}

// Filter apps
function filterApps() {
    const searchTerm = document.getElementById('app-search').value.toLowerCase();
    const cards = document.querySelectorAll('.app-card');
    
    cards.forEach(card => {
        const appName = card.querySelector('.app-name').textContent.toLowerCase();
        card.style.display = appName.includes(searchTerm) ? 'block' : 'none';
    });
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
        loadApps(); // Reload apps to show updated status
    } catch (error) {
        console.error(`Failed to ${action} app:`, error);
        addLog('error', `Failed to ${action} app: ${error.message}`);
    }
};

// Show app details
window.showAppDetails = function(appId) {
    const app = apps.find(a => a.id === appId);
    if (!app) return;
    
    selectedApp = app;
    
    document.getElementById('modal-app-name').textContent = app.name.toUpperCase();
    document.getElementById('modal-body').innerHTML = `
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
                <strong>Status:</strong> ${app.status}
            </div>
            <div>
                <strong>Port:</strong> ${app.port || 'N/A'}
            </div>
            <div>
                <strong>CPU Usage:</strong> ${app.cpu || 0}%
            </div>
            <div>
                <strong>Memory:</strong> ${app.memory || 0} MB
            </div>
            <div>
                <strong>Uptime:</strong> ${app.uptime || '0h'}
            </div>
            <div>
                <strong>Version:</strong> ${app.version || '1.0.0'}
            </div>
            <div style="grid-column: 1 / -1;">
                <strong>Description:</strong> ${app.description || 'No description available'}
            </div>
            <div style="grid-column: 1 / -1;">
                <strong>Container ID:</strong> ${app.containerId || 'N/A'}
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
function viewAppLogs() {
    if (!selectedApp) return;
    
    closeModal();
    switchView('logs');
    
    // Filter logs for selected app
    document.getElementById('log-level').value = 'all';
    addLog('info', `Viewing logs for ${selectedApp.name}`);
    
    // Load app-specific logs
    loadAppLogs(selectedApp.id);
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

// Initialize charts (simplified for demo)
function initializeCharts() {
    // This would typically use a charting library like Chart.js
    // For now, we'll create simple canvas visualizations
    drawChart('cpu-chart', 45);
    drawChart('memory-chart', 67);
    drawChart('network-chart', 23);
    drawChart('disk-chart', 78);
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
    drawChart('memory-chart', (metrics.memory / 1000) * 100 || 0);
    drawChart('network-chart', Math.min(metrics.network || 0, 100));
    drawChart('disk-chart', metrics.disk || 0);
}

// Uptime counter
function startUptimeCounter() {
    let seconds = 0;
    
    setInterval(() => {
        seconds++;
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;
        
        const uptime = `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
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

// Mock data functions for demonstration
function getMockApps() {
    return [
        { id: 'app1', name: 'research-assistant', status: 'running', cpu: 23, memory: 512, uptime: '2h 34m', port: 8091 },
        { id: 'app2', name: 'campaign-studio', status: 'running', cpu: 45, memory: 768, uptime: '5h 12m', port: 8092 },
        { id: 'app3', name: 'image-pipeline', status: 'stopped', cpu: 0, memory: 0, uptime: '0h', port: 8093 },
        { id: 'app4', name: 'prompt-manager', status: 'running', cpu: 12, memory: 256, uptime: '1h 45m', port: 8094 },
        { id: 'app5', name: 'task-planner', status: 'error', cpu: 78, memory: 1024, uptime: '0h 23m', port: 8095 }
    ];
}

function getMockResources() {
    return [
        { id: 'res1', name: 'PostgreSQL', type: 'postgres', status: 'online' },
        { id: 'res2', name: 'Redis', type: 'redis', status: 'online' },
        { id: 'res3', name: 'n8n', type: 'n8n', status: 'online' },
        { id: 'res4', name: 'Ollama', type: 'ollama', status: 'offline' },
        { id: 'res5', name: 'Windmill', type: 'windmill', status: 'online' },
        { id: 'res6', name: 'Node-RED', type: 'node-red', status: 'online' },
        { id: 'res7', name: 'Qdrant', type: 'qdrant', status: 'online' },
        { id: 'res8', name: 'MinIO', type: 'minio', status: 'offline' }
    ];
}

function getMockMetrics() {
    return {
        cpu: Math.floor(Math.random() * 100),
        memory: Math.floor(Math.random() * 2048),
        network: Math.floor(Math.random() * 1000),
        disk: Math.floor(Math.random() * 100)
    };
}