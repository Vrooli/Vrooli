/**
 * ExecutionsPage - Test Executions History View
 * Handles loading, rendering, filtering, and managing test executions
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { apiClient } from '../core/ApiClient.js';
import { notificationManager } from '../managers/NotificationManager.js';
import { selectionManager } from '../managers/SelectionManager.js';
import { dialogManager } from '../managers/DialogManager.js';
import {
    normalizeCollection,
    normalizeId,
    formatTimestamp,
    calculateDuration,
    getStatusClass,
    escapeHtml
} from '../utils/index.js';
import { enableDragScroll, refreshIcons } from '../utils/domHelpers.js';

/**
 * ExecutionsPage class - Manages test executions page functionality
 */
export class ExecutionsPage {
    constructor(
        eventBusInstance = eventBus,
        stateManagerInstance = stateManager,
        apiClientInstance = apiClient,
        notificationManagerInstance = notificationManager,
        selectionManagerInstance = selectionManager,
        dialogManagerInstance = dialogManager
    ) {
        this.eventBus = eventBusInstance;
        this.stateManager = stateManagerInstance;
        this.apiClient = apiClientInstance;
        this.notificationManager = notificationManagerInstance;
        this.selectionManager = selectionManagerInstance;
        this.dialogManager = dialogManagerInstance;

        // DOM references
        this.executionsTableContainer = null;

        // Current data
        this.currentExecutionRows = [];
        this.renderedExecutionIds = [];

        // Filters
        this.executionFilters = {
            search: '',
            status: 'all'
        };

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize executions page
     */
    initialize() {
        this.setupDOMReferences();
        this.setupEventListeners();

        if (this.debug) {
            console.log('[ExecutionsPage] Executions page initialized');
        }
    }

    /**
     * Setup DOM element references
     * @private
     */
    setupDOMReferences() {
        this.executionsTableContainer = document.getElementById('executions-table');
    }

    /**
     * Setup event listeners
     * @private
     */
    setupEventListeners() {
        // Listen for page load requests
        this.eventBus.on(EVENT_TYPES.PAGE_LOAD_REQUESTED, (event) => {
            if (event.data.page === 'executions') {
                this.load();
            }
        });

        // Listen for execution updates (real-time)
        this.eventBus.on(EVENT_TYPES.EXECUTION_UPDATED, () => {
            const activePage = this.stateManager.get('activePage');
            if (activePage === 'executions') {
                this.load();
            }
        });

        // Listen for execution completion
        this.eventBus.on(EVENT_TYPES.EXECUTION_COMPLETED, () => {
            const activePage = this.stateManager.get('activePage');
            if (activePage === 'executions') {
                this.load();
            }
        });

        // Listen for filter changes
        this.eventBus.on(EVENT_TYPES.FILTER_CHANGED, (event) => {
            if (event.data.collection === 'executions') {
                this.executionFilters = {
                    ...this.executionFilters,
                    ...event.data.filters
                };
                this.renderExecutionsTableWithCurrentData();
            }
        });
    }

    /**
     * Load test executions data
     */
    async load() {
        if (!this.executionsTableContainer) {
            return;
        }

        if (this.debug) {
            console.log('[ExecutionsPage] Loading test executions');
        }

        this.executionsTableContainer.innerHTML = '<div class="loading"><div class="spinner"></div>Loading test executions...</div>';

        try {
            // Set loading state
            this.stateManager.setLoading('executions', true);

            // Load executions
            const executionsResponse = await this.apiClient.getTestExecutions();
            const executions = normalizeCollection(executionsResponse, 'executions');

            // Update state
            this.stateManager.setData('executions', executions);

            // Prune invalid selections
            this.selectionManager.pruneExecutionSelection(executions);

            // Transform to display format
            if (executions && executions.length > 0) {
                const formattedExecutions = executions.map(exec => ({
                    id: normalizeId(exec.id),
                    suiteName: this.lookupSuiteName(exec.suite_id) || exec.suite_name || 'Unknown Suite',
                    status: exec.status,
                    statusClass: getStatusClass(exec.status),
                    duration: calculateDuration(exec.start_time, exec.end_time),
                    passed: exec.passed_tests || 0,
                    failed: exec.failed_tests || 0,
                    timestamp: exec.start_time
                }));

                this.currentExecutionRows = formattedExecutions;
                this.renderExecutionsTableWithCurrentData();
            } else {
                this.currentExecutionRows = [];
                this.renderedExecutionIds = [];
                this.executionsTableContainer.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No test executions found</p>';
                this.selectionManager.updateExecutionSelectionUI();
            }

            // Emit success event
            this.eventBus.emit(EVENT_TYPES.PAGE_LOADED, { page: 'executions' });

        } catch (error) {
            console.error('[ExecutionsPage] Failed to load test executions:', error);
            this.executionsTableContainer.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load test executions</p>';
            this.currentExecutionRows = [];
            this.renderedExecutionIds = [];
            this.selectionManager.updateExecutionSelectionUI();
        } finally {
            this.stateManager.setLoading('executions', false);
        }
    }

    /**
     * Lookup suite name by suite ID
     * @param {string} suiteId - Suite ID
     * @returns {string|null} Suite name or null
     * @private
     */
    lookupSuiteName(suiteId) {
        if (!suiteId) {
            return null;
        }

        const suites = this.stateManager.get('data.suites') || [];
        const match = suites.find(suite => suite.id === suiteId);
        return match ? (match.scenario_name || match.name) : null;
    }

    /**
     * Apply filters to execution rows
     * @param {Array} rows - Array of execution rows
     * @returns {Array} Filtered rows
     * @private
     */
    applyExecutionFilters(rows) {
        if (!Array.isArray(rows)) {
            return [];
        }

        let filtered = rows;

        // Apply search filter
        const search = (this.executionFilters.search || '').trim().toLowerCase();
        if (search) {
            filtered = filtered.filter((row) => {
                const nameMatch = (row.suiteName || '').toLowerCase().includes(search);
                const statusMatch = (row.status || '').toLowerCase().includes(search);
                const idMatch = row.id && normalizeId(row.id).includes(search);
                return nameMatch || statusMatch || idMatch;
            });
        }

        // Apply status filter
        const statusFilter = this.executionFilters.status;
        if (statusFilter && statusFilter !== 'all') {
            filtered = filtered.filter((row) => (row.statusClass || getStatusClass(row.status)) === statusFilter);
        }

        return filtered;
    }

    /**
     * Render executions table with current data
     * @param {HTMLElement} container - Container element (optional)
     */
    renderExecutionsTableWithCurrentData(container = this.executionsTableContainer) {
        if (!container) {
            return;
        }

        // Handle empty data
        if (!Array.isArray(this.currentExecutionRows) || this.currentExecutionRows.length === 0) {
            container.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No test executions found</p>';
            this.renderedExecutionIds = [];
            this.selectionManager.updateExecutionSelectionUI();
            return;
        }

        // Apply filters
        const filtered = this.applyExecutionFilters(this.currentExecutionRows);

        // Handle no results after filtering
        if (!filtered.length) {
            container.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No executions match the current filters.</p>';
            this.renderedExecutionIds = [];
            this.selectionManager.updateExecutionSelectionUI();
            return;
        }

        // Render table
        container.innerHTML = this.renderExecutionsTable(filtered);
        container.scrollLeft = 0;

        // Update rendered IDs
        this.renderedExecutionIds = filtered
            .map((row) => normalizeId(row.id))
            .filter(Boolean);

        // Enable drag scrolling
        enableDragScroll(container);

        // Bind actions
        this.bindExecutionsTableActions(container);

        // Refresh icons
        refreshIcons();

        // Update selection UI
        this.selectionManager.updateExecutionSelectionUI();
    }

    /**
     * Render executions table HTML
     * @param {Array} executions - Array of formatted executions
     * @returns {string} HTML string
     * @private
     */
    renderExecutionsTable(executions) {
        if (!executions || executions.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No test executions found</p>';
        }

        return `
            <table class="table executions-table" data-selectable="true">
                <thead>
                    <tr>
                        <th class="cell-select">
                            <input type="checkbox" data-execution-select-all aria-label="Select all executions">
                        </th>
                        <th>Suite Name</th>
                        <th>Status</th>
                        <th>Duration</th>
                        <th>Passed</th>
                        <th>Failed</th>
                        <th>Time</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${executions.map((exec) => this.renderExecutionRow(exec)).join('')}
                </tbody>
            </table>
        `;
    }

    /**
     * Render single execution row
     * @param {Object} exec - Execution object
     * @returns {string} HTML string
     * @private
     */
    renderExecutionRow(exec) {
        const executionId = normalizeId(exec.id);
        const isChecked = this.selectionManager.isExecutionSelected(executionId);
        const failedClass = exec.failed > 0 ? 'text-error' : 'text-muted';
        const durationLabel = Number.isFinite(exec.duration) ? `${exec.duration}s` : '—';
        const statusClass = exec.statusClass || getStatusClass(exec.status);

        return `
            <tr data-execution-id="${executionId}">
                <td class="cell-select">
                    <input type="checkbox" data-execution-id="${executionId}" ${isChecked ? 'checked' : ''} aria-label="Select execution">
                </td>
                <td class="cell-scenario"><strong>${escapeHtml(exec.suiteName)}</strong></td>
                <td class="cell-status"><span class="status ${statusClass}">${escapeHtml(exec.status)}</span></td>
                <td>${durationLabel}</td>
                <td class="text-success">${exec.passed}</td>
                <td class="${failedClass}">${exec.failed}</td>
                <td class="cell-created">${formatTimestamp(exec.timestamp)}</td>
                <td class="cell-actions">
                    <button class="btn icon-btn" type="button" data-action="view-execution" data-execution-id="${executionId}" aria-label="View execution details">
                        <i data-lucide="bar-chart-3"></i>
                    </button>
                    <button class="btn icon-btn" type="button" data-action="delete-execution" data-execution-id="${executionId}" aria-label="Delete execution">
                        <i data-lucide="trash-2"></i>
                    </button>
                </td>
            </tr>
        `;
    }

    /**
     * Bind actions to executions table
     * @param {HTMLElement} container - Container element
     * @private
     */
    bindExecutionsTableActions(container) {
        if (!container) {
            return;
        }

        // Prevent duplicate binding
        if (container.dataset.bound === 'true') {
            return;
        }

        // Checkbox selection handler
        container.addEventListener('change', (event) => {
            // Select all handler - check this first
            if (event.target.hasAttribute('data-execution-select-all')) {
                const selectAllCheckbox = event.target;
                if (selectAllCheckbox.checked) {
                    this.selectionManager.selectAllExecutions(this.renderedExecutionIds);
                } else {
                    this.selectionManager.clearExecutionSelection();
                }
                return;
            }

            // Individual checkbox handler
            if (event.target.type === 'checkbox' && event.target.hasAttribute('data-execution-id')) {
                const executionId = event.target.dataset.executionId;
                if (executionId) {
                    this.selectionManager.toggleExecutionSelection(executionId);
                }
            }
        });

        // Action buttons handler
        container.addEventListener('click', (event) => {
            const button = event.target.closest('[data-action]');
            if (!button) {
                return;
            }

            const action = button.dataset.action;

            if (action === 'view-execution' && button.dataset.executionId) {
                this.viewExecution(button.dataset.executionId, button);
            } else if (action === 'delete-execution' && button.dataset.executionId) {
                this.deleteExecution(button.dataset.executionId);
            }
        });

        // Row click handler (view execution details)
        container.addEventListener('click', (event) => {
            // Ignore clicks on checkboxes and buttons
            if (event.target.closest('input[type="checkbox"], button')) {
                return;
            }

            // Find the closest row element
            const row = event.target.closest('tr[data-execution-id]');
            if (row && row.dataset.executionId) {
                const executionId = row.dataset.executionId;
                if (executionId.trim()) {
                    this.viewExecution(executionId);
                }
            }
        });

        container.dataset.bound = 'true';
    }

    /**
     * View execution details
     * @param {string} executionId - Execution ID
     * @param {HTMLElement} triggerElement - Trigger element (optional)
     */
    viewExecution(executionId, triggerElement = null) {
        if (this.debug) {
            console.log('[ExecutionsPage] Viewing execution:', executionId);
        }

        // Open execution detail dialog
        this.dialogManager.openExecutionDetailDialog(executionId);

        // Load execution details into the dialog
        this.loadExecutionDetails(executionId);
    }

    /**
     * Load execution details into dialog
     * @param {string} executionId - Execution ID
     * @private
     */
    async loadExecutionDetails(executionId) {
        const contentContainer = document.getElementById('execution-detail-content');
        if (!contentContainer) {
            return;
        }

        contentContainer.innerHTML = '<div class="loading"><div class="spinner"></div>Loading execution details...</div>';

        try {
            const execution = await this.apiClient.getTestExecution(executionId);
            if (!execution) {
                contentContainer.innerHTML = '<div class="execution-detail-empty">Execution not found.</div>';
                return;
            }

            // Update dialog header
            const titleEl = document.getElementById('execution-detail-title');
            const idEl = document.getElementById('execution-detail-id');
            const subtitleEl = document.getElementById('execution-detail-subtitle');
            const statusEl = document.getElementById('execution-detail-status');

            if (titleEl) {
                titleEl.textContent = execution.suite_name || 'Test Execution';
            }
            if (idEl) {
                idEl.textContent = `Execution ID: ${executionId}`;
            }
            if (subtitleEl) {
                const duration = calculateDuration(execution.start_time, execution.end_time);
                subtitleEl.textContent = `Duration: ${duration}s • Started ${formatTimestamp(execution.start_time)}`;
            }
            if (statusEl) {
                const statusClass = getStatusClass(execution.status);
                statusEl.innerHTML = `
                    <span class="status ${statusClass}">${escapeHtml(execution.status || 'unknown')}</span>
                    <span class="execution-detail-chip">
                        <i data-lucide="check-circle"></i>
                        ${execution.passed_tests || 0} passed
                    </span>
                    <span class="execution-detail-chip">
                        <i data-lucide="x-circle"></i>
                        ${execution.failed_tests || 0} failed
                    </span>
                `;
            }

            // Render execution content
            contentContainer.innerHTML = this.renderExecutionDetailContent(execution);
            refreshIcons();

        } catch (error) {
            console.error('[ExecutionsPage] Failed to load execution details:', error);
            contentContainer.innerHTML = '<div class="execution-detail-empty" style="color: var(--accent-error);">Failed to load execution details.</div>';
        }
    }

    /**
     * Render execution detail content
     * @param {Object} execution - Execution data
     * @returns {string} HTML string
     * @private
     */
    renderExecutionDetailContent(execution) {
        const failedTests = Array.isArray(execution.failed_test_details) ? execution.failed_test_details : [];

        if (failedTests.length === 0 && execution.failed_tests === 0) {
            return '<div class="execution-detail-empty" style="color: var(--accent-success);">All tests passed! 🎉</div>';
        }

        if (failedTests.length === 0) {
            return '<div class="execution-detail-empty">No detailed failure information available.</div>';
        }

        const failedTestsHtml = failedTests.map((test, index) => {
            const testName = test.name || test.test_name || `Test ${index + 1}`;
            const error = test.error || test.message || 'No error message available';

            return `
                <details class="execution-detail-failed-test">
                    <summary>
                        <i data-lucide="alert-circle"></i>
                        ${escapeHtml(testName)}
                    </summary>
                    <div class="execution-detail-assertion">
                        <strong>Error:</strong>
                        <pre>${escapeHtml(error)}</pre>
                    </div>
                </details>
            `;
        }).join('');

        return `
            <div class="suite-detail-section">
                <h3>Failed Tests (${failedTests.length})</h3>
                <div class="execution-detail-failed-tests">
                    ${failedTestsHtml}
                </div>
            </div>
        `;
    }

    /**
     * Delete single execution
     * @param {string} executionId - Execution ID
     */
    async deleteExecution(executionId) {
        if (!confirm('Are you sure you want to delete this execution?')) {
            return;
        }

        if (this.debug) {
            console.log('[ExecutionsPage] Deleting execution:', executionId);
        }

        try {
            await this.apiClient.deleteTestExecution(executionId);

            this.notificationManager.showSuccess('Execution deleted successfully');

            // Reload data
            await this.load();

        } catch (error) {
            console.error('[ExecutionsPage] Failed to delete execution:', error);
            this.notificationManager.showError(`Failed to delete execution: ${error.message}`);
        }
    }

    /**
     * Delete selected executions (bulk delete)
     */
    async deleteSelectedExecutions() {
        const selectedIds = this.selectionManager.getSelectedExecutionIds();

        if (selectedIds.length === 0) {
            this.notificationManager.showWarning('No executions selected');
            return;
        }

        if (!confirm(`Are you sure you want to delete ${selectedIds.length} execution${selectedIds.length === 1 ? '' : 's'}?`)) {
            return;
        }

        if (this.debug) {
            console.log('[ExecutionsPage] Deleting selected executions:', selectedIds);
        }

        try {
            this.notificationManager.showInfo(`Deleting ${selectedIds.length} execution${selectedIds.length === 1 ? '' : 's'}...`);

            // Delete all selected executions in parallel
            await Promise.all(
                selectedIds.map(id => this.apiClient.deleteTestExecution(id))
            );

            this.notificationManager.showSuccess(`Deleted ${selectedIds.length} execution${selectedIds.length === 1 ? '' : 's'}!`);

            // Clear selection
            this.selectionManager.clearExecutionSelection();

            // Reload data
            await this.load();

        } catch (error) {
            console.error('[ExecutionsPage] Failed to delete selected executions:', error);
            this.notificationManager.showError(`Failed to delete executions: ${error.message}`);
        }
    }

    /**
     * Clear all executions
     */
    async clearAllExecutions() {
        const count = this.currentExecutionRows.length;

        if (count === 0) {
            this.notificationManager.showWarning('No executions to clear');
            return;
        }

        if (!confirm(`Are you sure you want to delete ALL ${count} execution${count === 1 ? '' : 's'}? This cannot be undone.`)) {
            return;
        }

        if (this.debug) {
            console.log('[ExecutionsPage] Clearing all executions');
        }

        try {
            this.notificationManager.showInfo('Clearing all executions...');

            // Delete all executions
            const allIds = this.currentExecutionRows.map(row => row.id);
            await Promise.all(
                allIds.map(id => this.apiClient.deleteTestExecution(id))
            );

            this.notificationManager.showSuccess(`Cleared ${count} execution${count === 1 ? '' : 's'}!`);

            // Clear selection
            this.selectionManager.clearExecutionSelection();

            // Reload data
            await this.load();

        } catch (error) {
            console.error('[ExecutionsPage] Failed to clear all executions:', error);
            this.notificationManager.showError(`Failed to clear executions: ${error.message}`);
        }
    }

    /**
     * Refresh executions data
     */
    async refresh() {
        await this.load();
        this.notificationManager.showSuccess('Executions refreshed!');
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[ExecutionsPage] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            hasExecutionsTableContainer: this.executionsTableContainer !== null,
            currentExecutionRowsCount: this.currentExecutionRows.length,
            renderedExecutionIdsCount: this.renderedExecutionIds.length,
            filters: { ...this.executionFilters }
        };
    }
}

// Export singleton instance
export const executionsPage = new ExecutionsPage();

// Export default for convenience
export default executionsPage;
