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
    initializeLogsModal();
});

// Initialize the radar visualization
function initializeRadar() {
    agentRadar = new AgentRadar('agentRadar');
    console.log('🎯 Agent radar initialized');
}

// Fetch agents from API
async function fetchAgentStatus() {
    const apiPort = window.API_PORT || '15000';
    const response = await fetch(`http://localhost:${apiPort}/api/v1/agents`);
    
    if (!response.ok) {
        throw new Error(`API responded with status: ${response.status}`);
    }
    
    const data = await response.json();
    return data.agents || [];
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
        console.log('Fetch error details:', error);
        addTerminalLog(`Failed to fetch agents: ${error.message}`, 'error');
        if (error.message.includes('CORS') || error.message.includes('fetch')) {
            addTerminalLog('API may not be running or CORS issue detected', 'warning');
        }
        currentAgents = [];
        updateAgentCount(0);
        renderAgents();
    }
}

// Render agent cards
function renderAgents() {
    const grid = document.getElementById('agentGrid');
    
    if (!grid) {
        console.error('agentGrid element not found');
        return;
    }
    
    grid.innerHTML = '';
    
    if (currentAgents.length === 0) {
        console.log('No agents found, showing empty state');
        grid.classList.add('empty');
        grid.innerHTML = '<div class="no-agents">No active agents found. Agents will appear here when resource services start agents.</div>';
        
        // Update radar with empty data
        if (agentRadar) {
            agentRadar.updateAgents([]);
        }
        return;
    }
    
    console.log(`Rendering ${currentAgents.length} agents`);
    // Remove empty class if agents exist
    grid.classList.remove('empty');
    
    currentAgents.forEach((agent, index) => {
        const card = createAgentCard(agent);
        card.style.animationDelay = `${index * 0.1}s`;
        grid.appendChild(card);
    });
    
    // Initialize Lucide icons for dynamically created content
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
    
    // Update radar with current agents
    if (agentRadar) {
        agentRadar.updateAgents(currentAgents);
    }
}

function getUniqueResourceCount(agents) {
    const resourceTypes = new Set(agents.map(agent => agent.type));
    return resourceTypes.size;
}

// Update agent count in header
function updateAgentCount(count) {
    const agentCountElement = document.getElementById('agentCount');
    if (agentCountElement) {
        agentCountElement.textContent = `${count} AGENTS ACTIVE`;
    } else {
        console.warn('agentCount element not found in DOM');
    }
}

// Start real-time updates
function startRealtimeUpdates() {
    // Refresh agent data every 30 seconds
    setInterval(fetchAndRenderAgents, 30000);
}

// Create agent card element
function createAgentCard(agent) {
    const card = document.createElement('div');
    card.className = 'agent-card';
    card.innerHTML = `
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
                Last heartbeat: ${getTimeSince(agent.last_heartbeat)}
            </span>
        </div>
        
        <div class="agent-metrics">
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="hash"></i>
                    PID
                </div>
                <div class="metric-value">${agent.metrics.pid || 'N/A'}</div>
            </div>
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="clock"></i>
                    Start Time
                </div>
                <div class="metric-value">${agent.metrics.start_time ? agent.metrics.start_time.split(' ')[1] : 'N/A'}</div>
            </div>
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="timer"></i>
                    Uptime
                </div>
                <div class="metric-value">${agent.metrics.uptime || 'N/A'}</div>
            </div>
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="server"></i>
                    Resource
                </div>
                <div class="metric-value">${agent.type}</div>
            </div>
        </div>
        
        <div class="agent-controls">
            <button class="control-btn" onclick="controlAgent('${agent.id}', 'start')">
                <i data-lucide="${agent.status === 'inactive' ? 'play' : 'rotate-cw'}"></i>
                ${agent.status === 'inactive' ? 'START' : 'RESTART'}
            </button>
            <button class="control-btn" onclick="controlAgent('${agent.id}', 'stop')">
                <i data-lucide="square"></i>
                STOP
            </button>
            <button class="control-btn" onclick="controlAgent('${agent.id}', 'logs')">
                <i data-lucide="scroll-text"></i>
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
        // Update the text content
        agentCountElement.textContent = `${count} AGENT${count !== 1 ? 'S' : ''} ACTIVE`;
        
        // Update the status icon color (we replaced status-dot with status-icon)
        const statusIcon = agentCountElement.parentElement.querySelector('.status-icon');
        if (statusIcon) {
            // Remove old status classes
            statusIcon.classList.remove('active', 'warning', 'error');
            // Add new status class based on agent count
            statusIcon.classList.add(count > 0 ? 'active' : 'warning');
        }
    } else {
        console.warn('agentCount element not found in DOM');
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

// Initialize logs modal functionality
function initializeLogsModal() {
    const logsButton = document.getElementById('logsButton');
    const logsModal = document.getElementById('logsModal');
    const logsCloseBtn = document.getElementById('logsCloseBtn');
    
    // Open modal when logs button is clicked
    logsButton.addEventListener('click', () => {
        logsModal.classList.add('show');
        addTerminalLog('Logs viewer opened', 'info');
    });
    
    // Close modal when close button is clicked
    logsCloseBtn.addEventListener('click', () => {
        logsModal.classList.remove('show');
    });
    
    // Close modal when clicking outside the modal content
    logsModal.addEventListener('click', (e) => {
        if (e.target === logsModal) {
            logsModal.classList.remove('show');
        }
    });
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
    const logsModal = document.getElementById('logsModal');
    
    // Handle Escape key for modal
    if (e.key === 'Escape' && logsModal && logsModal.classList.contains('show')) {
        logsModal.classList.remove('show');
        return;
    }
    
    if (e.ctrlKey || e.metaKey) {
        switch(e.key) {
            case 'r':
                e.preventDefault();
                addTerminalLog('Refreshing agent status...', 'info');
                fetchAndRenderAgents();
                break;
            case 'l':
                e.preventDefault();
                // Open logs modal instead of just scrolling
                if (logsModal) {
                    logsModal.classList.add('show');
                    addTerminalLog('Logs viewer opened (Ctrl+L)', 'info');
                }
                break;
        }
    }
});

// Remove duplicate function - using the one defined earlier in the file

// WebSocket connection for real-time updates (placeholder)
function connectWebSocket() {
    // const ws = new WebSocket('ws://localhost:8100/ws');
    // ws.onmessage = (event) => {
    //     const data = JSON.parse(event.data);
    //     handleRealtimeUpdate(data);
    // };
}