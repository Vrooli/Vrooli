// Ecosystem Manager - Main Application
import { resolveApiBase, resolveWsBase } from '@vrooli/api-base';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import { ApiClient } from './modules/ApiClient.js';
import { TaskManager } from './modules/TaskManager.js';
import { SettingsManager } from './modules/SettingsManager.js';
import { ProcessMonitor } from './modules/ProcessMonitor.js';
import { UIComponents } from './modules/UIComponents.js';
import { WebSocketHandler } from './modules/WebSocketHandler.js';
import { DragDropHandler } from './modules/DragDropHandler.js';
import { AutoSteerManager } from './modules/AutoSteerManager.js';
import { ProfilePerformanceManager } from './modules/ProfilePerformanceManager.js';
import { RECYCLER_TEST_PRESETS } from './data/recycler-test-presets.js';
import { TagMultiSelect } from './components/TagMultiSelect.js';

const BRIDGE_FLAG = '__ecosystemManagerBridgeInitialized';
const API_BASE = resolveApiBase({ appendSuffix: true, apiSuffix: '/api' });
const WS_BASE = resolveWsBase({ appendSuffix: true, apiSuffix: '/ws' });

const bootstrapIframeBridge = () => {
    if (typeof window === 'undefined') {
        return;
    }
    if (window[BRIDGE_FLAG]) {
        return;
    }

    if (window.parent && window.parent !== window) {
        let parentOrigin;
        try {
            if (document.referrer) {
                parentOrigin = new URL(document.referrer).origin;
            }
        } catch (error) {
            console.warn('[EcosystemManager] Unable to resolve parent origin for iframe bridge:', error);
        }

        initIframeBridgeChild({ appId: 'ecosystem-manager', parentOrigin });
    }

    window[BRIDGE_FLAG] = true;
};

bootstrapIframeBridge();

class EcosystemManager {
    constructor() {
        // API configuration - resolve via shared api-base utilities
        this.apiBase = API_BASE;
        this.wsBase = WS_BASE;

        // Initialize API client
        this.api = new ApiClient(this.apiBase);

        // Initialize modules
        this.taskManager = new TaskManager(
            this.apiBase,
            this.showToast.bind(this),
            this.showLoading.bind(this)
        );
        this.settingsManager = new SettingsManager(this.apiBase, this.showToast.bind(this));
        this.autoSteerManager = new AutoSteerManager(this.apiBase, this.showToast.bind(this));
        this.profilePerformanceManager = new ProfilePerformanceManager(this.apiBase, this.showToast.bind(this));
        this.processMonitor = new ProcessMonitor(this.apiBase, this.showToast.bind(this));
        this.webSocketHandler = new WebSocketHandler(
            this.wsBase,
            this.handleWebSocketMessage.bind(this)
        );
        this.dragDropHandler = new DragDropHandler(this.handleTaskDrop.bind(this));
        
        // State
        this.isLoading = false;
        this.rateLimitEndTime = null;
        this.rateLimitPaused = false;
        this.rateLimitTargetEnd = null;
        this.rateLimitNotificationSuppressed = false;
        this.rateLimitOverlayVisible = false;
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
            'completed-finalized': [],
            failed: [],
            'failed-blocked': [],
            archived: []
        };
        this.defaultColumnVisibility = {
            pending: true,
            'in-progress': true,
            completed: true,
            'completed-finalized': true,
            failed: true,
            'failed-blocked': true,
            archived: false
        };
        this.columnVisibility = { ...this.defaultColumnVisibility };
        this.columnToggleButtons = new Map();
        this.pendingTargetRefresh = null;
        this.filterState = {
            type: '',
            operation: '',
            priority: '',
            search: ''
        };
        this.filterQueryParamMap = {
            type: 'filterType',
            operation: 'filterOperation',
            priority: 'filterPriority',
            search: 'filterSearch'
        };

        this.recyclerPromptDirty = false;
        this.recyclerPromptRefreshTimer = null;
        this.recyclerPromptLoading = false;

        this.recyclerTestPresets = RECYCLER_TEST_PRESETS;
        this.recyclerSuiteResults = [];
        this.recyclerSuiteRunning = false;
        this.recyclerSuiteCancelRequested = false;
        this.recyclerTestMode = 'custom';

        // Bind methods
        this.init = this.init.bind(this);
        this.refreshAll = this.refreshAll.bind(this);
        this.handlePopState = this.handlePopState.bind(this);
        this.toggleFilterPanel = this.toggleFilterPanel.bind(this);
        this.handleFilterKeyDown = this.handleFilterKeyDown.bind(this);
        this.updateActionButtonLabelsForViewport = this.updateActionButtonLabelsForViewport.bind(this);
        this.handleGlobalHotkeys = this.handleGlobalHotkeys.bind(this);

        this.horizontalScrollLockUntil = 0;
        this.isFilterDialogOpen = false;
        this.filterPanelHideTimeout = null;
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
        this.initializeFilterControls();
        this.updateFilterSummaryUI(this.filterState);
        this.initializeRecyclerTestbed();

        // Initialize tabs if they exist
        const tabButtons = document.querySelectorAll('.tab-button');
        tabButtons.forEach(button => {
            button.addEventListener('click', (e) => this.switchTab(e.target.dataset.tab));
        });
        
        // Set active tab
        if (tabButtons.length > 0) {
            this.switchTab('tasks');
        }

