/**
 * PendingApprovalCard Component
 *
 * Displays a pending tool call that requires user approval before execution.
 * Provides approve/reject actions with optional rejection reason.
 */

import { useState } from "react";
import { Shield, Check, X, Loader2, ChevronDown, ChevronUp } from "lucide-react";
import { Button } from "../ui/button";
import type { PendingApproval } from "../../hooks/useCompletion";

interface PendingApprovalCardProps {
  approval: PendingApproval;
  onApprove: (id: string) => void;
  onReject: (id: string, reason?: string) => void;
  isProcessing: boolean;
}

export function PendingApprovalCard({
  approval,
  onApprove,
  onReject,
  isProcessing,
}: PendingApprovalCardProps) {
  const [showRejectInput, setShowRejectInput] = useState(false);
  const [rejectReason, setRejectReason] = useState("");
  const [isExpanded, setIsExpanded] = useState(false);

  // Parse and format arguments for display
  // Defensive: args might not be a string in edge cases
  const formatArguments = (args: unknown): string => {
    if (typeof args === "string") {
      try {
        const parsed = JSON.parse(args);
        return JSON.stringify(parsed, null, 2);
      } catch {
        return args;
      }
    } else if (args && typeof args === "object") {
      return JSON.stringify(args, null, 2);
    }
    return String(args ?? "{}");
  };

  const formattedArgs = formatArguments(approval.arguments);
  const isLongArgs = formattedArgs.length > 200;

  const handleReject = () => {
    if (showRejectInput) {
      onReject(approval.id, rejectReason || undefined);
      setShowRejectInput(false);
      setRejectReason("");
    } else {
      setShowRejectInput(true);
    }
  };

  const handleCancelReject = () => {
    setShowRejectInput(false);
    setRejectReason("");
  };

  return (
    <div
      className="border border-yellow-500/30 bg-yellow-500/5 rounded-lg p-4 my-2"
      data-testid={`pending-approval-${approval.id}`}
    >
      {/* Header */}
      <div className="flex items-center gap-2 mb-3">
        <div className="w-8 h-8 rounded-full bg-yellow-500/20 flex items-center justify-center">
          <Shield className="h-4 w-4 text-yellow-400" />
        </div>
        <div className="flex-1">
          <span className="font-medium text-white">Approval Required</span>
          <p className="text-xs text-slate-400">
            Tool <code className="bg-slate-800 px-1.5 py-0.5 rounded text-yellow-300">{approval.toolName}</code> is
            waiting for approval
          </p>
        </div>
      </div>

      {/* Arguments */}
      <div className="mb-4">
        <div className="flex items-center justify-between mb-1">
          <span className="text-xs text-slate-500">Arguments</span>
          {isLongArgs && (
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className="text-xs text-slate-400 hover:text-slate-300 flex items-center gap-1"
            >
              {isExpanded ? (
                <>
                  <ChevronUp className="h-3 w-3" />
                  Collapse
                </>
              ) : (
                <>
                  <ChevronDown className="h-3 w-3" />
                  Expand
                </>
              )}
            </button>
          )}
        </div>
        <pre
          className={`text-xs text-slate-300 bg-slate-800/50 rounded p-2 overflow-x-auto ${
            !isExpanded && isLongArgs ? "max-h-24" : ""
          } overflow-y-auto`}
        >
          {isExpanded || !isLongArgs ? formattedArgs : formattedArgs.slice(0, 200) + "..."}
        </pre>
      </div>

      {/* Actions */}
      {!showRejectInput ? (
        <div className="flex gap-2">
          <Button
            onClick={() => onApprove(approval.id)}
            disabled={isProcessing}
            className="flex-1 bg-green-600 hover:bg-green-700 text-white"
            data-testid={`approve-${approval.id}`}
          >
            {isProcessing ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <>
                <Check className="h-4 w-4 mr-1" />
                Approve
              </>
            )}
          </Button>
          <Button
            variant="secondary"
            onClick={handleReject}
            disabled={isProcessing}
            className="flex-1"
            data-testid={`reject-${approval.id}`}
          >
            <X className="h-4 w-4 mr-1" />
            Reject
          </Button>
        </div>
      ) : (
        <div className="space-y-2">
          <input
            type="text"
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            placeholder="Reason for rejection (optional)"
            className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-yellow-500/50"
            autoFocus
            data-testid={`reject-reason-${approval.id}`}
          />
          <div className="flex gap-2">
            <Button
              onClick={handleCancelReject}
              variant="ghost"
              size="sm"
              className="flex-1"
            >
              Cancel
            </Button>
            <Button
              onClick={handleReject}
              disabled={isProcessing}
              className="flex-1 bg-red-600 hover:bg-red-700 text-white"
              data-testid={`confirm-reject-${approval.id}`}
            >
              {isProcessing ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                "Confirm Reject"
              )}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
