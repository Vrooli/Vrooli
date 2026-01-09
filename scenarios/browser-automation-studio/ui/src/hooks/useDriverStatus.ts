/**
 * useDriverStatus Hook
 *
 * Subscribes to real-time playwright-driver health status updates via WebSocket.
 * This enables the UI to show the driver's current state (healthy, degraded, unhealthy, etc.)
 * and estimated recovery time when the driver is unavailable.
 */

import { useEffect, useState } from 'react';
import { useWebSocket } from '../contexts/WebSocketContext';
import { logger } from '../utils/logger';

/** Driver health status values */
export type DriverHealthStatus =
  | 'healthy'
  | 'degraded'
  | 'unhealthy'
  | 'restarting'
  | 'unrecoverable'
  | 'unknown';

/** Circuit breaker state */
export type CircuitBreakerState = 'open' | 'closed' | 'half-open' | 'disabled';

/** Driver health information from the API */
export interface DriverHealth {
  /** Current health status */
  status: DriverHealthStatus;
  /** Circuit breaker state */
  circuitBreaker: CircuitBreakerState;
  /** Number of active browser sessions */
  activeSessions: number;
  /** Number of restarts since API started */
  restartCount: number;
  /** Uptime in milliseconds */
  uptimeMs: number;
  /** Last error message, if any */
  lastError?: string;
  /** Estimated time until recovery, in milliseconds */
  estimatedRecoveryMs?: number;
  /** Timestamp of this health update */
  updatedAt: string;
}

/** Hook return value */
export interface UseDriverStatusResult {
  /** Current driver health, null if not yet received */
  health: DriverHealth | null;
  /** Whether WebSocket is connected */
  isConnected: boolean;
  /** Whether subscribed to driver status updates */
  isSubscribed: boolean;
  /** Force a reconnection attempt */
  reconnect: () => void;
}

/**
 * Hook to subscribe to driver status updates.
 *
 * @example
 * ```tsx
 * function DriverIndicator() {
 *   const { health, isConnected } = useDriverStatus();
 *
 *   if (!health) return <span>Loading...</span>;
 *
 *   return (
 *     <span className={health.status === 'healthy' ? 'text-green' : 'text-red'}>
 *       {health.status}
 *     </span>
 *   );
 * }
 * ```
 */
export function useDriverStatus(): UseDriverStatusResult {
  const { isConnected, lastMessage, send, reconnect } = useWebSocket();
  const [health, setHealth] = useState<DriverHealth | null>(null);
  const [isSubscribed, setIsSubscribed] = useState(false);

  // Subscribe when connected
  useEffect(() => {
    if (isConnected && !isSubscribed) {
      logger.debug('Subscribing to driver status', {
        component: 'useDriverStatus',
        action: 'subscribe',
      });
      send({ type: 'subscribe_driver_status' });
    }

    // Unsubscribe on disconnect
    if (!isConnected) {
      setIsSubscribed(false);
    }
  }, [isConnected, isSubscribed, send]);

  // Handle incoming messages
  useEffect(() => {
    if (!lastMessage) return;

    // Handle subscription confirmation
    if (lastMessage.type === 'driver_status_subscribed') {
      logger.debug('Driver status subscription confirmed', {
        component: 'useDriverStatus',
        action: 'subscribed',
      });
      setIsSubscribed(true);
      return;
    }

    // Handle status updates
    if (lastMessage.type === 'driver_status') {
      const data = lastMessage as unknown as Record<string, unknown>;
      const newHealth: DriverHealth = {
        status: (data.status as DriverHealthStatus) || 'unknown',
        circuitBreaker: (data.circuit_breaker as CircuitBreakerState) || 'closed',
        activeSessions: (data.active_sessions as number) || 0,
        restartCount: (data.restart_count as number) || 0,
        uptimeMs: (data.uptime_ms as number) || 0,
        lastError: data.last_error as string | undefined,
        estimatedRecoveryMs: data.estimated_recovery_ms as number | undefined,
        updatedAt: (data.updated_at as string) || new Date().toISOString(),
      };

      setHealth(newHealth);

      logger.debug('Driver status updated', {
        component: 'useDriverStatus',
        action: 'update',
        status: newHealth.status,
        restartCount: newHealth.restartCount,
      });
    }
  }, [lastMessage]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (isConnected) {
        send({ type: 'unsubscribe_driver_status' });
      }
    };
  }, [isConnected, send]);

  return {
    health,
    isConnected,
    isSubscribed,
    reconnect,
  };
}

/**
 * Get a human-readable label for a driver status.
 */
export function getStatusLabel(status: DriverHealthStatus): string {
  switch (status) {
    case 'healthy':
      return 'Connected';
    case 'degraded':
      return 'Degraded';
    case 'unhealthy':
      return 'Disconnected';
    case 'restarting':
      return 'Restarting...';
    case 'unrecoverable':
      return 'Failed';
    default:
      return 'Unknown';
  }
}

/**
 * Get a color class for a driver status.
 */
export function getStatusColor(status: DriverHealthStatus): string {
  switch (status) {
    case 'healthy':
      return 'text-green-500';
    case 'degraded':
      return 'text-yellow-500';
    case 'unhealthy':
      return 'text-red-500';
    case 'restarting':
      return 'text-yellow-500';
    case 'unrecoverable':
      return 'text-red-600';
    default:
      return 'text-gray-500';
  }
}

/**
 * Format milliseconds as a human-readable duration.
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${Math.round(ms / 1000)}s`;
  if (ms < 3600000) return `${Math.round(ms / 60000)}m`;
  return `${Math.round(ms / 3600000)}h`;
}
