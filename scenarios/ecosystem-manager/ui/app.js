// Ecosystem Manager - Main Application
import { TaskManager } from './modules/TaskManager.js';
import { SettingsManager } from './modules/SettingsManager.js';
import { ProcessMonitor } from './modules/ProcessMonitor.js';
import { UIComponents } from './modules/UIComponents.js';
import { WebSocketHandler } from './modules/WebSocketHandler.js';
import { DragDropHandler } from './modules/DragDropHandler.js';

class EcosystemManager {
    constructor() {
        // API Configuration - Use relative path so Vite proxy handles it
        this.apiBase = '/api';
        
        // Initialize modules
        this.taskManager = new TaskManager(
            this.apiBase, 
            this.showToast.bind(this), 
            this.showLoading.bind(this)
        );
        this.settingsManager = new SettingsManager(this.apiBase, this.showToast.bind(this));
        this.processMonitor = new ProcessMonitor(this.apiBase, this.showToast.bind(this));
        this.webSocketHandler = new WebSocketHandler(
            this.apiBase, 
            this.handleWebSocketMessage.bind(this)
        );
        this.dragDropHandler = new DragDropHandler(this.handleTaskDrop.bind(this));
        
        // State
        this.isLoading = false;
        this.rateLimitEndTime = null;
        this.refreshCountdownInterval = null;
        this.lastRefreshTime = Date.now();
        this.refreshInterval = 30; // Default 30 seconds
        
        // Bind methods
        this.init = this.init.bind(this);
        this.refreshAll = this.refreshAll.bind(this);
    }

    async init() {
        console.log('Initializing Ecosystem Manager...');
        
        // Initialize UI
        this.initializeUI();

        // Prepare process monitor dropdown interactions
        this.processMonitor.initializeDropdown();

        // Ensure cached theme is applied
        SettingsManager.applyCachedTheme();
        
        // Load initial data
        await this.loadInitialData();
        
        // Start monitoring
        await this.processMonitor.startMonitoring();
        
        // Connect WebSocket
        this.webSocketHandler.connect();
        
        // Set up event listeners
        this.setupEventListeners();
        
        // Initialize drag and drop
        this.dragDropHandler.initializeDragDrop();
        
        // Load resources and scenarios in the background (non-blocking)
        // This prevents slow startup while still having the data ready when needed
        setTimeout(() => {
            this.loadAvailableResourcesAndScenarios().catch(err => 
                console.error('Failed to load resources/scenarios:', err)
            );
        }, 100);
        
        
        console.log('Ecosystem Manager initialized');
    }

    initializeUI() {
        // Set up modals
        this.setupModals();
        
        // Initialize tabs if they exist
        const tabButtons = document.querySelectorAll('.tab-button');
        tabButtons.forEach(button => {
            button.addEventListener('click', (e) => this.switchTab(e.target.dataset.tab));
        });
        
        // Set active tab
        if (tabButtons.length > 0) {
            this.switchTab('tasks');
        }
    }

