const API_BASE = `http://localhost:${window.location.port === '3000' ? '8090' : window.location.port.replace('3', '8')}`;

class TaskPlanner {
    constructor() {
        this.tasks = {
            backlog: [],
            staged: [],
            progress: [],
            completed: []
        };
        this.currentFocus = { tab: 0, task: -1 };
        this.tabNames = ['backlog', 'staged', 'progress', 'completed'];
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadTasks();
        this.startPolling();
    }

    setupEventListeners() {
        // Parse button
        document.getElementById('parseBtn').addEventListener('click', () => this.parseTasks());
        
        // Suggestion button
        document.getElementById('suggestBtn').addEventListener('click', () => this.getSuggestions());

        // Tab switching
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => this.switchTab(e.target.dataset.tab));
        });

        // Global keyboard navigation
        document.addEventListener('keydown', (e) => this.handleGlobalKeydown(e));

        // Enhanced textarea controls
        document.getElementById('taskInput').addEventListener('keydown', (e) => {
            if (e.ctrlKey && e.key === 'Enter') {
                e.preventDefault();
                this.parseTasks();
            } else if (e.key === 'Escape') {
                e.target.blur();
            }
        });

        // Help modal toggle
        this.createHelpModal();
        document.getElementById('helpBtn').addEventListener('click', () => this.toggleHelpModal());
    }

    switchTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.tab-btn').forEach(btn => {
            const isActive = btn.dataset.tab === tabName;
            btn.classList.toggle('active', isActive);
            btn.setAttribute('aria-selected', isActive);
        });

        // Update tab panes
        document.querySelectorAll('.tab-pane').forEach(pane => {
            pane.classList.toggle('active', pane.id === tabName);
        });

        // Update focus tracking
        this.currentFocus.tab = this.tabNames.indexOf(tabName);
        this.currentFocus.task = -1;
    }

    async parseTasks() {
        const input = document.getElementById('taskInput').value.trim();
        if (!input) {
            this.showStatus('Please enter some text to parse', 'error');
            return;
        }

        this.showStatus('Parsing tasks with AI...', 'loading');
        const parseBtn = document.getElementById('parseBtn');
        parseBtn.disabled = true;

        try {
            const response = await fetch(`${API_BASE}/api/parse-text`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ text: input })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            
            if (data.tasks_created > 0) {
                this.showStatus(`Successfully created ${data.tasks_created} tasks`, 'success');
                document.getElementById('taskInput').value = '';
                await this.loadTasks();
            } else {
                this.showStatus('No tasks could be extracted from the text', 'error');
            }
        } catch (error) {
            console.error('Parse error:', error);
            this.showStatus('Failed to parse tasks. Please try again.', 'error');
        } finally {
            parseBtn.disabled = false;
        }
    }

    async loadTasks() {
        try {
            const response = await fetch(`${API_BASE}/api/tasks`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const allTasks = await response.json();
            
            // Categorize tasks by status
            this.tasks = {
                backlog: [],
                staged: [],
                progress: [],
                completed: []
            };

            allTasks.forEach(task => {
                switch (task.status) {
                    case 'BACKLOG':
                        this.tasks.backlog.push(task);
                        break;
                    case 'STAGED':
                        this.tasks.staged.push(task);
                        break;
                    case 'IN_PROGRESS':
                        this.tasks.progress.push(task);
                        break;
                    case 'COMPLETED':
                        this.tasks.completed.push(task);
                        break;
                }
            });

            this.renderTasks();
            this.updateStats();
        } catch (error) {
            console.error('Load tasks error:', error);
        }
    }

    renderTasks() {
        this.renderTaskList('backlog', this.tasks.backlog);
        this.renderTaskList('staged', this.tasks.staged);
        this.renderTaskList('progress', this.tasks.progress);
        this.renderTaskList('completed', this.tasks.completed);
    }

    renderTaskList(status, tasks) {
        const container = document.getElementById(`${status}List`);
        
        if (tasks.length === 0) {
            container.innerHTML = '<div class="empty-state">No tasks in this category</div>';
            return;
        }

        container.innerHTML = tasks.map(task => `
            <div class="task-card" data-task-id="${task.id}">
                <div class="task-header">
                    <div class="task-title">${this.escapeHtml(task.title)}</div>
                    <span class="task-priority priority-${task.priority || 'medium'}">${task.priority || 'medium'}</span>
                </div>
                ${task.description ? `<div class="task-description">${this.escapeHtml(task.description)}</div>` : ''}
                <div class="task-meta">
                    <span>Created: ${new Date(task.created_at).toLocaleDateString()}</span>
                    ${task.confidence ? `<span>AI Confidence: ${Math.round(task.confidence * 100)}%</span>` : ''}
                </div>
                <div class="task-actions">
                    ${this.getTaskActions(status, task)}
                </div>
            </div>
        `).join('');

        // Add event listeners to action buttons
        container.querySelectorAll('.btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const action = e.target.dataset.action;
                const taskId = e.target.closest('.task-card').dataset.taskId;
                this.handleTaskAction(action, taskId);
            });
        });
    }

    getTaskActions(status, task) {
        switch (status) {
            case 'backlog':
                return '<button class="btn btn-secondary" data-action="research">Research & Plan</button>';
            case 'staged':
                return '<button class="btn btn-success" data-action="implement">Implement</button>';
            case 'progress':
                return '<button class="btn btn-secondary" data-action="view">View Details</button>';
            case 'completed':
                return '<button class="btn btn-secondary" data-action="view">View Implementation</button>';
            default:
                return '';
        }
    }

    async handleTaskAction(action, taskId) {
        try {
            switch (action) {
                case 'research':
                    await this.researchTask(taskId);
                    break;
                case 'implement':
                    await this.implementTask(taskId);
                    break;
                case 'view':
                    await this.viewTask(taskId);
                    break;
            }
        } catch (error) {
            console.error(`Action ${action} failed:`, error);
            this.showStatus(`Failed to ${action} task`, 'error');
        }
    }

    async researchTask(taskId) {
        this.showStatus('Researching task...', 'loading');
        
        try {
            const response = await fetch(`${API_BASE}/api/tasks/${taskId}/research`, {
                method: 'POST'
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            this.showStatus('Task researched and moved to staged', 'success');
            await this.loadTasks();
        } catch (error) {
            console.error('Research error:', error);
            this.showStatus('Failed to research task', 'error');
        }
    }

    async implementTask(taskId) {
        this.showStatus('Starting implementation...', 'loading');
        
        try {
            const response = await fetch(`${API_BASE}/api/tasks/${taskId}/implement`, {
                method: 'POST'
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            this.showStatus('Implementation started', 'success');
            await this.loadTasks();
        } catch (error) {
            console.error('Implementation error:', error);
            this.showStatus('Failed to start implementation', 'error');
        }
    }

    async viewTask(taskId) {
        try {
            const response = await fetch(`${API_BASE}/api/tasks/${taskId}`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const task = await response.json();
            alert(JSON.stringify(task, null, 2));
        } catch (error) {
            console.error('View task error:', error);
            this.showStatus('Failed to load task details', 'error');
        }
    }

    updateStats() {
        const total = Object.values(this.tasks).reduce((sum, list) => sum + list.length, 0);
        const completed = this.tasks.completed.length;
        const completionRate = total > 0 ? Math.round((completed / total) * 100) : 0;
        
        // Calculate average confidence
        const allTasks = Object.values(this.tasks).flat();
        const avgConfidence = allTasks.length > 0 
            ? Math.round(allTasks.reduce((sum, task) => sum + (task.confidence || 0), 0) / allTasks.length * 100)
            : 0;

        document.getElementById('totalTasks').textContent = total;
        document.getElementById('completionRate').textContent = `${completionRate}%`;
        document.getElementById('avgTime').textContent = '2.5h'; // Placeholder
        document.getElementById('aiConfidence').textContent = `${avgConfidence}%`;
    }

    showStatus(message, type) {
        const statusEl = document.getElementById('parseStatus');
        statusEl.className = `status-message ${type}`;
        statusEl.textContent = message;
        
        if (type !== 'loading') {
            setTimeout(() => {
                statusEl.className = 'status-message';
            }, 5000);
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    async getSuggestions() {
        this.showStatus('Generating smart task suggestions...', 'loading');
        const suggestBtn = document.getElementById('suggestBtn');
        suggestBtn.disabled = true;

        try {
            // Gather context from existing tasks for better suggestions
            const allTasks = Object.values(this.tasks).flat();
            const context = {
                existing_tasks: allTasks.map(t => ({ title: t.title, description: t.description, priority: t.priority })),
                task_counts: {
                    backlog: this.tasks.backlog.length,
                    staged: this.tasks.staged.length,
                    progress: this.tasks.progress.length,
                    completed: this.tasks.completed.length
                }
            };

            const response = await fetch(`${API_BASE}/api/suggest-tasks`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ context })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            
            if (data.suggestions && data.suggestions.length > 0) {
                // Display suggestions in the textarea
                const suggestionText = data.suggestions.map(s => `• ${s.title}${s.description ? ': ' + s.description : ''}`).join('\\n');
                document.getElementById('taskInput').value = suggestionText;
                this.showStatus(`Generated ${data.suggestions.length} intelligent task suggestions`, 'success');
            } else {
                this.showStatus('No suggestions could be generated. Try adding some tasks first.', 'error');
            }
        } catch (error) {
            console.error('Suggestion error:', error);
            this.showStatus('Failed to generate suggestions. Please try again.', 'error');
        } finally {
            suggestBtn.disabled = false;
        }
    }

    startPolling() {
        // Poll for updates every 10 seconds
        setInterval(() => {
            this.loadTasks();
        }, 10000);
    }

    handleGlobalKeydown(e) {
        // Help modal
        if (e.key === '?' || e.key === 'h') {
            if (!e.target.matches('input, textarea')) {
                e.preventDefault();
                this.toggleHelpModal();
                return;
            }
        }

        // Close modals
        if (e.key === 'Escape') {
            this.closeModals();
            return;
        }

        // Skip navigation if typing in input
        if (e.target.matches('input, textarea')) {
            return;
        }

        // Tab navigation (1-4 keys)
        if (['1', '2', '3', '4'].includes(e.key)) {
            e.preventDefault();
            const tabIndex = parseInt(e.key) - 1;
            this.switchTab(this.tabNames[tabIndex]);
            this.currentFocus.tab = tabIndex;
            this.currentFocus.task = -1;
            return;
        }

        // Arrow key navigation
        if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(e.key)) {
            e.preventDefault();
            this.handleArrowNavigation(e.key);
            return;
        }

        // Quick actions
        switch (e.key) {
            case 'n':
            case 'N':
                e.preventDefault();
                document.getElementById('taskInput').focus();
                break;
            case 'r':
            case 'R':
                e.preventDefault();
                this.loadTasks();
                this.showStatus('Tasks refreshed', 'success');
                break;
            case 'Enter':
            case ' ':
                e.preventDefault();
                this.activateCurrentTask();
                break;
        }
    }

    handleArrowNavigation(key) {
        const currentTab = this.tabNames[this.currentFocus.tab];
        const currentTasks = this.tasks[currentTab];

        if (key === 'ArrowLeft' || key === 'ArrowRight') {
            // Switch tabs
            const direction = key === 'ArrowRight' ? 1 : -1;
            const newTabIndex = Math.max(0, Math.min(this.tabNames.length - 1, this.currentFocus.tab + direction));
            
            if (newTabIndex !== this.currentFocus.tab) {
                this.switchTab(this.tabNames[newTabIndex]);
                this.currentFocus.tab = newTabIndex;
                this.currentFocus.task = Math.min(this.currentFocus.task, this.tasks[this.tabNames[newTabIndex]].length - 1);
                this.updateTaskFocus();
            }
        } else if (key === 'ArrowUp' || key === 'ArrowDown') {
            // Navigate tasks within tab
            if (currentTasks.length === 0) return;
            
            const direction = key === 'ArrowDown' ? 1 : -1;
            const newTaskIndex = Math.max(-1, Math.min(currentTasks.length - 1, this.currentFocus.task + direction));
            
            this.currentFocus.task = newTaskIndex;
            this.updateTaskFocus();
        }
    }

    updateTaskFocus() {
        // Remove existing focus
        document.querySelectorAll('.task-card.keyboard-focus').forEach(card => {
            card.classList.remove('keyboard-focus');
        });

        // Add focus to current task
        if (this.currentFocus.task >= 0) {
            const currentTab = this.tabNames[this.currentFocus.tab];
            const taskCards = document.querySelectorAll(`#${currentTab}List .task-card`);
            
            if (taskCards[this.currentFocus.task]) {
                taskCards[this.currentFocus.task].classList.add('keyboard-focus');
                taskCards[this.currentFocus.task].scrollIntoView({ block: 'nearest' });
            }
        }
    }

    activateCurrentTask() {
        if (this.currentFocus.task >= 0) {
            const currentTab = this.tabNames[this.currentFocus.tab];
            const currentTasks = this.tasks[currentTab];
            
            if (currentTasks[this.currentFocus.task]) {
                const task = currentTasks[this.currentFocus.task];
                const action = this.getDefaultAction(currentTab);
                if (action) {
                    this.handleTaskAction(action, task.id);
                }
            }
        }
    }

    getDefaultAction(status) {
        switch (status) {
            case 'backlog': return 'research';
            case 'staged': return 'implement';
            case 'progress':
            case 'completed': return 'view';
            default: return null;
        }
    }

    createHelpModal() {
        const helpModal = document.createElement('div');
        helpModal.id = 'helpModal';
        helpModal.className = 'modal';
        helpModal.innerHTML = `
            <div class="modal-content">
                <div class="modal-header">
                    <h3>Keyboard Shortcuts</h3>
                    <button class="modal-close" onclick="taskPlanner.closeModals()">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="shortcut-section">
                        <h4>Navigation</h4>
                        <div class="shortcut-item"><kbd>1-4</kbd> Switch between tabs</div>
                        <div class="shortcut-item"><kbd>←→</kbd> Switch tabs</div>
                        <div class="shortcut-item"><kbd>↑↓</kbd> Navigate tasks</div>
                        <div class="shortcut-item"><kbd>Enter/Space</kbd> Activate current task</div>
                    </div>
                    <div class="shortcut-section">
                        <h4>Actions</h4>
                        <div class="shortcut-item"><kbd>N</kbd> New task (focus input)</div>
                        <div class="shortcut-item"><kbd>R</kbd> Refresh tasks</div>
                        <div class="shortcut-item"><kbd>Ctrl+Enter</kbd> Parse tasks (in input)</div>
                        <div class="shortcut-item"><kbd>Esc</kbd> Close modals/blur input</div>
                    </div>
                    <div class="shortcut-section">
                        <h4>Help</h4>
                        <div class="shortcut-item"><kbd>?</kbd> or <kbd>H</kbd> Show this help</div>
                    </div>
                </div>
            </div>
        `;
        document.body.appendChild(helpModal);
    }

    toggleHelpModal() {
        const modal = document.getElementById('helpModal');
        modal.style.display = modal.style.display === 'block' ? 'none' : 'block';
    }

    closeModals() {
        document.querySelectorAll('.modal').forEach(modal => {
            modal.style.display = 'none';
        });
    }
}

// Initialize when DOM is ready
let taskPlanner;
document.addEventListener('DOMContentLoaded', () => {
    taskPlanner = new TaskPlanner();
});