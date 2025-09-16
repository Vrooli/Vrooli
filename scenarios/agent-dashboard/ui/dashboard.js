// Dashboard Page JavaScript
// Main dashboard functionality with agent grid and radar

let currentAgents = [];
let filteredAgents = [];
let agentRadar = null;

// Filter and sort state
let currentSearchQuery = '';
let currentSortOption = 'name-asc';

document.addEventListener('DOMContentLoaded', async () => {
    initializeRadar();
    initializeDashboardControls();
    await fetchAndRenderAgents();
    startRealtimeUpdates();
});

function initializeRadar() {
    const radarContainer = document.getElementById('agentRadar');
    if (radarContainer && typeof AgentRadar !== 'undefined') {
        agentRadar = new AgentRadar('agentRadar');
        console.log('🎯 Agent radar initialized');
    }
}

function initializeDashboardControls() {
    const searchInput = document.getElementById('agentSearch');
    const searchClear = document.getElementById('searchClear');
    const sortSelect = document.getElementById('agentSort');
    
    if (searchInput) {
        // Real-time search
        searchInput.addEventListener('input', (e) => {
            currentSearchQuery = e.target.value.trim();
            updateSearchClearButton();
            applyFiltersAndSort();
        });
        
        // Clear search
        if (searchClear) {
            searchClear.addEventListener('click', () => {
                searchInput.value = '';
                currentSearchQuery = '';
                updateSearchClearButton();
                applyFiltersAndSort();
                searchInput.focus();
            });
        }
    }
    
    if (sortSelect) {
        sortSelect.addEventListener('change', (e) => {
            currentSortOption = e.target.value;
            applyFiltersAndSort();
        });
    }
    
    console.log('🎛️ Dashboard controls initialized');
}

function updateSearchClearButton() {
    const searchClear = document.getElementById('searchClear');
    if (searchClear) {
        searchClear.style.display = currentSearchQuery ? 'flex' : 'none';
    }
}

function filterAgents(agents, searchQuery) {
    if (!searchQuery) return agents;
    
    const query = searchQuery.toLowerCase();
    return agents.filter(agent => {
        // Search by name, type, status, PID, or command
        const searchableText = [
            agent.name || '',
            agent.type || '',
            agent.status || '',
            agent.pid ? agent.pid.toString() : '',
            agent.command || ''
        ].join(' ').toLowerCase();
        
        return searchableText.includes(query);
    });
}

function sortAgents(agents, sortOption) {
    const sorted = [...agents];
    
    switch (sortOption) {
        case 'name-asc':
            return sorted.sort((a, b) => (a.name || '').localeCompare(b.name || ''));
        case 'name-desc':
            return sorted.sort((a, b) => (b.name || '').localeCompare(a.name || ''));
        case 'type-asc':
            return sorted.sort((a, b) => (a.type || '').localeCompare(b.type || ''));
        case 'type-desc':
            return sorted.sort((a, b) => (b.type || '').localeCompare(a.type || ''));
        case 'status-active':
            return sorted.sort((a, b) => {
                if (a.status === 'active' && b.status !== 'active') return -1;
                if (b.status === 'active' && a.status !== 'active') return 1;
                return (a.name || '').localeCompare(b.name || '');
            });
        case 'status-inactive':
            return sorted.sort((a, b) => {
                if (a.status !== 'active' && b.status === 'active') return -1;
                if (b.status !== 'active' && a.status === 'active') return 1;
                return (a.name || '').localeCompare(b.name || '');
            });
        case 'uptime-newest':
            return sorted.sort((a, b) => {
                const aTime = new Date(a.start_time || a.StartTime || 0);
                const bTime = new Date(b.start_time || b.StartTime || 0);
                return bTime - aTime; // Newer first
            });
        case 'uptime-oldest':
            return sorted.sort((a, b) => {
                const aTime = new Date(a.start_time || a.StartTime || 0);
                const bTime = new Date(b.start_time || b.StartTime || 0);
                return aTime - bTime; // Older first
            });
        case 'memory-high':
            return sorted.sort((a, b) => {
                const aMemory = (a.metrics && a.metrics.memory_mb) || 0;
                const bMemory = (b.metrics && b.metrics.memory_mb) || 0;
                return bMemory - aMemory; // Higher first
            });
        case 'memory-low':
            return sorted.sort((a, b) => {
                const aMemory = (a.metrics && a.metrics.memory_mb) || 0;
                const bMemory = (b.metrics && b.metrics.memory_mb) || 0;
                return aMemory - bMemory; // Lower first
            });
        default:
            return sorted;
    }
}