    setupModals() {
        // Close modal when clicking outside
        document.querySelectorAll('.modal').forEach(modal => {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    modal.classList.remove('show');
                    // Re-enable body scroll when closing modal by clicking backdrop
                    document.body.style.overflow = '';
                }
            });
        });
    }

    async loadInitialData() {
        try {
            // Load settings first
            const settings = await this.settingsManager.loadSettings();
            this.settingsManager.applySettingsToUI(settings);
            
            // Store refresh interval from settings
            this.refreshInterval = settings.refresh_interval || 30;
            
            // Start refresh countdown timer if processor is active
            if (settings.active) {
                this.startRefreshCountdown();
            }
            
            // Load tasks
            await this.loadAllTasks();
            
            // Fetch queue processor status
            await this.fetchQueueProcessorStatus();
            
            // Update grid layout after initial load
            this.updateGridLayout();
            
        } catch (error) {
            console.error('Error loading initial data:', error);
            this.showToast('Failed to load initial data', 'error');
        }
    }
    
    startRefreshCountdown() {
        // Clear any existing countdown
        if (this.refreshCountdownInterval) {
            clearInterval(this.refreshCountdownInterval);
        }
        
        // Update the countdown every second
        this.refreshCountdownInterval = setInterval(() => {
            const now = Date.now();
            const elapsed = Math.floor((now - this.lastRefreshTime) / 1000);
            const remaining = Math.max(0, this.refreshInterval - elapsed);
            
            const countdownElement = document.getElementById('refresh-countdown');
            if (countdownElement) {
                countdownElement.textContent = remaining;
            }
            
            // If countdown reaches 0, reset the timer
            if (remaining === 0) {
                this.lastRefreshTime = now;
            }
        }, 1000);
    }
    
    stopRefreshCountdown() {
        if (this.refreshCountdownInterval) {
            clearInterval(this.refreshCountdownInterval);
            this.refreshCountdownInterval = null;
        }
        
        const countdownElement = document.getElementById('refresh-countdown');
        if (countdownElement) {
            countdownElement.textContent = '--';
        }
    }

    setupEventListeners() {
        // Refresh button
        const refreshBtn = document.getElementById('refresh-all-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.refreshAll());
        }
        
        // Create task button
        const createTaskBtn = document.getElementById('create-task-btn');
        if (createTaskBtn) {
            createTaskBtn.addEventListener('click', () => this.showCreateTaskModal());
        }
        
        // Create task form
        const createTaskForm = document.getElementById('create-task-form');
        if (createTaskForm) {
            createTaskForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                await this.handleCreateTask();
            });
        }
        
        // Settings form
        const settingsForm = document.getElementById('settings-form');
        if (settingsForm) {
            settingsForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                await this.saveSettingsFromForm();
            });
        }
        
        // Queue processor toggle
        const processorToggle = document.getElementById('queue-processor-toggle');
        if (processorToggle) {
            processorToggle.addEventListener('change', async (e) => {
                await this.toggleQueueProcessor(e.target.checked);
            });
        }
    }

    // Task Management Methods
    async loadAllTasks() {
        const statuses = ['pending', 'in-progress', 'completed', 'failed'];
        const promises = statuses.map(status => this.loadTasksForStatus(status));
        await Promise.all(promises);
    }

    async loadTasksForStatus(status) {
        try {
            const tasks = await this.taskManager.loadTasks(status);
            this.renderTasks(tasks, status);
        } catch (error) {
            if (error.isRateLimit) {
                this.handleRateLimit(error.retryAfter);
            } else {
                console.error(`Error loading ${status} tasks:`, error);
                this.showToast(`Failed to load ${status} tasks`, 'error');
            }
        }
    }

    renderTasks(tasks, status) {
        const container = document.getElementById(`${status}-tasks`);
        if (!container) return;
        
        container.innerHTML = '';
        
        if (tasks.length === 0) {
            container.innerHTML = `<div class="empty-state">No ${status} tasks</div>`;
            return;
        }
        
        tasks.forEach(task => {
            const card = UIComponents.createTaskCard(task, this.processMonitor.runningProcesses);
            this.dragDropHandler.setupTaskCardDragHandlers(card, task.id, status);
            
            // Add click handler for task details
            card.addEventListener('click', (e) => {
                // Handle delete button click
                const deleteBtn = e.target.closest('.task-delete-btn');
                if (deleteBtn) {
                    e.stopPropagation();
                    const taskId = deleteBtn.dataset.taskId;
                    const taskStatus = deleteBtn.dataset.taskStatus;
                    this.deleteTask(taskId, taskStatus);
                } else {
                    this.showTaskDetails(task.id);
                }
            });
            
            container.appendChild(card);
        });
        
        // Update counter
        const counter = document.querySelector(`[data-status="${status}"] .task-count`);
        if (counter) {
            counter.textContent = tasks.length;
        }
    }

    async showCreateTaskModal() {
        const modal = document.getElementById('create-task-modal');
        if (modal) {
            // Initialize the form with default values
            await this.updateFormForType();
            await this.updateFormForOperation();
            modal.classList.add('show');
            // Disable body scroll when showing modal
            document.body.style.overflow = 'hidden';
        }
    }

    async handleCreateTask() {
        const form = document.getElementById('create-task-form');
        const formData = new FormData(form);
        
        const taskData = {
            title: formData.get('title'),
            type: formData.get('type'),
            operation: formData.get('operation'),
            priority: formData.get('priority'),
            notes: formData.get('notes'),
            status: 'pending'
        };
        
        // Add target for improver operations
        if (taskData.operation === 'improver') {
            taskData.target = formData.get('target');
        }
        
        this.showLoading(true);
        
        try {
            const result = await this.taskManager.createTask(taskData);
            
            if (result.success) {
                this.showToast('Task created successfully', 'success');
                this.closeModal('create-task-modal');
                form.reset(); // Reset the form
                await this.refreshColumn('pending');
            } else {
                throw new Error(result.error || 'Failed to create task');
            }
        } catch (error) {
            console.error('Error creating task:', error);
            this.showToast(`Failed to create task: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async showTaskDetails(taskId) {
        this.showLoading(true);
        
        try {
            const task = await this.taskManager.getTaskDetails(taskId);
            this.renderTaskDetailsModal(task);
        } catch (error) {
            console.error('Failed to load task details:', error);
            this.showToast(`Failed to load task details: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    renderTaskDetailsModal(task) {
        const modal = document.getElementById('task-details-modal');
        const titleElement = document.getElementById('task-details-title');
        const contentElement = document.getElementById('task-details-content');
        
        titleElement.textContent = 'Edit Task';
        
        // Use the enhanced two-column layout
        contentElement.innerHTML = this.getTaskDetailsHTML(task);
        
        modal.classList.add('show');
        // Disable body scroll when showing modal
        document.body.style.overflow = 'hidden';
    }

    getTaskDetailsHTML(task) {
        const isRunning = this.processMonitor.isTaskRunning(task.id);
        const runningProcess = isRunning ? this.processMonitor.getRunningProcess(task.id) : null;
        
        return `
            <form id="edit-task-form">
                <div class="task-details-container task-details-grid">
                    <!-- Left Column: Form Fields -->
                    <div class="task-form-column">
                        <!-- Basic Information -->
                        <div class="form-group">
                            <label for="edit-task-title">Title *</label>
                            <input type="text" id="edit-task-title" name="title" value="${this.escapeHtml(task.title)}" required>
                        </div>
                        
                        <!-- Task Status and Priority -->
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-status">Status</label>
                                <select id="edit-task-status" name="status">
                                    <option value="pending" ${task.status === 'pending' ? 'selected' : ''}>Pending</option>
                                    <option value="in-progress" ${task.status === 'in-progress' ? 'selected' : ''}>In Progress</option>
                                    <option value="completed" ${task.status === 'completed' ? 'selected' : ''}>Completed</option>
                                    <option value="failed" ${task.status === 'failed' ? 'selected' : ''}>Failed</option>
                                </select>
                            </div>
                            
                            <div class="form-group">
                                <label for="edit-task-priority">Priority</label>
                                <select id="edit-task-priority" name="priority">
                                    <option value="low" ${task.priority === 'low' ? 'selected' : ''}>Low</option>
                                    <option value="medium" ${task.priority === 'medium' ? 'selected' : ''}>Medium</option>
                                    <option value="high" ${task.priority === 'high' ? 'selected' : ''}>High</option>
                                    <option value="critical" ${task.priority === 'critical' ? 'selected' : ''}>Critical</option>
                                </select>
                            </div>
                        </div>
                        
                        <!-- Current Phase and Operation Type -->
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-phase">Current Phase</label>
                                <select id="edit-task-phase" name="current_phase">
                                    <option value="">No Phase</option>
                                    <option value="pending" ${task.current_phase === 'pending' ? 'selected' : ''}>Pending</option>
                                    <option value="in-progress" ${task.current_phase === 'in-progress' ? 'selected' : ''}>In Progress</option>
                                    <option value="completed" ${task.current_phase === 'completed' ? 'selected' : ''}>Completed</option>
                                    <option value="failed" ${task.current_phase === 'failed' ? 'selected' : ''}>Failed</option>
                                </select>
                            </div>
                            
                            <div class="form-group">
                                <label for="edit-task-operation">Type</label>
                                <select id="edit-task-operation" name="operation">
                                    <option value="generator" ${task.operation === 'generator' ? 'selected' : ''}>Generator</option>
                                    <option value="improver" ${task.operation === 'improver' ? 'selected' : ''}>Improver</option>
                                </select>
                            </div>
                        </div>
                        
                        <!-- Notes -->
                        <div class="form-group">
                            <label for="edit-task-notes">Notes</label>
                            <textarea id="edit-task-notes" name="notes" rows="16" 
                                      placeholder="Additional details, requirements, or context...">${this.escapeHtml(task.notes || '')}</textarea>
                        </div>
                    </div>
                    
                    <!-- Right Column: Execution Results and Task Information -->
                    <div class="task-info-column">
                        ${this.getTaskExecutionInfoHTML(task, isRunning, runningProcess)}
                    </div>
                </div>
                
                <!-- Action Buttons -->
                <div class="form-actions">
                    <button type="button" class="btn btn-secondary" onclick="ecosystemManager.closeModal('task-details-modal')">
                        <i class="fas fa-times"></i>
                        Cancel
                    </button>
                    
                    <button type="button" class="btn btn-info" onclick="ecosystemManager.viewTaskPrompt('${task.id}')" title="View the prompt that was/will be sent to Claude">
                        <i class="fas fa-file-alt"></i>
                        View Prompt
                    </button>
                    
                    <button type="button" class="btn btn-primary" onclick="ecosystemManager.saveTaskChanges('${task.id}')">
                        <i class="fas fa-save"></i>
                        Save Changes
                    </button>
                </div>
            </form>
        `;
    }

    getTaskExecutionInfoHTML(task, isRunning, runningProcess) {
        let html = '';
        
        // Process Controls
        if (isRunning && runningProcess) {
            html += `
                <div class="task-execution-status executing">
                    <i class="fas fa-brain fa-spin"></i>
                    <span>Task is currently executing with Claude Code</span>
                    <span class="execution-timer">${this.processMonitor.formatDuration(runningProcess.start_time)}</span>
                    <button type="button" class="btn btn-secondary" onclick="ecosystemManager.processMonitor.openLogViewer('${task.id}')">
                        <i class="fas fa-terminal"></i>
                        Follow Logs
                    </button>
                    <button type="button" class="process-terminate-btn" onclick="ecosystemManager.terminateProcess('${task.id}')">
                        <i class="fas fa-stop"></i>
                        Terminate
                    </button>
                </div>
            `;
        }
        
        // Task Results
        if (task.results && (task.status === 'completed' || task.status === 'failed')) {
            html += this.getTaskResultsHTML(task.results);
        }
        
        // Task Information
        html += `
            <div class="form-group">
                <label>Task Information</label>
                <div style="background: var(--light-gray); padding: 0.8rem; border-radius: var(--border-radius); font-size: 0.9rem;">
                    <div><strong>ID:</strong> ${task.id}</div>
                    ${task.created_at ? `<div><strong>Created:</strong> ${new Date(task.created_at).toLocaleString()}</div>` : ''}
                    ${task.started_at ? `<div><strong>Started:</strong> ${new Date(task.started_at).toLocaleString()}</div>` : ''}
                    ${task.completed_at ? `<div><strong>Completed:</strong> ${new Date(task.completed_at).toLocaleString()}</div>` : ''}
                </div>
            </div>
        `;
        
        return html;
    }

    getTaskResultsHTML(results) {
        return `
            <div class="form-group">
                <label>Execution Results</label>
                <div class="execution-results ${results.success ? 'success' : 'error'}">
                    <div style="margin-bottom: 0.5rem;">
                        <strong>Status:</strong> 
                        <span class="${results.success ? 'status-success' : 'status-error'}">
                            ${results.success ? '✅ Success' : '❌ Failed'}
                        </span>
                        ${results.timeout_failure ? '<span style="color: #ff9800; margin-left: 8px;">⏰ TIMEOUT</span>' : ''}
                    </div>
                    
                    ${results.execution_time || results.timeout_allowed || results.prompt_size ? `
                        <div style="margin-bottom: 0.5rem; padding: 0.5rem; background: rgba(0, 0, 0, 0.05); border-radius: 4px;">
                            <div style="display: flex; justify-content: space-between; font-size: 0.9em;">
                                ${results.execution_time ? `<span><strong>⏱️ Runtime:</strong> ${results.execution_time}</span>` : ''}
                                ${results.timeout_allowed ? `<span><strong>⏰ Timeout:</strong> ${results.timeout_allowed}</span>` : ''}
                            </div>
                            ${results.prompt_size ? `<div style="font-size: 0.9em; margin-top: 4px;"><strong>📝 Prompt Size:</strong> ${results.prompt_size}</div>` : ''}
                            ${results.started_at ? `<div style="font-size: 0.8em; color: #666; margin-top: 4px;">Started: ${new Date(results.started_at).toLocaleString()}</div>` : ''}
                        </div>
                    ` : ''}
                    
                    ${results.error ? `
                        <div style="margin-bottom: 0.5rem;">
                            <strong>Error:</strong> 
                            <pre class="status-error" style="margin-top: 0.5rem; padding: 0.5rem; background: rgba(244, 67, 54, 0.1); border-radius: 4px; white-space: pre-wrap;">${this.escapeHtml(this.taskManager.formatErrorText(results.error))}</pre>
                        </div>
                    ` : ''}

                    ${results.max_turns_exceeded ? `
                        <div style="margin-bottom: 0.5rem;">
                            <strong>Max Turns:</strong>
                            <div class="status-warning" style="margin-top: 0.5rem; padding: 0.5rem; background: rgba(255, 193, 7, 0.15); border-radius: 4px;">
                                Claude stopped after reaching the configured MAX_TURNS limit. Increase the limit or simplify the task.
                            </div>
                        </div>
                    ` : ''}
                    
                    ${results.output ? `
                        <details style="margin-top: 0.5rem;">
                            <summary class="output-summary">
                                📋 View Claude Output (click to expand)
                            </summary>
                            <pre class="claude-output">${this.escapeHtml(results.output)}</pre>
                        </details>
                    ` : ''}
                </div>
            </div>
        `;
    }

    async saveTaskChanges(taskId) {
        const form = document.getElementById('edit-task-form');
        const formData = new FormData(form);
        
        const updates = {
            title: formData.get('title'),
            status: formData.get('current_phase') || formData.get('status'), // Map current_phase to status for file movement
            priority: formData.get('priority'),
            current_phase: formData.get('current_phase'),
            operation: formData.get('operation'),
            notes: formData.get('notes')
        };
        
        this.showLoading(true);
        
        try {
            const result = await this.taskManager.updateTask(taskId, updates);
            
            if (result.success) {
                this.showToast('Task updated successfully', 'success');
                this.closeModal('task-details-modal');
                await this.refreshAll();
            } else {
                throw new Error(result.error || 'Failed to update task');
            }
        } catch (error) {
            console.error('Error updating task:', error);
            this.showToast(`Failed to update task: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async deleteTask(taskId, status) {
        if (!confirm('Are you sure you want to delete this task?')) {
            return;
        }
        
        this.showLoading(true);
        
        try {
            const result = await this.taskManager.deleteTask(taskId, status);
            
            if (result.success) {
                this.showToast('Task deleted successfully', 'success');
                await this.refreshColumn(status);
            } else {
                throw new Error(result.error || 'Failed to delete task');
            }
        } catch (error) {
            console.error('Error deleting task:', error);
            this.showToast(`Failed to delete task: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async viewTaskPrompt(taskId) {
        this.showLoading(true);
        
        try {
            const data = await this.taskManager.getTaskPrompt(taskId);
            this.showPromptModal(data);
        } catch (error) {
            console.error('Error loading prompt:', error);
            this.showToast(`Failed to load prompt: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    showPromptModal(data) {
        const modal = document.getElementById('prompt-viewer-modal');
        const contentElement = document.getElementById('prompt-content');
        
        if (modal && contentElement) {
            let displayContent = '';
            
            // Check if we have the actual assembled prompt
            if (data.prompt && typeof data.prompt === 'string') {
                // We have the actual assembled prompt - show it
                displayContent = data.prompt;
                
                // Add metadata header
                let metadata = '=== PROMPT METADATA ===\n';
                metadata += `Task ID: ${data.task_id || 'Unknown'}\n`;
                metadata += `Operation: ${data.operation || 'Unknown'}\n`;
                metadata += `Prompt Length: ${data.prompt_length || data.prompt.length} characters\n`;
                if (data.prompt_cached) {
                    metadata += `Source: Cached prompt (from previous execution)\n`;
                } else {
                    metadata += `Source: Freshly assembled\n`;
                }
                metadata += `\n${'='.repeat(50)}\n\n`;
                
                displayContent = metadata + displayContent;
            } else {
                // Fallback: show configuration data if actual prompt is not available
                displayContent = '=== PROMPT CONFIGURATION ===\n\n';
                displayContent += 'Note: This shows the prompt configuration. The actual assembled prompt is not available.\n\n';
                
                if (data.operation_config) {
                    displayContent += `Operation: ${data.operation_config.name || data.operation}\n`;
                    displayContent += `Type: ${data.operation_config.type || ''}\n`;
                    displayContent += `Target: ${data.operation_config.target || ''}\n`;
                    displayContent += `Description: ${data.operation_config.description || ''}\n`;
                }
                
                if (data.task_details) {
                    displayContent += '\n=== TASK DETAILS ===\n';
                    displayContent += `ID: ${data.task_details.id || ''}\n`;
                    displayContent += `Title: ${data.task_details.title || ''}\n`;
                    displayContent += `Type: ${data.task_details.type || ''}\n`;
                    displayContent += `Operation: ${data.task_details.operation || ''}\n`;
                }
                
                if (data.prompt_sections) {
                    displayContent += '\n=== PROMPT SECTIONS ===\n';
                    data.prompt_sections.forEach((section, i) => {
                        displayContent += `  ${i + 1}. ${section}\n`;
                    });
                }
            }
            
            contentElement.textContent = displayContent || 'No prompt data available';
            modal.classList.add('show');
            // Disable body scroll when showing modal
            document.body.style.overflow = 'hidden';
        }
    }

    // Drag and Drop
    async handleTaskDrop(taskId, fromStatus, toStatus) {
        this.showLoading(true);
        
        try {
            // If moving from in-progress to any other status, automatically terminate the running process
            if (fromStatus === 'in-progress' && toStatus !== 'in-progress') {
                const isRunning = this.processMonitor.isTaskRunning(taskId);
                if (isRunning) {
                    console.log(`Auto-terminating running process for task ${taskId} (moved from in-progress to ${toStatus})`);
                    try {
                        await this.processMonitor.terminateProcess(taskId);
                        this.showToast('Running task automatically stopped', 'info');
                    } catch (terminateError) {
                        console.warn('Failed to auto-terminate process:', terminateError);
                        // Continue with the move even if termination fails
                    }
                }
            }
            
            // Clear task state when moving to specific columns
            const updates = { status: toStatus };
            
            // Set appropriate current_phase based on the target status
            // Use empty string to clear, as the backend preserves non-empty values
            if (toStatus === 'pending') {
                updates.current_phase = '';  // Empty string to clear
            } else if (toStatus === 'in-progress') {
                updates.current_phase = 'in-progress';
            } else if (toStatus === 'completed') {
                updates.current_phase = 'completed';
            } else if (toStatus === 'failed') {
                updates.current_phase = 'failed';
            }
            
            const result = await this.taskManager.updateTask(taskId, updates);
            
            if (result.success) {
                this.showToast(`Task moved to ${toStatus}`, 'success');
                
                // Give the backend a moment to complete the file move operation
                // This prevents seeing duplicates during the transition
                await new Promise(resolve => setTimeout(resolve, 300));
                
                // Refresh both columns to update task counts and positions
                await Promise.all([
                    this.loadTasksForStatus(fromStatus),
                    this.loadTasksForStatus(toStatus)
                ]);
            } else {
                throw new Error(result.error || 'Failed to move task');
            }
        } catch (error) {
            console.error('Error moving task:', error);
            this.showToast(`Failed to move task: ${error.message}`, 'error');
            
            // Small delay before refresh to let backend settle
            await new Promise(resolve => setTimeout(resolve, 300));
            
            // Reload both columns to restore correct state
            await Promise.all([
                this.loadTasksForStatus(fromStatus),
                this.loadTasksForStatus(toStatus)
            ]);
        } finally {
            this.showLoading(false);
        }
    }

    // Settings Management
    async saveSettingsFromForm() {
        const settings = this.settingsManager.getSettingsFromForm();
        
        this.showLoading(true);
        
        try {
            const result = await this.settingsManager.saveSettings(settings);
            
            if (result.success) {
                // Apply theme immediately after successful save and update original theme
                this.settingsManager.applyTheme(settings.theme || 'light');
                this.settingsManager.originalTheme = settings.theme || 'light'; // Update original theme
                
                // Update processor status UI immediately
                this.settingsManager.updateProcessorToggleUI(settings.active);
                
                this.showToast('Settings saved successfully', 'success');
                this.closeModal('settings-modal');
            } else {
                throw new Error(result.error || 'Failed to save settings');
            }
        } catch (error) {
            console.error('Error saving settings:', error);
            this.showToast(`Failed to save settings: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async toggleQueueProcessor(enabled) {
        const settings = { ...this.settingsManager.settings, queueProcessingEnabled: enabled };
        
        try {
            const result = await this.settingsManager.saveSettings(settings);
            
            if (result.success) {
                this.settingsManager.updateProcessorToggleUI(enabled);
                this.showToast(`Queue processor ${enabled ? 'enabled' : 'disabled'}`, 'success');
            } else {
                throw new Error(result.error || 'Failed to toggle queue processor');
            }
        } catch (error) {
            console.error('Error toggling queue processor:', error);
            this.showToast(`Failed to toggle queue processor: ${error.message}`, 'error');
            // Reset toggle
            document.getElementById('queue-processor-toggle').checked = !enabled;
        }
    }

    // Queue Processing
    async fetchQueueProcessorStatus() {
        try {
            const response = await fetch(`${this.apiBase}/queue/status`);
            if (!response.ok) {
                throw new Error(`Failed to fetch queue status: ${response.statusText}`);
            }
            
            const data = await response.json();
            this.updateQueueStatusUI(data);
        } catch (error) {
            console.error('Error fetching queue status:', error);
        }
    }

    updateQueueStatusUI(status) {
        // Update queue metrics
        const metrics = {
            'queue-pending': status.pending_count || 0,
            'queue-running': status.running_count || 0,
            'queue-completed': status.completed_count || 0,
            'queue-failed': status.failed_count || 0
        };
        
        Object.entries(metrics).forEach(([id, value]) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = value;
            }
        });

        const availableSlotsEl = document.getElementById('available-slots');
        if (availableSlotsEl && typeof status.available_slots === 'number') {
            availableSlotsEl.textContent = status.available_slots;
        }

        const maxSlotsEl = document.getElementById('max-slots');
        if (maxSlotsEl && typeof status.max_concurrent === 'number') {
            maxSlotsEl.textContent = status.max_concurrent;
        }

        // Update last processed time
        if (status.last_processed_at) {
            const element = document.getElementById('last-processed-time');
            if (element) {
                element.textContent = new Date(status.last_processed_at).toLocaleString();
            }
        }
    }

    async triggerQueueProcessing() {
        this.showLoading(true);
        
        try {
            const response = await fetch(`${this.apiBase}/queue/trigger`, {
                method: 'POST'
            });
            
            if (!response.ok) {
                throw new Error(`Failed to trigger processing: ${response.statusText}`);
            }
            
            const result = await response.json();
            this.showToast('Queue processing triggered successfully', 'success');
            await this.fetchQueueProcessorStatus();
        } catch (error) {
            console.error('Error triggering queue processing:', error);
            this.showToast(`Failed to trigger processing: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    // Process Management
    async terminateProcess(taskId) {
        if (!confirm('Are you sure you want to terminate this process?')) {
            return;
        }
        
        this.showLoading(true);
        
        try {
            const result = await this.processMonitor.terminateProcess(taskId);
            
            if (result.success) {
                this.showToast('Process terminated successfully', 'success');
                await this.refreshAll();
            } else {
                throw new Error(result.error || 'Failed to terminate process');
            }
        } catch (error) {
            console.error('Error terminating process:', error);
            this.showToast(`Failed to terminate process: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async handleProcessStarted(info) {
        const taskId = typeof info === 'string' ? info : (info?.task_id || info?.id);
        if (!taskId) {
            return;
        }

        const eventData = (info && typeof info === 'object') ? info : {};
        const startTime = eventData.start_time ? new Date(eventData.start_time) : new Date();
        const startIso = startTime.toISOString();
        const agentId = eventData.agent_id || '';
        const processId = eventData.process_id;

        // Get the task's current status
        try {
            const task = await this.taskManager.getTaskDetails(taskId);
            const oldCard = document.getElementById(`task-${taskId}`);
            const oldStatus = oldCard ? oldCard.closest('.kanban-column')?.dataset.status : task.status;
            
            // If task is not already in-progress, move it there
            if (task.status !== 'in-progress') {
                // Update task status to in-progress
                await this.taskManager.updateTask(taskId, { status: 'in-progress' });
                
                // Refresh both columns to move the task
                await Promise.all([
                    this.loadTasksForStatus(oldStatus),
                    this.loadTasksForStatus('in-progress')
                ]);
            } else {
                // Task already in in-progress, just update the card UI
                const card = document.getElementById(`task-${taskId}`);
                if (card) {
                    card.classList.add('task-executing');
                    
                    // Add execution indicator if not present
                    if (!card.querySelector('.task-execution-indicator')) {
                        const indicator = document.createElement('div');
                        indicator.className = 'task-execution-indicator';
                        indicator.innerHTML = `
                            <i class="fas fa-brain fa-spin"></i>
                            <span>Executing with Claude...</span>
                        `;
                        card.appendChild(indicator);
                    }
                }
            }

            // Record running state for downstream UI components
            this.processMonitor.runningProcesses[task.id] = {
                task_id: task.id,
                status: 'running',
                start_time: startIso,
                agent_id: agentId,
                process_id: processId,
                duration: this.processMonitor.formatDuration(startIso)
            };

            this.refreshTaskCard(task.id);
            this.startElapsedTimeCounter(task.id, startTime);
            this.processMonitor.renderProcessWidget(this.processMonitor.runningProcesses);
        } catch (error) {
            console.error('Error handling process start:', error);
        }
    }

    updateTaskProgress(task) {
        const card = document.getElementById(`task-${task.id}`);
        if (card) {
            // Update progress indicator
            const phaseElement = card.querySelector('.task-phase');
            if (phaseElement) {
                phaseElement.innerHTML = `<i class="fas fa-spinner fa-pulse"></i> ${task.current_phase || 'Processing'}`;
            } else if (task.current_phase) {
                // Add phase element if it doesn't exist
                const titleElement = card.querySelector('.task-title');
                if (titleElement) {
                    const newPhase = document.createElement('div');
                    newPhase.className = 'task-phase';
                    newPhase.innerHTML = `<i class="fas fa-spinner fa-pulse"></i> ${task.current_phase}`;
                    titleElement.insertAdjacentElement('afterend', newPhase);
                }
            }
        }
    }

    handleClaudeExecutionStarted(task) {
        const card = document.getElementById(`task-${task.id}`);
        if (card) {
            // Add executing class
            card.classList.add('task-executing');
            
            // Update or add execution indicator
            let indicator = card.querySelector('.task-execution-indicator');
            if (!indicator) {
                indicator = document.createElement('div');
                indicator.className = 'task-execution-indicator';
                card.appendChild(indicator);
            }
            
            const startTime = new Date();
            indicator.innerHTML = `
                <i class="fas fa-brain fa-spin"></i>
                <div class="execution-details">
                    <span>Executing with Claude...</span>
                    <div class="phase-info">${task.current_phase || 'Processing'}</div>
                    <div class="duration-info" id="duration-${task.id}">Running: 0s</div>
                </div>
                <button class="btn-stop-execution" onclick="event.stopPropagation(); ecosystemManager.stopTaskExecution('${task.id}')" title="Stop execution">
                    <i class="fas fa-stop"></i>
                </button>
            `;
            
            // Also update process monitor tracking with duration calculation
            this.processMonitor.runningProcesses[task.id] = {
                task_id: task.id,
                status: 'running',
                start_time: startTime.toISOString(),
                duration: '0s'
            };
            
            // Start elapsed time counter
            this.startElapsedTimeCounter(task.id, startTime);
            this.processMonitor.renderProcessWidget(this.processMonitor.runningProcesses);
        }
    }

    startElapsedTimeCounter(taskId, startTime) {
        // Clear any existing timer
        if (this.elapsedTimers && this.elapsedTimers[taskId]) {
            clearInterval(this.elapsedTimers[taskId]);
        }
        
        // Initialize timers object if needed
        if (!this.elapsedTimers) {
            this.elapsedTimers = {};
        }
        
        // Update elapsed time every second
        this.elapsedTimers[taskId] = setInterval(() => {
            const durationElement = document.getElementById(`duration-${taskId}`);
            if (durationElement) {
                const elapsed = Math.floor((Date.now() - startTime.getTime()) / 1000);
                const hours = Math.floor(elapsed / 3600);
                const minutes = Math.floor((elapsed % 3600) / 60);
                const seconds = elapsed % 60;
                
                let durationText;
                if (hours > 0) {
                    durationText = `${hours}h ${minutes}m ${seconds}s`;
                } else if (minutes > 0) {
                    durationText = `${minutes}m ${seconds}s`;
                } else {
                    durationText = `${seconds}s`;
                }
                
                durationElement.textContent = `Running: ${durationText}`;
                
                // Also update the process monitor data
                if (this.processMonitor.runningProcesses[taskId]) {
                    this.processMonitor.runningProcesses[taskId].duration = durationText;
                }

                const chip = document.querySelector(`.process-detail-item[data-task-id="${taskId}"]`);
                if (chip) {
                    const chipParts = [taskId, durationText];
                    const agent = this.processMonitor.runningProcesses[taskId]?.agent_id;
                    if (agent) chipParts.push(agent);
                    chip.textContent = chipParts.filter(Boolean).join(' · ');
                }
            } else {
                // Element no longer exists, clear timer
                clearInterval(this.elapsedTimers[taskId]);
                delete this.elapsedTimers[taskId];
            }
        }, 1000);
    }

    async handleProcessCompleted(taskId) {
        // Clean up elapsed timer
        if (this.elapsedTimers && this.elapsedTimers[taskId]) {
            clearInterval(this.elapsedTimers[taskId]);
            delete this.elapsedTimers[taskId];
        }
        
        // Clean up process monitor tracking
        if (this.processMonitor.runningProcesses[taskId]) {
            delete this.processMonitor.runningProcesses[taskId];
        }
        this.processMonitor.renderProcessWidget(this.processMonitor.runningProcesses);

        // Refresh the task to get updated results and check if it moved to a new column
        try {
            const task = await this.taskManager.getTaskDetails(taskId);
            const oldCard = document.getElementById(`task-${taskId}`);
            const oldStatus = oldCard ? oldCard.closest('.kanban-column')?.dataset.status : null;
            
            // If task moved to a different status, refresh both columns
            if (oldStatus && oldStatus !== task.status) {
                await Promise.all([
                    this.loadTasksForStatus(oldStatus),
                    this.loadTasksForStatus(task.status)
                ]);
            } else {
                // Just refresh the task card in place
                this.refreshTaskCard(taskId);
            }
        } catch (error) {
            console.error('Error handling process completion:', error);
            // Fallback: refresh the task card
            this.refreshTaskCard(taskId);
        }
    }

    async refreshTaskCard(taskId) {
        try {
            const task = await this.taskManager.getTaskDetails(taskId);
            const card = document.getElementById(`task-${taskId}`);
            
            if (card) {
                const newCard = UIComponents.createTaskCard(task, this.processMonitor.runningProcesses);
                this.dragDropHandler.setupTaskCardDragHandlers(newCard, task.id, task.status);
                
                // Add click handler
                newCard.addEventListener('click', (e) => {
                    // Handle delete button click
                    const deleteBtn = e.target.closest('.task-delete-btn');
                    if (deleteBtn) {
                        e.stopPropagation();
                        const taskId = deleteBtn.dataset.taskId;
                        const taskStatus = deleteBtn.dataset.taskStatus;
                        this.deleteTask(taskId, taskStatus);
                    } else {
                        this.showTaskDetails(task.id);
                    }
                });
                
                card.replaceWith(newCard);
            }
        } catch (error) {
            console.error('Error refreshing task card:', error);
        }
    }

    // WebSocket Handling
    handleWebSocketMessage(message) {
        console.log('WebSocket message:', message);
        
        switch (message.type) {
            case 'task_started':
                this.handleProcessStarted(message.data || message);
                break;
            case 'task_progress':
                // Update task card to show progress
                if (message.data) {
                    this.updateTaskProgress(message.data);
                }
                break;
            case 'task_executing':
                // Claude Code has started executing
                if (message.data) {
                    this.handleClaudeExecutionStarted(message.data);
                }
                break;
            case 'claude_execution_complete':
                // Claude Code execution finished
                if (message.data) {
                    this.refreshTaskCard(message.data.id);
                }
                break;
            case 'task_completed':
                this.handleProcessCompleted(message.task_id || message.data?.id);
                break;
            case 'task_failed':
                this.handleProcessCompleted(message.task_id || message.data?.id);
                break;
            case 'task_status_changed':
                // Handle real-time task status changes
                this.handleTaskStatusChanged(message.data.task_id, message.data.old_status, message.data.new_status);
                break;
            case 'queue_status':
                this.updateQueueStatusUI(message.data);
                break;
            case 'log_entry':
                if (this.processMonitor && message.data) {
                    this.processMonitor.addLogEntry(message.data);
                }
                break;
        }
    }

    async handleTaskStatusChanged(taskId, oldStatus, newStatus) {
        console.log(`Task ${taskId} status changed from ${oldStatus} to ${newStatus}`);
        // Handle real-time task status changes by refreshing both affected columns
        if (oldStatus !== newStatus) {
            await Promise.all([
                this.loadTasksForStatus(oldStatus),
                this.loadTasksForStatus(newStatus)
            ]);
        }
    }

    // UI Helper Methods
    async refreshAll() {
        this.showLoading(true);
        
        try {
            await this.loadAllTasks();
            await this.fetchQueueProcessorStatus();
            this.showToast('All data refreshed', 'success');
        } catch (error) {
            console.error('Error refreshing data:', error);
            this.showToast('Failed to refresh data', 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async refreshColumn(status) {
        await this.loadTasksForStatus(status);
    }

    switchTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.tab-button').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tabName);
        });
        
        // Update tab content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.toggle('active', content.id === `${tabName}-tab`);
        });
    }

    switchSettingsTab(tabName) {
        // Update settings tab buttons
        document.querySelectorAll('.settings-tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tabName);
        });
        
        // Update settings tab content
        document.querySelectorAll('.settings-tab-content').forEach(content => {
            content.classList.toggle('active', content.dataset.tab === tabName);
        });
    }

    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            // If closing settings modal, revert theme preview
            if (modalId === 'settings-modal') {
                this.settingsManager.revertThemePreview();
            }
            modal.classList.remove('show');
            // Re-enable body scroll when closing modal
            document.body.style.overflow = '';
        }
    }

    showToast(message, type = 'info') {
        UIComponents.showToast(message, type);
    }

    showLoading(show) {
        UIComponents.showLoading(show);
        this.isLoading = show;
    }

    handleRateLimit(retryAfter) {
        this.rateLimitEndTime = Date.now() + (retryAfter * 1000);
        UIComponents.showRateLimitNotification(retryAfter);
    }

    dismissRateLimitNotification(button) {
        UIComponents.dismissRateLimitNotification(button);
    }

    escapeHtml(text) {
        return UIComponents.escapeHtml(text);
    }

    // Additional UI Methods for HTML handlers
    openSettingsModal() {
        const modal = document.getElementById('settings-modal');
        if (modal) {
            modal.classList.add('show');
            // Disable body scroll when showing modal
            document.body.style.overflow = 'hidden';
            // Load current settings into form
            this.settingsManager.loadSettings().then(settings => {
                this.settingsManager.applySettingsToUI(settings);
            });
        }
    }

    previewTheme(theme) {
        // Apply theme immediately for preview, but don't save it yet
        this.settingsManager.applyTheme(theme);
    }

    resetSettingsToDefault() {
        if (confirm('Are you sure you want to reset all settings to defaults?')) {
            this.settingsManager.resetToDefaults();
            this.showToast('Settings reset to defaults', 'success');
        }
    }

    async toggleProcessor() {
        try {
            // Get current settings first
            const currentSettings = await this.settingsManager.loadSettings();
            
            // Toggle the active state
            const newActiveState = !currentSettings.active;
            const updatedSettings = { ...currentSettings, active: newActiveState };
            
            this.showLoading(true);
            
            // Save the updated settings
            const result = await this.settingsManager.saveSettings(updatedSettings);
            
            if (result.success) {
                // Update UI immediately
                this.settingsManager.updateProcessorToggleUI(newActiveState);
                
                // Start or stop refresh countdown based on active state
                if (newActiveState) {
                    this.startRefreshCountdown();
                } else {
                    this.stopRefreshCountdown();
                }
                
                this.showToast(`Processor ${newActiveState ? 'activated' : 'paused'}`, 'success');
            } else {
                throw new Error(result.error || 'Failed to toggle processor');
            }
        } catch (error) {
            console.error('Error toggling processor:', error);
            this.showToast(`Failed to toggle processor: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async stopTaskExecution(taskId) {
        if (!confirm('Are you sure you want to stop this task execution?')) {
            return;
        }

        try {
            this.showLoading(true);
            
            const response = await fetch(`${this.apiBase}/queue/processes/terminate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    task_id: taskId
                })
            });
            
            if (!response.ok) {
                throw new Error(`Failed to stop execution: ${response.statusText}`);
            }
            
            const result = await response.json();
            
            if (result.success) {
                this.showToast('Task execution stopped', 'success');
                if (this.processMonitor.runningProcesses[taskId]) {
                    delete this.processMonitor.runningProcesses[taskId];
                    this.processMonitor.renderProcessWidget(this.processMonitor.runningProcesses);
                }
                // Refresh the task card to show updated state
                await this.refreshTaskCard(taskId);
            } else {
                throw new Error(result.message || 'Failed to stop execution');
            }
        } catch (error) {
            console.error('Error stopping task execution:', error);
            this.showToast(`Failed to stop execution: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    applyFilters() {
        const type = document.getElementById('filter-type')?.value;
        const operation = document.getElementById('filter-operation')?.value;
        const priority = document.getElementById('filter-priority')?.value;
        const searchText = document.getElementById('search-input')?.value?.toLowerCase();

        document.querySelectorAll('.task-card').forEach(card => {
            let visible = true;

            if (type && !card.classList.contains(type)) visible = false;
            if (operation && !card.classList.contains(operation)) visible = false;
            if (priority && !card.classList.contains(priority)) visible = false;
            if (searchText) {
                const title = card.querySelector('.task-title')?.textContent?.toLowerCase();
                const notes = card.querySelector('.task-notes')?.textContent?.toLowerCase();
                if (!title?.includes(searchText) && !notes?.includes(searchText)) {
                    visible = false;
                }
            }

            card.style.display = visible ? 'block' : 'none';
        });
    }

    clearFilters() {
        document.getElementById('filter-type').value = '';
        document.getElementById('filter-operation').value = '';
        document.getElementById('filter-priority').value = '';
        document.getElementById('search-input').value = '';
        this.applyFilters();
    }

    async updateFormForType() {
        // When type changes and we're in improver mode, reload the targets
        const operation = document.querySelector('input[name="operation"]:checked')?.value;
        if (operation === 'improver') {
            await this.loadAvailableTargets();
        }
    }

    async updateFormForOperation() {
        const operation = document.querySelector('input[name="operation"]:checked')?.value;
        const targetGroup = document.getElementById('target-group');
        if (targetGroup) {
            targetGroup.style.display = operation === 'improver' ? 'block' : 'none';
            
            // Load available targets for improver operations
            if (operation === 'improver') {
                // Check if resources/scenarios are loaded yet
                if (!this.availableResources && !this.availableScenarios) {
                    // Try to load them now if not already loaded
                    await this.loadAvailableResourcesAndScenarios();
                }
                await this.loadAvailableTargets();
            }
        }
    }

    async loadAvailableResourcesAndScenarios() {
        try {
            // Load both in parallel for better performance
            const [resourcesResponse, scenariosResponse] = await Promise.all([
                fetch(`${this.apiBase}/resources`),
                fetch(`${this.apiBase}/scenarios`)
            ]);
            
            if (resourcesResponse.ok) {
                this.availableResources = await resourcesResponse.json();
            } else {
                this.availableResources = [];
            }
            
            if (scenariosResponse.ok) {
                this.availableScenarios = await scenariosResponse.json();
            } else {
                this.availableScenarios = [];
            }
            
            console.log('Loaded resources:', this.availableResources?.length || 0, 'items');
            console.log('Loaded scenarios:', this.availableScenarios?.length || 0, 'items');
        } catch (error) {
            console.error('Error loading resources and scenarios:', error);
            this.availableResources = [];
            this.availableScenarios = [];
        }
    }

    async loadAvailableTargets() {
        const type = document.querySelector('input[name="type"]:checked')?.value;
        const targetSelect = document.getElementById('task-target');
        
        if (!targetSelect) return;
        
        // Show loading state
        targetSelect.disabled = true;
        targetSelect.innerHTML = '<option value="">Loading available targets...</option>';
        
        // If resources/scenarios aren't loaded yet, wait a bit for them
        if (!this.availableResources && !this.availableScenarios) {
            // Wait up to 3 seconds for resources to load
            let waitCount = 0;
            while ((!this.availableResources && !this.availableScenarios) && waitCount < 30) {
                await new Promise(resolve => setTimeout(resolve, 100));
                waitCount++;
            }
        }
        
        // Use pre-loaded resources or scenarios
        const availableTargets = type === 'resource' ? this.availableResources : this.availableScenarios;
        
        // Enable and populate the select
        targetSelect.disabled = false;
        
        if (!availableTargets || availableTargets.length === 0) {
            targetSelect.innerHTML = `<option value="">No ${type}s available to improve</option>`;
        } else {
            targetSelect.innerHTML = '<option value="">Select target to improve...</option>';
            availableTargets.forEach(target => {
                const option = document.createElement('option');
                // Use the name property which both resources and scenarios have
                option.value = target.name;
                // Display name with status if available
                const statusLabel = target.status === 'implemented' ? ' ✓' : '';
                option.textContent = target.name + statusLabel;
                targetSelect.appendChild(option);
            });
        }
    }

    updateSliderValue(sliderId, valueId) {
        const slider = document.getElementById(sliderId);
        const valueDisplay = document.getElementById(valueId);
        if (slider && valueDisplay) {
            valueDisplay.textContent = slider.value;
        }
    }

    resetRateLimit() {
        this.rateLimitEndTime = null;
        this.showToast('Rate limit pause has been reset', 'success');
        const statusElement = document.getElementById('rate-limit-status');
        if (statusElement) {
            statusElement.innerHTML = '<span style="color: var(--success-color);">No rate limit currently active</span>';
        }
    }

    toggleLogAutoScroll() {
        if (this.processMonitor) {
            this.processMonitor.toggleAutoScroll();
        }
    }

    clearLogViewer() {
        if (this.processMonitor) {
            this.processMonitor.clearLogs();
        }
    }

    closeLogViewer() {
        const modal = document.getElementById('log-viewer-modal');
        if (modal) {
            modal.classList.remove('show');
            // Re-enable body scroll when closing modal
            document.body.style.overflow = '';
        }
    }

    hideColumn(status) {
        const column = document.querySelector(`[data-status="${status}"]`);
        if (column) {
            column.classList.add('hidden');
            this.updateGridLayout();
            this.showToast(`${status.charAt(0).toUpperCase() + status.slice(1)} column hidden`, 'info');
        }
    }

    showColumn(status) {
        const column = document.querySelector(`[data-status="${status}"]`);
        if (column) {
            column.classList.remove('hidden');
            this.updateGridLayout();
            this.showToast(`${status.charAt(0).toUpperCase() + status.slice(1)} column shown`, 'info');
        }
    }

    updateGridLayout() {
        const board = document.querySelector('.kanban-board');
        if (!board) return;

        // Count visible columns
        const visibleColumns = board.querySelectorAll('.kanban-column:not(.hidden)').length;
        
        // Remove all column classes
        board.classList.remove('columns-1', 'columns-2', 'columns-3', 'columns-4');
        
        // Add appropriate class based on visible columns
        if (visibleColumns > 0) {
            board.classList.add(`columns-${visibleColumns}`);
        }
    }
}

// Initialize the application when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.ecosystemManager = new EcosystemManager();
    window.ecosystemManager.init();
});
