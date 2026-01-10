// Investigation settings tab - configuration for investigation agents

import { useCallback, useEffect, useState } from "react";
import { ExternalLink, Microscope, RotateCcw, Settings, Wrench, Zap } from "lucide-react";
import { Button } from "../../ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../ui/card";
import { Input } from "../../ui/input";
import { Label } from "../../ui/label";
import { Textarea } from "../../ui/textarea";
import { Checkbox } from "../../ui/checkbox";
import { ensureProfile } from "../../../hooks/useApi";
import type {
  InvestigationContextFlags,
  InvestigationDepth,
  InvestigationSettings,
  InvestigationTagRule,
} from "../../../types";
import { DEFAULT_INVESTIGATION_CONTEXT } from "../../../types";

type AgentSubtab = "investigation" | "apply";

interface InvestigationTabProps {
  settings: InvestigationSettings | null;
  loading: boolean;
  error: string | null;
  onSave: (settings: Partial<{
    promptTemplate: string;
    applyPromptTemplate: string;
    defaultDepth: InvestigationDepth;
    defaultContext: InvestigationContextFlags;
    investigationTagAllowlist: InvestigationTagRule[];
  }>) => Promise<InvestigationSettings | void>;
  onReset: () => Promise<InvestigationSettings | void>;
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
  description: string;
}[] = [
  {
    key: "runSummaries",
    label: "Run summaries",
    description: "Include run summary data (always lightweight)",
  },
  {
    key: "runEvents",
    label: "Run events",
    description: "Include run events (essential for debugging)",
  },
  {
    key: "runDiffs",
    label: "Run diffs",
    description: "Include code changes made during runs",
  },
  {
    key: "fullLogs",
    label: "Full logs",
    description: "Include complete run logs (can be large)",
  },
];

