// Ecosystem Manager UI JavaScript

class EcosystemManager {
    constructor() {
        this.apiBase = '/api';
        this.tasks = {};
        this.filteredTasks = {};
        this.operations = {};
        this.resources = [];
        this.scenarios = [];
        this.ws = null;
        this.wsReconnectInterval = null;
        
        // Configuration - will be loaded dynamically
        this.categoryOptions = {
            resource: [],
            scenario: []
        };
        
        // Initialize
        this.init();
    }
    
    async init() {
        console.log('🚀 Initializing Ecosystem Manager UI...');
        
        try {
            // Initialize dark mode before loading content
            this.initDarkMode();
            
            // Set up dynamic layout adjustment
            this.adjustMainLayout();
            window.addEventListener('resize', () => this.adjustMainLayout());
            
            // Load critical data first (in parallel for speed)
            await Promise.all([
                this.loadOperations(),
                this.loadCategories(),
                this.loadAllTasks()  // Load tasks immediately
            ]);
            
            // Load discovered resources and scenarios in background
            // This doesn't block initial UI rendering
            this.loadDiscoveryData().catch(error => {
                console.warn('Failed to load discovery data (non-critical):', error);
            });
            
            // Connect to WebSocket for real-time updates
            this.connectWebSocket();
            
            // Set up auto-refresh (as fallback)
            this.setupAutoRefresh();
            
            // Set up event listeners
            this.setupEventListeners();
            
            console.log('✅ Ecosystem Manager UI initialized successfully');
        } catch (error) {
            console.error('❌ Failed to initialize UI:', error);
            this.showToast('Failed to initialize application', 'error');
        }
    }
    
    adjustMainLayout() {
        const fixedContainer = document.getElementById('fixed-top-container');
        const mainElement = document.querySelector('main');
        
        if (fixedContainer && mainElement) {
            const containerHeight = fixedContainer.offsetHeight;
            mainElement.style.marginTop = `${containerHeight + 20}px`;
        }
    }
    
    async loadOperations() {
        try {
            const response = await fetch(`${this.apiBase}/operations`);
            if (!response.ok) throw new Error('Failed to load operations');
            
            this.operations = await response.json();
            console.log('📝 Loaded operations:', Object.keys(this.operations));
        } catch (error) {
            console.error('Failed to load operations:', error);
            throw error;
        }
    }
    
    async loadCategories() {
        try {
            const response = await fetch(`${this.apiBase}/categories`);
            if (!response.ok) throw new Error('Failed to load categories');
            
            const categories = await response.json();
            
            // Convert to UI format
            if (categories.resource_categories) {
                this.categoryOptions.resource = Object.entries(categories.resource_categories).map(([key, label]) => ({
                    value: key,
                    label: label
                }));
            }
            
            if (categories.scenario_categories) {
                this.categoryOptions.scenario = Object.entries(categories.scenario_categories).map(([key, label]) => ({
                    value: key,
                    label: label
                }));
            }
            
            console.log('📂 Loaded categories:', this.categoryOptions);
            this.updateCategoryFilters();
        } catch (error) {
            console.error('Failed to load categories:', error);
            // Fall back to default categories
            this.categoryOptions = {
                resource: [
                    { value: 'ai-ml', label: 'AI/ML' },
                    { value: 'storage', label: 'Storage' },
                    { value: 'automation', label: 'Automation' },
                    { value: 'monitoring', label: 'Monitoring' },
                    { value: 'communication', label: 'Communication' },
                    { value: 'security', label: 'Security' }
                ],
                scenario: [
                    { value: 'productivity', label: 'Productivity' },
                    { value: 'ai-tools', label: 'AI Tools' },
                    { value: 'business', label: 'Business' },
                    { value: 'personal', label: 'Personal' },
                    { value: 'automation', label: 'Automation' },
                    { value: 'entertainment', label: 'Entertainment' }
                ]
            };
        }
    }
    
    async loadDiscoveryData() {
        try {
            // Load resources and scenarios in parallel
            const [resourcesResponse, scenariosResponse] = await Promise.all([
                fetch(`${this.apiBase}/resources`),
                fetch(`${this.apiBase}/scenarios`)
            ]);
            
            if (resourcesResponse.ok) {
                this.resources = await resourcesResponse.json();
                console.log(`🔧 Discovered ${this.resources.length} resources:`, this.resources.map(r => r.name));
            } else {
                console.warn('Failed to load resources:', resourcesResponse.statusText);
                this.resources = [];
            }
            
            if (scenariosResponse.ok) {
                this.scenarios = await scenariosResponse.json();
                console.log(`📋 Discovered ${this.scenarios.length} scenarios:`, this.scenarios.map(s => s.name));
            } else {
                console.warn('Failed to load scenarios:', scenariosResponse.statusText);
                this.scenarios = [];
            }
        } catch (error) {
            console.error('Failed to load discovery data:', error);
            // Don't throw - this isn't critical for basic functionality
            this.resources = [];
            this.scenarios = [];
        }
    }
    
