// Swarm Manager - Main Application Logic

const API_BASE = `${window.location.protocol}//${window.location.host}`;
let currentTask = null;
let draggedElement = null;
let refreshInterval = null;

// Initialize application
document.addEventListener('DOMContentLoaded', () => {
    initializeDragAndDrop();
    loadTasks();
    loadAgents();
    loadConfig();
    startAutoRefresh();
});

// Auto-refresh data
function startAutoRefresh() {
    refreshInterval = setInterval(() => {
        loadTasks();
        loadAgents();
        updateMetrics();
    }, 5000); // Refresh every 5 seconds
}

// Load tasks from API
async function loadTasks() {
    try {
        const response = await fetch(`${API_BASE}/api/tasks`);
        const data = await response.json();
        
        // Clear existing tasks
        document.querySelectorAll('.task-container').forEach(container => {
            container.innerHTML = '';
        });
        
        // Group tasks by status and type
        const taskGroups = {
            'backlog-manual': [],
            'backlog-generated': [],
            'staged': [],
            'active': [],
            'completed': [],
            'failed': []
        };
        
        data.tasks.forEach(task => {
            if (task.status === 'backlog') {
                if (task.created_by === 'ai') {
                    taskGroups['backlog-generated'].push(task);
                } else {
                    taskGroups['backlog-manual'].push(task);
                }
            } else {
                taskGroups[task.status]?.push(task);
            }
        });
        
        // Render tasks
        Object.keys(taskGroups).forEach(groupId => {
            const container = document.getElementById(`${groupId}-list`) || 
                            document.getElementById(groupId);
            if (container) {
                taskGroups[groupId].forEach(task => {
                    container.appendChild(createTaskCard(task));
                });
            }
        });
        
        // Update counts
        updateTaskCounts();
        
    } catch (error) {
        console.error('Failed to load tasks:', error);
    }
}

// Create task card element
function createTaskCard(task) {
    const card = document.createElement('div');
    card.className = 'task-card';
    card.draggable = true;
    card.dataset.taskId = task.id;
    card.dataset.status = task.status;
    
    // Calculate priority level
    let priorityClass = 'low';
    if (task.priority_score > 500) priorityClass = 'high';
    else if (task.priority_score > 200) priorityClass = 'medium';
    
    card.innerHTML = `
        ${task.priority_score ? `<div class="task-priority ${priorityClass}">${Math.round(task.priority_score)}</div>` : ''}
        <div class="task-title">${task.title || 'Untitled Task'}</div>
        <div class="task-meta">
            <span class="task-type">${task.type || 'general'}</span>
            <span><i class="fas fa-bullseye"></i> ${task.target || 'none'}</span>
            <span class="task-agent"><i class="fas fa-user-robot"></i> ${getAgentDisplayName(task.assigned_agent)}</span>
            ${task.created_by === 'ai' ? '<span><i class="fas fa-robot"></i> AI</span>' : ''}
        </div>
    `;
    
    card.addEventListener('click', () => showTaskDetails(task));
    
    return card;
}

