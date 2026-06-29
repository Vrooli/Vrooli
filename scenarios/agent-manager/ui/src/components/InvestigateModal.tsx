import { useCallback, useEffect, useRef, useState } from "react";
import {
  AlertCircle,
  ChevronDown,
  Paperclip,
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
import { AttachmentPreview } from "./AttachmentPreview";
import type {
  InvestigationContextFlags,
  InvestigationDepth,
} from "../types";
import { DEFAULT_INVESTIGATION_CONTEXT } from "../types";
import { useInvestigationSettings } from "../hooks/useApi";
import { useAttachments } from "../hooks/useAttachments";

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
    scopePaths?: string[],
    attachmentIds?: string[],
    overrides?: { runnerType?: string; modelPreset?: string }
  ) => Promise<void>;
  loading?: boolean;
  error?: string | null;
}

const runnerOverrideOptions: { value: string; label: string }[] = [
  { value: "", label: "Default (profile)" },
  { value: "claude-code", label: "Claude Code" },
  { value: "codex", label: "Codex" },
  { value: "opencode", label: "OpenCode" },
  { value: "grok", label: "Grok" },
];

const presetOverrideOptions: { value: string; label: string }[] = [
  { value: "", label: "Default (profile)" },
  { value: "CHEAP", label: "Cheap" },
  { value: "FAST", label: "Fast" },
  { value: "SMART", label: "Smart" },
];

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

  // Runner + preset overrides (optional). Empty string = "use the investigation
  // profile's default" so the backend keeps auto-healing through its preset chain.
  const [runnerOverride, setRunnerOverride] = useState<string>("");
  const [presetOverride, setPresetOverride] = useState<string>("");

  // Image attachments — uploaded eagerly via the shared attachments hook.
  const {
    attachments,
    addAttachment,
    removeAttachment,
    clearAttachments,
    getUploadedIds,
    isUploading,
  } = useAttachments();
  const imageInputRef = useRef<HTMLInputElement>(null);

  // Get default settings
  const { data: settings } = useInvestigationSettings();

  // Reset state when modal opens
  useEffect(() => {
    if (!open) {
      setCustomContext("");
      setShowContext(false);
      setProjectRoot(defaultProjectRoot);
      setScopePaths(defaultScopePaths);
      setRunnerOverride("");
      setPresetOverride("");
      clearAttachments();
    }
  }, [open, defaultProjectRoot, defaultScopePaths, clearAttachments]);

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

  const handleImageSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      addAttachment(file);
      e.target.value = "";
    }
  };

  const handleSubmit = async () => {
    const attachmentIds = getUploadedIds();
    const overrides =
      runnerOverride || presetOverride
        ? {
            runnerType: runnerOverride || undefined,
            modelPreset: presetOverride || undefined,
          }
        : undefined;
    await onSubmit(
      customContext.trim(),
      depth,
      contextFlags,
      projectRoot.trim() || undefined,
      scopePaths.length > 0 ? scopePaths : undefined,
      attachmentIds.length > 0 ? attachmentIds : undefined,
      overrides
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* Wide modal for 2-column layout on large screens */}
      <DialogContent className="max-w-lg lg:max-w-[90vw] xl:max-w-7xl">
        <DialogHeader onClose={() => onOpenChange(false)}>
          <DialogTitle className="flex items-center gap-2">
            <Search className="h-5 w-5" />
            {title}
          </DialogTitle>
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>

        <DialogBody>
          {/* 2-column layout: options left, context right on large screens */}
          <div className="grid gap-6 lg:grid-cols-2">
            {/* Left Column: Main Options */}
            <div className="space-y-5">
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

              {/* Agent overrides: optional runner + preset picker. The default profile
                  already walks a model fallback chain, so leaving these on "Default"
                  is usually correct. Switch them when you want to deliberately run
                  the investigation on a different runner or preset. */}
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label htmlFor="investigate-runner-override">Runner</Label>
                  <select
                    id="investigate-runner-override"
                    value={runnerOverride}
                    onChange={(event) => setRunnerOverride(event.target.value)}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {runnerOverrideOptions.map((option) => (
                      <option key={option.value || "default"} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="space-y-1">
                  <Label htmlFor="investigate-preset-override">Preset</Label>
                  <select
                    id="investigate-preset-override"
                    value={presetOverride}
                    onChange={(event) => setPresetOverride(event.target.value)}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {presetOverrideOptions.map((option) => (
                      <option key={option.value || "default"} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
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
            </div>

            {/* Right Column: Additional Context + Quick Focus */}
            <div className="flex flex-col space-y-4">
              {/* Additional Context (larger textarea) */}
              <div className="flex flex-col space-y-2 flex-1">
                <Label htmlFor="customContext">Additional Context (optional)</Label>
                <Textarea
                  id="customContext"
                  value={customContext}
                  onChange={(e) => setCustomContext(e.target.value)}
                  placeholder="Provide any additional context for this investigation...

Examples:
• Suspected root cause or hypothesis
• Specific files or functions to examine
• Related issues or previous attempts
• Business context or constraints"
                  className="flex-1 min-h-[120px] lg:min-h-[200px] resize-none"
                />
                <p className="text-xs text-muted-foreground">
                  Share any extra details, suspected causes, or specific areas to investigate.
                </p>
              </div>

              {/* Image Attachments */}
              <div className="space-y-2">
                <Label>Image Attachments</Label>
                {attachments.length > 0 && (
                  <AttachmentPreview
                    attachments={attachments}
                    onRemove={removeAttachment}
                    isUploading={isUploading}
                  />
                )}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => imageInputRef.current?.click()}
                >
                  <Paperclip className="h-4 w-4 mr-2" />
                  Attach Image
                </Button>
                <input
                  ref={imageInputRef}
                  type="file"
                  accept="image/jpeg,image/png,image/gif,image/webp"
                  onChange={handleImageSelect}
                  className="hidden"
                />
                <p className="text-xs text-muted-foreground">
                  Screenshots of errors, UI states, or diagrams to aid the investigation.
                </p>
              </div>

              {/* Quick Focus Suggestion Cards */}
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
            </div>
          </div>

          {error && (
            <div className="mt-4 flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
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
          <Button
            onClick={handleSubmit}
            disabled={loading || isUploading}
            className="gap-2"
          >
            {loading ? (
              "Starting..."
            ) : isUploading ? (
              "Uploading..."
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
