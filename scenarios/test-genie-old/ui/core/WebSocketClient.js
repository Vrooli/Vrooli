/**
 * WebSocketClient - Real-time Communication Layer
 * Handles WebSocket connections with automatic reconnection and event broadcasting
 */

import { eventBus, EVENT_TYPES } from './EventBus.js';
import { TIMING } from '../utils/constants.js';

/**
 * WebSocket message types
 */
export const WS_MESSAGE_TYPES = {
    SUBSCRIBE: 'subscribe',
    UNSUBSCRIBE: 'unsubscribe',
    EXECUTION_UPDATE: 'execution_update',
    TEST_COMPLETION: 'test_completion',
    SYSTEM_STATUS: 'system_status',
    VAULT_UPDATE: 'vault_update',
    COVERAGE_UPDATE: 'coverage_update'
};

/**
 * WebSocketClient class - Real-time bidirectional communication
 */
export class WebSocketClient {
    constructor(eventBusInstance = eventBus) {
        this.eventBus = eventBusInstance;
        this.connection = null;
        this.url = null;

        // Connection state
        this.isConnected = false;
        this.isConnecting = false;
        this.shouldReconnect = true;

        // Reconnection configuration
        this.reconnectDelay = TIMING.WEBSOCKET_RECONNECT_DELAY;
        this.maxReconnectDelay = 30000; // 30 seconds
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;

        // Subscribed topics
        this.subscribedTopics = new Set();
        this.defaultTopics = ['system_status', 'executions', 'test_completion'];

        // Message queue for when disconnected
        this.messageQueue = [];
        this.maxQueueSize = 100;

        // Heartbeat/ping configuration
        this.heartbeatInterval = null;
        this.heartbeatTimeout = 30000; // 30 seconds
        this.lastHeartbeat = null;

        // Debug mode
        this.debug = false;

        // Metrics
        this.metrics = {
            messagesReceived: 0,
            messagesSent: 0,
            reconnections: 0,
            errors: 0,
            lastConnectedAt: null,
            lastDisconnectedAt: null
        };
    }

    /**
     * Connect to WebSocket server
     * @param {string} url - WebSocket URL (optional, auto-detects if not provided)
     * @returns {Promise<void>}
     */
    async connect(url = null) {
        if (this.isConnected || this.isConnecting) {
            if (this.debug) {
                console.log('[WebSocketClient] Already connected or connecting');
            }
            return;
        }

        // Auto-detect WebSocket URL if not provided
        if (!url) {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            url = `${protocol}//${window.location.host}/ws`;
        }

        this.url = url;
        this.isConnecting = true;

        try {
            if (this.debug) {
                console.log(`[WebSocketClient] Connecting to ${url}`);
            }

            this.connection = new WebSocket(url);

            // Set up event handlers
            this.connection.onopen = this._handleOpen.bind(this);
            this.connection.onmessage = this._handleMessage.bind(this);
            this.connection.onclose = this._handleClose.bind(this);
            this.connection.onerror = this._handleError.bind(this);

            // Wait for connection to open
            await new Promise((resolve, reject) => {
                const timeout = setTimeout(() => {
                    reject(new Error('WebSocket connection timeout'));
                }, 5000);

                this.connection.addEventListener('open', () => {
                    clearTimeout(timeout);
                    resolve();
                }, { once: true });

                this.connection.addEventListener('error', (error) => {
                    clearTimeout(timeout);
                    reject(error);
                }, { once: true });
            });

        } catch (error) {
            this.isConnecting = false;
            console.error('[WebSocketClient] Connection failed:', error);
            this.eventBus.emit(EVENT_TYPES.WS_ERROR, { error: error.message });

            // Attempt reconnection
            if (this.shouldReconnect) {
                this._scheduleReconnect();
            }

            throw error;
        }
    }

    /**
     * Disconnect from WebSocket server
     */
    disconnect() {
        if (this.debug) {
            console.log('[WebSocketClient] Disconnecting');
        }

        this.shouldReconnect = false;

        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }

