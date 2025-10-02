/**
 * EventBus - Pub/Sub Event System for Test Genie
 * Enables loose coupling between components through event-driven architecture
 */

/**
 * Event Types Constants
 */
export const EVENT_TYPES = {
    // Data Events
    DATA_LOADED: 'data:loaded',
    DATA_UPDATED: 'data:updated',
    DATA_ERROR: 'data:error',

    // Page Events
    PAGE_CHANGED: 'page:changed',
    PAGE_LOADED: 'page:loaded',
    PAGE_LOAD_REQUESTED: 'page:load:requested',

    // Selection Events
    SELECTION_CHANGED: 'selection:changed',
    SUITE_SELECTED: 'suite:selected',
    EXECUTION_SELECTED: 'execution:selected',

    // Suite Events
    SUITE_CREATED: 'suite:created',
    SUITE_UPDATED: 'suite:updated',
    SUITE_DELETED: 'suite:deleted',
    SUITE_EXECUTED: 'suite:executed',

    // Execution Events
    EXECUTION_STARTED: 'execution:started',
    EXECUTION_COMPLETED: 'execution:completed',
    EXECUTION_FAILED: 'execution:failed',
    EXECUTION_UPDATED: 'execution:updated',
    EXECUTION_DELETED: 'execution:deleted',

    // WebSocket Events
    WS_CONNECTED: 'websocket:connected',
    WS_DISCONNECTED: 'websocket:disconnected',
    WS_ERROR: 'websocket:error',
    WS_MESSAGE: 'websocket:message',

    // UI Events
    DIALOG_OPENED: 'dialog:opened',
    DIALOG_CLOSED: 'dialog:closed',
    UI_DIALOG_OPEN: 'ui:dialog:open',
    UI_DIALOG_CLOSE: 'ui:dialog:close',
    UI_DIALOG_OPENED: 'ui:dialog:opened',
    UI_DIALOG_CLOSED: 'ui:dialog:closed',
    UI_SUITE_DETAIL_CLOSE: 'ui:suite:detail:close',
    UI_EXECUTION_DETAIL_CLOSE: 'ui:execution:detail:close',
    NOTIFICATION_SHOW: 'notification:show',
    FILTER_CHANGED: 'filter:changed',

    // System Events
    HEALTH_CHANGED: 'system:health:changed',
    HEALTH_CHECK_REQUESTED: 'system:health:check:requested',
    ERROR_OCCURRED: 'system:error',

    // Vault Events
    VAULT_CREATED: 'vault:created',
    VAULT_EXECUTED: 'vault:executed',
    VAULT_UPDATED: 'vault:updated',

    // Coverage Events
    COVERAGE_GENERATED: 'coverage:generated',
    COVERAGE_UPDATED: 'coverage:updated',

    // Reports Events
    REPORTS_LOADED: 'reports:loaded',
    REPORTS_UPDATED: 'reports:updated'
};

/**
 * EventBus class - Central event management system
 */
export class EventBus {
    constructor() {
        // Map of event type -> array of handlers
        this.listeners = new Map();

        // Map of handler -> cleanup info (for debugging and memory management)
        this.handlerMetadata = new Map();

        // Enable debug logging
        this.debug = false;

        // Event history for debugging (last 100 events)
        this.eventHistory = [];
        this.maxHistorySize = 100;
    }

    /**
     * Subscribe to an event
     * @param {string} eventType - Event type to listen for
     * @param {Function} handler - Callback function
     * @param {Object} options - Subscription options
     * @returns {Function} Unsubscribe function
     */
    on(eventType, handler, options = {}) {
        if (!eventType || typeof handler !== 'function') {
            console.warn('EventBus.on: Invalid event type or handler', { eventType, handler });
            return () => {};
        }

        const { once = false, priority = 0, context = null } = options;

        // Wrap handler if needed
        const wrappedHandler = {
            original: handler,
            callback: once ? this._createOnceHandler(eventType, handler) : handler,
            priority,
            context,
            once
        };

        // Get or create listener array for this event type
        if (!this.listeners.has(eventType)) {
            this.listeners.set(eventType, []);
        }

        const listeners = this.listeners.get(eventType);
        listeners.push(wrappedHandler);

        // Sort by priority (higher priority first)
        listeners.sort((a, b) => b.priority - a.priority);

        // Store metadata for debugging
        this.handlerMetadata.set(handler, {
            eventType,
            addedAt: new Date(),
            callCount: 0,
            once
        });

        if (this.debug) {
            console.log(`[EventBus] Subscribed to '${eventType}'`, { priority, once, listenerCount: listeners.length });
        }

        // Return unsubscribe function
        return () => this.off(eventType, handler);
    }

    /**
     * Subscribe to an event, but only trigger once
     * @param {string} eventType - Event type to listen for
     * @param {Function} handler - Callback function
     * @param {Object} options - Subscription options
     * @returns {Function} Unsubscribe function
     */
    once(eventType, handler, options = {}) {
        return this.on(eventType, handler, { ...options, once: true });
    }

