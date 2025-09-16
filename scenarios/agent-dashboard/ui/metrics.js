// Metrics Page JavaScript
// System-wide performance metrics and analytics

let allAgents = [];
let metricsData = {};

document.addEventListener('DOMContentLoaded', async () => {
    await loadMetrics();
    setupControls();
    startAutoRefresh();
});

async function loadMetrics() {
    try {
        // Load all agents
        allAgents = await window.agentAPI.fetchAgents();
        
        // Calculate summary metrics
        updateSummaryCards();
        
        // Load detailed metrics for each agent
        await loadDetailedMetrics();
        
        // Render visualizations
        renderResourceChart();
        renderPerformanceTable();
        renderRecentEvents();
        
    } catch (error) {
        console.error('Failed to load metrics:', error);
    }
}

function updateSummaryCards() {
    const totalAgents = allAgents.length;
    const activeAgents = allAgents.filter(a => a.status === 'active').length;
    
    document.getElementById('totalAgents').textContent = totalAgents;
    document.getElementById('activeAgents').textContent = activeAgents;
    
    // Calculate average CPU and total memory (using mock data for now)
    let totalCpu = 0;
    let totalMemory = 0;
    
    allAgents.forEach(agent => {
        if (agent.metrics) {
            totalCpu += agent.metrics.cpu_percent || 0;
            totalMemory += agent.metrics.memory_mb || 0;
        }
    });
    
    const avgCpu = totalAgents > 0 ? (totalCpu / totalAgents) : 0;
    document.getElementById('avgCpu').textContent = `${Number(avgCpu).toFixed(1)}%`;
    document.getElementById('totalMemory').textContent = `${Number(totalMemory).toFixed(0)} MB`;
}

async function loadDetailedMetrics() {
    // Load metrics for each agent
    for (const agent of allAgents) {
        try {
            const metrics = await window.agentAPI.fetchAgentMetrics(agent.id);
            metricsData[agent.id] = processMetricsData(metrics);
        } catch (error) {
            console.error(`Failed to load metrics for agent ${agent.id}:`, error);
            // Use empty metrics on error
            metricsData[agent.id] = processMetricsData({});
        }
    }
}

function processMetricsData(metrics) {
    // Process and normalize metrics data from backend
    return {
        cpu_usage: metrics.cpu_usage || 0,
        memory_mb: metrics.memory_mb || 0,
        io_read_bytes: metrics.io_read_bytes || 0,
        io_write_bytes: metrics.io_write_bytes || 0,
        thread_count: metrics.thread_count || 1,
        fd_count: metrics.fd_count || 0,
        response_time_ms: metrics.response_time_ms || 0,
        tasks_total: metrics.tasks_total || 0,
        tasks_completed: metrics.tasks_completed || 0,
        tasks_failed: metrics.tasks_failed || 0,
        success_rate: metrics.success_rate || 0,
        api_calls: metrics.api_calls || 0
    };
}

function renderResourceChart() {
    const resourceCounts = {};
    
    // Count agents by type
    allAgents.forEach(agent => {
        resourceCounts[agent.type] = (resourceCounts[agent.type] || 0) + 1;
    });
    
    const chartContainer = document.getElementById('resourceChart');
    chartContainer.innerHTML = '';
    
    // Create simple text-based chart
    const total = allAgents.length;
    Object.entries(resourceCounts).forEach(([type, count]) => {
        const percentage = total > 0 ? Math.round((count / total) * 100) : 0;
        
        const item = document.createElement('div');
        item.className = 'chart-item';
        item.innerHTML = `
            <div class="chart-label">${type.toUpperCase()}</div>
            <div class="chart-bar">
                <div class="chart-fill" style="width: ${percentage}%"></div>
            </div>
            <div class="chart-value">${count} (${percentage}%)</div>
        `;
        chartContainer.appendChild(item);
    });
    
    if (Object.keys(resourceCounts).length === 0) {
        chartContainer.innerHTML = '<div class="no-data">No agent data available</div>';
    }
}

