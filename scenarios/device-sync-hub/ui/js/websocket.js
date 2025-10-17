// WebSocket connection manager for Device Sync Hub

class WebSocketManager {
  constructor(apiConfig, authManager, deviceManager) {
    this.wsUrl = this.getWebSocketUrl(apiConfig.apiUrl);
    this.authManager = authManager;
    this.deviceManager = deviceManager;
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.connectionStatus = 'offline';
    this.messageHandlers = new Map();
    this.onStatusChange = null;
  }

  // Convert HTTP URL to WebSocket URL
  getWebSocketUrl(apiUrl) {
    return apiUrl.replace(/^http/, 'ws');
  }

  // Set connection status change handler
  setStatusChangeHandler(handler) {
    this.onStatusChange = handler;
  }

  // Update connection status
  updateStatus(status) {
    this.connectionStatus = status;
    if (this.onStatusChange) {
      this.onStatusChange(status);
    }
  }

  // Get current connection status
  getStatus() {
    return this.connectionStatus;
  }

  // Add message handler for specific message types
  addMessageHandler(messageType, handler) {
    if (!this.messageHandlers.has(messageType)) {
      this.messageHandlers.set(messageType, []);
    }
    this.messageHandlers.get(messageType).push(handler);
  }

  // Remove message handler
  removeMessageHandler(messageType, handler) {
    if (this.messageHandlers.has(messageType)) {
      const handlers = this.messageHandlers.get(messageType);
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    }
  }

  // Initialize WebSocket connection
  connect() {
    if (!this.authManager.isAuthenticated()) {
      console.warn('Cannot connect WebSocket: not authenticated');
      return;
    }

    this.updateStatus('connecting');
    
    // Build WebSocket URL with authentication
    const deviceId = this.deviceManager.getDeviceId();
    const token = this.authManager.getToken();
    const wsUrl = `${this.wsUrl}/api/v1/sync/websocket?token=${encodeURIComponent(token)}&device_id=${encodeURIComponent(deviceId)}`;
    
    try {
      this.ws = new WebSocket(wsUrl);
      this.setupEventHandlers();
    } catch (error) {
      console.error('WebSocket connection error:', error);
      this.updateStatus('error');
      this.scheduleReconnect();
    }
  }

  // Setup WebSocket event handlers
  setupEventHandlers() {
    if (!this.ws) return;

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.updateStatus('online');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        this.handleMessage(message);
      } catch (error) {
        console.error('WebSocket message parse error:', error);
      }
    };

    this.ws.onclose = (event) => {
      console.log('WebSocket disconnected:', event.code, event.reason);
      this.updateStatus('offline');
      
      // Don't reconnect if it was a clean close
      if (event.code !== 1000) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.updateStatus('error');
    };
  }

  // Handle incoming WebSocket messages
  handleMessage(message) {
    const messageType = message.type;
    
    if (this.messageHandlers.has(messageType)) {
      const handlers = this.messageHandlers.get(messageType);
      handlers.forEach(handler => {
        try {
          handler(message);
        } catch (error) {
          console.error(`Error in WebSocket message handler for ${messageType}:`, error);
        }
      });
    } else {
      console.log('Unhandled WebSocket message type:', messageType, message);
    }
  }

  // Send message through WebSocket
  send(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
      return true;
    } else {
      console.warn('WebSocket not connected, cannot send message:', message);
      return false;
    }
  }

  // Schedule reconnection attempt
  scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      this.updateStatus('failed');
      return;
    }

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;
    
    console.log(`Scheduling WebSocket reconnection attempt ${this.reconnectAttempts} in ${delay}ms`);
    
    setTimeout(() => {
      if (this.authManager.isAuthenticated()) {
        this.connect();
      }
    }, delay);
  }

  // Disconnect WebSocket
  disconnect() {
    if (this.ws) {
      this.ws.close(1000, 'User disconnect');
      this.ws = null;
    }
    this.updateStatus('offline');
  }

  // Check if WebSocket is connected
  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN;
  }
}

// Export for module use
window.WebSocketManager = WebSocketManager;