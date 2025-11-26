// Agent Details Page JavaScript
// Individual agent detailed view

let currentAgent = null;
let agentId = null;

document.addEventListener('DOMContentLoaded', async () => {
    agentId = getAgentIdFromUrl();
    if (!agentId) {
        showError('No agent ID specified');
        return;
    }
    
    setupTabNavigation();
    await loadAgentDetails();
    setupActionButtons();
    setupEnvironmentSearch();
    startAutoRefresh();
    startLiveUpdates();
});

function getAgentIdFromUrl() {
    const params = new URLSearchParams(window.location.search);
    return params.get('id');
}

async function loadAgentDetails() {
    try {
        currentAgent = await window.agentAPI.fetchAgentDetails(agentId);
        renderAgentDetails();
        await loadQuickMetrics();
    } catch (error) {
        console.error('Failed to load agent details:', error);
        showError('Failed to load agent details');
    }
}

function renderAgentDetails() {
    if (!currentAgent) return;
    
    // Update title
    document.getElementById('agentTitle').innerHTML = `
        <i data-lucide="bot"></i>
        ${currentAgent.name}
    `;
    
    // Overview Tab
    // Update status section
    const statusBadge = document.getElementById('agentStatus');
    statusBadge.className = `status-badge ${currentAgent.status}`;
    statusBadge.innerHTML = `
        <i data-lucide="${getStatusIcon(currentAgent.status)}"></i>
        ${currentAgent.status}
    `;
    
    document.getElementById('agentType').textContent = currentAgent.type.toUpperCase();
    document.getElementById('agentPid').textContent = currentAgent.pid || 'N/A';
    document.getElementById('agentUptime').textContent = currentAgent.uptime || 'N/A';
    
    // Update health score
    const healthScore = calculateHealthScore(currentAgent);
    const healthScoreEl = document.getElementById('healthScore');
    const healthBarEl = document.getElementById('healthBar');
    if (healthScoreEl) {
        healthScoreEl.textContent = `${healthScore}%`;
    }
    if (healthBarEl) {
        healthBarEl.style.width = `${healthScore}%`;
        healthBarEl.className = `health-fill ${getHealthClass(healthScore)}`;
    }
    
    // Update information section
    document.getElementById('agentId').textContent = currentAgent.id;
    
    // Extract resource from agent ID (format: "resource:agent-id")
    const resourceName = currentAgent.id.split(':')[0] || currentAgent.type;
    const resourceEl = document.getElementById('agentResource');
    if (resourceEl) {
        resourceEl.textContent = resourceName.toUpperCase();
    }
    
    const startTimeEl = document.getElementById('agentStartTime');
    if (startTimeEl) {
        startTimeEl.textContent = formatDateTime(currentAgent.start_time || currentAgent.StartTime);
    }
    
    const lastSeenEl = document.getElementById('agentLastSeen');
    if (lastSeenEl) {
        lastSeenEl.textContent = formatDateTime(currentAgent.last_seen);
    }
    
    const versionEl = document.getElementById('agentVersion');
    if (versionEl) {
        versionEl.textContent = currentAgent.version || '1.0.0';
    }
    
    // Set API endpoint
    const port = currentAgent.port || extractPortFromCommand(currentAgent.command) || 'N/A';
    const endpointEl = document.getElementById('agentEndpoint');
    if (endpointEl) {
        const builder = typeof window.buildAgentDashboardApiUrl === 'function' ? window.buildAgentDashboardApiUrl : null;
        const derivedUrl = builder ? builder(`/agents/${encodeURIComponent(currentAgent.id)}`) : (port !== 'N/A' ? `http://localhost:${port}` : null);
        endpointEl.textContent = derivedUrl || 'N/A';
    }
    
    // Update capabilities
    const capabilitiesDiv = document.getElementById('agentCapabilities');
    if (capabilitiesDiv) {
        if (currentAgent.capabilities && currentAgent.capabilities.length > 0) {
            capabilitiesDiv.innerHTML = currentAgent.capabilities.map(cap => 
                `<span class="capability-badge"><i data-lucide="check"></i>${cap}</span>`
            ).join('');
            const capCountEl = document.getElementById('capabilityCount');
            if (capCountEl) capCountEl.textContent = currentAgent.capabilities.length;
        } else {
            capabilitiesDiv.innerHTML = '<span class="no-data">No capabilities defined</span>';
            const capCountEl = document.getElementById('capabilityCount');
            if (capCountEl) capCountEl.textContent = '0';
        }
    }
    
    // Update integration count
    const integrations = currentAgent.integrations || [];
    const intCountEl = document.getElementById('integrationCount');
    if (intCountEl) intCountEl.textContent = integrations.length;
    
    // Update command and execution details
    const cmdEl = document.getElementById('agentCommand');
    if (cmdEl) cmdEl.textContent = currentAgent.command || 'N/A';
    
    const workDirEl = document.getElementById('agentWorkDir');
    if (workDirEl) workDirEl.textContent = currentAgent.working_directory || currentAgent.cwd || '/app';
    
    const userEl = document.getElementById('agentUser');
    if (userEl) userEl.textContent = currentAgent.user || 'agent';
    
    // Update performance overview
    updatePerformanceOverview();
    
    // System Tab
    renderSystemInformation();
    
    // Network Tab
    renderNetworkInformation();
    
    // Environment Tab
    renderEnvironmentVariables();
    
    // Update action buttons
    updateActionButtons();
    
    // Update view logs buttons
    const viewLogsBtn = document.getElementById('viewLogsBtn');
    if (viewLogsBtn) {
        viewLogsBtn.href = `/logs.html?agent=${currentAgent.id}`;
    }
    const viewLogsBtn2 = document.getElementById('viewLogsBtn2');
    if (viewLogsBtn2) {
        viewLogsBtn2.href = `/logs.html?agent=${currentAgent.id}`;
    }
    
    // Reinitialize icons
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
}

