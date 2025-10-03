// Multiple Log Viewer Module
// Allows opening multiple log viewers for different agents simultaneously

class MultiLogViewer {
    constructor() {
        this.viewers = new Map();
        this.zIndexCounter = 1000;
        this.offsetCounter = 0;
    }

    // Create a new floating log viewer for an agent
    createLogViewer(agent) {
        // Check if viewer already exists for this agent
        if (this.viewers.has(agent.id)) {
            this.focusViewer(agent.id);
            return;
        }

        const viewerId = `log-viewer-${agent.id}`;
        const offset = this.offsetCounter * 30;
        this.offsetCounter = (this.offsetCounter + 1) % 10;

        // Create viewer container
        const viewer = document.createElement('div');
        viewer.id = viewerId;
        viewer.className = 'floating-log-viewer';
        viewer.style.zIndex = this.zIndexCounter++;
        viewer.style.top = `${100 + offset}px`;
        viewer.style.left = `${100 + offset}px`;
        
        viewer.innerHTML = `
            <div class="log-viewer-header" data-agent-id="${agent.id}">
                <div class="log-viewer-title">
                    <i data-lucide="scroll-text"></i>
                    ${agent.name} (${agent.type}) - Logs
                </div>
                <div class="log-viewer-controls">
                    <button class="log-control-btn" onclick="multiLogViewer.toggleFollow('${agent.id}')" title="Auto-follow">
                        <i data-lucide="eye" id="follow-icon-${agent.id}"></i>
                    </button>
                    <button class="log-control-btn" onclick="multiLogViewer.clearLogs('${agent.id}')" title="Clear">
                        <i data-lucide="trash-2"></i>
                    </button>
                    <button class="log-control-btn" onclick="multiLogViewer.downloadLogs('${agent.id}')" title="Download">
                        <i data-lucide="download"></i>
                    </button>
                    <button class="log-control-btn" onclick="multiLogViewer.minimizeViewer('${agent.id}')" title="Minimize">
                        <i data-lucide="minus"></i>
                    </button>
                    <button class="log-control-btn" onclick="multiLogViewer.closeViewer('${agent.id}')" title="Close">
                        <i data-lucide="x"></i>
                    </button>
                </div>
            </div>
            <div class="log-viewer-body">
                <div class="log-viewer-filters">
                    <select class="log-filter" id="level-filter-${agent.id}" onchange="multiLogViewer.filterLogs('${agent.id}')">
                        <option value="all">All Levels</option>
                        <option value="error">Errors</option>
                        <option value="warning">Warnings</option>
                        <option value="info">Info</option>
                        <option value="debug">Debug</option>
                    </select>
                    <input type="text" class="log-search" id="search-${agent.id}" 
                           placeholder="Search logs..." 
                           onkeyup="multiLogViewer.searchLogs('${agent.id}')">
                    <span class="log-stats" id="stats-${agent.id}">0 lines</span>
                </div>
                <div class="log-content" id="log-content-${agent.id}">
                    <div class="log-loading">Loading logs...</div>
                </div>
            </div>
        `;

        document.body.appendChild(viewer);
        
        // Initialize Lucide icons for the new viewer
        if (typeof lucide !== 'undefined') {
            lucide.createIcons();
        }

        // Make viewer draggable
        this.makeDraggable(viewer);
        
        // Store viewer state
        this.viewers.set(agent.id, {
            element: viewer,
            agent: agent,
            logs: [],
            following: false,
            followInterval: null,
            minimized: false
        });

        // Load initial logs
        this.loadLogs(agent.id);

        // Add to terminal log
        if (typeof addTerminalLog !== 'undefined') {
            addTerminalLog(`Opened log viewer for ${agent.name}`, 'info');
        }
    }