    async loadAllTasks() {
        const statuses = ['pending', 'in-progress', 'review', 'completed', 'failed'];
        
        for (const status of statuses) {
            await this.loadTasks(status);
        }
        
        this.updateStats();
        this.applyFilters();
    }
    
    async loadTasks(status) {
        try {
            const response = await fetch(`${this.apiBase}/tasks?status=${status}`);
            if (!response.ok) throw new Error(`Failed to load ${status} tasks`);
            
            const tasks = await response.json();
            this.tasks[status] = tasks;
            
            console.log(`📋 Loaded ${tasks.length} ${status} tasks`);
        } catch (error) {
            console.error(`Failed to load ${status} tasks:`, error);
            this.tasks[status] = [];
        }
    }
    
    connectWebSocket() {
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${wsProtocol}//${window.location.host}/ws`;
        
        console.log('🔌 Connecting to WebSocket:', wsUrl);
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('✅ WebSocket connected');
            this.showToast('Connected to real-time updates', 'success');
            
            // Clear any reconnection interval
            if (this.wsReconnectInterval) {
                clearInterval(this.wsReconnectInterval);
                this.wsReconnectInterval = null;
            }
        };
        
        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleWebSocketMessage(message);
            } catch (error) {
                console.error('Failed to parse WebSocket message:', error);
            }
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.showToast('Real-time connection error', 'error');
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.showToast('Real-time updates disconnected', 'warning');
            
            // Attempt to reconnect every 5 seconds
            if (!this.wsReconnectInterval) {
                this.wsReconnectInterval = setInterval(() => {
                    console.log('Attempting to reconnect WebSocket...');
                    this.connectWebSocket();
                }, 5000);
            }
        };
    }
    
    handleWebSocketMessage(message) {
        console.log('📨 WebSocket message:', message);
        
        switch (message.type) {
            case 'connected':
                console.log('Connected to server:', message.message);
                break;
                
            case 'task_progress':
                this.updateTaskProgress(message.data);
                break;
                
            case 'task_completed':
                this.handleTaskCompleted(message.data);
                break;
                
            case 'task_failed':
                this.handleTaskFailed(message.data);
                break;
                
            case 'task_created':
                this.handleTaskCreated(message.data);
                break;
                
            default:
                console.log('Unknown message type:', message.type);
        }
    }
    
    updateTaskProgress(task) {
        // Update task in memory
        const oldStatus = this.findTaskStatus(task.id);
        if (oldStatus && this.tasks[oldStatus]) {
            const index = this.tasks[oldStatus].findIndex(t => t.id === task.id);
            if (index !== -1) {
                this.tasks[oldStatus][index] = task;
            }
        }
        
        // Update UI
        this.updateTaskCard(task);
        this.showToast(`Task ${task.id} progress: ${task.progress_percentage}%`, 'info');
    }
    
    handleTaskCompleted(task) {
        // Move task from in-progress to completed
        this.moveTaskBetweenColumns(task, 'in-progress', 'completed');
        this.showToast(`Task ${task.title} completed successfully!`, 'success');
    }
    
    handleTaskFailed(task) {
        // Move task from in-progress to failed
        this.moveTaskBetweenColumns(task, 'in-progress', 'failed');
        this.showToast(`Task ${task.title} failed: ${task.results?.error}`, 'error');
    }
    
    handleTaskCreated(task) {
        // Add new task to pending
        if (!this.tasks.pending) {
            this.tasks.pending = [];
        }
        this.tasks.pending.push(task);
        this.renderTasks();
        this.showToast(`New task created: ${task.title}`, 'info');
    }
    
    findTaskStatus(taskId) {
        for (const status in this.tasks) {
            if (this.tasks[status].find(t => t.id === taskId)) {
                return status;
            }
        }
        return null;
    }
    
    moveTaskBetweenColumns(task, fromStatus, toStatus) {
        // Remove from old status
        if (this.tasks[fromStatus]) {
            const index = this.tasks[fromStatus].findIndex(t => t.id === task.id);
            if (index !== -1) {
                this.tasks[fromStatus].splice(index, 1);
            }
        }
        
        // Add to new status
        if (!this.tasks[toStatus]) {
            this.tasks[toStatus] = [];
        }
        this.tasks[toStatus].push(task);
        
        // Re-render
        this.renderTasks();
    }
    
    updateTaskCard(task) {
        const card = document.querySelector(`[data-task-id="${task.id}"]`);
        if (card) {
            // Update progress bar
            const progressBar = card.querySelector('.progress-fill');
            if (progressBar) {
                progressBar.style.width = `${task.progress_percentage || 0}%`;
                progressBar.classList.add('progress-animating');
                setTimeout(() => progressBar.classList.remove('progress-animating'), 500);
            }
            
            // Update phase
            const phaseElement = card.querySelector('.task-phase');
            if (phaseElement && task.current_phase) {
                phaseElement.textContent = `Phase: ${task.current_phase}`;
            }
            
            // Update priority badge
            const priorityBadge = card.querySelector('.priority-badge');
            if (priorityBadge) {
                priorityBadge.className = `priority-badge priority-${task.priority}`;
                priorityBadge.textContent = task.priority;
            }
        }
    }
    
    setupAutoRefresh() {
        // Refresh every 30 seconds (as fallback if WebSocket fails)
        setInterval(() => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
                console.log('WebSocket not connected, refreshing tasks...');
                this.loadAllTasks();
            }
        }, 30000);
    }
    
    setupEventListeners() {
        // Form submission
        document.getElementById('create-task-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleCreateTask();
        });
        
        // Type change in create form
        document.getElementById('task-type').addEventListener('change', () => {
            this.updateFormForType();
        });
        
        // Operation change in create form  
        document.getElementById('task-operation').addEventListener('change', () => {
            this.updateFormForOperation();
        });
        
        // Close modals on background click
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.closeAllModals();
            }
        });
        
        // Escape key closes modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeAllModals();
            }
        });
    }
    
    updateStats() {
        // Stats removed from header - column counts provide this information
    }
    
    applyFilters() {
        const typeFilter = document.getElementById('filter-type').value;
        const operationFilter = document.getElementById('filter-operation').value;
        const categoryFilter = document.getElementById('filter-category').value;
        const priorityFilter = document.getElementById('filter-priority').value;
        const searchTerm = document.getElementById('search-input').value.toLowerCase();
        
        // Apply filters to each status
        Object.keys(this.tasks).forEach(status => {
            let filtered = this.tasks[status] || [];
            
            if (typeFilter) {
                filtered = filtered.filter(task => task.type === typeFilter);
            }
            
            if (operationFilter) {
                filtered = filtered.filter(task => task.operation === operationFilter);
            }
            
            if (categoryFilter) {
                filtered = filtered.filter(task => task.category === categoryFilter);
            }
            
            if (priorityFilter) {
                filtered = filtered.filter(task => task.priority === priorityFilter);
            }
            
            if (searchTerm) {
                filtered = filtered.filter(task => 
                    task.title.toLowerCase().includes(searchTerm) ||
                    task.id.toLowerCase().includes(searchTerm) ||
                    (task.category && task.category.toLowerCase().includes(searchTerm))
                );
            }
            
            this.filteredTasks[status] = filtered;
        });
        
        this.renderAllColumns();
    }
    
    clearFilters() {
        document.getElementById('filter-type').value = '';
        document.getElementById('filter-operation').value = '';
        document.getElementById('filter-category').value = '';
        document.getElementById('filter-priority').value = '';
        document.getElementById('search-input').value = '';
        
        this.applyFilters();
    }
    
    renderTasks() {
        // Wrapper function to render all tasks
        this.renderAllColumns();
    }
    
    renderAllColumns() {
        const statuses = ['pending', 'in-progress', 'review', 'completed', 'failed'];
        
        statuses.forEach(status => {
            this.renderColumn(status);
        });
        
        // Adjust layout after rendering
        setTimeout(() => this.adjustMainLayout(), 100);
    }
    
    renderColumn(status) {
        const tasks = this.filteredTasks[status] || [];
        const container = document.getElementById(`${status}-tasks`);
        const countElement = document.getElementById(`${status}-count`);
        
        // Update count
        if (countElement) {
            countElement.textContent = tasks.length;
        }
        
        // Check if container exists
        if (!container) {
            console.warn(`Container for ${status} tasks not found`);
            return;
        }
        
        // Clear container
        container.innerHTML = '';
        
        if (tasks.length === 0) {
            container.innerHTML = `
                <div class="text-center text-muted">
                    <i class="fas fa-inbox" style="font-size: 2rem; margin-bottom: 1rem; opacity: 0.3;"></i>
                    <div>No tasks in ${status.replace('-', ' ')}</div>
                </div>
            `;
            return;
        }
        
        // Render tasks
        tasks.forEach(task => {
            const taskCard = this.createTaskCard(task);
            container.appendChild(taskCard);
        });
    }
    
    createTaskCard(task) {
        const card = document.createElement('div');
        card.className = 'task-card';
        card.setAttribute('data-task-id', task.id);
        card.onclick = () => this.showTaskDetails(task.id);
        
        const typeIcon = task.type === 'resource' ? 'fas fa-cog' : 'fas fa-bullseye';
        const operationClass = task.operation;
        
        card.innerHTML = `
            <div class="task-header">
                <div class="task-icon ${task.type} ${operationClass}">
                    <i class="${typeIcon}"></i>
                </div>
                <div class="task-title">${this.escapeHtml(task.title)}</div>
            </div>
            
            <div class="task-meta">
                <span><i class="fas fa-layer-group"></i> ${task.type}</span>
                <span><i class="fas fa-tools"></i> ${task.operation}</span>
                ${task.effort_estimate ? `<span><i class="fas fa-clock"></i> ${task.effort_estimate}</span>` : ''}
            </div>
            
            <div class="task-tags">
                <span class="task-tag priority-badge priority-${task.priority}">${task.priority}</span>
                ${task.category ? `<span class="task-tag category-${task.category}">${task.category}</span>` : ''}
                ${task.tags ? task.tags.map(tag => `<span class="task-tag">${tag}</span>`).join('') : ''}
            </div>
            
            ${task.current_phase ? `
                <div class="task-phase">Phase: ${task.current_phase}</div>
            ` : ''}
            
            ${task.progress_percentage !== undefined ? `
                <div class="task-progress">
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${task.progress_percentage}%"></div>
                    </div>
                    <div class="progress-text">
                        <span>Progress</span>
                        <span>${task.progress_percentage}%</span>
                    </div>
                </div>
            ` : ''}
            
            ${task.current_phase ? `
                <div class="progress-text mb-2">
                    <span><i class="fas fa-play"></i> ${task.current_phase}</span>
                </div>
            ` : ''}
            
            <div class="task-footer">
                <div class="task-id">${task.id}</div>
                <div class="task-actions">
                    <button class="btn-icon" onclick="event.stopPropagation(); ecosystemManager.updateTaskStatus('${task.id}')" title="Update Status">
                        <i class="fas fa-edit"></i>
                    </button>
                </div>
            </div>
        `;
        
        return card;
    }
    
    updateCategoryFilters() {
        // Update filter dropdowns if they exist
        const filterCategorySelect = document.getElementById('filter-category');
        if (filterCategorySelect) {
            const currentValue = filterCategorySelect.value;
            filterCategorySelect.innerHTML = '<option value="">All Categories</option>';
            
            // Add all categories from both types
            const allCategories = new Map();
            Object.entries(this.categoryOptions).forEach(([type, cats]) => {
                cats.forEach(cat => {
                    if (!allCategories.has(cat.value)) {
                        allCategories.set(cat.value, cat.label);
                    }
                });
            });
            
            Array.from(allCategories.entries()).forEach(([value, label]) => {
                const option = document.createElement('option');
                option.value = value;
                option.textContent = label;
                filterCategorySelect.appendChild(option);
            });
            
            // Restore previous selection if it still exists
            if (currentValue && filterCategorySelect.querySelector(`option[value="${currentValue}"]`)) {
                filterCategorySelect.value = currentValue;
            }
        }
    }
    
    updateFormForType() {
        const type = document.getElementById('task-type').value;
        const categorySelect = document.getElementById('task-category');
        
        // Clear existing options
        categorySelect.innerHTML = '<option value="">Select category...</option>';
        
        if (type && this.categoryOptions[type]) {
            this.categoryOptions[type].forEach(option => {
                const optionElement = document.createElement('option');
                optionElement.value = option.value;
                optionElement.textContent = option.label;
                categorySelect.appendChild(optionElement);
            });
        }
        
        // Update target options if improver is selected
        const operation = document.getElementById('task-operation').value;
        if (operation === 'improver' && type) {
            this.updateTargetOptions(type);
        }
    }
    
    updateFormForOperation() {
        const operation = document.getElementById('task-operation').value;
        const type = document.getElementById('task-type').value;
        const targetGroup = document.getElementById('target-group');
        
        if (operation === 'improver') {
            targetGroup.style.display = 'block';
            document.getElementById('task-target').required = true;
            
            // Populate target options with discovered resources/scenarios
            this.updateTargetOptions(type);
        } else {
            targetGroup.style.display = 'none';
            document.getElementById('task-target').required = false;
        }
    }
    
    updateTargetOptions(type) {
        const targetSelect = document.getElementById('task-target');
        
        // Convert input to select if it's not already
        if (targetSelect.tagName !== 'SELECT') {
            const newSelect = document.createElement('select');
            newSelect.id = 'task-target';
            newSelect.name = 'target';
            newSelect.required = true;
            newSelect.className = targetSelect.className;
            targetSelect.parentNode.replaceChild(newSelect, targetSelect);
        }
        
        const select = document.getElementById('task-target');
        select.innerHTML = '<option value="">Select target to improve...</option>';
        
        let targets = [];
        if (type === 'resource') {
            targets = this.resources;
        } else if (type === 'scenario') {
            targets = this.scenarios;
        }
        
        targets.forEach(target => {
            const option = document.createElement('option');
            option.value = target.name;
            option.textContent = `${target.name} (${target.category || 'uncategorized'})`;
            if (target.prd_completion_percentage !== undefined) {
                option.textContent += ` - ${target.prd_completion_percentage}% complete`;
            }
            if (!target.healthy) {
                option.textContent += ' ⚠️';
                option.style.color = '#f39c12';
            }
            select.appendChild(option);
        });
        
        if (targets.length === 0) {
            const option = document.createElement('option');
            option.value = '';
            option.textContent = `No ${type}s discovered`;
            option.disabled = true;
            select.appendChild(option);
        }
    }
    
    async handleCreateTask() {
        const form = document.getElementById('create-task-form');
        const formData = new FormData(form);
        
        const taskData = {
            title: formData.get('title'),
            type: formData.get('type'),
            operation: formData.get('operation'),
            priority: formData.get('priority') || 'medium',
            effort_estimate: formData.get('effort_estimate'),
            notes: formData.get('notes') || ''
        };
        
        // Add target for improver operations and infer category from it
        if (taskData.operation === 'improver' && formData.get('target')) {
            const targetName = formData.get('target');
            taskData.requirements = {
                [`target_${taskData.type}`]: targetName
            };
            
            // Try to get category from the target resource/scenario
            if (taskData.type === 'resource') {
                const targetResource = this.resources.find(r => r.name === targetName);
                if (targetResource && targetResource.category) {
                    taskData.category = targetResource.category;
                }
            } else if (taskData.type === 'scenario') {
                const targetScenario = this.scenarios.find(s => s.name === targetName);
                if (targetScenario && targetScenario.category) {
                    taskData.category = targetScenario.category;
                }
            }
        }
        
        // Set a default category if none was inferred
        if (!taskData.category) {
            taskData.category = taskData.type === 'resource' ? 'general' : 'general';
        }
        
        this.showLoading(true);
        
        try {
            const response = await fetch(`${this.apiBase}/tasks`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(taskData)
            });
            
            if (!response.ok) {
                const error = await response.text();
                throw new Error(error);
            }
            
            const createdTask = await response.json();
            
            this.showToast(`Task created successfully: ${createdTask.id}`, 'success');
            this.closeCreateTaskModal();
            form.reset();
            
            // Refresh tasks
            await this.loadAllTasks();
            
        } catch (error) {
            console.error('Failed to create task:', error);
            this.showToast(`Failed to create task: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }
    
    async showTaskDetails(taskId) {
        this.showLoading(true);
        
        try {
            const response = await fetch(`${this.apiBase}/tasks/${taskId}`);
            if (!response.ok) throw new Error('Failed to load task details');
            
            const task = await response.json();
            
            const modal = document.getElementById('task-details-modal');
            const titleElement = document.getElementById('task-details-title');
            const contentElement = document.getElementById('task-details-content');
            
            titleElement.textContent = `Edit Task`;
            
            // Get category options based on task type
            const categoryOptions = this.categoryOptions[task.type] || [];
            
            contentElement.innerHTML = `
                <form id="edit-task-form" onsubmit="return false;">
                    <div style="padding: 1.5rem;">
                        <!-- Basic Information -->
                        <div class="form-group">
                            <label for="edit-task-title">Title *</label>
                            <input type="text" id="edit-task-title" name="title" value="${this.escapeHtml(task.title)}" required>
                        </div>
                        
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-type">Type</label>
                                <select id="edit-task-type" name="type" disabled>
                                    <option value="resource" ${task.type === 'resource' ? 'selected' : ''}>Resource</option>
                                    <option value="scenario" ${task.type === 'scenario' ? 'selected' : ''}>Scenario</option>
                                </select>
                                <small class="form-help">Type cannot be changed after creation</small>
                            </div>
                            
                            <div class="form-group">
                                <label for="edit-task-operation">Operation</label>
                                <select id="edit-task-operation" name="operation" disabled>
                                    <option value="generator" ${task.operation === 'generator' ? 'selected' : ''}>Generator</option>
                                    <option value="improver" ${task.operation === 'improver' ? 'selected' : ''}>Improver</option>
                                </select>
                                <small class="form-help">Operation cannot be changed after creation</small>
                            </div>
                        </div>
                        
                        <!-- Status and Priority -->
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-status">Status *</label>
                                <select id="edit-task-status" name="status" required>
                                    <option value="pending" ${task.status === 'pending' ? 'selected' : ''}>Pending</option>
                                    <option value="in-progress" ${task.status === 'in-progress' ? 'selected' : ''}>In Progress</option>
                                    <option value="review" ${task.status === 'review' ? 'selected' : ''}>Review</option>
                                    <option value="completed" ${task.status === 'completed' ? 'selected' : ''}>Completed</option>
                                    <option value="failed" ${task.status === 'failed' ? 'selected' : ''}>Failed</option>
                                </select>
                            </div>
                            
                            <div class="form-group">
                                <label for="edit-task-priority">Priority *</label>
                                <select id="edit-task-priority" name="priority" required>
                                    <option value="low" ${task.priority === 'low' ? 'selected' : ''}>Low</option>
                                    <option value="medium" ${task.priority === 'medium' ? 'selected' : ''}>Medium</option>
                                    <option value="high" ${task.priority === 'high' ? 'selected' : ''}>High</option>
                                    <option value="critical" ${task.priority === 'critical' ? 'selected' : ''}>Critical</option>
                                </select>
                            </div>
                        </div>
                        
                        <!-- Category and Effort -->
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-category">Category</label>
                                <select id="edit-task-category" name="category">
                                    <option value="">Select category...</option>
                                    ${categoryOptions.map(cat => `
                                        <option value="${cat.value}" ${task.category === cat.value ? 'selected' : ''}>
                                            ${cat.label}
                                        </option>
                                    `).join('')}
                                </select>
                            </div>
                            
                            <div class="form-group">
                                <label for="edit-task-effort">Estimated Effort</label>
                                <select id="edit-task-effort" name="effort_estimate">
                                    <option value="1h" ${task.effort_estimate === '1h' ? 'selected' : ''}>1 hour</option>
                                    <option value="2h" ${task.effort_estimate === '2h' ? 'selected' : ''}>2 hours</option>
                                    <option value="4h" ${task.effort_estimate === '4h' ? 'selected' : ''}>4 hours</option>
                                    <option value="8h" ${task.effort_estimate === '8h' ? 'selected' : ''}>8 hours</option>
                                    <option value="16h+" ${task.effort_estimate === '16h+' ? 'selected' : ''}>16+ hours</option>
                                </select>
                            </div>
                        </div>
                        
                        <!-- Progress -->
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-progress">Progress %</label>
                                <input type="number" id="edit-task-progress" name="progress_percentage" 
                                       min="0" max="100" value="${task.progress_percentage || 0}">
                            </div>
                            
                            <div class="form-group">
                                <label for="edit-task-phase">Current Phase</label>
                                <input type="text" id="edit-task-phase" name="current_phase" 
                                       value="${this.escapeHtml(task.current_phase || '')}"
                                       placeholder="e.g., research, implementation, testing">
                            </div>
                        </div>
                        
                        <!-- Impact and Urgency -->
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-impact">Impact Score (1-10)</label>
                                <input type="number" id="edit-task-impact" name="impact_score" 
                                       min="1" max="10" value="${task.impact_score || 5}">
                            </div>
                            
                            <div class="form-group">
                                <label for="edit-task-urgency">Urgency</label>
                                <select id="edit-task-urgency" name="urgency">
                                    <option value="low" ${task.urgency === 'low' ? 'selected' : ''}>Low</option>
                                    <option value="normal" ${task.urgency === 'normal' ? 'selected' : ''}>Normal</option>
                                    <option value="high" ${task.urgency === 'high' ? 'selected' : ''}>High</option>
                                    <option value="critical" ${task.urgency === 'critical' ? 'selected' : ''}>Critical</option>
                                </select>
                            </div>
                        </div>
                        
                        <!-- Notes -->
                        <div class="form-group">
                            <label for="edit-task-notes">Notes</label>
                            <textarea id="edit-task-notes" name="notes" rows="4" 
                                      placeholder="Additional details, requirements, or context...">${this.escapeHtml(task.notes || '')}</textarea>
                        </div>
                        
                        <!-- Metadata (read-only) -->
                        <div class="form-group">
                            <label>Task Information</label>
                            <div style="background: var(--light-gray); padding: 0.8rem; border-radius: var(--border-radius); font-size: 0.9rem;">
                                <div><strong>ID:</strong> ${task.id}</div>
                                ${task.created_at ? `<div><strong>Created:</strong> ${new Date(task.created_at).toLocaleString()}</div>` : ''}
                                ${task.started_at ? `<div><strong>Started:</strong> ${new Date(task.started_at).toLocaleString()}</div>` : ''}
                                ${task.completed_at ? `<div><strong>Completed:</strong> ${new Date(task.completed_at).toLocaleString()}</div>` : ''}
                            </div>
                        </div>
                        
                        <!-- Action Buttons -->
                        <div class="form-actions">
                            <button type="button" class="btn btn-secondary" onclick="ecosystemManager.closeTaskDetailsModal()">
                                <i class="fas fa-times"></i>
                                Cancel
                            </button>
                            
                            <button type="button" class="btn btn-primary" onclick="ecosystemManager.saveTaskChanges('${task.id}')">
                                <i class="fas fa-save"></i>
                                Save Changes
                            </button>
                        </div>
                    </div>
                </form>
            `;
            
            modal.classList.add('show');
            
        } catch (error) {
            console.error('Failed to load task details:', error);
            this.showToast('Failed to load task details', 'error');
        } finally {
            this.showLoading(false);
        }
    }
    
    async updateTaskStatus(taskId, newStatus = null, progress = null) {
        if (!newStatus) {
            // Show a simple prompt for now
            newStatus = prompt('Enter new status (pending, in-progress, completed, failed):');
            if (!newStatus) return;
        }
        
        if (progress === null && newStatus === 'in-progress') {
            progress = parseInt(prompt('Enter progress percentage (0-100):') || '0');
        }
        
        try {
            const updateData = { status: newStatus };
            if (progress !== null) {
                updateData.progress_percentage = progress;
            }
            
            const response = await fetch(`${this.apiBase}/tasks/${taskId}/status`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(updateData)
            });
            
            if (!response.ok) throw new Error('Failed to update task status');
            
            this.showToast('Task status updated successfully', 'success');
            await this.loadAllTasks();
            this.closeAllModals();
            
        } catch (error) {
            console.error('Failed to update task status:', error);
            this.showToast('Failed to update task status', 'error');
        }
    }
    
    async startTask(taskId) {
        await this.updateTaskStatus(taskId, 'in-progress', 0);
    }
    
    async completeTask(taskId) {
        await this.updateTaskStatus(taskId, 'completed', 100);
    }
    
    async saveTaskChanges(taskId) {
        this.showLoading(true);
        
        try {
            const form = document.getElementById('edit-task-form');
            const formData = new FormData(form);
            
            // Build the update object from form data
            const updateData = {
                id: taskId,
                title: formData.get('title'),
                status: formData.get('status'),
                priority: formData.get('priority'),
                category: formData.get('category') || '',
                effort_estimate: formData.get('effort_estimate') || '',
                progress_percentage: parseInt(formData.get('progress_percentage')) || 0,
                current_phase: formData.get('current_phase') || '',
                impact_score: parseInt(formData.get('impact_score')) || 5,
                urgency: formData.get('urgency') || 'normal',
                notes: formData.get('notes') || '',
                updated_at: new Date().toISOString()
            };
            
            console.log('Saving task changes:', updateData);
            
            const response = await fetch(`${this.apiBase}/tasks/${taskId}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(updateData)
            });
            
            if (!response.ok) {
                const errorData = await response.text();
                throw new Error(`Failed to update task: ${response.status} ${errorData}`);
            }
            
            const updatedTask = await response.json();
            console.log('Task updated successfully:', updatedTask);
            
            // Update the task in local cache
            this.tasks[taskId] = updatedTask;
            
            // Close the modal
            this.closeTaskDetailsModal();
            
            // Refresh the UI
            this.renderAllTasks();
            this.updateStats();
            
            this.showToast('Task updated successfully!', 'success');
            
        } catch (error) {
            console.error('Failed to save task changes:', error);
            this.showToast(`Failed to save changes: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }
    
    async refreshColumn(status) {
        await this.loadTasks(status);
        this.updateStats();
        this.applyFilters();
        this.showToast(`Refreshed ${status} tasks`, 'success');
    }
    
    toggleCompletedVisibility() {
        const completedColumn = document.querySelector('[data-status="completed"]');
        const toggleBtn = document.getElementById('toggle-completed');
        
        if (completedColumn.style.display === 'none') {
            completedColumn.style.display = 'flex';
            toggleBtn.innerHTML = '<i class="fas fa-eye"></i>';
        } else {
            completedColumn.style.display = 'none';
            toggleBtn.innerHTML = '<i class="fas fa-eye-slash"></i>';
        }
    }
    
    // Modal functions
    openCreateTaskModal() {
        document.getElementById('create-task-modal').classList.add('show');
        document.getElementById('task-title').focus();
    }
    
    closeCreateTaskModal() {
        document.getElementById('create-task-modal').classList.remove('show');
    }
    
    closeTaskDetailsModal() {
        document.getElementById('task-details-modal').classList.remove('show');
    }
    
    closeAllModals() {
        document.querySelectorAll('.modal').forEach(modal => {
            modal.classList.remove('show');
        });
    }
    
    updateFormForType() {
        const type = document.getElementById('task-type').value;
        const operation = document.getElementById('task-operation').value;
        
        // Update target dropdown if needed
        if (operation === 'improver') {
            this.updateTargetOptions(type);
        }
    }
    
    updateFormForOperation() {
        const operation = document.getElementById('task-operation').value;
        const type = document.getElementById('task-type').value;
        const targetGroup = document.getElementById('target-group');
        
        if (operation === 'improver' && type) {
            // Show target field for improver operations
            targetGroup.style.display = 'block';
            document.getElementById('task-target').required = true;
            this.updateTargetOptions(type);
        } else {
            // Hide target field for generator operations
            targetGroup.style.display = 'none';
            document.getElementById('task-target').required = false;
            document.getElementById('task-target').value = '';
        }
    }
    
    updateTargetOptions(type) {
        const targetSelect = document.getElementById('task-target');
        targetSelect.innerHTML = '<option value="">Select target to improve...</option>';
        
        // Get the appropriate list based on type
        const items = type === 'resource' ? this.resources : this.scenarios;
        
        if (items && items.length > 0) {
            items.forEach(item => {
                const option = document.createElement('option');
                option.value = item.name;
                option.textContent = `${item.name}${item.description ? ' - ' + item.description.substring(0, 50) : ''}`;
                targetSelect.appendChild(option);
            });
        } else {
            const option = document.createElement('option');
            option.value = '';
            option.textContent = `No ${type}s discovered`;
            option.disabled = true;
            targetSelect.appendChild(option);
        }
    }
    
    // Utility functions
    showLoading(show = true) {
        const overlay = document.getElementById('loading-overlay');
        if (show) {
            overlay.classList.add('show');
        } else {
            overlay.classList.remove('show');
        }
    }
    
    showToast(message, type = 'info') {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        
        const icon = {
            success: 'fas fa-check-circle',
            error: 'fas fa-exclamation-circle',
            warning: 'fas fa-exclamation-triangle',
            info: 'fas fa-info-circle'
        }[type] || 'fas fa-info-circle';
        
        toast.innerHTML = `
            <i class="${icon}"></i>
            <span>${this.escapeHtml(message)}</span>
        `;
        
        container.appendChild(toast);
        
        // Auto remove after 5 seconds
        setTimeout(() => {
            if (toast.parentNode) {
                toast.parentNode.removeChild(toast);
            }
        }, 5000);
        
        // Click to dismiss
        toast.addEventListener('click', () => {
            if (toast.parentNode) {
                toast.parentNode.removeChild(toast);
            }
        });
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
    
    // Dark mode functionality
    initDarkMode() {
        // Check for saved theme preference or default to light mode
        const savedTheme = localStorage.getItem('dark-mode');
        if (savedTheme === 'enabled') {
            document.body.classList.add('dark-mode');
            this.updateDarkModeIcon(true);
        }
    }
    
    toggleDarkMode() {
        const body = document.body;
        const isDarkMode = body.classList.contains('dark-mode');
        
        if (isDarkMode) {
            body.classList.remove('dark-mode');
            localStorage.setItem('dark-mode', 'disabled');
            this.updateDarkModeIcon(false);
        } else {
            body.classList.add('dark-mode');
            localStorage.setItem('dark-mode', 'enabled');
            this.updateDarkModeIcon(true);
        }
    }
    
    updateDarkModeIcon(isDarkMode) {
        const icon = document.getElementById('dark-mode-icon');
        if (icon) {
            icon.className = isDarkMode ? 'fas fa-sun' : 'fas fa-moon';
        }
    }
}

// Initialize when DOM is loaded
let ecosystemManager;

// Global functions for inline event handlers
window.openCreateTaskModal = () => ecosystemManager?.openCreateTaskModal();
window.closeCreateTaskModal = () => ecosystemManager?.closeCreateTaskModal();
window.closeTaskDetailsModal = () => ecosystemManager?.closeTaskDetailsModal();
window.applyFilters = () => ecosystemManager?.applyFilters();
window.clearFilters = () => ecosystemManager?.clearFilters();
window.refreshColumn = (status) => ecosystemManager?.refreshColumn(status);
window.toggleCompletedVisibility = () => ecosystemManager?.toggleCompletedVisibility();
window.updateFormForType = () => ecosystemManager?.updateFormForType();
window.updateFormForOperation = () => ecosystemManager?.updateFormForOperation();
window.toggleDarkMode = () => ecosystemManager?.toggleDarkMode();

document.addEventListener('DOMContentLoaded', () => {
    ecosystemManager = new EcosystemManager();
});