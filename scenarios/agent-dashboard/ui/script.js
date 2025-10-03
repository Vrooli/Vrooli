// Agent Dashboard - Interactive Script

// Global agents array (populated from API)
let currentAgents = [];
let agentSummary = {
    total: 0,
    running: 0,
    completed: 0,
    failed: 0,
    stopped: 0,
    timestamp: null,
};
let agentRadar = null;
let performanceHistory = null;
let activeLogViewers = [];

const getApiPort = () => window.API_PORT || '15000';
const getApiBaseUrl = () => `http://localhost:${getApiPort()}/api/v1`;

// Initialize dashboard
document.addEventListener('DOMContentLoaded', async () => {
    // Initialize radar visualization
    initializeRadar();
    
    // Initialize performance history tracking
    initializePerformanceHistory();
    
    await fetchAndRenderAgents();
    startRealtimeUpdates();
    initializeTerminal();
    initializeLogsModal();
    initializeSearchAndFilter();

    const launchBtn = document.getElementById('launchAgentBtn');
    if (launchBtn) {
        launchBtn.addEventListener('click', () => launchCodexAgent());
    }
});

// Initialize the radar visualization
function initializeRadar() {
    agentRadar = new AgentRadar('agentRadar');
    console.log('🎯 Agent radar initialized');
    
    // Setup radar theme selector after radar is initialized
    const radarThemeSelect = document.getElementById('radarTheme');
    if (radarThemeSelect && agentRadar) {
        radarThemeSelect.addEventListener('change', (e) => {
            const selectedTheme = e.target.value;
            if (agentRadar && agentRadar.setTheme) {
                agentRadar.setTheme(selectedTheme);
                // Save theme preference
                localStorage.setItem('radarTheme', selectedTheme);
                addTerminalLog(`Radar theme changed to: ${selectedTheme}`, 'info');
            }
        });
        
        // Load saved theme preference
        const savedTheme = localStorage.getItem('radarTheme');
        if (savedTheme && agentRadar.setTheme) {
            radarThemeSelect.value = savedTheme;
            agentRadar.setTheme(savedTheme);
        }
    }
}

// Initialize performance history tracking
function initializePerformanceHistory() {
    performanceHistory = new PerformanceHistory();
    console.log('📊 Performance history tracking initialized');
}

// Fetch snapshot from API
async function fetchAgentSnapshot() {
    const response = await fetch(`${getApiBaseUrl()}/agents`);

    if (!response.ok) {
        throw new Error(`API responded with status: ${response.status}`);
    }

    return await response.json();
}