function applyFiltersAndSort() {
    // Filter first, then sort
    const filtered = filterAgents(currentAgents, currentSearchQuery);
    filteredAgents = sortAgents(filtered, currentSortOption);
    
    // Update the grid with filtered agents
    renderAgentGrid(filteredAgents);
    updateAgentCounts();
    updateRadar();
}

function updateAgentCounts() {
    const totalCount = document.getElementById('totalAgentCount');
    const visibleCount = document.getElementById('visibleAgentCount');
    
    if (totalCount) totalCount.textContent = currentAgents.length;
    if (visibleCount) visibleCount.textContent = filteredAgents.length;
}

async function fetchAndRenderAgents() {
    try {
        currentAgents = await window.agentAPI.fetchAgents();
        
        // Apply current filters and sort
        applyFiltersAndSort();
        
        // Update header agent count
        if (window.updateAgentCount) {
            window.updateAgentCount(currentAgents.length);
        }
    } catch (error) {
        console.error('Failed to fetch agents:', error);
        currentAgents = [];
        filteredAgents = [];
        renderEmptyState();
        updateAgentCounts();
    }
}

function renderAgentGrid(agentsToRender = null) {
    const grid = document.getElementById('agentGrid');
    if (!grid) return;
    
    // Use provided agents or fall back to filtered agents
    const agents = agentsToRender || filteredAgents;
    
    grid.innerHTML = '';
    
    if (agents.length === 0) {
        if (currentSearchQuery) {
            renderSearchEmptyState();
        } else {
            renderEmptyState();
        }
        return;
    }
    
    grid.classList.remove('empty');
    
    agents.forEach((agent, index) => {
        const card = createAgentCard(agent);
        const cardElement = document.createElement('div');
        cardElement.innerHTML = card;
        cardElement.firstElementChild.style.animationDelay = `${index * 0.1}s`;
        grid.appendChild(cardElement.firstElementChild);
        
        // Initialize uptime for this card
        const uptimeElement = cardElement.querySelector('[data-uptime-value]');
        if (uptimeElement) {
            // Check multiple possible timestamp fields
            const startTime = agent.start_time || agent.StartTime || agent.startTime;
            if (startTime) {
                const uptimeText = formatLiveUptime(startTime, agent.status === 'active');
                uptimeElement.textContent = uptimeText;
            }
        }
    });
    
    // Reinitialize Lucide icons for new content
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
}

function renderEmptyState() {
    const grid = document.getElementById('agentGrid');
    if (grid) {
        grid.classList.add('empty');
        grid.innerHTML = '<div class="no-agents">No active agents found. Agents will appear here when resource services start agents.</div>';
    }
    updateRadar();
}

function renderSearchEmptyState() {
    const grid = document.getElementById('agentGrid');
    if (grid) {
        grid.classList.add('empty');
        grid.innerHTML = `
            <div class="no-agents">
                <div style="margin-bottom: 10px;">No agents match "${currentSearchQuery}"</div>
                <div style="font-size: 14px; opacity: 0.7;">Try a different search term or clear the search to see all agents.</div>
            </div>
        `;
    }
    updateRadar();
}

function updateRadar() {
    if (agentRadar) {
        // Use filtered agents for radar display
        agentRadar.updateAgents(filteredAgents);
    }
}

function startRealtimeUpdates() {
    // Refresh agent data every 30 seconds
    setInterval(fetchAndRenderAgents, 30000);
    
    // Update live uptime counters every second
    setInterval(updateLiveUptimes, 1000);
}

