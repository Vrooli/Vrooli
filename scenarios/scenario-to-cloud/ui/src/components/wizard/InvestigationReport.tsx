import { X, Clock, Coins, FileText, Copy, Check, XCircle, CheckCircle2, StopCircle, Wrench, Loader2 } from "lucide-react";
import { useState } from "react";
import { Button } from "../ui/button";
import type { Investigation } from "../../types/investigation";
import type { ApplyFixesRequest } from "../../types/investigation";

interface InvestigationReportProps {
  investigation: Investigation;
  onClose: () => void;
  onApplyFixes?: (investigationId: string, options: ApplyFixesRequest) => Promise<void>;
  isApplyingFixes?: boolean;
}

function getStatusConfig(status: Investigation["status"]) {
  switch (status) {
    case "completed":
      return {
        icon: CheckCircle2,
        iconColor: "text-emerald-400",
        label: "Completed",
        badgeClass: "bg-emerald-500/20 text-emerald-300",
        title: "Investigation Report",
      };
    case "failed":
      return {
        icon: XCircle,
        iconColor: "text-red-400",
        label: "Failed",
        badgeClass: "bg-red-500/20 text-red-300",
        title: "Investigation Failed",
      };
    case "cancelled":
      return {
        icon: StopCircle,
        iconColor: "text-slate-400",
        label: "Cancelled",
        badgeClass: "bg-slate-500/20 text-slate-300",
        title: "Investigation Cancelled",
      };
    default:
      return {
        icon: FileText,
        iconColor: "text-blue-400",
        label: status,
        badgeClass: "bg-blue-500/20 text-blue-300",
        title: "Investigation Details",
      };
  }
}

