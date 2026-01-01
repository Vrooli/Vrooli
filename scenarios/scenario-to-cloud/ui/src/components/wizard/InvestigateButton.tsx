import { useState } from "react";
import { Search, Loader2, AlertCircle, Bot, X } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { useDeploymentInvestigation } from "../../hooks/useInvestigation";

interface InvestigateButtonProps {
  deploymentId: string;
  disabled?: boolean;
  onInvestigationStarted?: (investigationId: string) => void;
}

export function InvestigateButton({
  deploymentId,
  disabled,
  onInvestigationStarted,
}: InvestigateButtonProps) {
  const [showOptions, setShowOptions] = useState(false);
  const [autoFix, setAutoFix] = useState(false);
  const [note, setNote] = useState("");

  const {
    isAgentAvailable,
    isAgentEnabled,
    isAgentLoading,
    trigger,
    isTriggering,
    triggerError,
    isRunning,
    activeInvestigationId,
  } = useDeploymentInvestigation(deploymentId);

  const handleClick = () => {
    if (isRunning) {
      // If already running, notify parent to show the investigation
      if (activeInvestigationId) {
        onInvestigationStarted?.(activeInvestigationId);
      }
      return;
    }

    setShowOptions(true);
  };

  const handleTrigger = async () => {
    try {
      const inv = await trigger({ auto_fix: autoFix, note: note.trim() || undefined });
      if (inv) {
        onInvestigationStarted?.(inv.id);
        setShowOptions(false);
        setAutoFix(false);
        setNote("");
      }
    } catch (e) {
      // Error is captured in triggerError
    }
  };

  const handleCancel = () => {
    setShowOptions(false);
    setAutoFix(false);
    setNote("");
  };

  // Agent not configured
  if (!isAgentEnabled) {
    return (
      <Button variant="outline" disabled title="Agent manager integration is not enabled">
        <Bot className="h-4 w-4 mr-1.5 opacity-50" />
        <span className="opacity-50">Investigate</span>
      </Button>
    );
  }

  return (
    <>
      {/* Main button */}
      <Button
        variant="outline"
        onClick={handleClick}
        disabled={disabled || isAgentLoading || isTriggering}
      >
        {isAgentLoading ? (
          <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
        ) : isRunning ? (
          <Loader2 className="h-4 w-4 mr-1.5 animate-spin text-blue-400" />
        ) : !isAgentAvailable ? (
          <AlertCircle className="h-4 w-4 mr-1.5 text-amber-400" />
        ) : (
          <Search className="h-4 w-4 mr-1.5" />
        )}
        {isRunning ? "View Investigation" : "Investigate Failure"}
      </Button>

      {/* Options modal */}
      {showOptions && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-slate-700 rounded-lg shadow-xl max-w-md w-full mx-4">
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
              <div className="flex items-center gap-2">
                <Bot className="h-5 w-5 text-blue-400" />
                <h2 className="text-lg font-semibold text-white">Investigation Options</h2>
              </div>
              <Button variant="ghost" size="sm" onClick={handleCancel}>
                <X className="h-4 w-4" />
              </Button>
            </div>

            {/* Content */}
            <div className="px-6 py-4 space-y-4">
              {!isAgentAvailable && (
                <Alert variant="warning" title="Agent Manager Unavailable">
                  The agent-manager service is not running. Start it to enable investigations.
                </Alert>
              )}

              {triggerError && (
                <Alert variant="error" title="Investigation Failed">
                  {triggerError instanceof Error ? triggerError.message : String(triggerError)}
                </Alert>
              )}

              {/* Note input */}
              <div>
                <label className="block text-sm text-slate-300 mb-1.5">
                  Context for the investigation
                </label>
                <textarea
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  placeholder="E.g., 'This started failing after updating the database schema...'"
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-sm text-slate-200 placeholder:text-slate-500 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  rows={3}
                />
                <p className="text-xs text-slate-500 mt-1">Optional - provide any context that might help the investigation</p>
              </div>

              {/* Auto-fix toggle */}
              <label className="flex items-start gap-3 p-3 bg-slate-800/50 rounded-lg border border-slate-700 cursor-pointer hover:bg-slate-800 transition-colors">
                <input
                  type="checkbox"
                  checked={autoFix}
                  onChange={(e) => setAutoFix(e.target.checked)}
                  className="mt-0.5 rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500"
                />
                <div>
                  <span className="text-sm text-slate-200 font-medium">Allow auto-fix</span>
                  <p className="text-xs text-slate-400 mt-0.5">Agent may attempt safe fixes like restarting services or clearing disk space</p>
                </div>
              </label>
            </div>

            {/* Footer */}
            <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-slate-700">
              <Button variant="ghost" onClick={handleCancel}>
                Cancel
              </Button>
              <Button
                onClick={handleTrigger}
                disabled={!isAgentAvailable || isTriggering}
              >
                {isTriggering ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
                    Starting...
                  </>
                ) : (
                  <>
                    <Search className="h-4 w-4 mr-1.5" />
                    Start Investigation
                  </>
                )}
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
