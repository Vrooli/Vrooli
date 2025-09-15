// Dashboard Page JavaScript
// Main dashboard functionality with agent grid and radar

let currentAgents = [];
let agentRadar = null;

document.addEventListener('DOMContentLoaded', async () => {
    initializeRadar();
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

async function fetchAndRenderAgents() {
    try {
        currentAgents = await window.agentAPI.fetchAgents();
        renderAgentGrid();
        if (window.updateAgentCount) {
            window.updateAgentCount(currentAgents.length);
        }
        updateRadar();
    } catch (error) {
        console.error('Failed to fetch agents:', error);
        currentAgents = [];
        renderEmptyState();
    }
}

function renderAgentGrid() {
    const grid = document.getElementById('agentGrid');
    if (!grid) return;
    
    grid.innerHTML = '';
    
    if (currentAgents.length === 0) {
        renderEmptyState();
        return;
    }
    
    grid.classList.remove('empty');
    
    currentAgents.forEach((agent, index) => {
        const card = createAgentCard(agent);
        const cardElement = document.createElement('div');
        cardElement.innerHTML = card;
        cardElement.firstElementChild.style.animationDelay = `${index * 0.1}s`;
        grid.appendChild(cardElement.firstElementChild);
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

function updateRadar() {
    if (agentRadar) {
        agentRadar.updateAgents(currentAgents);
    }
}

function startRealtimeUpdates() {
    // Refresh agent data every 30 seconds
    setInterval(fetchAndRenderAgents, 30000);
}

function createAgentCard(agent) {
    return `
        <div class="agent-card" data-agent-id="${agent.id}">
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
                    ${agent.status}
                </span>
                <span style="color: var(--text-secondary); font-size: 12px;">
                    <i data-lucide="clock"></i>
                    Uptime: ${agent.uptime || 'N/A'}
                </span>
            </div>
            
            <div class="agent-metrics-mini">
                <div class="metric-mini">
                    <span>PID:</span> ${agent.pid || 'N/A'}
                </div>
                <div class="metric-mini">
                    <span>Started:</span> ${formatTime(agent.start_time)}
                </div>
            </div>
            
            <div class="agent-controls">
                <a href="/agent.html?id=${agent.id}" class="control-btn">
                    <i data-lucide="eye"></i>
                    DETAILS
                </a>
                <button class="control-btn" onclick="quickStop('${agent.id}')">
                    <i data-lucide="square"></i>
                    STOP
                </button>
                <a href="/logs.html?agent=${agent.id}" class="control-btn">
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
        }
    }
});