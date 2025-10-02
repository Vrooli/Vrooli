/**
 * StateManager - Centralized Application State Management
 * Manages all application state with reactive updates via EventBus
 */

import { eventBus, EVENT_TYPES } from './EventBus.js';

/**
 * StateManager class - Centralized state with event-driven updates
 */
export class StateManager {
    constructor(eventBusInstance = eventBus) {
        this.eventBus = eventBusInstance;

        // Initialize state structure
        this.state = {
            // Current page
            activePage: 'dashboard',

            // Data collections
            data: {
                suites: [],
                executions: [],
                metrics: {},
                coverage: [],
                scenarios: [],
                vaults: [],
                reports: {
                    overview: null,
                    trends: null,
                    insights: null
                }
            },

            // UI filters
            filters: {
                suites: {
                    search: '',
                    status: 'all'
                },
                executions: {
                    search: '',
                    status: 'all'
                }
            },

            // Selections (checkbox states)
            selections: {
                selectedSuiteIds: new Set(),
                selectedExecutionIds: new Set()
            },

            // UI state
            ui: {
                // Active overlays/dialogs
                activeSuiteDetailId: null,
                activeExecutionDetailId: null,
                activeVaultId: null,
                lastSuiteDetailTrigger: null,
                lastExecutionDetailTrigger: null,

                // Dialog states
                generateDialogOpen: false,
                generateDialogScenarioName: '',
                generateDialogIsBulkMode: false,

                vaultDialogOpen: false,
                coverageDialogOpen: false,
                healthDialogOpen: false,

                // Sidebar state
                sidebarOpen: false,
                sidebarCollapsed: false,

                // Loading states
                loading: {
                    suites: false,
                    executions: false,
                    coverage: false,
                    reports: false,
                    vaults: false
                }
            },

            // System state
            system: {
                healthy: true,
                healthData: null,
                healthUpdatedAt: null,
                apiBaseUrl: '/api/v1'
            },

            // Settings
            settings: {
                coverageTarget: 80,
                phases: new Set(['dependencies', 'structure', 'unit', 'integration', 'business', 'performance']),
                mobileBreakpoint: 768
            },

            // Caches
            caches: {
                suiteDetails: new Map(),
                renderedSuiteIds: [],
                renderedExecutionIds: [],
                availableSuiteIds: []
            }
        };

        // State history for undo/debugging (last 50 state snapshots)
        this.history = [];
        this.maxHistorySize = 50;

        // Computed/derived state cache
        this.computed = new Map();

        // Watchers (callbacks for specific state paths)
        this.watchers = new Map();

        // Debug mode
        this.debug = false;
    }

    /**
     * Get state value by path
     * @param {string} path - Dot notation path (e.g., 'data.suites')
     * @returns {*}
     */
    get(path) {
        if (!path) return this.state;

        const keys = path.split('.');
        let value = this.state;

        for (const key of keys) {
            if (value === null || value === undefined) {
                return undefined;
            }
            value = value[key];
        }

        return value;
    }

    /**
     * Set state value by path
     * @param {string} path - Dot notation path
     * @param {*} value - New value
     * @param {Object} options - Update options
     */
    set(path, value, options = {}) {
        const { silent = false, merge = false } = options;

        const keys = path.split('.');
        const lastKey = keys.pop();
        let target = this.state;

        // Navigate to parent object
        for (const key of keys) {
            if (!target[key] || typeof target[key] !== 'object') {
                target[key] = {};
            }
            target = target[key];
        }

        // Store old value for comparison
        const oldValue = target[lastKey];

        // Update value (merge if requested)
        if (merge && typeof value === 'object' && !Array.isArray(value) && value !== null) {
            target[lastKey] = { ...target[lastKey], ...value };
        } else {
            target[lastKey] = value;
        }

        // Save to history
        this._addToHistory(path, oldValue, value);

        if (this.debug) {
            console.log(`[StateManager] Set '${path}'`, { oldValue, newValue: value, merge });
        }

        // Trigger watchers
        this._triggerWatchers(path, value, oldValue);

        // Emit event unless silent
        if (!silent) {
            this.eventBus.emit(EVENT_TYPES.DATA_UPDATED, {
                path,
                value,
                oldValue
            });
        }
    }

    /**
     * Update nested state (merge)
     * @param {string} path - Dot notation path
     * @param {*} value - Value to merge
     */
    update(path, value) {
        this.set(path, value, { merge: true });
    }

