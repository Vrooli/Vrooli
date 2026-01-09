import { useCallback, useEffect, useState } from "react";
import {
  AlertCircle,
  ChevronDown,
  Folder,
  RefreshCw,
  Search,
  Zap,
  Settings,
  Microscope,
} from "lucide-react";
import { Button } from "./ui/button";
import { Checkbox } from "./ui/checkbox";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import type {
  DetectedScenario,
  InvestigationContextFlags,
  InvestigationDepth,
} from "../types";
import { DEFAULT_INVESTIGATION_CONTEXT } from "../types";
import { detectScenariosForRuns, useInvestigationSettings } from "../hooks/useApi";

interface InvestigateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  confirmLabel: string;
  /** Run IDs to investigate - used for scenario detection */
  runIds?: string[];
  onSubmit: (
    customContext: string,
    depth: InvestigationDepth,
    context?: InvestigationContextFlags,
    scenarioOverride?: string
  ) => Promise<void>;
  loading?: boolean;
  error?: string | null;
}

const depthOptions: {
  value: InvestigationDepth;
  label: string;
  description: string;
  icon: React.ReactNode;
}[] = [
  {
    value: "quick",
    label: "Quick",
    description: "Fast analysis of error messages and immediate causes",
    icon: <Zap className="h-4 w-4" />,
  },
  {
    value: "standard",
    label: "Standard",
    description: "Balanced analysis with targeted code exploration",
    icon: <Settings className="h-4 w-4" />,
  },
  {
    value: "deep",
    label: "Deep",
    description: "Thorough investigation exploring all relevant code paths",
    icon: <Microscope className="h-4 w-4" />,
  },
];

const contextOptions: {
  key: keyof InvestigationContextFlags;
  label: string;
  shortDesc: string;
}[] = [
  { key: "runSummaries", label: "Run summaries", shortDesc: "Summary data" },
  { key: "runEvents", label: "Run events", shortDesc: "Essential for debugging" },
  { key: "runDiffs", label: "Run diffs", shortDesc: "Code changes" },
  { key: "scenarioDocs", label: "Scenario docs", shortDesc: "CLAUDE.md, README" },
  { key: "fullLogs", label: "Full logs", shortDesc: "Can be large" },
];

