import { useState } from "react";
import { Search, Loader2, AlertCircle, Bot, X, Check } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { useDeploymentInvestigation } from "../../hooks/useInvestigation";

// Context items that can be included in investigations
const CONTEXT_OPTIONS = [
  {
    key: "error-info",
    label: "Error Information",
    description: "Failed step and error message",
    defaultChecked: true,
    recommended: true,
    details: [
      "Failed step name (if available)",
      "Raw error message text (if available)",
    ],
  },
  {
    key: "deployment-manifest",
    label: "Deployment Manifest",
    description: "Full dependency and target configuration",
    defaultChecked: true,
    details: [
      "Scenario ID and target VPS settings",
      "Dependencies (resources and scenarios)",
      "Ports and edge configuration",
    ],
  },
  {
    key: "vps-connection",
    label: "VPS Connection Details",
    description: "SSH command, host, credentials",
    defaultChecked: true,
    details: [
      "SSH command template",
      "Host, user, port, key path, workdir",
      "Domain and expected ports (if defined)",
    ],
  },
  {
    key: "deployment-history",
    label: "Deployment History",
    description: "Timeline of deployment events",
    defaultChecked: true,
    details: [
      "Event timeline JSON from previous deployment runs",
      "Only included when history exists",
    ],
  },
  {
    key: "preflight-results",
    label: "Preflight Results",
    description: "VPS capability check results",
    defaultChecked: false,
    details: [
      "Preflight checks JSON (ports, system capabilities)",
      "Only included when preflight results exist",
    ],
  },
  {
    key: "setup-results",
    label: "Setup Results",
    description: "Vrooli installation phase output",
    defaultChecked: false,
    details: [
      "Setup phase output JSON",
      "Only included when setup results exist",
    ],
  },
  {
    key: "deploy-results",
    label: "Deploy Results",
    description: "Deployment execution phase output",
    defaultChecked: false,
    details: [
      "Deployment execution output JSON",
      "Only included when deploy results exist",
    ],
  },
  {
    key: "architecture-guide",
    label: "Architecture Guide",
    description: "How Vrooli deployments work",
    defaultChecked: true,
    details: [
      "Dependency resolution flow",
      "Deployment pipeline steps",
      "Common failure patterns and fixes",
    ],
  },
] as const;

type ContextKey = typeof CONTEXT_OPTIONS[number]["key"];

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
  const [selectedContexts, setSelectedContexts] = useState<Set<ContextKey>>(
    () => new Set(CONTEXT_OPTIONS.filter(o => o.defaultChecked).map(o => o.key))
  );

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
      const inv = await trigger({
        auto_fix: autoFix,
        note: note.trim() || undefined,
        include_contexts: Array.from(selectedContexts),
      });
      if (inv) {
        onInvestigationStarted?.(inv.id);
        setShowOptions(false);
        setAutoFix(false);
        setNote("");
        // Reset to defaults
        setSelectedContexts(new Set(CONTEXT_OPTIONS.filter(o => o.defaultChecked).map(o => o.key)));
      }
    } catch (e) {
      // Error is captured in triggerError
    }
  };

  const handleCancel = () => {
    setShowOptions(false);
    setAutoFix(false);
    setNote("");
    // Reset to defaults
    setSelectedContexts(new Set(CONTEXT_OPTIONS.filter(o => o.defaultChecked).map(o => o.key)));
  };

  const toggleContext = (key: ContextKey) => {
    setSelectedContexts(prev => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
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
          <div className="bg-slate-900 border border-slate-700 rounded-lg shadow-xl w-[95vw] max-w-[80rem] mx-4">
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
            <div className="px-6 py-4 space-y-4 max-h-[70vh] overflow-y-auto">
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

              <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                <div className="space-y-4">
                  {/* Note input */}
                  <div>
                    <label className="block text-sm text-slate-300 mb-1.5">
                      Additional Notes
                    </label>
                    <textarea
                      value={note}
                      onChange={(e) => setNote(e.target.value)}
                      placeholder="E.g., 'This started failing after updating the database schema...'"
                      className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-sm text-slate-200 placeholder:text-slate-500 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      rows={6}
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

                {/* Context selection */}
                <div>
                  <label className="block text-sm text-slate-300 mb-2">
                    Context to Include
                  </label>
                  <div className="space-y-2">
                    {CONTEXT_OPTIONS.map((option) => (
                      <label
                        key={option.key}
                        className={`flex items-start gap-3 p-3 rounded-md cursor-pointer border transition-colors ${
                          selectedContexts.has(option.key)
                            ? "bg-slate-800/70 border-blue-500/60"
                            : "bg-slate-900 border-slate-800 hover:bg-slate-800/50"
                        }`}
                      >
                        <div className="relative flex items-center justify-center mt-0.5">
                          <input
                            type="checkbox"
                            checked={selectedContexts.has(option.key)}
                            onChange={() => toggleContext(option.key)}
                            className="sr-only"
                          />
                          <div
                            className={`w-4 h-4 rounded border flex items-center justify-center transition-colors ${
                              selectedContexts.has(option.key)
                                ? "bg-blue-500 border-blue-500"
                                : "bg-slate-800 border-slate-600"
                            }`}
                          >
                            {selectedContexts.has(option.key) && (
                              <Check className="h-3 w-3 text-white" />
                            )}
                          </div>
                        </div>
                        <div className="flex-1 min-w-0 space-y-1">
                          <div className="flex items-center gap-2">
                            <span className="text-sm text-slate-200">{option.label}</span>
                            {"recommended" in option && option.recommended && (
                              <span className="text-[10px] px-1.5 py-0.5 bg-blue-500/20 text-blue-400 rounded">
                                recommended
                              </span>
                            )}
                          </div>
                          <p className="text-xs text-slate-500">{option.description}</p>
                          <details className="group">
                            <summary className="text-xs text-blue-400 cursor-pointer list-none flex items-center gap-2">
                              <span className="inline-block transition-transform group-open:rotate-90">&gt;</span>
                              View included data
                            </summary>
                            <div className="mt-2 text-xs text-slate-400 space-y-1">
                              {option.details.map((detail) => (
                                <div key={detail} className="flex gap-2">
                                  <span className="text-slate-500">•</span>
                                  <span>{detail}</span>
                                </div>
                              ))}
                            </div>
                          </details>
                        </div>
                      </label>
                    ))}
                  </div>
                </div>
              </div>
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
