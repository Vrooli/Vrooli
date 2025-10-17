// Agents Page JavaScript
// Detailed agent listing grouped by resource with filtering

let allAgents = [];
let groupedAgents = {};
let resourceStats = {};

document.addEventListener('DOMContentLoaded', async () => {
    await loadAgents();
    setupFilters();
    startAutoRefresh();
});

async function loadAgents() {
    try {
        allAgents = await window.agentAPI.fetchAgents();
        
        // If no agents found, create mock data to show resource structure
        if (!allAgents || allAgents.length === 0) {
            console.log('No agents found, showing resource structure with empty groups');
            // Show empty resource groups for common resources
            showEmptyResourceGroups();
        } else {
            groupAgentsByResource();
            updateTypeFilter();
            applyFilters();
        }
        
        updateAgentCount(allAgents.length);
    } catch (error) {
        console.error('Failed to load agents:', error);
        // Show empty resource groups even on error
        showEmptyResourceGroups();
    }
}

function showEmptyResourceGroups() {
    // Common resources that might have agents
    const commonResources = [
        'ollama', 'claude-code', 'n8n', 'postgres', 'redis', 
        'qdrant', 'huginn', 'litellm', 'whisper', 'comfyui'
    ];
    
    // Initialize empty groups for display
    groupedAgents = {};
    resourceStats = {};
    
    commonResources.forEach(resource => {
        groupedAgents[resource] = [];
        resourceStats[resource] = {
            total: 0,
            active: 0,
            inactive: 0,
            error: 0
        };
    });
    
    updateTypeFilter();
    applyFilters();
}

function groupAgentsByResource() {
    // Reset groups
    groupedAgents = {};
    resourceStats = {};
    
    // Group agents by resource type
    allAgents.forEach(agent => {
        const resourceType = agent.type || 'unknown';
        
        if (!groupedAgents[resourceType]) {
            groupedAgents[resourceType] = [];
            resourceStats[resourceType] = {
                total: 0,
                active: 0,
                inactive: 0,
                error: 0
            };
        }
        
        groupedAgents[resourceType].push(agent);
        resourceStats[resourceType].total++;
        resourceStats[resourceType][agent.status]++;
    });
}

function updateTypeFilter() {
    const typeFilter = document.getElementById('filterType');
    const types = Object.keys(groupedAgents).sort();
    
    // Clear existing options except "All Types"
    typeFilter.innerHTML = '<option value="all">All Resources</option>';
    
    types.forEach(type => {
        const stats = resourceStats[type];
        const option = document.createElement('option');
        option.value = type;
        option.textContent = `${type.toUpperCase()} (${stats.total})`;
        typeFilter.appendChild(option);
    });
}

function setupFilters() {
    document.getElementById('filterStatus').addEventListener('change', applyFilters);
    document.getElementById('filterType').addEventListener('change', applyFilters);
}

function applyFilters() {
    const statusFilter = document.getElementById('filterStatus').value;
    const typeFilter = document.getElementById('filterType').value;
    
    renderResourceGroups(statusFilter, typeFilter);
}

function renderResourceGroups(statusFilter = 'all', typeFilter = 'all') {
    const container = document.getElementById('resourceGroups');
    container.innerHTML = '';
    
    // Get resources to display
    const resourcesToShow = typeFilter === 'all' 
        ? Object.keys(groupedAgents).sort()
        : [typeFilter].filter(t => groupedAgents[t]);
    
    if (resourcesToShow.length === 0) {
        container.innerHTML = `
            <div class="empty-message">
                <i data-lucide="server-off"></i>
                <p>No resources configured</p>
                <p style="font-size: 14px; margin-top: 10px;">Resources with agent support will appear here when configured.</p>
            </div>
        `;
        if (typeof lucide !== 'undefined') {
            lucide.createIcons();
        }
        return;
    }
    
    // Render each resource group
    resourcesToShow.forEach(resourceType => {
        const agents = groupedAgents[resourceType] || [];
        const stats = resourceStats[resourceType] || { total: 0, active: 0, inactive: 0, error: 0 };
        
        // Filter agents by status if needed
        const filteredAgents = statusFilter === 'all' 
            ? agents 
            : agents.filter(a => a.status === statusFilter);
        
        // Always show the resource group, even if empty
        // Only hide if filtering by status and no matches
        if (statusFilter !== 'all' && agents.length > 0 && filteredAgents.length === 0) {
            return; // Skip only if resource has agents but none match filter
        }
        
        const resourceGroup = createResourceGroup(resourceType, filteredAgents, stats, agents.length === 0);
        container.appendChild(resourceGroup);
    });
    
    // Reinitialize Lucide icons
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
}

