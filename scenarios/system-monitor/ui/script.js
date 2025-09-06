// System Monitor Dashboard JavaScript
console.log('=== SYSTEM MONITOR SCRIPT LOADED ===');
console.log('Document ready state:', document.readyState);
console.log('DOM elements at script load:', {
    statusDot: !!document.getElementById('status-dot'),
    statusText: !!document.getElementById('status-text'), 
    systemStatus: !!document.getElementById('system-status')
});

class SystemMonitor {
    constructor() {
        // Use same-origin URLs - all API calls go through the UI server proxy
        this.baseUrl = `${window.location.protocol}//${window.location.host}`;
        
        this.isOnline = false;
        this.metrics = {
            cpu: 0,
            memory: 0,
            tcp: 0
        };
        
        // Track expanded cards and panels
        this.expandedCards = new Set();
        this.expandedPanels = new Set();
        
        // Detailed metrics cache
        this.detailedMetrics = null;
        this.processMetrics = null;
        this.infrastructureMetrics = null;
        
        this.init();
    }

    init() {
        console.log('DEBUG: Initializing SystemMonitor');
        this.addTerminalLine('DEBUG: SystemMonitor initialization started', 'info');
        this.addTerminalLine(`DEBUG: Base URL is ${this.baseUrl}`, 'info');
        
        // Wait for health check before starting metrics polling
        this.checkApiHealth().then(() => {
            console.log('DEBUG: Health check completed, starting polling');
            this.addTerminalLine('DEBUG: Health check completed, starting metrics polling', 'success');
            this.startMetricsPolling();
            this.startDetailedMetricsPolling();
            this.loadInvestigations();
        }).catch((error) => {
            console.log('DEBUG: Health check promise failed:', error);
            this.addTerminalLine(`DEBUG: Health check promise failed: ${error.message}`, 'error');
        });
        
        this.setupEventListeners();
        this.initializeExpandableCards();
        this.addTerminalLine('System Monitor dashboard initialized', 'success');
        
        // Make terminal visible by default for debugging
        document.getElementById('terminal').classList.add('open');
    }

    setupEventListeners() {
        // Auto-refresh every 30 seconds
        setInterval(() => this.refreshDashboard(), 30000);
        
        // Handle window focus to refresh data
        window.addEventListener('focus', () => this.refreshDashboard());
    }

    async checkApiHealth() {
        console.log('DEBUG: Starting health check, baseUrl:', this.baseUrl);
        console.log('DEBUG: Current isOnline state:', this.isOnline);
        
        try {
            const response = await fetch(`${this.baseUrl}/health`);
            console.log('DEBUG: Fetch response status:', response.status);
            
            const data = await response.json();
            console.log('DEBUG: Response data:', data);
            
            if (data.status === 'healthy') {
                console.log('DEBUG: API is healthy, setting status to online');
                this.setSystemStatus(true);
                // Only log on initial connection, not every health check
                if (!this.isOnline) {
                    this.addTerminalLine('API health check: HEALTHY', 'success');
                    console.log('DEBUG: Added success terminal line');
                } else {
                    console.log('DEBUG: Already online, skipping terminal log');
                }
            } else {
                console.log('DEBUG: API status not healthy:', data.status);
                this.setSystemStatus(false);
                this.addTerminalLine('API health check: UNHEALTHY', 'error');
            }
        } catch (error) {
            console.log('DEBUG: Health check failed with error:', error);
            this.setSystemStatus(false);
            // Only log errors when status changes
            if (this.isOnline) {
                this.addTerminalLine(`API connection failed: ${error.message}`, 'error');
            } else {
                console.log('DEBUG: Already offline, skipping error log');
            }
        }
    }