// Show task details modal
function showTaskDetails(task) {
    currentTask = task;
    
    document.getElementById('task-details-title').textContent = task.title || 'Task Details';
    
    // Reset tabs to details view
    switchTaskTab('details');
    
    const content = document.getElementById('task-details-content');
    content.innerHTML = `
        <div class="task-detail-section">
            <h4>Description</h4>
            <p>${task.description || 'No description provided'}</p>
        </div>
        <div class="task-detail-section">
            <h4>Details</h4>
            <div class="detail-row">
                <span>ID:</span> <span>${task.id}</span>
            </div>
            <div class="detail-row">
                <span>Type:</span> <span>${task.type || 'general'}</span>
            </div>
            <div class="detail-row">
                <span>Target:</span> <span>${task.target || 'none'}</span>
            </div>
            <div class="detail-row">
                <span>Status:</span> <span>${task.status}</span>
            </div>
            <div class="detail-row">
                <span>Created By:</span> <span>${task.created_by || 'unknown'}</span>
            </div>
            <div class="detail-row">
                <span>Priority Score:</span> <span>${task.priority_score || 'Not calculated'}</span>
            </div>
            <div class="detail-row">
                <span>Assigned Agent:</span> <span class="agent-badge">${getAgentDisplayName(task.assigned_agent)}</span>
            </div>
        </div>
        ${task.priority_estimates ? `
        <div class="task-detail-section">
            <h4>Priority Estimates</h4>
            <div class="detail-row">
                <span>Impact:</span> <span>${task.priority_estimates.impact || 'N/A'}</span>
            </div>
            <div class="detail-row">
                <span>Urgency:</span> <span>${task.priority_estimates.urgency || 'N/A'}</span>
            </div>
            <div class="detail-row">
                <span>Success Probability:</span> <span>${task.priority_estimates.success_prob || 'N/A'}</span>
            </div>
            <div class="detail-row">
                <span>Resource Cost:</span> <span>${task.priority_estimates.resource_cost || 'N/A'}</span>
            </div>
        </div>
        ` : ''}
        ${task.notes ? `
        <div class="task-detail-section">
            <h4>Notes</h4>
            <p>${task.notes}</p>
        </div>
        ` : ''}
    `;
    
    document.getElementById('task-details-modal').classList.add('open');
}

function hideTaskDetails() {
    document.getElementById('task-details-modal').classList.remove('open');
    currentTask = null;
    
    // Reset to details tab
    switchTaskTab('details');
}

// Load agents
async function loadAgents() {
    try {
        const response = await fetch(`${API_BASE}/api/agents`);
        const data = await response.json();
        
        const agentList = document.getElementById('agent-list');
        agentList.innerHTML = '';
        
        if (data.agents.length === 0) {
            agentList.innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 1rem;">No active agents</p>';
        } else {
            data.agents.forEach(agent => {
                const agentCard = createAgentCard(agent);
                agentList.appendChild(agentCard);
            });
        }
        
        // Update agent count in header
        document.querySelector('#agent-count span').textContent = `${data.agents.length} Agents`;
        
    } catch (error) {
        console.error('Failed to load agents:', error);
    }
}

// Create agent card
function createAgentCard(agent) {
    const card = document.createElement('div');
    card.className = 'agent-card';
    
    const statusClass = agent.status === 'working' ? '' : 
                       agent.status === 'error' ? 'error' : 'idle';
    
    card.innerHTML = `
        <div class="agent-name">
            <span class="agent-status ${statusClass}"></span>
            ${agent.name}
        </div>
        ${agent.current_task_title ? 
            `<div class="agent-task">Working on: ${agent.current_task_title}</div>` :
            `<div class="agent-task">Idle</div>`
        }
        ${agent.resource_usage ? `
        <div class="agent-resources">
            <span>CPU: ${agent.resource_usage.cpu || 0}%</span>
            <span>MEM: ${agent.resource_usage.memory || 0}%</span>
        </div>
        ` : ''}
    `;
    
    return card;
}

// Load configuration
async function loadConfig() {
    try {
        const response = await fetch(`${API_BASE}/api/config`);
        const config = await response.json();
        
        // Set priority weights
        if (config.priority_weights) {
            setSliderValue('weight-impact', config.priority_weights.impact || 1.0);
            setSliderValue('weight-urgency', config.priority_weights.urgency || 0.8);
            setSliderValue('weight-success', config.priority_weights.success || 0.6);
            setSliderValue('weight-cost', config.priority_weights.cost || 0.5);
        }
        
        // Set system settings
        document.getElementById('yolo-mode').checked = config.yolo_mode || false;
        document.getElementById('max-concurrent').value = config.max_concurrent_tasks || 5;
        document.getElementById('min-backlog').value = config.min_backlog_size || 10;
        
    } catch (error) {
        console.error('Failed to load config:', error);
    }
}