    /**
     * Delete state value by path
     * @param {string} path - Dot notation path
     */
    delete(path) {
        const keys = path.split('.');
        const lastKey = keys.pop();
        let target = this.state;

        for (const key of keys) {
            if (!target[key]) return;
            target = target[key];
        }

        const oldValue = target[lastKey];
        delete target[lastKey];

        if (this.debug) {
            console.log(`[StateManager] Deleted '${path}'`, { oldValue });
        }

        this.eventBus.emit(EVENT_TYPES.DATA_UPDATED, {
            path,
            value: undefined,
            oldValue,
            deleted: true
        });
    }

    /**
     * Watch for changes to a specific path
     * @param {string} path - Path to watch
     * @param {Function} callback - Callback(newValue, oldValue)
     * @returns {Function} Unwatch function
     */
    watch(path, callback) {
        if (!this.watchers.has(path)) {
            this.watchers.set(path, []);
        }

        this.watchers.get(path).push(callback);

        // Return unwatch function
        return () => {
            const callbacks = this.watchers.get(path);
            if (callbacks) {
                const index = callbacks.indexOf(callback);
                if (index !== -1) {
                    callbacks.splice(index, 1);
                }
            }
        };
    }

    /**
     * Get computed/derived value (cached)
     * @param {string} key - Computed key
     * @param {Function} computeFn - Function to compute value
     * @returns {*}
     */
    getComputed(key, computeFn) {
        if (!this.computed.has(key)) {
            this.computed.set(key, computeFn(this.state));
        }
        return this.computed.get(key);
    }

    /**
     * Invalidate computed cache
     * @param {string} key - Computed key (or null for all)
     */
    invalidateComputed(key = null) {
        if (key === null) {
            this.computed.clear();
        } else {
            this.computed.delete(key);
        }
    }

    /**
     * Get full state snapshot (deep clone)
     * @returns {Object}
     */
    getSnapshot() {
        return JSON.parse(JSON.stringify(this.state));
    }

    /**
     * Restore state from snapshot
     * @param {Object} snapshot - State snapshot
     */
    restoreSnapshot(snapshot) {
        this.state = JSON.parse(JSON.stringify(snapshot));
        this.invalidateComputed();
        this.eventBus.emit(EVENT_TYPES.DATA_UPDATED, {
            path: '*',
            value: this.state,
            restored: true
        });
    }

    /**
     * Reset state to initial values
     */
    reset() {
        const initialSnapshot = this.history[0];
        if (initialSnapshot) {
            this.restoreSnapshot(initialSnapshot.state);
        }
    }

    /**
     * Batch updates (multiple sets in one event emission)
     * @param {Function} updates - Function that performs multiple set() calls
     */
    batch(updates) {
        const originalEmit = this.eventBus.emit.bind(this.eventBus);
        const events = [];

        // Temporarily capture events instead of emitting
        this.eventBus.emit = (type, data) => {
            events.push({ type, data });
        };

        try {
            updates();
        } finally {
            // Restore original emit
            this.eventBus.emit = originalEmit;

            // Emit single batch update event
            this.eventBus.emit(EVENT_TYPES.DATA_UPDATED, {
                batch: true,
                updates: events
            });
        }
    }

    // Convenience methods for common operations

    /**
     * Set active page
     * @param {string} page
     */
    setActivePage(page) {
        this.set('activePage', page);
        this.eventBus.emit(EVENT_TYPES.PAGE_CHANGED, { page });
    }

    /**
     * Set data for a collection
     * @param {string} collection - Collection name (e.g., 'suites')
     * @param {Array} data
     */
    setData(collection, data) {
        this.set(`data.${collection}`, data);
        this.eventBus.emit(EVENT_TYPES.DATA_LOADED, { collection, data });
    }

    /**
     * Get data for a collection
     * @param {string} collection
     * @returns {Array}
     */
    getData(collection) {
        return this.get(`data.${collection}`) || [];
    }

    /**
     * Toggle suite selection
     * @param {string} suiteId
     */
    toggleSuiteSelection(suiteId) {
        const selected = this.state.selections.selectedSuiteIds;
        if (selected.has(suiteId)) {
            selected.delete(suiteId);
        } else {
            selected.add(suiteId);
        }
        this.eventBus.emit(EVENT_TYPES.SELECTION_CHANGED, {
            type: 'suite',
            suiteId,
            selected: selected.has(suiteId)
        });
    }

