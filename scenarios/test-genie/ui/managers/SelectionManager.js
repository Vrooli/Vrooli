/**
 * SelectionManager - Multi-select Checkbox Management
 * Handles checkbox selection state for suites, executions, and other list views
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { normalizeId } from '../utils/formatters.js';

/**
 * SelectionManager class - Manages multi-select checkbox interactions
 */
export class SelectionManager {
    constructor(eventBusInstance = eventBus, stateManagerInstance = stateManager) {
        this.eventBus = eventBusInstance;
        this.stateManager = stateManagerInstance;

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize selection manager
     */
    initialize() {
        this.setupEventListeners();

        if (this.debug) {
            console.log('[SelectionManager] Selection system initialized');
        }
    }

    /**
     * Setup event listeners
     * @private
     */
    setupEventListeners() {
        // Listen for data updates to prune invalid selections
        this.eventBus.on(EVENT_TYPES.DATA_LOADED, (event) => {
            const { collection, data } = event.data;

            if (collection === 'suites') {
                this.pruneSuiteSelection(data);
                this.updateSuiteSelectionUI();
            } else if (collection === 'executions') {
                this.pruneExecutionSelection(data);
                this.updateExecutionSelectionUI();
            }
        });

        // Listen for clear selection requests
        this.eventBus.on(EVENT_TYPES.SELECTION_CLEAR, (event) => {
            const { collection } = event.data;

            if (collection === 'suites') {
                this.clearSuiteSelection();
            } else if (collection === 'executions') {
                this.clearExecutionSelection();
            } else {
                this.clearAllSelections();
            }
        });
    }

    /**
     * Toggle suite selection
     * @param {string} suiteId - Suite ID
     */
    toggleSuiteSelection(suiteId) {
        const normalizedId = normalizeId(suiteId);
        if (!normalizedId) {
            return;
        }

        const selectedIds = this.stateManager.get('selections.selectedSuiteIds');

        if (selectedIds.has(normalizedId)) {
            selectedIds.delete(normalizedId);
        } else {
            selectedIds.add(normalizedId);
        }

        // Trigger state update
        this.stateManager.toggleSuiteSelection(normalizedId);
        this.updateSuiteSelectionUI();

        this.eventBus.emit(EVENT_TYPES.SELECTION_CHANGED, {
            collection: 'suites',
            selectedIds: Array.from(selectedIds)
        });
    }

    /**
     * Toggle execution selection
     * @param {string} executionId - Execution ID
     */
    toggleExecutionSelection(executionId) {
        const normalizedId = normalizeId(executionId);
        if (!normalizedId) {
            return;
        }

        const selectedIds = this.stateManager.get('selections.selectedExecutionIds');

        if (selectedIds.has(normalizedId)) {
            selectedIds.delete(normalizedId);
        } else {
            selectedIds.add(normalizedId);
        }

        // Trigger state update
        this.stateManager.toggleExecutionSelection(normalizedId);
        this.updateExecutionSelectionUI();

        this.eventBus.emit(EVENT_TYPES.SELECTION_CHANGED, {
            collection: 'executions',
            selectedIds: Array.from(selectedIds)
        });
    }

    /**
     * Select all suites
     * @param {Array<string>} suiteIds - All available suite IDs
     */
    selectAllSuites(suiteIds = []) {
        const normalizedIds = suiteIds.map(id => normalizeId(id)).filter(Boolean);
        const selectedIds = this.stateManager.get('selections.selectedSuiteIds');

        normalizedIds.forEach(id => selectedIds.add(id));
        this.stateManager.set('selections.selectedSuiteIds', selectedIds);
        this.updateSuiteSelectionUI();

        this.eventBus.emit(EVENT_TYPES.SELECTION_CHANGED, {
            collection: 'suites',
            selectedIds: Array.from(selectedIds)
        });
    }

    /**
     * Select all executions
     * @param {Array<string>} executionIds - All available execution IDs
     */
    selectAllExecutions(executionIds = []) {
        const normalizedIds = executionIds.map(id => normalizeId(id)).filter(Boolean);
        const selectedIds = this.stateManager.get('selections.selectedExecutionIds');

        normalizedIds.forEach(id => selectedIds.add(id));
        this.stateManager.set('selections.selectedExecutionIds', selectedIds);
        this.updateExecutionSelectionUI();

        this.eventBus.emit(EVENT_TYPES.SELECTION_CHANGED, {
            collection: 'executions',
            selectedIds: Array.from(selectedIds)
        });
    }

    /**
     * Clear suite selection
     */
    clearSuiteSelection() {
        const selectedIds = this.stateManager.get('selections.selectedSuiteIds');
        selectedIds.clear();
        this.stateManager.set('selections.selectedSuiteIds', selectedIds);
        this.updateSuiteSelectionUI();

        this.eventBus.emit(EVENT_TYPES.SELECTION_CHANGED, {
            collection: 'suites',
            selectedIds: []
        });
    }

    /**
     * Clear execution selection
     */
    clearExecutionSelection() {
        const selectedIds = this.stateManager.get('selections.selectedExecutionIds');
        selectedIds.clear();
        this.stateManager.set('selections.selectedExecutionIds', selectedIds);
        this.updateExecutionSelectionUI();

        this.eventBus.emit(EVENT_TYPES.SELECTION_CHANGED, {
            collection: 'executions',
            selectedIds: []
        });
    }

    /**
     * Clear all selections
     */
    clearAllSelections() {
        this.clearSuiteSelection();
        this.clearExecutionSelection();
    }

    /**
     * Get selected suite IDs
     * @returns {Array<string>}
     */
    getSelectedSuiteIds() {
        const selectedIds = this.stateManager.get('selections.selectedSuiteIds');
        return Array.from(selectedIds);
    }

    /**
     * Get selected execution IDs
     * @returns {Array<string>}
     */
    getSelectedExecutionIds() {
        const selectedIds = this.stateManager.get('selections.selectedExecutionIds');
        return Array.from(selectedIds);
    }

    /**
     * Check if suite is selected
     * @param {string} suiteId
     * @returns {boolean}
     */
    isSuiteSelected(suiteId) {
        const normalizedId = normalizeId(suiteId);
        if (!normalizedId) return false;

        const selectedIds = this.stateManager.get('selections.selectedSuiteIds');
        return selectedIds.has(normalizedId);
    }

    /**
     * Check if execution is selected
     * @param {string} executionId
     * @returns {boolean}
     */
    isExecutionSelected(executionId) {
        const normalizedId = normalizeId(executionId);
        if (!normalizedId) return false;

        const selectedIds = this.stateManager.get('selections.selectedExecutionIds');
        return selectedIds.has(normalizedId);
    }

    /**
     * Update suite selection UI
     * Updates checkboxes and action button visibility
     */
    updateSuiteSelectionUI() {
        const suites = this.stateManager.get('data.suites') || [];
        const selectedIds = this.stateManager.get('selections.selectedSuiteIds');

        const suiteIds = suites.map(suite => normalizeId(suite.id)).filter(Boolean);
        const selectedCount = suiteIds.filter(id => selectedIds.has(id)).length;
        const hasSelection = selectedCount > 0;
        const allSelected = suiteIds.length > 0 && selectedCount === suiteIds.length;

        // Update "select all" checkbox
        const selectAllCheckbox = document.querySelector('input[data-suite-select-all]');
        if (selectAllCheckbox) {
            selectAllCheckbox.checked = allSelected;
            selectAllCheckbox.indeterminate = !allSelected && selectedCount > 0;
        }

        // Update action buttons visibility
        const runSelectedButton = document.getElementById('suites-run-selected-btn');
        if (runSelectedButton) {
            runSelectedButton.style.display = hasSelection ? '' : 'none';
        }

        const runSelectedLabel = document.getElementById('suites-run-selected-label');
        if (runSelectedLabel) {
            runSelectedLabel.textContent = allSelected ? 'Run All' : 'Run Selected';
        }

        // Update individual checkboxes
        document.querySelectorAll('input[data-suite-id]').forEach(checkbox => {
            const suiteId = normalizeId(checkbox.dataset.suiteId);
            if (suiteId) {
                checkbox.checked = selectedIds.has(suiteId);
            }
        });
    }

    /**
     * Update execution selection UI
     * Updates checkboxes and action button visibility
     */
    updateExecutionSelectionUI() {
        const executions = this.stateManager.get('data.executions') || [];
        const selectedIds = this.stateManager.get('selections.selectedExecutionIds');

        const executionIds = executions.map(exec => normalizeId(exec.id)).filter(Boolean);
        const selectedCount = executionIds.filter(id => selectedIds.has(id)).length;
        const hasSelection = selectedCount > 0;
        const allSelected = executionIds.length > 0 && selectedCount === executionIds.length;

        // Update "select all" checkbox
        const selectAllCheckbox = document.querySelector('input[data-execution-select-all]');
        if (selectAllCheckbox) {
            selectAllCheckbox.checked = allSelected;
            selectAllCheckbox.indeterminate = !allSelected && selectedCount > 0;
        }

        // Update action buttons visibility
        const deleteSelectedButton = document.getElementById('executions-delete-selected-btn');
        if (deleteSelectedButton) {
            deleteSelectedButton.style.display = hasSelection ? '' : 'none';
        }

        // Update individual checkboxes
        document.querySelectorAll('input[data-execution-id]').forEach(checkbox => {
            const executionId = normalizeId(checkbox.dataset.executionId);
            if (executionId) {
                checkbox.checked = selectedIds.has(executionId);
            }
        });
    }

    /**
     * Prune suite selection - remove IDs that no longer exist
     * @param {Array<Object>} suites - Current suites
     */
    pruneSuiteSelection(suites) {
        const selectedIds = this.stateManager.get('selections.selectedSuiteIds');
        const validIds = new Set();

        (suites || []).forEach(suite => {
            if (!suite) return;

            if (typeof suite === 'string') {
                const id = normalizeId(suite);
                if (id) validIds.add(id);
                return;
            }

            if (typeof suite === 'object') {
                const id = normalizeId(suite.id || suite.latestSuiteId);
                if (id) validIds.add(id);
            }
        });

        // Remove invalid selections
        Array.from(selectedIds).forEach(id => {
            if (!validIds.has(id)) {
                selectedIds.delete(id);
            }
        });

        this.stateManager.set('selections.selectedSuiteIds', selectedIds);
    }

    /**
     * Prune execution selection - remove IDs that no longer exist
     * @param {Array<Object>} executions - Current executions
     */
    pruneExecutionSelection(executions) {
        const selectedIds = this.stateManager.get('selections.selectedExecutionIds');
        const validIds = new Set(
            (executions || []).map(exec => normalizeId(exec.id)).filter(Boolean)
        );

        // Remove invalid selections
        Array.from(selectedIds).forEach(id => {
            if (!validIds.has(id)) {
                selectedIds.delete(id);
            }
        });

        this.stateManager.set('selections.selectedExecutionIds', selectedIds);
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[SelectionManager] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            selectedSuites: this.getSelectedSuiteIds(),
            selectedExecutions: this.getSelectedExecutionIds(),
            suiteCount: this.getSelectedSuiteIds().length,
            executionCount: this.getSelectedExecutionIds().length
        };
    }
}

// Export singleton instance
export const selectionManager = new SelectionManager();

// Export default for convenience
export default selectionManager;