function renderPerformanceTable() {
    const tbody = document.getElementById('performanceTable');
    tbody.innerHTML = '';
    
    if (allAgents.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="5" class="empty-message">No agents to display</td>
            </tr>
        `;
        return;
    }
    
    // Show top 10 agents by activity
    const activeAgents = allAgents
        .filter(a => a.status === 'active')
        .slice(0, 10);
    
    activeAgents.forEach(agent => {
        const metrics = metricsData[agent.id] || {};
        const row = document.createElement('tr');
        
        // Format IO stats for display
        const ioReadMB = Number((metrics.io_read_bytes || 0) / (1024 * 1024)).toFixed(1);
        const ioWriteMB = Number((metrics.io_write_bytes || 0) / (1024 * 1024)).toFixed(1);
        
        row.innerHTML = `
            <td class="agent-name-cell">
                <i data-lucide="bot"></i>
                ${agent.name}
            </td>
            <td>
                <div class="cpu-indicator ${getCpuClass(metrics.cpu_usage)}">
                    ${Number(metrics.cpu_usage || 0).toFixed(1)}%
                </div>
            </td>
            <td>${Number(metrics.memory_mb || 0).toFixed(0)} MB</td>
            <td title="Threads: ${metrics.thread_count || 1}, FDs: ${metrics.fd_count || 0}">
                T:${metrics.thread_count || 1} / FD:${metrics.fd_count || 0}
            </td>
            <td title="Read: ${ioReadMB}MB, Write: ${ioWriteMB}MB">
                ↓${ioReadMB} / ↑${ioWriteMB} MB
            </td>
        `;
        
        tbody.appendChild(row);
    });
    
    // Reinitialize icons
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
}

function getCpuClass(cpu) {
    if (!cpu) return '';
    if (cpu > 80) return 'high';
    if (cpu > 50) return 'medium';
    return 'low';
}

function getSuccessClass(rate) {
    if (!rate) return '';
    if (rate >= 95) return 'excellent';
    if (rate >= 80) return 'good';
    return 'poor';
}

function renderRecentEvents() {
    const eventsList = document.getElementById('eventsList');
    
    // Generate mock events for demonstration
    const events = [
        { time: new Date(Date.now() - 60000), type: 'start', message: 'Agent ollama:mock-agent started' },
        { time: new Date(Date.now() - 180000), type: 'stop', message: 'Agent n8n:worker-1 stopped' },
        { time: new Date(Date.now() - 300000), type: 'error', message: 'Agent postgres:mock-agent error: connection timeout' },
        { time: new Date(Date.now() - 600000), type: 'scan', message: 'Resource scan completed: 5 agents found' },
        { time: new Date(Date.now() - 900000), type: 'health', message: 'Health check passed for all agents' }
    ];
    
    eventsList.innerHTML = '';
    
    events.forEach(event => {
        const eventDiv = document.createElement('div');
        eventDiv.className = `event-item event-${event.type}`;
        
        eventDiv.innerHTML = `
            <div class="event-time">${formatEventTime(event.time)}</div>
            <div class="event-message">
                <i data-lucide="${getEventIcon(event.type)}"></i>
                ${event.message}
            </div>
        `;
        
        eventsList.appendChild(eventDiv);
    });
    
    // Reinitialize icons
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
}

function getEventIcon(type) {
    switch(type) {
        case 'start': return 'play';
        case 'stop': return 'square';
        case 'error': return 'alert-triangle';
        case 'scan': return 'search';
        case 'health': return 'shield-check';
        default: return 'info';
    }
}

function formatEventTime(date) {
    const now = new Date();
    const diff = now - date;
    
    if (diff < 60000) return 'Just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    
    return date.toLocaleTimeString();
}

function setupControls() {
    document.getElementById('refreshMetrics').addEventListener('click', loadMetrics);
    document.getElementById('timeRange').addEventListener('change', loadMetrics);
}

function startAutoRefresh() {
    // Refresh metrics every 30 seconds
    setInterval(loadMetrics, 30000);
}

// Update agent count in header
window.addEventListener('load', async () => {
    try {
        const agents = await window.agentAPI.fetchAgents();
        const activeCount = agents.filter(a => a.status === 'active').length;
        
        const agentCountElement = document.getElementById('agentCount');
        if (agentCountElement) {
            agentCountElement.textContent = `${activeCount} AGENT${activeCount !== 1 ? 'S' : ''} ACTIVE`;
            
            const statusIcon = agentCountElement.parentElement.querySelector('.status-icon');
            if (statusIcon) {
                statusIcon.classList.remove('active', 'warning', 'error');
                statusIcon.classList.add(activeCount > 0 ? 'active' : 'warning');
            }
        }
    } catch (error) {
        console.error('Failed to update agent count:', error);
    }
});