        if (this.connection) {
            this.connection.close();
            this.connection = null;
        }

        this.isConnected = false;
        this.isConnecting = false;
    }

    /**
     * Send message to server
     * @param {Object} message - Message object
     * @returns {boolean} Success
     */
    send(message) {
        if (!this.isConnected) {
            if (this.debug) {
                console.warn('[WebSocketClient] Not connected, queueing message');
            }

            // Queue message for later
            this._queueMessage(message);
            return false;
        }

        try {
            const jsonMessage = JSON.stringify(message);
            this.connection.send(jsonMessage);
            this.metrics.messagesSent++;

            if (this.debug) {
                console.log('[WebSocketClient] Sent message:', message);
            }

            return true;
        } catch (error) {
            console.error('[WebSocketClient] Send error:', error);
            this.metrics.errors++;
            return false;
        }
    }

    /**
     * Subscribe to topics
     * @param {Array<string>} topics - Topics to subscribe to
     */
    subscribe(topics) {
        const topicsArray = Array.isArray(topics) ? topics : [topics];

        topicsArray.forEach(topic => {
            this.subscribedTopics.add(topic);
        });

        if (this.isConnected) {
            this.send({
                type: WS_MESSAGE_TYPES.SUBSCRIBE,
                payload: {
                    topics: topicsArray
                }
            });
        }

        if (this.debug) {
            console.log('[WebSocketClient] Subscribed to topics:', topicsArray);
        }
    }

    /**
     * Unsubscribe from topics
     * @param {Array<string>} topics - Topics to unsubscribe from
     */
    unsubscribe(topics) {
        const topicsArray = Array.isArray(topics) ? topics : [topics];

        topicsArray.forEach(topic => {
            this.subscribedTopics.delete(topic);
        });

        if (this.isConnected) {
            this.send({
                type: WS_MESSAGE_TYPES.UNSUBSCRIBE,
                payload: {
                    topics: topicsArray
                }
            });
        }

        if (this.debug) {
            console.log('[WebSocketClient] Unsubscribed from topics:', topicsArray);
        }
    }

    /**
     * Check if connected
     * @returns {boolean}
     */
    isReady() {
        return this.isConnected && this.connection && this.connection.readyState === WebSocket.OPEN;
    }

    /**
     * Get connection metrics
     * @returns {Object}
     */
    getMetrics() {
        return {
            ...this.metrics,
            isConnected: this.isConnected,
            subscribedTopics: Array.from(this.subscribedTopics),
            queuedMessages: this.messageQueue.length
        };
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[WebSocketClient] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    // Event handlers

    /**
     * Handle connection opened
     * @private
     */
    _handleOpen() {
        console.log('[WebSocketClient] Connection established');

        this.isConnected = true;
        this.isConnecting = false;
        this.reconnectAttempts = 0;
        this.metrics.lastConnectedAt = new Date();

        // Start heartbeat
        this._startHeartbeat();

        // Subscribe to default topics
        this.subscribe([...this.defaultTopics, ...Array.from(this.subscribedTopics)]);

        // Process queued messages
        this._processMessageQueue();

        // Emit connection event
        this.eventBus.emit(EVENT_TYPES.WS_CONNECTED, {
            url: this.url,
            reconnected: this.reconnectAttempts > 0
        });
    }

    /**
     * Handle message received
     * @private
     */
    _handleMessage(event) {
        this.metrics.messagesReceived++;
        this.lastHeartbeat = Date.now();

        try {
            const data = JSON.parse(event.data);

            if (this.debug) {
                console.log('[WebSocketClient] Received message:', data);
            }

            // Emit generic WebSocket message event
            this.eventBus.emit(EVENT_TYPES.WS_MESSAGE, data);

            // Handle specific message types
            this._handleMessageType(data);

        } catch (error) {
            console.error('[WebSocketClient] Message parse error:', error);
            this.metrics.errors++;
        }
    }

    /**
     * Handle connection closed
     * @private
     */
    _handleClose(event) {
        console.log('[WebSocketClient] Connection closed', {
            code: event.code,
            reason: event.reason,
            clean: event.wasClean
        });

        this.isConnected = false;
        this.isConnecting = false;
        this.metrics.lastDisconnectedAt = new Date();

        // Stop heartbeat
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }

        // Emit disconnection event
        this.eventBus.emit(EVENT_TYPES.WS_DISCONNECTED, {
            code: event.code,
            reason: event.reason
        });

        // Attempt reconnection if not intentionally closed
        if (this.shouldReconnect && event.code !== 1000) {
            this._scheduleReconnect();
        }
    }

    /**
     * Handle connection error
     * @private
     */
    _handleError(error) {
        console.error('[WebSocketClient] Connection error:', error);
        this.metrics.errors++;
        this.eventBus.emit(EVENT_TYPES.WS_ERROR, { error });
    }

    /**
     * Handle specific message types
     * @private
     */
    _handleMessageType(data) {
        const { type, payload } = data;

        switch (type) {
            case WS_MESSAGE_TYPES.EXECUTION_UPDATE:
                this.eventBus.emit(EVENT_TYPES.EXECUTION_UPDATED, payload);
                break;

            case WS_MESSAGE_TYPES.TEST_COMPLETION:
                this.eventBus.emit(EVENT_TYPES.EXECUTION_COMPLETED, payload);
                break;

            case WS_MESSAGE_TYPES.SYSTEM_STATUS:
                this.eventBus.emit(EVENT_TYPES.HEALTH_CHANGED, payload);
                break;

            case WS_MESSAGE_TYPES.VAULT_UPDATE:
                this.eventBus.emit(EVENT_TYPES.VAULT_UPDATED, payload);
                break;

            case WS_MESSAGE_TYPES.COVERAGE_UPDATE:
                this.eventBus.emit(EVENT_TYPES.COVERAGE_UPDATED, payload);
                break;

            default:
                if (this.debug) {
                    console.log('[WebSocketClient] Unknown message type:', type);
                }
        }
    }

    /**
     * Schedule reconnection attempt
     * @private
     */
    _scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('[WebSocketClient] Max reconnection attempts reached');
            return;
        }

        this.reconnectAttempts++;
        this.metrics.reconnections++;

        // Exponential backoff with max delay
        const delay = Math.min(
            this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
            this.maxReconnectDelay
        );

        if (this.debug) {
            console.log(`[WebSocketClient] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
        }

        setTimeout(() => {
            this.connect(this.url).catch(error => {
                console.error('[WebSocketClient] Reconnection failed:', error);
            });
        }, delay);
    }

    /**
     * Queue message for later sending
     * @private
     */
    _queueMessage(message) {
        if (this.messageQueue.length >= this.maxQueueSize) {
            // Remove oldest message
            this.messageQueue.shift();
        }
        this.messageQueue.push(message);
    }

    /**
     * Process queued messages
     * @private
     */
    _processMessageQueue() {
        if (this.messageQueue.length === 0) {
            return;
        }

        if (this.debug) {
            console.log(`[WebSocketClient] Processing ${this.messageQueue.length} queued messages`);
        }

        while (this.messageQueue.length > 0) {
            const message = this.messageQueue.shift();
            this.send(message);
        }
    }

    /**
     * Start heartbeat/ping mechanism
     * @private
     */
    _startHeartbeat() {
        this.lastHeartbeat = Date.now();

        this.heartbeatInterval = setInterval(() => {
            if (!this.isConnected) {
                return;
            }

            const timeSinceLastHeartbeat = Date.now() - this.lastHeartbeat;

            if (timeSinceLastHeartbeat > this.heartbeatTimeout) {
                console.warn('[WebSocketClient] Heartbeat timeout, reconnecting...');
                this.connection.close();
            } else {
                // Send ping
                this.send({ type: 'ping', timestamp: Date.now() });
            }
        }, this.heartbeatTimeout / 2);
    }
}

// Export singleton instance
export const wsClient = new WebSocketClient();

// Export default for convenience
export default wsClient;
