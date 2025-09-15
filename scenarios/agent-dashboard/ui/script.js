// Agent Dashboard - Interactive Script

// Global agents array (populated from API)
let currentAgents = [];
let agentRadar = null;

// Initialize dashboard
document.addEventListener('DOMContentLoaded', async () => {
    // Initialize radar visualization
    initializeRadar();
    
    await fetchAndRenderAgents();
    startRealtimeUpdates();
    initializeTerminal();
});

// Initialize the radar visualization
function initializeRadar() {
    agentRadar = new AgentRadar('agentRadar');
    console.log('🎯 Agent radar initialized');
}

// Fetch agents from API and render
async function fetchAndRenderAgents() {
    try {
        addTerminalLog('Fetching agent status from resources...', 'info');
        const agents = await fetchAgentStatus();
        currentAgents = agents;
        renderAgents();
        updateAgentCount(agents.length);
        addTerminalLog(`✓ Connected to ${agents.length} agents across ${getUniqueResourceCount(agents)} resources`, 'success');
    } catch (error) {
        addTerminalLog(`Failed to fetch agents: ${error.message}`, 'error');
        currentAgents = [];
        updateAgentCount(0);
        renderAgents();
    }
}

// Render agent cards
function renderAgents() {
    const grid = document.getElementById('agentGrid');
    grid.innerHTML = '';
    
    if (currentAgents.length === 0) {
        grid.innerHTML = '<div class="no-agents">No active agents found. Agents will appear here when resource services start agents.</div>';
        
        // Update radar with empty data
        if (agentRadar) {
            agentRadar.updateAgents([]);
        }
        return;
    }
    
    currentAgents.forEach((agent, index) => {
        const card = createAgentCard(agent);
        card.style.animationDelay = `${index * 0.1}s`;
        grid.appendChild(card);
    });
    
    // Update radar with current agents
    if (agentRadar) {
        agentRadar.updateAgents(currentAgents);
    }
}

function getUniqueResourceCount(agents) {
    const resourceTypes = new Set(agents.map(agent => agent.type));
    return resourceTypes.size;
}

// Create agent card element
function createAgentCard(agent) {
    const card = document.createElement('div');
    card.className = 'agent-card';
    card.innerHTML = `
        <div class="agent-header">
            <div class="agent-name">${agent.name}</div>
            <div class="agent-type">${agent.type.toUpperCase()}</div>
        </div>
        
        <div class="agent-status">
            <span class="status-badge ${agent.status}">${agent.status}</span>
            <span style="color: var(--text-secondary); font-size: 12px;">
                Last heartbeat: ${getTimeSince(agent.last_heartbeat)}
            </span>
        </div>
        
        <div class="agent-metrics">
            <div class="metric">
                <div class="metric-label">PID</div>
                <div class="metric-value">${agent.metrics.pid || 'N/A'}</div>
            </div>
            <div class="metric">
                <div class="metric-label">Start Time</div>
                <div class="metric-value">${agent.metrics.start_time ? agent.metrics.start_time.split(' ')[1] : 'N/A'}</div>
            </div>
            <div class="metric">
                <div class="metric-label">Uptime</div>
                <div class="metric-value">${agent.metrics.uptime || 'N/A'}</div>
            </div>
            <div class="metric">
                <div class="metric-label">Resource</div>
                <div class="metric-value">${agent.type}</div>
            </div>
        </div>
        
        <div class="agent-controls">
            <button class="control-btn" onclick="controlAgent('${agent.id}', 'start')">
                ${agent.status === 'inactive' ? 'START' : 'RESTART'}
            </button>
            <button class="control-btn" onclick="controlAgent('${agent.id}', 'stop')">
                STOP
            </button>
            <button class="control-btn" onclick="controlAgent('${agent.id}', 'logs')">
                LOGS
            </button>
        </div>
    `;
    
    // Add hover effect
    card.addEventListener('mouseenter', () => {
        addTerminalLog(`Inspecting agent: ${agent.name}`);
    });
    
    return card;
}

// Calculate time since last heartbeat
function getTimeSince(timestamp) {
    const now = new Date();
    const then = new Date(timestamp);
    const diff = Math.floor((now - then) / 1000);
    
    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return `${Math.floor(diff / 86400)}d ago`;
}

