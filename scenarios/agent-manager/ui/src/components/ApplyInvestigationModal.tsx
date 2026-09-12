import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AlertCircle, ChevronDown, ChevronRight, Info, Loader2, Paperclip, Search } from "lucide-react";
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
import { RecommendationItem } from "./RecommendationItem";
import { AttachmentPreview } from "./AttachmentPreview";
import { getInvestigationFindings } from "../hooks/useApi";
import { useAttachments } from "../hooks/useAttachments";
import type { InvestigationRecommendationCategory, Run } from "../types";
import { RunStatus } from "../types";

interface ApplyInvestigationModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The investigation run whose structured result holds the recommendations. */
  investigationRun: Run | null;
  onSubmit: (selected: string[], customContext: string, attachmentIds?: string[]) => Promise<void>;
  loading?: boolean;
  error?: string | null;
  /** Navigate to the investigation run's detail page (used by the "no structured recommendations" fallback). */
  onViewRun?: (runId: string) => void;
}

/** Statuses that mean the investigation run has finished (successfully or not). */
const TERMINAL_STATUSES = new Set<RunStatus>([RunStatus.COMPLETE, RunStatus.FAILED, RunStatus.CANCELLED]);

export function ApplyInvestigationModal({
  open,
  onOpenChange,
  investigationRun,
  onSubmit,
  loading = false,
  error = null,
  onViewRun,
}: ApplyInvestigationModalProps) {
  const findings = useMemo(() => getInvestigationFindings(investigationRun), [investigationRun]);
  const categories: InvestigationRecommendationCategory[] = findings?.categories ?? [];
  const isTerminal = investigationRun ? TERMINAL_STATUSES.has(investigationRun.status) : false;

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set());
  const [customContext, setCustomContext] = useState("");

  // Reset selection whenever the modal opens with a fresh set of findings —
  // default to everything selected.
  useEffect(() => {
    if (!open) return;
    if (!findings) {
      setSelected(new Set());
      setExpandedCategories(new Set());
      return;
    }
    const allTexts = findings.categories.flatMap((cat) => cat.recommendations.map((rec) => rec.text));
    setSelected(new Set(allTexts));
    setExpandedCategories(new Set(findings.categories.map((cat) => cat.name)));
  }, [open, findings]);

  // Image attachments — uploaded eagerly via the shared attachments hook.
  const { attachments, addAttachment, removeAttachment, clearAttachments, getUploadedIds, isUploading } =
    useAttachments();
  const imageInputRef = useRef<HTMLInputElement>(null);

  const handleImageSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      addAttachment(file);
      e.target.value = "";
    }
  };

  // Reset state when modal closes
  useEffect(() => {
    if (!open) {
      setCustomContext("");
      clearAttachments();
    }
  }, [open, clearAttachments]);

  const toggleCategory = (name: string) => {
    setExpandedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const toggleRecommendation = useCallback((text: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(text)) next.delete(text);
      else next.add(text);
      return next;
    });
  }, []);

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelected(new Set(categories.flatMap((cat) => cat.recommendations.map((rec) => rec.text))));
    } else {
      setSelected(new Set());
    }
  };

  const totalCount = categories.reduce((acc, cat) => acc + cat.recommendations.length, 0);
  const selectedCount = selected.size;
  const allSelected = totalCount > 0 && selectedCount === totalCount;
  const noneSelected = selectedCount === 0;

  const handleSubmit = async () => {
    const uploadedIds = getUploadedIds();
    const attachmentIds = uploadedIds.length > 0 ? uploadedIds : undefined;
    await onSubmit(Array.from(selected), customContext.trim(), attachmentIds);
  };

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
          {!findings && !isTerminal && (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <Loader2 className="h-8 w-8 animate-spin mb-3" />
              <p className="text-sm">Investigation in progress...</p>
              <p className="text-xs mt-1">Recommendations will appear once the investigation completes.</p>
            </div>
          )}

          {!findings && isTerminal && (
            <div className="space-y-4">
              <div className="flex items-start gap-2 rounded-md border border-amber-500/50 bg-amber-500/10 px-3 py-2 text-sm">
                <Info className="mt-0.5 h-4 w-4 text-amber-500 shrink-0" />
                <div>
                  <p className="font-medium text-amber-600">No structured recommendations available</p>
                  <p className="text-muted-foreground text-xs mt-1">
                    This investigation did not produce a structured recommendation set. Review the run summary
                    directly, or add context below for the apply agent.
                  </p>
                </div>
              </div>
              {investigationRun && onViewRun && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => onViewRun(investigationRun.id)}
                  className="gap-1.5"
                >
                  View investigation run
                </Button>
              )}
              <div className="space-y-2">
                <Label htmlFor="fallbackContext">Additional Context for Apply Agent</Label>
                <Textarea
                  id="fallbackContext"
                  value={customContext}
                  onChange={(e) => setCustomContext(e.target.value)}
                  placeholder="Add any context or instructions for the apply agent..."
                  rows={5}
                />
              </div>
            </div>
          )}

          {findings && (
            <div className="space-y-4">
              <div className="space-y-1">
                <p className="text-sm">{findings.summary}</p>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span>Category: {findings.primaryCategory}</span>
                  {findings.confidence && <span>· Confidence: {findings.confidence}</span>}
                </div>
              </div>

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
              </div>

              {/* Categories */}
              <div className="space-y-3">
                {categories.map((cat) => (
                  <div key={cat.name} className="rounded-lg border border-border overflow-hidden">
                    <button
                      type="button"
                      onClick={() => toggleCategory(cat.name)}
                      className="flex w-full items-center gap-2 bg-muted/30 px-3 py-2 text-left text-sm font-medium hover:text-primary"
                    >
                      {expandedCategories.has(cat.name) ? (
                        <ChevronDown className="h-4 w-4" />
                      ) : (
                        <ChevronRight className="h-4 w-4" />
                      )}
                      {cat.name}
                      <span className="text-xs text-muted-foreground font-normal">
                        ({cat.recommendations.filter((r) => selected.has(r.text)).length}
                        /{cat.recommendations.length})
                      </span>
                    </button>

                    {expandedCategories.has(cat.name) && (
                      <div className="p-3 space-y-2">
                        {cat.recommendations.map((rec) => (
                          <RecommendationItem
                            key={rec.text}
                            recommendation={rec}
                            selected={selected.has(rec.text)}
                            onToggle={() => toggleRecommendation(rec.text)}
                          />
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>

              <div className="space-y-2">
                <Label htmlFor="customContext">Additional Context for Apply Agent (optional)</Label>
                <Textarea
                  id="customContext"
                  value={customContext}
                  onChange={(e) => setCustomContext(e.target.value)}
                  placeholder="Add any context or instructions for the apply agent..."
                  rows={3}
                />
              </div>
            </div>
          )}

          {/* Image Attachments — hidden while the investigation is still running. */}
          {(findings || isTerminal) && (
            <div className="space-y-2 pt-2 border-t">
              <Label>Image Attachments (optional)</Label>
              {attachments.length > 0 && (
                <AttachmentPreview attachments={attachments} onRemove={removeAttachment} isUploading={isUploading} />
              )}
              <Button type="button" variant="outline" size="sm" onClick={() => imageInputRef.current?.click()}>
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
              <p className="text-xs text-muted-foreground">Screenshots or diagrams to guide the apply agent.</p>
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
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={loading || isUploading || (!findings && !isTerminal) || (!!findings && noneSelected)}
            className="gap-2"
          >
            {loading ? (
              "Applying..."
            ) : isUploading ? (
              "Uploading..."
            ) : findings ? (
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
