// Agent Dashboard - Interactive Script

// Sample agent data (would come from API in production)
const sampleAgents = [
    {
        id: 'agent-001',
        name: 'Huginn Scraper',
        type: 'huginn',
        status: 'active',
        description: 'Web scraping and monitoring agent',
        capabilities: ['web-scraping', 'monitoring', 'alerts'],
        metrics: {
            cpu: 23,
            memory: 512,
            tasks_completed: 1247,
            uptime: '3d 14h'
        },
        last_heartbeat: new Date(Date.now() - 30000).toISOString()
    },
    {
        id: 'agent-002',
        name: 'Claude Code',
        type: 'claude-code',
        status: 'active',
        description: 'AI coding assistant',
        capabilities: ['code-generation', 'debugging', 'refactoring'],
        metrics: {
            cpu: 45,
            memory: 1024,
            tasks_completed: 89,
            uptime: '1d 7h'
        },
        last_heartbeat: new Date(Date.now() - 15000).toISOString()
    },
    {
        id: 'agent-003',
        name: 'N8N Workflow Engine',
        type: 'n8n-workflow',
        status: 'active',
        description: 'Workflow automation engine',
        capabilities: ['automation', 'integration', 'orchestration'],
        metrics: {
            cpu: 31,
            memory: 768,
            tasks_completed: 3456,
            uptime: '7d 2h'
        },
        last_heartbeat: new Date(Date.now() - 45000).toISOString()
    },
    {
        id: 'agent-004',
        name: 'Agent-S2 Browser',
        type: 'agent-s2',
        status: 'inactive',
        description: 'Browser automation agent',
        capabilities: ['browser-automation', 'testing', 'scraping'],
        metrics: {
            cpu: 0,
            memory: 0,
            tasks_completed: 567,
            uptime: '0h'
        },
        last_heartbeat: new Date(Date.now() - 3600000).toISOString()
    },
    {
        id: 'agent-005',
        name: 'Vision Analyzer',
        type: 'custom',
        status: 'error',
        description: 'Computer vision processing',
        capabilities: ['image-analysis', 'ocr', 'object-detection'],
        metrics: {
            cpu: 78,
            memory: 2048,
            tasks_completed: 234,
            uptime: '12h'
        },
        last_heartbeat: new Date(Date.now() - 600000).toISOString()
    }
];

// Initialize dashboard
document.addEventListener('DOMContentLoaded', () => {
    renderAgents();
    startRealtimeUpdates();
    initializeTerminal();
});

// Render agent cards
function renderAgents() {
    const grid = document.getElementById('agentGrid');
    grid.innerHTML = '';
    
    sampleAgents.forEach((agent, index) => {
        const card = createAgentCard(agent);
        card.style.animationDelay = `${index * 0.1}s`;
        grid.appendChild(card);
    });
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
                <div class="metric-label">CPU Usage</div>
                <div class="metric-value">${agent.metrics.cpu}%</div>
            </div>
            <div class="metric">
                <div class="metric-label">Memory</div>
                <div class="metric-value">${agent.metrics.memory}MB</div>
            </div>
            <div class="metric">
                <div class="metric-label">Tasks</div>
                <div class="metric-value">${agent.metrics.tasks_completed}</div>
            </div>
            <div class="metric">
                <div class="metric-label">Uptime</div>
                <div class="metric-value">${agent.metrics.uptime}</div>
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
    const agent = sampleAgents.find(a => a.id === agentId);
    if (!agent) return;
    
    addTerminalLog(`Executing ${action.toUpperCase()} on ${agent.name}...`);
    
    // Simulate API call
    setTimeout(() => {
        switch(action) {
            case 'start':
                agent.status = 'active';
                addTerminalLog(`✓ ${agent.name} started successfully`, 'success');
                break;
            case 'stop':
                agent.status = 'inactive';
                addTerminalLog(`✓ ${agent.name} stopped`, 'warning');
                break;
            case 'logs':
                addTerminalLog(`[${agent.name}] Processing request...`);
                addTerminalLog(`[${agent.name}] Task completed: ID-${Math.random().toString(36).substr(2, 9)}`);
                addTerminalLog(`[${agent.name}] Memory usage: ${agent.metrics.memory}MB`);
                break;
        }
        renderAgents();
    }, 1000);
}

