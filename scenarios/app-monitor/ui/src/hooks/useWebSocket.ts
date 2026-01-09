import { useEffect, useCallback, useRef } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import type { WSMessage, App, SystemMetrics, LogEntry } from '@/types';
import { logger } from '@/services/logger';
import { resolveWsBase } from '@vrooli/api-base';

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

  // Determine WebSocket URL using api-base
  const getSocketUrl = useCallback(() => {
    return resolveWsBase({
      appendSuffix: true,
      apiSuffix: '/ws',
    });
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
      logger.info('[WebSocket] Connected');
      reconnectCount.current = 0;
      onConnection?.(true);
    },
    onClose: () => {
      logger.info('[WebSocket] Disconnected');
      onConnection?.(false);
    },
    onError: (error) => {
      logger.error('[WebSocket] Error', error);
      onError?.('WebSocket connection error');
    },
    shouldReconnect: () => {
      // Reconnect unless we've exceeded max attempts
      if (reconnectCount.current >= reconnectAttempts) {
        logger.error('[WebSocket] Max reconnection attempts reached');
        onError?.('Failed to reconnect to server');
        return false;
      }
      reconnectCount.current++;
      logger.info(`[WebSocket] Reconnecting... (attempt ${reconnectCount.current}/${reconnectAttempts})`);
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
      logger.debug('[WebSocket] Message received', { type: message.type, payload: message.payload });

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
          logger.info('[WebSocket] Connection status', message.payload);
          break;
        }

        case 'error': {
          const payload = message.payload;
          logger.error('[WebSocket] Server error', payload);
          const messageText = (typeof payload === 'object' && payload && 'message' in payload && typeof (payload as { message: unknown }).message === 'string')
            ? (payload as { message: string }).message
            : 'Server error';
          onError?.(messageText);
          break;
        }

        default:
          logger.warn('[WebSocket] Unknown message type', { type: message.type });
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
      logger.debug(`[WebSocket] Subscribed to app: ${appId}`);
    }
  }, [readyState, sendJsonMessage]);

  // Unsubscribe from app updates
  const unsubscribeFromApp = useCallback((appId: string) => {
    if (readyState === ReadyState.OPEN) {
      sendJsonMessage({
        type: 'unsubscribe',
        appId,
      });
      logger.debug(`[WebSocket] Unsubscribed from app: ${appId}`);
    }
  }, [readyState, sendJsonMessage]);

  // Send command
  const sendCommand = useCallback((command: string) => {
    if (readyState === ReadyState.OPEN) {
      sendJsonMessage({
        type: 'command',
        command,
      });
      logger.debug(`[WebSocket] Sent command: ${command}`);
    } else {
      logger.warn('[WebSocket] Cannot send command - not connected');
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
