import { useCallback, useEffect, useState } from "react";
import {
  AlertCircle,
  ChevronDown,
  ChevronRight,
  Info,
  Plus,
  RefreshCw,
  Search,
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
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import { RecommendationItem } from "./RecommendationItem";
import { extractRecommendations } from "../hooks/useApi";
import type {
  ExtractionResult,
  Recommendation,
  RecommendationCategory,
} from "../types";

interface ApplyInvestigationModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  investigationRunId: string;
  onSubmit: (customContext: string) => Promise<void>;
  loading?: boolean;
  error?: string | null;
}

type ModalState = "loading" | "success" | "fallback";

function generateId(): string {
  return `new-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

/**
 * Serializes selected recommendations into markdown format.
 * Notes are appended inline: "- Fix auth bug (Note: use JWT not sessions)"
 */
function serializeRecommendations(categories: RecommendationCategory[]): string {
  const lines: string[] = [];
  for (const cat of categories) {
    const selected = cat.recommendations.filter((r) => r.selected);
    if (selected.length === 0) continue;
    lines.push(`## ${cat.name}`);
    for (const rec of selected) {
      let line = `- ${rec.text}`;
      if (rec.note) {
        line += ` (Note: ${rec.note})`;
      }
      lines.push(line);
    }
    lines.push("");
  }
  return lines.join("\n").trim();
}

