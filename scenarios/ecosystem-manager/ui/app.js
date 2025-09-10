// Ecosystem Manager UI JavaScript

class EcosystemManager {
    constructor() {
        // Use relative paths - Vite proxy handles routing to the correct API port
        this.apiBase = '/api';
        this.tasks = {};
        this.filteredTasks = {};
        this.operations = {};
        this.resources = [];
        this.scenarios = [];
        this.ws = null;
        this.wsReconnectInterval = null;
        
        // Queue status tracking - will be updated from settings
        this.queueStatus = {
            maxConcurrent: 1,
            availableSlots: 1,
            processorActive: false, // Default to false for safety
            lastRefresh: Date.now(),
            refreshInterval: 30000, // 30 seconds default, will be updated from settings
            refreshTimer: null
        };
        this.autoRefreshInterval = null; // Track the auto-refresh interval
        
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
            // Initialize settings (including theme) before loading content
            await this.initSettings();
            
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
            
            // Initialize queue status display
            this.initQueueStatus();
            
            // Initialize process monitoring
            this.initProcessMonitoring();
            
            // CRITICAL: Trigger immediate queue processing on page load
            // Don't wait for the timer - check for tasks to process right now
            setTimeout(() => {
                this.triggerImmediateProcessing();
            }, 1000); // Small delay to let initialization complete
            
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
        this.updateQueueDisplay();
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
        // Use relative path - Vite proxy handles routing to the correct API port  
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
            this.handleWebSocketMessage(event);
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
    
    handleTaskStarted(task) {
        // Refresh to move task from pending to in-progress
        this.loadAllTasks();
        this.showToast(`Task started: ${task.title}`, 'info');
    }
    
    handleTaskCompleted(task) {
        // Refresh to move task to completed
        this.loadAllTasks();
        this.showToast(`Task ${task.title} completed successfully!`, 'success');
    }
    
    handleTaskFailed(task) {
        // Refresh to move task to failed
        this.loadAllTasks();
        
        // Show detailed error message
        const errorMsg = task.results?.error || 'Unknown error occurred';
        const truncatedError = errorMsg.length > 100 ? errorMsg.substring(0, 100) + '...' : errorMsg;
        this.showToast(`Task ${task.title} failed: ${truncatedError}`, 'error');
    }
    
    handleTaskCreated(task) {
        // Refresh to show new task
        this.loadAllTasks();
        this.showToast(`New task created: ${task.title}`, 'info');
    }
    
    handleTaskUpdated(task) {
        // Refresh to show updated task in correct column
        this.loadAllTasks();
        this.showToast(`Task updated: ${task.title}`, 'info');
    }
    
    handleQueueStatusChanged(status) {
        // Update queue status display
        this.queueStatus = status;
        this.updateQueueDisplay();
        // Refresh tasks in case status change affects them
        this.loadAllTasks();
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
        
        // Re-render and update queue status
        this.renderTasks();
        this.updateQueueDisplay();
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
        // Clear any existing auto-refresh interval
        if (this.autoRefreshInterval) {
            clearInterval(this.autoRefreshInterval);
        }
        
        // Use refresh interval from settings (convert seconds to milliseconds)
        const refreshMs = (this.settings?.refresh_interval || 30) * 1000;
        
        // Refresh based on settings interval (as fallback if WebSocket fails)
        this.autoRefreshInterval = setInterval(() => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
                console.log('WebSocket not connected, refreshing tasks...');
                this.loadAllTasks();
            }
        }, refreshMs);
        
