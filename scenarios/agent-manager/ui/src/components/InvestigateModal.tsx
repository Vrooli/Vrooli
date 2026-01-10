import { useCallback, useEffect, useState } from "react";
import {
  AlertCircle,
  ChevronDown,
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
import { ScopePathsManager } from "./ScopePathsManager";
import type {
  InvestigationContextFlags,
  InvestigationDepth,
} from "../types";
import { DEFAULT_INVESTIGATION_CONTEXT } from "../types";
import { useInvestigationSettings } from "../hooks/useApi";

interface InvestigateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  confirmLabel: string;
  /** Default project root from the run being investigated */
  defaultProjectRoot?: string;
  /** Default scope paths from the run being investigated */
  defaultScopePaths?: string[];
  /** Hide the depth selector (used for Apply Investigation which inherits depth) */
  hideDepthSelector?: boolean;
  onSubmit: (
    customContext: string,
    depth: InvestigationDepth,
    context?: InvestigationContextFlags,
    projectRoot?: string,
    scopePaths?: string[]
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
  { key: "fullLogs", label: "Full logs", shortDesc: "Can be large" },
];

/** Suggestion cards for common agent issues */
const suggestionCards: {
  label: string;
  description: string;
  context: string;
}[] = [
  {
    label: "Agent crashed",
    description: "Focus on error traces and final state",
    context: "The agent crashed or errored out. Please focus on the error traces, final state, and what action triggered the failure.",
  },
  {
    label: "Slow / excessive tokens",
    description: "Focus on turn counts and timing",
    context: "The agent was slow or used excessive tokens. Please analyze turn counts, timing patterns, and identify any loops or redundant work.",
  },
  {
    label: "Went off-topic",
    description: "Focus on task vs actual behavior",
    context: "The agent went off-topic or worked on the wrong thing. Please compare the original task instructions to what the agent actually did.",
  },
  {
    label: "Stopped early",
    description: "Focus on stop reasons and completion",
    context: "The agent stopped before completing the task. Please analyze why it stopped, what signals it may have misinterpreted as completion.",
  },
];

export function InvestigateModal({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  defaultProjectRoot = "",
  defaultScopePaths = [],
  hideDepthSelector = false,
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

  // Project and scope paths
  const [projectRoot, setProjectRoot] = useState(defaultProjectRoot);
  const [scopePaths, setScopePaths] = useState<string[]>(defaultScopePaths);

  // Get default settings
  const { data: settings } = useInvestigationSettings();

  // Reset state when modal opens
  useEffect(() => {
    if (!open) {
      setCustomContext("");
      setShowContext(false);
      setProjectRoot(defaultProjectRoot);
      setScopePaths(defaultScopePaths);
    }
  }, [open, defaultProjectRoot, defaultScopePaths]);

  // Apply defaults from settings when they load or modal opens
  useEffect(() => {
    if (open && settings) {
      setDepth(settings.defaultDepth);
      setContextFlags(settings.defaultContext);
    }
  }, [open, settings]);

  // Sync project root and scope paths when defaults change
  useEffect(() => {
    if (open) {
      setProjectRoot(defaultProjectRoot);
      setScopePaths(defaultScopePaths);
    }
  }, [open, defaultProjectRoot, defaultScopePaths]);

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

  const handleSuggestionClick = (suggestion: string) => {
    setCustomContext((prev) => {
      if (prev.trim()) {
        return `${prev.trim()}\n\n${suggestion}`;
      }
      return suggestion;
    });
  };

  const handleSubmit = async () => {
    await onSubmit(
      customContext.trim(),
      depth,
      contextFlags,
      projectRoot.trim() || undefined,
      scopePaths.length > 0 ? scopePaths : undefined
    );
  };

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
          {/* Investigation Scope */}
          <ScopePathsManager
            projectRoot={projectRoot}
            onProjectRootChange={setProjectRoot}
            scopePaths={scopePaths}
            onScopePathsChange={setScopePaths}
            defaultProjectRoot={defaultProjectRoot}
            defaultScopePaths={defaultScopePaths}
            scopePathsHelp="Directories where the investigation agent can make changes. Leave empty for read-only analysis."
          />

          {/* Suggestion Cards */}
          {!hideDepthSelector && (
            <div className="space-y-2">
              <Label>Quick Focus (click to add)</Label>
              <div className="grid grid-cols-2 gap-2">
                {suggestionCards.map((card) => (
                  <button
                    key={card.label}
                    type="button"
                    onClick={() => handleSuggestionClick(card.context)}
                    className="flex flex-col items-start gap-1 rounded-lg border border-border p-2 text-left transition-colors hover:border-primary/50 hover:bg-primary/5"
                  >
                    <span className="text-sm font-medium">{card.label}</span>
                    <span className="text-xs text-muted-foreground">
                      {card.description}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Investigation Depth - hidden for Apply Investigation */}
          {!hideDepthSelector && (
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
          )}

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