    // Make viewer draggable
    makeDraggable(viewer) {
        const header = viewer.querySelector('.log-viewer-header');
        let isDragging = false;
        let startX, startY, initialX, initialY;

        header.addEventListener('mousedown', (e) => {
            if (e.target.closest('.log-viewer-controls')) return;
            
            isDragging = true;
            startX = e.clientX;
            startY = e.clientY;
            initialX = viewer.offsetLeft;
            initialY = viewer.offsetTop;
            
            viewer.style.zIndex = this.zIndexCounter++;
            header.style.cursor = 'grabbing';
        });

        document.addEventListener('mousemove', (e) => {
            if (!isDragging) return;
            
            const deltaX = e.clientX - startX;
            const deltaY = e.clientY - startY;
            
            viewer.style.left = `${initialX + deltaX}px`;
            viewer.style.top = `${initialY + deltaY}px`;
        });

        document.addEventListener('mouseup', () => {
            isDragging = false;
            header.style.cursor = 'grab';
        });
    }

    // Load logs for a specific agent
    async loadLogs(agentId) {
        const viewer = this.viewers.get(agentId);
        if (!viewer) return;

        const contentDiv = document.getElementById(`log-content-${agentId}`);
        const statsSpan = document.getElementById(`stats-${agentId}`);
        
        try {
            const apiPort = window.API_PORT || '15000';
            const response = await fetch(`http://localhost:${apiPort}/api/v1/agents/${agentId}/logs?lines=150`);

            if (!response.ok) {
                throw new Error(`Failed to fetch logs: ${response.status}`);
            }

            const payload = await response.json();
            const logs = payload.logs || payload.data?.logs || [];
            viewer.logs = Array.isArray(logs) ? logs : [];

            this.displayLogs(agentId, viewer.logs);
            statsSpan.textContent = `${viewer.logs.length} lines`;

        } catch (error) {
            contentDiv.innerHTML = `<div class="log-error">Failed to load logs: ${error.message}</div>`;
            console.error('Failed to load logs:', error);
        }
    }

    // Display logs in the viewer
    displayLogs(agentId, logs) {
        const contentDiv = document.getElementById(`log-content-${agentId}`);
        if (!contentDiv) return;

        if (logs.length === 0) {
            contentDiv.innerHTML = '<div class="log-empty">No logs available</div>';
            return;
        }

        contentDiv.innerHTML = logs.map((log, index) => {
            const level = this.detectLogLevel(log);
            return `<div class="log-line log-${level}" data-line="${index}">${this.escapeHtml(log)}</div>`;
        }).join('');

        // Auto-scroll to bottom if following
        const viewer = this.viewers.get(agentId);
        if (viewer && viewer.following) {
            contentDiv.scrollTop = contentDiv.scrollHeight;
        }
    }

    // Detect log level from log content
    detectLogLevel(log) {
        const logLower = log.toLowerCase();
        if (logLower.includes('error') || logLower.includes('fail')) return 'error';
        if (logLower.includes('warn')) return 'warning';
        if (logLower.includes('debug')) return 'debug';
        return 'info';
    }

    // Escape HTML to prevent XSS
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Toggle auto-follow for logs
    toggleFollow(agentId) {
        const viewer = this.viewers.get(agentId);
        if (!viewer) return;

        viewer.following = !viewer.following;
        const icon = document.getElementById(`follow-icon-${agentId}`);
        
        if (viewer.following) {
            icon.setAttribute('data-lucide', 'eye-off');
            // Start following
            viewer.followInterval = setInterval(() => {
                this.loadLogs(agentId);
            }, 5000);
            
            if (typeof addTerminalLog !== 'undefined') {
                addTerminalLog(`Auto-follow enabled for ${viewer.agent.name}`, 'info');
            }
        } else {
            icon.setAttribute('data-lucide', 'eye');
            // Stop following
            if (viewer.followInterval) {
                clearInterval(viewer.followInterval);
                viewer.followInterval = null;
            }
            
            if (typeof addTerminalLog !== 'undefined') {
                addTerminalLog(`Auto-follow disabled for ${viewer.agent.name}`, 'info');
            }
        }
        
        // Re-render icon
        if (typeof lucide !== 'undefined') {
            lucide.createIcons();
        }
    }

