/**
 * Debug modal showing full pipeline store state as JSON.
 *
 * Uses a snapshot approach to avoid React Error #185: "Cannot update
 * component while rendering a different component". The store is polled
 * every 2s, so subscribing during render causes race conditions.
 */

import { useState, useCallback } from "react";
import { createPortal } from "react-dom";
import { X, Copy, Check, RefreshCw } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Button } from "../ui/button";
import { usePipelineStore } from "../../store";
import { writeToClipboard } from "../../lib/browser";

/** Helper to extract store state snapshot */
function getStoreSnapshot() {
  const state = usePipelineStore.getState();
  return {
    scenarioName: state.scenarioName,
    pipelineId: state.pipelineId,
    pipelineStatus: state.pipelineStatus,
    runStatus: state.runStatus,
    error: state.error,
    errorInfo: state.errorInfo,
    isPolling: state.isPolling,
    pollIntervalMs: state.pollIntervalMs,
    bundleResult: state.bundleResult,
    preflightResult: state.preflightResult,
    generateResult: state.generateResult,
    buildResult: state.buildResult,
    smokeTestResult: state.smokeTestResult,
    distributionResult: state.distributionResult,
    stageLogs: state.stageLogs,
    pipelineHistory: state.pipelineHistory,
    preflightSecrets: state.preflightSecrets,
    preflightOverride: state.preflightOverride,
  };
}

interface DebugJsonModalProps {
  open: boolean;
  onClose: () => void;
}

export function DebugJsonModal({ open, onClose }: DebugJsonModalProps) {
  const [copied, setCopied] = useState(false);

  // Don't render anything if not open
  if (!open) return null;

  return <DebugJsonModalContent onClose={onClose} copied={copied} setCopied={setCopied} />;
}

interface DebugJsonModalContentProps {
  onClose: () => void;
  copied: boolean;
  setCopied: (value: boolean) => void;
}

function DebugJsonModalContent({ onClose, copied, setCopied }: DebugJsonModalContentProps) {
  // Snapshot state on mount to avoid render-during-render issues (React Error #185).
  // The store polls every 2s, so subscribing would cause re-renders during render.
  // Use useState with lazy initializer to capture state once on mount.
  const [storeState, setStoreState] = useState(getStoreSnapshot);

  const handleRefresh = useCallback(() => {
    setStoreState(getStoreSnapshot());
  }, []);

  // Safe JSON stringify that handles non-serializable values
  const jsonString = JSON.stringify(storeState, (_key, value) => {
    if (value instanceof Date) return value.toISOString();
    if (typeof value === 'function') return '[Function]';
    if (value === undefined) return null;
    if (typeof value === 'bigint') return value.toString();
    return value;
  }, 2);

  const handleCopy = async () => {
    const result = await writeToClipboard(jsonString);
    if (result.success) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return createPortal(
    <div
      className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <Card className="w-full max-w-4xl max-h-[90vh] flex flex-col border-slate-800 bg-slate-950/90 shadow-xl">
        <CardHeader className="flex flex-row items-start justify-between gap-4 shrink-0">
          <div className="space-y-1">
            <CardTitle className="text-lg text-slate-100">Pipeline Store Debug</CardTitle>
            <p className="text-sm text-slate-400">
              Full pipeline store state for debugging purposes.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleRefresh}
              className="h-8 gap-2"
              title="Refresh snapshot"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleCopy}
              className="h-8 gap-2"
            >
              {copied ? (
                <>
                  <Check className="h-4 w-4 text-green-400" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="h-4 w-4" />
                  Copy
                </>
              )}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onClose}
              className="h-8 w-8 p-0"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden">
          <pre className="h-full overflow-auto rounded-lg bg-slate-900/80 p-4 text-xs text-slate-300 font-mono">
            {jsonString}
          </pre>
        </CardContent>
      </Card>
    </div>,
    document.body
  );
}