async function loadQuickMetrics() {
    try {
        const metrics = currentAgent.metrics || await window.agentAPI.fetchAgentMetrics(agentId);
        
        // Update metrics in all tabs
        const cpuUsage = metrics.cpu_percent || metrics.cpu_usage || 0;
        const memoryMb = metrics.memory_mb || 0;
        
        document.getElementById('metricCpu').textContent = 
            cpuUsage ? `${Number(cpuUsage).toFixed(1)}%` : 'N/A';
        document.getElementById('metricMemory').textContent = 
            memoryMb ? `${Number(memoryMb).toFixed(0)} MB` : 'N/A';
        document.getElementById('metricTasks').textContent = 
            metrics.tasks_completed || '0';
        document.getElementById('metricSuccess').textContent = 
            metrics.success_rate ? `${Number(metrics.success_rate || 100).toFixed(1)}%` : '100%';
        document.getElementById('metricErrors').textContent = 
            metrics.error_rate ? `${Number(metrics.error_rate).toFixed(1)}%` : '0%';
        document.getElementById('metricDiskIo').textContent = 
            metrics.disk_io ? `${formatBytes(metrics.disk_io)}/s` : 'N/A';
        document.getElementById('metricNetworkIo').textContent = 
            metrics.network_io ? `${formatBytes(metrics.network_io)}/s` : 'N/A';
        document.getElementById('metricQueue').textContent = 
            metrics.queue_depth || '0';
            
    } catch (error) {
        console.error('Failed to load metrics:', error);
        // Metrics are optional, don't show error
    }
}

function updateActionButtons() {
    const startBtn = document.getElementById('startBtn');
    const stopBtn = document.getElementById('stopBtn');
    const restartBtn = document.getElementById('restartBtn');
    
    if (currentAgent.status === 'active') {
        startBtn.style.display = 'none';
        stopBtn.style.display = 'inline-flex';
        restartBtn.style.display = 'inline-flex';
    } else if (currentAgent.status === 'inactive') {
        startBtn.style.display = 'inline-flex';
        stopBtn.style.display = 'none';
        restartBtn.style.display = 'none';
    } else {
        startBtn.style.display = 'none';
        stopBtn.style.display = 'none';
        restartBtn.style.display = 'none';
    }
}

