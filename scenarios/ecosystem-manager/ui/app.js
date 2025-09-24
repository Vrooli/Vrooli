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
        this.systemLogs = [];
        this.systemLogsFiltered = [];
        this.systemLogLevelFilter = 'all';
        this.promptOperations = null;
        this.promptPreviewLoading = false;
        this.lastPromptPreview = null;
        this.titleAutofillActive = false;
        this.lastAutofilledTitle = '';
        this.targetSelector = null;
        this.targetHelpElement = null;
        this.targetHelpDefault = '';
        this.tasksByStatus = {
            pending: [],
            'in-progress': [],
            review: [],
            completed: [],
            failed: []
        };
        this.pendingTargetRefresh = null;

        // Bind methods
        this.init = this.init.bind(this);
        this.refreshAll = this.refreshAll.bind(this);
    }

    async init() {
        console.log('Initializing Ecosystem Manager...');
        
        // Initialize UI
        this.initializeUI();

        // Pre-load prompt configuration metadata for the tester panel
        this.loadPromptOperations().catch(err => console.error('Failed to load prompt operations:', err));

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
        this.initializeTargetSelector();
        
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

    getSelectedTargets() {
        const targetSelect = document.getElementById('task-target');
        if (!targetSelect) {
            return [];
        }

        return Array.from(targetSelect.selectedOptions || [])
            .map(option => (option.value || '').trim())
            .filter(Boolean);
    }

    async handleImproverBulkCreate(taskData, targets, form) {
        const uniqueTargets = [];
        const seen = new Set();

        targets.forEach(target => {
            const key = target.toLowerCase();
            if (!seen.has(key)) {
                seen.add(key);
                uniqueTargets.push(target);
            }
        });

        const summary = {
            created: [],
            skipped: [],
            errors: []
        };

        for (const target of uniqueTargets) {
            const targetValue = (target || '').trim();
            if (!targetValue) {
                continue;
            }

            const payload = {
                ...taskData,
                target: targetValue,
                title: this.resolveTitleForTarget(taskData.title, taskData.operation, taskData.type, targetValue)
            };

            try {
                const result = await this.taskManager.createTask(payload);

                if (result?.task) {
                    summary.created.push({ target: targetValue, taskId: result.task.id });
                } else if (Array.isArray(result?.created) && result.created.length) {
                    result.created.forEach(entry => {
                        summary.created.push({
                            target: entry.target || targetValue,
                            taskId: entry.id
                        });
                    });

                    (result.skipped || []).forEach(entry => {
                        summary.skipped.push({
                            target: entry.target || targetValue,
                            reason: entry.reason || 'Already has active task'
                        });
                    });

                    (result.errors || []).forEach(entry => {
                        summary.errors.push({
                            target: entry.target || targetValue,
                            message: entry.error || entry.message || 'Unknown error'
                        });
                    });
                } else if (result?.success) {
                    // Some handlers may only signal success without returning the task payload
                    summary.created.push({ target: targetValue, taskId: result.task?.id || 'new-task' });
                } else {
                    summary.errors.push({ target: targetValue, message: 'Unexpected response from API' });
                }
            } catch (error) {
                if (error && error.status === 409) {
                    summary.skipped.push({
                        target: targetValue,
                        reason: error.message || 'Task already exists'
                    });
                } else {
                    summary.errors.push({
                        target: targetValue,
                        message: error.message || 'Failed to create task'
                    });
                }
            }
        }

        await this.handleBulkCreationOutcome(summary, form);
    }

    async handleTaskCreationResult(result, form) {
        if (!result?.success || !result?.task) {
            throw new Error(result?.error || 'Failed to create task');
        }

        this.showToast('Task created successfully', 'success');
        this.closeModal('create-task-modal');
        form.reset();
        this.resetCreateTaskTitleState();
        await this.refreshColumn('pending');
    }

    async handleBulkCreationOutcome(summary, form) {
        const createdCount = summary.created.length;
        const skippedCount = summary.skipped.length;
        const errorCount = summary.errors.length;

        const createdTargets = summary.created.map(entry => entry.target).filter(Boolean);
        const skippedTargets = summary.skipped.map(entry => entry.target).filter(Boolean);
        const errorDetails = summary.errors.map(entry => `${entry.target}: ${entry.message}`);

        if (createdCount > 0) {
            const messageParts = [`Created ${createdCount} task${createdCount > 1 ? 's' : ''}`];

            if (createdTargets.length) {
                messageParts.push(`Targets: ${createdTargets.join(', ')}`);
            }

            if (skippedCount > 0) {
                messageParts.push(`Skipped existing targets: ${skippedTargets.join(', ')}`);
            }

            if (errorCount > 0) {
                messageParts.push(`Errors: ${errorDetails.join('; ')}`);
            }

            const toastType = errorCount > 0 ? 'warning' : (skippedCount > 0 ? 'info' : 'success');
            this.showToast(messageParts.join('. '), toastType);

            this.closeModal('create-task-modal');
            form.reset();
            this.resetCreateTaskTitleState();
            await this.refreshColumn('pending');
        } else if (errorCount === 0 && skippedCount > 0) {
            this.showToast(`No tasks created. Already tracked: ${skippedTargets.join(', ')}`, 'info');
        } else if (errorCount > 0) {
            this.showToast(`Failed to create tasks: ${errorDetails.join('; ')}`, 'error');
        }
    }

    resolveTitleForTarget(baseTitle, operation, type, target) {
        const trimmedBase = (baseTitle || '').trim();
        const trimmedTarget = (target || '').trim();

        if (!trimmedTarget) {
            return trimmedBase || this.generateTaskTitle(operation, type, trimmedTarget);
        }

        if (trimmedBase.includes('{{target}}')) {
            return trimmedBase.replaceAll('{{target}}', trimmedTarget);
        }

        if (trimmedBase && trimmedBase.toLowerCase().includes(trimmedTarget.toLowerCase())) {
            return trimmedBase;
        }

        if (!trimmedBase) {
            return this.generateTaskTitle(operation, type, trimmedTarget);
        }

        return `${trimmedBase} (${trimmedTarget})`;
    }

    async fetchActiveTargetMap(type, operation) {
        const map = new Map();

        if (!type || !operation) {
            return map;
        }

        try {
            const params = new URLSearchParams({ type, operation });
            const response = await fetch(`${this.apiBase}/tasks/active-targets?${params.toString()}`);

            if (!response.ok) {
                throw new Error(`Failed to load active targets: ${response.status} ${response.statusText}`);
            }

            const data = await response.json();
            if (Array.isArray(data)) {
                data.forEach(entry => {
                    const targetValue = (entry?.target || '').trim();
                    const key = this.getTargetKey(type, operation, targetValue);
                    if (!key || map.has(key)) {
                        return;
                    }

                    map.set(key, {
                        taskId: entry.task_id || entry.taskId || '',
                        status: entry.status || '',
                        statusLabel: (entry.status || '').replace('-', ' ') || '',
                        title: entry.title || '',
                        target: targetValue
                    });
                });
            }
        } catch (error) {
            console.error('Failed to fetch active targets:', error);
        }

        return map;
    }

    extractTaskTargets(task) {
        if (!task) {
            return [];
        }

        if (Array.isArray(task.targets) && task.targets.length > 0) {
            return task.targets.filter(Boolean);
        }

        if (task.target) {
            return [task.target];
        }

        const inferred = this.inferLegacyTarget(task);
        return inferred ? [inferred] : [];
    }

    getTargetKey(type, operation, target) {
        const normalizedTarget = (target || '').trim().toLowerCase();
        if (!normalizedTarget) {
            return '';
        }

        return `${type}::${operation}::${normalizedTarget}`;
    }

    inferLegacyTarget(task) {
        if (!task || typeof task.title !== 'string') {
            return '';
        }

        const title = task.title.trim();
        if (!title) {
            return '';
        }

        const legacyPattern = /(enhance|improve|upgrade|fix|polish)\s+(resource|scenario)\s+([^\-\(\[]+)/i;
        const match = title.match(legacyPattern);
        if (match && match[3]) {
            return match[3].trim();
        }

        return '';
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

    initializeTargetSelector() {
        const wrapper = document.getElementById('task-target-selector');
        const select = document.getElementById('task-target');

        if (!wrapper || !select) {
            return;
        }

        const selection = wrapper.querySelector('.tag-multiselect-selection');
        const tagsContainer = wrapper.querySelector('.tag-multiselect-tags');
        const searchInput = wrapper.querySelector('#task-target-search');
        const dropdown = wrapper.querySelector('.tag-multiselect-dropdown');
        const optionsContainer = wrapper.querySelector('.tag-multiselect-options');
        const statusElement = wrapper.querySelector('.tag-multiselect-status');

        if (!selection || !tagsContainer || !searchInput || !dropdown || !optionsContainer || !statusElement) {
            console.warn('Target selector DOM is incomplete; skipping enhanced selector initialization');
            return;
        }

        this.targetHelpElement = document.getElementById('task-target-help');
        if (this.targetHelpElement) {
            this.targetHelpDefault = (this.targetHelpElement.textContent || '').trim();
            if (!this.targetHelpDefault) {
                this.targetHelpDefault = 'Select one or more targets to enhance. Already-tracked targets appear greyed out with their existing task ID.';
            }
            this.targetHelpElement.dataset.tone = this.targetHelpElement.dataset.tone || 'info';
        }

        this.targetSelector = new TagMultiSelect({
            selectElement: select,
            wrapperElement: wrapper,
            selectionElement: selection,
            tagsContainer,
            searchInput,
            dropdownElement: dropdown,
            optionsContainer,
            statusElement,
            placeholder: wrapper.dataset.placeholder || 'Search targets...'
        });

        this.targetSelector.setNoResultsMessage('No targets match that search');
    }

    setTargetHelp(message, tone = 'info') {
        if (!this.targetHelpElement) {
            return;
        }

        const trimmed = typeof message === 'string' ? message.trim() : '';

        if (trimmed) {
            this.targetHelpElement.textContent = trimmed;
            this.targetHelpElement.dataset.tone = tone;
        } else {
            this.targetHelpElement.textContent = this.targetHelpDefault;
            this.targetHelpElement.dataset.tone = 'info';
        }
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

        const taskTargetSelect = document.getElementById('task-target');
        if (taskTargetSelect) {
            taskTargetSelect.addEventListener('change', () => this.handleTargetSelectionChange());
        }

        const taskTitleInput = document.getElementById('task-title');
        if (taskTitleInput) {
            taskTitleInput.addEventListener('input', () => this.handleTitleInputChange());
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

        // Prompt tester controls
        const promptTypeSelect = document.getElementById('prompt-type');
        if (promptTypeSelect) {
            promptTypeSelect.addEventListener('change', () => {
                const currentOperation = document.getElementById('prompt-operation')?.value;
                this.updatePromptOperationOptions(currentOperation);
            });
        }

        const promptOperationSelect = document.getElementById('prompt-operation');
        if (promptOperationSelect) {
            promptOperationSelect.addEventListener('change', () => this.updatePromptOperationSummary());
        }

        const promptPreviewBtn = document.getElementById('prompt-preview-btn');
        if (promptPreviewBtn) {
            promptPreviewBtn.addEventListener('click', () => this.handlePromptPreview());
        }

        const promptCopyIcon = document.getElementById('prompt-preview-copy-icon');
        if (promptCopyIcon) {
            promptCopyIcon.addEventListener('click', () => this.handlePromptCopy());
        }

        const promptDetailsToggle = document.getElementById('prompt-preview-toggle');
        if (promptDetailsToggle) {
            promptDetailsToggle.addEventListener('click', () => this.togglePromptDetails());
        }
    }

    // Task Management Methods
    async loadAllTasks() {
        const statuses = ['pending', 'in-progress', 'review', 'completed', 'failed'];
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
                this.tasksByStatus[status] = [];
                this.scheduleTargetAvailabilityRefresh();
            }
        }
    }

    renderTasks(tasks, status) {
        const container = document.getElementById(`${status}-tasks`);
        if (!container) return;

        const normalizedTasks = Array.isArray(tasks) ? tasks : [];
        this.tasksByStatus[status] = normalizedTasks;

        container.innerHTML = '';
        
        if (normalizedTasks.length === 0) {
            container.innerHTML = `<div class="empty-state">No ${status} tasks</div>`;
            this.scheduleTargetAvailabilityRefresh();
            return;
        }
        
        normalizedTasks.forEach(task => {
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
            counter.textContent = normalizedTasks.length;
        }

        this.scheduleTargetAvailabilityRefresh();
    }

    scheduleTargetAvailabilityRefresh() {
        if (this.pendingTargetRefresh) {
            clearTimeout(this.pendingTargetRefresh);
        }

        this.pendingTargetRefresh = setTimeout(() => {
            this.pendingTargetRefresh = null;

            const modal = document.getElementById('create-task-modal');
            if (!modal || !modal.classList.contains('show')) {
                return;
            }

            const operation = document.querySelector('input[name="operation"]:checked')?.value;
            if (operation !== 'improver') {
                return;
            }

            this.loadAvailableTargets().catch(err => {
                console.error('Failed to refresh target availability:', err);
            });
        }, 150);
    }

    async showCreateTaskModal() {
        const modal = document.getElementById('create-task-modal');
        if (modal) {
            // Initialize the form with default values
            this.resetCreateTaskTitleState();
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
            title: (formData.get('title') || '').trim(),
            type: formData.get('type'),
            operation: formData.get('operation'),
            priority: formData.get('priority'),
            notes: formData.get('notes'),
            status: 'pending'
        };

        const selectedTargets = this.getSelectedTargets();

        this.showLoading(true);
        
        try {
            if (taskData.operation === 'improver') {
                taskData.title = '';
                if (selectedTargets.length === 0) {
                    this.showToast('Select at least one target to enhance', 'warning');
                    return;
                }

                await this.handleImproverBulkCreate(taskData, selectedTargets, form);
            } else {
                const result = await this.taskManager.createTask(taskData);
                await this.handleTaskCreationResult(result, form);
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
                                      placeholder="Additional notes or context...">${this.escapeHtml(task.notes || '')}</textarea>
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
                    ${Number.isInteger(task.completion_count) ? `<div><strong>Runs Completed:</strong> ${task.completion_count}</div>` : ''}
                    ${task.last_completed_at ? `<div><strong>Last Completed:</strong> ${new Date(task.last_completed_at).toLocaleString()}</div>` : (!task.last_completed_at && task.completed_at ? `<div><strong>Last Completed:</strong> ${new Date(task.completed_at).toLocaleString()}</div>` : '')}
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
                this.updatePromptOperationOptions();
                this.updatePromptOperationSummary();
                this.setPromptPreviewStatus('');
            });
        }
    }

    async openLogsModal() {
        const modal = document.getElementById('system-logs-modal');
        if (!modal) return;

        modal.classList.add('show');
        document.body.style.overflow = 'hidden';
        await this.refreshSystemLogs();
    }

    async refreshSystemLogs() {
        const container = document.getElementById('system-logs-container');
        if (container) {
            container.innerHTML = '<div class="logs-empty">Loading logs...</div>';
        }

        try {
            const response = await fetch(`${this.apiBase}/logs?limit=500`);
            if (!response.ok) {
                throw new Error(`Failed to load logs: ${response.status} ${response.statusText}`);
            }

            const data = await response.json();
            this.systemLogs = data.entries || [];
            this.systemLogsFiltered = this.systemLogs;
            this.renderSystemLogs();
        } catch (error) {
            console.error('Failed to load system logs:', error);
            if (container) {
                container.innerHTML = `<div class="logs-empty">${this.escapeHtml(error.message || 'Failed to load logs')}</div>`;
            }
            this.showToast('Failed to load API logs', 'error');
        }
    }

    renderSystemLogs() {
        const container = document.getElementById('system-logs-container');
        if (!container) return;

        const logs = this.systemLogsFiltered || [];

        if (logs.length === 0) {
            container.innerHTML = '<div class="logs-empty">No logs available.</div>';
            return;
        }

        const rows = logs.map(entry => {
            const ts = entry.timestamp ? new Date(entry.timestamp).toLocaleString() : '';
            const level = (entry.level || 'info').toLowerCase();
            const message = this.escapeHtml(entry.message || '');
            return `
                <div class="log-entry">
                    <span class="log-timestamp">${this.escapeHtml(ts)}</span>
                    <span class="log-level ${level}">${this.escapeHtml(level)}</span>
                    <span class="log-message">${message}</span>
                </div>
            `;
        });

        container.innerHTML = rows.join('');
    }

    filterSystemLogs() {
        const select = document.getElementById('system-log-level');
        if (select) {
            this.systemLogLevelFilter = select.value || 'all';
        }

        if (!this.systemLogs || this.systemLogs.length === 0) {
            this.systemLogsFiltered = [];
            this.renderSystemLogs();
            return;
        }

        if (this.systemLogLevelFilter === 'all') {
            this.systemLogsFiltered = this.systemLogs;
        } else {
            const targetLevel = this.systemLogLevelFilter.toLowerCase();
            this.systemLogsFiltered = this.systemLogs.filter(entry => {
                const level = (entry.level || '').toLowerCase();
                return level === targetLevel;
            });
        }

        this.renderSystemLogs();
    }

    async copySystemLogs() {
        try {
            const lines = (this.systemLogsFiltered || []).map(entry => {
                const ts = entry.timestamp ? new Date(entry.timestamp).toISOString() : '';
                const level = entry.level || '';
                const message = entry.message || '';
                return `[${level}] ${ts} ${message}`;
            });

            const text = lines.join('\n');
            await navigator.clipboard.writeText(text);
            this.showToast('Logs copied to clipboard', 'success');
        } catch (error) {
            console.error('Failed to copy logs:', error);
            this.showToast('Failed to copy logs', 'error');
        }
    }

    // Prompt Tester Methods
    async loadPromptOperations() {
        try {
            const response = await fetch(`${this.apiBase}/operations`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            this.promptOperations = data || {};
        } catch (error) {
            console.warn('Prompt operations unavailable:', error);
            this.promptOperations = null;
        } finally {
            this.updatePromptOperationOptions();
        }
    }

    updatePromptOperationOptions(preservedValue) {
        const typeSelect = document.getElementById('prompt-type');
        const operationSelect = document.getElementById('prompt-operation');
        if (!operationSelect) return;

        const selectedType = (typeSelect && typeSelect.value) ? typeSelect.value : 'resource';
        const previousValue = preservedValue || operationSelect.value || 'generator';

        let availableOperations = ['generator', 'improver'];
        if (this.promptOperations && typeof this.promptOperations === 'object') {
            const combos = Object.keys(this.promptOperations)
                .filter(key => key.startsWith(`${selectedType}-`))
                .map(key => key.split('-')[1])
                .filter(Boolean);

            if (combos.length > 0) {
                availableOperations = [...new Set(combos)];
            }
        }

        const currentValue = operationSelect.value;
        operationSelect.innerHTML = '';

        availableOperations.forEach(op => {
            const option = document.createElement('option');
            option.value = op;
            option.textContent = op.charAt(0).toUpperCase() + op.slice(1);
            operationSelect.appendChild(option);
        });

        if (availableOperations.includes(previousValue)) {
            operationSelect.value = previousValue;
        } else if (availableOperations.includes(currentValue)) {
            operationSelect.value = currentValue;
        }

        if (!operationSelect.value && availableOperations.length > 0) {
            operationSelect.value = availableOperations[0];
        }

        this.updatePromptOperationSummary();
    }

    updatePromptOperationSummary() {
        const summaryElement = document.getElementById('prompt-operation-summary');
        if (!summaryElement) return;

        const typeSelect = document.getElementById('prompt-type');
        const operationSelect = document.getElementById('prompt-operation');
        const taskType = typeSelect ? typeSelect.value : 'resource';
        const operation = operationSelect ? operationSelect.value : '';

        if (!this.promptOperations) {
            summaryElement.innerHTML = '<div class="prompt-operation-empty">Prompt configuration metadata not available yet.</div>';
            return;
        }

        if (!operation) {
            summaryElement.innerHTML = '<div class="prompt-operation-empty">Select an operation to view configuration details.</div>';
            return;
        }

        const key = `${taskType}-${operation}`;
        const config = this.promptOperations[key];

        if (!config) {
            summaryElement.innerHTML = `<div class="prompt-operation-empty">No configuration found for <code>${this.escapeHtml(key)}</code>.</div>`;
            return;
        }

        const description = config.description
            ? `<p class="prompt-operation-description">${this.escapeHtml(config.description)}</p>`
            : '';

        const additionalSections = Array.isArray(config.additional_sections) && config.additional_sections.length
            ? config.additional_sections.map(section => `<span class="prompt-chip">${this.escapeHtml(section)}</span>`).join(' ')
            : '<span class="prompt-chip">Base sections only</span>';

        let collapsibleHtml = `<div class="prompt-operation-subtitle"><strong>Additional Sections:</strong> ${additionalSections}</div>`;

        if (config.effort_allocation && typeof config.effort_allocation === 'object') {
            const items = Object.entries(config.effort_allocation).map(([section, share]) =>
                `<li>${this.escapeHtml(section)}: ${this.escapeHtml(String(share))}</li>`
            ).join('');
            collapsibleHtml += `<div class="prompt-operation-list"><strong>Effort Allocation:</strong><ul>${items}</ul></div>`;
        }

        if (Array.isArray(config.success_criteria) && config.success_criteria.length) {
            const items = config.success_criteria.map(item => `<li>${this.escapeHtml(item)}</li>`).join('');
            collapsibleHtml += `<div class="prompt-operation-list"><strong>Success Criteria:</strong><ul>${items}</ul></div>`;
        }

        if (Array.isArray(config.principles) && config.principles.length) {
            const items = config.principles.map(item => `<li>${this.escapeHtml(item)}</li>`).join('');
            collapsibleHtml += `<div class="prompt-operation-list"><strong>Guiding Principles:</strong><ul>${items}</ul></div>`;
        }

        summaryElement.innerHTML = `
            <div class="prompt-operation-summary-header">
                <div>
                    <div class="prompt-operation-header"><strong>${this.escapeHtml(config.name || key)}</strong></div>
                    ${description}
                </div>
                <button type="button" class="prompt-operation-toggle" aria-expanded="false">
                    <i class="fas fa-chevron-down"></i>
                    Show Details
                </button>
            </div>
            <div class="prompt-operation-details">${collapsibleHtml}</div>
        `;

        summaryElement.classList.add('collapsed');

        const toggleButton = summaryElement.querySelector('.prompt-operation-toggle');
        const detailsElement = summaryElement.querySelector('.prompt-operation-details');
        if (toggleButton && detailsElement) {
            toggleButton.addEventListener('click', () => {
                const isCollapsed = summaryElement.classList.toggle('collapsed');
                toggleButton.setAttribute('aria-expanded', (!isCollapsed).toString());
                toggleButton.innerHTML = isCollapsed
                    ? '<i class="fas fa-chevron-down"></i> Show Details'
                    : '<i class="fas fa-chevron-up"></i> Hide Details';
            });
        }
    }

    setPromptPreviewStatus(message, type = 'info') {
        const statusElement = document.getElementById('prompt-preview-status');
        if (!statusElement) return;

        statusElement.textContent = message || '';
        statusElement.className = 'prompt-preview-status';

        if (type) {
            statusElement.classList.add(`status-${type}`);
        }
    }

    togglePromptPreviewLoading(isLoading) {
        const previewBtn = document.getElementById('prompt-preview-btn');
        const copyIcon = document.getElementById('prompt-preview-copy-icon');

        this.promptPreviewLoading = !!isLoading;

        if (previewBtn) {
            if (isLoading) {
                if (!previewBtn.dataset.originalContent) {
                    previewBtn.dataset.originalContent = previewBtn.innerHTML;
                }
                previewBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Assembling...';
                previewBtn.disabled = true;
            } else {
                const original = previewBtn.dataset.originalContent;
                if (original) {
                    previewBtn.innerHTML = original;
                }
                previewBtn.disabled = false;
            }
        }

        if (copyIcon) {
            if (isLoading) {
                copyIcon.disabled = true;
            } else if (this.lastPromptPreview && this.lastPromptPreview.prompt) {
                copyIcon.disabled = false;
            }
        }
    }

    async handlePromptPreview() {
        this.setPromptPreviewStatus('Assembling prompt...', 'loading');
        this.togglePromptPreviewLoading(true);

        const typeField = document.getElementById('prompt-type');
        const operationField = document.getElementById('prompt-operation');
        const titleField = document.getElementById('prompt-title');
        const priorityField = document.getElementById('prompt-priority');
        const notesField = document.getElementById('prompt-notes');

        const payload = {
            display: 'preview',
            task: {
                type: typeField ? typeField.value : 'resource',
                operation: operationField ? operationField.value : 'generator',
                title: titleField ? titleField.value.trim() : '',
                priority: priorityField ? priorityField.value : 'medium',
                notes: notesField ? notesField.value.trim() : '',
            }
        };

        try {
            const response = await fetch(`${this.apiBase}/prompt-viewer`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            const data = await this.parsePromptPreviewResponse(response);
            this.lastPromptPreview = data;
            this.renderPromptPreview(data);

            const charCount = data.prompt_size || (data.prompt ? data.prompt.length : 0);
            const operationLabel = data.operation || `${payload.task.type}-${payload.task.operation}`;
            this.setPromptPreviewStatus(`Preview ready for ${operationLabel}. ${charCount.toLocaleString()} characters.`, 'success');
        } catch (error) {
            console.error('Prompt preview failed:', error);
            this.setPromptPreviewStatus(error.message || 'Failed to assemble prompt', 'error');
        } finally {
            this.togglePromptPreviewLoading(false);
        }
    }

    async parsePromptPreviewResponse(response) {
        if (!response.ok) {
            const errorText = await response.text();
            const error = new Error(errorText || `Prompt preview failed with status ${response.status}`);
            error.status = response.status;
            throw error;
        }
        return await response.json();
    }

    renderPromptPreview(data) {
        const placeholder = document.getElementById('prompt-preview-placeholder');
        const detailsContainer = document.getElementById('prompt-preview-details');
        const metadataElement = document.getElementById('prompt-preview-metadata');
        const sectionsElement = document.getElementById('prompt-preview-sections');
        const summaryElement = document.getElementById('prompt-preview-summary');
        const outputWrapper = document.getElementById('prompt-preview-output-wrapper');
        const outputElement = document.getElementById('prompt-preview-output');
        const copyIcon = document.getElementById('prompt-preview-copy-icon');

        if (!detailsContainer || !metadataElement || !sectionsElement || !summaryElement || !outputElement || !outputWrapper) {
            return;
        }

        if (placeholder) {
            placeholder.hidden = true;
        }

        detailsContainer.hidden = false;
        const task = data.task || {};
        const taskTitle = task.title || data.title || 'Prompt Preview Task';
        const taskType = task.type || data.task_type || 'resource';
        const operation = task.operation || data.operation || 'generator';
        const priority = task.priority || '—';
        const notes = task.notes || '';
        const category = task.category || '—';
        const tags = Array.isArray(task.tags) ? task.tags : [];
        const charCount = data.prompt_size || (data.prompt ? data.prompt.length : 0);
        const promptSizeKb = data.prompt_size_kb || (charCount ? (charCount / 1024).toFixed(2) : '0.00');
        const promptSizeMb = data.prompt_size_mb || (charCount ? (charCount / 1024 / 1024).toFixed(3) : '0.000');

        const metadataItems = [
            { label: 'Title', value: taskTitle },
            { label: 'Type / Operation', value: `${taskType} / ${operation}` },
            { label: 'Priority', value: priority },
            { label: 'Category', value: category },
            {
                label: 'Prompt Size',
                value: `${charCount.toLocaleString()} chars (${promptSizeKb} KB / ${promptSizeMb} MB)`
            }
        ];

        if (tags.length > 0) {
            const chips = tags
                .map(tag => `<span class="prompt-chip">${this.escapeHtml(tag)}</span>`)
                .join(' ');
            metadataItems.push({ label: 'Tags', value: chips, allowHtml: true });
        }

        if (notes) {
            const escapedNotes = this.escapeHtml(notes).replace(/\n/g, '<br>');
            metadataItems.push({ label: 'Notes', value: escapedNotes, allowHtml: true });
        }

        summaryElement.innerHTML = `
            <span class="prompt-summary-label">Prompt Size</span>
            <span class="prompt-summary-value">${charCount.toLocaleString()} chars</span>
            <span class="prompt-summary-secondary">${promptSizeKb} KB · ${promptSizeMb} MB</span>
        `;

        metadataElement.innerHTML = metadataItems.map(item => {
            const label = this.escapeHtml(item.label);
            const value = item.allowHtml ? item.value : this.escapeHtml(item.value);
            return `
                <div class="prompt-meta-row">
                    <span class="prompt-meta-label">${label}</span>
                    <span class="prompt-meta-value">${value}</span>
                </div>
            `;
        }).join('');
        metadataElement.hidden = false;

        const sections = Array.isArray(data.sections) ? data.sections : [];
        const detailedSections = Array.isArray(data.sections_detailed) ? data.sections_detailed : [];

        if (detailedSections.length > 0) {
            const colorPalette = [
                '#00897B', '#3949AB', '#6D4C41', '#AF3D4E', '#1E88E5', '#0081CF', '#455A64', '#6A1B9A'
            ];

            const cards = detailedSections.map((section, index) => {
                const color = colorPalette[index % colorPalette.length];
                const title = section.title || section.key || `Section ${index + 1}`;
                const source = section.relative_path || section.key || '—';
                const includes = Array.isArray(section.includes) && section.includes.length > 0
                    ? section.includes
                    : null;
                const safeContent = this.escapeHtml(section.content || '');
                const headerLabel = section.key === 'task-context' ? 'Task Context' : `Section ${index + 1}`;

                const includeMarkup = includes
                    ? includes.map(item => `<span class="prompt-section-include">${this.escapeHtml(item)}</span>`).join(' ')
                    : '<span class="prompt-section-include empty">No nested includes</span>';

                return `
                    <details class="prompt-section-card" data-section-index="${index}">
                        <summary>
                            <span class="prompt-section-badge" style="background-color: ${color};">${index + 1}</span>
                            <span class="prompt-section-summary">
                                <span class="prompt-section-label">${this.escapeHtml(headerLabel)}</span>
                                <span class="prompt-section-title">${this.escapeHtml(title)}</span>
                                <span class="prompt-section-path">${this.escapeHtml(source)}</span>
                            </span>
                            <span class="prompt-section-toggle"><i class="fas fa-chevron-down"></i></span>
                        </summary>
                        <div class="prompt-section-meta">
                            <div class="prompt-section-meta-row">
                                <span class="prompt-section-meta-label">Source file</span>
                                <span class="prompt-section-meta-value">${this.escapeHtml(source)}</span>
                            </div>
                            <div class="prompt-section-meta-row">
                                <span class="prompt-section-meta-label">Includes</span>
                                <span class="prompt-section-meta-value includes">${includeMarkup}</span>
                            </div>
                        </div>
                        <div class="prompt-section-content">
                            <div class="prompt-section-controls">
                                <button type="button" class="prompt-section-copy" data-section-index="${index}">
                                    <i class="fas fa-copy"></i>
                                    <span>Copy Section</span>
                                </button>
                                <span class="prompt-section-length">${(section.content || '').length.toLocaleString()} chars</span>
                            </div>
                            <pre>${safeContent}</pre>
                        </div>
                    </details>
                `;
            }).join('');

            sectionsElement.innerHTML = `
                <div class="prompt-sections-header">
                    <h5>Prompt Sections (${detailedSections.length})</h5>
                    <span class="prompt-sections-subtitle">Expand a section to inspect its source content</span>
                </div>
                <div class="prompt-section-grid">${cards}</div>
            `;

            // Attach copy handlers after rendering
            const copyButtons = sectionsElement.querySelectorAll('.prompt-section-copy');
            copyButtons.forEach(button => {
                button.addEventListener('click', event => {
                    const idx = parseInt(event.currentTarget.dataset.sectionIndex, 10);
                    this.handlePromptSectionCopy(idx);
                });
            });
        } else if (sections.length > 0) {
            const items = sections.map(section => `<li>${this.escapeHtml(section)}</li>`).join('');
            sectionsElement.innerHTML = `<h5>Prompt Sections (${sections.length})</h5><ul>${items}</ul>`;
        } else {
            sectionsElement.innerHTML = '<h5>Prompt Sections</h5><div class="prompt-operation-empty">No sections were resolved.</div>';
        }
        sectionsElement.hidden = false;

        this.setPromptDetailsCollapsed(true);

        if (data.prompt && typeof data.prompt === 'string' && data.prompt.length > 0) {
            outputElement.textContent = data.prompt;
            outputElement.hidden = false;
            outputWrapper.hidden = false;
            if (copyIcon) {
                copyIcon.disabled = false;
            }
        } else {
            outputElement.textContent = '';
            outputElement.hidden = true;
            outputWrapper.hidden = true;
            if (copyIcon) {
                copyIcon.disabled = true;
            }
        }
    }

    async handlePromptCopy() {
        if (!this.lastPromptPreview || !this.lastPromptPreview.prompt) {
            this.showToast('No prompt available to copy', 'warning');
            return;
        }

        try {
            await navigator.clipboard.writeText(this.lastPromptPreview.prompt);
            this.showToast('Prompt copied to clipboard', 'success');
        } catch (error) {
            console.error('Failed to copy prompt:', error);
            this.showToast('Failed to copy prompt', 'error');
        }
    }

    async handlePromptSectionCopy(index) {
        if (!this.lastPromptPreview || !Array.isArray(this.lastPromptPreview.sections_detailed)) {
            this.showToast('Section data unavailable', 'warning');
            return;
        }

        const section = this.lastPromptPreview.sections_detailed[index];
        if (!section || !section.content) {
            this.showToast('Section content unavailable', 'warning');
            return;
        }

        try {
            await navigator.clipboard.writeText(section.content);
            this.showToast(`Copied section ${index + 1}`, 'success');
        } catch (error) {
            console.error('Failed to copy section content:', error);
            this.showToast('Failed to copy section content', 'error');
        }
    }

    setPromptDetailsCollapsed(collapsed = true) {
        const detailsContainer = document.getElementById('prompt-preview-details');
        const toggleButton = document.getElementById('prompt-preview-toggle');
        if (!detailsContainer || !toggleButton) {
            return;
        }

        detailsContainer.classList.toggle('collapsed', collapsed);
        toggleButton.setAttribute('aria-expanded', (!collapsed).toString());
        toggleButton.innerHTML = collapsed
            ? '<i class="fas fa-chevron-down"></i> Show Details'
            : '<i class="fas fa-chevron-up"></i> Hide Details';
    }

    togglePromptDetails() {
        const detailsContainer = document.getElementById('prompt-preview-details');
        if (!detailsContainer) {
            return;
        }

        const shouldCollapse = !detailsContainer.classList.contains('collapsed');
        this.setPromptDetailsCollapsed(shouldCollapse);
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

        this.updateTitleVisibility(operation);
        this.maybeAutofillTaskTitle();
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
            } else {
                const targetSelect = document.getElementById('task-target');
                if (targetSelect) {
                    Array.from(targetSelect.options || []).forEach(option => {
                        option.selected = false;
                    });
                    targetSelect.dispatchEvent(new Event('change', { bubbles: true }));
                }

                if (this.targetSelector) {
                    this.targetSelector.clearOptions();
                    this.targetSelector.setDisabled(true, { message: 'Switch to "Enhance" to choose targets', tone: 'info' });
                    this.targetSelector.setStatus('', 'info');
                }

                this.setTargetHelp('Switch to "Enhance" to choose targets.', 'info');
            }
        }

        this.updateTitleVisibility(operation);
        this.maybeAutofillTaskTitle();
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
        const operation = document.querySelector('input[name="operation"]:checked')?.value || 'improver';
        const targetSelect = document.getElementById('task-target');

        if (!targetSelect) {
            return;
        }

        const multiSelect = this.targetSelector;
        const previousSelection = new Set(this.getSelectedTargets());

        this.setTargetHelp('Loading available targets...', 'info');

        targetSelect.disabled = true;
        targetSelect.innerHTML = '';

        if (multiSelect) {
            multiSelect.setDisabled(true, { message: 'Loading available targets...', tone: 'info', reason: 'loading' });
            multiSelect.clearOptions();
        }

        if (!this.availableResources && !this.availableScenarios) {
            let waitCount = 0;
            while ((!this.availableResources && !this.availableScenarios) && waitCount < 30) {
                await new Promise(resolve => setTimeout(resolve, 100));
                waitCount++;
            }
        }

        const availableTargets = type === 'resource' ? this.availableResources : this.availableScenarios;
        const emptyMessage = `No ${type || 'resource'}s available to improve`;

        if (multiSelect) {
            multiSelect.setEmptyMessage(emptyMessage);
        }

        if (!Array.isArray(availableTargets) || availableTargets.length === 0) {
            this.setTargetHelp(emptyMessage, 'warning');
            targetSelect.disabled = true;
            if (multiSelect) {
                multiSelect.setDisabled(true, { message: emptyMessage, tone: 'warning' });
                multiSelect.setStatus(emptyMessage, 'warning');
                multiSelect.clearOptions();
            }
            return;
        }

        let activeTargetMap;
        try {
            activeTargetMap = await this.fetchActiveTargetMap(type, operation);
        } catch (error) {
            console.error('Failed to load target availability:', error);
            const errorMessage = 'Failed to load target availability. Try again.';
            this.setTargetHelp(errorMessage, 'error');
            if (multiSelect) {
                multiSelect.setDisabled(true, { message: errorMessage, tone: 'error' });
                multiSelect.setStatus(errorMessage, 'error');
                multiSelect.clearOptions();
            }
            return;
        }

        availableTargets
            .slice()
            .sort((a, b) => a.name.localeCompare(b.name))
            .forEach(target => {
                const option = document.createElement('option');
                option.value = target.name;

                const key = this.getTargetKey(type, operation, target.name);
                const existing = activeTargetMap.get(key);

                if (existing) {
                    option.disabled = true;
                    option.textContent = `⛔ ${target.name} — tracked by ${existing.taskId} (${existing.statusLabel})`;
                    option.title = `Task ${existing.taskId} (${existing.statusLabel}) already targets ${target.name}.`;
                    option.classList.add('target-unavailable');
                } else {
                    option.textContent = target.name;
                }

                targetSelect.appendChild(option);
            });

        previousSelection.forEach(value => {
            const option = Array.from(targetSelect.options).find(opt => opt.value === value && !opt.disabled);
            if (option) {
                option.selected = true;
            }
        });

        targetSelect.disabled = false;

        if (multiSelect) {
            multiSelect.setDisabled(false);
            multiSelect.setStatus('', 'info');
            multiSelect.refreshFromSelect();
        }

        this.setTargetHelp('', 'info');
        this.maybeAutofillTaskTitle();
    }

    updateTitleVisibility(operation) {
        const titleGroup = document.getElementById('title-group');
        const titleInput = document.getElementById('task-title');

        if (!titleGroup || !titleInput) {
            return;
        }

        if (operation === 'improver') {
            titleGroup.style.display = 'none';
            titleInput.required = false;
            titleInput.value = '';
            this.resetCreateTaskTitleState();
        } else {
            titleGroup.style.display = '';
            titleInput.required = true;
        }
    }

    resetCreateTaskTitleState() {
        this.titleAutofillActive = false;
        this.lastAutofilledTitle = '';
    }

    handleTargetSelectionChange() {
        this.maybeAutofillTaskTitle();
    }

    handleTitleInputChange() {
        const titleInput = document.getElementById('task-title');
        if (!titleInput) return;

        if (!titleInput.value.trim()) {
            this.resetCreateTaskTitleState();
            return;
        }

        if (titleInput.value !== this.lastAutofilledTitle) {
            this.titleAutofillActive = false;
        }
    }

    maybeAutofillTaskTitle() {
        const titleInput = document.getElementById('task-title');

        if (!titleInput) {
            return;
        }

        const selectedTargets = this.getSelectedTargets();

        if (selectedTargets.length !== 1) {
            if (this.titleAutofillActive) {
                titleInput.value = '';
                this.resetCreateTaskTitleState();
            }
            return;
        }

        const target = (selectedTargets[0] || '').trim();

        if (!target) {
            if (this.titleAutofillActive) {
                titleInput.value = '';
                this.resetCreateTaskTitleState();
            }
            return;
        }

        const operation = document.querySelector('input[name="operation"]:checked')?.value;
        const type = document.querySelector('input[name="type"]:checked')?.value;

        const generatedTitle = this.generateTaskTitle(operation, type, target);
        if (!generatedTitle) {
            return;
        }

        const currentValue = titleInput.value || '';
        const shouldUpdate = this.titleAutofillActive || !currentValue.trim();

        if (shouldUpdate || currentValue === this.lastAutofilledTitle) {
            titleInput.value = generatedTitle;
            this.titleAutofillActive = true;
            this.lastAutofilledTitle = generatedTitle;
        }
    }

    generateTaskTitle(operation, type, target) {
        if (!operation || !type || !target) {
            return '';
        }

        const operationLabel = this.getOperationDisplayName(operation);
        const typeLabel = type.toLowerCase();

        if (!operationLabel) {
            return '';
        }

        return `${operationLabel} ${typeLabel} ${target}`;
    }

    getOperationDisplayName(operation) {
        if (!operation) {
            return '';
        }

        const mapping = {
            generator: 'Create',
            improver: 'Enhance'
        };

        if (mapping[operation]) {
            return mapping[operation];
        }

        return operation.charAt(0).toUpperCase() + operation.slice(1);
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

class TagMultiSelect {
    constructor({
        selectElement,
        wrapperElement,
        selectionElement,
        tagsContainer,
        searchInput,
        dropdownElement,
        optionsContainer,
        statusElement,
        placeholder = ''
    }) {
        this.select = selectElement;
        this.wrapper = wrapperElement;
        this.selection = selectionElement;
        this.tagsContainer = tagsContainer;
        this.searchInput = searchInput;
        this.dropdown = dropdownElement;
        this.optionsContainer = optionsContainer;
        this.statusElement = statusElement;
        this.placeholder = placeholder;

        this.optionElements = new Map();
        this.optionsData = new Map();
        this.filteredOptions = [];
        this.highlightIndex = -1;
        this.disabled = Boolean(this.select?.disabled);
        this.isOpen = false;
        this.emptyMessage = 'No options available';
        this.noResultsMessage = 'No options match that search';
        this.stateMessage = '';
        this.stateTone = 'info';
        this.showingStateMessage = false;
        this.stateIsBusy = false;
        this.dropdownParent = this.dropdown ? this.dropdown.parentElement : null;
        this.portalActive = false;
        this.portalMargin = 8;
        this.boundRepositionDropdown = this.repositionDropdown.bind(this);

        this.handleDocumentClick = this.handleDocumentClick.bind(this);

        this.init();
    }

    init() {
        if (!this.select || !this.wrapper || !this.selection || !this.searchInput || !this.dropdown || !this.optionsContainer) {
            console.warn('TagMultiSelect initialization skipped due to missing elements');
            return;
        }

        this.select.classList.add('tag-multiselect-hidden-select');
        this.select.setAttribute('aria-hidden', 'true');
        this.select.tabIndex = -1;

        this.wrapper.classList.add('tag-multiselect-enhanced');
        this.searchInput.placeholder = this.placeholder;
        this.searchInput.disabled = this.disabled;
        this.selection.setAttribute('aria-expanded', 'false');
        this.selection.setAttribute('aria-disabled', this.disabled ? 'true' : 'false');
        this.selection.setAttribute('aria-busy', 'false');
        this.statusElement.hidden = true;
        this.dropdown.hidden = true;

        this.selection.addEventListener('click', (event) => {
            if (event.target.closest('.tag-multiselect-remove')) {
                return;
            }

            if (this.disabled) {
                return;
            }

            if (!this.isOpen) {
                this.openDropdown();
            }
            this.focusSearchInput();
        });

        this.searchInput.addEventListener('focus', () => {
            if (!this.isOpen && !this.disabled) {
                this.openDropdown();
            }
        });

        this.searchInput.addEventListener('input', (event) => {
            this.filterOptions(event.target.value);
        });

        this.searchInput.addEventListener('keydown', (event) => {
            this.handleKeyDown(event);
        });

        document.addEventListener('click', this.handleDocumentClick);

        this.updateTags();
        this.refreshFromSelect();
    }

    setEmptyMessage(message) {
        this.emptyMessage = message || 'No options available';
    }

    setNoResultsMessage(message) {
        this.noResultsMessage = message || 'No options match that search';
    }

    setStatus(message, tone = 'info') {
        if (!this.statusElement) {
            return;
        }

        if (!message) {
            this.statusElement.textContent = '';
            this.statusElement.dataset.tone = 'info';
            this.statusElement.hidden = true;
            return;
        }

        this.statusElement.textContent = message;
        this.statusElement.dataset.tone = tone;
        this.statusElement.hidden = false;
    }

    setDisabled(disabled, options = {}) {
        this.disabled = Boolean(disabled);
        this.searchInput.disabled = this.disabled;
        this.wrapper.classList.toggle('is-disabled', this.disabled);
        this.selection.setAttribute('aria-disabled', this.disabled ? 'true' : 'false');

        if (this.disabled && options.message) {
            this.stateMessage = options.message;
            this.stateTone = options.tone || 'info';
            this.showingStateMessage = true;
            this.stateIsBusy = options.reason === 'loading';
        } else if (!this.disabled) {
            this.stateMessage = '';
            this.stateTone = 'info';
            this.showingStateMessage = false;
            this.stateIsBusy = false;
        } else if (this.disabled && !options.message) {
            this.showingStateMessage = false;
            this.stateIsBusy = false;
        }

        this.selection.setAttribute('aria-busy', this.stateIsBusy ? 'true' : 'false');
        this.wrapper.classList.toggle('is-busy', this.stateIsBusy);
        this.updateTags();

        if (this.disabled) {
            this.closeDropdown();
        }
    }

    clearOptions() {
        this.optionElements.clear();
        this.optionsData.clear();
        this.filteredOptions = [];
        this.optionsContainer.innerHTML = '';
        this.highlightIndex = -1;
        if (this.searchInput) {
            this.searchInput.value = '';
        }
        this.updateTags();
        this.repositionDropdown();
    }

    refreshFromSelect(preserveHighlightValue = null) {
        if (!this.select) {
            return;
        }

        const currentQuery = this.searchInput.value || '';
        const highlightValue = preserveHighlightValue || this.getHighlightedValue();

        this.optionElements.clear();
        this.optionsData.clear();
        this.filteredOptions = [];
        this.optionsContainer.innerHTML = '';

        const options = Array.from(this.select.options || []).filter(option => option.value);

        if (options.length === 0) {
            this.highlightIndex = -1;
            this.updateTags();
            if (!this.showingStateMessage) {
                this.setStatus(this.emptyMessage, 'empty');
            }
            this.repositionDropdown();
            return;
        }

        this.setStatus('', 'info');

        options.forEach((option, index) => {
            const data = {
                value: option.value,
                label: option.textContent,
                disabled: option.disabled,
                selected: option.selected,
                searchText: `${option.textContent || ''} ${option.value}`.toLowerCase()
            };

            this.optionsData.set(option.value, data);

            const optionElement = document.createElement('div');
            optionElement.className = 'tag-multiselect-option';
            optionElement.dataset.value = option.value;
            optionElement.id = `${this.select.id}-option-${index}`;
            optionElement.setAttribute('role', 'option');
            optionElement.setAttribute('aria-selected', String(option.selected));

            if (data.disabled) {
                optionElement.classList.add('is-disabled');
                optionElement.setAttribute('aria-disabled', 'true');
            } else {
                optionElement.tabIndex = -1;
                optionElement.addEventListener('mousedown', (event) => event.preventDefault());
                optionElement.addEventListener('click', (event) => {
                    event.preventDefault();
                    this.toggleOption(data.value);
                });
            }

            if (data.selected) {
                optionElement.classList.add('is-selected');
            }

            if (option.classList.contains('target-unavailable')) {
                optionElement.classList.add('is-unavailable');
            }

            const check = document.createElement('span');
            check.className = 'tag-multiselect-option-check';
            if (data.selected && !data.disabled) {
                const icon = document.createElement('i');
                icon.className = 'fas fa-check';
                check.appendChild(icon);
            }

            const label = document.createElement('span');
            label.className = 'tag-multiselect-option-label';
            label.textContent = data.label;

            optionElement.appendChild(check);
            optionElement.appendChild(label);

            this.optionsContainer.appendChild(optionElement);
            this.optionElements.set(option.value, optionElement);
        });

        this.filterOptions(currentQuery, { preserveHighlightValue: highlightValue });
        this.updateTags();
    }

    filterOptions(query, options = {}) {
        const normalizedQuery = (query || '').trim().toLowerCase();
        const filtered = [];

        this.optionElements.forEach((element, value) => {
            const optionData = this.optionsData.get(value);
            if (!optionData) {
                element.classList.add('is-hidden');
                return;
            }

            const matches = !normalizedQuery || optionData.searchText.includes(normalizedQuery);
            element.classList.toggle('is-hidden', !matches);

            if (matches) {
                filtered.push({
                    value,
                    element,
                    disabled: optionData.disabled
                });
            }
        });

        this.filteredOptions = filtered;

        if (filtered.length === 0) {
            if (this.optionsData.size === 0) {
                this.setStatus(this.emptyMessage, 'empty');
            } else if (normalizedQuery) {
                this.setStatus(this.noResultsMessage, 'empty');
            } else {
                this.setStatus('', 'info');
            }

            this.highlightIndex = -1;
            this.applyHighlight();
            return;
        }

        this.setStatus('', 'info');

        if (options.preserveHighlightValue) {
            const existingIndex = filtered.findIndex(item => item.value === options.preserveHighlightValue && !item.disabled);
            this.highlightIndex = existingIndex >= 0 ? existingIndex : this.findFirstSelectableIndex(filtered);
        } else if (this.highlightIndex === -1 || this.highlightIndex >= filtered.length || filtered[this.highlightIndex]?.disabled) {
            this.highlightIndex = this.findFirstSelectableIndex(filtered);
        }

        this.applyHighlight();
        this.repositionDropdown();
    }

    findFirstSelectableIndex(list) {
        return list.findIndex(item => !item.disabled);
    }

    updateTags() {
        if (!this.tagsContainer) {
            return;
        }

        this.tagsContainer.innerHTML = '';

        if (this.showingStateMessage) {
            this.tagsContainer.classList.remove('is-empty');
            this.tagsContainer.classList.add('is-placeholder');

            const placeholder = document.createElement('span');
            const toneSuffix = this.stateTone ? ` tag-multiselect-placeholder--${this.stateTone}` : '';
            placeholder.className = `tag-multiselect-placeholder${toneSuffix}`;
            placeholder.textContent = this.stateMessage || this.placeholder;
            this.tagsContainer.appendChild(placeholder);
            this.searchInput.placeholder = '';
            if (this.searchInput) {
                this.searchInput.style.display = 'none';
            }
            return;
        }

        if (this.searchInput) {
            this.searchInput.style.display = '';
        }
        this.tagsContainer.classList.remove('is-placeholder');

        const selected = Array.from(this.select.selectedOptions || [])
            .filter(option => option.value && !option.disabled);

        if (!selected.length) {
            this.tagsContainer.classList.add('is-empty');
            this.searchInput.placeholder = this.placeholder;
            return;
        }

        this.tagsContainer.classList.remove('is-empty');
        this.searchInput.placeholder = '';

        selected.forEach(option => {
            const tag = document.createElement('span');
            tag.className = 'tag-multiselect-tag';
            tag.dataset.value = option.value;
            tag.title = option.textContent;

            const label = document.createElement('span');
            label.className = 'tag-multiselect-tag-label';
            label.textContent = option.textContent;

            const removeButton = document.createElement('button');
            removeButton.type = 'button';
            removeButton.className = 'tag-multiselect-remove';
            removeButton.setAttribute('aria-label', `Remove ${option.textContent}`);

            const icon = document.createElement('i');
            icon.className = 'fas fa-times';
            removeButton.appendChild(icon);

            removeButton.addEventListener('click', (event) => {
                event.preventDefault();
                this.removeOption(option.value);
            });

            tag.appendChild(label);
            tag.appendChild(removeButton);
            this.tagsContainer.appendChild(tag);
        });
    }

    getSelectedValues() {
        return Array.from(this.select.selectedOptions || [])
            .filter(option => option.value)
            .map(option => option.value);
    }

    toggleOption(value) {
        const option = Array.from(this.select.options || []).find(opt => opt.value === value);
        if (!option || option.disabled) {
            return;
        }

        const highlightValue = this.getHighlightedValue();
        option.selected = !option.selected;
        this.refreshFromSelect(highlightValue);
        this.dispatchChange();
    }

    removeOption(value) {
        const option = Array.from(this.select.options || []).find(opt => opt.value === value);
        if (!option || option.disabled) {
            return;
        }

        const highlightValue = this.getHighlightedValue();
        option.selected = false;
        this.refreshFromSelect(highlightValue);
        this.dispatchChange();
        this.focusSearchInput();
    }

    handleKeyDown(event) {
        switch (event.key) {
            case 'ArrowDown':
                event.preventDefault();
                if (!this.isOpen) {
                    this.openDropdown();
                }
                this.moveHighlight(1);
                break;
            case 'ArrowUp':
                event.preventDefault();
                if (!this.isOpen) {
                    this.openDropdown();
                }
                this.moveHighlight(-1);
                break;
            case 'Enter':
                if (this.isOpen && this.highlightIndex !== -1) {
                    const highlighted = this.filteredOptions[this.highlightIndex];
                    if (highlighted && !highlighted.disabled) {
                        event.preventDefault();
                        this.toggleOption(highlighted.value);
                    }
                }
                break;
            case 'Escape':
                if (this.isOpen) {
                    event.preventDefault();
                    this.closeDropdown(true);
                }
                break;
            case 'Backspace':
                if (!this.searchInput.value) {
                    const selectedValues = this.getSelectedValues();
                    const lastValue = selectedValues[selectedValues.length - 1];
                    if (lastValue) {
                        event.preventDefault();
                        this.removeOption(lastValue);
                    }
                }
                break;
            case 'Tab':
                this.closeDropdown();
                break;
            default:
                break;
        }
    }

    moveHighlight(step) {
        if (!this.filteredOptions.length) {
            this.highlightIndex = -1;
            this.applyHighlight();
            return;
        }

        const length = this.filteredOptions.length;
        let newIndex = this.highlightIndex;
        let attempts = 0;

        do {
            if (newIndex === -1) {
                newIndex = step > 0 ? 0 : length - 1;
            } else {
                newIndex = (newIndex + step + length) % length;
            }
            attempts++;
        } while (this.filteredOptions[newIndex]?.disabled && attempts <= length);

        if (attempts > length) {
            this.highlightIndex = -1;
        } else {
            this.highlightIndex = newIndex;
        }

        this.applyHighlight();
    }

    getHighlightedValue() {
        if (this.highlightIndex === -1) {
            return null;
        }

        return this.filteredOptions[this.highlightIndex]?.value || null;
    }

    setHighlightByValue(value) {
        if (!value) {
            this.highlightIndex = -1;
            this.applyHighlight();
            return;
        }

        const index = this.filteredOptions.findIndex(item => item.value === value && !item.disabled);
        this.highlightIndex = index;
        this.applyHighlight();
    }

    applyHighlight() {
        this.optionElements.forEach((element) => {
            element.classList.remove('is-highlighted');
        });

        const highlighted = this.filteredOptions[this.highlightIndex];
        if (!highlighted) {
            this.searchInput.removeAttribute('aria-activedescendant');
            return;
        }

        highlighted.element.classList.add('is-highlighted');
        this.searchInput.setAttribute('aria-activedescendant', highlighted.element.id);
        this.ensureOptionInView(highlighted.element);
    }

    ensureOptionInView(element) {
        if (!element || !this.dropdown || this.dropdown.hidden) {
            return;
        }

        const container = this.dropdown;
        const elementTop = element.offsetTop;
        const elementBottom = elementTop + element.offsetHeight;
        const viewTop = container.scrollTop;
        const viewBottom = viewTop + container.clientHeight;

        if (elementTop < viewTop) {
            container.scrollTop = elementTop;
        } else if (elementBottom > viewBottom) {
            container.scrollTop = elementBottom - container.clientHeight;
        }
    }

    ensureDropdownPortal() {
        if (this.portalActive || !this.dropdown) {
            return;
        }

        if (!this.dropdownParent) {
            this.dropdownParent = this.dropdown.parentElement;
        }

        document.body.appendChild(this.dropdown);
        this.dropdown.classList.add('tag-multiselect-dropdown-portal');
        this.portalActive = true;
        window.addEventListener('resize', this.boundRepositionDropdown);
        window.addEventListener('scroll', this.boundRepositionDropdown, true);
    }

    teardownDropdownPortal() {
        if (!this.portalActive || !this.dropdownParent) {
            return;
        }

        window.removeEventListener('resize', this.boundRepositionDropdown);
        window.removeEventListener('scroll', this.boundRepositionDropdown, true);

        this.dropdownParent.appendChild(this.dropdown);
        this.dropdown.classList.remove('tag-multiselect-dropdown-portal', 'is-flipped');
        this.dropdown.style.top = '';
        this.dropdown.style.left = '';
        this.dropdown.style.width = '';
        this.dropdown.style.maxHeight = '';
        this.portalActive = false;
        if (this.wrapper) {
            this.wrapper.classList.remove('dropdown-open-top');
        }
    }

    repositionDropdown() {
        if (!this.portalActive || !this.dropdown || !this.selection) {
            return;
        }

        const rect = this.selection.getBoundingClientRect();
        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;
        const margin = this.portalMargin;

        const maxWidth = Math.max(200, viewportWidth - margin * 2);
        const targetWidth = Math.min(rect.width, maxWidth);
        let left = rect.left;

        if (left + targetWidth > viewportWidth - margin) {
            left = viewportWidth - targetWidth - margin;
        }
        left = Math.max(margin, left);

        this.dropdown.style.width = `${Math.round(targetWidth)}px`;
        this.dropdown.style.maxHeight = `${Math.max(180, Math.round(viewportHeight - margin * 2))}px`;

        // Force layout to get accurate height after width adjustment
        const dropdownRect = this.dropdown.getBoundingClientRect();
        let dropdownHeight = dropdownRect.height || 0;

        if (!dropdownHeight) {
            dropdownHeight = Math.min(320, viewportHeight - margin * 2);
        }

        let top = rect.bottom + margin;
        let placement = 'bottom';

        if (top + dropdownHeight > viewportHeight - margin) {
            const spaceBelow = viewportHeight - rect.bottom - margin;
            const spaceAbove = rect.top - margin;
            if (spaceAbove > spaceBelow) {
                top = Math.max(margin, rect.top - margin - dropdownHeight);
                placement = 'top';
            } else {
                top = Math.max(margin, Math.min(viewportHeight - margin - dropdownHeight, rect.bottom + margin));
            }
        }

        this.dropdown.style.top = `${Math.round(top)}px`;
        this.dropdown.style.left = `${Math.round(left)}px`;

        if (placement === 'top') {
            this.dropdown.classList.add('is-flipped');
            this.wrapper.classList.add('dropdown-open-top');
        } else {
            this.dropdown.classList.remove('is-flipped');
            this.wrapper.classList.remove('dropdown-open-top');
        }
    }

    openDropdown() {
        if (this.disabled || this.isOpen) {
            return;
        }

        this.isOpen = true;
        this.ensureDropdownPortal();
        this.dropdown.hidden = false;
        this.wrapper.classList.add('is-open');
        this.selection.setAttribute('aria-expanded', 'true');
        this.filterOptions(this.searchInput.value || '');
        this.dropdown.scrollTop = 0;
        this.repositionDropdown();
    }

    closeDropdown(focusInput = false) {
        if (!this.isOpen) {
            return;
        }

        this.isOpen = false;
        this.dropdown.hidden = true;
        this.wrapper.classList.remove('is-open');
        this.selection.setAttribute('aria-expanded', 'false');
        this.wrapper.classList.remove('dropdown-open-top');
        this.teardownDropdownPortal();

        if (focusInput) {
            this.focusSearchInput();
        }
    }

    handleDocumentClick(event) {
        if (!this.wrapper.contains(event.target) && !(this.dropdown && this.dropdown.contains(event.target))) {
            this.closeDropdown();
        }
    }

    focusSearchInput() {
        if (this.searchInput.disabled) {
            return;
        }

        this.searchInput.focus({ preventScroll: true });
    }

    dispatchChange() {
        // Keep the native select in sync with observers
        if (this.select) {
            this.select.dispatchEvent(new Event('change', { bubbles: true }));
        }
    }
}

// Initialize the application when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.ecosystemManager = new EcosystemManager();
    window.ecosystemManager.init();
});
