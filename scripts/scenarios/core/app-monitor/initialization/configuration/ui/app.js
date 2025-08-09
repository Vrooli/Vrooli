class AppMonitor {
    constructor() {
        this.apiUrl = 'http://localhost:8081';
        this.wsUrl = 'ws://localhost:3002/ws';
        this.ws = null;
        this.apps = [];
        this.selectedApp = null;
        this.alerts = [];
        
        this.init();
    }

    async init() {
        console.log('🚀 Initializing App Monitor Dashboard');
        
        // Initialize WebSocket connection
        this.connectWebSocket();
        
        // Load initial data
        await this.loadApps();
        
        // Set up periodic refresh
        setInterval(() => this.loadApps(), 30000);
        
        // Set up event listeners
        this.setupEventListeners();
        
        console.log('✅ App Monitor initialized');
    }

    connectWebSocket() {
        console.log('🔌 Connecting to WebSocket...');
        
        this.ws = new WebSocket(this.wsUrl);
        
        this.ws.onopen = () => {
            console.log('✅ WebSocket connected');
            this.updateConnectionStatus('connected');
            
            // Subscribe to events
            this.ws.send(JSON.stringify({
                type: 'subscribe',
                data: {
                    channels: ['app-events', 'app-alerts', 'resource-alerts', 'notifications']
                }
            }));
        };
        
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
        };
        
        this.ws.onclose = () => {
            console.log('❌ WebSocket disconnected');
            this.updateConnectionStatus('disconnected');
            
            // Attempt to reconnect after 5 seconds
            setTimeout(() => this.connectWebSocket(), 5000);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.updateConnectionStatus('disconnected');
        };
    }

    handleWebSocketMessage(message) {
        console.log('📨 WebSocket message:', message);
        
        switch (message.type) {
            case 'app-event':
                this.handleAppEvent(message.data);
                break;
            case 'app-alert':
            case 'resource-alert':
                this.handleAlert(message.data);
                break;
            case 'notification':
                this.showNotification(message.data);
                break;
            case 'system-status':
                this.updateSystemStatus(message.data);
                break;
        }
    }

    handleAppEvent(event) {
        console.log('App event:', event);
        
        // Refresh apps list
        this.loadApps();
        
        // Show notification
        this.showNotification({
            type: 'info',
            message: `App ${event.app_name} ${event.type.replace('app_', '')}`
        });
    }

    handleAlert(alert) {
        console.log('Alert:', alert);
        
        this.alerts.unshift({
            id: Date.now(),
            type: alert.type,
            message: alert.message,
            timestamp: new Date().toISOString()
        });
        
        // Keep only last 50 alerts
        if (this.alerts.length > 50) {
            this.alerts = this.alerts.slice(0, 50);
        }
        
        this.renderAlerts();
        this.showNotification(alert);
    }

    showNotification(notification) {
        // Simple toast notification
        const toast = document.createElement('div');
        toast.className = `toast toast-${notification.type}`;
        toast.textContent = notification.message;
        
        // Add toast styles if not exists
        if (!document.querySelector('style[data-toast]')) {
            const style = document.createElement('style');
            style.setAttribute('data-toast', 'true');
            style.textContent = `
                .toast {
                    position: fixed;
                    top: 20px;
                    right: 20px;
                    padding: 1rem 1.5rem;
                    background: white;
                    border-radius: 8px;
                    box-shadow: 0 4px 12px rgba(0,0,0,0.2);
                    z-index: 1001;
                    animation: slideIn 0.3s ease;
                }
                .toast-error { border-left: 4px solid #f56565; }
                .toast-warning { border-left: 4px solid #fbb344; }
                .toast-info { border-left: 4px solid #4299e1; }
                @keyframes slideIn { from { transform: translateX(100%); } to { transform: translateX(0); } }
            `;
            document.head.appendChild(style);
        }
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.remove();
        }, 5000);
    }

    updateConnectionStatus(status) {
        const statusEl = document.getElementById('connection-status');
        statusEl.className = `connection-status ${status}`;
        
        const statusText = {
            connected: 'Connected',
            disconnected: 'Disconnected',
            connecting: 'Connecting...'
        };
        
        statusEl.innerHTML = `<i class="fas fa-circle"></i> ${statusText[status]}`;
    }

    async loadApps() {
        try {
            console.log('📊 Loading apps...');
            const response = await fetch(`${this.apiUrl}/api/apps`);
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            this.apps = await response.json();
            console.log(`✅ Loaded ${this.apps.length} apps`);
            
            this.renderApps();
            this.updateStats();
            this.updateLogFilter();
            
        } catch (error) {
            console.error('❌ Failed to load apps:', error);
            this.showNotification({
                type: 'error',
                message: 'Failed to load apps: ' + error.message
            });
        }
    }

    renderApps() {
        const grid = document.getElementById('apps-grid');
        
        if (this.apps.length === 0) {
            grid.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-desktop" style="font-size: 3rem; color: #cbd5e0; margin-bottom: 1rem;"></i>
                    <p style="color: #64748b;">No applications found</p>
                </div>
            `;
            return;
        }
        
        grid.innerHTML = this.apps.map(app => `
            <div class="app-card" onclick="showAppDetails('${app.id}')">
                <div class="app-card-header">
                    <div class="app-name">${app.name}</div>
                    <div class="app-status ${app.current_status || 'unknown'}">${app.current_status || 'unknown'}</div>
                </div>
                <div class="app-metrics">
                    <div class="metric">
                        <div class="metric-value">${app.cpu_usage ? app.cpu_usage.toFixed(1) + '%' : 'N/A'}</div>
                        <div class="metric-label">CPU</div>
                    </div>
                    <div class="metric">
                        <div class="metric-value">${app.memory_usage ? app.memory_usage.toFixed(1) + '%' : 'N/A'}</div>
                        <div class="metric-label">Memory</div>
                    </div>
                </div>
                <div style="margin-top: 1rem; font-size: 0.8rem; color: #64748b;">
                    Last update: ${app.last_update ? new Date(app.last_update).toLocaleString() : 'Never'}
                </div>
            </div>
        `).join('');
    }

    updateStats() {
        const total = this.apps.length;
        const running = this.apps.filter(app => app.current_status === 'running').length;
        const stopped = this.apps.filter(app => app.current_status === 'stopped').length;
        
        document.getElementById('total-apps').textContent = total;
        document.getElementById('running-apps').textContent = running;
        document.getElementById('stopped-apps').textContent = stopped;
    }

    updateLogFilter() {
        const select = document.getElementById('log-app-filter');
        select.innerHTML = '<option value="">All Apps</option>' +
            this.apps.map(app => `<option value="${app.id}">${app.name}</option>`).join('');
    }

    async showAppDetails(appId) {
        try {
            const response = await fetch(`${this.apiUrl}/api/apps/${appId}`);
            if (!response.ok) throw new Error('Failed to load app details');
            
            const app = await response.json();
            this.selectedApp = app;
            
            const modal = document.getElementById('app-modal');
            const modalName = document.getElementById('modal-app-name');
            const modalBody = document.getElementById('modal-body');
            
            modalName.textContent = app.name;
            modalBody.innerHTML = `
                <div class="app-details">
                    <div class="detail-section">
                        <h4>Basic Information</h4>
                        <p><strong>Status:</strong> <span class="app-status ${app.current_status}">${app.current_status}</span></p>
                        <p><strong>Scenario:</strong> ${app.scenario_name}</p>
                        <p><strong>Path:</strong> ${app.path}</p>
                        <p><strong>Created:</strong> ${new Date(app.created_at).toLocaleString()}</p>
                    </div>
                    <div class="detail-section">
                        <h4>Current Metrics</h4>
                        <p><strong>CPU Usage:</strong> ${app.cpu_usage ? app.cpu_usage.toFixed(2) + '%' : 'N/A'}</p>
                        <p><strong>Memory Usage:</strong> ${app.memory_usage ? app.memory_usage.toFixed(2) + '%' : 'N/A'}</p>
                        <p><strong>Network In:</strong> ${app.network_in ? formatBytes(app.network_in) : 'N/A'}</p>
                        <p><strong>Network Out:</strong> ${app.network_out ? formatBytes(app.network_out) : 'N/A'}</p>
                    </div>
                </div>
            `;
            
            // Update control buttons
            const startBtn = document.getElementById('start-btn');
            const stopBtn = document.getElementById('stop-btn');
            
            if (app.current_status === 'running') {
                startBtn.style.display = 'none';
                stopBtn.style.display = 'flex';
            } else {
                startBtn.style.display = 'flex';
                stopBtn.style.display = 'none';
            }
            
            modal.classList.add('active');
            
        } catch (error) {
            console.error('Failed to load app details:', error);
            this.showNotification({
                type: 'error',
                message: 'Failed to load app details'
            });
        }
    }

    async controlApp(action) {
        if (!this.selectedApp) return;
        
        try {
            const response = await fetch(`${this.apiUrl}/api/apps/${this.selectedApp.id}/${action}`, {
                method: 'POST'
            });
            
            if (!response.ok) throw new Error(`Failed to ${action} app`);
            
            const result = await response.json();
            this.showNotification({
                type: 'info',
                message: result.message
            });
            
            // Close modal and refresh
            this.closeModal();
            setTimeout(() => this.loadApps(), 1000);
            
        } catch (error) {
            console.error(`Failed to ${action} app:`, error);
            this.showNotification({
                type: 'error',
                message: `Failed to ${action} app`
            });
        }
    }

    closeModal() {
        const modal = document.getElementById('app-modal');
        modal.classList.remove('active');
        this.selectedApp = null;
    }

    async loadLogs(appId = null) {
        try {
            const url = appId ? 
                `${this.apiUrl}/api/apps/${appId}/logs?limit=50` :
                `${this.apiUrl}/api/logs?limit=50`;
            
            const response = await fetch(url);
            if (!response.ok) throw new Error('Failed to load logs');
            
            const logs = await response.json();
            this.renderLogs(logs);
            
        } catch (error) {
            console.error('Failed to load logs:', error);
            this.showNotification({
                type: 'error',
                message: 'Failed to load logs'
            });
        }
    }

    renderLogs(logs) {
        const container = document.getElementById('logs-container');
        
        if (logs.length === 0) {
            container.innerHTML = '<div style="padding: 2rem; text-align: center; color: #64748b;">No logs found</div>';
            return;
        }
        
        container.innerHTML = logs.map(log => `
            <div class="log-entry">
                <span class="log-timestamp">${new Date(log.timestamp).toLocaleString()}</span>
                <span class="log-level ${log.level}">${log.level}</span>
                <span class="log-message">${log.message}</span>
            </div>
        `).join('');
    }

    renderAlerts() {
        const container = document.getElementById('alerts-container');
        
        if (this.alerts.length === 0) {
            container.innerHTML = '<div style="padding: 2rem; text-align: center; color: #64748b;">No alerts</div>';
            return;
        }
        
        container.innerHTML = this.alerts.map(alert => `
            <div class="alert ${alert.type}">
                <div class="alert-header">
                    <div class="alert-title">${alert.type.toUpperCase()}</div>
                    <div class="alert-time">${new Date(alert.timestamp).toLocaleString()}</div>
                </div>
                <div class="alert-message">${alert.message}</div>
            </div>
        `).join('');
    }

    setupEventListeners() {
        // Modal close on background click
        document.getElementById('app-modal').addEventListener('click', (e) => {
            if (e.target.id === 'app-modal') {
                this.closeModal();
            }
        });
        
        // Escape key to close modal
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeModal();
            }
        });
    }
}

// Utility functions
function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Global functions for HTML event handlers
function showTab(tabName) {
    // Hide all tabs
    document.querySelectorAll('.tab-content').forEach(tab => {
        tab.classList.remove('active');
    });
    
    // Remove active from all buttons
    document.querySelectorAll('.tab-button').forEach(btn => {
        btn.classList.remove('active');
    });
    
    // Show selected tab
    document.getElementById(`${tabName}-tab`).classList.add('active');
    
    // Add active to clicked button
    event.target.classList.add('active');
    
    // Load data if needed
    if (tabName === 'logs') {
        appMonitor.loadLogs();
    }
}

function refreshApps() {
    appMonitor.loadApps();
}

function showAppDetails(appId) {
    appMonitor.showAppDetails(appId);
}

function controlApp(action) {
    appMonitor.controlApp(action);
}

function closeModal() {
    appMonitor.closeModal();
}

function filterLogs() {
    const select = document.getElementById('log-app-filter');
    const appId = select.value;
    appMonitor.loadLogs(appId || null);
}

// Initialize app when DOM is loaded
let appMonitor;
document.addEventListener('DOMContentLoaded', () => {
    appMonitor = new AppMonitor();
});