export function InvestigationTab({
  settings,
  loading,
  error,
  onSave,
  onReset,
}: InvestigationTabProps) {
  // Subtab state
  const [activeSubtab, setActiveSubtab] = useState<AgentSubtab>("investigation");

  // Local state for editing
  const [draftDepth, setDraftDepth] = useState<InvestigationDepth>("standard");
  const [draftContext, setDraftContext] = useState<InvestigationContextFlags>(DEFAULT_INVESTIGATION_CONTEXT);
  const [draftPrompt, setDraftPrompt] = useState("");
  const [draftApplyPrompt, setDraftApplyPrompt] = useState("");
  const [draftTagAllowlist, setDraftTagAllowlist] = useState<InvestigationTagRule[]>([]);
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);
  const [openingProfile, setOpeningProfile] = useState(false);

  // Sync from settings when they load
  useEffect(() => {
    if (settings) {
      setDraftDepth(settings.defaultDepth);
      setDraftContext(settings.defaultContext);
      setDraftPrompt(settings.promptTemplate);
      setDraftApplyPrompt(settings.applyPromptTemplate);
      setDraftTagAllowlist(settings.investigationTagAllowlist || []);
    }
  }, [settings]);

  const hasChanges = settings && (
    draftDepth !== settings.defaultDepth ||
    draftPrompt !== settings.promptTemplate ||
    draftApplyPrompt !== settings.applyPromptTemplate ||
    JSON.stringify(draftContext) !== JSON.stringify(settings.defaultContext) ||
    JSON.stringify(draftTagAllowlist) !== JSON.stringify(settings.investigationTagAllowlist)
  );

  const handleSave = useCallback(async () => {
    setSaving(true);
    setLocalError(null);
    try {
      await onSave({
        promptTemplate: draftPrompt,
        applyPromptTemplate: draftApplyPrompt,
        defaultDepth: draftDepth,
        defaultContext: draftContext,
        investigationTagAllowlist: draftTagAllowlist,
      });
    } catch (err) {
      setLocalError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }, [onSave, draftPrompt, draftApplyPrompt, draftDepth, draftContext, draftTagAllowlist]);

  const handleReset = useCallback(async () => {
    setResetting(true);
    setLocalError(null);
    try {
      await onReset();
    } catch (err) {
      setLocalError((err as Error).message);
    } finally {
      setResetting(false);
    }
  }, [onReset]);

  const handleContextChange = (key: keyof InvestigationContextFlags, checked: boolean) => {
    setDraftContext(prev => ({ ...prev, [key]: checked }));
  };

  const handleAddTagRule = () => {
    setDraftTagAllowlist(prev => ([
      ...prev,
      { pattern: "", isRegex: false, caseSensitive: false },
    ]));
  };

  const handleUpdateTagRule = (index: number, update: Partial<InvestigationTagRule>) => {
    setDraftTagAllowlist(prev => prev.map((rule, idx) => (
      idx === index ? { ...rule, ...update } : rule
    )));
  };

  const handleRemoveTagRule = (index: number) => {
    setDraftTagAllowlist(prev => prev.filter((_, idx) => idx !== index));
  };

  // Open profile with auto-creation if it doesn't exist
  const handleOpenProfile = useCallback(async (profileKey: string) => {
    setOpeningProfile(true);
    setLocalError(null);
    try {
      await ensureProfile(profileKey);
      window.location.href = `/profiles?profileKey=${profileKey}`;
    } catch (err) {
      setLocalError(`Failed to open profile: ${(err as Error).message}`);
      setOpeningProfile(false);
    }
  }, []);

  if (loading && !settings) {
    return (
      <div className="flex items-center justify-center py-8 text-muted-foreground">
        Loading investigation settings...
      </div>
    );
  }

  const displayError = localError || error;

  return (
    <div className="space-y-6">
      {/* Subtab Buttons */}
      <div className="flex gap-1 rounded-lg bg-muted p-1">
        <button
          type="button"
          onClick={() => setActiveSubtab("investigation")}
          className={`flex flex-1 items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
            activeSubtab === "investigation"
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <Microscope className="h-4 w-4" />
          Investigation
        </button>
        <button
          type="button"
          onClick={() => setActiveSubtab("apply")}
          className={`flex flex-1 items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
            activeSubtab === "apply"
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <Wrench className="h-4 w-4" />
          Apply Investigation
        </button>
      </div>

      {/* Investigation Subtab Content */}
      {activeSubtab === "investigation" && (
        <>
          {/* Agent Profile Card */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Investigation Agent Profile</CardTitle>
              <CardDescription className="text-xs">
                Analyzes failed runs to find behavioral root causes (read-only)
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-0">
              <Button
                variant="outline"
                size="sm"
                className="w-full gap-2"
                disabled={openingProfile}
                onClick={() => handleOpenProfile("agent-manager-investigation")}
              >
                <ExternalLink className="h-3 w-3" />
                {openingProfile ? "Opening..." : "Open Profile"}
              </Button>
            </CardContent>
          </Card>

          {/* Default Depth */}
          <div className="space-y-3">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              Default Investigation Depth
            </h3>
            <Card>
              <CardContent className="py-4">
                <div className="grid gap-2 sm:grid-cols-3">
                  {depthOptions.map((option) => (
                    <label
                      key={option.value}
                      className={`flex cursor-pointer items-start gap-2 rounded-lg border p-3 transition-colors ${
                        draftDepth === option.value
                          ? "border-primary bg-primary/5"
                          : "border-border hover:border-primary/50"
                      }`}
                    >
                      <input
                        type="radio"
                        name="defaultDepth"
                        value={option.value}
                        checked={draftDepth === option.value}
                        onChange={(e) => setDraftDepth(e.target.value as InvestigationDepth)}
                        className="mt-0.5"
                      />
                      <div className="flex-1">
                        <div className="flex items-center gap-1.5 text-sm font-medium">
                          {option.icon}
                          {option.label}
                        </div>
                        <p className="text-xs text-muted-foreground">{option.description}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Investigation Prompt Template */}
          <div className="space-y-3">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              Investigation Prompt Template
            </h3>
            <Card>
              <CardContent className="space-y-2 py-4">
                <Textarea
                  value={draftPrompt}
                  onChange={(e) => setDraftPrompt(e.target.value)}
                  placeholder="Enter the base instruction for investigation agents..."
                  rows={12}
                  className="font-mono text-xs"
                />
                <p className="text-xs text-muted-foreground">
                  Base instruction for investigation agents. Dynamic info (depth, scenarios, runs)
                  is provided as separate context attachments.
                </p>
              </CardContent>
            </Card>
          </div>
        </>
      )}

      {/* Apply Investigation Subtab Content */}
      {activeSubtab === "apply" && (
        <>
          {/* Agent Profile Card */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Apply Investigation Agent Profile</CardTitle>
              <CardDescription className="text-xs">
                Applies recommended fixes from investigation reports (has write capabilities)
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-0">
              <Button
                variant="outline"
                size="sm"
                className="w-full gap-2"
                disabled={openingProfile}
                onClick={() => handleOpenProfile("agent-manager-investigation-apply")}
              >
                <ExternalLink className="h-3 w-3" />
                {openingProfile ? "Opening..." : "Open Profile"}
              </Button>
            </CardContent>
          </Card>

          {/* Apply Investigation Prompt Template */}
          <div className="space-y-3">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              Apply Investigation Prompt Template
            </h3>
            <Card>
              <CardContent className="space-y-2 py-4">
                <Textarea
                  value={draftApplyPrompt}
                  onChange={(e) => setDraftApplyPrompt(e.target.value)}
                  placeholder="Enter the base instruction for apply investigation agents..."
                  rows={12}
                  className="font-mono text-xs"
                />
                <p className="text-xs text-muted-foreground">
                  Base instruction for apply investigation agents. The investigation run ID and
                  CLI commands are injected automatically.
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Apply Fixes Tag Allowlist */}
          <div className="space-y-3">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              Apply Fixes Tag Allowlist
            </h3>
            <Card>
              <CardContent className="space-y-3 py-4">
                <p className="text-xs text-muted-foreground">
                  Only runs with tags matching these rules will show Apply Fixes and trigger recommendation
                  extraction. Use glob patterns unless Regex is enabled (e.g., "investigation", "*-investigation").
                </p>
                {draftTagAllowlist.length === 0 && (
                  <div className="rounded-md border border-dashed border-muted-foreground/40 px-3 py-2 text-xs text-muted-foreground">
                    No tag rules defined. Defaults will apply on save.
                  </div>
                )}
                <div className="space-y-2">
                  {draftTagAllowlist.map((rule, index) => (
                    <div
                      key={`${rule.pattern}-${index}`}
                      className="rounded-md border border-border p-3 space-y-2"
                    >
                      <div className="flex items-center gap-2">
                        <Input
                          value={rule.pattern}
                          onChange={(e) => handleUpdateTagRule(index, { pattern: e.target.value })}
                          placeholder="Tag pattern (glob or regex)"
                          className="flex-1 text-xs"
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveTagRule(index)}
                          className="text-xs text-muted-foreground hover:text-destructive"
                        >
                          Remove
                        </Button>
                      </div>
                      <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
                        <label className="flex items-center gap-2">
                          <Checkbox
                            checked={rule.isRegex}
                            onCheckedChange={(checked) =>
                              handleUpdateTagRule(index, { isRegex: checked === true })
                            }
                          />
                          Regex
                        </label>
                        <label className="flex items-center gap-2">
                          <Checkbox
                            checked={rule.caseSensitive}
                            onCheckedChange={(checked) =>
                              handleUpdateTagRule(index, { caseSensitive: checked === true })
                            }
                          />
                          Case sensitive
                        </label>
                      </div>
                    </div>
                  ))}
                </div>
                <Button type="button" variant="outline" size="sm" onClick={handleAddTagRule}>
                  Add tag rule
                </Button>
              </CardContent>
            </Card>
          </div>
        </>
      )}

      {/* Shared: Default Context Attachments */}
      <div className="space-y-3">
        <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          Default Context Attachments
        </h3>
        <Card>
          <CardContent className="space-y-2 py-4">
            <p className="text-xs text-muted-foreground">
              Select which context to include by default (applies to both agent types)
            </p>
            <div className="grid gap-2 sm:grid-cols-2">
              {contextOptions.map((option) => (
                <label
                  key={option.key}
                  className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 transition-colors hover:border-primary/50"
                >
                  <Checkbox
                    checked={draftContext[option.key]}
                    onCheckedChange={(checked) =>
                      handleContextChange(option.key, checked === true)
                    }
                  />
                  <div className="flex-1">
                    <div className="text-sm font-medium">{option.label}</div>
                    <p className="text-xs text-muted-foreground">{option.description}</p>
                  </div>
                </label>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Error display */}
      {displayError && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {displayError}
        </div>
      )}

      {/* Action buttons */}
      <div className="flex justify-end gap-2">
        <Button
          variant="outline"
          onClick={handleReset}
          disabled={saving || resetting}
          className="gap-2"
        >
          <RotateCcw className="h-4 w-4" />
          {resetting ? "Resetting..." : "Reset to Defaults"}
        </Button>
        <Button
          onClick={handleSave}
          disabled={!hasChanges || saving || resetting}
        >
          {saving ? "Saving..." : "Save Changes"}
        </Button>
      </div>
    </div>
  );
}