        this.updateActionButtonLabelsForViewport();
    }

    updateActionButtonLabelsForViewport() {
        const isMobileViewport = window.matchMedia && window.matchMedia('(max-width: 640px)').matches;
        const labels = document.querySelectorAll('.form-actions .btn .btn-label[data-label-desktop]');

        labels.forEach(label => {
            const desktopLabel = label.dataset.labelDesktop;
            if (!desktopLabel) {
                return;
            }

            const mobileLabel = label.dataset.labelMobile || desktopLabel;
            const desiredText = isMobileViewport ? mobileLabel : desktopLabel;

            if (label.textContent !== desiredText) {
                label.textContent = desiredText;
            }
        });
    }

    initializeFilterControls() {
        const filtersSection = document.getElementById('filters');
        const filterPanel = document.getElementById('filter-panel');
        const filterBackdrop = document.getElementById('filters-backdrop');

        const stateFromUrl = this.getFiltersFromUrl();
        this.applyFilterStateToUI(stateFromUrl);
        this.filterState = this.collectFilterStateFromUI();
        this.filterTasks(this.filterState);
        window.addEventListener('popstate', this.handlePopState);

        this.initializeColumnVisibilityControls();

        if (filtersSection && filterPanel && !filtersSection.classList.contains('filters-open')) {
            filtersSection.hidden = true;
            filterPanel.hidden = true;
            filterPanel.setAttribute('inert', '');
            filtersSection.style.pointerEvents = 'none';
            filterPanel.style.pointerEvents = 'none';

            if (filterBackdrop) {
                filterBackdrop.hidden = true;
            }
        }

        const filterCloseBtn = document.getElementById('filter-close-btn');
        if (filterCloseBtn) {
            filterCloseBtn.addEventListener('click', () => this.toggleFilterPanel(false));
        }
        if (filterBackdrop) {
            filterBackdrop.addEventListener('click', () => this.toggleFilterPanel(false));
        }
    }

    initializeColumnVisibilityControls() {
        const container = document.getElementById('column-toggle-container');
        if (!container) {
            return;
        }

        const chips = Array.from(container.querySelectorAll('.column-toggle-chip'));
        if (chips.length === 0) {
            return;
        }

        chips.forEach(chip => {
            const status = chip.dataset.column;
            if (!status) {
                return;
            }

            this.columnToggleButtons.set(status, chip);

            if (typeof this.columnVisibility[status] !== 'boolean') {
                this.columnVisibility[status] = this.defaultColumnVisibility.hasOwnProperty(status)
                    ? this.defaultColumnVisibility[status]
                    : status !== 'archived';
            }

            const isVisible = Boolean(this.columnVisibility[status]);
            chip.setAttribute('aria-pressed', isVisible ? 'true' : 'false');

            chip.addEventListener('click', () => {
                const currentlyPressed = chip.getAttribute('aria-pressed') === 'true';
                this.setColumnVisibility(status, !currentlyPressed, { source: 'chip' });
            });
        });

        Object.entries(this.columnVisibility).forEach(([status, visible]) => {
            this.setColumnVisibility(status, Boolean(visible), { source: 'init', suppressToast: true, force: true });
        });
    }

    getFiltersFromUrl() {
        const params = new URLSearchParams(window.location.search || '');
        return {
            type: params.get(this.filterQueryParamMap.type) || '',
            operation: params.get(this.filterQueryParamMap.operation) || '',
            priority: params.get(this.filterQueryParamMap.priority) || '',
            search: params.get(this.filterQueryParamMap.search) || ''
        };
    }

    applyFilterStateToUI(state = {}) {
        const typeSelect = document.getElementById('filter-type');
        const operationSelect = document.getElementById('filter-operation');
        const prioritySelect = document.getElementById('filter-priority');
        const searchInput = document.getElementById('search-input');

        if (typeSelect && typeof state.type === 'string') {
            typeSelect.value = state.type;
        }

        if (operationSelect && typeof state.operation === 'string') {
            operationSelect.value = state.operation;
        }

        if (prioritySelect && typeof state.priority === 'string') {
            prioritySelect.value = state.priority;
        }

        if (searchInput && typeof state.search === 'string') {
            searchInput.value = state.search;
        }

        this.updateFilterSummaryUI(state);
    }

    collectFilterStateFromUI() {
        return {
            type: document.getElementById('filter-type')?.value || '',
            operation: document.getElementById('filter-operation')?.value || '',
            priority: document.getElementById('filter-priority')?.value || '',
            search: document.getElementById('search-input')?.value?.trim() || ''
        };
    }

    updateFilterSummaryUI(state = this.filterState || {}) {
        const toggleBtn = document.getElementById('filter-toggle-btn');
        const badge = document.getElementById('filter-active-count');

        const activeCount = ['type', 'operation', 'priority']
            .map(key => (state[key] || '').trim())
            .filter(Boolean)
            .length;

        if (badge) {
            if (activeCount > 0) {
                badge.textContent = String(activeCount);
                badge.hidden = false;
            } else {
                badge.hidden = true;
            }
        }

        if (toggleBtn) {
            const baseLabel = 'Filters';
            if (activeCount > 0) {
                const suffix = activeCount === 1 ? 'filter' : 'filters';
                toggleBtn.setAttribute('aria-label', `${baseLabel}, ${activeCount} ${suffix} active`);
                toggleBtn.title = `${activeCount} ${suffix} active`;
            } else {
                toggleBtn.setAttribute('aria-label', baseLabel);
                toggleBtn.title = baseLabel;
            }

            toggleBtn.classList.toggle('is-active', activeCount > 0);
        }
    }

    toggleFilterPanel(forceState) {
        const filtersSection = document.getElementById('filters');
        const filterPanel = document.getElementById('filter-panel');
        const toggleBtn = document.getElementById('filter-toggle-btn');
        const backdrop = document.getElementById('filters-backdrop');

        if (!filtersSection || !filterPanel || !toggleBtn) {
            return;
        }

        const isOpen = filtersSection.classList.contains('filters-open');
        const shouldOpen = typeof forceState === 'boolean' ? forceState : !isOpen;

        if (shouldOpen === isOpen) {
            return;
        }

        window.clearTimeout(this.filterPanelHideTimeout);

        if (shouldOpen) {
            filtersSection.hidden = false;
            filterPanel.hidden = false;
            filterPanel.removeAttribute('inert');
            filtersSection.style.pointerEvents = '';
            filterPanel.style.pointerEvents = '';

            if (backdrop) {
                backdrop.hidden = false;
            }
        } else {
            filterPanel.setAttribute('inert', '');
            filtersSection.style.pointerEvents = 'none';
            filterPanel.style.pointerEvents = 'none';
        }

        filtersSection.classList.toggle('filters-open', shouldOpen);
        filtersSection.setAttribute('aria-hidden', (!shouldOpen).toString());
        filterPanel.setAttribute('aria-hidden', (!shouldOpen).toString());
        toggleBtn.setAttribute('aria-expanded', shouldOpen.toString());

        if (shouldOpen) {
            filterPanel.setAttribute('role', 'dialog');
            filterPanel.setAttribute('aria-modal', 'true');
            filterPanel.setAttribute('aria-labelledby', 'filter-dialog-title');
            document.body.style.overflow = 'hidden';
            this.isFilterDialogOpen = true;
            document.addEventListener('keydown', this.handleFilterKeyDown);

            const focusable = filterPanel.querySelector('input:not([type="hidden"]), select, textarea, button:not([type="hidden"]):not([disabled]), [tabindex]:not([tabindex="-1"])');
            if (focusable && typeof focusable.focus === 'function') {
                focusable.focus({ preventScroll: true });
            } else {
                filterPanel.focus({ preventScroll: true });
            }
        } else {
            filterPanel.removeAttribute('aria-modal');
            document.removeEventListener('keydown', this.handleFilterKeyDown);
            this.isFilterDialogOpen = false;
            this.restoreBodyScrollIfSafe();

            const finalizeClose = () => {
                if (backdrop) {
                    backdrop.hidden = true;
                }
                filtersSection.hidden = true;
                filterPanel.hidden = true;
                filterPanel.setAttribute('inert', '');
                filtersSection.style.pointerEvents = 'none';
                filterPanel.style.pointerEvents = 'none';
                this.filterPanelHideTimeout = null;
            };

            this.filterPanelHideTimeout = window.setTimeout(finalizeClose, 220);

            if (document.activeElement && filterPanel.contains(document.activeElement)) {
                toggleBtn.focus({ preventScroll: true });
            }
        }
    }

    hasFilterStateChanged(newState = {}) {
        return ['type', 'operation', 'priority', 'search'].some(
            key => (newState[key] || '') !== (this.filterState?.[key] || '')
        );
    }

    isMobileViewport() {
        if (window.matchMedia) {
            return window.matchMedia('(max-width: 768px)').matches;
        }
        return window.innerWidth <= 768;
    }

    handleGlobalHotkeys(event) {
        if (event.defaultPrevented) {
            return;
        }

        if (event.key === '/' && !event.altKey && !event.ctrlKey && !event.metaKey && !event.shiftKey) {
            const activeElement = document.activeElement;
            const isTyping = activeElement && (
                activeElement.tagName === 'INPUT' ||
                activeElement.tagName === 'TEXTAREA' ||
                activeElement.isContentEditable
            );

            if (isTyping) {
                return;
            }

            event.preventDefault();
            this.toggleFilterPanel(true);

            requestAnimationFrame(() => {
                const searchInput = document.getElementById('search-input');
                if (searchInput) {
                    searchInput.focus({ preventScroll: true });
                    if (typeof searchInput.select === 'function') {
                        searchInput.select();
                    }
                }
            });
        }
    }

    handleFilterKeyDown(event) {
        if (!this.isFilterDialogOpen) {
            return;
        }

        if (event.key === 'Escape') {
            event.preventDefault();
            this.toggleFilterPanel(false);
            return;
        }

        if (event.key === 'Tab') {
            const filterPanel = document.getElementById('filter-panel');
            if (!filterPanel) {
                return;
            }

            const focusableSelectors = 'button:not([disabled]):not([tabindex="-1"]), select:not([disabled]), textarea:not([disabled]), input:not([disabled]):not([type="hidden"]), [tabindex]:not([tabindex="-1"])';
            const focusable = Array.from(filterPanel.querySelectorAll(focusableSelectors))
                .filter(el => !el.hasAttribute('hidden') && el.offsetParent !== null);

            if (focusable.length === 0) {
                return;
            }

            const first = focusable[0];
            const last = focusable[focusable.length - 1];
            const active = document.activeElement;

            if (event.shiftKey && active === first) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && active === last) {
                event.preventDefault();
                first.focus();
            }
        }
    }

    restoreBodyScrollIfSafe() {
        const openModal = document.querySelector('.modal.show');
        if (!openModal && !this.isFilterDialogOpen) {
            document.body.style.overflow = '';
        }
    }

    updateUrlWithFilters(filterState = {}) {
        const params = new URLSearchParams(window.location.search || '');

        Object.entries(this.filterQueryParamMap).forEach(([stateKey, paramKey]) => {
            const value = filterState[stateKey] || '';
            if (value) {
                params.set(paramKey, value);
            } else {
                params.delete(paramKey);
            }
        });

        const queryString = params.toString();
        const newUrl = `${window.location.pathname}${queryString ? `?${queryString}` : ''}${window.location.hash || ''}`;
        const currentUrl = `${window.location.pathname}${window.location.search}${window.location.hash || ''}`;

        if (newUrl !== currentUrl) {
            window.history.replaceState({ filters: filterState }, '', newUrl);
        }
    }

    handlePopState() {
        const stateFromUrl = this.getFiltersFromUrl();
        this.applyFilterStateToUI(stateFromUrl);
        this.filterState = this.collectFilterStateFromUI();
        this.filterTasks(this.filterState);
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
                    this.restoreBodyScrollIfSafe();
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
        window.addEventListener('resize', this.updateActionButtonLabelsForViewport);
        window.addEventListener('keydown', this.handleGlobalHotkeys);

        const kanbanBoard = document.querySelector('.kanban-board');
        if (kanbanBoard) {
            kanbanBoard.addEventListener('wheel', (event) => {
                if (event.defaultPrevented) {
                    return;
                }

                const prefersCoarse = window.matchMedia ? window.matchMedia('(pointer: coarse)').matches : false;
                if (prefersCoarse) {
                    return;
                }

                const now = Date.now();
                const horizontalLockActive = now < this.horizontalScrollLockUntil;

                const columnContent = event.target.closest('.column-content');
                if (columnContent && !horizontalLockActive) {
                    const canScrollVertically = columnContent.scrollHeight > columnContent.clientHeight;
                    if (canScrollVertically) {
                        const scrollTop = columnContent.scrollTop;
                        const atTop = scrollTop <= 0;
                        const atBottom = (columnContent.scrollHeight - columnContent.clientHeight - scrollTop) <= 1;

                        if ((event.deltaY < 0 && !atTop) || (event.deltaY > 0 && !atBottom)) {
                            return; // allow default vertical scrolling within the column
                        }
                    }
                }

                if (Math.abs(event.deltaY) > Math.abs(event.deltaX)) {
                    kanbanBoard.scrollLeft += event.deltaY;
                    this.horizontalScrollLockUntil = Date.now() + 250;
                    event.preventDefault();
                }
            }, { passive: false });
        }

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
        
        const settingsMobileSelect = document.getElementById('settings-mobile-tab');
        if (settingsMobileSelect) {
            settingsMobileSelect.addEventListener('change', (event) => {
                const tabName = (event.target.value || '').trim();
                if (tabName) {
                    this.switchSettingsTab(tabName);
                }
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

        const recyclerProviderSelect = document.getElementById('settings-recycler-model-provider');
        if (recyclerProviderSelect) {
            recyclerProviderSelect.addEventListener('change', (event) => {
                const provider = event.target.value;
                this.settingsManager.handleRecyclerProviderChange(provider).catch(err => {
                    console.error('Failed to refresh recycler models:', err);
                    this.showToast(`Failed to refresh ${provider} models: ${err.message}`, 'error');
                });
            });
        }

        const recyclerModelSelect = document.getElementById('settings-recycler-model-name');
        if (recyclerModelSelect) {
            recyclerModelSelect.addEventListener('change', (event) => {
                this.settingsManager.handleRecyclerModelSelection(event.target.value);
            });
        }

        const recyclerPresetSelect = document.getElementById('recycler-test-preset');
        if (recyclerPresetSelect) {
            recyclerPresetSelect.addEventListener('change', () => {
                this.updateRecyclerPresetPreview();
                this.applyRecyclerPresetToForm(this.getRecyclerPreset(recyclerPresetSelect.value));
            });
        }

        const recyclerModeButtons = document.querySelectorAll('[data-recycler-mode]');
        if (recyclerModeButtons.length) {
            recyclerModeButtons.forEach(button => {
                button.addEventListener('click', (event) => {
                    event.preventDefault();
                    const mode = button.getAttribute('data-recycler-mode') || 'custom';
                    this.setRecyclerTestMode(mode);
                });
            });
        }

        const recyclerRunSuiteBtn = document.getElementById('recycler-run-suite');
        if (recyclerRunSuiteBtn) {
            recyclerRunSuiteBtn.addEventListener('click', () => this.runRecyclerPresetSuite());
        }

        const recyclerClearSuiteBtn = document.getElementById('recycler-clear-suite');
        if (recyclerClearSuiteBtn) {
            recyclerClearSuiteBtn.addEventListener('click', () => this.clearRecyclerSuiteResults());
        }

        const recyclerTestBtn = document.getElementById('recycler-test-run');
        if (recyclerTestBtn) {
            recyclerTestBtn.addEventListener('click', () => this.runRecyclerTest());
        }

        const recyclerOutputField = document.getElementById('recycler-test-output');
        if (recyclerOutputField) {
            recyclerOutputField.addEventListener('input', () => this.handleRecyclerOutputChange());
        }

        const recyclerPromptField = document.getElementById('recycler-test-prompt');
        if (recyclerPromptField) {
            recyclerPromptField.addEventListener('input', () => this.handleRecyclerPromptInput());
        }

        const recyclerPromptRefreshBtn = document.getElementById('recycler-prompt-refresh');
        if (recyclerPromptRefreshBtn) {
            recyclerPromptRefreshBtn.addEventListener('click', () => {
                this.reloadRecyclerPrompt({ force: true, showToast: true });
            });
        }
    }

    // Recycler Testbed Helpers
    initializeRecyclerTestbed() {
        const presetSelect = document.getElementById('recycler-test-preset');
        if (!presetSelect) {
            return;
        }

        presetSelect.innerHTML = '<option value="">Select a preset transcript…</option>';
        this.recyclerTestPresets.forEach(preset => {
            const option = document.createElement('option');
            option.value = preset.id;
            const classification = this.formatClassification(preset.expected);
            option.textContent = `${preset.label} (${classification})`;
            presetSelect.appendChild(option);
        });

        this.updateRecyclerPresetPreview();
        this.setRecyclerTestMode(this.recyclerTestMode);

        this.reloadRecyclerPrompt({ force: true }).catch(err => {
            console.error('Failed to load initial recycler prompt:', err);
        });
    }

    handleRecyclerOutputChange() {
        const outputField = document.getElementById('recycler-test-output');
        const promptField = document.getElementById('recycler-test-prompt');
        if (!outputField || !promptField) {
            return;
        }

        const rawOutput = outputField.value || '';

        if (rawOutput.trim() === '') {
            if (this.recyclerPromptRefreshTimer) {
                clearTimeout(this.recyclerPromptRefreshTimer);
                this.recyclerPromptRefreshTimer = null;
            }
            promptField.value = '';
            this.recyclerPromptDirty = false;
            return;
        }

        if (this.recyclerPromptDirty) {
            return;
        }

        if (this.recyclerPromptRefreshTimer) {
            clearTimeout(this.recyclerPromptRefreshTimer);
        }

        this.recyclerPromptRefreshTimer = setTimeout(() => {
            this.refreshRecyclerPromptFromOutput().catch(err => {
                console.error('Failed to refresh recycler prompt:', err);
            });
        }, 450);
    }

    handleRecyclerPromptInput() {
        this.recyclerPromptDirty = true;
        if (this.recyclerPromptRefreshTimer) {
            clearTimeout(this.recyclerPromptRefreshTimer);
            this.recyclerPromptRefreshTimer = null;
        }
    }

    async reloadRecyclerPrompt(options = {}) {
        const { force = false, showToast = false } = options;
        if (force) {
            this.recyclerPromptDirty = false;
        }
        if (this.recyclerPromptRefreshTimer) {
            clearTimeout(this.recyclerPromptRefreshTimer);
            this.recyclerPromptRefreshTimer = null;
        }

        return this.refreshRecyclerPromptFromOutput({ force, showToast });
    }

    async refreshRecyclerPromptFromOutput(options = {}) {
        const { force = false, showToast = false } = options;
        const outputField = document.getElementById('recycler-test-output');
        const promptField = document.getElementById('recycler-test-prompt');
        if (!outputField || !promptField) {
            return;
        }

        if (this.recyclerPromptRefreshTimer) {
            clearTimeout(this.recyclerPromptRefreshTimer);
            this.recyclerPromptRefreshTimer = null;
        }

        const outputText = outputField.value || '';
        const trimmed = outputText.trim();

        if (trimmed === '') {
            promptField.value = '';
            this.recyclerPromptDirty = false;
            if (showToast) {
                this.showToast('Enter mock output to build a prompt.', 'info');
            }
            return;
        }

        if (!force && this.recyclerPromptDirty) {
            return;
        }

        if (this.recyclerPromptLoading) {
            return;
        }

        this.recyclerPromptLoading = true;
        try {
            const prompt = await this.fetchRecyclerPrompt(outputText);
            this.applyPromptToTextarea(prompt);
            if (showToast) {
                this.showToast('Default recycler prompt loaded.', 'success');
            }
        } catch (error) {
            console.error('Failed to load recycler prompt:', error);
            this.showToast(`Failed to load default prompt: ${error.message}`, 'error');
        } finally {
            this.recyclerPromptLoading = false;
        }
    }

    async fetchRecyclerPrompt(outputText) {
        const response = await fetch(`${this.apiBase}/recycler/preview-prompt`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ output_text: outputText })
        });

        let data = {};
        try {
            data = await response.json();
        } catch (error) {
            console.warn('Failed to parse recycler prompt response:', error);
        }

        if (!response.ok) {
            const message = data?.error || response.statusText || 'Failed to load prompt';
            throw new Error(message);
        }

        const prompt = typeof data?.prompt === 'string' ? data.prompt : '';
        if (prompt.trim() === '') {
            throw new Error('Preview response missing prompt content');
        }

        return prompt;
    }

    applyPromptToTextarea(prompt, options = {}) {
        const promptField = document.getElementById('recycler-test-prompt');
        if (!promptField) {
            return;
        }

        promptField.value = prompt;
        const { markDirty = false } = options;
        this.recyclerPromptDirty = Boolean(markDirty);
    }

    setRecyclerTestMode(mode) {
        const normalized = mode === 'suite' ? 'suite' : 'custom';
        this.recyclerTestMode = normalized;

        const buttons = document.querySelectorAll('[data-recycler-mode]');
        buttons.forEach(button => {
            const buttonMode = button.getAttribute('data-recycler-mode');
            const isActive = buttonMode === normalized;
            button.setAttribute('aria-pressed', isActive ? 'true' : 'false');
            button.classList.toggle('btn-primary', isActive);
            button.classList.toggle('btn-outline', !isActive);
        });

        const sections = document.querySelectorAll('.recycler-mode-section');
        sections.forEach(section => {
            const sectionMode = section.getAttribute('data-mode');
            if (sectionMode === normalized) {
                section.removeAttribute('hidden');
            } else {
                section.setAttribute('hidden', 'hidden');
            }
        });

        this.renderRecyclerSuiteResults();
    }

    updateRecyclerPresetPreview() {
        const presetSelect = document.getElementById('recycler-test-preset');
        const expectedLabel = document.getElementById('recycler-preset-expected');
        if (!expectedLabel) {
            return;
        }

        const presetId = presetSelect?.value || '';
        const preset = this.getRecyclerPreset(presetId);
        expectedLabel.textContent = preset ? this.formatClassification(preset.expected) : '—';
    }

    getRecyclerPreset(presetId) {
        if (!presetId) {
            return null;
        }
        return this.recyclerTestPresets.find(preset => preset.id === presetId) || null;
    }

    applyRecyclerPresetToForm(preset) {
        if (!preset) {
            return;
        }

        const outputField = document.getElementById('recycler-test-output');
        if (outputField) {
            outputField.value = preset.payload?.output_text ?? '';
        }

        this.reloadRecyclerPrompt({ force: true }).catch(err => {
            console.error('Failed to reload recycler prompt for preset:', err);
        });
    }

    async runRecyclerPresetSuite() {
        if (this.recyclerSuiteRunning) {
            if (!this.recyclerSuiteCancelRequested) {
                this.recyclerSuiteCancelRequested = true;
                this.updateRecyclerSuiteControls();
                this.updateRecyclerSuiteSummary();
                this.renderRecyclerSuiteResults();
                this.showToast('Canceling preset suite after the current run completes.', 'info');
            }
            return;
        }

        if (!this.recyclerTestPresets.length) {
            this.showToast('No recycler presets configured.', 'warning');
            return;
        }

        this.recyclerSuiteResults = [];
        this.recyclerSuiteCancelRequested = false;
        this.setRecyclerTestMode('suite');
        this.setRecyclerSuiteRunning(true);

        let cancelled = false;
        try {
            for (const preset of this.recyclerTestPresets) {
                if (this.recyclerSuiteCancelRequested) {
                    cancelled = true;
                    break;
                }

                await this.runRecyclerTestWithPreset(preset, {
                    silent: true,
                    showLoading: false,
                });

                if (this.recyclerSuiteCancelRequested) {
                    cancelled = true;
                    break;
                }
            }
            if (cancelled) {
                this.showToast('Preset suite cancelled.', 'warning');
            } else {
                this.showToast('Preset suite completed.', 'success');
            }
        } catch (error) {
            console.error('Recycler preset suite failed:', error);
            this.showToast(`Preset suite failed: ${error.message}`, 'error');
        } finally {
            this.recyclerSuiteCancelRequested = false;
            this.setRecyclerSuiteRunning(false);
        }
    }

    setRecyclerSuiteRunning(isRunning) {
        this.recyclerSuiteRunning = isRunning;

        this.updateRecyclerSuiteControls();

        const summary = document.getElementById('recycler-suite-summary');
        if (summary) {
            summary.textContent = isRunning ? `Running… 0/${this.recyclerTestPresets.length} processed` : summary.textContent;
        }

        this.renderRecyclerSuiteResults();
    }

    updateRecyclerSuiteControls() {
        const isRunning = this.recyclerSuiteRunning;
        const isCancelling = this.recyclerSuiteCancelRequested;

        const runButton = document.getElementById('recycler-run-suite');
        if (runButton) {
            runButton.disabled = false;
            runButton.textContent = isRunning
                ? (isCancelling ? 'Canceling…' : 'Cancel Preset Suite')
                : 'Run All Presets';
        }

        const clearButton = document.getElementById('recycler-clear-suite');
        if (clearButton) {
            clearButton.disabled = isRunning;
        }

        const singleRunButton = document.getElementById('recycler-test-run');
        if (singleRunButton) {
            singleRunButton.disabled = isRunning;
        }
    }

    clearRecyclerSuiteResults() {
        if (!this.recyclerSuiteResults.length) {
            return;
        }
        this.recyclerSuiteResults = [];
        this.renderRecyclerSuiteResults();
        this.showToast('Cleared recycler preset history.', 'info');
    }

    renderRecyclerSuiteResults() {
        const container = document.getElementById('recycler-test-suite-results');
        const tableBody = document.getElementById('recycler-suite-results-body');
        if (!container || !tableBody) {
            return;
        }

        const isSuiteMode = this.recyclerTestMode === 'suite';
        const isRunning = this.recyclerSuiteRunning;

        if (!isSuiteMode && !isRunning) {
            container.style.display = 'none';
            tableBody.innerHTML = '';
            this.updateRecyclerSuiteSummary();
            return;
        }

        if (this.recyclerSuiteResults.length === 0) {
            if (isRunning) {
                container.style.display = 'block';
                tableBody.innerHTML = this.renderRecyclerLoadingRow(0);
                this.updateRecyclerSuiteSummary();
            } else {
                container.style.display = 'none';
                tableBody.innerHTML = '';
                this.updateRecyclerSuiteSummary();
            }
            return;
        }

        tableBody.innerHTML = this.recyclerSuiteResults.map(result => {
            const matchClass = result.match ? 'match' : 'mismatch';
            const expectedLabel = this.escapeHtml(this.formatClassification(result.expected));
            const actualLabel = this.escapeHtml(this.formatClassification(result.actual));
            const noteSnippet = result.note
                ? this.escapeHtml(this.truncateText(result.note, 180)).replace(/\n/g, '<br>')
                : '<em>No note returned</em>';
            const providerInfo = [result.provider, result.model].filter(Boolean).join(' · ') || '—';
            const outcomeChips = [
                `<span class="status-chip ${matchClass}">${result.match ? 'match' : 'mismatch'}</span>`,
                `<span class="status-chip ${result.success ? 'match' : 'fallback'}">${result.success ? 'llm' : 'fallback'}</span>`
            ];
            return `
                <tr class="${matchClass}">
                    <td>
                        <strong>${this.escapeHtml(result.label || result.id)}</strong>
                        <span class="preset-id">${this.escapeHtml(result.id)}</span>
                    </td>
                    <td><span class="status-chip ${matchClass}">${expectedLabel}</span></td>
                    <td><span class="status-chip ${matchClass}">${actualLabel}</span></td>
                    <td>
                        <div class="suite-chip-row">${outcomeChips.join(' ')}</div>
                        <div class="suite-provider">${this.escapeHtml(providerInfo)}</div>
                        ${result.error ? `<div class="suite-error">${this.escapeHtml(result.error)}</div>` : ''}
                    </td>
                    <td><div class="note-snippet">${noteSnippet}</div></td>
                </tr>
            `;
        }).join('');

        if (isRunning) {
            tableBody.innerHTML += this.renderRecyclerLoadingRow(this.recyclerSuiteResults.length);
        }

        container.style.display = isSuiteMode || isRunning ? 'block' : 'none';
        this.updateRecyclerSuiteSummary();
    }

    renderRecyclerLoadingRow(processedCount) {
        const total = this.recyclerTestPresets.length;
        const progress = `${processedCount}/${total}`;
        const isCancelling = this.recyclerSuiteCancelRequested;
        return `
            <tr class="pending">
                <td colspan="5">
                    <div class="suite-loading-row">
                        <span class="loading-spinner-icon"><i class="fas fa-spinner fa-spin"></i></span>
                        <span>${isCancelling ? 'Canceling preset suite…' : 'Running preset suite…'} ${progress} processed</span>
                    </div>
                </td>
            </tr>
        `;
    }

    updateRecyclerSuiteSummary() {
        const summary = document.getElementById('recycler-suite-summary');
        if (!summary) {
            return;
        }

        const isRunning = this.recyclerSuiteRunning;
        const isCancelling = this.recyclerSuiteCancelRequested;
        const totalPresets = this.recyclerTestPresets.length;

        if (this.recyclerSuiteResults.length === 0) {
            summary.textContent = isRunning
                ? `${isCancelling ? 'Canceling…' : 'Running…'} 0/${totalPresets} processed`
                : '';
            return;
        }

        const matches = this.recyclerSuiteResults.filter(result => result.match).length;
        const processed = this.recyclerSuiteResults.length;
        if (isRunning) {
            summary.textContent = `${isCancelling ? 'Canceling…' : 'Running…'} ${processed}/${totalPresets} processed (${matches} match)`;
        } else {
            summary.textContent = `${matches}/${processed} matches`;
        }
    }

    async runRecyclerTestWithPreset(preset, options = {}) {
        if (!preset) {
            return null;
        }

        const { silent = false, showLoading = true } = options;
        const payload = { ...preset.payload };
        Object.assign(payload, this.getRecyclerModelOverrides());

        this.resetRecyclerResultCard();
        if (!silent) {
            this.showRecyclerResultLoading();
        }

        const globalLoading = silent ? showLoading : false;

        try {
            const data = await this.executeRecyclerTest(payload, { showLoading: globalLoading });
            if (!silent && data?.prompt) {
                this.applyPromptToTextarea(data.prompt);
            }
            const expectedRaw = (preset.expected || '').toLowerCase();
            const actualRaw = (data?.result?.classification || 'unknown').toLowerCase();
            const match = expectedRaw === actualRaw;

            this.renderRecyclerTestResult(data, {
                expectedClassification: expectedRaw,
                presetLabel: preset.label,
            });

            this.recyclerSuiteResults.push({
                id: preset.id,
                label: preset.label,
                expected: expectedRaw,
                actual: actualRaw,
                note: data?.result?.note || '',
                provider: data?.provider || '',
                model: data?.model || '',
                success: Boolean(data?.success),
                error: data?.error || '',
                match,
            });
            this.renderRecyclerSuiteResults();

            if (!silent) {
                const toastType = match ? 'success' : 'warning';
                const expectedLabel = this.formatClassification(expectedRaw);
                const actualLabel = this.formatClassification(actualRaw);
                const toastMessage = match
                    ? `${preset.label}: classification matched (${actualLabel})`
                    : `${preset.label}: expected ${expectedLabel} but got ${actualLabel}`;
                this.showToast(toastMessage, toastType);

                if (!data.success) {
                    this.showToast('Recycler summarizer fell back to default result', 'warning');
                }
            }

            return data;
        } catch (error) {
            this.showRecyclerTestError(error);
            throw error;
        }
    }

    // Task Management Methods
    async loadAllTasks() {
        const statuses = ['pending', 'in-progress', 'review', 'completed', 'completed-finalized', 'failed', 'failed-blocked', 'archived'];
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
            const friendlyStatus = status.replace(/-/g, ' ');
            const titleCaseStatus = friendlyStatus.charAt(0).toUpperCase() + friendlyStatus.slice(1);
            container.innerHTML = `<div class="empty-state">No ${titleCaseStatus} tasks</div>`;
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
                    return;
                }

                const archiveBtn = e.target.closest('.task-archive-btn');
                if (archiveBtn) {
                    e.stopPropagation();
                    const taskId = archiveBtn.dataset.taskId;
                    const taskStatus = archiveBtn.dataset.taskStatus || task.status;
                    this.archiveTask(taskId, taskStatus);
                    return;
                }

                const restoreBtn = e.target.closest('.task-restore-btn');
                if (restoreBtn) {
                    e.stopPropagation();
                    const taskId = restoreBtn.dataset.taskId;
                    this.restoreTask(taskId);
                    return;
                }

                this.showTaskDetails(task.id);
            });
            
            container.appendChild(card);
        });
        
        // Update counter
        const counter = document.querySelector(`[data-status="${status}"] .task-count`);
        if (counter) {
            counter.textContent = normalizedTasks.length;
        }

        this.filterTasks(this.filterState);
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
            await this.loadAutoSteerProfilesIntoSelect();
            modal.classList.add('show');
            // Disable body scroll when showing modal
            document.body.style.overflow = 'hidden';
        }
    }

    async loadAutoSteerProfilesIntoSelect() {
        try {
            const select = document.getElementById('task-autosteer-profile');
            if (!select) return;

            const profiles = await this.autoSteerManager.loadProfiles();

            // Clear existing options except the first one (None)
            while (select.options.length > 1) {
                select.remove(1);
            }

            // Add profile options
            profiles.forEach(profile => {
                const option = document.createElement('option');
                option.value = profile.id;
                option.textContent = profile.name;
                select.appendChild(option);
            });
        } catch (error) {
            console.error('Failed to load Auto Steer profiles:', error);
        }
    }

    handleAutoSteerProfileChange(profileId) {
        const previewContainer = document.getElementById('autosteer-profile-preview');
        if (!previewContainer) return;

        if (!profileId) {
            previewContainer.style.display = 'none';
            previewContainer.innerHTML = '';
            return;
        }

        const profile = this.autoSteerManager.profiles.find(p => p.id === profileId);
        if (!profile) {
            previewContainer.style.display = 'none';
            return;
        }

        previewContainer.style.display = 'block';
        previewContainer.innerHTML = this.renderAutoSteerProfilePreview(profile);
    }

    renderAutoSteerProfilePreview(profile) {
        const phasesCount = (profile.phases || profile.config?.phases || []).length;
        const phases = profile.phases || profile.config?.phases || [];

        const phaseIcons = phases.slice(0, 8).map(phase => {
            const icon = this.autoSteerManager.getModeIcon(phase.mode);
            const modeName = this.autoSteerManager.formatModeName(phase.mode);
            return `<span class="phase-timeline-item" title="${modeName} (max ${phase.max_iterations} iterations)">${icon}</span>`;
        }).join('');

        const morePhases = phasesCount > 8 ? `<span class="phase-timeline-more">+${phasesCount - 8}</span>` : '';

        return `
            <div class="autosteer-profile-preview-card">
                <div class="autosteer-profile-preview-header">
                    <h4><i class="fas fa-route"></i> ${this.escapeHtml(profile.name)}</h4>
                </div>
                <p class="autosteer-profile-preview-description">${this.escapeHtml(profile.description || '')}</p>

                <div class="autosteer-profile-preview-phases">
                    <div class="phase-timeline-label">
                        <strong>${phasesCount}</strong> phase${phasesCount !== 1 ? 's' : ''}
                    </div>
                    <div class="phase-timeline">
                        ${phaseIcons}${morePhases}
                    </div>
                </div>
            </div>
        `;
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

        // Include Auto Steer profile if selected
        const autoSteerProfileId = formData.get('auto_steer_profile_id');
        if (autoSteerProfileId && autoSteerProfileId !== '') {
            taskData.auto_steer_profile_id = autoSteerProfileId;
        }

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
        this.updateActionButtonLabelsForViewport();

        // Initialize custom interactions
        this.initializeAutoRequeueToggle(task);

        modal.classList.add('show');
        // Disable body scroll when showing modal
        document.body.style.overflow = 'hidden';
    }

    getTaskDetailsHTML(task) {
        const isRunning = this.processMonitor.isTaskRunning(task.id);
        const runningProcess = isRunning ? this.processMonitor.getRunningProcess(task.id) : null;
        
        return `
            <form id="edit-task-form" class="task-details-form">
                <div class="task-details-container">
                    <div class="task-details-grid">
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
                                        <option value="in-progress" ${task.status === 'in-progress' ? 'selected' : ''}>Active</option>
                                        <option value="completed" ${task.status === 'completed' ? 'selected' : ''}>Completed</option>
                                        <option value="completed-finalized" ${task.status === 'completed-finalized' ? 'selected' : ''}>Finished</option>
                                        <option value="failed" ${task.status === 'failed' ? 'selected' : ''}>Failed</option>
                                        <option value="failed-blocked" ${task.status === 'failed-blocked' ? 'selected' : ''}>Blocked</option>
                                        <option value="archived" ${task.status === 'archived' ? 'selected' : ''}>Archived</option>
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

                            <!-- Auto-Requeue Toggle -->
                            <div class="form-group">
                                <label id="auto-requeue-label">Queue Automation</label>
                                <div class="auto-requeue-toggle toggle-group" role="radiogroup" aria-labelledby="auto-requeue-label">
                                    <input type="radio" class="toggle-input" id="auto-requeue-on" name="auto_requeue_mode" value="true" ${task.processor_auto_requeue === false ? '' : 'checked'}>
                                    <label class="toggle-btn" for="auto-requeue-on">
                                        <i class="fas fa-play"></i>
                                        Auto
                                    </label>
                                    <input type="radio" class="toggle-input" id="auto-requeue-off" name="auto_requeue_mode" value="false" ${task.processor_auto_requeue === false ? 'checked' : ''}>
                                    <label class="toggle-btn" for="auto-requeue-off">
                                        <i class="fas fa-hand-paper"></i>
                                        Manual
                                    </label>
                                </div>
                                <div class="auto-requeue-hint">Keep this enabled so the queue processor can pick the task automatically. Disable it only when you want to hold the task for manual review.</div>
                            </div>

                            <!-- Current Phase and Operation Type -->
                            <div class="form-row">
                                <div class="form-group">
                                    <label for="edit-task-phase">Current Phase</label>
                                    <select id="edit-task-phase" name="current_phase">
                                        <option value="">No Phase</option>
                                        <option value="pending" ${task.current_phase === 'pending' ? 'selected' : ''}>Pending</option>
                                        <option value="in-progress" ${task.current_phase === 'in-progress' ? 'selected' : ''}>Active</option>
                                        <option value="completed" ${task.current_phase === 'completed' ? 'selected' : ''}>Completed</option>
                                        <option value="finalized" ${task.current_phase === 'finalized' ? 'selected' : ''}>Finalized</option>
                                        <option value="failed" ${task.current_phase === 'failed' ? 'selected' : ''}>Failed</option>
                                        <option value="blocked" ${task.current_phase === 'blocked' ? 'selected' : ''}>Blocked</option>
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
                            <div id="auto-requeue-alert" class="auto-requeue-alert ${task.processor_auto_requeue === false ? 'disabled' : 'enabled'}">
                                <i class="fas ${task.processor_auto_requeue === false ? 'fa-exclamation-triangle' : 'fa-check-circle'}"></i>
                                <div>
                                    <div class="alert-title">${task.processor_auto_requeue === false ? 'Auto requeue disabled' : 'Auto requeue enabled'}</div>
                                    <div class="alert-body">${task.processor_auto_requeue === false ? 'The queue processor will skip this task until you re-enable auto requeue or launch it manually.' : 'This task is eligible for automated execution whenever a processing slot opens.'}</div>
                                </div>
                            </div>

                            ${this.getTaskExecutionInfoHTML(task, isRunning, runningProcess)}
                        </div>
                    </div>
                </div>
                
                <!-- Action Buttons -->
                <div class="form-actions form-actions--flush">
                    <button type="button" class="btn btn-secondary" onclick="ecosystemManager.closeModal('task-details-modal')" aria-label="Cancel">
                        <i class="fas fa-times" aria-hidden="true"></i>
                        <span class="btn-label" data-label-desktop="Cancel" data-label-mobile="Cancel">Cancel</span>
                    </button>
                    
                    <button type="button" class="btn btn-info" onclick="ecosystemManager.viewTaskPrompt('${task.id}')" title="View the prompt that was/will be sent to Claude" aria-label="View Prompt">
                        <i class="fas fa-file-alt" aria-hidden="true"></i>
                        <span class="btn-label" data-label-desktop="View Prompt" data-label-mobile="Prompt">View Prompt</span>
                    </button>
                    
                    <button type="button" class="btn btn-primary" onclick="ecosystemManager.saveTaskChanges('${task.id}')" aria-label="Save Changes">
                        <i class="fas fa-save" aria-hidden="true"></i>
                        <span class="btn-label" data-label-desktop="Save Changes" data-label-mobile="Save">Save Changes</span>
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
        if (task.results && ['completed', 'failed', 'completed-finalized', 'failed-blocked', 'archived'].includes(task.status)) {
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
                    ${Number.isInteger(task.consecutive_completion_claims) ? `<div><strong>Completion Streak:</strong> ${task.consecutive_completion_claims}</div>` : ''}
                    ${Number.isInteger(task.consecutive_failures) ? `<div><strong>Failure Streak:</strong> ${task.consecutive_failures}</div>` : ''}
                    ${typeof task.processor_auto_requeue === 'boolean' ? `<div><strong>Auto Requeue:</strong> ${task.processor_auto_requeue ? 'Enabled' : 'Disabled'}</div>` : ''}
                    ${task.last_completed_at ? `<div><strong>Last Completed:</strong> ${new Date(task.last_completed_at).toLocaleString()}</div>` : (!task.last_completed_at && task.completed_at ? `<div><strong>Last Completed:</strong> ${new Date(task.completed_at).toLocaleString()}</div>` : '')}
                </div>
            </div>
        `;
        
        return html;
    }

    initializeAutoRequeueToggle() {
        const alert = document.getElementById('auto-requeue-alert');
        const enableInput = document.getElementById('auto-requeue-on');
        const disableInput = document.getElementById('auto-requeue-off');

        if (!alert || !enableInput || !disableInput) {
            return;
        }

        const updateAlert = () => {
            const isEnabled = enableInput.checked;
            alert.classList.toggle('enabled', isEnabled);
            alert.classList.toggle('disabled', !isEnabled);

            const icon = alert.querySelector('i');
            if (icon) {
                icon.className = `fas ${isEnabled ? 'fa-check-circle' : 'fa-exclamation-triangle'}`;
            }

            const title = alert.querySelector('.alert-title');
            if (title) {
                title.textContent = isEnabled ? 'Auto requeue enabled' : 'Auto requeue disabled';
            }

            const body = alert.querySelector('.alert-body');
            if (body) {
                body.textContent = isEnabled
                    ? 'This task is eligible for automated execution whenever a processing slot opens.'
                    : 'The queue processor will skip this task until you re-enable auto requeue or launch it manually.';
            }
        };

        enableInput.addEventListener('change', updateAlert);
        disableInput.addEventListener('change', updateAlert);

        updateAlert();
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

                    ${results.recycler_classification ? `
                        <div style="margin-bottom: 0.5rem; font-size: 0.9em;">
                            <strong>Recycler Classification:</strong> ${this.escapeHtml(results.recycler_classification.replace(/_/g, ' '))}
                            ${results.recycler_updated_at ? `<span style="margin-left: 0.5rem; color: #666;">(updated ${new Date(results.recycler_updated_at).toLocaleString()})</span>` : ''}
                        </div>
                    ` : ''}
                    
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
            status: formData.get('status'),
            priority: formData.get('priority'),
            current_phase: formData.get('current_phase'),
            operation: formData.get('operation'),
            notes: formData.get('notes')
        };

        const autoRequeueInputs = form.querySelectorAll('input[name="auto_requeue_mode"]');
        let autoRequeueEnabled = true;
        autoRequeueInputs.forEach(input => {
            if (input.checked) {
                autoRequeueEnabled = input.value === 'true';
            }
        });
        updates.processor_auto_requeue = autoRequeueEnabled;

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

    async archiveTask(taskId, fromStatus) {
        this.showLoading(true);

        try {
            const result = await this.taskManager.updateTask(taskId, {
                status: 'archived',
                current_phase: 'archived',
                processor_auto_requeue: false
            });

            if (result.success) {
                this.showToast('Task archived', 'info');
                await Promise.all([
                    fromStatus && fromStatus !== 'archived' ? this.loadTasksForStatus(fromStatus) : Promise.resolve(),
                    this.loadTasksForStatus('archived')
                ]);
            } else {
                throw new Error(result.error || 'Failed to archive task');
            }
        } catch (error) {
            console.error('Error archiving task:', error);
            this.showToast(`Failed to archive task: ${error.message}`, 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async restoreTask(taskId) {
        if (!confirm('Restore this task to Pending?')) {
            return;
        }

        this.showLoading(true);

        try {
            const result = await this.taskManager.updateTask(taskId, {
                status: 'pending',
                current_phase: '',
                processor_auto_requeue: true
            });

            if (result.success) {
                this.showToast('Task restored to Pending', 'success');
                await Promise.all([
                    this.loadTasksForStatus('archived'),
                    this.loadTasksForStatus('pending')
                ]);
            } else {
                throw new Error(result.error || 'Failed to restore task');
            }
        } catch (error) {
            console.error('Error restoring task:', error);
            this.showToast(`Failed to restore task: ${error.message}`, 'error');
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

            const autoRequeueHoldStatuses = ['completed-finalized', 'failed-blocked', 'archived'];
            const targetDisablesAuto = autoRequeueHoldStatuses.includes(toStatus);
            const sourceDisablesAuto = autoRequeueHoldStatuses.includes(fromStatus);
            const autoRequeueChange = targetDisablesAuto ? false : (sourceDisablesAuto && !targetDisablesAuto ? true : null);

            // Set appropriate current_phase based on the target status
            // Use empty string to clear, as the backend preserves non-empty values
            switch (toStatus) {
                case 'pending':
                    updates.current_phase = '';
                    break;
                case 'in-progress':
                    updates.current_phase = 'in-progress';
                    break;
                case 'completed':
                    updates.current_phase = 'completed';
                    break;
                case 'completed-finalized':
                    updates.current_phase = 'finalized';
                    break;
                case 'failed':
                    updates.current_phase = 'failed';
                    break;
                case 'failed-blocked':
                    updates.current_phase = 'blocked';
                    break;
                case 'archived':
                    updates.current_phase = 'archived';
                    break;
                default:
                    updates.current_phase = '';
            }
            
            const result = await this.taskManager.updateTask(taskId, updates);
            
            if (result.success) {
                if (autoRequeueChange !== null) {
                    try {
                        await this.taskManager.updateTask(taskId, {
                            status: toStatus,
                            processor_auto_requeue: autoRequeueChange,
                        });
                    } catch (autoError) {
                        console.warn('Failed to update auto requeue state during drag-drop:', autoError);
                        this.showToast('Status moved, but auto requeue toggle could not be updated.', 'warning');
                    }
                }

                const friendlyStatus = toStatus.replace(/-/g, ' ');
                this.showToast(`Task moved to ${friendlyStatus}`, 'success');
                
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
                const resolvedTheme = settings.theme || 'dark';
                this.settingsManager.applyTheme(resolvedTheme, { markUserPreference: true });
                this.settingsManager.originalTheme = resolvedTheme; // Update original theme
                
                // Update processor status UI immediately
                this.settingsManager.updateProcessorToggleUI(settings.active);
                
                this.showToast('Settings saved successfully', 'success');
                this.closeModal('settings-modal');
                await this.fetchQueueProcessorStatus();
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
        if (!status || typeof status !== 'object') {
            return;
        }

        const lastProcessedElement = document.getElementById('last-processed-time');
        if (lastProcessedElement) {
            if (status.last_processed_at) {
                lastProcessedElement.textContent = new Date(status.last_processed_at).toLocaleString();
            } else {
                lastProcessedElement.textContent = '—';
            }
        }

        const processorToggle = document.getElementById('queue-processor-toggle');
        if (processorToggle && typeof status.settings_active === 'boolean') {
            processorToggle.checked = Boolean(status.settings_active);

            const pausedByRateLimit = Boolean(status.rate_limit_info && status.rate_limit_info.paused);
            processorToggle.disabled = pausedByRateLimit;
            processorToggle.title = pausedByRateLimit
                ? 'Processor paused while waiting for the provider rate limit to clear.'
                : '';
        }

        const rateLimitInfo = status.rate_limit_info || {};
        const rateLimitStatusElement = document.getElementById('rate-limit-status');
        const paused = Boolean(rateLimitInfo.paused);

        if (rateLimitStatusElement) {
            if (paused) {
                const remainingSeconds = this.getRateLimitRemainingSeconds(rateLimitInfo);
                const pauseUntilDate = rateLimitInfo.pause_until ? new Date(rateLimitInfo.pause_until) : null;
                const resumeLabel = pauseUntilDate
                    ? `${pauseUntilDate.toLocaleDateString()} ${pauseUntilDate.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}`.trim()
                    : 'as soon as capacity returns';

                rateLimitStatusElement.innerHTML = `
                    <div class="rate-limit-status-card">
                        <span class="rate-limit-status-title">
                            <i class="fas fa-hourglass-half"></i>
                            Provider rate limit active
                        </span>
                        <div class="rate-limit-status-body">
                            Queue resumes around <strong>${this.escapeHtml(resumeLabel)}</strong>
                            (${this.escapeHtml(this.formatSeconds(remainingSeconds))} remaining)
                        </div>
                        <div class="rate-limit-status-help">
                            Wait for the provider cooldown to expire or use “Reset Rate Limit Pause” once you have confirmed new capacity.
                        </div>
                    </div>
                `;
            } else {
                rateLimitStatusElement.innerHTML = `
                    <span class="rate-limit-status-clear">
                        <i class="fas fa-check-circle"></i>
                        No rate limit currently active
                    </span>
                `;
            }
        }

        this.handleRateLimitStateUpdate(paused, rateLimitInfo);

        // Update task counter display in settings and badge
        this.updateTaskCounterDisplay(status);
    }

    updateTaskCounterDisplay(status) {
        const tasksRemainingDisplay = document.getElementById('tasks-remaining-display');
        const processorRemainingBadge = document.getElementById('processor-remaining-count');

        if (!status) {
            return;
        }

        const maxTasksDisabled = Boolean(status.max_tasks_disabled);
        const maxTasks = status.max_tasks || 0;
        const tasksRemaining = status.tasks_remaining;

        // Update settings display
        if (tasksRemainingDisplay) {
            if (maxTasksDisabled) {
                tasksRemainingDisplay.textContent = '(disabled - processor runs continuously)';
            } else if (maxTasks > 0) {
                tasksRemainingDisplay.textContent = `(${tasksRemaining} remaining)`;
            } else {
                tasksRemainingDisplay.textContent = '';
            }
        }

        // Update notification badge
        if (processorRemainingBadge) {
            processorRemainingBadge.hidden = true;
            if (!maxTasksDisabled && maxTasks > 0 && typeof tasksRemaining === 'number') {
                processorRemainingBadge.textContent = tasksRemaining;
                processorRemainingBadge.hidden = false;
            }
        }
    }

    getRateLimitRemainingSeconds(info) {
        if (!info) {
            return 0;
        }

        const rawRemaining = Number(info.remaining_secs);
        if (Number.isFinite(rawRemaining) && rawRemaining >= 0) {
            return Math.round(rawRemaining);
        }

        if (info.pause_until) {
            const pauseUntil = new Date(info.pause_until).getTime();
            const diffMs = pauseUntil - Date.now();
            if (Number.isFinite(diffMs)) {
                return Math.max(0, Math.round(diffMs / 1000));
            }
        }

        return 0;
    }

    handleRateLimitStateUpdate(paused, info) {
        if (this.rateLimitOverlayVisible && !document.querySelector('.rate-limit-notification')) {
            this.rateLimitOverlayVisible = false;
        }

        if (paused) {
            const remainingSeconds = this.getRateLimitRemainingSeconds(info);
            const pauseUntilDate = info && info.pause_until ? new Date(info.pause_until) : null;
            const targetEnd = pauseUntilDate
                ? pauseUntilDate.getTime()
                : Date.now() + (Math.max(remainingSeconds, 1) * 1000);

            const isNewEvent = !this.rateLimitPaused
                || !this.rateLimitTargetEnd
                || Math.abs(this.rateLimitTargetEnd - targetEnd) > 2000;

            this.rateLimitPaused = true;
            this.rateLimitTargetEnd = targetEnd;
            this.rateLimitEndTime = targetEnd;

            if (isNewEvent) {
                this.rateLimitNotificationSuppressed = false;
            }

            const shouldShowOverlay = !this.rateLimitNotificationSuppressed
                && (!this.rateLimitOverlayVisible || isNewEvent);

            if (shouldShowOverlay) {
                this.handleRateLimit(Math.max(remainingSeconds, 1));
            }
        } else {
            this.rateLimitPaused = false;
            this.rateLimitTargetEnd = null;
            this.rateLimitEndTime = null;
            this.rateLimitNotificationSuppressed = false;

            if (this.rateLimitOverlayVisible) {
                this.clearRateLimitNotification();
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
                        return;
                    }

                    const archiveBtn = e.target.closest('.task-archive-btn');
                    if (archiveBtn) {
                        e.stopPropagation();
                        const taskId = archiveBtn.dataset.taskId;
                        const taskStatus = archiveBtn.dataset.taskStatus || task.status;
                        this.archiveTask(taskId, taskStatus);
                        return;
                    }

                    const restoreBtn = e.target.closest('.task-restore-btn');
                    if (restoreBtn) {
                        e.stopPropagation();
                        const taskId = restoreBtn.dataset.taskId;
                        this.restoreTask(taskId);
                        return;
                    }

                    this.showTaskDetails(task.id);
                });
                
                card.replaceWith(newCard);
                this.filterTasks(this.filterState);
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
            case 'task_finalized':
                this.showToast('Task moved to finalized lane', 'success');
                this.refreshColumn('completed-finalized').catch(console.error);
                this.refreshColumn('completed').catch(console.error);
                break;
            case 'task_blocked':
                this.showToast('Task marked as blocked after repeated failures', 'error');
                this.refreshColumn('failed-blocked').catch(console.error);
                this.refreshColumn('failed').catch(console.error);
                break;
            case 'task_recycled':
                this.refreshColumn('pending').catch(console.error);
                this.refreshColumn('completed').catch(console.error);
                this.refreshColumn('failed').catch(console.error);
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
        if (oldStatus === newStatus) {
            if (newStatus) {
                await this.loadTasksForStatus(newStatus);
            }
            return;
        }

        const statusesToRefresh = new Set();
        if (oldStatus) statusesToRefresh.add(oldStatus);
        if (newStatus) statusesToRefresh.add(newStatus);

        await Promise.all(
            Array.from(statusesToRefresh).map(status => this.loadTasksForStatus(status))
        );
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

        const mobileSelect = document.getElementById('settings-mobile-tab');
        if (mobileSelect && mobileSelect.value !== tabName) {
            mobileSelect.value = tabName;
        }
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
            this.restoreBodyScrollIfSafe();
        }
    }

    showToast(message, type = 'info') {
        UIComponents.showToast(message, type);
    }

    buildRecyclerPayloadFromForm(options = {}) {
        const { requireOutput = true } = options;
        const outputField = document.getElementById('recycler-test-output');
        const outputText = outputField?.value ?? '';
        const promptField = document.getElementById('recycler-test-prompt');
        const promptText = promptField?.value ?? '';

        const trimmedOutput = outputText.trim();

        if (requireOutput && !trimmedOutput) {
            this.showToast('Provide mock output text to test the recycler.', 'error');
            if (outputField) {
                outputField.focus();
            }
            return null;
        }

        const payload = { output_text: trimmedOutput };
        if (promptText && promptText.trim() !== '') {
            payload.prompt_override = promptText;
        }
        Object.assign(payload, this.getRecyclerModelOverrides());
        return payload;
    }

    getRecyclerModelOverrides() {
        const overrides = {};
        const providerField = document.getElementById('settings-recycler-model-provider');
        const modelField = document.getElementById('settings-recycler-model-name');

        const provider = (providerField?.value || '').toString().trim().toLowerCase();
        if (provider) {
            overrides.model_provider = provider;
        }

        const model = (modelField?.value || '').toString().trim();
        if (model) {
            overrides.model_name = model;
        }

        return overrides;
    }

    resetRecyclerResultCard() {
        const resultContainer = document.getElementById('recycler-test-result');
        if (resultContainer) {
            resultContainer.style.display = 'none';
            resultContainer.innerHTML = '';
        }
    }

    showRecyclerResultLoading() {
        const resultContainer = document.getElementById('recycler-test-result');
        if (!resultContainer) {
            return;
        }
        resultContainer.style.display = 'block';
        resultContainer.innerHTML = `
            <div class="recycler-result-skeleton">
                <div class="skeleton-line w-30"></div>
                <div class="skeleton-line w-20"></div>
                <div class="skeleton-block"></div>
            </div>
        `;
    }

    async executeRecyclerTest(payload, { showLoading = true } = {}) {
        if (showLoading) {
            this.showLoading(true);
        }

        try {
            const response = await fetch(`${this.apiBase}/recycler/test`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            let data = {};
            try {
                data = await response.json();
            } catch (parseError) {
                console.warn('Failed to parse recycler test response as JSON:', parseError);
            }

            if (!response.ok) {
                const message = data?.error || response.statusText || 'Recycler test failed';
                throw new Error(message);
            }

            return data;
        } catch (error) {
            throw error instanceof Error ? error : new Error(String(error));
        } finally {
            if (showLoading) {
                this.showLoading(false);
            }
        }
    }

    showRecyclerTestError(error) {
        console.error('Recycler test failed:', error);
        this.showToast(`Recycler test failed: ${error.message}`, 'error');

        const resultContainer = document.getElementById('recycler-test-result');
        if (resultContainer) {
            resultContainer.style.display = 'block';
            resultContainer.innerHTML = `
                <strong>Recycler test failed</strong><br>
                <span class="recycler-result-meta" style="color: var(--error-color);">${this.escapeHtml(error.message || 'Unknown error')}</span>
            `;
        }
    }

    async runRecyclerTest() {
        const payload = this.buildRecyclerPayloadFromForm();
        if (!payload) {
            return;
        }

        this.resetRecyclerResultCard();
        this.showRecyclerResultLoading();

        try {
            const data = await this.executeRecyclerTest(payload, { showLoading: false });
            if (data?.prompt) {
                this.applyPromptToTextarea(data.prompt);
            }
            this.renderRecyclerTestResult(data);

            if (data.success) {
                this.showToast('Recycler summarizer test succeeded', 'success');
            } else {
                this.showToast('Recycler summarizer fell back to default result', 'warning');
            }
        } catch (error) {
            this.showRecyclerTestError(error);
        }
    }

    renderRecyclerTestResult(data, options = {}) {
        const container = document.getElementById('recycler-test-result');
        if (!container) return;

        const result = data?.result || {};
        const rawClassification = (result.classification || 'unknown').toLowerCase();
        const formattedClassification = this.formatClassification(rawClassification);
        const noteHtml = result.note ? this.escapeHtml(result.note).replace(/\n/g, '<br>') : '<em>No note returned</em>';

        const provider = data?.provider ? this.escapeHtml(data.provider) : '—';
        const model = data?.model ? this.escapeHtml(data.model) : '—';
        const statusLabel = data?.success ? 'Summarizer Result' : 'Fallback Result';
        const presetLabel = options.presetLabel ? `<div class="recycler-result-label">${this.escapeHtml(options.presetLabel)}</div>` : '';

        const expectedRaw = options.expectedClassification ? options.expectedClassification.toLowerCase() : null;
        const expectedLabel = expectedRaw ? this.formatClassification(expectedRaw) : null;
        const match = expectedRaw ? expectedRaw === rawClassification : null;
        const classificationChipClass = expectedLabel ? (match ? 'match' : 'mismatch') : '';

        const chips = [
            `<span class="recycler-result-chip ${classificationChipClass}">${this.escapeHtml(formattedClassification)}</span>`
        ];
        if (expectedLabel) {
            chips.push(`<span class="recycler-result-chip ${match ? 'match' : 'mismatch'}">Expected: ${this.escapeHtml(expectedLabel)}</span>`);
        }

        let extra = '';
        if (data?.error) {
            extra = `<div class="recycler-result-meta" style="color: var(--error-color);">${this.escapeHtml(data.error)}</div>`;
        }

        container.style.display = 'block';
        container.innerHTML = `
            <div class="recycler-result-header">
                <div>
                    <strong>${statusLabel}</strong>
                    ${presetLabel}
                </div>
                <div class="recycler-result-meta">
                    ${chips.join(' ')}
                    <span>Provider: ${provider}</span>
                    <span>Model: ${model}</span>
                </div>
            </div>
            ${extra}
            <div class="recycler-note-preview">${noteHtml}</div>
        `;
    }

    showLoading(show) {
        UIComponents.showLoading(show);
        this.isLoading = show;
    }

    handleRateLimit(retryAfter) {
        const seconds = Number.isFinite(retryAfter) ? Math.max(1, Math.round(retryAfter)) : 60;
        this.rateLimitEndTime = Date.now() + (seconds * 1000);
        UIComponents.showRateLimitNotification(seconds);
        this.rateLimitOverlayVisible = true;
    }

    dismissRateLimitNotification(button) {
        UIComponents.dismissRateLimitNotification(button);
        this.rateLimitOverlayVisible = false;

        if (button) {
            this.rateLimitNotificationSuppressed = true;
        }
    }

    clearRateLimitNotification() {
        UIComponents.dismissRateLimitNotification();
        this.rateLimitOverlayVisible = false;
        this.rateLimitNotificationSuppressed = false;
        this.rateLimitEndTime = null;
    }

    formatSeconds(seconds) {
        if (!Number.isFinite(seconds) || seconds <= 0) {
            return 'less than a minute';
        }

        const totalSeconds = Math.round(seconds);
        const minutes = Math.floor(totalSeconds / 60);
        const remainingSeconds = totalSeconds % 60;

        if (minutes && remainingSeconds) {
            return `${minutes}m ${remainingSeconds}s`;
        }

        if (minutes) {
            return `${minutes}m`;
        }

        return `${remainingSeconds}s`;
    }

    escapeHtml(text) {
        return UIComponents.escapeHtml(text);
    }

    formatClassification(value) {
        const normalized = (value || '').toString().trim().toLowerCase();
        if (!normalized) {
            return '—';
        }

        switch (normalized) {
            case 'full_complete':
                return 'full complete';
            case 'significant_progress':
                return 'significant progress';
            case 'some_progress':
                return 'some progress';
            case 'partial_progress':
                return 'some progress';
            case 'uncertain':
                return 'uncertain';
            case 'unknown':
                return 'unknown';
            default:
                return normalized.replace(/_/g, ' ');
        }
    }

    truncateText(text, max = 160) {
        if (typeof text !== 'string') {
            return '';
        }
        if (text.length <= max) {
            return text;
        }
        return `${text.slice(0, Math.max(0, max - 1))}…`;
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

                const activeSettingsButton = document.querySelector('.settings-tab-btn.active');
                this.switchSettingsTab(activeSettingsButton?.dataset.tab || 'processor');
            });

            // Initialize Auto Steer manager if not already initialized
            if (this.autoSteerManager && !this.autoSteerManager.profiles.length) {
                this.autoSteerManager.initialize().catch(err => {
                    console.error('Failed to initialize Auto Steer manager:', err);
                });
            }
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
        this.settingsManager.applyTheme(theme, { markUserPreference: false });
    }

    resetSettingsToDefault() {
        if (confirm('Are you sure you want to reset all settings to defaults?')) {
            this.settingsManager.resetToDefaults();
            this.showToast('Settings reset to defaults', 'success');
        }
    }

    async fetchResumeDiagnostics() {
        try {
            const response = await fetch(`${this.apiBase}/queue/resume-diagnostics`);
            if (!response.ok) {
                throw new Error(`Resume diagnostics request failed: ${response.status}`);
            }
            const data = await response.json();
            return data?.diagnostics ?? null;
        } catch (error) {
            console.error('Failed to retrieve resume diagnostics:', error);
            return null;
        }
    }

    formatIdPreview(ids = [], limit = 5) {
        if (!Array.isArray(ids) || ids.length === 0) {
            return '';
        }
        const unique = [...new Set(ids.filter(Boolean))];
        if (unique.length <= limit) {
            return unique.join(', ');
        }
        const shown = unique.slice(0, limit).join(', ');
        const remaining = unique.length - limit;
        return `${shown}, +${remaining} more`;
    }

    buildResumeConfirmationMessage(diagnostics) {
        if (!diagnostics) {
            return 'Enable automatic processing for the queue processor?';
        }

        const actions = [];

        if (diagnostics.rate_limit_paused) {
            const resumeAt = diagnostics.rate_limit_resume_at || 'unknown time';
            actions.push(`Clear the active rate limit pause (was waiting until ${resumeAt}).`);
        }

        const internal = diagnostics.internal_executing_task_ids || [];
        if (internal.length > 0) {
            const preview = this.formatIdPreview(internal);
            actions.push(`Terminate ${internal.length} tracked execution${internal.length === 1 ? '' : 's'}${preview ? ` (${preview})` : ''}.`);
        }

        const agentTasks = diagnostics.active_agent_task_ids || [];
        if (agentTasks.length > 0) {
            const preview = this.formatIdPreview(agentTasks);
            actions.push(`Stop ${agentTasks.length} active Claude agent${agentTasks.length === 1 ? '' : 's'}${preview ? ` (${preview})` : ''}.`);
        }

        const orphans = diagnostics.orphan_in_progress_task_ids || [];
        if (orphans.length > 0) {
            const preview = this.formatIdPreview(orphans);
            actions.push(`Move ${orphans.length} in-progress task${orphans.length === 1 ? '' : 's'} back to pending${preview ? ` (${preview})` : ''}.`);
        }

        if (actions.length === 0) {
            return 'Enable automatic processing for the queue processor?';
        }

        const notes = Array.isArray(diagnostics.notes) && diagnostics.notes.length > 0
            ? `\n\nDiagnostics notes:\n- ${diagnostics.notes.join('\n- ')}`
            : '';

        return `Resuming the processor will run recovery steps to ensure it picks up cleanly:\n\n- ${actions.join('\n- ')}${notes}\n\nContinue?`;
    }

    formatResumeResetSummary(summary) {
        if (!summary || typeof summary !== 'object') {
            return '';
        }

        const parts = [];
        if (summary.rate_limit_cleared) {
            parts.push('cleared rate limit pause');
        }
        if (Array.isArray(summary.agents_stopped) && summary.agents_stopped.length > 0) {
            const count = summary.agents_stopped.length;
            parts.push(`stopped ${count} Claude agent${count === 1 ? '' : 's'}`);
        }
        if (Array.isArray(summary.processes_terminated) && summary.processes_terminated.length > 0) {
            const count = summary.processes_terminated.length;
            parts.push(`terminated ${count} execution${count === 1 ? '' : 's'}`);
        }
        if (Array.isArray(summary.tasks_moved_to_pending) && summary.tasks_moved_to_pending.length > 0) {
            const count = summary.tasks_moved_to_pending.length;
            parts.push(`moved ${count} task${count === 1 ? '' : 's'} to pending`);
        }

        if (parts.length === 0) {
            return '';
        }

        return parts.join('; ');
    }

    async toggleProcessor() {
        try {
            // Get current settings first
            const currentSettings = await this.settingsManager.loadSettings();
            
            // Toggle the active state
            const newActiveState = !currentSettings.active;
            const updatedSettings = { ...currentSettings, active: newActiveState };
            
            let diagnostics = null;
            if (newActiveState) {
                diagnostics = await this.fetchResumeDiagnostics();

                if (!diagnostics) {
                    const proceed = confirm('Unable to evaluate queue state automatically. Resume processor and attempt recovery anyway?');
                    if (!proceed) {
                        this.showToast('Processor activation cancelled', 'info');
                        return;
                    }
                } else if (diagnostics.needs_confirmation) {
                    const message = this.buildResumeConfirmationMessage(diagnostics);
                    const proceed = confirm(message);
                    if (!proceed) {
                        this.showToast('Processor activation cancelled', 'info');
                        return;
                    }
                }
            }

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
                
                let toastMessage = `Processor ${newActiveState ? 'activated' : 'paused'}`;
                if (newActiveState && result.resume_reset_summary) {
                    const summaryText = this.formatResumeResetSummary(result.resume_reset_summary);
                    if (summaryText) {
                        toastMessage += ` – ${summaryText}`;
                    }
                }
                this.showToast(toastMessage, 'success');
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

    filterTasks(filterState = {}) {
        const type = filterState.type || '';
        const operation = filterState.operation || '';
        const priority = filterState.priority || '';
        const searchText = ((filterState.search || '').trim()).toLowerCase();

        document.querySelectorAll('.task-card').forEach(card => {
            let visible = true;

            if (type && !card.classList.contains(type)) visible = false;
            if (operation && !card.classList.contains(operation)) visible = false;
            if (priority && !card.classList.contains(priority)) visible = false;

            if (visible && searchText) {
                const title = card.querySelector('.task-title')?.textContent?.toLowerCase() || '';
                const notes = card.dataset.notesText || '';
                visible = title.includes(searchText) || notes.includes(searchText);
            }

            card.style.display = visible ? 'block' : 'none';
        });
    }

    applyFilters(options = {}) {
        const skipUrlUpdate = options.skipUrlUpdate || false;
        const newState = this.collectFilterStateFromUI();
        const stateChanged = this.hasFilterStateChanged(newState);

        this.filterState = newState;
        this.filterTasks(newState);
        this.updateFilterSummaryUI(newState);

        if (!skipUrlUpdate && stateChanged) {
            this.updateUrlWithFilters(newState);
        }
    }

    clearFilters() {
        const typeSelect = document.getElementById('filter-type');
        const operationSelect = document.getElementById('filter-operation');
        const prioritySelect = document.getElementById('filter-priority');
        const searchInput = document.getElementById('search-input');

        if (typeSelect) typeSelect.value = '';
        if (operationSelect) operationSelect.value = '';
        if (prioritySelect) prioritySelect.value = '';
        if (searchInput) searchInput.value = '';

        Object.entries(this.defaultColumnVisibility).forEach(([status, visible]) => {
            this.setColumnVisibility(status, visible, { source: 'clear', suppressToast: true, force: true });
        });

        this.applyFilters();
        this.toggleFilterPanel(false);
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
        this.rateLimitPaused = false;
        this.rateLimitTargetEnd = null;
        this.rateLimitNotificationSuppressed = false;
        this.clearRateLimitNotification();

        this.showToast('Rate limit pause has been reset', 'success');

        const statusElement = document.getElementById('rate-limit-status');
        if (statusElement) {
            statusElement.innerHTML = `
                <span class="rate-limit-status-clear">
                    <i class="fas fa-check-circle"></i>
                    No rate limit currently active
                </span>
            `;
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
            this.restoreBodyScrollIfSafe();
        }
    }

    // ==================== Execution History Methods ====================

    // Switch tabs in the system logs modal
    switchSystemTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('#system-logs-modal .log-tab').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tabName);
        });

        // Update tab content
        document.querySelectorAll('#system-logs-modal .tab-content').forEach(content => {
            content.classList.toggle('active', content.id === `${tabName}-content`);
        });

        // Load execution history if switching to that tab
        if (tabName === 'execution-history') {
            this.loadAllExecutionHistory();
        }

        // Load profile performance if switching to that tab
        if (tabName === 'profile-performance') {
            this.profilePerformanceManager.initialize();
        }
    }

    async loadAllExecutionHistory() {
        const listContainer = document.getElementById('execution-history-list');
        if (!listContainer) return;

        // Show loading state
        listContainer.innerHTML = `
            <div class="execution-history-loading">
                <i class="fas fa-spinner fa-spin"></i>
                <p>Loading execution history...</p>
            </div>
        `;

        try {
            const data = await this.api.getAllExecutionHistory();

            // Store all executions for filtering
            this.allExecutions = data.executions || [];

            // Show empty state if no executions exist at all
            if (this.allExecutions.length === 0) {
                listContainer.innerHTML = `
                    <div class="execution-history-empty">
                        <i class="fas fa-clock"></i>
                        <p>No execution history yet</p>
                        <p class="log-hint">Task execution history will appear here once tasks start running</p>
                    </div>
                `;
                return;
            }

            // Populate task filter dropdown
            this.populateTaskFilter();

            // Apply initial filters
            this.filterExecutionHistory();
        } catch (error) {
            logger.error('Failed to load execution history:', error);
            listContainer.innerHTML = `
                <div class="execution-history-empty">
                    <i class="fas fa-exclamation-triangle"></i>
                    <p>Failed to load execution history</p>
                    <p class="log-hint">${this.escapeHtml(error.message)}</p>
                </div>
            `;
        }
    }

    refreshExecutionHistory() {
        this.loadAllExecutionHistory();
    }

    populateTaskFilter() {
        const taskFilter = document.getElementById('execution-task-filter');
        if (!taskFilter || !this.allExecutions) return;

        // Get unique task IDs
        const taskIds = [...new Set(this.allExecutions.map(e => e.task_id))].sort();

        // Keep "All Tasks" and add task IDs
        const currentValue = taskFilter.value;
        taskFilter.innerHTML = '<option value="all">All Tasks</option>' +
            taskIds.map(id => `<option value="${this.escapeHtml(id)}">${this.escapeHtml(id)}</option>`).join('');
        taskFilter.value = currentValue;
    }

    filterExecutionHistory() {
        const listContainer = document.getElementById('execution-history-list');
        if (!listContainer || !this.allExecutions) return;

        const searchTerm = document.getElementById('execution-search')?.value.toLowerCase() || '';
        const statusFilter = document.getElementById('execution-status-filter')?.value || 'all';
        const taskFilter = document.getElementById('execution-task-filter')?.value || 'all';

        const filtered = this.allExecutions.filter(execution => {
            // Search filter
            if (searchTerm && !execution.task_id.toLowerCase().includes(searchTerm) &&
                !execution.execution_id.toLowerCase().includes(searchTerm)) {
                return false;
            }

            // Status filter
            if (statusFilter !== 'all') {
                if (statusFilter === 'success' && !execution.success) return false;
                if (statusFilter === 'failure' && (execution.success || execution.timeout)) return false;
                if (statusFilter === 'timeout' && !execution.timeout) return false;
            }

            // Task filter
            if (taskFilter !== 'all' && execution.task_id !== taskFilter) {
                return false;
            }

            return true;
        });

        if (filtered.length === 0) {
            listContainer.innerHTML = `
                <div class="execution-history-empty">
                    <i class="fas fa-search"></i>
                    <p>No executions match your filters</p>
                    <p class="log-hint">Try adjusting your search or filter criteria</p>
                </div>
            `;
            return;
        }

        // Render filtered executions
        listContainer.innerHTML = filtered.map(execution =>
            this.renderExecutionHistoryItem(execution, execution.task_id)
        ).join('');
    }

    async loadExecutionHistory(taskId) {
        const listContainer = document.getElementById('execution-history-list');
        if (!listContainer) return;

        // Show loading state
        listContainer.innerHTML = `
            <div class="execution-history-loading">
                <i class="fas fa-spinner fa-spin"></i>
                <p>Loading execution history...</p>
            </div>
        `;

        try {
            const data = await this.api.getExecutionHistory(taskId);

            if (!data.executions || data.executions.length === 0) {
                listContainer.innerHTML = `
                    <div class="execution-history-empty">
                        <i class="fas fa-history"></i>
                        <p>No execution history found</p>
                        <p class="log-hint">Execution history will appear here after the task runs</p>
                    </div>
                `;
                return;
            }

            // Render execution history items
            listContainer.innerHTML = data.executions.map(execution => this.renderExecutionHistoryItem(execution, taskId)).join('');
        } catch (error) {
            logger.error('Failed to load execution history:', error);
            listContainer.innerHTML = `
                <div class="execution-history-empty">
                    <i class="fas fa-exclamation-triangle"></i>
                    <p>Failed to load execution history</p>
                    <p class="log-hint">${this.escapeHtml(error.message)}</p>
                </div>
            `;
        }
    }

    renderExecutionHistoryItem(execution, taskId) {
        const timestamp = new Date(execution.start_time);
        const duration = execution.duration || 'N/A';
        const statusBadge = this.getExecutionStatusBadge(execution);

        // Display task title if available, otherwise fall back to task ID
        const taskDisplayName = execution.task_title || taskId;
        const taskTypeOperation = execution.task_type && execution.task_operation
            ? `${execution.task_type} / ${execution.task_operation}`
            : '';

        return `
            <div class="execution-history-item" onclick="ecosystemManager.viewExecutionDetail('${taskId}', '${execution.execution_id}')">
                <div class="execution-history-header">
                    <span class="execution-task-name">${this.escapeHtml(taskDisplayName)}</span>
                    <span class="execution-timestamp">${timestamp.toLocaleString()}</span>
                </div>
                ${taskTypeOperation ? `
                <div class="execution-task-meta">
                    <span class="execution-task-type">${this.escapeHtml(taskTypeOperation)}</span>
                </div>
                ` : ''}
                <div class="execution-info">
                    <span><i class="fas fa-clock"></i> ${duration}</span>
                    ${statusBadge}
                    ${execution.exit_reason ? `<span><i class="fas fa-info-circle"></i> ${this.escapeHtml(execution.exit_reason)}</span>` : ''}
                </div>
                <div class="execution-id-small">${this.escapeHtml(execution.execution_id)}</div>
            </div>
        `;
    }

    getExecutionStatusBadge(execution) {
        if (execution.success) {
            return '<span class="execution-status-badge success">Success</span>';
        } else if (execution.timeout) {
            return '<span class="execution-status-badge timeout">Timeout</span>';
        } else {
            return '<span class="execution-status-badge failure">Failed</span>';
        }
    }

    async viewExecutionDetail(taskId, executionId) {
        // Store current execution context for tab switching
        this.currentExecutionTaskId = taskId;
        this.currentExecutionId = executionId;

        // Hide list, show detail view
        document.getElementById('execution-history-list').style.display = 'none';
        document.getElementById('execution-detail-view').style.display = 'block';
        document.getElementById('execution-detail-title').textContent = `Execution: ${executionId}`;

        // Load the prompt by default
        this.switchDetailTab('prompt');
        await this.loadExecutionPrompt(taskId, executionId);
    }

    showExecutionList() {
        document.getElementById('execution-history-list').style.display = 'block';
        document.getElementById('execution-detail-view').style.display = 'none';
    }

    switchDetailTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.detail-tab').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.detailTab === tabName);
        });

        // Update tab content
        document.querySelectorAll('.detail-tab-content').forEach(content => {
            const contentId = content.id.replace('-view', '').replace('execution-', '');
            content.classList.toggle('active', contentId === tabName);
        });

        // Load content using stored execution context
        const taskId = this.currentExecutionTaskId;
        const executionId = this.currentExecutionId;

        if (!taskId || !executionId) return;

        if (tabName === 'prompt') {
            this.loadExecutionPrompt(taskId, executionId);
        } else if (tabName === 'output') {
            this.loadExecutionOutput(taskId, executionId);
        } else if (tabName === 'metadata') {
            this.loadExecutionMetadata(taskId, executionId);
        }
    }

    async loadExecutionPrompt(taskId, executionId) {
        const textElement = document.getElementById('execution-prompt-text');
        if (!textElement) return;

        textElement.textContent = 'Loading prompt...';

        try {
            const data = await this.api.getExecutionPrompt(taskId, executionId);
            textElement.textContent = data.prompt || 'No prompt available';
        } catch (error) {
            logger.error('Failed to load execution prompt:', error);
            textElement.textContent = `Error loading prompt: ${error.message}`;
        }
    }

    async loadExecutionOutput(taskId, executionId) {
        const textElement = document.getElementById('execution-output-text');
        if (!textElement) return;

        textElement.textContent = 'Loading output...';

        try {
            const data = await this.api.getExecutionOutput(taskId, executionId);
            textElement.textContent = data.output || 'No output available';
        } catch (error) {
            logger.error('Failed to load execution output:', error);
            textElement.textContent = `Error loading output: ${error.message}`;
        }
    }

    async loadExecutionMetadata(taskId, executionId) {
        const container = document.getElementById('execution-metadata-content');
        if (!container) return;

        container.innerHTML = '<p>Loading metadata...</p>';

        try {
            const execution = await this.api.getExecutionMetadata(taskId, executionId);

            container.innerHTML = `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Execution ID:</span>
                    <span class="execution-metadata-value">${this.escapeHtml(execution.execution_id)}</span>
                </div>
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Task ID:</span>
                    <span class="execution-metadata-value">${this.escapeHtml(execution.task_id)}</span>
                </div>
                ${execution.task_title ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Task Title:</span>
                    <span class="execution-metadata-value">${this.escapeHtml(execution.task_title)}</span>
                </div>
                ` : ''}
                ${execution.task_type ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Task Type:</span>
                    <span class="execution-metadata-value">${this.escapeHtml(execution.task_type)}</span>
                </div>
                ` : ''}
                ${execution.task_operation ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Task Operation:</span>
                    <span class="execution-metadata-value">${this.escapeHtml(execution.task_operation)}</span>
                </div>
                ` : ''}
                ${execution.agent_tag ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Agent Tag:</span>
                    <span class="execution-metadata-value">${this.escapeHtml(execution.agent_tag)}</span>
                </div>
                ` : ''}
                ${execution.process_id ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Process ID:</span>
                    <span class="execution-metadata-value">${execution.process_id}</span>
                </div>
                ` : ''}
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Started At:</span>
                    <span class="execution-metadata-value">${new Date(execution.start_time).toLocaleString()}</span>
                </div>
                ${execution.end_time ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Completed At:</span>
                    <span class="execution-metadata-value">${new Date(execution.end_time).toLocaleString()}</span>
                </div>
                ` : ''}
                ${execution.duration ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Duration:</span>
                    <span class="execution-metadata-value">${execution.duration}</span>
                </div>
                ` : ''}
                ${execution.timeout_allowed ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Timeout Allowed:</span>
                    <span class="execution-metadata-value">${execution.timeout_allowed}</span>
                </div>
                ` : ''}
                ${execution.prompt_size ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Prompt Size:</span>
                    <span class="execution-metadata-value">${(execution.prompt_size / 1024).toFixed(2)} KB</span>
                </div>
                ` : ''}
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Success:</span>
                    <span class="execution-metadata-value">${execution.success ? 'Yes' : 'No'}</span>
                </div>
                ${execution.exit_reason ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Exit Reason:</span>
                    <span class="execution-metadata-value">${this.escapeHtml(execution.exit_reason)}</span>
                </div>
                ` : ''}
                ${execution.rate_limited ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Rate Limited:</span>
                    <span class="execution-metadata-value">Yes${execution.retry_after ? ` (retry after ${execution.retry_after}s)` : ''}</span>
                </div>
                ` : ''}
                ${execution.error ? `
                <div class="execution-metadata-row">
                    <span class="execution-metadata-label">Error:</span>
                    <span class="execution-metadata-value">${this.escapeHtml(execution.error)}</span>
                </div>
                ` : ''}
            `;
        } catch (error) {
            logger.error('Failed to load execution metadata:', error);
            container.innerHTML = `<p>Error loading metadata: ${this.escapeHtml(error.message)}</p>`;
        }
    }

    // ==================== End Execution History Methods ====================

    setColumnVisibility(status, isVisible, options = {}) {
        const column = document.querySelector(`[data-status="${status}"]`);
        if (!column) {
            return;
        }

        const { suppressToast = false, source = 'programmatic', force = false } = options;

        if (typeof this.columnVisibility[status] !== 'boolean') {
            this.columnVisibility[status] = this.defaultColumnVisibility.hasOwnProperty(status)
                ? this.defaultColumnVisibility[status]
                : true;
        }

        const wasVisible = this.columnVisibility[status];
        if (!force && wasVisible === isVisible) {
            return;
        }

        this.columnVisibility[status] = isVisible;
        column.classList.toggle('hidden', !isVisible);
        this.updateGridLayout();

        const chip = this.columnToggleButtons.get(status);
        if (chip) {
            chip.setAttribute('aria-pressed', isVisible ? 'true' : 'false');
        }

        if (!suppressToast) {
            const label = this.getColumnDisplayName(status);
            if (isVisible) {
                this.showToast(`${label} column shown`, 'success');
            } else {
                const hint = source === 'column-button' ? ' Use Filters → Columns to show it again.' : '';
                this.showToast(`${label} column hidden.${hint}`, 'info');
            }
        }
    }

    getColumnDisplayName(status) {
        const mapping = {
            pending: 'Pending',
            'in-progress': 'Active',
            completed: 'Completed',
            'completed-finalized': 'Finished',
            failed: 'Failed',
            'failed-blocked': 'Blocked',
            archived: 'Archived'
        };

        if (mapping[status]) {
            return mapping[status];
        }

        return status
            .split('-')
            .map(part => part.charAt(0).toUpperCase() + part.slice(1))
            .join(' ');
    }

    hideColumn(status) {
        this.setColumnVisibility(status, false, { source: 'column-button' });
    }

    showColumn(status) {
        this.setColumnVisibility(status, true, { source: 'column-button' });
    }

    updateGridLayout() {
        const board = document.querySelector('.kanban-board');
        if (!board) return;

        // Count visible columns
        const visibleColumns = board.querySelectorAll('.kanban-column:not(.hidden)').length;
        
        // Remove all column classes
        board.classList.remove('columns-1', 'columns-2', 'columns-3', 'columns-4', 'columns-5', 'columns-6', 'columns-7');
        
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