// Set slider value and display
function setSliderValue(sliderId, value) {
    const slider = document.getElementById(sliderId);
    if (slider) {
        slider.value = value;
        const valueDisplay = slider.closest('.weight-input-group').querySelector('.weight-value');
        if (valueDisplay) {
            valueDisplay.textContent = parseFloat(value).toFixed(1);
        }
    }
}

// Update metrics
async function updateMetrics() {
    try {
        const response = await fetch(`${API_BASE}/api/metrics`);
        const metrics = await response.json();
        
        // Update task rate
        const taskRate = metrics.tasks_per_hour || 0;
        document.querySelector('#task-rate span').textContent = `${taskRate}/hr`;
        
    } catch (error) {
        console.error('Failed to update metrics:', error);
    }
}

// Update task counts
function updateTaskCounts() {
    document.querySelectorAll('.task-column').forEach(column => {
        const status = column.dataset.status;
        const taskCount = column.querySelectorAll('.task-card').length;
        const countElement = column.querySelector('.task-count');
        if (countElement) {
            countElement.textContent = taskCount;
        }
    });
}

// Initialize drag and drop
function initializeDragAndDrop() {
    // Add event listeners to all task containers
    document.querySelectorAll('.task-container').forEach(container => {
        container.addEventListener('dragover', handleDragOver);
        container.addEventListener('drop', handleDrop);
        container.addEventListener('dragleave', handleDragLeave);
    });
    
    // Add event listeners to task cards (will be added dynamically)
    document.addEventListener('dragstart', (e) => {
        if (e.target.classList.contains('task-card')) {
            draggedElement = e.target;
            e.target.classList.add('dragging');
            e.dataTransfer.effectAllowed = 'move';
        }
    });
    
    document.addEventListener('dragend', (e) => {
        if (e.target.classList.contains('task-card')) {
            e.target.classList.remove('dragging');
            draggedElement = null;
        }
    });
}

function handleDragOver(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    
    const container = e.currentTarget;
    const afterElement = getDragAfterElement(container, e.clientY);
    
    if (draggedElement && container.classList.contains('task-container')) {
        if (afterElement == null) {
            container.appendChild(draggedElement);
        } else {
            container.insertBefore(draggedElement, afterElement);
        }
    }
}

function handleDrop(e) {
    e.preventDefault();
    
    if (!draggedElement) return;
    
    const newStatus = getStatusFromContainer(e.currentTarget);
    const taskId = draggedElement.dataset.taskId;
    
    // Update task status on backend
    moveTask(taskId, newStatus);
}

function handleDragLeave(e) {
    // Clean up any visual indicators if needed
}

function getDragAfterElement(container, y) {
    const draggableElements = [...container.querySelectorAll('.task-card:not(.dragging)')];
    
    return draggableElements.reduce((closest, child) => {
        const box = child.getBoundingClientRect();
        const offset = y - box.top - box.height / 2;
        
        if (offset < 0 && offset > closest.offset) {
            return { offset: offset, element: child };
        } else {
            return closest;
        }
    }, { offset: Number.NEGATIVE_INFINITY }).element;
}

function getStatusFromContainer(container) {
    const column = container.closest('.task-column');
    if (column) {
        return column.dataset.status;
    }
    
    // Check for specific container IDs
    if (container.id.includes('backlog')) return 'backlog';
    if (container.id.includes('staged')) return 'staged';
    if (container.id.includes('active')) return 'active';
    if (container.id.includes('completed')) return 'completed';
    if (container.id.includes('failed')) return 'failed';
    
    return 'backlog';
}

// Move task to new status
async function moveTask(taskId, newStatus) {
    // TODO: Implement actual file system move via API
    console.log(`Moving task ${taskId} to ${newStatus}`);
    
    // For now, just reload tasks
    setTimeout(loadTasks, 500);
}

// Settings panel
function toggleSettings() {
    const panel = document.getElementById('settings-panel');
    panel.classList.toggle('open');
}