// Control agent actions
async function controlAgent(agentId, action) {
    const agent = currentAgents.find(a => a.id === agentId);
    if (!agent) return;
    
    addTerminalLog(`Executing ${action.toUpperCase()} on ${agent.name} (${agent.type})...`);
    
    try {
        switch(action) {
            case 'start':
                // For real implementation, would call resource API to start agent
                addTerminalLog(`Note: Agent control requires direct resource management`, 'warning');
                addTerminalLog(`Use: resource-${agent.type} agents start`, 'info');
                break;
            case 'stop':
                // For real implementation, would call resource API to stop agent  
                addTerminalLog(`Note: Agent control requires direct resource management`, 'warning');
                addTerminalLog(`Use: resource-${agent.type} agents stop ${agentId}`, 'info');
                break;
            case 'logs':
                addTerminalLog(`[${agent.name}] Agent ID: ${agent.id}`);
                addTerminalLog(`[${agent.name}] PID: ${agent.metrics.pid}`);
                addTerminalLog(`[${agent.name}] Command: ${agent.metrics.command}`);
                addTerminalLog(`[${agent.name}] Uptime: ${agent.metrics.uptime}`);
                break;
        }
    } catch (error) {
        addTerminalLog(`Failed to ${action} ${agent.name}: ${error.message}`, 'error');
    }
}

// Update agent count in header
function updateAgentCount(count) {
    const agentCountElement = document.getElementById('agentCount');
    if (agentCountElement) {
        const statusDot = agentCountElement.parentElement.querySelector('.status-dot');
        
        agentCountElement.textContent = `${count} AGENT${count !== 1 ? 'S' : ''} ACTIVE`;
        
        // Update status dot color based on agent count
        statusDot.className = 'status-dot ' + (count > 0 ? 'active' : 'warning');
    }
}

// Terminal logging
function addTerminalLog(message, type = 'default') {
    const terminal = document.getElementById('terminalOutput');
    const line = document.createElement('div');
    line.className = 'terminal-line';
    
    let prefix = '$>';
    let color = 'var(--primary-cyan)';
    
    switch(type) {
        case 'success':
            color = 'var(--accent-green)';
            break;
        case 'warning':
            color = 'var(--accent-orange)';
            break;
        case 'error':
            color = 'var(--accent-red)';
            prefix = '!>';
            break;
        case 'info':
            color = 'var(--primary-magenta)';
            break;
    }
    
    line.innerHTML = `<span class="terminal-prompt" style="color: ${color}">${prefix}</span> ${message}`;
    terminal.appendChild(line);
    
    // Auto-scroll to bottom
    terminal.scrollTop = terminal.scrollHeight;
    
    // Limit terminal history
    if (terminal.children.length > 50) {
        terminal.removeChild(terminal.children[0]);
    }
}

// Initialize terminal with welcome message
function initializeTerminal() {
    setTimeout(() => {
        addTerminalLog('Agent Command Center v3.0.0 initialized', 'success');
        addTerminalLog('Unified Resource Agent Tracking System', 'info');
    }, 1000);
}

// Real-time updates from API
function startRealtimeUpdates() {
    // Refresh agent data every 30 seconds
    setInterval(async () => {
        await fetchAndRenderAgents();
    }, 30000);
    
    // Status updates in terminal
    setInterval(() => {
        const activeAgents = currentAgents.filter(a => a.status === 'active');
        const resourceCount = getUniqueResourceCount(currentAgents);
        const messages = [
            `System health check: ${activeAgents.length} agents across ${resourceCount} resources`,
            `Active resources: ${[...new Set(activeAgents.map(a => a.type))].join(', ')}`,
            `Monitoring agent registry files across 18 resources`,
        ];
        
        if (messages.length > 0) {
            addTerminalLog(messages[Math.floor(Math.random() * messages.length)], 'info');
        }
    }, 15000);
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey || e.metaKey) {
        switch(e.key) {
            case 'r':
                e.preventDefault();
                addTerminalLog('Refreshing agent status...', 'info');
                fetchAndRenderAgents();
                break;
            case 'l':
                e.preventDefault();
                document.getElementById('terminalOutput').scrollTop = 
                    document.getElementById('terminalOutput').scrollHeight;
                break;
        }
    }
});

// API Integration - fetch real agent data
async function fetchAgentStatus() {
    try {
        // Use injected API_PORT from server
        const apiPort = window.API_PORT || '20000';
        const response = await fetch(`http://localhost:${apiPort}/api/v1/agents`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        const data = await response.json();
        
        // New API format returns AgentsResponse directly
        // { agents: [], last_scan: "", scan_in_progress: bool, errors: [] }
        if (data.errors && data.errors.length > 0) {
            console.warn('Agent discovery errors:', data.errors);
            // Log errors but don't fail completely
            data.errors.forEach(error => {
                addTerminalLog(`⚠ ${error.resource_name}: ${error.error}`, 'warn');
            });
        }
        
        addTerminalLog(`Last scan: ${new Date(data.last_scan).toLocaleTimeString()}, ${data.scan_in_progress ? 'scan in progress' : 'scan complete'}`, 'info');
        
        return data.agents || [];
    } catch (error) {
        console.error('Failed to fetch agent status:', error);
        throw error;
    }
}

// WebSocket connection for real-time updates (placeholder)
function connectWebSocket() {
    // const ws = new WebSocket('ws://localhost:8100/ws');
    // ws.onmessage = (event) => {
    //     const data = JSON.parse(event.data);
    //     handleRealtimeUpdate(data);
    // };
}