export function InvestigateModal({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  runIds,
  onSubmit,
  loading = false,
  error = null,
}: InvestigateModalProps) {
  const [customContext, setCustomContext] = useState("");
  const [depth, setDepth] = useState<InvestigationDepth>("standard");
  const [contextFlags, setContextFlags] = useState<InvestigationContextFlags>(
    DEFAULT_INVESTIGATION_CONTEXT
  );
  const [showContext, setShowContext] = useState(false);

  // Scenario detection
  const [detectedScenarios, setDetectedScenarios] = useState<DetectedScenario[]>([]);
  const [scenariosLoading, setScenariosLoading] = useState(false);
  const [selectedScenario, setSelectedScenario] = useState<string | undefined>();
  const [showScenarioDropdown, setShowScenarioDropdown] = useState(false);

  // Get default settings
  const { data: settings } = useInvestigationSettings();

  // Reset state when modal opens
  useEffect(() => {
    if (!open) {
      setCustomContext("");
      setShowContext(false);
      setDetectedScenarios([]);
      setSelectedScenario(undefined);
      setShowScenarioDropdown(false);
    }
  }, [open]);

  // Apply defaults from settings when they load or modal opens
  useEffect(() => {
    if (open && settings) {
      setDepth(settings.defaultDepth);
      setContextFlags(settings.defaultContext);
    }
  }, [open, settings]);

  // Detect scenarios when modal opens with run IDs
  useEffect(() => {
    if (open && runIds && runIds.length > 0) {
      setScenariosLoading(true);
      detectScenariosForRuns(runIds)
        .then((scenarios) => {
          setDetectedScenarios(scenarios);
          // Auto-select the scenario with most runs
          if (scenarios.length > 0) {
            const sorted = [...scenarios].sort((a, b) => b.runCount - a.runCount);
            setSelectedScenario(sorted[0].projectRoot);
          }
        })
        .catch(() => {
          setDetectedScenarios([]);
        })
        .finally(() => {
          setScenariosLoading(false);
        });
    }
  }, [open, runIds]);

  const handleContextChange = (
    key: keyof InvestigationContextFlags,
    checked: boolean
  ) => {
    setContextFlags((prev) => ({ ...prev, [key]: checked }));
  };

  const handleUseDefaults = useCallback(() => {
    if (settings) {
      setDepth(settings.defaultDepth);
      setContextFlags(settings.defaultContext);
    }
  }, [settings]);

  const handleSubmit = async () => {
    await onSubmit(customContext.trim(), depth, contextFlags, selectedScenario);
  };

  const primaryScenario = detectedScenarios.find(
    (s) => s.projectRoot === selectedScenario
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader onClose={() => onOpenChange(false)}>
          <DialogTitle className="flex items-center gap-2">
            <Search className="h-5 w-5" />
            {title}
          </DialogTitle>
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>

        <DialogBody className="space-y-5">
          {/* Detected Scenario */}
          {runIds && runIds.length > 0 && (
            <div className="space-y-2">
              <Label>Detected Scenario</Label>
              {scenariosLoading ? (
                <div className="flex items-center gap-2 rounded-lg border border-border p-3 text-sm text-muted-foreground">
                  <RefreshCw className="h-4 w-4 animate-spin" />
                  Detecting scenario...
                </div>
              ) : detectedScenarios.length === 0 ? (
                <div className="rounded-lg border border-border p-3 text-sm text-muted-foreground">
                  No scenario detected from selected runs
                </div>
              ) : (
                <div className="relative">
                  <button
                    type="button"
                    onClick={() => setShowScenarioDropdown(!showScenarioDropdown)}
                    className="flex w-full items-center justify-between rounded-lg border border-border p-3 text-left transition-colors hover:border-primary/50"
                  >
                    <div className="flex items-center gap-2">
                      <Folder className="h-4 w-4 text-primary" />
                      <div>
                        <div className="text-sm font-medium">
                          {primaryScenario?.name ?? "Unknown"}
                        </div>
                        <div className="text-xs text-muted-foreground truncate max-w-[280px]">
                          {primaryScenario?.projectRoot ?? "No path"}
                        </div>
                        {primaryScenario?.keyFiles &&
                          primaryScenario.keyFiles.length > 0 && (
                            <div className="text-xs text-muted-foreground">
                              Key files: {primaryScenario.keyFiles.slice(0, 3).join(", ")}
                              {primaryScenario.keyFiles.length > 3 && "..."}
                            </div>
                          )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-muted-foreground">
                        {primaryScenario?.runCount ?? 0} run
                        {(primaryScenario?.runCount ?? 0) !== 1 ? "s" : ""}
                      </span>
                      {detectedScenarios.length > 1 && (
                        <ChevronDown className="h-4 w-4 text-muted-foreground" />
                      )}
                    </div>
                  </button>

                  {/* Scenario dropdown */}
                  {showScenarioDropdown && detectedScenarios.length > 1 && (
                    <div className="absolute z-10 mt-1 w-full rounded-lg border border-border bg-popover shadow-md">
                      {detectedScenarios.map((scenario) => (
                        <button
                          key={scenario.projectRoot}
                          type="button"
                          onClick={() => {
                            setSelectedScenario(scenario.projectRoot);
                            setShowScenarioDropdown(false);
                          }}
                          className={`flex w-full items-center justify-between px-3 py-2 text-left transition-colors first:rounded-t-lg last:rounded-b-lg hover:bg-muted ${
                            selectedScenario === scenario.projectRoot
                              ? "bg-primary/5"
                              : ""
                          }`}
                        >
                          <div className="flex items-center gap-2">
                            <Folder className="h-4 w-4 text-primary" />
                            <span className="text-sm">{scenario.name}</span>
                          </div>
                          <span className="text-xs text-muted-foreground">
                            {scenario.runCount} run{scenario.runCount !== 1 ? "s" : ""}
                          </span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Investigation Depth */}
          <div className="space-y-2">
            <Label>Investigation Depth</Label>
            <div className="grid gap-2">
              {depthOptions.map((option) => (
                <label
                  key={option.value}
                  className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors ${
                    depth === option.value
                      ? "border-primary bg-primary/5"
                      : "border-border hover:border-primary/50"
                  }`}
                >
                  <input
                    type="radio"
                    name="depth"
                    value={option.value}
                    checked={depth === option.value}
                    onChange={(e) =>
                      setDepth(e.target.value as InvestigationDepth)
                    }
                    className="mt-1"
                  />
                  <div className="flex-1">
                    <div className="flex items-center gap-2 font-medium">
                      {option.icon}
                      {option.label}
                    </div>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {option.description}
                    </p>
                  </div>
                </label>
              ))}
            </div>
          </div>

          {/* Context Selection (collapsible) */}
          <div className="space-y-2">
            <button
              type="button"
              onClick={() => setShowContext(!showContext)}
              className="flex w-full items-center justify-between text-left"
            >
              <Label className="cursor-pointer">Context to Include</Label>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleUseDefaults();
                  }}
                  className="h-6 text-xs"
                >
                  Use Defaults
                </Button>
                <ChevronDown
                  className={`h-4 w-4 text-muted-foreground transition-transform ${
                    showContext ? "rotate-180" : ""
                  }`}
                />
              </div>
            </button>

            {showContext && (
              <div className="grid grid-cols-2 gap-2 rounded-lg border border-border p-3">
                {contextOptions.map((option) => (
                  <label
                    key={option.key}
                    className="flex cursor-pointer items-center gap-2"
                  >
                    <Checkbox
                      checked={contextFlags[option.key]}
                      onCheckedChange={(checked) =>
                        handleContextChange(option.key, checked === true)
                      }
                    />
                    <div>
                      <span className="text-sm">{option.label}</span>
                      <span className="ml-1 text-xs text-muted-foreground">
                        ({option.shortDesc})
                      </span>
                    </div>
                  </label>
                ))}
              </div>
            )}
          </div>

          {/* Custom Context */}
          <div className="space-y-2">
            <Label htmlFor="customContext">Additional Context (optional)</Label>
            <Textarea
              id="customContext"
              value={customContext}
              onChange={(e) => setCustomContext(e.target.value)}
              placeholder="Provide any additional context for this investigation..."
              rows={3}
            />
            <p className="text-xs text-muted-foreground">
              Share any extra details, suspected causes, or specific areas to
              investigate.
            </p>
          </div>

          {error && (
            <div className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              <AlertCircle className="mt-0.5 h-4 w-4" />
              <span>{error}</span>
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={loading} className="gap-2">
            {loading ? (
              "Starting..."
            ) : (
              <>
                <Search className="h-4 w-4" />
                {confirmLabel}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
