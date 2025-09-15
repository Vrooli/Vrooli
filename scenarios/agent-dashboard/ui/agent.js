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
    
    await loadAgentDetails();
    setupActionButtons();
    startAutoRefresh();
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
    
    // Update information section
    document.getElementById('agentId').textContent = currentAgent.id;
    document.getElementById('agentStartTime').textContent = formatDateTime(currentAgent.start_time);
    document.getElementById('agentLastSeen').textContent = formatDateTime(currentAgent.last_seen);
    document.getElementById('agentVersion').textContent = currentAgent.version || '1.0.0';
    
    // Update capabilities
    const capabilitiesDiv = document.getElementById('agentCapabilities');
    if (currentAgent.capabilities && currentAgent.capabilities.length > 0) {
        capabilitiesDiv.innerHTML = currentAgent.capabilities.map(cap => 
            `<span class="capability-badge">${cap}</span>`
        ).join('');
    } else {
        capabilitiesDiv.innerHTML = '<span class="no-data">No capabilities defined</span>';
    }
    
    // Update command
    document.getElementById('agentCommand').textContent = currentAgent.command || 'N/A';
    
    // Update action buttons
    updateActionButtons();
    
    // Update view logs button
    document.getElementById('viewLogsBtn').href = `/logs.html?agent=${currentAgent.id}`;
    
    // Reinitialize icons
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
}

async function loadQuickMetrics() {
    try {
        const metrics = await window.agentAPI.fetchAgentMetrics(agentId);
        
        document.getElementById('metricCpu').textContent = 
            metrics.cpu_usage ? `${metrics.cpu_usage}%` : 'N/A';
        document.getElementById('metricMemory').textContent = 
            metrics.memory_mb ? `${metrics.memory_mb} MB` : 'N/A';
        document.getElementById('metricTasks').textContent = 
            metrics.tasks_completed || '0';
        document.getElementById('metricSuccess').textContent = 
            metrics.success_rate ? `${metrics.success_rate}%` : 'N/A';
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
    document.getElementById('startBtn').addEventListener('click', async () => {
        if (confirm('Start this agent?')) {
            try {
                await window.agentAPI.startAgent(agentId);
                await loadAgentDetails();
            } catch (error) {
                alert(`Failed to start agent: ${error.message}`);
            }
        }
    });
    
    document.getElementById('stopBtn').addEventListener('click', async () => {
        if (confirm('Stop this agent?')) {
            try {
                await window.agentAPI.stopAgent(agentId);
                await loadAgentDetails();
            } catch (error) {
                alert(`Failed to stop agent: ${error.message}`);
            }
        }
    });
    
    document.getElementById('restartBtn').addEventListener('click', async () => {
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