async function saveSettings() {
    const config = {
        priority_weights: {
            impact: parseFloat(document.getElementById('weight-impact').value),
            urgency: parseFloat(document.getElementById('weight-urgency').value),
            success: parseFloat(document.getElementById('weight-success').value),
            cost: parseFloat(document.getElementById('weight-cost').value)
        },
        yolo_mode: document.getElementById('yolo-mode').checked,
        max_concurrent_tasks: parseInt(document.getElementById('max-concurrent').value),
        min_backlog_size: parseInt(document.getElementById('min-backlog').value)
    };
    
    try {
        const response = await fetch(`${API_BASE}/api/config`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(config)
        });
        
        if (response.ok) {
            alert('Settings saved successfully');
            toggleSettings();
        } else {
            alert('Failed to save settings');
        }
    } catch (error) {
        console.error('Failed to save settings:', error);
        alert('Failed to save settings');
    }
}

// Add task modal
function showAddTaskModal() {
    document.getElementById('add-task-modal').classList.add('open');
}

function hideAddTaskModal() {
    document.getElementById('add-task-modal').classList.remove('open');
    // Clear form
    document.getElementById('task-title').value = '';
    document.getElementById('task-description').value = '';
    document.getElementById('task-type').value = 'feature';
    document.getElementById('task-target').value = '';
    document.getElementById('task-agent').value = 'claude-code';
}

async function createTask() {
    const task = {
        title: document.getElementById('task-title').value,
        description: document.getElementById('task-description').value,
        type: document.getElementById('task-type').value,
        target: document.getElementById('task-target').value,
        assigned_agent: document.getElementById('task-agent').value,
        created_by: 'human'
    };
    
    if (!task.title) {
        alert('Task title is required');
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/api/tasks`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(task)
        });
        
        if (response.ok) {
            hideAddTaskModal();
            loadTasks();
        } else {
            alert('Failed to create task');
        }
    } catch (error) {
        console.error('Failed to create task:', error);
        alert('Failed to create task');
    }
}

// Task actions
async function executeCurrentTask() {
    if (!currentTask) return;
    
    try {
        const response = await fetch(`${API_BASE}/api/tasks/${currentTask.id}/execute`, {
            method: 'POST'
        });
        
        if (response.ok) {
            hideTaskDetails();
            loadTasks();
        } else {
            alert('Failed to execute task');
        }
    } catch (error) {
        console.error('Failed to execute task:', error);
        alert('Failed to execute task');
    }
}

async function analyzeCurrentTask() {
    if (!currentTask) return;
    
    try {
        const response = await fetch(`${API_BASE}/api/tasks/${currentTask.id}/analyze`, {
            method: 'POST'
        });
        
        if (response.ok) {
            alert('Task analysis started');
            hideTaskDetails();
            setTimeout(loadTasks, 2000);
        } else {
            alert('Failed to analyze task');
        }
    } catch (error) {
        console.error('Failed to analyze task:', error);
        alert('Failed to analyze task');
    }
}

async function deleteCurrentTask() {
    if (!currentTask) return;
    
    if (!confirm(`Delete task "${currentTask.title}"?`)) return;
    
    try {
        const response = await fetch(`${API_BASE}/api/tasks/${currentTask.id}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            hideTaskDetails();
            loadTasks();
        } else {
            alert('Failed to delete task');
        }
    } catch (error) {
        console.error('Failed to delete task:', error);
        alert('Failed to delete task');
    }
}

// Update slider values in real-time
document.querySelectorAll('input[type="range"]').forEach(slider => {
    slider.addEventListener('input', (e) => {
        const value = parseFloat(e.target.value).toFixed(1);
        e.target.closest('.weight-input-group').querySelector('.weight-value').textContent = value;
    });
});

// Reset settings to defaults
function resetSettings() {
    if (!confirm('Reset all settings to default values?')) return;
    
    // Reset priority weights
    setSliderValue('weight-impact', 1.0);
    setSliderValue('weight-urgency', 0.8);
    setSliderValue('weight-success', 0.6);
    setSliderValue('weight-cost', 0.5);
    
    // Reset system settings
    document.getElementById('yolo-mode').checked = false;
    document.getElementById('max-concurrent').value = 5;
    document.getElementById('min-backlog').value = 10;
    document.getElementById('default-agent').value = 'claude-code';
    
    alert('Settings reset to defaults. Click "Save Settings" to apply.');
}