export function InvestigationReport({ investigation, onClose, onApplyFixes, isApplyingFixes }: InvestigationReportProps) {
  const [copied, setCopied] = useState(false);
  const [immediate, setImmediate] = useState(true);
  const [permanent, setPermanent] = useState(false);
  const [prevention, setPrevention] = useState(false);
  const [fixNote, setFixNote] = useState("");

  const handleCopy = async () => {
    if (investigation.findings) {
      await navigator.clipboard.writeText(investigation.findings);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleApplyFixes = async () => {
    if (!onApplyFixes) return;
    await onApplyFixes(investigation.id, {
      immediate,
      permanent,
      prevention,
      note: fixNote.trim() || undefined,
    });
  };

  const canApplyFixes = investigation.status === "completed" && investigation.findings && onApplyFixes;
  const hasSelectedFix = immediate || permanent || prevention;

  const details = investigation.details;
  const createdAt = new Date(investigation.created_at);
  const statusConfig = getStatusConfig(investigation.status);
  const StatusIcon = statusConfig.icon;

  // Check if this is a fix application report (not an investigation)
  const isFixReport = details?.operation_mode?.startsWith("fix-application");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-slate-900 border border-slate-700 rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <StatusIcon className={`h-5 w-5 ${statusConfig.iconColor}`} />
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-semibold text-white">{statusConfig.title}</h2>
                <span className={`px-2 py-0.5 rounded text-xs font-medium ${statusConfig.badgeClass}`}>
                  {statusConfig.label}
                </span>
              </div>
              <p className="text-xs text-slate-400 font-mono">ID: {investigation.id}</p>
            </div>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Metadata */}
        <div className="px-6 py-3 bg-slate-800/50 border-b border-slate-700">
          <div className="flex flex-wrap gap-4 text-sm">
            <div className="flex items-center gap-1.5 text-slate-300">
              <Clock className="h-4 w-4 text-slate-400" />
              <span>
                {createdAt.toLocaleDateString()} {createdAt.toLocaleTimeString()}
              </span>
            </div>

            {details?.duration_seconds && (
              <div className="flex items-center gap-1.5 text-slate-300">
                <span className="text-slate-500">Duration:</span>
                <span>{Math.round(details.duration_seconds / 60)}m {details.duration_seconds % 60}s</span>
              </div>
            )}

            {details?.tokens_used && (
              <div className="flex items-center gap-1.5 text-slate-300">
                <span className="text-slate-500">Tokens:</span>
                <span>{details.tokens_used.toLocaleString()}</span>
              </div>
            )}

            {details?.cost_estimate && details.cost_estimate > 0 && (
              <div className="flex items-center gap-1.5 text-slate-300">
                <Coins className="h-4 w-4 text-slate-400" />
                <span>${details.cost_estimate.toFixed(4)}</span>
              </div>
            )}

            {details?.operation_mode && (
              <div className="flex items-center gap-1.5">
                <span
                  className={`px-2 py-0.5 rounded text-xs font-medium ${
                    details.operation_mode === "auto-fix"
                      ? "bg-amber-500/20 text-amber-300"
                      : "bg-slate-700 text-slate-300"
                  }`}
                >
                  {details.operation_mode}
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Findings content */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4">
          {/* Error message - show prominently at top if present */}
          {investigation.error_message && (
            <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-4">
              <p className="text-sm text-red-400 font-medium mb-1">Error</p>
              <p className="text-sm text-red-300">{investigation.error_message}</p>
            </div>
          )}

          {/* Agent run details if available */}
          {investigation.agent_run_id && (
            <div className="bg-slate-800/50 border border-slate-700 rounded-lg p-4">
              <p className="text-sm text-slate-400 font-medium mb-1">Agent Run ID</p>
              <p className="text-xs text-slate-300 font-mono">{investigation.agent_run_id}</p>
            </div>
          )}

          {/* Investigation details metadata */}
          {details && (details.trigger_reason || details.deployment_step) && (
            <div className="bg-slate-800/50 border border-slate-700 rounded-lg p-4">
              <p className="text-sm text-slate-400 font-medium mb-2">Context</p>
              <div className="space-y-1 text-sm">
                {details.trigger_reason && (
                  <p className="text-slate-300">
                    <span className="text-slate-500">Reason: </span>
                    {details.trigger_reason}
                  </p>
                )}
                {details.deployment_step && (
                  <p className="text-slate-300">
                    <span className="text-slate-500">Failed Step: </span>
                    {details.deployment_step}
                  </p>
                )}
              </div>
            </div>
          )}

          {/* Findings */}
          {investigation.findings ? (
            <div className="prose prose-invert prose-sm max-w-none">
              <p className="text-sm text-slate-400 font-medium mb-2">Findings</p>
              <pre className="whitespace-pre-wrap font-mono text-sm text-slate-300 bg-slate-800/50 p-4 rounded-lg">
                {investigation.findings}
              </pre>
            </div>
          ) : !investigation.error_message ? (
            <p className="text-slate-500 text-center py-8">No findings available.</p>
          ) : null}
        </div>

        {/* Apply Fixes Section - only show for completed investigations with findings, not for fix reports */}
        {canApplyFixes && !isFixReport && (
          <div className="px-6 py-4 bg-slate-800/30 border-t border-slate-700">
            <div className="flex items-center gap-2 mb-3">
              <Wrench className="h-4 w-4 text-blue-400" />
              <span className="text-sm font-medium text-slate-200">Apply Fixes</span>
            </div>
            <div className="flex flex-wrap gap-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={immediate}
                  onChange={(e) => setImmediate(e.target.checked)}
                  className="rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500"
                />
                <div>
                  <span className="text-sm text-slate-200">Immediate Fix</span>
                  <p className="text-xs text-slate-500">Run commands on VPS now</p>
                </div>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={permanent}
                  onChange={(e) => setPermanent(e.target.checked)}
                  className="rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500"
                />
                <div>
                  <span className="text-sm text-slate-200">Permanent Fix</span>
                  <p className="text-xs text-slate-500">Modify code/config</p>
                </div>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={prevention}
                  onChange={(e) => setPrevention(e.target.checked)}
                  className="rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500"
                />
                <div>
                  <span className="text-sm text-slate-200">Prevention</span>
                  <p className="text-xs text-slate-500">Add monitoring/alerts</p>
                </div>
              </label>
            </div>
            <div className="mt-3">
              <label className="block text-sm text-slate-300 mb-1">
                Additional Instructions (optional)
              </label>
              <textarea
                value={fixNote}
                onChange={(e) => setFixNote(e.target.value)}
                placeholder="Any additional context or instructions for the fix agent..."
                className="w-full px-3 py-2 rounded-md border border-slate-600 bg-slate-800 text-slate-200 placeholder-slate-500 text-sm focus:ring-blue-500 focus:border-blue-500 resize-none"
                rows={2}
              />
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-slate-700">
          <div className="flex items-center gap-2">
            {investigation.findings && (
              <Button variant="outline" size="sm" onClick={handleCopy}>
                {copied ? (
                  <>
                    <Check className="h-4 w-4 mr-1.5 text-emerald-400" />
                    Copied!
                  </>
                ) : (
                  <>
                    <Copy className="h-4 w-4 mr-1.5" />
                    Copy Report
                  </>
                )}
              </Button>
            )}
          </div>
          <div className="flex items-center gap-2">
            {canApplyFixes && !isFixReport && (
              <Button
                onClick={handleApplyFixes}
                disabled={!hasSelectedFix || isApplyingFixes}
              >
                {isApplyingFixes ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
                    Applying...
                  </>
                ) : (
                  <>
                    <Wrench className="h-4 w-4 mr-1.5" />
                    Apply Selected Fixes
                  </>
                )}
              </Button>
            )}
            <Button variant="outline" onClick={onClose}>Close</Button>
          </div>
        </div>
      </div>
    </div>
  );
}