    // Clear logs in viewer
    clearLogs(agentId) {
        const viewer = this.viewers.get(agentId);
        if (!viewer) return;

        viewer.logs = [];
        this.displayLogs(agentId, []);
        
        const statsSpan = document.getElementById(`stats-${agentId}`);
        if (statsSpan) {
            statsSpan.textContent = '0 lines';
        }
    }

    // Download logs
    downloadLogs(agentId) {
        const viewer = this.viewers.get(agentId);
        if (!viewer) return;

        const content = viewer.logs.join('\n');
        const blob = new Blob([content], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${viewer.agent.name}-logs-${new Date().toISOString()}.txt`;
        a.click();
        URL.revokeObjectURL(url);
        
        if (typeof addTerminalLog !== 'undefined') {
            addTerminalLog(`Downloaded logs for ${viewer.agent.name}`, 'success');
        }
    }

    // Filter logs by level
    filterLogs(agentId) {
        const viewer = this.viewers.get(agentId);
        if (!viewer) return;

        const level = document.getElementById(`level-filter-${agentId}`).value;
        const lines = document.querySelectorAll(`#log-content-${agentId} .log-line`);
        
        let visibleCount = 0;
        lines.forEach(line => {
            if (level === 'all' || line.classList.contains(`log-${level}`)) {
                line.style.display = 'block';
                visibleCount++;
            } else {
                line.style.display = 'none';
            }
        });
        
        const statsSpan = document.getElementById(`stats-${agentId}`);
        if (statsSpan) {
            statsSpan.textContent = `${visibleCount} / ${viewer.logs.length} lines`;
        }
    }

    // Search logs
    searchLogs(agentId) {
        const viewer = this.viewers.get(agentId);
        if (!viewer) return;

        const searchTerm = document.getElementById(`search-${agentId}`).value.toLowerCase();
        const lines = document.querySelectorAll(`#log-content-${agentId} .log-line`);
        
        let visibleCount = 0;
        lines.forEach(line => {
            if (!searchTerm || line.textContent.toLowerCase().includes(searchTerm)) {
                line.style.display = 'block';
                visibleCount++;
            } else {
                line.style.display = 'none';
            }
        });
        
        const statsSpan = document.getElementById(`stats-${agentId}`);
        if (statsSpan) {
            statsSpan.textContent = `${visibleCount} / ${viewer.logs.length} lines`;
        }
    }

    // Minimize viewer
    minimizeViewer(agentId) {
        const viewer = this.viewers.get(agentId);
        if (!viewer) return;

        viewer.minimized = !viewer.minimized;
        const body = viewer.element.querySelector('.log-viewer-body');
        
        if (viewer.minimized) {
            body.style.display = 'none';
            viewer.element.style.height = 'auto';
        } else {
            body.style.display = 'block';
            viewer.element.style.height = '';
        }
    }

    // Close viewer
    closeViewer(agentId) {
        const viewer = this.viewers.get(agentId);
        if (!viewer) return;

        // Stop following if active
        if (viewer.followInterval) {
            clearInterval(viewer.followInterval);
        }

        // Remove from DOM
        viewer.element.remove();
        
        // Remove from map
        this.viewers.delete(agentId);
        
        if (typeof addTerminalLog !== 'undefined') {
            addTerminalLog(`Closed log viewer for ${viewer.agent.name}`, 'info');
        }
    }

    // Focus a viewer (bring to front)
    focusViewer(agentId) {
        const viewer = this.viewers.get(agentId);
        if (!viewer) return;

        viewer.element.style.zIndex = this.zIndexCounter++;
        
        // Flash the header to indicate focus
        const header = viewer.element.querySelector('.log-viewer-header');
        header.style.animation = 'flash 0.3s';
        setTimeout(() => {
            header.style.animation = '';
        }, 300);
    }

    // Close all viewers
    closeAllViewers() {
        this.viewers.forEach((viewer, agentId) => {
            this.closeViewer(agentId);
        });
    }
}

// Create global instance
const multiLogViewer = new MultiLogViewer();

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = MultiLogViewer;
}