// Helper function to get display name for agents
function getAgentDisplayName(agentId) {
    const agentNames = {
        'claude-code': 'Claude',
        'general-ai': 'General AI',
        'code-specialist': 'Code Specialist',
        'system-admin': 'System Admin',
        'data-analyst': 'Data Analyst',
        'ui-designer': 'UI Designer'
    };
    return agentNames[agentId] || agentId || 'Claude';
}

// Task modal tab functions
function switchTaskTab(tabName) {
    // Update tab buttons
    document.querySelectorAll('.task-tab').forEach(tab => {
        tab.classList.remove('active');
    });
    document.querySelector(`[onclick="switchTaskTab('${tabName}')"]`).classList.add('active');
    
    // Update tab content
    document.querySelectorAll('.tab-pane').forEach(pane => {
        pane.classList.remove('active');
    });
    
    if (tabName === 'details') {
        document.getElementById('task-details-content').classList.add('active');
    } else if (tabName === 'logs') {
        document.getElementById('task-logs-content').classList.add('active');
        loadTaskLogs();
    } else if (tabName === 'history') {
        document.getElementById('task-history-content').classList.add('active');
        loadTaskHistory();
    }
}

// Load task logs
async function loadTaskLogs() {
    if (!currentTask) return;
    
    const logsLoading = document.getElementById('logs-loading');
    const logsEmpty = document.getElementById('logs-empty');
    const logsList = document.getElementById('logs-list');
    
    // Show loading state
    logsLoading.style.display = 'block';
    logsEmpty.style.display = 'none';
    logsList.innerHTML = '';
    
    try {
        const response = await fetch(`${API_BASE}/api/tasks/${currentTask.id}/logs`);
        const data = await response.json();
        
        logsLoading.style.display = 'none';
        
        if (data.events.length === 0) {
            logsEmpty.style.display = 'block';
            return;
        }
        
        // Display events
        data.events.forEach(event => {
            const logEntry = createLogEntry(event);
            logsList.appendChild(logEntry);
        });
        
    } catch (error) {
        console.error('Failed to load task logs:', error);
        logsLoading.style.display = 'none';
        logsEmpty.style.display = 'block';
        document.getElementById('logs-empty').innerHTML = '<i class="fas fa-exclamation-triangle"></i> Failed to load logs.';
    }
}

// Create log entry element
function createLogEntry(event) {
    const entry = document.createElement('div');
    entry.className = 'log-entry';
    
    const typeClass = event.type === 'start' ? 'start' : 
                     event.type === 'finish' ? 'finish' : 'error';
    
    const typeIcon = event.type === 'start' ? 'play' : 
                    event.type === 'finish' ? 'check' : 'exclamation-triangle';
    
    const timestamp = new Date(event.ts).toLocaleString();
    const duration = event.duration_sec ? `${event.duration_sec}s` : '';
    const exitCode = event.exit_code !== undefined ? `Exit: ${event.exit_code}` : '';
    
    entry.innerHTML = `
        <div class="log-header">
            <div class="log-type ${typeClass}">
                <i class="fas fa-${typeIcon}"></i>
                ${event.type.charAt(0).toUpperCase() + event.type.slice(1)} ${event.task}
                ${event.scenario ? `<span class="log-scenario">${event.scenario}</span>` : ''}
            </div>
            <div class="log-timestamp">${timestamp}</div>
        </div>
        <div class="log-details">
            PID: ${event.pid}
            ${duration ? ` | Duration: ${duration}` : ''}
            ${exitCode ? ` | ${exitCode}` : ''}
        </div>
        ${event.error ? `<div class="log-error">${event.error}</div>` : ''}
    `;
    
    return entry;
}