        console.log(`Auto-refresh set to ${refreshMs/1000} seconds`);
    }
    
    setupEventListeners() {
        // Form submission
        document.getElementById('create-task-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleCreateTask();
        });
        
        // Type change in create form (for radio buttons)
        document.querySelectorAll('input[name="type"]').forEach(radio => {
            radio.addEventListener('change', () => {
                this.updateFormForType();
            });
        });
        
        // Operation change in create form (for radio buttons)
        document.querySelectorAll('input[name="operation"]').forEach(radio => {
            radio.addEventListener('change', () => {
                this.updateFormForOperation();
            });
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
    
    // Drag and Drop Methods
    handleDragStart(e, task) {
        this.draggedTask = task;
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/html', e.target.outerHTML);
        
        // Add visual feedback
        e.target.classList.add('dragging');
        
        // Disable clicking during drag
        e.target.onclick = null;
        
        // Start auto-scroll detection
        this.startAutoScroll();
        
        // Add dragging indicator to body for visual feedback
        document.body.classList.add('dragging-active');
    }
    
    handleDragEnd(e) {
        e.target.classList.remove('dragging');
        
        // Restore click handler after drag with proper binding
        const taskId = e.target.getAttribute('data-task-id');
        e.target.onclick = (event) => {
            // Don't trigger if clicking on delete button
            if (!event.target.closest('.btn-delete')) {
                this.showTaskDetails(taskId);
            }
        };
        
        // Clean up drag indicators
        document.querySelectorAll('.kanban-column').forEach(col => {
            col.classList.remove('drag-over');
        });
        
        // Stop auto-scroll
        this.stopAutoScroll();
        
        // Remove dragging indicator from body
        document.body.classList.remove('dragging-active');
        
        this.draggedTask = null;
    }
    
    handleDragOver(e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        
        const column = e.currentTarget;
        column.classList.add('drag-over');
    }
    
    handleDragEnter(e) {
        e.preventDefault();
        e.currentTarget.classList.add('drag-over');
    }
    
    handleDragLeave(e) {
        e.preventDefault();
        if (!e.currentTarget.contains(e.relatedTarget)) {
            e.currentTarget.classList.remove('drag-over');
        }
    }
    
    async handleDrop(e, targetStatus) {
        e.preventDefault();
        e.currentTarget.classList.remove('drag-over');
        
        if (!this.draggedTask) return;
        
        const sourceStatus = this.findTaskStatus(this.draggedTask.id);
        
        // Don't do anything if dropped in same column
        if (sourceStatus === targetStatus) return;
        
        // Clean up task state when moving to a different status
        const cleanedTask = { ...this.draggedTask };
        
        // Clear execution results when moving from completed/failed to pending/in-progress
        const isBackwardsTransition = (sourceStatus === 'completed' || sourceStatus === 'failed') && 
                                      (targetStatus === 'pending' || targetStatus === 'in-progress');
        
        if (isBackwardsTransition) {
            // Clear all execution-related data for a fresh start
            delete cleanedTask.results;
            delete cleanedTask.started_at;
            delete cleanedTask.completed_at;
            cleanedTask.progress_percentage = 0;
            cleanedTask.current_phase = '';
        }
        
        // Update phase based on new status
        if (targetStatus === 'pending') {
            cleanedTask.current_phase = 'pending';
            if (!isBackwardsTransition) {
                cleanedTask.progress_percentage = 0;
            }
        } else if (targetStatus === 'in-progress') {
            cleanedTask.current_phase = 'in-progress';
            if (!isBackwardsTransition && (cleanedTask.progress_percentage === 0 || cleanedTask.progress_percentage === 100)) {
                cleanedTask.progress_percentage = 25;
            }
        } else if (targetStatus === 'review') {
            cleanedTask.current_phase = 'review';
            cleanedTask.progress_percentage = 75;
        } else if (targetStatus === 'completed') {
            cleanedTask.current_phase = 'completed';
            cleanedTask.progress_percentage = 100;
        } else if (targetStatus === 'failed') {
            cleanedTask.current_phase = 'failed';
        }
        
        // Update task status via API
        try {
            // Build the request body - explicitly include all fields we want to update
            const requestBody = {
                ...cleanedTask,
                status: targetStatus,
                updated_at: new Date().toISOString()
            };
            
            // Log for debugging
            console.log(`Moving task ${this.draggedTask.id} from ${sourceStatus} to ${targetStatus}`, {
                isBackwardsTransition,
                hasResults: !!requestBody.results,
                requestBody
            });
            
            const response = await fetch(`${this.apiBase}/tasks/${this.draggedTask.id}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestBody)
            });
            
            if (!response.ok) {
                throw new Error(`Failed to update task: ${response.status}`);
            }
            
            const updatedTask = await response.json();
            
            // Log the response for debugging
            console.log(`Task ${updatedTask.id} updated response:`, {
                hasResults: !!updatedTask.results,
                status: updatedTask.status,
                updatedTask
            });
            
            // Ensure results are cleared for pending/in-progress if they somehow persist
            if ((targetStatus === 'pending' || targetStatus === 'in-progress') && 
                (sourceStatus === 'completed' || sourceStatus === 'failed')) {
                if (updatedTask.results) {
                    console.warn(`WARNING: Backend didn't clear results for task ${updatedTask.id} when moving to ${targetStatus}. Clearing locally.`);
                    delete updatedTask.results;
                    delete updatedTask.started_at;
                    delete updatedTask.completed_at;
                }
            }
            
            // Refresh all tasks to ensure UI is in sync with backend
            await this.loadAllTasks();
            this.updateStats();
            this.applyFilters();
            
            this.showToast(`Task moved to ${targetStatus.replace('-', ' ')}`, 'success');
            
            // Trigger queue processing if task moved to pending or in-progress
            if (targetStatus === 'pending' || targetStatus === 'in-progress') {
                try {
                    await this.triggerQueueProcessing();
                    console.log('Queue processing triggered after task move');
                } catch (error) {
                    console.warn('Could not trigger queue processing after move:', error);
                    // Don't fail the move operation if trigger fails
                }
            }
            
        } catch (error) {
            console.error('Failed to update task status:', error);
            this.showToast(`Failed to move task: ${error.message}`, 'error');
        }
    }
    
    // Auto-scrolling during drag operations
    startAutoScroll() {
        // Stop any existing auto-scroll
        this.stopAutoScroll();
        
        // Track mouse/touch position and scroll state
        this.autoScrollData = {
            active: true,
            mouseX: 0,
            mouseY: 0,
            scrollSpeed: 15,
            edgeSize: 80, // Distance from edge to trigger scroll
            animationId: null
        };
        
        // Handle both mouse and touch events for position tracking
        this.handleAutoScrollMove = (e) => {
            if (!this.autoScrollData.active) return;
            
            // Get coordinates from mouse or touch event
            if (e.type === 'touchmove' && e.touches && e.touches.length > 0) {
                this.autoScrollData.mouseX = e.touches[0].clientX;
                this.autoScrollData.mouseY = e.touches[0].clientY;
            } else if (e.type === 'dragover') {
                this.autoScrollData.mouseX = e.clientX;
                this.autoScrollData.mouseY = e.clientY;
            }
        };
        
        // Listen for drag/touch movement
        document.addEventListener('dragover', this.handleAutoScrollMove);
        document.addEventListener('touchmove', this.handleAutoScrollMove, { passive: false });
        
        // Auto-scroll animation loop
        const performAutoScroll = () => {
            if (!this.autoScrollData || !this.autoScrollData.active) return;
            
            const { mouseX, mouseY, scrollSpeed, edgeSize } = this.autoScrollData;
            const viewportWidth = window.innerWidth;
            const viewportHeight = window.innerHeight;
            
            // Calculate distance from edges
            const distanceFromLeft = mouseX;
            const distanceFromRight = viewportWidth - mouseX;
            const distanceFromTop = mouseY;
            const distanceFromBottom = viewportHeight - mouseY;
            
            let scrollX = 0;
            let scrollY = 0;
            
            // Horizontal scrolling
            if (distanceFromLeft < edgeSize && distanceFromLeft > 0) {
                // Scroll left - stronger scroll the closer to edge
                scrollX = -scrollSpeed * (1 - distanceFromLeft / edgeSize);
            } else if (distanceFromRight < edgeSize && distanceFromRight > 0) {
                // Scroll right
                scrollX = scrollSpeed * (1 - distanceFromRight / edgeSize);
            }
            
            // Vertical scrolling
            if (distanceFromTop < edgeSize && distanceFromTop > 0) {
                // Scroll up
                scrollY = -scrollSpeed * (1 - distanceFromTop / edgeSize);
            } else if (distanceFromBottom < edgeSize && distanceFromBottom > 0) {
                // Scroll down
                scrollY = scrollSpeed * (1 - distanceFromBottom / edgeSize);
            }
            
            // Apply scrolling to the window
            if (scrollX !== 0 || scrollY !== 0) {
                window.scrollBy(scrollX, scrollY);
                
                // Also handle horizontal scrolling on the kanban board for responsive layouts
                const kanbanBoard = document.querySelector('.kanban-board');
                if (kanbanBoard) {
                    const kanbanRect = kanbanBoard.getBoundingClientRect();
                    const mouseRelativeX = mouseX - kanbanRect.left;
                    
                    // Check if kanban board has horizontal overflow
                    if (kanbanBoard.scrollWidth > kanbanBoard.clientWidth) {
                        if (mouseRelativeX < edgeSize && kanbanBoard.scrollLeft > 0) {
                            kanbanBoard.scrollLeft -= scrollSpeed * (1 - mouseRelativeX / edgeSize);
                        } else if (mouseRelativeX > kanbanRect.width - edgeSize && 
                                 kanbanBoard.scrollLeft < kanbanBoard.scrollWidth - kanbanBoard.clientWidth) {
                            kanbanBoard.scrollLeft += scrollSpeed * (1 - (kanbanRect.width - mouseRelativeX) / edgeSize);
                        }
                    }
                }
                
                // Also check individual columns for vertical scrolling
                const columns = document.querySelectorAll('.column-content');
                columns.forEach(column => {
                    const columnRect = column.getBoundingClientRect();
                    // Check if mouse is over this column
                    if (mouseX >= columnRect.left && mouseX <= columnRect.right) {
                        const mouseRelativeY = mouseY - columnRect.top;
                        
                        // Vertical scrolling within column
                        if (column.scrollHeight > column.clientHeight) {
                            if (mouseRelativeY < edgeSize && column.scrollTop > 0) {
                                column.scrollTop -= scrollSpeed * (1 - mouseRelativeY / edgeSize);
                            } else if (mouseRelativeY > columnRect.height - edgeSize && 
                                     column.scrollTop < column.scrollHeight - column.clientHeight) {
                                column.scrollTop += scrollSpeed * (1 - (columnRect.height - mouseRelativeY) / edgeSize);
                            }
                        }
                    }
                });
            }
            
            // Continue the animation loop
            this.autoScrollData.animationId = requestAnimationFrame(performAutoScroll);
        };
        
        // Start the animation loop
        this.autoScrollData.animationId = requestAnimationFrame(performAutoScroll);
    }
    
    stopAutoScroll() {
        if (this.autoScrollData) {
            this.autoScrollData.active = false;
            
            if (this.autoScrollData.animationId) {
                cancelAnimationFrame(this.autoScrollData.animationId);
            }
        }
        
        if (this.handleAutoScrollMove) {
            document.removeEventListener('dragover', this.handleAutoScrollMove);
            document.removeEventListener('touchmove', this.handleAutoScrollMove);
            this.handleAutoScrollMove = null;
        }
        
        this.autoScrollData = null;
    }
    
    // Touch support for mobile drag and drop
    addTouchDragSupport(element, task) {
        let touchItem = null;
        let touchOffset = null;
        let draggedClone = null;
        
        element.addEventListener('touchstart', (e) => {
            // Prevent default to avoid scrolling while dragging
            e.preventDefault();
            
            const touch = e.touches[0];
            touchItem = element;
            this.draggedTask = task;
            
            // Calculate offset from touch point to element position
            const rect = element.getBoundingClientRect();
            touchOffset = {
                x: touch.clientX - rect.left,
                y: touch.clientY - rect.top
            };
            
            // Create a visual clone that follows the finger
            draggedClone = element.cloneNode(true);
            draggedClone.style.position = 'fixed';
            draggedClone.style.width = rect.width + 'px';
            draggedClone.style.opacity = '0.8';
            draggedClone.style.zIndex = '10000';
            draggedClone.style.pointerEvents = 'none';
            draggedClone.style.transform = 'rotate(2deg)';
            draggedClone.style.transition = 'none';
            document.body.appendChild(draggedClone);
            
            // Position the clone at the touch point
            draggedClone.style.left = (touch.clientX - touchOffset.x) + 'px';
            draggedClone.style.top = (touch.clientY - touchOffset.y) + 'px';
            
            // Add dragging class to original element
            element.classList.add('dragging');
            
            // Start auto-scroll
            this.startAutoScroll();
            
            // Add dragging indicator to body for visual feedback
            document.body.classList.add('dragging-active');
        }, { passive: false });
        
        element.addEventListener('touchmove', (e) => {
            if (!touchItem || !draggedClone) return;
            e.preventDefault();
            
            const touch = e.touches[0];
            
            // Move the clone with the finger
            draggedClone.style.left = (touch.clientX - touchOffset.x) + 'px';
            draggedClone.style.top = (touch.clientY - touchOffset.y) + 'px';
            
            // Find the element under the touch point
            draggedClone.style.display = 'none'; // Temporarily hide clone to get element below
            const elementBelow = document.elementFromPoint(touch.clientX, touch.clientY);
            draggedClone.style.display = '';
            
            // Check if we're over a drop zone (kanban column)
            if (elementBelow) {
                const dropZone = elementBelow.closest('.kanban-column');
                
                // Remove drag-over class from all columns
                document.querySelectorAll('.kanban-column').forEach(col => {
                    col.classList.remove('drag-over');
                });
                
                // Add drag-over class to current column
                if (dropZone) {
                    dropZone.classList.add('drag-over');
                }
            }
        }, { passive: false });
        
        element.addEventListener('touchend', async (e) => {
            if (!touchItem || !draggedClone) return;
            e.preventDefault();
            
            const touch = e.changedTouches[0];
            
            // Remove the clone
            if (draggedClone) {
                draggedClone.remove();
                draggedClone = null;
            }
            
            // Remove dragging class
            element.classList.remove('dragging');
            
            // Find the element under the touch point
            const elementBelow = document.elementFromPoint(touch.clientX, touch.clientY);
            
            if (elementBelow) {
                const dropZone = elementBelow.closest('.kanban-column');
                
                if (dropZone) {
                    // Get the target status from the column
                    const targetStatus = dropZone.getAttribute('data-status');
                    
                    if (targetStatus) {
                        // Simulate the drop event
                        const fakeDropEvent = {
                            preventDefault: () => {},
                            currentTarget: dropZone
                        };
                        
                        await this.handleDrop(fakeDropEvent, targetStatus);
                    }
                }
            }
            
            // Clean up
            document.querySelectorAll('.kanban-column').forEach(col => {
                col.classList.remove('drag-over');
            });
            
            // Stop auto-scroll
            this.stopAutoScroll();
            
            // Remove dragging indicator from body
            document.body.classList.remove('dragging-active');
            
            touchItem = null;
            touchOffset = null;
            this.draggedTask = null;
        }, { passive: false });
        
        // Prevent context menu on long press
        element.addEventListener('contextmenu', (e) => {
            e.preventDefault();
        });
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
        
        // Add drop zone event listeners to column
        const column = container.closest('.kanban-column');
        if (column && !column.hasAttribute('data-drop-listeners-added')) {
            column.addEventListener('dragover', (e) => this.handleDragOver(e));
            column.addEventListener('dragenter', (e) => this.handleDragEnter(e));
            column.addEventListener('dragleave', (e) => this.handleDragLeave(e));
            column.addEventListener('drop', (e) => this.handleDrop(e, status));
            column.setAttribute('data-drop-listeners-added', 'true');
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
    
    getPhaseIcon(phase) {
        const phaseIconMap = {
            'pending': 'fa-clock',
            'in-progress': 'fa-spinner fa-spin',
            'review': 'fa-eye',
            'completed': 'fa-check-circle',
            'failed': 'fa-exclamation-triangle',
            'cancelled': 'fa-times-circle'
        };
        return phaseIconMap[phase] || 'fa-circle';
    }
    
    createTaskCard(task) {
        const card = document.createElement('div');
        card.className = 'task-card';
        card.setAttribute('data-task-id', task.id);
        card.setAttribute('draggable', 'true');
        
        // Set onclick with proper binding
        card.onclick = (e) => {
            // Don't trigger if clicking on delete button
            if (!e.target.closest('.btn-delete')) {
                this.showTaskDetails(task.id);
            }
        };
        
        // Add drag event listeners
        card.addEventListener('dragstart', (e) => this.handleDragStart(e, task));
        card.addEventListener('dragend', (e) => this.handleDragEnd(e));
        
        // Add touch event support for mobile drag and drop
        this.addTouchDragSupport(card, task);
        
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
                <div class="task-phase-chip">
                    <i class="fas ${this.getPhaseIcon(task.current_phase)}"></i> ${task.current_phase}
                </div>
            ` : ''}
            
            ${task.status === 'failed' && task.results && task.results.error ? `
                <div class="task-error">
                    <i class="fas fa-exclamation-triangle"></i>
                    <span class="error-message">${this.formatErrorText(task.results.error)}</span>
                </div>
            ` : ''}
            
            <div class="task-footer">
                <div class="task-id">${task.id}</div>
                <div class="task-actions">
                    <button class="btn-icon btn-delete" onclick="event.stopPropagation(); ecosystemManager.deleteTask('${task.id}', '${task.status}')" title="Delete Task">
                        <i class="fas fa-trash"></i>
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
    
    // Removed duplicate functions - see lines 1688 and 1698 for the actual implementations
    
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
            
            // Refresh all tasks to show the new task
            await this.loadAllTasks();
            this.updateStats();
            this.applyFilters();
            
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
            console.log(`Fetching task details for: ${taskId}`);
            const response = await fetch(`${this.apiBase}/tasks/${taskId}`);
            
            if (!response.ok) {
                const errorText = await response.text();
                console.error(`Failed to load task ${taskId}:`, response.status, errorText);
                throw new Error(`Failed to load task details: ${response.status}`);
            }
            
            const task = await response.json();
            console.log('Task details loaded:', task);
            
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
                        
                        <!-- Task Results (only show for completed/failed tasks) -->
                        ${task.results && (task.status === 'completed' || task.status === 'failed') ? `
                            <div class="form-group">
                                <label>Execution Results</label>
                                <div class="execution-results ${task.results.success ? 'success' : 'error'}">
                                    <div style="margin-bottom: 0.5rem;">
                                        <strong>Status:</strong> 
                                        <span class="${task.results.success ? 'status-success' : 'status-error'}">
                                            ${task.results.success ? '✅ Success' : '❌ Failed'}
                                        </span>
                                        ${task.results.timeout_failure ? '<span style="color: #ff9800; margin-left: 8px;">⏰ TIMEOUT</span>' : ''}
                                    </div>
                                    
                                    <!-- Timing Information -->
                                    ${task.results.execution_time || task.results.timeout_allowed || task.results.prompt_size ? `
                                        <div style="margin-bottom: 0.5rem; padding: 0.5rem; background: rgba(0, 0, 0, 0.05); border-radius: 4px;">
                                            <div style="display: flex; justify-content: space-between; font-size: 0.9em;">
                                                ${task.results.execution_time ? `<span><strong>⏱️ Runtime:</strong> ${task.results.execution_time}</span>` : ''}
                                                ${task.results.timeout_allowed ? `<span><strong>⏰ Timeout:</strong> ${task.results.timeout_allowed}</span>` : ''}
                                            </div>
                                            ${task.results.prompt_size ? `<div style="font-size: 0.9em; margin-top: 4px;"><strong>📝 Prompt Size:</strong> ${task.results.prompt_size}</div>` : ''}
                                            ${task.results.started_at ? `<div style="font-size: 0.8em; color: #666; margin-top: 4px;">Started: ${new Date(task.results.started_at).toLocaleString()}</div>` : ''}
                                        </div>
                                    ` : ''}
                                    ${task.results.error ? `
                                        <div style="margin-bottom: 0.5rem;">
                                            <strong>Error:</strong> 
                                            <div class="status-error" style="margin-top: 0.5rem; padding: 0.5rem; background: rgba(244, 67, 54, 0.1); border-radius: 4px;">${this.formatErrorText(task.results.error)}</div>
                                        </div>
                                    ` : ''}
                                    ${task.results.output ? `
                                        <details style="margin-top: 0.5rem;">
                                            <summary class="output-summary">
                                                📋 View Claude Output (click to expand)
                                            </summary>
                                            <pre class="claude-output">${this.escapeHtml(task.results.output)}</pre>
                                        </details>
                                    ` : ''}
                                </div>
                            </div>
                        ` : ''}
                        
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
                            
                            <button type="button" class="btn btn-info" onclick="ecosystemManager.viewTaskPrompt('${task.id}')" title="View the prompt that was/will be sent to Claude">
                                <i class="fas fa-file-alt"></i>
                                View Prompt
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
            console.error('Failed to load task details:', error, error.stack);
            this.showToast(`Failed to load task details: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }
    
    
    async deleteTask(taskId, status) {
        // No confirmation needed for completed or failed tasks
        if (status !== 'completed' && status !== 'failed') {
            const confirmed = confirm(`Are you sure you want to delete this ${status} task?`);
            if (!confirmed) return;
        }
        
        try {
            const response = await fetch(`${this.apiBase}/tasks/${taskId}`, {
                method: 'DELETE'
            });
            
            if (!response.ok) {
                throw new Error('Failed to delete task');
            }
            
            // Refresh all tasks to ensure UI is in sync with backend
            await this.loadAllTasks();
            this.updateStats();
            this.applyFilters();
            
            this.showToast('Task deleted successfully', 'success');
        } catch (error) {
            console.error('Failed to delete task:', error);
            this.showToast('Failed to delete task', 'error');
        }
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
            
            // Close the modal
            this.closeTaskDetailsModal();
            
            // Refresh all tasks to show the updated task in the correct column
            await this.loadAllTasks();
            this.updateStats();
            this.applyFilters();
            
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
    
    async viewTaskPrompt(taskId) {
        try {
            const response = await fetch(`${this.apiBase}/tasks/${taskId}/prompt/assembled`);
            if (!response.ok) {
                throw new Error('Failed to fetch prompt');
            }
            
            const data = await response.json();
            
            // Remove any existing prompt modal first
            const existingModal = document.getElementById('promptModal');
            if (existingModal) {
                existingModal.remove();
            }
            
            // Create a modal to display the prompt
            const modalDiv = document.createElement('div');
            modalDiv.id = 'promptModal';
            modalDiv.className = 'modal show';
            modalDiv.style.cssText = 'position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 999999;';
            
            modalDiv.innerHTML = `
                <div class="modal-content" style="background: white; border-radius: 8px; max-width: 900px; width: 90%; max-height: 90vh; display: flex; flex-direction: column; overflow: hidden;">
                    <div class="modal-header" style="padding: 1rem; border-bottom: 1px solid #e0e0e0; display: flex; justify-content: space-between; align-items: center;">
                        <h2 style="margin: 0;">Task Prompt - ${taskId}</h2>
                        <button class="modal-close" onclick="document.getElementById('promptModal').remove()" style="background: none; border: none; font-size: 1.5rem; cursor: pointer;">
                            <i class="fas fa-times"></i>
                        </button>
                    </div>
                    <div class="modal-body" style="padding: 1rem; overflow-y: auto; flex: 1;">
                        <div style="margin-bottom: 1rem; padding: 0.75rem; background: #f5f5f5; border-radius: 4px;">
                            <div style="display: flex; justify-content: space-between; flex-wrap: wrap; gap: 1rem;">
                                <div>
                                    <strong>Prompt Length:</strong> ${data.prompt_length.toLocaleString()} characters
                                </div>
                                <div>
                                    <strong>Source:</strong> ${data.prompt_cached ? '📁 Cached from execution' : '🔄 Freshly assembled'}
                                </div>
                                <div>
                                    <strong>Task Status:</strong> ${data.task_status}
                                </div>
                            </div>
                        </div>
                        
                        <div style="position: relative;">
                            <button class="btn btn-sm btn-secondary" 
                                    onclick="navigator.clipboard.writeText(document.getElementById('promptContent').textContent); ecosystemManager.showToast('Prompt copied to clipboard', 'success')"
                                    style="position: absolute; top: 0.5rem; right: 0.5rem; z-index: 10;">
                                <i class="fas fa-copy"></i> Copy
                            </button>
                            <pre id="promptContent" style="background: #f5f5f5; padding: 1rem; border-radius: 4px; overflow: auto; max-height: 60vh; white-space: pre-wrap; word-wrap: break-word; margin: 0;">${this.escapeHtml(data.prompt)}</pre>
                        </div>
                    </div>
                    <div class="modal-footer" style="padding: 1rem; border-top: 1px solid #e0e0e0; display: flex; justify-content: flex-end;">
                        <button class="btn btn-secondary" onclick="document.getElementById('promptModal').remove()">
                            Close
                        </button>
                    </div>
                </div>
            `;
            
            // Add click handler to close on backdrop click
            modalDiv.addEventListener('click', (e) => {
                if (e.target === modalDiv) {
                    modalDiv.remove();
                }
            });
            
            // Append to body
            document.body.appendChild(modalDiv);
            
        } catch (error) {
            console.error('Failed to fetch prompt:', error);
            this.showToast(`Failed to fetch prompt: ${error.message}`, 'error');
        }
    }
    
    closeAllModals() {
        document.querySelectorAll('.modal').forEach(modal => {
            modal.classList.remove('show');
        });
    }
    
    async refreshAll() {
        this.showLoading(true);
        try {
            // Set timeout for the entire refresh operation
            const refreshPromise = Promise.race([
                this.performRefreshOperations(),
                new Promise((_, reject) => 
                    setTimeout(() => reject(new Error('Refresh timeout')), 10000)
                )
            ]);

            await refreshPromise;
            this.showToast('All data refreshed', 'success');
        } catch (error) {
            console.error('Failed to refresh all data:', error);
            if (error.message === 'Refresh timeout') {
                this.showToast('Refresh timed out - server may be slow', 'warning');
            } else {
                this.showToast('Failed to refresh data', 'error');
            }
        } finally {
            this.showLoading(false);
        }
    }

    async performRefreshOperations() {
        // Load all tasks with timeout
        await this.loadAllTasks();
        
        // Fetch and update queue status
        await this.fetchQueueProcessorStatus();
        
        // Trigger IMMEDIATE queue processing to start claude-code if there are pending tasks
        try {
            await this.triggerImmediateProcessing();
        } catch (error) {
            console.warn('Could not trigger immediate processing:', error);
            // Don't fail the entire refresh if trigger fails
        }
        
        // Update stats
        this.updateStats();
        
        // Apply filters
        this.applyFilters();
    }

    async triggerQueueProcessing() {
        try {
            const response = await fetch(`${this.apiBase}/queue/trigger`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                signal: AbortSignal.timeout(5000) // 5 second timeout
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.message || 'Failed to trigger queue processing');
            }

            const result = await response.json();
            console.log('Queue processing triggered:', result);
            return result;
        } catch (error) {
            console.warn('Failed to trigger queue processing:', error);
            throw error;
        }
    }
    
    updateFormForType() {
        const type = document.querySelector('input[name="type"]:checked')?.value || 'resource';
        const operation = document.querySelector('input[name="operation"]:checked')?.value || 'generator';
        
        // Update target dropdown if needed
        if (operation === 'improver') {
            this.updateTargetOptions(type);
        }
    }
    
    updateFormForOperation() {
        const operation = document.querySelector('input[name="operation"]:checked')?.value || 'generator';
        const type = document.querySelector('input[name="type"]:checked')?.value || 'resource';
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
        
        // Auto remove after timeout (longer for errors)
        const timeout = type === 'error' ? 15000 : 8000; // 15s for errors, 8s for others
        setTimeout(() => {
            if (toast.parentNode) {
                toast.parentNode.removeChild(toast);
            }
        }, timeout);
        
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
    
    // Format error text for display with proper line breaks and structure
    formatErrorText(errorText) {
        if (!errorText) return '';
        
        // First escape HTML to prevent XSS
        const escapedText = this.escapeHtml(errorText);
        
        // Convert \n to actual line breaks
        const withLineBreaks = escapedText.replace(/\\n/g, '\n');
        
        // Convert actual newlines to <br> tags
        return withLineBreaks.replace(/\n/g, '<br>');
    }
    
    // Dark mode functionality
    // Settings Management
    async initSettings() {
        // Load settings from backend API, fallback to localStorage, then defaults
        await this.loadSettingsFromBackend();
        
        // Apply settings to queue status
        this.queueStatus.maxConcurrent = this.settings.slots || 1;
        this.queueStatus.refreshInterval = (this.settings.refresh_interval || 30) * 1000; // Convert to milliseconds
        this.queueStatus.processorActive = this.settings.active || false;
        
        console.log('Applied settings to queue:', {
            slots: this.queueStatus.maxConcurrent,
            refreshInterval: this.queueStatus.refreshInterval,
            active: this.queueStatus.processorActive
        });
        
        // If settings say processor should be active, apply that to backend
        // This handles the case where API starts paused but settings say active
        if (this.settings.active) {
            console.log('Settings indicate processor should be active, applying to backend...');
            this.applyProcessorActiveState(true);
        }
        
        // Apply theme
        this.applyTheme(this.settings.theme);
        
        // Set up media query listener for auto theme
        if (window.matchMedia) {
            const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
            mediaQuery.addListener(() => this.applyTheme(this.settings.theme));
        }
    }
    
    async loadSettingsFromBackend() {
        const defaultSettings = {
            // Display settings
            theme: 'light',
            
            // Queue processor settings
            slots: 1,
            refresh_interval: 30,
            active: false,
            
            // Agent settings
            max_turns: 60,
            allowed_tools: 'Read,Write,Edit,Bash,LS,Glob,Grep',
            skip_permissions: true,
            task_timeout: 30
        };
        
        try {
            // Try loading from backend first
            const response = await fetch(`${this.apiBase}/settings`);
            if (response.ok) {
                const data = await response.json();
                if (data.success && data.settings) {
                    this.settings = data.settings;
                    console.log('Settings loaded from backend:', this.settings);
                    return;
                }
            }
        } catch (error) {
            console.warn('Failed to load settings from backend:', error);
        }
        
        // Fallback to localStorage
        try {
            const saved = localStorage.getItem('ecosystem-manager-settings');
            if (saved) {
                this.settings = { ...defaultSettings, ...JSON.parse(saved) };
                console.log('Settings loaded from localStorage (backend unavailable)');
                return;
            }
        } catch (error) {
            console.warn('Failed to load settings from localStorage:', error);
        }
        
        // Final fallback to defaults
        this.settings = defaultSettings;
        console.log('Settings loaded from defaults');
    }
    
    loadSettings() {
        // Legacy function - now replaced by loadSettingsFromBackend
        return this.settings || this.loadSettingsFromBackend();
    }
    
    async saveSettings(settings) {
        try {
            const newSettings = { ...this.settings, ...settings };
            
            // Save to backend API
            const response = await fetch(`${this.apiBase}/settings`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(newSettings)
            });
            
            if (response.ok) {
                const data = await response.json();
                if (data.success) {
                    this.settings = data.settings;
                    
                    // Also save to localStorage as backup
                    localStorage.setItem('ecosystem-manager-settings', JSON.stringify(this.settings));
                    
                    // Apply theme immediately if it changed
                    if (settings.theme) {
                        this.applyTheme(settings.theme);
                    }
                    
                    console.log('Settings saved to backend successfully');
                    return true;
                }
            }
            
            throw new Error('Backend save failed');
            
        } catch (error) {
            console.error('Failed to save settings to backend:', error);
            
            // Fallback to localStorage only
            try {
                this.settings = { ...this.settings, ...settings };
                localStorage.setItem('ecosystem-manager-settings', JSON.stringify(this.settings));
                
                if (settings.theme) {
                    this.applyTheme(settings.theme);
                }
                
                console.log('Settings saved to localStorage as fallback');
                return true;
            } catch (localError) {
                console.error('Failed to save settings even to localStorage:', localError);
                return false;
            }
        }
    }
    
    async applyProcessorActiveState(shouldBeActive) {
        // Apply the processor active state by re-saving settings
        // This triggers the backend to start/stop the processor
        try {
            const response = await fetch(`${this.apiBase}/settings`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    ...this.settings,
                    active: shouldBeActive
                })
            });
            
            if (response.ok) {
                const data = await response.json();
                if (data.success) {
                    console.log(`Processor ${shouldBeActive ? 'activated' : 'paused'} successfully`);
                    this.queueStatus.processorActive = shouldBeActive;
                    this.updateQueueDisplay();
                }
            }
        } catch (error) {
            console.warn('Failed to apply processor active state:', error);
        }
    }
    
    applyTheme(theme) {
        const body = document.body;
        
        switch (theme) {
            case 'dark':
                body.classList.add('dark-mode');
                break;
            case 'light':
                body.classList.remove('dark-mode');
                break;
            case 'auto':
                if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
                    body.classList.add('dark-mode');
                } else {
                    body.classList.remove('dark-mode');
                }
                break;
        }
    }
    
    openSettingsModal() {
        const modal = document.getElementById('settings-modal');
        
        // Populate form with current settings
        this.populateSettingsForm();
        
        modal.classList.add('show');
    }
    
    closeSettingsModal() {
        const modal = document.getElementById('settings-modal');
        modal.classList.remove('show');
    }
    
    populateSettingsForm() {
        // Display settings
        document.getElementById('settings-theme').value = this.settings.theme;
        
        // Queue processor settings
        document.getElementById('settings-slots').value = this.settings.slots;
        document.getElementById('slots-value').textContent = this.settings.slots;
        document.getElementById('settings-refresh').value = this.settings.refresh_interval;
        document.getElementById('settings-active').checked = this.settings.active;
        
        // Agent settings
        document.getElementById('settings-max-turns').value = this.settings.max_turns;
        document.getElementById('max-turns-value').textContent = this.settings.max_turns;
        document.getElementById('settings-tools').value = this.settings.allowed_tools;
        document.getElementById('settings-skip-permissions').checked = this.settings.skip_permissions;
        document.getElementById('settings-task-timeout').value = this.settings.task_timeout || 30;
        document.getElementById('task-timeout-value').textContent = this.settings.task_timeout || 30;
    }
    
    async saveSettingsFromForm() {
        const form = document.getElementById('settings-form');
        const formData = new FormData(form);
        
        const newSettings = {
            // Display settings
            theme: formData.get('theme'),
            
            // Queue processor settings
            slots: parseInt(formData.get('slots')),
            refresh_interval: parseInt(formData.get('refresh_interval')),
            active: formData.get('active') === 'on',
            
            // Agent settings
            max_turns: parseInt(formData.get('max_turns')),
            allowed_tools: formData.get('allowed_tools'),
            skip_permissions: formData.get('skip_permissions') === 'on',
            task_timeout: parseInt(formData.get('task_timeout'))
        };
        
        this.showLoading(true);
        
        try {
            const success = await this.saveSettings(newSettings);
            if (success) {
                // Apply updated settings to queue status
                this.queueStatus.maxConcurrent = newSettings.slots;
                this.queueStatus.refreshInterval = newSettings.refresh_interval * 1000; // Convert to milliseconds
                this.queueStatus.processorActive = newSettings.active;
                
                // Restart the refresh timer with new interval
                this.queueStatus.lastRefresh = Date.now(); // Reset the timer
                this.startRefreshTimer();
                
                // Update auto-refresh with new interval
                this.setupAutoRefresh();
                
                // Update queue display immediately
                this.updateQueueDisplay();
                
                this.closeSettingsModal();
                this.showToast('Settings saved successfully', 'success');
            } else {
                this.showToast('Failed to save settings', 'error');
            }
        } catch (error) {
            console.error('Error saving settings:', error);
            this.showToast('Failed to save settings', 'error');
        } finally {
            this.showLoading(false);
        }
    }
    
    async resetSettingsToDefault() {
        this.showLoading(true);
        
        try {
            // Call the backend reset API
            const response = await fetch(`${this.apiBase}/settings/reset`, {
                method: 'POST'
            });
            
            if (response.ok) {
                const data = await response.json();
                if (data.success) {
                    this.settings = data.settings;
                    
                    // Update localStorage as backup
                    localStorage.setItem('ecosystem-manager-settings', JSON.stringify(this.settings));
                    
                    // Update UI
                    this.populateSettingsForm();
                    this.applyTheme(this.settings.theme);
                    
                    this.showToast('Settings reset to defaults', 'info');
                    console.log('Settings reset via backend API');
                    return;
                }
            }
            
            throw new Error('Backend reset failed');
            
        } catch (error) {
            console.error('Failed to reset settings via backend:', error);
            
            // Fallback to local reset
            const defaultSettings = {
                theme: 'light',
                slots: 1,
                refresh_interval: 30,
                active: false,
                max_turns: 60,
                allowed_tools: 'Read,Write,Edit,Bash,LS,Glob,Grep',
                skip_permissions: true,
                task_timeout: 30
            };
            
            const success = await this.saveSettings(defaultSettings);
            if (success) {
                this.populateSettingsForm();
                this.showToast('Settings reset to defaults (fallback)', 'info');
            } else {
                this.showToast('Failed to reset settings', 'error');
            }
        } finally {
            this.showLoading(false);
        }
    }
    
    updateSliderValue(sliderId, valueId) {
        const slider = document.getElementById(sliderId);
        const valueSpan = document.getElementById(valueId);
        if (slider && valueSpan) {
            valueSpan.textContent = slider.value;
        }
    }
    
    
    // Legacy function name for backwards compatibility
    initDarkMode() {
        this.initSettings();
    }
    
    // Queue Status Management
    initQueueStatus() {
        // Initialize UI elements with settings-based values
        this.updateQueueDisplay();
        
        // Start the refresh countdown timer
        this.startRefreshTimer();
        
        // Fetch initial queue processor status
        this.fetchQueueProcessorStatus();
        
        console.log('🔄 Queue status indicators initialized with settings:', {
            slots: this.queueStatus.maxConcurrent,
            refreshInterval: this.queueStatus.refreshInterval / 1000 + 's',
            active: this.queueStatus.processorActive
        });
    }
    
    updateQueueDisplay() {
        // Calculate available slots based on in-progress tasks
        const inProgressCount = (this.tasks['in-progress'] || []).length;
        this.queueStatus.availableSlots = Math.max(0, this.queueStatus.maxConcurrent - inProgressCount);
        
        // Update UI elements
        const availableSlotsEl = document.getElementById('available-slots');
        const maxSlotsEl = document.getElementById('max-slots');
        const processorStatusEl = document.getElementById('processor-status');
        const processorIconEl = document.getElementById('processor-status-icon');
        
        if (availableSlotsEl) availableSlotsEl.textContent = this.queueStatus.availableSlots;
        if (maxSlotsEl) maxSlotsEl.textContent = this.queueStatus.maxConcurrent;
        
        if (processorStatusEl && processorIconEl) {
            if (this.queueStatus.processorActive) {
                processorStatusEl.textContent = 'Active';
                processorIconEl.className = 'fas fa-play';
                processorIconEl.style.color = 'var(--success-color)';
            } else {
                processorStatusEl.textContent = 'Paused';
                processorIconEl.className = 'fas fa-pause';
                processorIconEl.style.color = 'var(--warning-color)';
            }
        }
    }
    
    startRefreshTimer() {
        // Clear existing timer
        if (this.queueStatus.refreshTimer) {
            clearInterval(this.queueStatus.refreshTimer);
        }
        
        // Update countdown every second
        this.queueStatus.refreshTimer = setInterval(() => {
            const now = Date.now();
            const elapsed = now - this.queueStatus.lastRefresh;
            const remaining = Math.max(0, this.queueStatus.refreshInterval - elapsed);
            const secondsRemaining = Math.ceil(remaining / 1000);
            
            const countdownEl = document.getElementById('refresh-countdown');
            if (countdownEl) {
                countdownEl.textContent = secondsRemaining;
                
                // Add visual indicator when refreshing
                const timerEl = document.querySelector('.queue-timer');
                if (secondsRemaining <= 1 && timerEl) {
                    timerEl.classList.add('refreshing');
                } else if (timerEl) {
                    timerEl.classList.remove('refreshing');
                }
            }
            
            // Reset timer when cycle completes
            if (remaining === 0) {
                this.queueStatus.lastRefresh = now;
                // Fetch fresh queue status from API
                this.fetchQueueProcessorStatus();
                // Also update queue display when refresh happens
                this.updateQueueDisplay();
            }
        }, 1000);
    }
    
    async fetchQueueProcessorStatus() {
        try {
            const response = await fetch(`${this.apiBase}/queue/status`);
            if (response.ok) {
                const status = await response.json();
                // Only update processor active status, don't override settings-based values
                this.queueStatus.processorActive = status.processor_active;
                // Don't override maxConcurrent or refreshInterval - those come from settings
                // Just update available slots based on current task count
                this.updateQueueDisplay();
            }
        } catch (error) {
            console.warn('Failed to fetch queue processor status:', error);
            // Fallback to health endpoint  
            try {
                const healthResponse = await fetch('/health');
                if (healthResponse.ok) {
                    const healthStatus = await healthResponse.json();
                    this.queueStatus.processorActive = healthStatus.maintenanceState === 'active';
                    this.updateQueueDisplay();
                }
            } catch (fallbackError) {
                console.warn('Failed to fetch health status as fallback:', fallbackError);
            }
        }
    }
    
    // ====== PROCESS MONITOR & LOG VIEWER FUNCTIONALITY ======
    
    runningProcesses = {};
    logViewerOpen = false;
    logAutoScroll = true;
    activeTaskTimers = {};
    
    // Initialize process monitoring
    async initProcessMonitoring() {
        // Start periodic process monitoring
        setInterval(() => {
            this.fetchRunningProcesses();
        }, 5000); // Check every 5 seconds
        
        // Initial load
        this.fetchRunningProcesses();
    }
    
    async fetchRunningProcesses() {
        try {
            const response = await fetch(`${this.apiBase}/processes/running`);
            if (!response.ok) throw new Error('Failed to fetch processes');
            
            const data = await response.json();
            this.runningProcesses = {};
            
            // Convert array to object for easier lookup
            if (data.processes && Array.isArray(data.processes)) {
                data.processes.forEach(process => {
                    this.runningProcesses[process.task_id] = process;
                });
            }
            
            this.updateProcessMonitor();
            this.updateExecutionTimers();
            
        } catch (error) {
            console.warn('Failed to fetch running processes:', error);
            // Clear process monitor if API fails
            this.runningProcesses = {};
            this.updateProcessMonitor();
        }
    }
    
    updateProcessMonitor() {
        const processMonitor = document.getElementById('process-monitor');
        const processCount = document.getElementById('running-process-count');
        const processDetails = document.getElementById('process-details');
        
        const activeProcesses = Object.keys(this.runningProcesses);
        const count = activeProcesses.length;
        
        if (count > 0) {
            processMonitor.style.display = 'flex';
            processCount.textContent = count;
            
            // Show details of active processes
            const processInfo = activeProcesses.map(taskId => {
                const process = this.runningProcesses[taskId];
                const duration = this.formatDuration(process.start_time);
                return `${taskId.slice(0, 20)}... (${duration})`;
            }).join(', ');
            
            processDetails.textContent = processInfo;
            processDetails.title = activeProcesses.map(taskId => {
                const process = this.runningProcesses[taskId];
                return `Task: ${taskId}\nPID: ${process.process_id}\nRuntime: ${process.duration}`;
            }).join('\n\n');
            
        } else {
            processMonitor.style.display = 'none';
        }
        
        // Update task cards with execution status
        this.updateTaskExecutionStatus();
    }
    
    updateTaskExecutionStatus() {
        // Update all in-progress task cards
        Object.keys(this.runningProcesses).forEach(taskId => {
            const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
            if (taskCard) {
                this.addExecutionStatusToCard(taskCard, taskId);
            }
        });
        
        // Remove execution status from cards no longer running
        document.querySelectorAll('.task-execution-status').forEach(statusEl => {
            const taskCard = statusEl.closest('.task-card');
            const taskId = taskCard?.getAttribute('data-task-id');
            if (taskId && !this.runningProcesses[taskId]) {
                statusEl.remove();
            }
        });
    }
    
    addExecutionStatusToCard(taskCard, taskId) {
        // Don't add if already exists
        if (taskCard.querySelector('.task-execution-status')) return;
        
        const process = this.runningProcesses[taskId];
        if (!process) return;
        
        const statusEl = document.createElement('div');
        statusEl.className = 'task-execution-status executing';
        statusEl.innerHTML = `
            <i class="fas fa-brain fa-spin"></i>
            <span>Executing with Claude Code...</span>
            <span class="execution-timer" id="timer-${taskId}">--:--</span>
            <button class="process-terminate-btn" onclick="ecosystemManager.terminateProcess('${taskId}')" title="Terminate Process">
                <i class="fas fa-stop"></i>
            </button>
        `;
        
        // Insert after task-meta
        const taskMeta = taskCard.querySelector('.task-meta');
        if (taskMeta) {
            taskMeta.insertAdjacentElement('afterend', statusEl);
        } else {
            // Fallback: insert at end of card
            taskCard.appendChild(statusEl);
        }
        
        // Start timer for this task
        this.startExecutionTimer(taskId, process.start_time);
    }
    
    startExecutionTimer(taskId, startTime) {
        if (this.activeTaskTimers[taskId]) {
            clearInterval(this.activeTaskTimers[taskId]);
        }
        
        this.activeTaskTimers[taskId] = setInterval(() => {
            const timerEl = document.getElementById(`timer-${taskId}`);
            if (timerEl && this.runningProcesses[taskId]) {
                const duration = this.formatDuration(startTime);
                timerEl.textContent = duration;
            } else {
                // Clean up timer if element or process no longer exists
                clearInterval(this.activeTaskTimers[taskId]);
                delete this.activeTaskTimers[taskId];
            }
        }, 1000);
    }
    
    updateExecutionTimers() {
        Object.keys(this.runningProcesses).forEach(taskId => {
            const process = this.runningProcesses[taskId];
            const timerEl = document.getElementById(`timer-${taskId}`);
            if (timerEl) {
                timerEl.textContent = this.formatDuration(process.start_time);
            }
        });
    }
    
    formatDuration(startTime) {
        const start = new Date(startTime);
        const now = new Date();
        const diffMs = now - start;
        const diffSec = Math.floor(diffMs / 1000);
        const minutes = Math.floor(diffSec / 60);
        const seconds = diffSec % 60;
        return `${minutes}:${seconds.toString().padStart(2, '0')}`;
    }
    
    async terminateProcess(taskId) {
        const confirmed = confirm(`Are you sure you want to terminate the Claude Code process for task "${taskId}"? This will stop the current execution.`);
        if (!confirmed) return;
        
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
            
            this.showToast('Process terminated successfully', 'success');
            
            // Immediately update UI
            delete this.runningProcesses[taskId];
            this.updateProcessMonitor();
            
            // Clear timer
            if (this.activeTaskTimers[taskId]) {
                clearInterval(this.activeTaskTimers[taskId]);
                delete this.activeTaskTimers[taskId];
            }
            
        } catch (error) {
            console.error('Failed to terminate process:', error);
            this.showToast(`Failed to terminate process: ${error.message}`, 'error');
        }
    }
    
    // ====== LOG VIEWER FUNCTIONALITY ======
    
    openLogViewer(taskId) {
        const modal = document.getElementById('log-viewer-modal');
        const title = document.getElementById('log-viewer-title');
        
        title.textContent = `Task Execution Logs - ${taskId}`;
        modal.classList.add('show');
        this.logViewerOpen = true;
        this.currentLogTaskId = taskId;
        
        // Clear previous logs
        this.clearLogViewer();
        
        // Start log streaming (if process is running)
        if (this.runningProcesses[taskId]) {
            this.startLogStreaming(taskId);
        } else {
            this.addLogEntry('info', 'Task is not currently executing. Logs will appear when execution starts.');
        }
    }
    
    closeLogViewer() {
        const modal = document.getElementById('log-viewer-modal');
        modal.classList.remove('show');
        this.logViewerOpen = false;
        this.currentLogTaskId = null;
        
        // Stop log streaming
        if (this.logStreamInterval) {
            clearInterval(this.logStreamInterval);
            this.logStreamInterval = null;
        }
    }
    
    clearLogViewer() {
        const logOutput = document.getElementById('log-output');
        logOutput.innerHTML = `
            <div class="log-placeholder">
                <i class="fas fa-terminal"></i>
                <p>Waiting for task execution logs...</p>
                <p class="log-hint">Logs will appear here when the task starts executing with Claude Code</p>
            </div>
        `;
    }
    
    toggleLogAutoScroll() {
        this.logAutoScroll = !this.logAutoScroll;
        const button = document.getElementById('log-auto-scroll-toggle');
        if (this.logAutoScroll) {
            button.classList.add('log-auto-scroll-enabled');
            this.scrollLogToBottom();
        } else {
            button.classList.remove('log-auto-scroll-enabled');
        }
    }
    
    addLogEntry(type, message) {
        const logOutput = document.getElementById('log-output');
        
        // Remove placeholder if it exists
        const placeholder = logOutput.querySelector('.log-placeholder');
        if (placeholder) {
            placeholder.remove();
        }
        
        const timestamp = new Date().toLocaleTimeString();
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry ${type}`;
        logEntry.innerHTML = `
            <span class="log-timestamp">[${timestamp}]</span>
            <span class="log-message">${this.escapeHtml(message)}</span>
        `;
        
        logOutput.appendChild(logEntry);
        
        // Auto-scroll if enabled
        if (this.logAutoScroll) {
            this.scrollLogToBottom();
        }
    }
    
    scrollLogToBottom() {
        const logOutput = document.getElementById('log-output');
        logOutput.scrollTop = logOutput.scrollHeight;
    }
    
    startLogStreaming(taskId) {
        // Mock log streaming - in real implementation, this would connect to actual logs
        // For now, we'll simulate logs based on task phases
        this.addLogEntry('info', `Starting execution for task: ${taskId}`);
        this.addLogEntry('info', 'Assembled prompt and calling Claude Code...');
        
        // Simulate periodic log updates
        this.logStreamInterval = setInterval(() => {
            if (this.runningProcesses[taskId]) {
                const messages = [
                    'Claude Code is analyzing the task...',
                    'Reading files and understanding context...',
                    'Executing commands and making changes...',
                    'Validating changes and running tests...',
                ];
                const randomMessage = messages[Math.floor(Math.random() * messages.length)];
                this.addLogEntry('info', randomMessage);
            }
        }, 10000); // Add a message every 10 seconds
    }
    
    // ====== ENHANCED TASK DETAILS MODAL ======
    
    // Override the existing showTaskDetails to add log viewer button
    async showTaskDetails(taskId) {
        this.showLoading(true);
        
        try {
            const response = await fetch(`${this.apiBase}/tasks/${taskId}`);
            if (!response.ok) {
                throw new Error(`Failed to load task: ${response.status} ${response.statusText}`);
            }
            
            const task = await response.json();
            console.log('Task details loaded:', task);
            
            const modal = document.getElementById('task-details-modal');
            const titleElement = document.getElementById('task-details-title');
            const contentElement = document.getElementById('task-details-content');
            
            titleElement.textContent = `Edit Task`;
            
            // Get category options based on task type
            const categoryOptions = this.categoryOptions[task.type] || [];
            
            contentElement.innerHTML = `
                <form id="edit-task-form">
                    <div class="task-details-container">
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
                                    <option value="review" ${task.status === 'review' ? 'selected' : ''}>Review</option>
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
                        
                        <!-- Category and Effort -->
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-category">Category</label>
                                <select id="edit-task-category" name="category">
                                    <option value="">No Category</option>
                                    ${categoryOptions.map(cat => 
                                        `<option value="${cat.value}" ${task.category === cat.value ? 'selected' : ''}>${cat.label}</option>`
                                    ).join('')}
                                </select>
                            </div>
                            
                            <div class="form-group">
                                <label for="edit-task-effort">Effort Estimate</label>
                                <select id="edit-task-effort" name="effort_estimate">
                                    <option value="1h" ${task.effort_estimate === '1h' ? 'selected' : ''}>1 hour</option>
                                    <option value="2h" ${task.effort_estimate === '2h' ? 'selected' : ''}>2 hours</option>
                                    <option value="4h" ${task.effort_estimate === '4h' ? 'selected' : ''}>4 hours</option>
                                    <option value="8h" ${task.effort_estimate === '8h' ? 'selected' : ''}>8 hours</option>
                                    <option value="16h+" ${task.effort_estimate === '16h+' ? 'selected' : ''}>16+ hours</option>
                                </select>
                            </div>
                        </div>
                        
                        <!-- Progress and Phase -->
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-progress">Progress (%)</label>
                                <input type="range" id="edit-task-progress" name="progress_percentage" min="0" max="100" 
                                       value="${task.progress_percentage || 0}" 
                                       oninput="document.getElementById('progress-value').textContent = this.value + '%'">
                                <span id="progress-value">${task.progress_percentage || 0}%</span>
                            </div>
                            
                            <div class="form-group">
                                <label for="edit-task-phase">Current Phase</label>
                                <select id="edit-task-phase" name="current_phase">
                                    <option value="">No Phase</option>
                                    <option value="initialization" ${task.current_phase === 'initialization' ? 'selected' : ''}>Initialization</option>
                                    <option value="research" ${task.current_phase === 'research' ? 'selected' : ''}>Research</option>
                                    <option value="implementation" ${task.current_phase === 'implementation' ? 'selected' : ''}>Implementation</option>
                                    <option value="testing" ${task.current_phase === 'testing' ? 'selected' : ''}>Testing</option>
                                    <option value="documentation" ${task.current_phase === 'documentation' ? 'selected' : ''}>Documentation</option>
                                    <option value="completed" ${task.current_phase === 'completed' ? 'selected' : ''}>Completed</option>
                                    <option value="prompt_assembled" ${task.current_phase === 'prompt_assembled' ? 'selected' : ''}>Prompt Assembled</option>
                                </select>
                            </div>
                        </div>
                        
                        <!-- Impact and Urgency -->
                        <div class="form-row">
                            <div class="form-group">
                                <label for="edit-task-impact">Impact Score (1-10)</label>
                                <input type="range" id="edit-task-impact" name="impact_score" min="1" max="10" 
                                       value="${task.impact_score || 5}" 
                                       oninput="document.getElementById('impact-value').textContent = this.value">
                                <span id="impact-value">${task.impact_score || 5}</span>
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
                        
                        <!-- Process Controls (show if task is running) -->
                        ${this.runningProcesses[taskId] ? `
                            <div class="task-execution-status executing">
                                <i class="fas fa-brain fa-spin"></i>
                                <span>Task is currently executing with Claude Code</span>
                                <span class="execution-timer">${this.formatDuration(this.runningProcesses[taskId].start_time)}</span>
                                <button type="button" class="btn btn-secondary" onclick="ecosystemManager.openLogViewer('${taskId}')">
                                    <i class="fas fa-terminal"></i>
                                    Follow Logs
                                </button>
                                <button type="button" class="process-terminate-btn" onclick="ecosystemManager.terminateProcess('${taskId}')">
                                    <i class="fas fa-stop"></i>
                                    Terminate
                                </button>
                            </div>
                        ` : ''}
                        
                        <!-- Notes -->
                        <div class="form-group">
                            <label for="edit-task-notes">Notes</label>
                            <textarea id="edit-task-notes" name="notes" rows="4" 
                                      placeholder="Additional details, requirements, or context...">${this.escapeHtml(task.notes || '')}</textarea>
                        </div>
                        
                        <!-- Task Results (only show for completed/failed tasks) -->
                        ${task.results && (task.status === 'completed' || task.status === 'failed') ? `
                            <div class="form-group">
                                <label>Execution Results</label>
                                <div class="execution-results ${task.results.success ? 'success' : 'error'}">
                                    <div style="margin-bottom: 0.5rem;">
                                        <strong>Status:</strong> 
                                        <span class="${task.results.success ? 'status-success' : 'status-error'}">
                                            ${task.results.success ? '✅ Success' : '❌ Failed'}
                                        </span>
                                        ${task.results.timeout_failure ? '<span style="color: #ff9800; margin-left: 8px;">⏰ TIMEOUT</span>' : ''}
                                    </div>
                                    
                                    <!-- Timing Information -->
                                    ${task.results.execution_time || task.results.timeout_allowed || task.results.prompt_size ? `
                                        <div style="margin-bottom: 0.5rem; padding: 0.5rem; background: rgba(0, 0, 0, 0.05); border-radius: 4px;">
                                            <div style="display: flex; justify-content: space-between; font-size: 0.9em;">
                                                ${task.results.execution_time ? `<span><strong>⏱️ Runtime:</strong> ${task.results.execution_time}</span>` : ''}
                                                ${task.results.timeout_allowed ? `<span><strong>⏰ Timeout:</strong> ${task.results.timeout_allowed}</span>` : ''}
                                            </div>
                                            ${task.results.prompt_size ? `<div style="font-size: 0.9em; margin-top: 4px;"><strong>📝 Prompt Size:</strong> ${task.results.prompt_size}</div>` : ''}
                                            ${task.results.started_at ? `<div style="font-size: 0.8em; color: #666; margin-top: 4px;">Started: ${new Date(task.results.started_at).toLocaleString()}</div>` : ''}
                                        </div>
                                    ` : ''}
                                    ${task.results.error ? `
                                        <div style="margin-bottom: 0.5rem;">
                                            <strong>Error:</strong> 
                                            <div class="status-error" style="margin-top: 0.5rem; padding: 0.5rem; background: rgba(244, 67, 54, 0.1); border-radius: 4px;">${this.formatErrorText(task.results.error)}</div>
                                        </div>
                                    ` : ''}
                                    ${task.results.output ? `
                                        <details style="margin-top: 0.5rem;">
                                            <summary class="output-summary">
                                                📋 View Claude Output (click to expand)
                                            </summary>
                                            <pre class="claude-output">${this.escapeHtml(task.results.output)}</pre>
                                        </details>
                                    ` : ''}
                                </div>
                            </div>
                        ` : ''}
                        
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
                            
                            <button type="button" class="btn btn-info" onclick="ecosystemManager.viewTaskPrompt('${task.id}')" title="View the prompt that was/will be sent to Claude">
                                <i class="fas fa-file-alt"></i>
                                View Prompt
                            </button>
                            
                            ${task.status === 'in-progress' && !this.runningProcesses[taskId] ? `
                                <button type="button" class="btn btn-secondary" onclick="ecosystemManager.openLogViewer('${taskId}')">
                                    <i class="fas fa-terminal"></i>
                                    View Logs
                                </button>
                            ` : ''}
                            
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
            console.error('Failed to load task details:', error, error.stack);
            this.showToast(`Failed to load task details: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }
    
    // ====== WEBSOCKET ENHANCEMENTS ======
    
    handleWebSocketMessage(event) {
        try {
            const message = JSON.parse(event.data);
            console.log('WebSocket message:', message);
            
            switch (message.type) {
                case 'task_progress':
                    this.handleTaskProgressUpdate(message.data);
                    break;
                case 'task_completed':
                    this.handleTaskCompleted(message.data);
                    break;
                case 'task_failed':
                    this.handleTaskFailed(message.data);
                    break;
                case 'process_started':
                    this.handleProcessStarted(message.data);
                    break;
                case 'process_terminated':
                    this.handleProcessTerminated(message.data);
                    break;
                case 'log_entry':
                    this.handleLogEntry(message.data);
                    break;
                default:
                    console.log('Unknown WebSocket message type:', message.type);
            }
        } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
        }
    }
    
    handleTaskProgressUpdate(taskData) {
        // Update task in memory
        const status = taskData.status;
        if (this.tasks[status]) {
            const taskIndex = this.tasks[status].findIndex(t => t.id === taskData.id);
            if (taskIndex !== -1) {
                this.tasks[status][taskIndex] = taskData;
            }
        }
        
        // Update task card UI
        this.updateTaskCard(taskData);
    }
    
    handleTaskCompleted(taskData) {
        this.handleTaskStatusChange(taskData, 'completed');
        // Remove from running processes
        delete this.runningProcesses[taskData.id];
        this.updateProcessMonitor();
    }
    
    handleTaskFailed(taskData) {
        this.handleTaskStatusChange(taskData, 'failed');
        // Remove from running processes
        delete this.runningProcesses[taskData.id];
        this.updateProcessMonitor();
    }
    
    handleProcessStarted(data) {
        const { task_id, process_id, start_time } = data;
        this.runningProcesses[task_id] = {
            task_id,
            process_id,
            start_time
        };
        this.updateProcessMonitor();
        
        // Add log entry if log viewer is open for this task
        if (this.logViewerOpen && this.currentLogTaskId === task_id) {
            this.addLogEntry('success', `Process started (PID: ${process_id})`);
        }
    }
    
    handleProcessTerminated(data) {
        const { task_id } = data;
        delete this.runningProcesses[task_id];
        this.updateProcessMonitor();
        
        // Add log entry if log viewer is open for this task
        if (this.logViewerOpen && this.currentLogTaskId === task_id) {
            this.addLogEntry('warning', 'Process terminated');
        }
    }
    
    handleLogEntry(data) {
        const { task_id, level, message } = data;
        if (this.logViewerOpen && this.currentLogTaskId === task_id) {
            this.addLogEntry(level, message);
        }
    }
    
    handleTaskStatusChange(taskData, newStatus) {
        // Move task between status arrays
        const oldStatus = this.findTaskStatus(taskData.id);
        if (oldStatus && oldStatus !== newStatus) {
            // Remove from old status
            this.tasks[oldStatus] = this.tasks[oldStatus].filter(t => t.id !== taskData.id);
            
            // Add to new status
            if (!this.tasks[newStatus]) {
                this.tasks[newStatus] = [];
            }
            this.tasks[newStatus].push(taskData);
            
            // Re-render and update queue status
            this.renderTasks();
            this.updateQueueDisplay();
        }
    }
    
    findTaskStatus(taskId) {
        for (const [status, tasks] of Object.entries(this.tasks)) {
            if (tasks.some(t => t.id === taskId)) {
                return status;
            }
        }
        return null;
    }
    
    // ====== IMMEDIATE PROCESSING FUNCTIONALITY ======
    
    async triggerImmediateProcessing() {
        console.log('🚀 Triggering immediate queue processing...');
        try {
            // First, check if processor is active
            const statusResponse = await fetch(`${this.apiBase}/queue/status`);
            if (statusResponse.ok) {
                const status = await statusResponse.json();
                if (!status.processor_active) {
                    console.log('⚠️ Processor not active, skipping immediate processing');
                    return;
                }
                
                console.log(`📊 Queue status: ${status.pending_count} pending, ${status.executing_count} executing, ${status.available_slots} slots available`);
                
                // If there are pending tasks and available slots, trigger processing
                if (status.pending_count > 0 && status.available_slots > 0) {
                    console.log('✅ Conditions met for immediate processing - triggering...');
                    await this.triggerQueueProcessing();
                    
                    // Also refresh the UI to show any changes
                    setTimeout(() => {
                        this.loadAllTasks();
                        this.fetchRunningProcesses();
                    }, 2000);
                } else {
                    console.log('ℹ️ No processing needed:', {
                        pending: status.pending_count,
                        available: status.available_slots
                    });
                }
            }
        } catch (error) {
            console.error('Failed to trigger immediate processing:', error);
            // Don't show error toast as this is background operation
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
// Form functions
window.updateFormForType = () => ecosystemManager?.updateFormForType();
window.updateFormForOperation = () => ecosystemManager?.updateFormForOperation();

// Settings functions
window.openSettingsModal = () => ecosystemManager?.openSettingsModal();
window.closeSettingsModal = () => ecosystemManager?.closeSettingsModal();
window.saveSettings = () => ecosystemManager?.saveSettingsFromForm();
window.resetSettingsToDefault = () => ecosystemManager?.resetSettingsToDefault();
window.updateSliderValue = (sliderId, valueId) => ecosystemManager?.updateSliderValue(sliderId, valueId);

// Process monitoring and log viewer functions
window.terminateProcess = (taskId) => ecosystemManager?.terminateProcess(taskId);
window.openLogViewer = (taskId) => ecosystemManager?.openLogViewer(taskId);
window.closeLogViewer = () => ecosystemManager?.closeLogViewer();
window.toggleLogAutoScroll = () => ecosystemManager?.toggleLogAutoScroll();
window.clearLogViewer = () => ecosystemManager?.clearLogViewer();

// Legacy dark mode function (for backwards compatibility)
window.toggleDarkMode = () => ecosystemManager?.toggleDarkMode();

document.addEventListener('DOMContentLoaded', () => {
    ecosystemManager = new EcosystemManager();
    // Make it globally accessible for debugging
    window.ecosystemManager = ecosystemManager;
});