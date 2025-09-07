// Ecosystem Manager UI JavaScript

class EcosystemManager {
    constructor() {
        this.apiBase = '/api';
        this.tasks = {};
        this.filteredTasks = {};
        this.operations = {};
        this.resources = [];
        this.scenarios = [];
        
        // Configuration
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
        
        // Initialize
        this.init();
    }
    
    async init() {
        console.log('🚀 Initializing Ecosystem Manager UI...');
        
        try {
            // Load operations configuration
            await this.loadOperations();
            
            // Load discovered resources and scenarios
            await this.loadDiscoveryData();
            
            // Load initial tasks
            await this.loadAllTasks();
            
            // Set up auto-refresh
            this.setupAutoRefresh();
            
            // Set up event listeners
            this.setupEventListeners();
            
            console.log('✅ Ecosystem Manager UI initialized successfully');
        } catch (error) {
            console.error('❌ Failed to initialize UI:', error);
            this.showToast('Failed to initialize application', 'error');
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
        const statuses = ['pending', 'in-progress', 'completed', 'failed'];
        
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
    
    setupAutoRefresh() {
        // Refresh every 30 seconds
        setInterval(() => {
            this.loadAllTasks();
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
        const totalTasks = Object.values(this.tasks).flat().length;
        const activeTasks = (this.tasks['pending'] || []).length + (this.tasks['in-progress'] || []).length;
        const completedTasks = (this.tasks['completed'] || []).length;
        
        document.getElementById('total-tasks').textContent = totalTasks;
        document.getElementById('active-tasks').textContent = activeTasks;
        document.getElementById('completed-tasks').textContent = completedTasks;
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
    
    renderAllColumns() {
        const statuses = ['pending', 'in-progress', 'completed', 'failed'];
        
        statuses.forEach(status => {
            this.renderColumn(status);
        });
    }
    
    renderColumn(status) {
        const tasks = this.filteredTasks[status] || [];
        const container = document.getElementById(`${status}-tasks`);
        const countElement = document.getElementById(`${status.replace('-', '-')}-count`);
        
        // Update count
        if (countElement) {
            countElement.textContent = tasks.length;
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
                <span class="task-tag priority-${task.priority}">${task.priority}</span>
                ${task.category ? `<span class="task-tag category-${task.category}">${task.category}</span>` : ''}
                ${task.tags ? task.tags.map(tag => `<span class="task-tag">${tag}</span>`).join('') : ''}
            </div>
            
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
            category: formData.get('category'),
            priority: formData.get('priority') || 'medium',
            effort_estimate: formData.get('effort_estimate'),
            notes: formData.get('notes') || ''
        };
        
        // Add target for improver operations
        if (taskData.operation === 'improver' && formData.get('target')) {
            taskData.requirements = {
                [`target_${taskData.type}`]: formData.get('target')
            };
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
            
            titleElement.textContent = `Task: ${task.title}`;
            
            contentElement.innerHTML = `
                <div style="padding: 1.5rem;">
                    <div class="form-row mb-3">
                        <div>
                            <strong>Type:</strong> ${task.type}
                        </div>
                        <div>
                            <strong>Operation:</strong> ${task.operation}
                        </div>
                    </div>
                    
                    <div class="form-row mb-3">
                        <div>
                            <strong>Status:</strong> 
                            <span class="task-tag priority-${task.priority}">${task.status}</span>
                        </div>
                        <div>
                            <strong>Priority:</strong> 
                            <span class="task-tag priority-${task.priority}">${task.priority}</span>
                        </div>
                    </div>
                    
                    ${task.category ? `
                        <div class="mb-3">
                            <strong>Category:</strong> 
                            <span class="task-tag category-${task.category}">${task.category}</span>
                        </div>
                    ` : ''}
                    
                    ${task.progress_percentage !== undefined ? `
                        <div class="mb-3">
                            <strong>Progress:</strong>
                            <div class="progress-bar" style="margin-top: 0.5rem;">
                                <div class="progress-fill" style="width: ${task.progress_percentage}%"></div>
                            </div>
                            <div class="progress-text">
                                <span>${task.current_phase || 'Not started'}</span>
                                <span>${task.progress_percentage}%</span>
                            </div>
                        </div>
                    ` : ''}
                    
                    ${task.effort_estimate ? `
                        <div class="mb-3">
                            <strong>Estimated Effort:</strong> ${task.effort_estimate}
                        </div>
                    ` : ''}
                    
                    ${task.created_at ? `
                        <div class="mb-3">
                            <strong>Created:</strong> ${new Date(task.created_at).toLocaleString()}
                        </div>
                    ` : ''}
                    
                    ${task.started_at ? `
                        <div class="mb-3">
                            <strong>Started:</strong> ${new Date(task.started_at).toLocaleString()}
                        </div>
                    ` : ''}
                    
                    ${task.notes ? `
                        <div class="mb-3">
                            <strong>Notes:</strong>
                            <div style="margin-top: 0.5rem; padding: 0.8rem; background: var(--light-gray); border-radius: var(--border-radius);">
                                ${this.escapeHtml(task.notes)}
                            </div>
                        </div>
                    ` : ''}
                    
                    ${task.requirements ? `
                        <div class="mb-3">
                            <strong>Requirements:</strong>
                            <pre style="margin-top: 0.5rem; padding: 0.8rem; background: var(--light-gray); border-radius: var(--border-radius); font-size: 0.9rem; overflow-x: auto;">${JSON.stringify(task.requirements, null, 2)}</pre>
                        </div>
                    ` : ''}
                    
                    <div class="form-actions">
                        <button class="btn btn-secondary" onclick="ecosystemManager.updateTaskStatus('${task.id}')">
                            <i class="fas fa-edit"></i>
                            Update Status
                        </button>
                        
                        ${task.status === 'pending' ? `
                            <button class="btn btn-primary" onclick="ecosystemManager.startTask('${task.id}')">
                                <i class="fas fa-play"></i>
                                Start Task
                            </button>
                        ` : ''}
                        
                        ${task.status === 'in-progress' ? `
                            <button class="btn btn-primary" onclick="ecosystemManager.completeTask('${task.id}')">
                                <i class="fas fa-check"></i>
                                Mark Complete
                            </button>
                        ` : ''}
                    </div>
                </div>
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
}

// Global functions for inline event handlers
window.openCreateTaskModal = () => ecosystemManager.openCreateTaskModal();
window.closeCreateTaskModal = () => ecosystemManager.closeCreateTaskModal();
window.closeTaskDetailsModal = () => ecosystemManager.closeTaskDetailsModal();
window.applyFilters = () => ecosystemManager.applyFilters();
window.clearFilters = () => ecosystemManager.clearFilters();
window.refreshColumn = (status) => ecosystemManager.refreshColumn(status);
window.toggleCompletedVisibility = () => ecosystemManager.toggleCompletedVisibility();
window.updateFormForType = () => ecosystemManager.updateFormForType();
window.updateFormForOperation = () => ecosystemManager.updateFormForOperation();

// Initialize when DOM is loaded
let ecosystemManager;
document.addEventListener('DOMContentLoaded', () => {
    ecosystemManager = new EcosystemManager();
});