// Orchestration functions
async function orchestrateAgents(mode) {
    addTerminalLog(`Initiating ${mode.toUpperCase()} orchestration protocol...`, 'info');
    
    // Simulate orchestration
    setTimeout(() => {
        switch(mode) {
            case 'optimize':
                addTerminalLog('Analyzing resource utilization...', 'info');
                setTimeout(() => {
                    addTerminalLog('✓ Resources optimized. Efficiency increased by 23%', 'success');
                }, 1500);
                break;
            case 'scale':
                addTerminalLog('Calculating optimal agent configuration...', 'info');
                setTimeout(() => {
                    addTerminalLog('✓ Auto-scaling complete. 2 new agents deployed', 'success');
                }, 1500);
                break;
            case 'balance':
                addTerminalLog('Redistributing workload...', 'info');
                setTimeout(() => {
                    addTerminalLog('✓ Load balanced across 5 active agents', 'success');
                }, 1500);
                break;
            case 'emergency':
                addTerminalLog('! EMERGENCY PROTOCOL ACTIVATED !', 'error');
                setTimeout(() => {
                    addTerminalLog('Isolating compromised agents...', 'warning');
                    setTimeout(() => {
                        addTerminalLog('✓ System secured. All agents operational', 'success');
                    }, 1000);
                }, 1000);
                break;
        }
    }, 500);
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
        addTerminalLog('Agent Command Center v2.0.1 initialized', 'success');
        addTerminalLog(`Connected to ${sampleAgents.length} agents`, 'info');
    }, 2000);
}

// Simulate real-time updates
function startRealtimeUpdates() {
    setInterval(() => {
        // Update heartbeats
        sampleAgents.forEach(agent => {
            if (agent.status === 'active') {
                agent.last_heartbeat = new Date().toISOString();
                
                // Simulate metric changes
                agent.metrics.cpu = Math.max(0, Math.min(100, 
                    agent.metrics.cpu + (Math.random() - 0.5) * 10));
                agent.metrics.memory = Math.max(256, Math.min(2048,
                    agent.metrics.memory + (Math.random() - 0.5) * 50));
                    
                // Occasionally increment task count
                if (Math.random() > 0.7) {
                    agent.metrics.tasks_completed++;
                }
            }
        });
        
        // Re-render agents
        renderAgents();
    }, 5000);
    
    // Occasional status updates in terminal
    setInterval(() => {
        const activeAgents = sampleAgents.filter(a => a.status === 'active');
        const messages = [
            `System health check: ${activeAgents.length} agents operational`,
            `Network latency: ${Math.floor(Math.random() * 50 + 10)}ms`,
            `Queue depth: ${Math.floor(Math.random() * 100)} tasks pending`,
            `Memory usage: ${Math.floor(Math.random() * 30 + 40)}% total`,
            `Throughput: ${Math.floor(Math.random() * 1000 + 500)} ops/sec`
        ];
        
        addTerminalLog(messages[Math.floor(Math.random() * messages.length)], 'info');
    }, 10000);
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey || e.metaKey) {
        switch(e.key) {
            case 'r':
                e.preventDefault();
                addTerminalLog('Refreshing agent status...', 'info');
                renderAgents();
                break;
            case 'o':
                e.preventDefault();
                orchestrateAgents('optimize');
                break;
            case 'l':
                e.preventDefault();
                document.getElementById('terminalOutput').scrollTop = 
                    document.getElementById('terminalOutput').scrollHeight;
                break;
        }
    }
});

// API Integration (placeholder for real implementation)
async function fetchAgentStatus() {
    try {
        const response = await fetch('/api/agents/status');
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('Failed to fetch agent status:', error);
        addTerminalLog('Warning: Using offline mode', 'warning');
        return sampleAgents;
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