export function ApplyInvestigationModal({
  open,
  onOpenChange,
  investigationRunId,
  onSubmit,
  loading = false,
  error = null,
}: ApplyInvestigationModalProps) {
  const [state, setState] = useState<ModalState>("loading");
  const [categories, setCategories] = useState<RecommendationCategory[]>([]);
  const [rawText, setRawText] = useState("");
  const [extractedFrom, setExtractedFrom] = useState<"summary" | "events">(
    "summary"
  );
  const [extractionError, setExtractionError] = useState<string | null>(null);
  const [fallbackText, setFallbackText] = useState("");
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(
    new Set()
  );
  const [showRawOutput, setShowRawOutput] = useState(false);
  const [newCategoryName, setNewCategoryName] = useState("");
  const [showAddCategory, setShowAddCategory] = useState(false);

  // Fetch recommendations when modal opens
  useEffect(() => {
    if (!open || !investigationRunId) return;

    setState("loading");
    setExtractionError(null);

    extractRecommendations(investigationRunId)
      .then((result: ExtractionResult) => {
        setRawText(result.rawText || "");
        setExtractedFrom(result.extractedFrom);

        if (result.success && result.categories.length > 0) {
          setCategories(result.categories);
          // Expand all categories by default
          setExpandedCategories(new Set(result.categories.map((c) => c.id)));
          setState("success");
        } else {
          setExtractionError(result.error || "No recommendations found");
          setFallbackText(result.rawText || "");
          setState("fallback");
        }
      })
      .catch((err) => {
        setExtractionError(err.message);
        setFallbackText("");
        setState("fallback");
      });
  }, [open, investigationRunId]);

  // Reset state when modal closes
  useEffect(() => {
    if (!open) {
      setState("loading");
      setCategories([]);
      setRawText("");
      setExtractionError(null);
      setFallbackText("");
      setExpandedCategories(new Set());
      setShowRawOutput(false);
      setShowAddCategory(false);
      setNewCategoryName("");
    }
  }, [open]);

  const toggleCategory = (id: string) => {
    setExpandedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleSelectAll = (selected: boolean) => {
    setCategories((prev) =>
      prev.map((cat) => ({
        ...cat,
        recommendations: cat.recommendations.map((rec) => ({
          ...rec,
          selected,
        })),
      }))
    );
  };

  const handleUpdateRecommendation = useCallback(
    (categoryId: string, updated: Recommendation) => {
      setCategories((prev) =>
        prev.map((cat) =>
          cat.id === categoryId
            ? {
                ...cat,
                recommendations: cat.recommendations.map((rec) =>
                  rec.id === updated.id ? updated : rec
                ),
              }
            : cat
        )
      );
    },
    []
  );

  const handleDeleteRecommendation = useCallback(
    (categoryId: string, recId: string) => {
      setCategories((prev) =>
        prev
          .map((cat) =>
            cat.id === categoryId
              ? {
                  ...cat,
                  recommendations: cat.recommendations.filter(
                    (rec) => rec.id !== recId
                  ),
                }
              : cat
          )
          .filter((cat) => cat.recommendations.length > 0)
      );
    },
    []
  );

  const handleAddRecommendation = useCallback((categoryId: string) => {
    setCategories((prev) =>
      prev.map((cat) =>
        cat.id === categoryId
          ? {
              ...cat,
              recommendations: [
                ...cat.recommendations,
                {
                  id: generateId(),
                  text: "New recommendation",
                  selected: true,
                  isNew: true,
                },
              ],
            }
          : cat
      )
    );
  }, []);

  const handleAddCategory = () => {
    if (!newCategoryName.trim()) return;
    const newCat: RecommendationCategory = {
      id: generateId(),
      name: newCategoryName.trim(),
      recommendations: [
        {
          id: generateId(),
          text: "New recommendation",
          selected: true,
          isNew: true,
        },
      ],
      isNew: true,
    };
    setCategories((prev) => [...prev, newCat]);
    setExpandedCategories((prev) => new Set([...prev, newCat.id]));
    setNewCategoryName("");
    setShowAddCategory(false);
  };

  const handleDeleteCategory = useCallback((categoryId: string) => {
    setCategories((prev) => prev.filter((cat) => cat.id !== categoryId));
  }, []);

  const handleSubmit = async () => {
    if (state === "success") {
      const serialized = serializeRecommendations(categories);
      await onSubmit(serialized);
    } else {
      await onSubmit(fallbackText.trim());
    }
  };

  const allSelected = categories.every((cat) =>
    cat.recommendations.every((rec) => rec.selected)
  );
  const noneSelected = categories.every((cat) =>
    cat.recommendations.every((rec) => !rec.selected)
  );
  const selectedCount = categories.reduce(
    (acc, cat) => acc + cat.recommendations.filter((r) => r.selected).length,
    0
  );
  const totalCount = categories.reduce(
    (acc, cat) => acc + cat.recommendations.length,
    0
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col">
        <DialogHeader onClose={() => onOpenChange(false)}>
          <DialogTitle className="flex items-center gap-2">
            <Search className="h-5 w-5" />
            Apply Investigation Recommendations
          </DialogTitle>
          <DialogDescription>
            Select the recommendations you want to apply from this investigation.
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="flex-1 overflow-y-auto space-y-4">
          {state === "loading" && (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <RefreshCw className="h-8 w-8 animate-spin mb-3" />
              <p className="text-sm">Extracting recommendations...</p>
              <p className="text-xs mt-1">
                Using local LLM to analyze investigation output
              </p>
            </div>
          )}

          {state === "fallback" && (
            <div className="space-y-4">
              <div className="flex items-start gap-2 rounded-md border border-amber-500/50 bg-amber-500/10 px-3 py-2 text-sm">
                <Info className="mt-0.5 h-4 w-4 text-amber-500 shrink-0" />
                <div>
                  <p className="font-medium text-amber-600">
                    Could not extract structured recommendations
                  </p>
                  <p className="text-muted-foreground text-xs mt-1">
                    {extractionError ||
                      "The investigation output will be passed directly to the apply agent."}
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="fallbackContext">
                  Additional Context for Apply Agent
                </Label>
                <Textarea
                  id="fallbackContext"
                  value={fallbackText}
                  onChange={(e) => setFallbackText(e.target.value)}
                  placeholder="Add any context or instructions for the apply agent..."
                  rows={5}
                />
              </div>

              {rawText && (
                <div className="space-y-2">
                  <button
                    type="button"
                    onClick={() => setShowRawOutput(!showRawOutput)}
                    className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
                  >
                    {showRawOutput ? (
                      <ChevronDown className="h-4 w-4" />
                    ) : (
                      <ChevronRight className="h-4 w-4" />
                    )}
                    View raw investigation output
                  </button>
                  {showRawOutput && (
                    <pre className="rounded-md border bg-muted/50 p-3 text-xs overflow-x-auto max-h-48">
                      {rawText}
                    </pre>
                  )}
                </div>
              )}
            </div>
          )}

          {state === "success" && (
            <div className="space-y-4">
              {/* Selection controls */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Checkbox
                    checked={allSelected}
                    onCheckedChange={(checked) => handleSelectAll(!!checked)}
                    className={noneSelected ? "opacity-50" : ""}
                  />
                  <span className="text-sm text-muted-foreground">
                    {selectedCount} of {totalCount} selected
                  </span>
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span>Source: {extractedFrom}</span>
                </div>
              </div>

              {/* Categories */}
              <div className="space-y-3">
                {categories.map((cat) => (
                  <div
                    key={cat.id}
                    className="rounded-lg border border-border overflow-hidden"
                  >
                    {/* Category header */}
                    <div className="flex items-center justify-between bg-muted/30 px-3 py-2">
                      <button
                        type="button"
                        onClick={() => toggleCategory(cat.id)}
                        className="flex items-center gap-2 text-sm font-medium hover:text-primary"
                      >
                        {expandedCategories.has(cat.id) ? (
                          <ChevronDown className="h-4 w-4" />
                        ) : (
                          <ChevronRight className="h-4 w-4" />
                        )}
                        {cat.name}
                        {cat.isNew && (
                          <span className="text-xs text-primary">(new)</span>
                        )}
                        <span className="text-xs text-muted-foreground font-normal">
                          ({cat.recommendations.filter((r) => r.selected).length}
                          /{cat.recommendations.length})
                        </span>
                      </button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDeleteCategory(cat.id)}
                        className="h-7 text-xs text-muted-foreground hover:text-destructive"
                      >
                        Remove
                      </Button>
                    </div>

                    {/* Category content */}
                    {expandedCategories.has(cat.id) && (
                      <div className="p-3 space-y-2">
                        {cat.recommendations.map((rec) => (
                          <RecommendationItem
                            key={rec.id}
                            recommendation={rec}
                            onUpdate={(updated) =>
                              handleUpdateRecommendation(cat.id, updated)
                            }
                            onDelete={() =>
                              handleDeleteRecommendation(cat.id, rec.id)
                            }
                          />
                        ))}
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => handleAddRecommendation(cat.id)}
                          className="w-full justify-center gap-1 text-muted-foreground hover:text-foreground"
                        >
                          <Plus className="h-3.5 w-3.5" />
                          Add recommendation
                        </Button>
                      </div>
                    )}
                  </div>
                ))}

                {/* Add category */}
                {showAddCategory ? (
                  <div className="flex items-center gap-2">
                    <Input
                      value={newCategoryName}
                      onChange={(e) => setNewCategoryName(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          handleAddCategory();
                        } else if (e.key === "Escape") {
                          setShowAddCategory(false);
                          setNewCategoryName("");
                        }
                      }}
                      placeholder="Category name..."
                      className="flex-1"
                      autoFocus
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={handleAddCategory}
                    >
                      Add
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setShowAddCategory(false);
                        setNewCategoryName("");
                      }}
                    >
                      Cancel
                    </Button>
                  </div>
                ) : (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setShowAddCategory(true)}
                    className="w-full gap-1"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    Add category
                  </Button>
                )}
              </div>

              {/* Raw output toggle */}
              {rawText && (
                <div className="pt-2 border-t">
                  <button
                    type="button"
                    onClick={() => setShowRawOutput(!showRawOutput)}
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                  >
                    {showRawOutput ? (
                      <ChevronDown className="h-3.5 w-3.5" />
                    ) : (
                      <ChevronRight className="h-3.5 w-3.5" />
                    )}
                    View raw investigation output
                  </button>
                  {showRawOutput && (
                    <pre className="mt-2 rounded-md border bg-muted/50 p-3 text-xs overflow-x-auto max-h-32">
                      {rawText}
                    </pre>
                  )}
                </div>
              )}
            </div>
          )}

          {error && (
            <div className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
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
            disabled={
              loading ||
              state === "loading" ||
              (state === "success" && selectedCount === 0)
            }
            className="gap-2"
          >
            {loading ? (
              "Applying..."
            ) : state === "success" ? (
              <>
                <Search className="h-4 w-4" />
                Apply {selectedCount} Recommendation{selectedCount !== 1 ? "s" : ""}
              </>
            ) : (
              <>
                <Search className="h-4 w-4" />
                Apply Investigation
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