// Load task execution history
async function loadTaskHistory() {
    if (!currentTask) return;
    
    const historyContainer = document.getElementById('execution-history');
    historyContainer.innerHTML = '<div class="logs-loading"><i class="fas fa-spinner fa-pulse"></i> Loading execution history...</div>';
    
    try {
        const response = await fetch(`${API_BASE}/api/tasks/${currentTask.id}/logs`);
        const data = await response.json();
        
        historyContainer.innerHTML = '';
        
        if (data.executions.length === 0) {
            historyContainer.innerHTML = '<div class="logs-empty"><i class="fas fa-info-circle"></i> No execution history available.</div>';
            return;
        }
        
        // Display execution history
        data.executions.forEach(execution => {
            const historyEntry = createHistoryEntry(execution);
            historyContainer.appendChild(historyEntry);
        });
        
    } catch (error) {
        console.error('Failed to load task history:', error);
        historyContainer.innerHTML = '<div class="logs-empty"><i class="fas fa-exclamation-triangle"></i> Failed to load execution history.</div>';
    }
}

// Create history entry element
function createHistoryEntry(execution) {
    const entry = document.createElement('div');
    entry.className = 'execution-entry';
    
    const startTime = execution.started_at ? new Date(execution.started_at).toLocaleString() : 'Not started';
    const endTime = execution.completed_at ? new Date(execution.completed_at).toLocaleString() : 'Not completed';
    const duration = execution.duration_seconds ? `${execution.duration_seconds}s` : 'N/A';
    
    entry.innerHTML = `
        <div class="execution-header">
            <div class="execution-title">
                <strong>${execution.task_title}</strong>
                ${execution.scenario_used ? `<span class="log-scenario">${execution.scenario_used}</span>` : ''}
            </div>
            <div class="execution-status ${execution.status}">${execution.status}</div>
        </div>
        <div class="execution-details">
            <div class="execution-detail">
                <span>Started:</span> <strong>${startTime}</strong>
            </div>
            <div class="execution-detail">
                <span>Completed:</span> <strong>${endTime}</strong>
            </div>
            <div class="execution-detail">
                <span>Duration:</span> <strong>${duration}</strong>
            </div>
            <div class="execution-detail">
                <span>Priority Score:</span> <strong>${execution.priority_score || 'N/A'}</strong>
            </div>
        </div>
        ${execution.error_details ? `<div class="execution-error">${execution.error_details}</div>` : ''}
        ${execution.output_summary ? `<div class="execution-summary">Output: ${execution.output_summary}</div>` : ''}
    `;
    
    return entry;
}

// Refresh task logs
function refreshTaskLogs() {
    loadTaskLogs();
}

// Add styles for task details
const style = document.createElement('style');
style.textContent = `
.task-detail-section {
    margin-bottom: 1.5rem;
}

.task-detail-section h4 {
    color: var(--text-primary);
    font-size: 0.9rem;
    font-weight: 600;
    margin-bottom: 0.5rem;
}

.task-detail-section p {
    color: var(--text-secondary);
    font-size: 0.85rem;
    line-height: 1.5;
}

.detail-row {
    display: flex;
    justify-content: space-between;
    padding: 0.25rem 0;
    font-size: 0.85rem;
}

.detail-row span:first-child {
    color: var(--text-secondary);
    font-weight: 500;
}

.detail-row span:last-child {
    color: var(--text-primary);
}

.agent-badge {
    background: linear-gradient(45deg, var(--cyber-cyan), var(--matrix-green)) !important;
    color: var(--primary-bg) !important;
    padding: 0.2rem 0.5rem !important;
    border-radius: 12px !important;
    font-size: 0.75rem !important;
    font-weight: 600 !important;
    text-transform: uppercase !important;
    letter-spacing: 0.5px !important;
}

.task-agent {
    background: var(--panel-bg);
    color: var(--cyber-cyan);
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    font-size: 0.7rem;
    border: 1px solid var(--cyber-cyan);
}
`;
document.head.appendChild(style);