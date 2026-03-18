/**
 * AsyncOperationDrawer - Slide-out drawer for async operation details and history.
 *
 * Shows full operation details including progress, skills, result JSON, and actions.
 * Can also display history of completed operations with pagination.
 */

import { useState, useEffect, useCallback } from "react";
import { X, ChevronRight } from "lucide-react";
import { Button } from "../ui/button";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import { OperationDetail, HistoryList, formatToolName } from "./AsyncOperationParts";

interface AsyncOperationDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  /** Specific operation to show, or null for history view */
  operation: AsyncStatusUpdate | null;
  /** Tool arguments JSON - used to extract and display skills */
  toolArguments?: string;
  completedOperations: AsyncStatusUpdate[];
  /** Map of tool_call_id to arguments JSON for history items */
  argumentsMap?: Map<string, string>;
  onRefresh: (toolCallId: string) => Promise<void>;
  onCancel: (toolCallId: string) => Promise<void>;
  onInsertReference: (operation: AsyncStatusUpdate) => void;
  onLoadMoreHistory: () => Promise<void>;
  hasMoreHistory: boolean;
}

export function AsyncOperationDrawer({
  isOpen,
  onClose,
  operation,
  toolArguments,
  completedOperations,
  argumentsMap,
  onRefresh,
  onCancel,
  onInsertReference,
  onLoadMoreHistory,
  hasMoreHistory,
}: AsyncOperationDrawerProps) {
  const [selectedOp, setSelectedOp] = useState<AsyncStatusUpdate | null>(operation);

  // Sync selectedOp with operation prop
  useEffect(() => {
    setSelectedOp(operation);
  }, [operation]);

  const handleBack = useCallback(() => {
    setSelectedOp(null);
  }, []);

  // Get arguments for the selected operation
  const getSelectedOpArguments = (): string | undefined => {
    if (!selectedOp) return undefined;
    if (operation && selectedOp.tool_call_id === operation.tool_call_id) {
      return toolArguments;
    }
    return argumentsMap?.get(selectedOp.tool_call_id);
  };

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/50 z-40" onClick={onClose} />

      {/* Drawer */}
      <div className="fixed right-0 top-0 bottom-0 w-96 max-w-full bg-slate-900 border-l border-slate-800 z-50 flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-slate-800">
          <div className="flex items-center gap-2">
            {selectedOp && (
              <Button variant="ghost" size="sm" onClick={handleBack} className="h-7 px-2 text-slate-400 hover:text-slate-200">
                <ChevronRight className="h-4 w-4 rotate-180" />
              </Button>
            )}
            <h2 className="text-sm font-medium text-white">
              {selectedOp ? formatToolName(selectedOp.tool_name) : "Operation History"}
            </h2>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose} className="h-7 w-7 p-0 text-slate-400 hover:text-slate-200">
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Content */}
        {selectedOp ? (
          <OperationDetail
            operation={selectedOp}
            toolArguments={getSelectedOpArguments()}
            onRefresh={onRefresh}
            onCancel={onCancel}
            onInsertReference={onInsertReference}
          />
        ) : (
          <HistoryList
            operations={completedOperations}
            onSelectOperation={setSelectedOp}
            onLoadMore={onLoadMoreHistory}
            hasMore={hasMoreHistory}
            onInsertReference={onInsertReference}
          />
        )}
      </div>
    </>
  );
}