function createResourceGroup(resourceType, agents, stats, isEmpty = false) {
    const group = document.createElement('div');
    group.className = 'resource-group';
    group.setAttribute('data-resource', resourceType);
    
    // Create header with stats
    const header = document.createElement('div');
    header.className = 'resource-header';
    
    const agentCount = agents.length;
    const countText = isEmpty ? 'No agents' : `${agentCount} agent${agentCount !== 1 ? 's' : ''}`;
    
    header.innerHTML = `
        <div class="resource-title">
            <i data-lucide="server"></i>
            <span class="resource-name">${resourceType.toUpperCase()}</span>
            <span class="resource-count ${isEmpty ? 'empty' : ''}">${countText}</span>
        </div>
        <div class="resource-stats">
            ${!isEmpty ? `
                <span class="stat-badge active">Active: ${stats.active}</span>
                <span class="stat-badge inactive">Inactive: ${stats.inactive}</span>
                ${stats.error > 0 ? `<span class="stat-badge error">Error: ${stats.error}</span>` : ''}
            ` : `
                <span class="stat-badge empty">No agents configured</span>
            `}
            <button class="collapse-btn" onclick="toggleResourceGroup('${resourceType}')">
                <i data-lucide="chevron-down"></i>
            </button>
        </div>
    `;
    group.appendChild(header);
    
    // Create table container
    const tableContainer = document.createElement('div');
    tableContainer.className = 'resource-table-container';
    tableContainer.id = `resource-${resourceType}`;
    
    if (agents.length === 0) {
        // Show empty state message
        tableContainer.innerHTML = `
            <div class="resource-empty-state">
                <i data-lucide="bot-off"></i>
                <p>No agents detected in ${resourceType}</p>
                <p class="hint">Agents will appear here when the resource starts them</p>
            </div>
        `;
    } else {
        // Create table with agents
        const table = document.createElement('table');
        table.className = 'agents-table';
        
        table.innerHTML = `
            <thead>
                <tr>
                    <th>Status</th>
                    <th>Name</th>
                    <th>PID</th>
                    <th>Uptime</th>
                    <th>Last Seen</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                ${agents.map(agent => createTableRowHTML(agent)).join('')}
            </tbody>
        `;
        
        tableContainer.appendChild(table);
    }
    
    group.appendChild(tableContainer);
    
    return group;
}

function createTableRowHTML(agent) {
    return `
        <tr class="agent-row status-${agent.status}">
            <td>
                <span class="status-badge ${agent.status}">
                    <i data-lucide="${getStatusIcon(agent.status)}"></i>
                    ${agent.status}
                </span>
            </td>
            <td class="agent-name-cell">
                <i data-lucide="bot"></i>
                ${agent.name}
            </td>
            <td class="mono">${agent.pid || 'N/A'}</td>
            <td>${agent.uptime || 'N/A'}</td>
            <td>${formatTimestamp(agent.last_seen)}</td>
            <td class="action-buttons">
                <button class="action-btn-small" onclick="viewAgent('${agent.id}')" title="View Details">
                    <i data-lucide="eye"></i>
                </button>
                <button class="action-btn-small" onclick="viewLogs('${agent.id}')" title="View Logs">
                    <i data-lucide="terminal"></i>
                </button>
                ${agent.status === 'active' ? 
                    `<button class="action-btn-small" onclick="stopAgent('${agent.id}')" title="Stop Agent">
                        <i data-lucide="square"></i>
                    </button>` :
                    `<button class="action-btn-small" onclick="startAgent('${agent.id}')" title="Start Agent">
                        <i data-lucide="play"></i>
                    </button>`
                }
            </td>
        </tr>
    `;
}

