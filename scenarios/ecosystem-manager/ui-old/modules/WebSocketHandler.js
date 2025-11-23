// WebSocket Handler Module
import { logger } from '../utils/logger.js';

export class WebSocketHandler {
    constructor(wsBase, onMessage) {
        this.wsBase = wsBase;
        this.onMessage = onMessage;
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
    }

    connect() {
        const wsUrl = this.wsBase || this.buildFallbackUrl();
        if (!wsUrl) {
            logger.error('WebSocket URL unavailable');
            return;
        }

        logger.ws('Connecting to WebSocket:', wsUrl);

        try {
            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                logger.ws('WebSocket connected');
                this.reconnectAttempts = 0;
                this.reconnectDelay = 1000;
            };

            this.ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    if (this.onMessage) {
                        this.onMessage(message);
                    }
                } catch (error) {
                    logger.error('Failed to parse WebSocket message:', error);
                }
            };

            this.ws.onerror = (error) => {
                logger.error('WebSocket error:', error);
            };

            this.ws.onclose = () => {
                logger.ws('WebSocket disconnected');
                this.attemptReconnect();
            };
        } catch (error) {
            logger.error('Failed to create WebSocket:', error);
            this.attemptReconnect();
        }
    }

    buildFallbackUrl() {
        if (typeof window === 'undefined' || !window.location) {
            return null;
        }

        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        return `${wsProtocol}//${window.location.host}/ws`;
    }

    attemptReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            logger.error('Max reconnection attempts reached');
            return;
        }

        this.reconnectAttempts++;
        logger.ws(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts}) in ${this.reconnectDelay}ms...`);

        setTimeout(() => {
            this.connect();
            this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000); // Exponential backoff, max 30 seconds
        }, this.reconnectDelay);
    }

    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        } else {
            logger.error('WebSocket is not connected');
        }
    }

    close() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }
}
