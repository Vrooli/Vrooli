// Process Monitoring Module
export class ProcessMonitor {
    constructor(apiBase, showToast) {
        this.apiBase = apiBase;
        this.showToast = showToast;
        this.runningProcesses = {};
        this.monitoringInterval = null;
        this.logAutoScroll = true;
        this.logUpdateInterval = null;
    }

    async startMonitoring() {
        // Initial fetch
        await this.fetchRunningProcesses();
        
        // Set up polling interval
        this.monitoringInterval = setInterval(async () => {
            await this.fetchRunningProcesses();
        }, 5000); // Poll every 5 seconds
    }

    stopMonitoring() {
        if (this.monitoringInterval) {
            clearInterval(this.monitoringInterval);
            this.monitoringInterval = null;
        }
    }

    async fetchRunningProcesses() {
        try {
            const response = await fetch(`${this.apiBase}/queue/status`);
            if (!response.ok) {
                console.error('Failed to fetch queue status:', response.statusText);
                return;
            }
            
            const data = await response.json();
            
            // Update running processes map
            const newRunningProcesses = {};
            if (data.running_tasks && Array.isArray(data.running_tasks)) {
                data.running_tasks.forEach(task => {
                    newRunningProcesses[task.id] = {
                        task_id: task.id,
                        start_time: task.started_at || new Date().toISOString(),
                        status: 'running'
                    };
                });
            }
            
            // Check for changes
            const oldIds = new Set(Object.keys(this.runningProcesses));
            const newIds = new Set(Object.keys(newRunningProcesses));
            
            // Find completed processes
            for (const taskId of oldIds) {
                if (!newIds.has(taskId)) {
                    // Process completed
                    if (window.ecosystemManager) {
                        window.ecosystemManager.handleProcessCompleted(taskId);
                    }
                }
            }
            
            // Find new processes
            for (const taskId of newIds) {
                if (!oldIds.has(taskId)) {
                    // New process started
                    if (window.ecosystemManager) {
                        window.ecosystemManager.handleProcessStarted(taskId);
                    }
                }
            }
            
            this.runningProcesses = newRunningProcesses;
            return this.runningProcesses;
        } catch (error) {
            console.error('Error fetching running processes:', error);
        }
    }

    async terminateProcess(taskId) {
        try {
            const response = await fetch(`${this.apiBase}/queue/terminate/${taskId}`, {
                method: 'POST'
            });
            
            if (!response.ok) {
                throw new Error(`Failed to terminate process: ${response.statusText}`);
            }
            
            const result = await response.json();
            
            // Remove from running processes
            delete this.runningProcesses[taskId];
            
            return result;
        } catch (error) {
            console.error('Failed to terminate process:', error);
            throw error;
        }
    }

    isTaskRunning(taskId) {
        return !!this.runningProcesses[taskId];
    }

    getRunningProcess(taskId) {
        return this.runningProcesses[taskId];
    }

    formatDuration(startTime) {
        if (!startTime) return '00:00';
        
        const start = new Date(startTime);
        const now = new Date();
        const duration = Math.floor((now - start) / 1000);
        
        const hours = Math.floor(duration / 3600);
        const minutes = Math.floor((duration % 3600) / 60);
        const seconds = duration % 60;
        
        if (hours > 0) {
            return `${hours}h ${minutes}m ${seconds}s`;
        } else if (minutes > 0) {
            return `${minutes}m ${seconds}s`;
        } else {
            return `${seconds}s`;
        }
    }

    // Log Viewer Methods
    openLogViewer(taskId) {
        const modal = document.getElementById('log-viewer-modal');
        if (modal) {
            modal.classList.add('show');
            // Disable body scroll when showing modal
            document.body.style.overflow = 'hidden';
            document.getElementById('log-viewer-task-id').textContent = taskId;
            this.clearLogViewer();
            this.startLogUpdates(taskId);
        }
    }

    closeLogViewer() {
        const modal = document.getElementById('log-viewer-modal');
        if (modal) {
            modal.classList.remove('show');
            // Re-enable body scroll when closing modal
            document.body.style.overflow = '';
            this.stopLogUpdates();
        }
    }

    clearLogViewer() {
        const logOutput = document.getElementById('log-output');
        if (logOutput) {
            logOutput.innerHTML = `
                <div class="log-placeholder">
                    <i class="fas fa-terminal"></i>
                    <p>Waiting for task execution logs...</p>
                    <p class="log-hint">Logs will appear here when the task starts executing with Claude Code</p>
                </div>
            `;
        }
    }

    startLogUpdates(taskId) {
        // Simulate log streaming
        this.logUpdateInterval = setInterval(() => {
            if (Math.random() > 0.3) {
                const messages = [
                    'Initializing Claude Code...',
                    'Loading task configuration...',
                    'Assembling prompt from templates...',
                    'Connecting to Claude API...',
                    'Sending request to Claude...',
                    'Processing response...',
                    'Executing generated code...',
                    'Validating results...'
                ];
                const randomMessage = messages[Math.floor(Math.random() * messages.length)];
                this.addLogEntry('info', randomMessage);
            }
        }, 3000);
    }

    stopLogUpdates() {
        if (this.logUpdateInterval) {
            clearInterval(this.logUpdateInterval);
            this.logUpdateInterval = null;
        }
    }

    addLogEntry(type, message) {
        const logOutput = document.getElementById('log-output');
        if (!logOutput) return;
        
        // Remove placeholder if it exists
        const placeholder = logOutput.querySelector('.log-placeholder');
        if (placeholder) {
            placeholder.remove();
        }
        
        const entry = document.createElement('div');
        entry.className = `log-entry log-${type}`;
        entry.innerHTML = `
            <span class="log-timestamp">${new Date().toLocaleTimeString()}</span>
            <span class="log-message">${this.escapeHtml(message)}</span>
        `;
        
        logOutput.appendChild(entry);
        
        if (this.logAutoScroll) {
            this.scrollLogToBottom();
        }
    }

    scrollLogToBottom() {
        const logOutput = document.getElementById('log-output');
        if (logOutput) {
            logOutput.scrollTop = logOutput.scrollHeight;
        }
    }

    toggleLogAutoScroll() {
        this.logAutoScroll = !this.logAutoScroll;
        const button = document.getElementById('log-auto-scroll-toggle');
        if (this.logAutoScroll) {
            button?.classList.add('log-auto-scroll-enabled');
            this.scrollLogToBottom();
        } else {
            button?.classList.remove('log-auto-scroll-enabled');
        }
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}