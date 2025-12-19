/**
 * useExecutionFrameStream Hook
 *
 * Subscribes to live frame streaming during workflow execution.
 * Uses WebSocket to receive execution frames in real-time.
 *
 * Usage:
 *   const { frame, isStreaming, error } = useExecutionFrameStream(executionId);
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { logger } from '@/utils/logger';

export interface ExecutionFrame {
  /** Base64 encoded image data */
  data: string;
  /** Media type (e.g., "image/jpeg") */
  mediaType: string;
  /** Frame width in pixels */
  width: number;
  /** Frame height in pixels */
  height: number;
  /** ISO timestamp when frame was captured */
  capturedAt: string;
}

interface UseExecutionFrameStreamOptions {
  /** Whether streaming is enabled. Pass false to disable subscription. */
  enabled?: boolean;
  /** Callback when a new frame is received */
  onFrame?: (frame: ExecutionFrame) => void;
}

interface UseExecutionFrameStreamReturn {
  /** The latest frame as a data URL (e.g., "data:image/jpeg;base64,...") */
  frameUrl: string | null;
  /** The latest frame metadata */
  frame: ExecutionFrame | null;
  /** Whether currently streaming */
  isStreaming: boolean;
  /** Whether subscribed to execution frames */
  isSubscribed: boolean;
  /** Frame count received since subscription */
  frameCount: number;
  /** Error message if any */
  error: string | null;
  /** Manually unsubscribe from frame streaming */
  unsubscribe: () => void;
}

export function useExecutionFrameStream(
  executionId: string | null,
  options: UseExecutionFrameStreamOptions = {}
): UseExecutionFrameStreamReturn {
  const { enabled = true, onFrame } = options;

  const [frameUrl, setFrameUrl] = useState<string | null>(null);
  const [frame, setFrame] = useState<ExecutionFrame | null>(null);
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [isStreaming, setIsStreaming] = useState(false);
  const [frameCount, setFrameCount] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const { isConnected, lastMessage, send } = useWebSocket();
  const subscribedIdRef = useRef<string | null>(null);
  const onFrameRef = useRef(onFrame);
  onFrameRef.current = onFrame;

  // Subscribe to execution frame streaming
  const subscribe = useCallback((execId: string) => {
    if (!isConnected) {
      logger.debug('Cannot subscribe - WebSocket not connected', {
        component: 'useExecutionFrameStream',
        action: 'subscribe',
        executionId: execId,
      });
      return;
    }

    logger.debug('Subscribing to execution frames', {
      component: 'useExecutionFrameStream',
      action: 'subscribe',
      executionId: execId,
    });

    send({ type: 'subscribe_execution_frames', execution_id: execId });
    subscribedIdRef.current = execId;
    setIsSubscribed(true);
    setError(null);
    setFrameCount(0);
  }, [isConnected, send]);

  // Unsubscribe from execution frame streaming
  const unsubscribe = useCallback(() => {
    if (subscribedIdRef.current) {
      logger.debug('Unsubscribing from execution frames', {
        component: 'useExecutionFrameStream',
        action: 'unsubscribe',
        executionId: subscribedIdRef.current,
      });

      send({ type: 'unsubscribe_execution_frames' });
      subscribedIdRef.current = null;
      setIsSubscribed(false);
      setIsStreaming(false);
    }
  }, [send]);

  // Handle subscription changes
  useEffect(() => {
    if (!enabled || !executionId) {
      // Unsubscribe if disabled or no execution ID
      if (subscribedIdRef.current) {
        unsubscribe();
      }
      return;
    }

    // Subscribe if not already subscribed to this execution
    if (subscribedIdRef.current !== executionId && isConnected) {
      // Unsubscribe from previous execution if any
      if (subscribedIdRef.current) {
        unsubscribe();
      }
      subscribe(executionId);
    }

    return () => {
      // Cleanup on unmount
      if (subscribedIdRef.current) {
        unsubscribe();
      }
    };
  }, [executionId, enabled, isConnected, subscribe, unsubscribe]);

  // Handle WebSocket messages
  useEffect(() => {
    if (!lastMessage || !isSubscribed) return;

    // Handle subscription confirmation
    if (lastMessage.type === 'execution_frame_subscribed') {
      logger.debug('Subscription confirmed', {
        component: 'useExecutionFrameStream',
        action: 'handleMessage',
        executionId: lastMessage.execution_id,
      });
      setIsStreaming(true);
      return;
    }

    // Handle execution frames
    if (lastMessage.type === 'execution_frame') {
      const msg = lastMessage as unknown as {
        execution_id: string;
        data: string;
        media_type: string;
        width: number;
        height: number;
        captured_at: string;
      };

      // Only process frames for our subscribed execution
      if (msg.execution_id !== subscribedIdRef.current) return;

      const newFrame: ExecutionFrame = {
        data: msg.data,
        mediaType: msg.media_type || 'image/jpeg',
        width: msg.width,
        height: msg.height,
        capturedAt: msg.captured_at,
      };

      // Build data URL for direct use in <img> src
      const dataUrl = `data:${newFrame.mediaType};base64,${newFrame.data}`;

      setFrame(newFrame);
      setFrameUrl(dataUrl);
      setFrameCount((prev) => prev + 1);
      setIsStreaming(true);

      // Invoke callback if provided
      if (onFrameRef.current) {
        onFrameRef.current(newFrame);
      }
    }
  }, [lastMessage, isSubscribed]);

  // Reset state when execution changes
  useEffect(() => {
    setFrameUrl(null);
    setFrame(null);
    setFrameCount(0);
    setIsStreaming(false);
    setError(null);
  }, [executionId]);

  return {
    frameUrl,
    frame,
    isStreaming,
    isSubscribed,
    frameCount,
    error,
    unsubscribe,
  };
}

export default useExecutionFrameStream;
