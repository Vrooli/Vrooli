// System Monitor Dashboard JavaScript
class SystemMonitor {
    constructor() {
        // This gets replaced by server.js at runtime with actual port
        this.apiPort = '8080';
        this.baseUrl = `http://${window.location.hostname}:${this.apiPort}`;
        
        this.isOnline = false;
        this.metrics = {
            cpu: 0,
            memory: 0,
            tcp: 0
        };
        
        this.init();
    }

    init() {
        // Wait for health check before starting metrics polling
        this.checkApiHealth().then(() => {
            this.startMetricsPolling();
            this.loadInvestigations();
        });
        this.setupEventListeners();
        this.addTerminalLine('System Monitor dashboard initialized', 'success');
    }

    setupEventListeners() {
        // Auto-refresh every 30 seconds
        setInterval(() => this.refreshDashboard(), 30000);
        
        // Handle window focus to refresh data
        window.addEventListener('focus', () => this.refreshDashboard());
    }

    async checkApiHealth() {
        try {
            const response = await fetch(`${this.baseUrl}/health`);
            const data = await response.json();
            
            if (data.status === 'healthy') {
                this.setSystemStatus(true);
                // Only log on initial connection, not every health check
                if (!this.isOnline) {
                    this.addTerminalLine('API health check: HEALTHY', 'success');
                }
            } else {
                this.setSystemStatus(false);
                this.addTerminalLine('API health check: UNHEALTHY', 'error');
            }
        } catch (error) {
            this.setSystemStatus(false);
            // Only log errors when status changes
            if (this.isOnline) {
                this.addTerminalLine(`API connection failed: ${error.message}`, 'error');
            }
        }
    }

    setSystemStatus(online) {
        this.isOnline = online;
        const statusDot = document.getElementById('status-dot');
        const statusText = document.getElementById('status-text');
        const systemStatus = document.getElementById('system-status');

        if (online) {
            statusDot.style.background = 'var(--success-green)';
            statusDot.style.boxShadow = '0 0 10px var(--success-green)';
            statusText.textContent = 'ONLINE';
            systemStatus.textContent = 'ONLINE';
            systemStatus.style.color = 'var(--success-green)';
        } else {
            statusDot.style.background = 'var(--danger-red)';
            statusDot.style.boxShadow = '0 0 10px var(--danger-red)';
            statusText.textContent = 'OFFLINE';
            systemStatus.textContent = 'OFFLINE';
            systemStatus.style.color = 'var(--danger-red)';
        }
    }

    async fetchMetrics() {
        if (!this.isOnline) return;

        try {
            const response = await fetch(`${this.baseUrl}/api/metrics/current`);
            const data = await response.json();
            
            this.metrics = {
                cpu: data.cpu_usage || 0,
                memory: data.memory_usage || 0,
                tcp: data.tcp_connections || 0
            };
            
            this.updateMetricsDisplay();
            this.checkAlerts();
            
        } catch (error) {
            // Only log metrics errors occasionally, not on every poll
            if (!this.lastMetricsError || Date.now() - this.lastMetricsError > 60000) {
                this.addTerminalLine(`Metrics fetch error: ${error.message}`, 'error');
                this.lastMetricsError = Date.now();
            }
        }
    }

    updateMetricsDisplay() {
        // Update CPU
        const cpuValue = document.getElementById('cpu-value');
        const cpuFill = document.getElementById('cpu-fill');
        cpuValue.textContent = this.metrics.cpu.toFixed(1);
        cpuFill.style.width = `${Math.min(this.metrics.cpu, 100)}%`;
        
        if (this.metrics.cpu > 80) {
            cpuFill.style.background = 'var(--danger-red)';
        } else if (this.metrics.cpu > 60) {
            cpuFill.style.background = 'var(--warning-orange)';
        } else {
            cpuFill.style.background = 'linear-gradient(90deg, var(--matrix-dark-green), var(--matrix-green))';
        }

        // Update Memory
        const memoryValue = document.getElementById('memory-value');
        const memoryFill = document.getElementById('memory-fill');
        memoryValue.textContent = this.metrics.memory.toFixed(1);
        memoryFill.style.width = `${Math.min(this.metrics.memory, 100)}%`;
        
        if (this.metrics.memory > 85) {
            memoryFill.style.background = 'var(--danger-red)';
        } else if (this.metrics.memory > 70) {
            memoryFill.style.background = 'var(--warning-orange)';
        } else {
            memoryFill.style.background = 'linear-gradient(90deg, var(--matrix-dark-green), var(--matrix-green))';
        }

        // Update TCP Connections
        const tcpValue = document.getElementById('tcp-value');
        const tcpFill = document.getElementById('tcp-fill');
        tcpValue.textContent = this.metrics.tcp;
        // Scale TCP connections to percentage (assume max 200 for visualization)
        const tcpPercent = Math.min((this.metrics.tcp / 200) * 100, 100);
        tcpFill.style.width = `${tcpPercent}%`;
    }