// Fetch agents from API and render
async function fetchAndRenderAgents() {
    try {
        addTerminalLog('Fetching Codex agent snapshot...', 'info');
        const snapshot = await fetchAgentSnapshot();

        currentAgents = snapshot.agents || [];
        agentSummary = {
            total: snapshot.total ?? currentAgents.length,
            running: snapshot.running ?? currentAgents.filter(a => a.status === 'running').length,
            completed: snapshot.completed ?? 0,
            failed: snapshot.failed ?? 0,
            stopped: snapshot.stopped ?? 0,
            timestamp: snapshot.timestamp || new Date().toISOString(),
        };

        updateCapabilityFilter();

        if (performanceHistory) {
            performanceHistory.updateFromAgents(currentAgents);
        }

        renderAgents();
        updateAgentCount(agentSummary);

        const runningCount = agentSummary.running;
        addTerminalLog(`✓ Snapshot updated: ${runningCount} running, ${agentSummary.total} total Codex agents`, 'success');
    } catch (error) {
        console.log('Fetch error details:', error);
        addTerminalLog(`Failed to fetch agents: ${error.message}`, 'error');
        addTerminalLog('Ensure the Agent Dashboard API is running and accessible.', 'warning');
        currentAgents = [];
        agentSummary = {
            total: 0,
            running: 0,
            completed: 0,
            failed: 0,
            stopped: 0,
            timestamp: null,
        };
        updateAgentCount(agentSummary);
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
function updateAgentCount(summary) {
    const agentCountElement = document.getElementById('agentCount');
    if (!agentCountElement) {
        console.warn('agentCount element not found in DOM');
        return;
    }

    const total = typeof summary === 'number' ? summary : summary?.total ?? 0;
    const running = typeof summary === 'number' ? summary : summary?.running ?? 0;

    agentCountElement.textContent = `${running} RUNNING / ${total} TOTAL`;

    const statusIcon = agentCountElement.parentElement?.querySelector('.status-icon');
    if (statusIcon) {
        statusIcon.classList.remove('active', 'warning', 'error');
        if (running > 0) {
            statusIcon.classList.add('active');
        } else if (total > 0) {
            statusIcon.classList.add('warning');
        } else {
            statusIcon.classList.add('error');
        }
    }
}

// Start real-time updates
function startRealtimeUpdates() {
    // Refresh agent data every 30 seconds
    setInterval(fetchAndRenderAgents, 30000);
}

// Initialize search and filter functionality
function initializeSearchAndFilter() {
    const searchInput = document.getElementById('agentSearch');
    const searchClear = document.getElementById('searchClear');
    const sortSelect = document.getElementById('agentSort');
    const typeFilter = document.getElementById('typeFilter');
    const statusFilter = document.getElementById('statusFilter');
    const capabilityFilter = document.getElementById('capabilityFilter');

    // Search functionality
    if (searchInput) {
        searchInput.addEventListener('input', (e) => {
            const searchTerm = e.target.value.toLowerCase();
            if (searchClear) {
                searchClear.style.display = searchTerm ? 'block' : 'none';
            }
            filterAndSortAgents();
        });
    }

    // Clear search button
    if (searchClear) {
        searchClear.addEventListener('click', () => {
            if (searchInput) {
                searchInput.value = '';
                searchClear.style.display = 'none';
                filterAndSortAgents();
            }
        });
    }

    // Sort functionality
    if (sortSelect) {
        sortSelect.addEventListener('change', filterAndSortAgents);
    }

    // Type filter
    if (typeFilter) {
        typeFilter.addEventListener('change', filterAndSortAgents);
    }

    // Status filter
    if (statusFilter) {
        statusFilter.addEventListener('change', filterAndSortAgents);
    }

    // Capability filter
    if (capabilityFilter) {
        capabilityFilter.addEventListener('change', filterAndSortAgents);
    }
}

// Filter and sort agents based on current UI controls
function filterAndSortAgents() {
    const searchInput = document.getElementById('agentSearch');
    const sortSelect = document.getElementById('agentSort');
    const typeFilter = document.getElementById('typeFilter');
    const statusFilter = document.getElementById('statusFilter');
    const capabilityFilter = document.getElementById('capabilityFilter');
    
    let filteredAgents = [...currentAgents];
    
    // Apply search filter
    if (searchInput && searchInput.value) {
        const searchTerm = searchInput.value.toLowerCase();
        filteredAgents = filteredAgents.filter(agent => {
            return agent.name.toLowerCase().includes(searchTerm) ||
                   agent.type.toLowerCase().includes(searchTerm) ||
                   agent.id.toLowerCase().includes(searchTerm) ||
                   (agent.command && agent.command.toLowerCase().includes(searchTerm));
        });
    }
    
    // Apply type filter
    if (typeFilter && typeFilter.value !== 'all') {
        filteredAgents = filteredAgents.filter(agent => agent.type === typeFilter.value);
    }
    
    // Apply status filter
    if (statusFilter && statusFilter.value !== 'all') {
        filteredAgents = filteredAgents.filter(agent => (agent.status || '').toLowerCase() === statusFilter.value.toLowerCase());
    }
    
    // Apply capability filter
    if (capabilityFilter && capabilityFilter.value) {
        filteredAgents = filteredAgents.filter(agent => {
            if (!agent.capabilities || agent.capabilities.length === 0) return false;
            return agent.capabilities.some(cap => 
                cap.toLowerCase().includes(capabilityFilter.value.toLowerCase())
            );
        });
    }
    
    // Apply sorting
    if (sortSelect) {
        const sortOption = sortSelect.value;
        filteredAgents.sort((a, b) => {
            switch (sortOption) {
                case 'name-asc':
                    return a.name.localeCompare(b.name);
                case 'name-desc':
                    return b.name.localeCompare(a.name);
                case 'type-asc':
                    return a.type.localeCompare(b.type);
                case 'type-desc':
                    return b.type.localeCompare(a.type);
                case 'status-running':
                    return (b.status === 'running' ? 1 : 0) - (a.status === 'running' ? 1 : 0);
                case 'status-completed':
                    return (b.status === 'completed' ? 1 : 0) - (a.status === 'completed' ? 1 : 0);
                case 'uptime-newest':
                    return new Date(b.start_time) - new Date(a.start_time);
                case 'uptime-oldest':
                    return new Date(a.start_time) - new Date(b.start_time);
                case 'memory-high':
                    return (b.metrics?.memory_mb || 0) - (a.metrics?.memory_mb || 0);
                case 'memory-low':
                    return (a.metrics?.memory_mb || 0) - (b.metrics?.memory_mb || 0);
                default:
                    return 0;
            }
        });
    }
    
    // Render filtered agents
    renderFilteredAgents(filteredAgents);
}

// Render filtered agents
function renderFilteredAgents(agents) {
    const grid = document.querySelector('.agents-grid') || document.getElementById('agentsGrid');
    if (!grid) {
        console.warn('Agents grid element not found');
        return;
    }
    
    // Clear current display
    grid.innerHTML = '';
    
    if (!agents || agents.length === 0) {
        grid.classList.add('empty');
        grid.innerHTML = '<div class="no-agents">No agents match your search criteria.</div>';
        
        // Update radar with empty data
        if (agentRadar) {
            agentRadar.updateAgents([]);
        }
        return;
    }
    
    // Remove empty class
    grid.classList.remove('empty');
    
    agents.forEach((agent, index) => {
        const card = createAgentCard(agent);
        card.style.animationDelay = `${index * 0.1}s`;
        grid.appendChild(card);
    });
    
    // Update Lucide icons
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
    
    // Update radar with filtered agents
    if (agentRadar) {
        agentRadar.updateAgents(agents);
    }
    
    // Update count summary
    updateAgentCount({
        total: agents.length,
        running: agents.filter(a => a.status === 'running').length,
    });
}

// Create agent card element
function createAgentCard(agent) {
    const card = document.createElement('div');
    card.className = 'agent-card';
    const status = (agent.status || 'unknown').toLowerCase();
    const statusLabel = formatStatus(status);
    const statusIcon = getStatusIcon(status);
    const lastSeen = agent.last_seen || agent.lastSeen;
    const uptime = agent.uptime || 'N/A';
    const pid = agent.pid || 'N/A';
    const startTime = formatTimestamp(agent.start_time || agent.startTime);
    const modeLabel = agent.mode ? agent.mode.toUpperCase() : 'AUTO';
    const exitCode = agent.exit_code ?? null;
    const cpuUsage = agent.metrics?.cpu_percent;
    const memoryMb = agent.metrics?.memory_mb;
    const taskPreview = agent.task ? truncateText(agent.task, 160) : 'No task metadata available.';

    card.innerHTML = `
        <div class="agent-header">
            <div class="agent-name">
                <i data-lucide="bot"></i>
                ${agent.name}
            </div>
            <div class="agent-type" title="Execution mode: ${modeLabel}">
                ${agent.type.toUpperCase()} · ${modeLabel}
            </div>
        </div>
        
        <div class="agent-status">
            <span class="status-badge ${status}">
                <i data-lucide="${statusIcon}"></i>
                ${statusLabel}
            </span>
            <span style="color: var(--text-secondary); font-size: 12px;">
                <i data-lucide="clock"></i>
                Last heartbeat: ${getTimeSince(lastSeen)}
            </span>
        </div>
        
        <div class="agent-metrics">
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="hash"></i>
                    PID
                </div>
                <div class="metric-value">${pid}</div>
            </div>
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="clock"></i>
                    Start Time
                </div>
                <div class="metric-value">${startTime}</div>
            </div>
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="timer"></i>
                    Uptime
                </div>
                <div class="metric-value">${uptime}</div>
            </div>
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="server"></i>
                    Resource
                </div>
                <div class="metric-value">${agent.type}</div>
            </div>
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="activity"></i>
                    CPU
                </div>
                <div class="metric-value">${cpuUsage !== undefined && cpuUsage !== null ? `${cpuUsage.toFixed(1)}%` : 'N/A'}</div>
            </div>
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="database"></i>
                    Memory
                </div>
                <div class="metric-value">${memoryMb !== undefined && memoryMb !== null ? `${memoryMb.toFixed(1)} MB` : 'N/A'}</div>
            </div>
            ${exitCode !== null ? `
            <div class="metric">
                <div class="metric-label">
                    <i data-lucide="check-circle"></i>
                    Exit Code
                </div>
                <div class="metric-value">${exitCode}</div>
            </div>` : ''}
        </div>
        
        <div class="agent-performance" id="perf-${agent.id}">
            <div class="performance-row">
                <span class="perf-label">CPU:</span>
                <div class="perf-sparkline">${performanceHistory ? performanceHistory.createSparkline(agent.id, 'cpu', 120, 25) : ''}</div>
            </div>
            <div class="performance-row">
                <span class="perf-label">MEM:</span>
                <div class="perf-sparkline">${performanceHistory ? performanceHistory.createSparkline(agent.id, 'memory', 120, 25) : ''}</div>
            </div>
        </div>

        <div class="agent-task">
            <div class="task-label"><i data-lucide="workflow"></i> Task</div>
            <div class="task-content" title="${agent.task || ''}">${taskPreview}</div>
        </div>
        
        <div class="agent-controls">
            <button class="control-btn" onclick="controlAgent('${agent.id}', 'start')" ${status === 'running' ? 'disabled' : ''}>
                <i data-lucide="${status === 'running' ? 'play-circle' : 'refresh-cw'}"></i>
                ${status === 'running' ? 'RUNNING' : 'REPLAY'}
            </button>
            <button class="control-btn" onclick="controlAgent('${agent.id}', 'stop')" ${status !== 'running' ? 'disabled' : ''}>
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

function formatStatus(status) {
    switch (status) {
        case 'running':
            return 'RUNNING';
        case 'starting':
            return 'STARTING';
        case 'completed':
            return 'COMPLETED';
        case 'failed':
            return 'FAILED';
        case 'timeout':
            return 'TIMEOUT';
        case 'stopped':
            return 'STOPPED';
        default:
            return (status || 'unknown').toUpperCase();
    }
}

function getStatusIcon(status) {
    switch (status) {
        case 'running':
            return 'zap';
        case 'starting':
            return 'loader';
        case 'completed':
            return 'check-circle';
        case 'failed':
            return 'alert-octagon';
        case 'timeout':
            return 'hourglass';
        case 'stopped':
            return 'pause-circle';
        default:
            return 'help-circle';
    }
}

function truncateText(text, length) {
    if (!text) return '';
    if (text.length <= length) return text;
    return `${text.slice(0, length - 3)}...`;
}

function formatTimestamp(value) {
    if (!value) return 'N/A';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return value;
    }
    return `${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
}

// Calculate time since last heartbeat
function getTimeSince(timestamp) {
    if (!timestamp) return 'unknown';
    const now = new Date();
    const then = new Date(timestamp);
    if (Number.isNaN(then.getTime())) return 'unknown';
    const diff = Math.floor((now - then) / 1000);
    
    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return `${Math.floor(diff / 86400)}d ago`;
}

// Update capability filter options based on discovered agents
function updateCapabilityFilter() {
    const capabilityFilter = document.getElementById('capabilityFilter');
    if (!capabilityFilter) return;
    
    // Collect all unique capabilities from all agents
    const allCapabilities = new Set();
    currentAgents.forEach(agent => {
        if (agent.capabilities && Array.isArray(agent.capabilities)) {
            agent.capabilities.forEach(cap => allCapabilities.add(cap));
        }
    });
    
    // Sort capabilities alphabetically
    const sortedCapabilities = Array.from(allCapabilities).sort();
    
    // Preserve current selection
    const currentValue = capabilityFilter.value;
    
    // Rebuild options
    capabilityFilter.innerHTML = '<option value="">All Capabilities</option>';
    sortedCapabilities.forEach(cap => {
        const option = document.createElement('option');
        option.value = cap;
        option.textContent = cap.charAt(0).toUpperCase() + cap.slice(1).replace(/-/g, ' ');
        capabilityFilter.appendChild(option);
    });
    
    // Restore selection if it still exists
    if (currentValue && sortedCapabilities.includes(currentValue)) {
        capabilityFilter.value = currentValue;
    }
}

// Control agent actions
async function controlAgent(agentId, action) {
    const agent = currentAgents.find(a => a.id === agentId);
    if (!agent) {
        addTerminalLog(`Agent ${agentId} not found in current snapshot`, 'error');
        return;
    }

    addTerminalLog(`Executing ${action.toUpperCase()} on ${agent.name} (${agent.type})...`, 'info');

    try {
        switch (action) {
            case 'start':
                await launchCodexAgent(agent);
                break;
            case 'stop':
                if (agent.status !== 'running') {
                    addTerminalLog(`Agent ${agent.name} is not running`, 'warning');
                    return;
                }
                await sendStopSignal(agentId, agent.name);
                break;
            case 'logs':
                if (typeof multiLogViewer !== 'undefined') {
                    multiLogViewer.createLogViewer(agent);
                } else {
                    const logs = await fetchAgentLogs(agentId, 50);
                    logs.forEach(line => addTerminalLog(`[${agent.name}] ${line}`));
                }
                break;
        }
    } catch (error) {
        addTerminalLog(`Failed to ${action} ${agent.name}: ${error.message}`, 'error');
    }
}

async function sendStopSignal(agentId, agentName) {
    const response = await fetch(`${getApiBaseUrl()}/agents/${encodeURIComponent(agentId)}/stop`, {
        method: 'POST',
    });
    const result = await response.json();
    if (!response.ok || result.success === false) {
        throw new Error(result.error || `HTTP ${response.status}`);
    }
    addTerminalLog(`Stop signal sent to ${agentName}`, 'info');
    await fetchAndRenderAgents();
}

async function fetchAgentLogs(agentId, lines = 100) {
    const response = await fetch(`${getApiBaseUrl()}/agents/${encodeURIComponent(agentId)}/logs?lines=${lines}`);
    if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
    }
    const payload = await response.json();
    const logs = payload.logs || payload.data?.logs || [];
    return Array.isArray(logs) ? logs : [];
}

async function launchCodexAgent(existingAgent = null) {
    let task = existingAgent?.task || '';
    let mode = (existingAgent?.mode || 'auto').toLowerCase();
    let label = existingAgent?.name || '';
    let timeoutSeconds = existingAgent?.timeout_seconds;
    let notes = existingAgent?.notes || '';

    if (!existingAgent) {
        task = prompt('Enter the Codex task to run:');
        if (!task) {
            addTerminalLog('Codex launch cancelled', 'warning');
            return;
        }
        const modeInput = prompt('Execution mode (auto/approve/always)', mode) || mode;
        mode = modeInput.toLowerCase();
        label = prompt('Optional label for this agent', label) || label;
        const timeoutInput = prompt('Timeout in seconds (blank for default)', timeoutSeconds ? String(timeoutSeconds) : '');
        if (timeoutInput && !Number.isNaN(Number(timeoutInput))) {
            timeoutSeconds = parseInt(timeoutInput, 10);
        }
        notes = prompt('Additional notes or context (optional)', notes) || notes;
    }

    const payload = {
        task,
        mode,
    };
    if (label) payload.label = label;
    if (timeoutSeconds && Number.isFinite(timeoutSeconds)) payload.timeout_seconds = timeoutSeconds;
    if (notes) payload.notes = notes;
    if (existingAgent?.capabilities?.length) payload.capabilities = existingAgent.capabilities;

    addTerminalLog('Launching Codex agent task...', 'info');

    const response = await fetch(`${getApiBaseUrl()}/agents`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
    });

    const result = await response.json();
    if (!response.ok || result.success === false) {
        throw new Error(result.error || `HTTP ${response.status}`);
    }

    const newAgent = result.data || {};
    addTerminalLog(`Codex agent ${newAgent.name || newAgent.id} started`, 'success');
    await fetchAndRenderAgents();
    return newAgent;
}

// Update agent count in header
// (duplicate definition removed; see top-level updateAgentCount)

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
        addTerminalLog('Agent Command Center v3.1.0 initialized', 'success');
        addTerminalLog('Codex orchestration channel engaged', 'info');
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
        const runningAgents = currentAgents.filter(a => a.status === 'running');
        const resourceCount = getUniqueResourceCount(currentAgents);
        const messages = [
            `System health: ${agentSummary.running} running / ${agentSummary.total} total agents`,
            `Active resources: ${[...new Set(runningAgents.map(a => a.type))].join(', ') || 'codex'}`,
            `Last snapshot: ${agentSummary.timestamp ? formatTimestamp(agentSummary.timestamp) : 'N/A'}`,
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
