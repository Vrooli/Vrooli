// Process Monitoring Module
export class ProcessMonitor {
    constructor(apiBase, showToast) {
        this.apiBase = apiBase;
        this.showToast = showToast;
        this.runningProcesses = {};
        this.monitoringInterval = null;
        this.logAutoScroll = true;
        this.logUpdateInterval = null;
        this.activeLogTaskId = null;
        this.logSequences = {};
        this.logPollingInterval = 2000;
        this.dropdownVisible = false;
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

    initializeDropdown() {
        const monitor = document.getElementById('process-monitor');
        const toggle = document.getElementById('process-monitor-toggle');
        const dropdown = document.getElementById('process-monitor-dropdown');
        if (!monitor || !toggle || !dropdown) {
            return;
        }

        toggle.addEventListener('click', (event) => {
            event.stopPropagation();
            if (!Object.keys(this.runningProcesses).length) {
                return;
            }
            this.dropdownVisible = !this.dropdownVisible;
            this.updateDropdownState();
        });

        document.addEventListener('click', (event) => {
            if (!monitor.contains(event.target) && this.dropdownVisible) {
                this.dropdownVisible = false;
                this.updateDropdownState();
            }
        });

        this.updateDropdownState();
    }

    updateDropdownState() {
        const dropdown = document.getElementById('process-monitor-dropdown');
        const arrow = document.getElementById('process-monitor-arrow');
        const toggle = document.getElementById('process-monitor-toggle');
        if (!dropdown || !arrow) {
            return;
        }

        const hasProcesses = Object.keys(this.runningProcesses).length > 0;
        const shouldShow = this.dropdownVisible && hasProcesses;
        dropdown.classList.toggle('show', shouldShow);
        arrow.classList.toggle('open', shouldShow);
        if (toggle) {
            toggle.setAttribute('aria-expanded', shouldShow ? 'true' : 'false');
        }
    }

    async fetchRunningProcesses() {
        try {
            const response = await fetch(`${this.apiBase}/processes/running`);
            if (!response.ok) {
                console.error('Failed to fetch running processes:', response.statusText);
                return;
            }
            
            const data = await response.json();
            
            // Update running processes map
            const newRunningProcesses = {};
            if (data.processes && Array.isArray(data.processes)) {
                data.processes.forEach(process => {
                    newRunningProcesses[process.task_id] = {
                        task_id: process.task_id,
                        process_id: process.process_id,
                        start_time: process.start_time,
                        agent_id: process.agent_id,
                        duration: this.formatDuration(process.start_time),
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
                        window.ecosystemManager.handleProcessStarted(newRunningProcesses[taskId]);
                    }
                }
            }
            
            // Also refresh any existing task cards to show updated running state
            for (const taskId of newIds) {
                if (window.ecosystemManager) {
                    window.ecosystemManager.refreshTaskCard(taskId);
                }
            }
            
            this.runningProcesses = newRunningProcesses;
            this.renderProcessWidget(newRunningProcesses);

            if (window.ecosystemManager) {
                Object.values(newRunningProcesses).forEach(proc => {
                    if (proc && proc.start_time) {
                        window.ecosystemManager.startElapsedTimeCounter(proc.task_id, new Date(proc.start_time));
                    }
                });
            }
            return this.runningProcesses;
        } catch (error) {
            console.error('Error fetching running processes:', error);
        }
    }

    async terminateProcess(taskId) {
        try {
            const response = await fetch(`${this.apiBase}/queue/processes/terminate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ task_id: taskId })
            });
            
            if (!response.ok) {
                throw new Error(`Failed to terminate process: ${response.statusText}`);
            }
            
            const result = await response.json();
            
            // Remove from running processes
            delete this.runningProcesses[taskId];
            this.renderProcessWidget(this.runningProcesses);
            
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
        if (!modal) return;

        modal.classList.add('show');
        document.body.style.overflow = 'hidden';
        document.getElementById('log-viewer-task-id').textContent = taskId;

        this.stopLogUpdates();
        this.activeLogTaskId = taskId;
        this.clearLogViewer(taskId);
        const autoScrollButton = document.getElementById('log-auto-scroll-toggle');
        if (autoScrollButton) {
            autoScrollButton.classList.toggle('log-auto-scroll-enabled', this.logAutoScroll);
        }
        this.logSequences[taskId] = this.logSequences[taskId] || 0;
        this.fetchTaskLogs(taskId, true);
        this.logUpdateInterval = setInterval(() => this.fetchTaskLogs(taskId), this.logPollingInterval);
    }

    closeLogViewer() {
        const modal = document.getElementById('log-viewer-modal');
        if (!modal) return;

        modal.classList.remove('show');
        document.body.style.overflow = '';
        this.stopLogUpdates();
        this.activeLogTaskId = null;
    }

    clearLogViewer(taskId = null) {
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
        if (taskId) {
            this.logSequences[taskId] = 0;
        }
    }

    clearLogs() {
        if (this.activeLogTaskId) {
            this.clearLogViewer(this.activeLogTaskId);
        } else {
            this.clearLogViewer();
        }
    }

    async fetchTaskLogs(taskId, initial = false) {
        try {
            // Viewer may have been closed while request was in flight
            if (this.activeLogTaskId !== taskId) {
                return;
            }

            const after = this.logSequences[taskId] || 0;
            const response = await fetch(`${this.apiBase}/tasks/${taskId}/logs?after=${after}`);
            if (!response.ok) {
                throw new Error(`${response.status} ${response.statusText}`);
            }

            const data = await response.json();
            if (this.activeLogTaskId !== taskId) {
                return;
            }

            if (Array.isArray(data.entries)) {
                data.entries.forEach(entry => {
                    const entryWithTask = { ...entry, task_id: taskId };
                    this.addLogEntry(entryWithTask);
                });
            }

            if (typeof data.next_sequence === 'number') {
                this.logSequences[taskId] = data.next_sequence;
            }

            if (!data.running && data.completed) {
                if (!data.entries || data.entries.length === 0) {
                    this.addLogEntry({ level: 'info', message: 'No logs were emitted for this task.' });
                }
                this.addLogEntry({ level: 'info', message: 'Task execution finished.' });
                this.stopLogUpdates();
            }
        } catch (error) {
            console.error('Failed to fetch task logs', error);
            if (initial) {
                this.addLogEntry({ level: 'error', message: `Failed to load logs: ${error.message}` });
            }
        }
    }

    stopLogUpdates() {
        if (this.logUpdateInterval) {
            clearInterval(this.logUpdateInterval);
            this.logUpdateInterval = null;
        }
    }

    addLogEntry(entry) {
        if (!entry) return;
        const taskId = entry.task_id || this.activeLogTaskId;
        if (!taskId) {
            return;
        }

        if (typeof entry.sequence === 'number') {
            const currentSeq = this.logSequences[taskId] || 0;
            if (entry.sequence > currentSeq) {
                this.logSequences[taskId] = entry.sequence;
            }
        }

        if (this.activeLogTaskId !== taskId) {
            return;
        }

        const logOutput = document.getElementById('log-output');
        if (!logOutput) return;

        const placeholder = logOutput.querySelector('.log-placeholder');
        if (placeholder) {
            placeholder.remove();
        }

        const level = entry.level || (entry.stream === 'stderr' ? 'error' : 'info');
        const timestamp = entry.timestamp ? new Date(entry.timestamp) : new Date();

        const row = document.createElement('div');
        row.className = `log-entry log-${level}`;
        row.innerHTML = `
            <span class="log-timestamp">${timestamp.toLocaleTimeString()}</span>
            <span class="log-message">${this.escapeHtml(entry.message || '')}</span>
        `;

        logOutput.appendChild(row);

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

    toggleAutoScroll() {
        this.logAutoScroll = !this.logAutoScroll;
        const button = document.getElementById('log-auto-scroll-toggle');
        if (button) {
            button.classList.toggle('log-auto-scroll-enabled', this.logAutoScroll);
        }
        if (this.logAutoScroll) {
            this.scrollLogToBottom();
        }
    }

    renderProcessWidget(processMap = {}) {
        const monitor = document.getElementById('process-monitor');
        const countEl = document.getElementById('running-process-count');
        const detailsEl = document.getElementById('process-monitor-dropdown');

        const running = Object.values(processMap || {});
        const count = running.length;

        const toggle = document.getElementById('process-monitor-toggle');
        if (toggle) {
            toggle.setAttribute('aria-expanded', this.dropdownVisible && count > 0 ? 'true' : 'false');
            toggle.setAttribute('aria-label', count === 1 ? 'View 1 running agent' : `View ${count} running agents`);
        }

        if (monitor) {
            monitor.style.display = count > 0 ? 'inline-flex' : 'none';
        }
        if (countEl) {
            countEl.textContent = count;
        }
        if (detailsEl) {
            detailsEl.innerHTML = '';
            if (count > 0) {
                running.forEach(proc => {
                    const item = document.createElement('div');
                    item.className = 'process-detail-item';
                    item.dataset.taskId = proc.task_id;

                    const title = document.createElement('strong');
                    title.textContent = proc.task_id || '';
                    item.appendChild(title);

                    const meta = document.createElement('div');
                    meta.className = 'process-detail-meta';

                    if (proc.duration) {
                        const duration = document.createElement('span');
                        duration.textContent = proc.duration;
                        meta.appendChild(duration);
                    }

                    if (proc.agent_id && proc.agent_id !== proc.task_id) {
                        const agent = document.createElement('span');
                        agent.textContent = proc.agent_id;
                        meta.appendChild(agent);
                    }

                    if (proc.process_id) {
                        const pid = document.createElement('span');
                        pid.textContent = `PID ${proc.process_id}`;
                        meta.appendChild(pid);
                    }

                    if (meta.children.length) {
                        item.appendChild(meta);
                    }

                    item.addEventListener('click', () => {
                        if (window.ecosystemManager) {
                            window.ecosystemManager.processMonitor.openLogViewer(proc.task_id);
                        }
                        this.dropdownVisible = false;
                        this.updateDropdownState();
                    });

                    detailsEl.appendChild(item);
                });
            }
        }

        if (!count) {
            this.dropdownVisible = false;
        }
        this.updateDropdownState();
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