function toggleResourceGroup(resourceType) {
    const container = document.getElementById(`resource-${resourceType}`);
    const button = event.currentTarget;
    const icon = button.querySelector('svg');
    
    if (container.style.display === 'none') {
        container.style.display = 'block';
        icon.setAttribute('data-lucide', 'chevron-down');
    } else {
        container.style.display = 'none';
        icon.setAttribute('data-lucide', 'chevron-right');
    }
    
    // Reinitialize icon
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
}

function createTableRow(agent) {
    const tr = document.createElement('tr');
    tr.className = `agent-row status-${agent.status}`;
    
    tr.innerHTML = `
        <td>
            <span class="status-badge ${agent.status}">
                <i data-lucide="${getStatusIcon(agent.status)}"></i>
                ${agent.status}
            </span>
        </td>
        <td class="agent-name-cell">
            <i data-lucide="bot"></i>
            ${agent.name}
        </td>
        <td>${agent.type.toUpperCase()}</td>
        <td class="mono">${agent.pid || 'N/A'}</td>
        <td>${agent.uptime || 'N/A'}</td>
        <td>${formatTimestamp(agent.last_seen)}</td>
        <td class="action-buttons">
            <button class="action-btn-small" onclick="viewAgent('${agent.id}')" title="View Details">
                <i data-lucide="eye"></i>
            </button>
            <button class="action-btn-small" onclick="viewLogs('${agent.id}')" title="View Logs">
                <i data-lucide="terminal"></i>
            </button>
            ${agent.status === 'active' ? 
                `<button class="action-btn-small" onclick="stopAgent('${agent.id}')" title="Stop Agent">
                    <i data-lucide="square"></i>
                </button>` :
                `<button class="action-btn-small" onclick="startAgent('${agent.id}')" title="Start Agent">
                    <i data-lucide="play"></i>
                </button>`
            }
        </td>
    `;
    
    return tr;
}

function getStatusIcon(status) {
    switch(status) {
        case 'active': return 'zap';
        case 'inactive': return 'pause';
        case 'error': return 'alert-triangle';
        default: return 'help-circle';
    }
}

function formatTimestamp(timestamp) {
    if (!timestamp) return 'Never';
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;
    
    if (diff < 60000) return 'Just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    
    return date.toLocaleDateString();
}

function viewAgent(agentId) {
    window.location.href = `/agent.html?id=${agentId}`;
}

function viewLogs(agentId) {
    window.location.href = `/logs.html?agent=${agentId}`;
}

async function stopAgent(agentId) {
    if (confirm('Stop this agent?')) {
        try {
            await window.agentAPI.stopAgent(agentId);
            await loadAgents();
        } catch (error) {
            alert(`Failed to stop agent: ${error.message}`);
        }
    }
}

async function startAgent(agentId) {
    if (confirm('Start this agent?')) {
        try {
            await window.agentAPI.startAgent(agentId);
            await loadAgents();
        } catch (error) {
            alert(`Failed to start agent: ${error.message}`);
        }
    }
}

function showError(message) {
    const tbody = document.getElementById('agentsTableBody');
    tbody.innerHTML = `
        <tr>
            <td colspan="7" class="error-message">
                <i data-lucide="alert-triangle"></i>
                ${message}
            </td>
        </tr>
    `;
}

function startAutoRefresh() {
    setInterval(loadAgents, 30000);
}

function updateAgentCount(count) {
    const agentCountElement = document.getElementById('agentCount');
    if (agentCountElement) {
        agentCountElement.textContent = `${count} AGENT${count !== 1 ? 'S' : ''} ACTIVE`;
        
        const statusIcon = agentCountElement.parentElement.querySelector('.status-icon');
        if (statusIcon) {
            statusIcon.classList.remove('active', 'warning', 'error');
            statusIcon.classList.add(count > 0 ? 'active' : 'warning');
        }
    }
}