function setupActionButtons() {
    const startBtn = document.getElementById('startBtn');
    const stopBtn = document.getElementById('stopBtn');
    const restartBtn = document.getElementById('restartBtn');
    
    if (startBtn) {
        startBtn.addEventListener('click', async () => {
            if (confirm('Start this agent?')) {
                try {
                    await window.agentAPI.startAgent(agentId);
                    await loadAgentDetails();
                } catch (error) {
                    alert(`Failed to start agent: ${error.message}`);
                }
            }
        });
    }
    
    if (stopBtn) {
        stopBtn.addEventListener('click', async () => {
            if (confirm('Stop this agent?')) {
                try {
                    await window.agentAPI.stopAgent(agentId);
                    await loadAgentDetails();
                } catch (error) {
                    alert(`Failed to stop agent: ${error.message}`);
                }
            }
        });
    }
    
    if (restartBtn) {
        restartBtn.addEventListener('click', async () => {
            if (confirm('Restart this agent?')) {
                try {
                    await window.agentAPI.stopAgent(agentId);
                    setTimeout(async () => {
                        await window.agentAPI.startAgent(agentId);
                        await loadAgentDetails();
                    }, 2000);
                } catch (error) {
                    alert(`Failed to restart agent: ${error.message}`);
                }
            }
        });
    }
}

function getStatusIcon(status) {
    switch(status) {
        case 'active': return 'zap';
        case 'inactive': return 'pause';
        case 'error': return 'alert-triangle';
        default: return 'help-circle';
    }
}

function formatDateTime(timestamp) {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp);
    return date.toLocaleString();
}

function showError(message) {
    document.querySelector('.main-content').innerHTML = `
        <div class="error-container">
            <i data-lucide="alert-triangle"></i>
            <h2>Error</h2>
            <p>${message}</p>
            <button onclick="history.back()" class="action-btn">Go Back</button>
        </div>
    `;
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
}

function startAutoRefresh() {
    setInterval(async () => {
        await loadAgentDetails();
    }, 30000);
}

function startLiveUpdates() {
    // Update uptime every second
    setInterval(updateLiveUptime, 1000);
    
    // Update performance metrics every 5 seconds
    setInterval(updatePerformanceOverview, 5000);
}

function updateLiveUptime() {
    const uptimeElement = document.querySelector('[data-uptime-live]');
    if (uptimeElement && currentAgent && currentAgent.status === 'active') {
        const startTime = currentAgent.start_time || currentAgent.StartTime;
        if (startTime) {
            uptimeElement.textContent = formatUptime(new Date(startTime));
        }
    }
}