function updateLiveUptimes() {
    // Only update live uptimes for currently visible (filtered) agents
    const agentCards = document.querySelectorAll('.agent-card[data-start-time][data-status="active"]');
    
    agentCards.forEach(card => {
        const startTime = card.dataset.startTime;
        const uptimeElement = card.querySelector('[data-uptime-value]');
        
        if (startTime && uptimeElement) {
            const uptimeText = formatLiveUptime(startTime, true);
            uptimeElement.textContent = uptimeText;
        }
    });
}

function formatLiveUptime(startTime, isRunning) {
    if (!startTime || !isRunning) {
        return 'N/A';
    }
    
    // Handle both string and Date objects
    const start = typeof startTime === 'string' ? new Date(startTime) : startTime;
    
    // Check if the date is valid
    if (isNaN(start.getTime())) {
        console.warn('Invalid start time:', startTime);
        return 'N/A';
    }
    
    const now = new Date();
    const diffMs = now - start;
    const diffSeconds = Math.floor(diffMs / 1000);
    
    // Handle negative time differences (clock skew)
    if (diffSeconds < 0) {
        return '0s';
    }
    
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

function createAgentCard(agent) {
    return `
        <div class="agent-card" data-agent-id="${agent.id}" data-start-time="${agent.start_time || agent.StartTime || agent.startTime || ''}" data-status="${agent.status}" onclick="window.location.href='/agent.html?id=${agent.id}'">
            <div class="agent-header">
                <div class="agent-name">
                    <i data-lucide="bot"></i>
                    ${agent.name}
                </div>
                <div class="agent-type">${agent.type.toUpperCase()}</div>
            </div>
            
            <div class="agent-status">
                <span class="status-badge ${agent.status}">
                    <i data-lucide="${agent.status === 'active' ? 'zap' : agent.status === 'error' ? 'alert-triangle' : 'pause'}"></i>
                    ${agent.status.toUpperCase()}
                </span>
                <span class="agent-uptime" data-uptime-target>
                    <i data-lucide="clock"></i>
                    Uptime: <span data-uptime-value>N/A</span>
                </span>
            </div>
            
            <div class="agent-metrics-mini">
                <div class="metric-mini">
                    <span>PID:</span> ${agent.pid || 'N/A'}
                </div>
                <div class="metric-mini">
                    <span>CPU:</span> ${agent.metrics && agent.metrics.cpu_percent ? Number(agent.metrics.cpu_percent).toFixed(1) + '%' : 'N/A'}
                </div>
                <div class="metric-mini">
                    <span>Memory:</span> ${agent.metrics && agent.metrics.memory_mb ? Number(agent.metrics.memory_mb).toFixed(0) + 'MB' : 'N/A'}
                </div>
            </div>
            
            <div class="agent-controls">
                <button class="control-btn" onclick="event.stopPropagation(); quickStop('${agent.id}')">
                    <i data-lucide="square"></i>
                    STOP
                </button>
                <a href="/logs.html?agent=${agent.id}" class="control-btn" onclick="event.stopPropagation()">
                    <i data-lucide="scroll-text"></i>
                    LOGS
                </a>
            </div>
        </div>
    `;
}

function formatTime(timestamp) {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
}


// Quick actions
async function quickStop(agentId) {
    if (confirm('Stop this agent?')) {
        try {
            await window.agentAPI.stopAgent(agentId);
            await fetchAndRenderAgents();
        } catch (error) {
            alert(`Failed to stop agent: ${error.message}`);
        }
    }
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey || e.metaKey) {
        switch(e.key) {
            case 'r':
                e.preventDefault();
                fetchAndRenderAgents();
                break;
            case 's':
                e.preventDefault();
                window.agentAPI.triggerScan().then(() => {
                    setTimeout(fetchAndRenderAgents, 2000);
                });
                break;
            case 'f':
                e.preventDefault();
                const searchInput = document.getElementById('agentSearch');
                if (searchInput) {
                    searchInput.focus();
                    searchInput.select();
                }
                break;
        }
    }
    
    // ESC to clear search
    if (e.key === 'Escape') {
        const searchInput = document.getElementById('agentSearch');
        if (searchInput && searchInput === document.activeElement) {
            searchInput.blur();
        }
        if (currentSearchQuery) {
            searchInput.value = '';
            currentSearchQuery = '';
            updateSearchClearButton();
            applyFiltersAndSort();
        }
    }
});