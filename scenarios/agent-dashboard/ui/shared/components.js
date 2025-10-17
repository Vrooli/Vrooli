// Shared UI Components Module
// Provides reusable HTML components for all pages

export function createHeader(activePage = 'dashboard') {
    return `
        <header class="header">
            <div class="logo">
                <div class="logo-icon">
                    <div class="logo-cube">
                        <div class="cube-face front"></div>
                        <div class="cube-face back"></div>
                        <div class="cube-face left"></div>
                        <div class="cube-face right"></div>
                        <div class="cube-face top"></div>
                        <div class="cube-face bottom"></div>
                    </div>
                </div>
                <div class="logo-text">AGENT COMMAND</div>
            </div>
            
            <nav class="nav-menu">
                <a href="/" class="nav-link ${activePage === 'dashboard' ? 'active' : ''}">
                    <i data-lucide="layout-dashboard"></i>
                    <span>Dashboard</span>
                </a>
                <a href="/agents" class="nav-link ${activePage === 'agents' ? 'active' : ''}">
                    <i data-lucide="bot"></i>
                    <span>Agents</span>
                </a>
                <a href="/logs" class="nav-link ${activePage === 'logs' ? 'active' : ''}">
                    <i data-lucide="terminal"></i>
                    <span>Logs</span>
                </a>
                <a href="/metrics" class="nav-link ${activePage === 'metrics' ? 'active' : ''}">
                    <i data-lucide="bar-chart"></i>
                    <span>Metrics</span>
                </a>
            </nav>
            
            <div class="system-status">
                <div class="status-indicator">
                    <i data-lucide="shield-check" class="status-icon active"></i>
                    <span>SYSTEM ONLINE</span>
                </div>
                <div class="status-indicator">
                    <i data-lucide="cpu" class="status-icon warning"></i>
                    <span id="agentCount">0 AGENTS ACTIVE</span>
                </div>
            </div>
        </header>
    `;
}

export function createBackgroundEffects() {
    return `
        <div class="cyber-grid"></div>
        <div class="scan-line"></div>
    `;
}

export function createPageTemplate(title, activePage, content) {
    return `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>${title} - Agent Command Center</title>
    <link href="https://fonts.googleapis.com/css2?family=Orbitron:wght@400;500;600;700;900&family=Share+Tech+Mono&family=Exo+2:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/styles.css">
    <script src="https://unpkg.com/lucide@latest/dist/umd/lucide.js"></script>
</head>
<body>
    ${createBackgroundEffects()}
    <div class="command-center">
        ${createHeader(activePage)}
        <main class="main-content">
            ${content}
        </main>
    </div>
    <script type="module" src="/shared/api.js"></script>
    <script>
        // Initialize Lucide icons
        document.addEventListener('DOMContentLoaded', () => {
            lucide.createIcons();
        });
    </script>
</body>
</html>
    `;
}

export function createAgentCard(agent) {
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
            
            <div class="agent-metrics">
                <div class="metric">
                    <div class="metric-label">
                        <i data-lucide="hash"></i>
                        PID
                    </div>
                    <div class="metric-value">${agent.pid || 'N/A'}</div>
                </div>
                <div class="metric">
                    <div class="metric-label">
                        <i data-lucide="clock"></i>
                        Started
                    </div>
                    <div class="metric-value">${formatTime(agent.start_time)}</div>
                </div>
            </div>
            
            <div class="agent-controls">
                <a href="/agent/${agent.id}" class="control-btn">
                    <i data-lucide="eye"></i>
                    VIEW
                </a>
                <button class="control-btn" onclick="window.agentActions.stop('${agent.id}')">
                    <i data-lucide="square"></i>
                    STOP
                </button>
                <button class="control-btn" onclick="window.location.href='/logs?agent=${agent.id}'">
                    <i data-lucide="scroll-text"></i>
                    LOGS
                </button>
            </div>
        </div>
    `;
}

function formatTime(timestamp) {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp);
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
}