    /**
     * Toggle execution selection
     * @param {string} executionId
     */
    toggleExecutionSelection(executionId) {
        const selected = this.state.selections.selectedExecutionIds;
        if (selected.has(executionId)) {
            selected.delete(executionId);
        } else {
            selected.add(executionId);
        }
        this.eventBus.emit(EVENT_TYPES.SELECTION_CHANGED, {
            type: 'execution',
            executionId,
            selected: selected.has(executionId)
        });
    }

    /**
     * Clear all selections
     * @param {string} type - 'suite' or 'execution' or 'all'
     */
    clearSelections(type = 'all') {
        if (type === 'suite' || type === 'all') {
            this.state.selections.selectedSuiteIds.clear();
        }
        if (type === 'execution' || type === 'all') {
            this.state.selections.selectedExecutionIds.clear();
        }
        this.eventBus.emit(EVENT_TYPES.SELECTION_CHANGED, { type, cleared: true });
    }

    /**
     * Set loading state
     * @param {string} key - Loading key (e.g., 'suites')
     * @param {boolean} loading
     */
    setLoading(key, loading) {
        this.set(`ui.loading.${key}`, loading, { silent: true });
    }

    /**
     * Update system health
     * @param {Object} healthData
     */
    updateHealth(healthData) {
        this.batch(() => {
            this.set('system.healthy', healthData.healthy !== false);
            this.set('system.healthData', healthData);
            this.set('system.healthUpdatedAt', new Date());
        });
        this.eventBus.emit(EVENT_TYPES.HEALTH_CHANGED, healthData);
    }

    /**
     * Set filter value
     * @param {string} collection - 'suites' or 'executions'
     * @param {string} filterKey - Filter key (e.g., 'search')
     * @param {*} value
     */
    setFilter(collection, filterKey, value) {
        this.set(`filters.${collection}.${filterKey}`, value);
    }

    /**
     * Get filter value
     * @param {string} collection
     * @param {string} filterKey
     * @returns {*}
     */
    getFilter(collection, filterKey) {
        return this.get(`filters.${collection}.${filterKey}`);
    }

    /**
     * Enable/disable debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[StateManager] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            stateKeys: Object.keys(this.state),
            dataCollections: Object.keys(this.state.data).map(key => ({
                key,
                itemCount: Array.isArray(this.state.data[key]) ? this.state.data[key].length : 'N/A'
            })),
            selections: {
                suites: this.state.selections.selectedSuiteIds.size,
                executions: this.state.selections.selectedExecutionIds.size
            },
            cacheSize: this.state.caches.suiteDetails.size,
            computedKeys: Array.from(this.computed.keys()),
            watcherPaths: Array.from(this.watchers.keys()),
            historySize: this.history.length
        };
    }

    // Private methods

    /**
     * Trigger watchers for a path
     * @private
     */
    _triggerWatchers(path, newValue, oldValue) {
        // Exact match watchers
        const watchers = this.watchers.get(path) || [];
        watchers.forEach(callback => {
            try {
                callback(newValue, oldValue, path);
            } catch (error) {
                console.error(`[StateManager] Error in watcher for '${path}':`, error);
            }
        });

        // Wildcard watchers (future enhancement)
        const wildcardWatchers = this.watchers.get('*') || [];
        wildcardWatchers.forEach(callback => {
            try {
                callback(newValue, oldValue, path);
            } catch (error) {
                console.error(`[StateManager] Error in wildcard watcher:`, error);
            }
        });
    }

    /**
     * Add state change to history
     * @private
     */
    _addToHistory(path, oldValue, newValue) {
        this.history.push({
            timestamp: Date.now(),
            path,
            oldValue: this._cloneValue(oldValue),
            newValue: this._cloneValue(newValue)
        });

        // Trim history if too large
        if (this.history.length > this.maxHistorySize) {
            this.history.shift();
        }
    }

    /**
     * Clone a value for history storage
     * @private
     */
    _cloneValue(value) {
        if (value instanceof Set || value instanceof Map) {
            return value; // Don't clone Sets/Maps
        }
        try {
            return JSON.parse(JSON.stringify(value));
        } catch {
            return value; // Return as-is if can't clone
        }
    }
}

// Export singleton instance
export const stateManager = new StateManager();

// Export default for convenience
export default stateManager;