function formatUptime(startTime) {
    const now = new Date();
    const diffMs = now - startTime;
    const diffSeconds = Math.floor(diffMs / 1000);
    
    if (diffSeconds < 60) {
        return `${diffSeconds}s`;
    }
    
    const diffMinutes = Math.floor(diffSeconds / 60);
    if (diffMinutes < 60) {
        return `${diffMinutes}m ${diffSeconds % 60}s`;
    }
    
    const diffHours = Math.floor(diffMinutes / 60);
    if (diffHours < 24) {
        return `${diffHours}h ${diffMinutes % 60}m`;
    }
    
    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}d ${diffHours % 24}h`;
}

function setupTabNavigation() {
    const tabBtns = document.querySelectorAll('.tab-btn');
    const tabPanels = document.querySelectorAll('.tab-panel');
    
    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const targetTab = btn.dataset.tab;
            
            // Update active states
            tabBtns.forEach(b => b.classList.remove('active'));
            tabPanels.forEach(p => p.classList.remove('active'));
            
            btn.classList.add('active');
            document.getElementById(`${targetTab}-tab`).classList.add('active');
            
            // Reinitialize icons in the new tab
            if (typeof lucide !== 'undefined') {
                lucide.createIcons();
            }
        });
    });
}

function calculateHealthScore(agent) {
    let score = 100;
    
    // Deduct points based on status
    if (agent.status !== 'active') score -= 50;
    if (agent.status === 'error') score -= 30;
    
    // Deduct points based on metrics
    if (agent.metrics) {
        const cpu = agent.metrics.cpu_percent || 0;
        const memory = agent.metrics.memory_mb || 0;
        
        if (cpu > 80) score -= 10;
        if (cpu > 90) score -= 10;
        if (memory > 1000) score -= 5;
        if (memory > 2000) score -= 10;
        
        const errorRate = agent.metrics.error_rate || 0;
        if (errorRate > 5) score -= 10;
        if (errorRate > 10) score -= 20;
    }
    
    return Math.max(0, Math.min(100, score));
}

function getHealthClass(score) {
    if (score >= 80) return 'health-good';
    if (score >= 50) return 'health-warning';
    return 'health-critical';
}

function extractPortFromCommand(command) {
    if (!command) return null;
    
    // Look for common port patterns
    const portPatterns = [
        /--port[\s=](\d+)/,
        /-p[\s=](\d+)/,
        /:([0-9]{2,5})(?:[\s\/]|$)/,
        /PORT[\s=](\d+)/i
    ];
    
    for (const pattern of portPatterns) {
        const match = command.match(pattern);
        if (match && match[1]) {
            return match[1];
        }
    }
    
    return null;
}

function updatePerformanceOverview() {
    if (!currentAgent || !currentAgent.metrics) return;
    
    const metrics = currentAgent.metrics;
    
    // CPU
    const cpu = metrics.cpu_percent || metrics.cpu_usage || 0;
    const cpuEl = document.getElementById('perfCpu');
    if (cpuEl) {
        cpuEl.textContent = `${cpu.toFixed(1)}%`;
        const cpuTrendEl = document.getElementById('cpuTrend');
        if (cpuTrendEl) {
            cpuTrendEl.innerHTML = getTrendIndicator(cpu, metrics.cpu_prev);
        }
    }
    
    // Memory
    const memory = metrics.memory_mb || 0;
    const memEl = document.getElementById('perfMemory');
    if (memEl) {
        memEl.textContent = `${memory.toFixed(0)} MB`;
        const memTrendEl = document.getElementById('memoryTrend');
        if (memTrendEl) {
            memTrendEl.innerHTML = getTrendIndicator(memory, metrics.memory_prev);
        }
    }
    
    // Requests
    const requests = metrics.requests_per_minute || 0;
    const reqEl = document.getElementById('perfRequests');
    if (reqEl) {
        reqEl.textContent = requests.toFixed(0);
        const reqTrendEl = document.getElementById('requestsTrend');
        if (reqTrendEl) {
            reqTrendEl.innerHTML = getTrendIndicator(requests, metrics.requests_prev);
        }
    }
    
    // Latency
    const latency = metrics.response_time_ms || 0;
    const latEl = document.getElementById('perfLatency');
    if (latEl) {
        latEl.textContent = `${latency.toFixed(0)}ms`;
        const latTrendEl = document.getElementById('latencyTrend');
        if (latTrendEl) {
            latTrendEl.innerHTML = getTrendIndicator(latency, metrics.latency_prev, true);
        }
    }
}

function getTrendIndicator(current, previous, inverse = false) {
    if (previous === undefined || previous === null) return '--';
    
    const diff = current - previous;
    const percentChange = previous !== 0 ? (diff / previous * 100) : 0;
    
    if (Math.abs(percentChange) < 1) {
        return '<span class="trend-stable">→ Stable</span>';
    }
    
    const isUp = diff > 0;
    const isGood = inverse ? !isUp : isUp;
    const arrow = isUp ? '↑' : '↓';
    const className = isGood ? 'trend-up' : 'trend-down';
    
    return `<span class="${className}">${arrow} ${Math.abs(percentChange).toFixed(1)}%</span>`;
}

function renderSystemInformation() {
    // Mock system data - in production, this would come from the agent
    const sysInfo = currentAgent.system_info || {};
    
    const setTextContent = (id, value) => {
        const el = document.getElementById(id);
        if (el) el.textContent = value;
    };
    
    setTextContent('sysHostname', sysInfo.hostname || 'localhost');
    setTextContent('sysPlatform', sysInfo.platform || 'Linux');
    setTextContent('sysArch', sysInfo.arch || 'x86_64');
    setTextContent('sysCores', sysInfo.cpu_cores || '4');
    setTextContent('sysTotalMem', formatBytes(sysInfo.total_memory || 8589934592));
    
    // Resource limits
    const limits = currentAgent.resource_limits || {};
    setTextContent('limitCpu', limits.cpu || 'Unlimited');
    setTextContent('limitMemory', limits.memory ? formatBytes(limits.memory) : 'Unlimited');
    setTextContent('limitDisk', limits.disk ? formatBytes(limits.disk) : 'Unlimited');
    setTextContent('limitConnections', limits.max_connections || '1000');
    
    // Process details
    const proc = currentAgent.process_info || {};
    setTextContent('procParentPid', proc.parent_pid || '1');
    setTextContent('procThreads', proc.thread_count || '1');
    setTextContent('procNice', proc.nice_level || '0');
    setTextContent('procFds', proc.file_descriptors || 'N/A');
}

function renderNetworkInformation() {
    const netInfo = currentAgent.network_info || {};
    
    const setTextContent = (id, value) => {
        const el = document.getElementById(id);
        if (el) el.textContent = value;
    };
    
    // Network configuration
    setTextContent('netIpAddress', netInfo.ip_address || '127.0.0.1');
    setTextContent('netPort', netInfo.port || extractPortFromCommand(currentAgent.command) || 'Dynamic');
    setTextContent('netProtocol', netInfo.protocol || 'HTTP/HTTPS');
    setTextContent('netDns', netInfo.dns_servers || '8.8.8.8, 8.8.4.4');
    
    // Active connections
    const connections = netInfo.connections || [];
    const connectionsDiv = document.getElementById('activeConnections');
    
    if (connections.length > 0) {
        connectionsDiv.innerHTML = connections.map(conn => `
            <div class="connection-item">
                <span class="connection-type">${conn.type || 'TCP'}</span>
                <span class="connection-endpoint">${conn.remote || 'Unknown'}</span>
                <span class="connection-state">${conn.state || 'ESTABLISHED'}</span>
            </div>
        `).join('');
    } else {
        connectionsDiv.innerHTML = '<div class="no-data">No active connections</div>';
    }
    
    // Dependencies
    const dependencies = currentAgent.dependencies || [];
    const depsDiv = document.getElementById('agentDependencies');
    
    if (dependencies.length > 0) {
        depsDiv.innerHTML = dependencies.map(dep => `
            <div class="dependency-card">
                <div class="dep-icon"><i data-lucide="package"></i></div>
                <div class="dep-info">
                    <div class="dep-name">${dep.name}</div>
                    <div class="dep-version">${dep.version || 'latest'}</div>
                </div>
                <div class="dep-status ${dep.status || 'connected'}">
                    ${dep.status || 'Connected'}
                </div>
            </div>
        `).join('');
    } else {
        // Show common dependencies based on agent type
        const commonDeps = getCommonDependencies(currentAgent.type);
        depsDiv.innerHTML = commonDeps.map(dep => `
            <div class="dependency-card">
                <div class="dep-icon"><i data-lucide="${dep.icon}"></i></div>
                <div class="dep-info">
                    <div class="dep-name">${dep.name}</div>
                    <div class="dep-type">${dep.type}</div>
                </div>
                <div class="dep-status connected">Available</div>
            </div>
        `).join('');
    }
}

function getCommonDependencies(agentType) {
    const deps = [];
    
    // Add common dependencies based on agent type
    deps.push({
        name: 'PostgreSQL',
        type: 'Database',
        icon: 'database'
    });
    
    deps.push({
        name: 'Redis',
        type: 'Cache',
        icon: 'zap'
    });
    
    if (agentType === 'ollama' || agentType === 'claude-code') {
        deps.push({
            name: 'Qdrant',
            type: 'Vector DB',
            icon: 'search'
        });
    }
    
    if (agentType === 'n8n') {
        deps.push({
            name: 'Webhook',
            type: 'Integration',
            icon: 'link'
        });
    }
    
    return deps;
}

function renderEnvironmentVariables() {
    const envVars = currentAgent.environment || {};
    const envDiv = document.getElementById('envVariables');
    
    // Add some common environment variables if not present
    const defaultEnvVars = {
        'NODE_ENV': 'production',
        'LOG_LEVEL': 'info',
        'API_PORT': extractPortFromCommand(currentAgent.command) || '3000',
        'RESOURCE_NAME': currentAgent.type.toUpperCase(),
        'AGENT_ID': currentAgent.id
    };

    const resolvedBase = typeof window.resolveAgentDashboardApiBase === 'function'
        ? window.resolveAgentDashboardApiBase()
        : null;
    if (resolvedBase) {
        defaultEnvVars.API_BASE_URL = resolvedBase;
    }
    
    const allEnvVars = { ...defaultEnvVars, ...envVars };
    const envEntries = Object.entries(allEnvVars);
    
    if (envEntries.length > 0) {
        envDiv.innerHTML = envEntries.map(([key, value]) => {
            const isSensitive = key.includes('KEY') || key.includes('SECRET') || key.includes('PASSWORD');
            const displayValue = isSensitive ? '••••••••' : value;
            
            return `
                <div class="env-var-item" data-env-key="${key.toLowerCase()}">
                    <span class="env-key">${key}</span>
                    <span class="env-value ${isSensitive ? 'sensitive' : ''}" 
                          onclick="${isSensitive ? 'toggleSensitive(this)' : 'copyToClipboard(this)'}" 
                          data-value="${value}">${displayValue}</span>
                </div>
            `;
        }).join('');
    } else {
        envDiv.innerHTML = '<div class="no-data">No environment variables configured</div>';
    }
    
    // Render configuration
    const config = currentAgent.configuration || {};
    const configDiv = document.getElementById('agentConfig');
    
    const configItems = [
        { label: 'Auto-restart', value: config.auto_restart !== false ? 'Enabled' : 'Disabled' },
        { label: 'Health Check', value: config.health_check !== false ? 'Enabled' : 'Disabled' },
        { label: 'Log Rotation', value: config.log_rotation || 'Daily' },
        { label: 'Max Retries', value: config.max_retries || '3' },
        { label: 'Timeout', value: config.timeout ? `${config.timeout}s` : '30s' }
    ];
    
    configDiv.innerHTML = configItems.map(item => `
        <div class="config-item">
            <span class="config-label">${item.label}:</span>
            <span class="config-value">${item.value}</span>
        </div>
    `).join('');
}

function setupEnvironmentSearch() {
    const searchInput = document.getElementById('envSearch');
    if (searchInput) {
        searchInput.addEventListener('input', (e) => {
            const searchTerm = e.target.value.toLowerCase();
            const envItems = document.querySelectorAll('.env-var-item');
            
            envItems.forEach(item => {
                const key = item.dataset.envKey;
                if (key.includes(searchTerm)) {
                    item.style.display = 'flex';
                } else {
                    item.style.display = 'none';
                }
            });
        });
    }
}

// Utility functions
function copyToClipboard(element) {
    const text = element.dataset?.value || element.textContent;
    navigator.clipboard.writeText(text).then(() => {
        const originalText = element.textContent;
        element.textContent = 'Copied!';
        element.classList.add('copied');
        setTimeout(() => {
            element.textContent = originalText;
            element.classList.remove('copied');
        }, 2000);
    });
}

function copyCommand() {
    const command = document.getElementById('agentCommand').textContent;
    navigator.clipboard.writeText(command).then(() => {
        // Show feedback
        const btn = event.target.closest('.copy-btn');
        btn.classList.add('copied');
        setTimeout(() => btn.classList.remove('copied'), 2000);
    });
}

function toggleSensitive(element) {
    const value = element.dataset.value;
    if (element.textContent === '••••••••') {
        element.textContent = value;
    } else {
        element.textContent = '••••••••';
    }
}

function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
