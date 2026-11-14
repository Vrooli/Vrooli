/** WebSocket progress with persisted reconnect recovery and export identity fencing. */
import { useState, useEffect, useCallback, useRef } from "react";
import { useWebSocket, useWebSocketMessage } from "@/contexts/WebSocketContext";
import type { ExportProgress } from "../api/executeExport";
import { getExportStatus } from "../api/executeExport";

export interface UseExportProgressOptions {
  exportId: string | null;
  executionId?: string;
  onComplete?: (progress: ExportProgress) => void;
  onError?: (error: string) => void;
}
export interface UseExportProgressResult {
  progress: ExportProgress | null;
  isSubscribed: boolean;
  statusError: string | null;
  reset: () => void;
  checkStatus: () => Promise<void>;
}

export function useExportProgress(options: UseExportProgressOptions): UseExportProgressResult {
  const { exportId, executionId, onComplete, onError } = options;
  const { send, isConnected } = useWebSocket();
  const [progress, setProgress] = useState<ExportProgress | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [isSubscribed, setIsSubscribed] = useState(false);
  const callbacks = useRef({ onComplete, onError });
  callbacks.current = { onComplete, onError };
  const identity = `${exportId ?? ""}/${executionId ?? ""}`;
  const fence = useRef({ identity, generation: 0, revision: 0, terminal: new Map<string, ExportProgress>() });
  if (fence.current.identity !== identity) {
    fence.current = { identity, generation: fence.current.generation + 1, revision: 0, terminal: fence.current.terminal };
  }
  const accept = useCallback((update: ExportProgress) => {
    const key = update.export_id;
    const terminal = fence.current.terminal.get(key);
    fence.current.revision += 1;
    setProgress(terminal ?? update);
    setStatusError(null);
    // Rehydration is required after reset/navigation, while callbacks remain
    // once per terminal export and late processing cannot regress that value.
    if (terminal) return;
    if (update.status === "completed" || update.status === "failed") {
      fence.current.terminal.set(key, update);
      if (fence.current.terminal.size > 128) {
        const oldest = fence.current.terminal.keys().next().value;
        if (oldest !== undefined) fence.current.terminal.delete(oldest);
      }
      if (update.status === "completed") callbacks.current.onComplete?.(update);
      else callbacks.current.onError?.(update.error ?? "Export failed");
    }
  }, []);

  const checkStatus = useCallback(async () => {
    if (!exportId) return;
    const { generation, revision } = fence.current;
    try {
      const status = await getExportStatus(exportId);
      if (generation !== fence.current.generation || revision !== fence.current.revision) return;
      accept({
        export_id: status.export_id,
        execution_id: status.execution_id,
        stage: status.status === "completed" ? "completed" : status.status === "failed" ? "failed" : "preparing",
        progress_percent: status.status === "completed" ? 100 : 0,
        status: status.status as ExportProgress["status"],
        storage_url: status.storage_url,
        file_size_bytes: status.file_size_bytes,
        error: status.error,
      });
    } catch (error) {
      if (generation === fence.current.generation && revision === fence.current.revision) {
        setStatusError(error instanceof Error ? error.message : "Unable to recover export status");
      }
    }
  }, [exportId, accept]);

  useEffect(() => {
    setProgress(null);
    setStatusError(null);
  }, [identity]);

  useEffect(() => {
    if ((!exportId && !executionId) || !isConnected) {
      setIsSubscribed(false);
      return;
    }
    send({ type: "subscribe_export", export_id: exportId ?? undefined, execution_id: executionId ?? undefined });
    setIsSubscribed(true);
    void checkStatus();
    return () => { send({ type: "unsubscribe_export" }); setIsSubscribed(false); };
  }, [exportId, executionId, isConnected, send, checkStatus]);

  useEffect(() => () => { fence.current.generation += 1; }, []);

  useWebSocketMessage((message) => {
    if (message.type !== "export_progress" || (!exportId && !executionId)) return;
    const update = message as unknown as ExportProgress;
    if (exportId ? update.export_id !== exportId : update.execution_id !== executionId) return;
    accept(update);
  });

  const reset = useCallback(() => {
    fence.current.generation += 1;
    fence.current.revision += 1;
    setProgress(null);
    setStatusError(null);
  }, []);
  return { progress, statusError, isSubscribed, reset, checkStatus };
}
export default useExportProgress;
