import { useEffect, useCallback, useRef } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import type { WSMessage, App, SystemMetrics, LogEntry } from '@/types';

interface WebSocketHookOptions {
  onAppUpdate?: (app: Partial<App>) => void;
  onMetricUpdate?: (metrics: SystemMetrics) => void;
  onLogEntry?: (log: LogEntry) => void;
  onConnection?: (status: boolean) => void;
  onError?: (error: string) => void;
  reconnectInterval?: number;
  reconnectAttempts?: number;
}

export function useAppWebSocket(options: WebSocketHookOptions = {}) {
  const {
    onAppUpdate,
    onMetricUpdate,
    onLogEntry,
    onConnection,
    onError,
    reconnectInterval = 3000,
    reconnectAttempts = 5,
  } = options;

  const messageHistory = useRef<WSMessage[]>([]);
  const reconnectCount = useRef(0);

  // Determine WebSocket URL based on environment
  const getSocketUrl = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    
    // In development, connect to Express server port
    if (import.meta.env.DEV) {
      // Use the UI_PORT from environment or fall back to default
      const serverPort = import.meta.env.VITE_UI_PORT || import.meta.env.UI_PORT || '8085';
      return `${protocol}//localhost:${serverPort}/ws`;
    }
    
    // In production, use the same host
    return `${protocol}//${host}/ws`;
  }, []);

  const {
    sendMessage,
    sendJsonMessage,
    lastMessage,
    lastJsonMessage,
    readyState,
    getWebSocket,
  } = useWebSocket(getSocketUrl(), {
    onOpen: () => {
      console.log('[WebSocket] Connected');
      reconnectCount.current = 0;
      onConnection?.(true);
    },
    onClose: () => {
      console.log('[WebSocket] Disconnected');
      onConnection?.(false);
    },
    onError: (error) => {
      console.error('[WebSocket] Error:', error);
      onError?.('WebSocket connection error');
    },
    shouldReconnect: () => {
      // Reconnect unless we've exceeded max attempts
      if (reconnectCount.current >= reconnectAttempts) {
        console.error('[WebSocket] Max reconnection attempts reached');
        onError?.('Failed to reconnect to server');
        return false;
      }
      reconnectCount.current++;
      console.log(`[WebSocket] Reconnecting... (attempt ${reconnectCount.current}/${reconnectAttempts})`);
      return true;
    },
    reconnectInterval,
    share: true, // Share connection between components
  });

  // Handle incoming messages
  useEffect(() => {
    if (lastJsonMessage !== null) {
      const message = lastJsonMessage as WSMessage;
      
      // Store message in history
      messageHistory.current = [...messageHistory.current.slice(-99), message];
      
      // Debug log
      console.debug('[WebSocket] Message received:', message.type, message.payload);
      
      // Route message to appropriate handler
      switch (message.type) {
        case 'app_update': {
          if (message.payload && typeof message.payload === 'object') {
            onAppUpdate?.(message.payload as Partial<App>);
          }
          break;
        }

        case 'metric_update': {
          if (message.payload && typeof message.payload === 'object') {
            onMetricUpdate?.(message.payload as SystemMetrics);
          }
          break;
        }

        case 'log_entry': {
          if (message.payload && typeof message.payload === 'object') {
            onLogEntry?.(message.payload as LogEntry);
          }
          break;
        }

        case 'connection': {
          console.log('[WebSocket] Connection status:', message.payload);
          break;
        }

        case 'error': {
          const payload = message.payload;
          console.error('[WebSocket] Server error:', payload);
          const messageText = (typeof payload === 'object' && payload && 'message' in payload && typeof (payload as { message: unknown }).message === 'string')
            ? (payload as { message: string }).message
            : 'Server error';
          onError?.(messageText);
          break;
        }

        default:
          console.warn('[WebSocket] Unknown message type:', message.type);
      }
    }
  }, [lastJsonMessage, onAppUpdate, onMetricUpdate, onLogEntry, onError]);

  // Subscribe to app updates
  const subscribeToApp = useCallback((appId: string) => {
    if (readyState === ReadyState.OPEN) {
      sendJsonMessage({
        type: 'subscribe',
        appId,
      });
      console.log(`[WebSocket] Subscribed to app: ${appId}`);
    }
  }, [readyState, sendJsonMessage]);

  // Unsubscribe from app updates
  const unsubscribeFromApp = useCallback((appId: string) => {
    if (readyState === ReadyState.OPEN) {
      sendJsonMessage({
        type: 'unsubscribe',
        appId,
      });
      console.log(`[WebSocket] Unsubscribed from app: ${appId}`);
    }
  }, [readyState, sendJsonMessage]);

  // Send command
  const sendCommand = useCallback((command: string) => {
    if (readyState === ReadyState.OPEN) {
      sendJsonMessage({
        type: 'command',
        command,
      });
      console.log(`[WebSocket] Sent command: ${command}`);
    } else {
      console.warn('[WebSocket] Cannot send command - not connected');
      onError?.('WebSocket not connected');
    }
  }, [readyState, sendJsonMessage, onError]);

  // Connection state helpers
  const isConnected = readyState === ReadyState.OPEN;
  const isConnecting = readyState === ReadyState.CONNECTING;
  const connectionState = {
    [ReadyState.CONNECTING]: 'Connecting',
    [ReadyState.OPEN]: 'Connected',
    [ReadyState.CLOSING]: 'Closing',
    [ReadyState.CLOSED]: 'Disconnected',
    [ReadyState.UNINSTANTIATED]: 'Uninstantiated',
  }[readyState];

  return {
    // State
    isConnected,
    isConnecting,
    connectionState,
    readyState,
    messageHistory: messageHistory.current,
    
    // Methods
    subscribeToApp,
    unsubscribeFromApp,
    sendCommand,
    sendMessage,
    sendJsonMessage,
    
    // Raw access
    getWebSocket,
    lastMessage,
    lastJsonMessage,
  };
}

export default useAppWebSocket;