    /**
     * Unsubscribe from an event
     * @param {string} eventType - Event type to stop listening for
     * @param {Function} handler - Handler to remove (if null, removes all handlers for event)
     */
    off(eventType, handler = null) {
        if (!this.listeners.has(eventType)) {
            return;
        }

        const listeners = this.listeners.get(eventType);

        if (handler === null) {
            // Remove all listeners for this event type
            const count = listeners.length;
            this.listeners.delete(eventType);
            if (this.debug) {
                console.log(`[EventBus] Removed all ${count} listeners for '${eventType}'`);
            }
        } else {
            // Remove specific handler
            const index = listeners.findIndex(l => l.original === handler);
            if (index !== -1) {
                listeners.splice(index, 1);
                this.handlerMetadata.delete(handler);

                if (this.debug) {
                    console.log(`[EventBus] Unsubscribed from '${eventType}'`, { remainingListeners: listeners.length });
                }

                // Clean up empty listener arrays
                if (listeners.length === 0) {
                    this.listeners.delete(eventType);
                }
            }
        }
    }

    /**
     * Emit an event
     * @param {string} eventType - Event type to emit
     * @param {*} data - Event data/payload
     * @returns {boolean} True if event had listeners
     */
    emit(eventType, data = null) {
        if (!this.listeners.has(eventType)) {
            if (this.debug) {
                console.log(`[EventBus] No listeners for '${eventType}'`);
            }
            return false;
        }

        const listeners = this.listeners.get(eventType);
        const event = {
            type: eventType,
            data,
            timestamp: Date.now(),
            propagationStopped: false
        };

        // Add to history
        this._addToHistory(event);

        if (this.debug) {
            console.log(`[EventBus] Emitting '${eventType}'`, { data, listenerCount: listeners.length });
        }

        // Create a copy of listeners array to avoid mutation issues
        const listenersCopy = [...listeners];
        let propagationStopped = false;

        // Execute all handlers
        for (const listener of listenersCopy) {
            if (propagationStopped) {
                break;
            }

            try {
                // Update metadata
                const metadata = this.handlerMetadata.get(listener.original);
                if (metadata) {
                    metadata.callCount++;
                }

                // Call handler with context if provided
                const eventWithStopPropagation = {
                    ...event,
                    stopPropagation: () => { propagationStopped = true; event.propagationStopped = true; }
                };

                if (listener.context) {
                    listener.callback.call(listener.context, eventWithStopPropagation);
                } else {
                    listener.callback(eventWithStopPropagation);
                }
            } catch (error) {
                console.error(`[EventBus] Error in handler for '${eventType}':`, error);

                // Emit error event (but avoid infinite loops)
                if (eventType !== EVENT_TYPES.ERROR_OCCURRED) {
                    this.emit(EVENT_TYPES.ERROR_OCCURRED, {
                        source: 'EventBus',
                        originalEvent: eventType,
                        error
                    });
                }
            }
        }

        return true;
    }

    /**
     * Check if an event has listeners
     * @param {string} eventType - Event type to check
     * @returns {boolean}
     */
    hasListeners(eventType) {
        return this.listeners.has(eventType) && this.listeners.get(eventType).length > 0;
    }

    /**
     * Get listener count for an event
     * @param {string} eventType - Event type to check
     * @returns {number}
     */
    getListenerCount(eventType) {
        return this.listeners.has(eventType) ? this.listeners.get(eventType).length : 0;
    }

    /**
     * Remove all listeners for all events
     */
    clear() {
        const totalListeners = Array.from(this.listeners.values()).reduce((sum, arr) => sum + arr.length, 0);
        this.listeners.clear();
        this.handlerMetadata.clear();
        this.eventHistory = [];

        if (this.debug) {
            console.log(`[EventBus] Cleared all ${totalListeners} listeners`);
        }
    }

    /**
     * Get all registered event types
     * @returns {Array<string>}
     */
    getEventTypes() {
        return Array.from(this.listeners.keys());
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        const info = {
            totalEventTypes: this.listeners.size,
            totalListeners: 0,
            eventTypes: {},
            recentEvents: this.eventHistory.slice(-10)
        };

        for (const [eventType, listeners] of this.listeners.entries()) {
            info.totalListeners += listeners.length;
            info.eventTypes[eventType] = {
                listenerCount: listeners.length,
                listeners: listeners.map(l => ({
                    priority: l.priority,
                    once: l.once,
                    hasContext: !!l.context,
                    callCount: this.handlerMetadata.get(l.original)?.callCount || 0
                }))
            };
        }

        return info;
    }

    /**
     * Enable/disable debug logging
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[EventBus] Debug logging ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Wait for a specific event (returns promise)
     * @param {string} eventType - Event type to wait for
     * @param {number} timeout - Timeout in milliseconds (0 = no timeout)
     * @returns {Promise}
     */
    waitFor(eventType, timeout = 0) {
        return new Promise((resolve, reject) => {
            let timeoutId = null;

            const unsubscribe = this.once(eventType, (event) => {
                if (timeoutId) clearTimeout(timeoutId);
                resolve(event.data);
            });

            if (timeout > 0) {
                timeoutId = setTimeout(() => {
                    unsubscribe();
                    reject(new Error(`Timeout waiting for event '${eventType}'`));
                }, timeout);
            }
        });
    }

    // Private methods

    /**
     * Create a handler that only fires once
     * @private
     */
    _createOnceHandler(eventType, handler) {
        return (event) => {
            handler(event);
            // Remove after execution
            this.off(eventType, handler);
        };
    }

    /**
     * Add event to history for debugging
     * @private
     */
    _addToHistory(event) {
        this.eventHistory.push({
            type: event.type,
            timestamp: event.timestamp,
            hasData: event.data !== null
        });

        // Trim history if too large
        if (this.eventHistory.length > this.maxHistorySize) {
            this.eventHistory.shift();
        }
    }
}

// Export singleton instance
export const eventBus = new EventBus();

// Export default for convenience
export default eventBus;