    setSystemStatus(online) {
        console.log('DEBUG: setSystemStatus called with:', online);
        this.addTerminalLine(`DEBUG: Setting system status to ${online ? 'ONLINE' : 'OFFLINE'}`, 'info');
        
        this.isOnline = online;
        const statusDot = document.getElementById('status-dot');
        const statusText = document.getElementById('status-text');
        const systemStatus = document.getElementById('system-status');

        console.log('DEBUG: DOM elements found:', {
            statusDot: !!statusDot, 
            statusText: !!statusText, 
            systemStatus: !!systemStatus
        });
        
        // Enhanced debugging for statusText element
        if (statusText) {
            console.log('DEBUG: statusText element before update:', statusText.textContent);
            console.log('DEBUG: statusText element classes:', statusText.className);
            console.log('DEBUG: statusText element parent:', statusText.parentElement);
        } else {
            console.error('ERROR: statusText element not found!');
            this.addTerminalLine('ERROR: Header status element not found in DOM', 'error');
        }

        if (online) {
            if (statusDot) {
                statusDot.style.background = 'var(--success-green)';
                statusDot.style.boxShadow = '0 0 10px var(--success-green)';
            }
            if (statusText) {
                statusText.textContent = 'ONLINE';
                console.log('DEBUG: statusText after update:', statusText.textContent);
            }
            if (systemStatus) {
                systemStatus.textContent = 'ONLINE';
                systemStatus.style.color = 'var(--success-green)';
            }
            console.log('DEBUG: Status updated to ONLINE');
        } else {
            if (statusDot) {
                statusDot.style.background = 'var(--danger-red)';
                statusDot.style.boxShadow = '0 0 10px var(--danger-red)';
            }
            if (statusText) {
                statusText.textContent = 'OFFLINE';
            }
            if (systemStatus) {
                systemStatus.textContent = 'OFFLINE';
                systemStatus.style.color = 'var(--danger-red)';
            }
            console.log('DEBUG: Status updated to OFFLINE');
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

    async fetchDetailedMetrics() {
        if (!this.isOnline) return;

        try {
            const response = await fetch(`${this.baseUrl}/api/metrics/detailed`);
            this.detailedMetrics = await response.json();
            this.updateDetailedDisplays();
        } catch (error) {
            if (!this.lastDetailedError || Date.now() - this.lastDetailedError > 60000) {
                this.addTerminalLine(`Detailed metrics fetch error: ${error.message}`, 'error');
                this.lastDetailedError = Date.now();
            }
        }
    }

    async fetchProcessMetrics() {
        if (!this.isOnline) return;

        try {
            const response = await fetch(`${this.baseUrl}/api/metrics/processes`);
            this.processMetrics = await response.json();
            this.updateProcessDisplay();
        } catch (error) {
            console.error('Process metrics fetch error:', error);
        }
    }

    async fetchInfrastructureMetrics() {
        if (!this.isOnline) return;

        try {
            const response = await fetch(`${this.baseUrl}/api/metrics/infrastructure`);
            this.infrastructureMetrics = await response.json();
            this.updateInfrastructureDisplay();
        } catch (error) {
            console.error('Infrastructure metrics fetch error:', error);
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
        let alertCounts = { cpu: 0, memory: 0, network: 0, system: 0 };
        
        if (this.metrics.cpu > 80) {
            alerts.push({
                type: 'HIGH_CPU',
                message: `CPU usage at ${this.metrics.cpu.toFixed(1)}%`,
                severity: 'high',
                card: 'cpu'
            });
            alertCounts.cpu++;
        }
        
        if (this.metrics.memory > 85) {
            alerts.push({
                type: 'HIGH_MEMORY',
                message: `Memory usage at ${this.metrics.memory.toFixed(1)}%`,
                severity: 'high',
                card: 'memory'
            });
            alertCounts.memory++;
        }
        
        if (this.metrics.tcp > 150) {
            alerts.push({
                type: 'HIGH_CONNECTIONS',
                message: `TCP connections at ${this.metrics.tcp}`,
                severity: 'medium',
                card: 'network'
            });
            alertCounts.network++;
        }
        
        // Check for detailed metrics alerts
        if (this.detailedMetrics) {
            // Check for CLOSE_WAIT connections (leak indicator)
            if (this.detailedMetrics.network_details?.tcp_states?.close_wait > 10) {
                alerts.push({
                    type: 'CONNECTION_LEAK',
                    message: `${this.detailedMetrics.network_details.tcp_states.close_wait} CLOSE_WAIT connections detected`,
                    severity: 'high',
                    card: 'network'
                });
                alertCounts.network++;
            }
            
            // Check for high file descriptor usage
            if (this.detailedMetrics.system_details?.file_descriptors?.percent > 80) {
                alerts.push({
                    type: 'HIGH_FD_USAGE',
                    message: `File descriptor usage at ${this.detailedMetrics.system_details.file_descriptors.percent.toFixed(1)}%`,
                    severity: 'medium',
                    card: 'system'
                });
                alertCounts.system++;
            }
        }
        
        this.updateAlertsDisplay(alerts);
        this.updateAlertBadges(alertCounts);
        this.handleSmartExpansion(alerts);
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
        
        let statusClass = '';
        let statusIcon = '';
        
        switch(investigation.status) {
            case 'in_progress':
                statusClass = 'status-progress';
                statusIcon = '🔍';
                break;
            case 'completed':
                statusClass = 'status-completed';
                statusIcon = '✅';
                break;
            case 'failed':
                statusClass = 'status-failed';
                statusIcon = '❌';
                break;
            default:
                statusClass = 'status-pending';
                statusIcon = '⏳';
        }
        
        investigationList.innerHTML = `
            <div class="investigation-item">
                <div class="investigation-header">
                    <span class="investigation-id">${investigation.id}</span>
                    <span class="investigation-status ${statusClass}">${statusIcon} ${investigation.status.toUpperCase()}</span>
                </div>
                <div class="investigation-findings">${investigation.findings}</div>
                <div class="investigation-timestamp">Started: ${new Date(investigation.start_time).toLocaleString()}</div>
            </div>
        `;
    }

    startMetricsPolling() {
        // Initial fetch
        this.fetchMetrics();
        
        // Poll every 5 seconds
        setInterval(() => this.fetchMetrics(), 5000);
    }

    startDetailedMetricsPolling() {
        // Initial fetch
        this.fetchDetailedMetrics();
        this.fetchProcessMetrics();
        this.fetchInfrastructureMetrics();
        
        // Poll detailed metrics every 10 seconds
        setInterval(() => {
            if (this.expandedCards.size > 0 || this.expandedPanels.size > 0) {
                this.fetchDetailedMetrics();
            }
        }, 10000);
        
        // Poll process metrics every 15 seconds when panel is expanded
        setInterval(() => {
            if (this.expandedPanels.has('process')) {
                this.fetchProcessMetrics();
            }
        }, 15000);
        
        // Poll infrastructure metrics every 15 seconds when panel is expanded
        setInterval(() => {
            if (this.expandedPanels.has('infrastructure')) {
                this.fetchInfrastructureMetrics();
            }
        }, 15000);
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

    // Expandable card and panel management
    initializeExpandableCards() {
        // All cards start collapsed
        const detailsElements = document.querySelectorAll('.metric-details');
        detailsElements.forEach(details => {
            details.style.display = 'none';
        });
        
        // All panels start collapsed
        const panelContents = document.querySelectorAll('.panel-content');
        panelContents.forEach(content => {
            content.style.display = 'none';
        });
    }

    toggleCard(cardId) {
        const card = document.getElementById(`${cardId}-card`);
        const details = document.getElementById(`${cardId}-details`);
        const arrow = document.getElementById(`${cardId}-arrow`);
        
        if (!card || !details) {
            console.error('Missing DOM elements for card expansion:', cardId);
            return;
        }
        
        if (this.expandedCards.has(cardId)) {
            // Collapse
            this.expandedCards.delete(cardId);
            card.classList.remove('expanded');
            details.classList.remove('expanded');
            arrow.classList.remove('expanded');
            details.style.display = 'none';
        } else {
            // Expand
            this.expandedCards.add(cardId);
            card.classList.add('expanded');
            details.style.display = 'block';
            // Small delay for smooth animation
            setTimeout(() => {
                details.classList.add('expanded');
                arrow.classList.add('expanded');
            }, 10);
            
            // Fetch fresh detailed data when expanding
            this.fetchDetailedMetrics();
        }
    }

    togglePanel(panelId) {
        const content = document.getElementById(`${panelId}-panel`);
        const arrow = document.getElementById(`${panelId}-panel-arrow`);
        
        if (this.expandedPanels.has(panelId)) {
            // Collapse
            this.expandedPanels.delete(panelId);
            content.classList.remove('expanded');
            arrow.classList.remove('expanded');
            content.style.display = 'none';
        } else {
            // Expand
            this.expandedPanels.add(panelId);
            content.style.display = 'block';
            setTimeout(() => {
                content.classList.add('expanded');
                arrow.classList.add('expanded');
            }, 10);
            
            // Fetch specific data based on panel
            if (panelId === 'process') {
                this.fetchProcessMetrics();
            } else if (panelId === 'infrastructure') {
                this.fetchInfrastructureMetrics();
            }
        }
    }

    updateAlertBadges(alertCounts) {
        const cards = ['cpu', 'memory', 'network', 'system'];
        cards.forEach(card => {
            const badge = document.getElementById(`${card}-alerts`);
            const count = alertCounts[card] || 0;
            
            if (count > 0) {
                badge.textContent = count;
                badge.classList.add('visible');
            } else {
                badge.classList.remove('visible');
            }
        });
    }

    handleSmartExpansion(alerts) {
        // Auto-expand cards with high-severity alerts
        alerts.forEach(alert => {
            if (alert.severity === 'high' && alert.card) {
                if (!this.expandedCards.has(alert.card)) {
                    this.toggleCard(alert.card);
                    this.addTerminalLine(`Auto-expanded ${alert.card} card due to ${alert.type}`, 'warning');
                }
            }
        });
    }

    updateDetailedDisplays() {
        if (!this.detailedMetrics) return;

        // Update CPU details
        if (this.expandedCards.has('cpu')) {
            this.updateCPUDetails();
        }

        // Update Memory details
        if (this.expandedCards.has('memory')) {
            this.updateMemoryDetails();
        }

        // Update Network details
        if (this.expandedCards.has('network')) {
            this.updateNetworkDetails();
        }

        // Update System details
        if (this.expandedCards.has('system')) {
            this.updateSystemDetails();
        }
    }

    updateCPUDetails() {
        const data = this.detailedMetrics.cpu_details;
        if (!data) return;

        // Update process list
        const processList = document.getElementById('cpu-processes');
        processList.innerHTML = data.top_processes.map(proc => `
            <div class="process-item">
                <span class="process-name">${proc.name} (${proc.pid})</span>
                <span class="process-stat">${proc.cpu_percent.toFixed(1)}% CPU</span>
            </div>
        `).join('');

        // Update load average
        document.getElementById('load-average').textContent = 
            data.load_average.map(l => l.toFixed(2)).join(', ');

        // Update context switches
        document.getElementById('context-switches').textContent = 
            `${data.context_switches.toLocaleString()}/sec`;

        // Update goroutines
        document.getElementById('total-goroutines').textContent = data.goroutines || 'N/A';
    }

    updateMemoryDetails() {
        const data = this.detailedMetrics.memory_details;
        if (!data) return;

        // Update process list
        const processList = document.getElementById('memory-processes');
        processList.innerHTML = data.top_processes.map(proc => `
            <div class="process-item">
                <span class="process-name">${proc.name} (${proc.pid})</span>
                <span class="process-stat">${proc.memory_mb.toFixed(0)} MB</span>
            </div>
        `).join('');

        // Update growth patterns
        const growthPatterns = document.getElementById('growth-patterns');
        growthPatterns.innerHTML = data.growth_patterns.map(pattern => `
            <div class="growth-item">
                <span class="process-name">${pattern.process}</span>
                <div class="growth-trend">
                    <span class="process-stat">+${pattern.growth_mb_per_hour.toFixed(1)} MB/hr</span>
                    <span class="risk-${pattern.risk_level}">${pattern.risk_level.toUpperCase()}</span>
                </div>
            </div>
        `).join('');

        // Update swap usage
        document.getElementById('swap-usage').textContent = 
            `${data.swap_usage.percent.toFixed(1)}% (${(data.swap_usage.used / 1024 / 1024 / 1024).toFixed(1)} GB)`;

        // Update disk usage
        document.getElementById('disk-usage').textContent = 
            `${data.disk_usage.percent.toFixed(1)}% (${(data.disk_usage.used / 1024 / 1024 / 1024).toFixed(1)} GB)`;
    }

    updateNetworkDetails() {
        const data = this.detailedMetrics.network_details;
        if (!data) return;

        // Update connection states
        const statesContainer = document.getElementById('connection-states');
        const states = data.tcp_states;
        statesContainer.innerHTML = `
            <div class="connection-item ${states.close_wait > 10 ? 'risk-high' : ''}">
                <span class="process-name">ESTABLISHED:</span>
                <span class="process-stat">${states.established}</span>
            </div>
            <div class="connection-item ${states.time_wait > 50 ? 'risk-medium' : ''}">
                <span class="process-name">TIME_WAIT:</span>
                <span class="process-stat">${states.time_wait}</span>
            </div>
            <div class="connection-item ${states.close_wait > 0 ? 'risk-high' : ''}">
                <span class="process-name">CLOSE_WAIT:</span>
                <span class="process-stat risk-${states.close_wait > 10 ? 'high' : states.close_wait > 0 ? 'medium' : 'low'}">${states.close_wait}</span>
            </div>
            <div class="connection-item">
                <span class="process-name">Others:</span>
                <span class="process-stat">${states.total - states.established - states.time_wait - states.close_wait}</span>
            </div>
        `;

        // Update port usage
        document.getElementById('port-usage').textContent = 
            `${data.port_usage.used}/${data.port_usage.total}`;

        // Update bandwidth
        document.getElementById('bandwidth').textContent = 
            `↗ ${data.network_stats.bandwidth_in_mbps} Mbps ↘ ${data.network_stats.bandwidth_out_mbps} Mbps`;

        // Update connection pools
        const poolsContainer = document.getElementById('connection-pools');
        poolsContainer.innerHTML = data.connection_pools.map(pool => `
            <div class="connection-item ${pool.healthy ? '' : 'risk-high'}">
                <span class="process-name">${pool.name}:</span>
                <span class="process-stat">${pool.active}/${pool.max_size} active (${pool.leak_risk} risk)</span>
            </div>
        `).join('');

        // Update DNS health
        document.getElementById('dns-health').textContent = 
            `${data.network_stats.dns_success_rate.toFixed(1)}% success (${data.network_stats.dns_latency_ms.toFixed(0)}ms)`;
    }

    updateSystemDetails() {
        const data = this.detailedMetrics.system_details;
        if (!data) return;

        // Update file descriptors
        document.getElementById('file-descriptors').textContent = 
            `${data.file_descriptors.used}/${data.file_descriptors.max} (${data.file_descriptors.percent.toFixed(1)}%)`;

        // Update service dependencies
        const servicesList = document.getElementById('service-dependencies');
        servicesList.innerHTML = data.service_dependencies.map(service => `
            <div class="service-item">
                <span class="service-name">${service.name}</span>
                <span class="service-stat risk-${service.status === 'healthy' ? 'low' : 'high'}">
                    ${service.status.toUpperCase()} (${service.latency_ms >= 0 ? service.latency_ms.toFixed(0) + 'ms' : 'timeout'})
                </span>
            </div>
        `).join('');

        // Update certificates
        const certsList = document.getElementById('certificates');
        certsList.innerHTML = data.certificates.map(cert => `
            <div class="service-item">
                <span class="service-name">${cert.domain}</span>
                <span class="service-stat risk-${cert.days_to_expiry < 30 ? 'high' : cert.days_to_expiry < 90 ? 'medium' : 'low'}">
                    ${cert.days_to_expiry} days until expiry
                </span>
            </div>
        `).join('');
    }

    updateProcessDisplay() {
        if (!this.processMetrics || !this.expandedPanels.has('process')) return;

        const data = this.processMetrics.process_health;

        // Update total processes
        document.getElementById('total-processes').textContent = data.total_processes;

        // Update zombie count
        const zombieCount = document.getElementById('zombie-count');
        zombieCount.textContent = data.zombie_processes.length;
        zombieCount.className = `stat-value ${data.zombie_processes.length > 0 ? 'risk-high' : ''}`;

        // Update zombie processes list
        const zombiesList = document.getElementById('zombie-processes');
        zombiesList.innerHTML = data.zombie_processes.length > 0 ? 
            data.zombie_processes.map(proc => `
                <div class="alert-item zombie">
                    <span class="process-name">PID ${proc.pid}: ${proc.name}</span>
                    <span class="process-stat">ZOMBIE</span>
                </div>
            `).join('') : '<div class="process-stat">No zombie processes detected</div>';

        // Update high thread count processes
        const threadsList = document.getElementById('high-thread-processes');
        threadsList.innerHTML = data.high_thread_count.map(proc => `
            <div class="thread-item high">
                <span class="process-name">${proc.name} (${proc.pid})</span>
                <span class="process-stat">${proc.threads} threads</span>
            </div>
        `).join('');

        // Update leak candidates
        const leaksList = document.getElementById('leak-candidates');
        leaksList.innerHTML = data.leak_candidates.map(proc => `
            <div class="leak-item critical">
                <span class="process-name">${proc.name} (${proc.pid})</span>
                <span class="process-stat">${proc.status.replace('_', ' ').toUpperCase()}</span>
            </div>
        `).join('');

        // Update resource matrix
        const matrixContainer = document.getElementById('resource-matrix');
        matrixContainer.innerHTML = `
            <div class="matrix-header">
                <div>Process</div>
                <div>PID</div>
                <div>CPU %</div>
                <div>Memory MB</div>
                <div>Threads</div>
                <div>File Descriptors</div>
            </div>
            ${this.processMetrics.resource_matrix.map(proc => `
                <div class="matrix-row">
                    <div class="matrix-cell">${proc.name}</div>
                    <div class="matrix-cell">${proc.pid}</div>
                    <div class="matrix-cell">${proc.cpu_percent.toFixed(1)}</div>
                    <div class="matrix-cell">${proc.memory_mb.toFixed(0)}</div>
                    <div class="matrix-cell">${proc.threads}</div>
                    <div class="matrix-cell">${proc.fds}</div>
                </div>
            `).join('')}
        `;
    }

    updateInfrastructureDisplay() {
        if (!this.infrastructureMetrics || !this.expandedPanels.has('infrastructure')) return;

        const data = this.infrastructureMetrics;

        // Update database pools
        const dbPoolsList = document.getElementById('database-pools');
        dbPoolsList.innerHTML = data.database_pools.map(pool => `
            <div class="pool-item ${pool.healthy ? 'healthy' : 'unhealthy'}">
                <span class="process-name">${pool.name}</span>
                <span class="process-stat">${pool.active}/${pool.max_size} active, ${pool.idle} idle</span>
            </div>
        `).join('');

        // Update HTTP pools
        const httpPoolsList = document.getElementById('http-pools');
        httpPoolsList.innerHTML = data.http_client_pools.map(pool => `
            <div class="pool-item ${pool.healthy ? 'healthy' : 'unhealthy'}">
                <span class="process-name">${pool.name}</span>
                <span class="process-stat">
                    ${pool.active}/${pool.max_size} active (${pool.leak_risk} leak risk)
                </span>
            </div>
        `).join('');

        // Update message queue stats
        document.getElementById('redis-subscribers').textContent = data.message_queues.redis_pubsub.subscribers;
        document.getElementById('redis-channels').textContent = data.message_queues.redis_pubsub.channels;
        document.getElementById('jobs-pending').textContent = data.message_queues.background_jobs.pending;
        document.getElementById('jobs-active').textContent = data.message_queues.background_jobs.active;
        document.getElementById('jobs-failed').textContent = data.message_queues.background_jobs.failed;

        // Update storage I/O stats
        document.getElementById('disk-queue-depth').textContent = data.storage_io.disk_queue_depth.toFixed(2);
        document.getElementById('io-wait').textContent = `${data.storage_io.io_wait_percent.toFixed(1)}%`;
        document.getElementById('read-rate').textContent = `${data.storage_io.read_mb_per_sec.toFixed(1)} MB/s`;
        document.getElementById('write-rate').textContent = `${data.storage_io.write_mb_per_sec.toFixed(1)} MB/s`;
    }

    async refreshDashboard() {
        // Silent refresh - no terminal output for auto-refreshes
        await this.checkApiHealth();
        await this.fetchMetrics();
        await this.fetchDetailedMetrics();
        if (this.expandedPanels.has('process')) {
            await this.fetchProcessMetrics();
        }
        if (this.expandedPanels.has('infrastructure')) {
            await this.fetchInfrastructureMetrics();
        }
        await this.loadInvestigations();
    }
}

// Global functions for expandable UI
function toggleCard(cardId) {
    console.log('toggleCard called for:', cardId);
    if (typeof monitor !== 'undefined') {
        monitor.toggleCard(cardId);
    } else {
        console.error('Monitor object not available');
    }
}

function togglePanel(panelId) {
    monitor.togglePanel(panelId);
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
    const button = document.querySelector('.btn-action');
    const investigationList = document.getElementById('investigation-list');
    
    // Disable button and show loading
    button.disabled = true;
    button.textContent = 'RUNNING INVESTIGATION...';
    button.style.opacity = '0.6';
    
    // Show loading in investigation panel
    investigationList.innerHTML = '<div class="investigation-loading">🔍 Claude Code is analyzing system metrics and logs...</div>';
    
    monitor.addTerminalLine('Triggering AI-powered anomaly investigation...', 'info');
    
    try {
        // Call the new trigger endpoint
        const response = await fetch(`${monitor.baseUrl}/api/investigations/trigger`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const result = await response.json();
        monitor.addTerminalLine(`Investigation started: ${result.investigation_id}`, 'success');
        
        // Start polling for results
        pollInvestigationResults();
        
    } catch (error) {
        monitor.addTerminalLine(`Failed to start investigation: ${error.message}`, 'error');
        investigationList.innerHTML = '<div class="investigation-error">❌ Failed to start investigation</div>';
        
        // Re-enable button
        button.disabled = false;
        button.textContent = 'RUN ANOMALY CHECK';
        button.style.opacity = '1';
    }
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

// Polling function for investigation results
function pollInvestigationResults() {
    const pollInterval = setInterval(async () => {
        try {
            const response = await fetch(`${monitor.baseUrl}/api/investigations/latest`);
            const investigation = await response.json();
            
            monitor.displayInvestigation(investigation);
            
            // If investigation is complete, stop polling and re-enable button
            if (investigation.status === 'completed' || investigation.status === 'failed') {
                clearInterval(pollInterval);
                
                const button = document.querySelector('.btn-action');
                button.disabled = false;
                button.textContent = 'RUN ANOMALY CHECK';
                button.style.opacity = '1';
                
                if (investigation.status === 'completed') {
                    monitor.addTerminalLine('Investigation completed successfully', 'success');
                } else {
                    monitor.addTerminalLine('Investigation failed', 'error');
                }
            }
        } catch (error) {
            console.error('Polling error:', error);
        }
    }, 2000); // Poll every 2 seconds
}

// Initialize the system monitor when page loads
let monitor;
document.addEventListener('DOMContentLoaded', () => {
    console.log('=== DOM CONTENT LOADED ===');
    console.log('Document ready state:', document.readyState);
    console.log('DOM elements at DOMContentLoaded:', {
        statusDot: !!document.getElementById('status-dot'),
        statusText: !!document.getElementById('status-text'), 
        systemStatus: !!document.getElementById('system-status')
    });
    
    console.log('Creating SystemMonitor instance...');
    monitor = new SystemMonitor();
    console.log('SystemMonitor instance created:', !!monitor);
    
    // Terminal stays hidden by default - user can toggle it if needed
});