    checkAlerts() {
        const alerts = [];
        
        if (this.metrics.cpu > 80) {
            alerts.push({
                type: 'HIGH_CPU',
                message: `CPU usage at ${this.metrics.cpu.toFixed(1)}%`,
                severity: 'high'
            });
        }
        
        if (this.metrics.memory > 85) {
            alerts.push({
                type: 'HIGH_MEMORY',
                message: `Memory usage at ${this.metrics.memory.toFixed(1)}%`,
                severity: 'high'
            });
        }
        
        if (this.metrics.tcp > 150) {
            alerts.push({
                type: 'HIGH_CONNECTIONS',
                message: `TCP connections at ${this.metrics.tcp}`,
                severity: 'medium'
            });
        }
        
        this.updateAlertsDisplay(alerts);
    }

    updateAlertsDisplay(alerts) {
        const alertCount = document.getElementById('alert-count');
        const alertList = document.getElementById('alert-list');
        
        alertCount.textContent = alerts.length;
        
        if (alerts.length === 0) {
            alertList.innerHTML = '<div class="no-alerts">NO ACTIVE ALERTS</div>';
            return;
        }
        
        alertList.innerHTML = alerts.map(alert => `
            <div class="alert-item ${alert.severity}">
                <div class="alert-header">
                    <span class="alert-type">${alert.type}</span>
                    <span class="alert-severity">${alert.severity.toUpperCase()}</span>
                </div>
                <div class="alert-message">${alert.message}</div>
            </div>
        `).join('');
    }

    async loadInvestigations() {
        if (!this.isOnline) return;

        try {
            const response = await fetch(`${this.baseUrl}/api/investigations/latest`);
            const investigation = await response.json();
            
            this.displayInvestigation(investigation);
            
        } catch (error) {
            // Only log investigation errors occasionally
            if (!this.lastInvestigationError || Date.now() - this.lastInvestigationError > 60000) {
                this.addTerminalLine(`Investigation fetch error: ${error.message}`, 'error');
                this.lastInvestigationError = Date.now();
            }
        }
    }

    displayInvestigation(investigation) {
        const investigationList = document.getElementById('investigation-list');
        
        investigationList.innerHTML = `
            <div class="investigation-item">
                <div class="investigation-header">
                    <span class="investigation-id">${investigation.id}</span>
                    <span class="investigation-status">${investigation.status}</span>
                </div>
                <div class="investigation-findings">${investigation.findings}</div>
            </div>
        `;
    }

    startMetricsPolling() {
        // Initial fetch
        this.fetchMetrics();
        
        // Poll every 5 seconds
        setInterval(() => this.fetchMetrics(), 5000);
    }

    addTerminalLine(message, type = 'info') {
        const terminal = document.getElementById('terminal-content');
        const timestamp = new Date().toLocaleTimeString();
        const line = document.createElement('div');
        line.className = `terminal-line ${type}`;
        line.textContent = `[${timestamp}] ${message}`;
        
        terminal.appendChild(line);
        terminal.scrollTop = terminal.scrollHeight;
        
        // Keep only last 100 lines
        const lines = terminal.querySelectorAll('.terminal-line');
        if (lines.length > 100) {
            lines[0].remove();
        }
    }

    async refreshDashboard() {
        // Silent refresh - no terminal output for auto-refreshes
        await this.checkApiHealth();
        await this.fetchMetrics();
        await this.loadInvestigations();
    }
}

// Global functions for UI interactions
async function generateReport(type) {
    try {
        const response = await fetch(`${monitor.baseUrl}/api/reports/generate`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ type })
        });
        
        const result = await response.json();
        monitor.addTerminalLine(`${type.toUpperCase()} report generated: ${result.report_id}`, 'success');
        
    } catch (error) {
        monitor.addTerminalLine(`Report generation failed: ${error.message}`, 'error');
    }
}

async function simulateAnomaly() {
    try {
        const response = await fetch(`${monitor.baseUrl}/api/test/anomaly/cpu`);
        const result = await response.json();
        
        if (result.anomaly_detected) {
            monitor.addTerminalLine(`Anomaly simulated: ${result.anomaly_type} at ${result.cpu_usage}%`, 'warning');
        }
        
    } catch (error) {
        monitor.addTerminalLine(`Anomaly simulation failed: ${error.message}`, 'error');
    }
}

async function triggerInvestigation() {
    monitor.addTerminalLine('Triggering anomaly investigation...', 'info');
    await monitor.loadInvestigations();
}

function toggleTerminal() {
    const terminal = document.getElementById('terminal');
    terminal.classList.toggle('open');
}

function refreshDashboard() {
    // Manual refresh with terminal output
    monitor.addTerminalLine('Manual refresh triggered...', 'info');
    monitor.refreshDashboard().then(() => {
        monitor.addTerminalLine('Dashboard refreshed', 'success');
    });
}

// Initialize the system monitor when page loads
let monitor;
document.addEventListener('DOMContentLoaded', () => {
    monitor = new SystemMonitor();
    
    // Terminal stays hidden by default - user can toggle it if needed
});