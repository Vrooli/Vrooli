// Logs Page JavaScript
// Real-time log viewer with filtering and following

let currentAgents = [];
let selectedAgent = null;
let isFollowing = false;
let followInterval = null;
let logLines = [];

document.addEventListener('DOMContentLoaded', async () => {
    await loadAgents();
    setupControls();
    checkUrlParams();
});

async function loadAgents() {
    try {
        currentAgents = await window.agentAPI.fetchAgents();
        populateAgentSelect();
    } catch (error) {
        console.error('Failed to load agents:', error);
    }
}

function populateAgentSelect() {
    const select = document.getElementById('agentSelect');
    
    // Keep system logs option
    select.innerHTML = '<option value="system">System Logs</option>';
    
    // Add agents
    currentAgents.forEach(agent => {
        const option = document.createElement('option');
        option.value = agent.id;
        option.textContent = `${agent.name} (${agent.type})`;
        select.appendChild(option);
    });
}

function checkUrlParams() {
    const params = new URLSearchParams(window.location.search);
    const agentId = params.get('agent');
    
    if (agentId) {
        document.getElementById('agentSelect').value = agentId;
        loadLogs();
    }
}

function setupControls() {
    document.getElementById('agentSelect').addEventListener('change', loadLogs);
    document.getElementById('logLevel').addEventListener('change', filterLogs);
    document.getElementById('refreshBtn').addEventListener('click', loadLogs);
    document.getElementById('followBtn').addEventListener('click', toggleFollow);
    document.getElementById('clearBtn').addEventListener('click', clearLogs);
    document.getElementById('exportBtn').addEventListener('click', downloadLogs);
    document.getElementById('fullscreenBtn').addEventListener('click', toggleFullscreen);
    
    // Enter key in line count input
    document.getElementById('lineCount').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            loadLogs();
        }
    });
}

async function loadLogs() {
    const agentId = document.getElementById('agentSelect').value;
    const lineCount = document.getElementById('lineCount').value;
    
    if (agentId === 'system') {
        displaySystemLogs();
        return;
    }
    
    try {
        const response = await window.agentAPI.fetchAgentLogs(agentId, lineCount);
        logLines = response.logs || [];
        displayLogs(logLines);
        updateStats();
    } catch (error) {
        console.error('Failed to load logs:', error);
        addLogLine('Failed to load logs: ' + error.message, 'error');
    }
}

function displaySystemLogs() {
    // Display mock system logs for demonstration
    logLines = [
        { time: new Date().toISOString(), level: 'info', message: 'System initialized' },
        { time: new Date().toISOString(), level: 'info', message: 'Agent discovery scan started' },
        { time: new Date().toISOString(), level: 'success', message: 'Found 5 active agents' },
        { time: new Date().toISOString(), level: 'warning', message: 'Resource ollama responding slowly' },
        { time: new Date().toISOString(), level: 'info', message: 'Health check completed' }
    ];
    
    const terminal = document.getElementById('terminalOutput');
    terminal.innerHTML = '';
    
    logLines.forEach(log => {
        addLogLine(`[${formatTime(log.time)}] ${log.message}`, log.level);
    });
    
    updateStats();
}

function displayLogs(logs) {
    const terminal = document.getElementById('terminalOutput');
    terminal.innerHTML = '';
    
    if (!logs || logs.length === 0) {
        addLogLine('No logs available', 'info');
        return;
    }
    
    logs.forEach(log => {
        // Parse log line if it's a string
        if (typeof log === 'string') {
            addLogLine(log);
        } else {
            addLogLine(`[${formatTime(log.time)}] ${log.message}`, log.level);
        }
    });
    
    terminal.scrollTop = terminal.scrollHeight;
}

function addLogLine(message, level = 'default') {
    const terminal = document.getElementById('terminalOutput');
    const line = document.createElement('div');
    line.className = 'terminal-line';
    
    let prefix = '$>';
    let color = 'var(--primary-cyan)';
    
    switch(level) {
        case 'success':
            color = 'var(--accent-green)';
            break;
        case 'warning':
            color = 'var(--accent-orange)';
            prefix = '!>';
            break;
        case 'error':
            color = 'var(--accent-red)';
            prefix = '!!>';
            break;
        case 'info':
            color = 'var(--primary-magenta)';
            break;
    }
    
    line.innerHTML = `<span class="terminal-prompt" style="color: ${color}">${prefix}</span> ${escapeHtml(message)}`;
    terminal.appendChild(line);
    
    // Limit terminal history
    if (terminal.children.length > 1000) {
        terminal.removeChild(terminal.children[0]);
    }
}

function filterLogs() {
    const level = document.getElementById('logLevel').value;
    
    if (level === 'all') {
        displayLogs(logLines);
        return;
    }
    
    const filtered = logLines.filter(log => {
        if (typeof log === 'string') {
            // Simple string matching for level keywords
            const lowerLog = log.toLowerCase();
            switch(level) {
                case 'error': return lowerLog.includes('error') || lowerLog.includes('fail');
                case 'warning': return lowerLog.includes('warn');
                case 'info': return lowerLog.includes('info');
                default: return true;
            }
        }
        return log.level === level;
    });
    
    displayLogs(filtered);
}

function toggleFollow() {
    const btn = document.getElementById('followBtn');
    const status = document.getElementById('followStatus');
    
    if (isFollowing) {
        isFollowing = false;
        clearInterval(followInterval);
        btn.innerHTML = '<i data-lucide="play"></i> FOLLOW';
        status.textContent = 'Not following';
        if (typeof lucide !== 'undefined') lucide.createIcons();
    } else {
        isFollowing = true;
        btn.innerHTML = '<i data-lucide="pause"></i> STOP';
        status.textContent = 'Following logs...';
        
        followInterval = setInterval(() => {
            loadLogs();
        }, 2000);
        if (typeof lucide !== 'undefined') lucide.createIcons();
    }
}

function clearLogs() {
    document.getElementById('terminalOutput').innerHTML = '';
    logLines = [];
    updateStats();
}

function downloadLogs() {
    const content = logLines.map(log => 
        typeof log === 'string' ? log : `[${log.time}] ${log.level}: ${log.message}`
    ).join('\n');
    
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `agent-logs-${Date.now()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
}

function toggleFullscreen() {
    const container = document.querySelector('.terminal-container');
    container.classList.toggle('fullscreen');
    
    const btn = document.getElementById('fullscreenBtn');
    if (container.classList.contains('fullscreen')) {
        btn.innerHTML = '<i data-lucide="minimize"></i>';
    } else {
        btn.innerHTML = '<i data-lucide="maximize"></i>';
    }
    if (typeof lucide !== 'undefined') lucide.createIcons();
}

function updateStats() {
    document.getElementById('logStats').textContent = `${logLines.length} lines loaded`;
}

function formatTime(timestamp) {
    if (!timestamp) return '';
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
}

function escapeHtml(text) {
    const map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, m => map[m]);
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && document.querySelector('.terminal-container.fullscreen')) {
        toggleFullscreen();
    }
    
    if (e.ctrlKey || e.metaKey) {
        switch(e.key) {
            case 'l':
                e.preventDefault();
                clearLogs();
                break;
            case 'r':
                e.preventDefault();
                loadLogs();
                break;
            case 'f':
                e.preventDefault();
                toggleFollow();
                break;
        }
    }
});