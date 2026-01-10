/**
 * Hook for subscribing to export progress updates via WebSocket.
 *
 * This hook manages the WebSocket subscription for export progress
 * and provides real-time updates during export rendering.
 */

import { useState, useEffect, useCallback, useRef } from "react";
import { useWebSocket } from "@/contexts/WebSocketContext";
import type { ExportProgress } from "../api/executeExport";
import { getExportStatus } from "../api/executeExport";

export interface UseExportProgressOptions {
  /** Export ID to subscribe to progress updates for */
  exportId: string | null;
  /** Execution ID - can subscribe by either export or execution ID */
  executionId?: string;
  /** Called when export completes successfully */
  onComplete?: (progress: ExportProgress) => void;
  /** Called when export fails */
  onError?: (error: string) => void;
}

export interface UseExportProgressResult {
  /** Current progress state (null if not started or complete) */
  progress: ExportProgress | null;
  /** Whether we're currently subscribed and listening */
  isSubscribed: boolean;
  /** Reset progress state (e.g., to start a new export) */
  reset: () => void;
  /** Manually check export status (useful after reconnection) */
  checkStatus: () => Promise<void>;
}

/**
 * Subscribe to export progress updates via WebSocket.
 */
export function useExportProgress(
  options: UseExportProgressOptions
): UseExportProgressResult {
  const { exportId, executionId, onComplete, onError } = options;
  const { send, lastMessage, isConnected } = useWebSocket();

  const [progress, setProgress] = useState<ExportProgress | null>(null);
  const [isSubscribed, setIsSubscribed] = useState(false);

  // Track callbacks in refs to avoid effect dependency changes
  const onCompleteRef = useRef(onComplete);
  const onErrorRef = useRef(onError);
  onCompleteRef.current = onComplete;
  onErrorRef.current = onError;

  // Subscribe when exportId or executionId changes
  useEffect(() => {
    const subId = exportId ?? executionId;
    if (!subId || !isConnected) {
      setIsSubscribed(false);
      return;
    }

    // Send subscription message
    send({
      type: "subscribe_export",
      export_id: exportId ?? undefined,
      execution_id: executionId ?? undefined,
    });
    setIsSubscribed(true);

    // Unsubscribe on cleanup
    return () => {
      send({ type: "unsubscribe_export" });
      setIsSubscribed(false);
    };
  }, [exportId, executionId, isConnected, send]);

  // Handle incoming messages
  useEffect(() => {
    if (!lastMessage || lastMessage.type !== "export_progress") {
      return;
    }

    const msg = lastMessage as unknown as ExportProgress & { type: string };

    // Check if this message is for our export
    if (exportId && msg.export_id !== exportId) {
      return;
    }
    if (!exportId && executionId && msg.execution_id !== executionId) {
      return;
    }

    const progressUpdate: ExportProgress = {
      export_id: msg.export_id,
      execution_id: msg.execution_id,
      stage: msg.stage,
      progress_percent: msg.progress_percent,
      status: msg.status,
      storage_url: msg.storage_url,
      file_size_bytes: msg.file_size_bytes,
      error: msg.error,
      timestamp: msg.timestamp,
    };

    setProgress(progressUpdate);

    // Call callbacks on completion/error
    if (progressUpdate.status === "completed") {
      onCompleteRef.current?.(progressUpdate);
    } else if (progressUpdate.status === "failed") {
      onErrorRef.current?.(progressUpdate.error ?? "Export failed");
    }
  }, [lastMessage, exportId, executionId]);

  const reset = useCallback(() => {
    setProgress(null);
  }, []);

  const checkStatus = useCallback(async () => {
    if (!exportId) return;

    try {
      const status = await getExportStatus(exportId);
      setProgress({
        export_id: status.export_id,
        execution_id: status.execution_id,
        stage:
          status.status === "completed"
            ? "completed"
            : status.status === "failed"
              ? "failed"
              : "preparing",
        progress_percent: status.status === "completed" ? 100 : 0,
        status: status.status as ExportProgress["status"],
        storage_url: status.storage_url,
        file_size_bytes: status.file_size_bytes,
        error: status.error,
      });

      if (status.status === "completed") {
        onCompleteRef.current?.({
          export_id: status.export_id,
          execution_id: status.execution_id,
          stage: "completed",
          progress_percent: 100,
          status: "completed",
          storage_url: status.storage_url,
          file_size_bytes: status.file_size_bytes,
        });
      } else if (status.status === "failed") {
        onErrorRef.current?.(status.error ?? "Export failed");
      }
    } catch (err) {
      console.error("Failed to check export status:", err);
    }
  }, [exportId]);

  return {
    progress,
    isSubscribed,
    reset,
    checkStatus,
  };
}

export default useExportProgress;
