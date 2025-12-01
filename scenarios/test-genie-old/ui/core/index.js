/**
 * Core Module Index
 * Central export point for all core data layer modules
 */

// EventBus - Pub/sub event system
export { EventBus, eventBus, EVENT_TYPES } from './EventBus.js';

// StateManager - Centralized state management
export { StateManager, stateManager } from './StateManager.js';

// ApiClient - HTTP API communication
export { ApiClient, apiClient } from './ApiClient.js';

// WebSocketClient - Real-time bidirectional communication
export { WebSocketClient, wsClient, WS_MESSAGE_TYPES } from './WebSocketClient.js';

// Export default object with all singletons
export default {
    eventBus,
    stateManager,
    apiClient,
    wsClient
};
