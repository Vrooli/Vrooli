/**
 * Capture Card
 *
 * Displays a raw capture in the unified action feed with classification status.
 *
 * States:
 * 1. Classifying: spinner + original text
 * 2. Classified: original text + list of suggested items with accept/edit/dismiss
 * 3. Failed: error state with retry button
 */

import { useState } from "react";
import { Check, CheckCheck, Loader2, Pencil, RefreshCw, X } from "lucide-react";
import { Button } from "../ui/button";
import { Card } from "../ui/card";
import { TagList } from "../ui/tag-list";
import { backlogService } from "../../services/backlog-service";
import { captureService } from "../../services/capture-service";
import { useCaptureStore } from "../../stores/capture-store";
import { useBacklogStore } from "../../stores";
import { formatRelativeTime } from "../../lib";
import { BACKLOG_KIND_LABELS } from "../../types";
import { selectors } from "../../consts/selectors";
import type { Capture, CaptureClassificationItem } from "../../types";
import type { BacklogFormValues } from "../../types";

interface CaptureCardProps {
  capture: Capture;
  onEditItem?: (prefill: BacklogFormValues) => void;
}

export function CaptureCard({ capture, onEditItem }: CaptureCardProps) {
  const [acceptingIndex, setAcceptingIndex] = useState<number | null>(null);
  const [acceptedIndices, setAcceptedIndices] = useState<Set<number>>(new Set());
  const [dismissedIndices, setDismissedIndices] = useState<Set<number>>(new Set());
  const [isRetrying, setIsRetrying] = useState(false);
  const removeCapture = useCaptureStore((s) => s.removeCapture);
  const updateCapture = useCaptureStore((s) => s.updateCapture);
  const upsertBacklogItem = useBacklogStore((s) => s.upsertItem);

  const items = capture.classification?.items ?? [];
  const allResolved = items.length > 0 && items.every((_, i) => acceptedIndices.has(i) || dismissedIndices.has(i));

  const handleAcceptItem = async (item: CaptureClassificationItem, index: number) => {
    setAcceptingIndex(index);
    try {
      const created = await backlogService.create({
        name: item.title.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, ""),
        title: item.title,
        description: item.description,
        kind: item.kind,
        status: "backlog",
        priority: item.priority,
        tags: item.tags,
      });
      upsertBacklogItem(created);
      const next = new Set(acceptedIndices);
      next.add(index);
      setAcceptedIndices(next);

      // Auto-delete capture when all items resolved.
      if (items.every((_, i) => next.has(i) || dismissedIndices.has(i))) {
        await captureService.remove(capture.id);
        removeCapture(capture.id);
      }
    } catch {
      // Leave item in unresolved state for retry.
    } finally {
      setAcceptingIndex(null);
    }
  };

  const handleAcceptAll = async () => {
    for (let i = 0; i < items.length; i++) {
      if (!acceptedIndices.has(i) && !dismissedIndices.has(i)) {
        await handleAcceptItem(items[i], i);
      }
    }
  };

  const handleDismissItem = (index: number) => {
    const next = new Set(dismissedIndices);
    next.add(index);
    setDismissedIndices(next);

    // Auto-delete capture when all items resolved.
    if (items.every((_, i) => acceptedIndices.has(i) || next.has(i))) {
      captureService.remove(capture.id).then(() => removeCapture(capture.id));
    }
  };

  const handleDismissCapture = async () => {
    await captureService.remove(capture.id);
    removeCapture(capture.id);
  };

  const handleRetry = async () => {
    setIsRetrying(true);
    try {
      await captureService.classify(capture.id);
      updateCapture(capture.id, { status: "classifying", classification: null });
    } catch {
      // Keep failed state.
    } finally {
      setIsRetrying(false);
    }
  };

  const handleEditItem = (item: CaptureClassificationItem, index: number) => {
    if (onEditItem) {
      const next = new Set(dismissedIndices);
      next.add(index);
      setDismissedIndices(next);
      onEditItem({
        name: item.title.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, ""),
        title: item.title,
        description: item.description,
        kind: item.kind,
        status: "backlog",
        priority: item.priority,
        tags: item.tags,
      });
    }
  };

  const borderColor =
    capture.status === "classified" && items.length > 0 ? "border-l-emerald-500" :
    capture.status === "classified" && items.length === 0 ? "border-l-slate-500" :
    capture.status === "failed" ? "border-l-red-500" :
    "border-l-cyan-500";

  return (
    <Card
      className={`border-l-4 ${borderColor} relative`}
      data-testid={selectors.captures.card}
    >
      <div className="p-4">
        {/* Original text */}
        <p className="mb-2 text-sm text-slate-300 line-clamp-3">{capture.text}</p>
        <p className="mb-3 text-xs text-slate-600">{formatRelativeTime(capture.created)}</p>

        {/* Classifying state */}
        {capture.status === "classifying" && (
          <div className="flex items-center gap-2 text-xs text-cyan-400">
            <Loader2 className="h-3 w-3 animate-spin" />
            Classifying...
          </div>
        )}

        {/* Failed state */}
        {capture.status === "failed" && (
          <div className="flex items-center gap-2">
            <span className="text-xs text-red-400">Classification failed</span>
            <Button
              variant="outline"
              size="sm"
              onClick={handleRetry}
              disabled={isRetrying}
              data-testid={selectors.captures.retryButton}
            >
              <RefreshCw className={`mr-1 h-3 w-3 ${isRetrying ? "animate-spin" : ""}`} />
              Retry
            </Button>
          </div>
        )}

        {/* Classified but nothing actionable (no-op) */}
        {capture.status === "classified" && items.length === 0 && (
          <div className="flex items-center justify-between">
            <span className="text-xs text-slate-500">Nothing actionable detected</span>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleDismissCapture}
            >
              Dismiss
            </Button>
          </div>
        )}

        {/* Classified state — show suggested items */}
        {capture.status === "classified" && items.length > 0 && !allResolved && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-slate-400">
                {items.length} item{items.length !== 1 ? "s" : ""} suggested
              </span>
              {items.length > 1 && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleAcceptAll}
                  data-testid={selectors.captures.acceptAllButton}
                >
                  <CheckCheck className="mr-1 h-3 w-3" />
                  Accept All
                </Button>
              )}
            </div>

            {items.map((item, index) => {
              if (acceptedIndices.has(index) || dismissedIndices.has(index)) return null;
              return (
                <div
                  key={index}
                  className="rounded-lg border border-slate-700 bg-slate-800/50 p-3"
                >
                  <div className="mb-1 flex items-center gap-2">
                    <span className="rounded bg-slate-700 px-1.5 py-0.5 text-xs font-medium text-slate-300">
                      {BACKLOG_KIND_LABELS[item.kind] ?? item.kind}
                    </span>
                    <span className="text-xs text-slate-500">P{item.priority}</span>
                    {item.confidence >= 0.8 && (
                      <span className="text-xs text-emerald-500">{Math.round(item.confidence * 100)}%</span>
                    )}
                  </div>
                  <h4 className="mb-1 text-sm font-medium text-slate-200">{item.title}</h4>
                  {item.description && (
                    <p className="mb-2 text-xs text-slate-400 line-clamp-2">{item.description}</p>
                  )}
                  {item.tags.length > 0 && (
                    <div className="mb-2">
                      <TagList tags={item.tags} maxTags={3} />
                    </div>
                  )}
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleAcceptItem(item, index)}
                      disabled={acceptingIndex === index}
                      data-testid={selectors.captures.itemAcceptButton}
                    >
                      {acceptingIndex === index ? (
                        <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                      ) : (
                        <Check className="mr-1 h-3 w-3" />
                      )}
                      Accept
                    </Button>
                    {onEditItem && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleEditItem(item, index)}
                        data-testid={selectors.captures.itemEditButton}
                      >
                        <Pencil className="mr-1 h-3 w-3" />
                        Edit
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDismissItem(index)}
                      data-testid={selectors.captures.itemDismissButton}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Dismiss button (top-right) */}
        <button
          onClick={handleDismissCapture}
          className="absolute right-2 top-2 rounded p-1 text-slate-600 transition-colors hover:bg-slate-700 hover:text-slate-400"
          title="Dismiss capture"
          data-testid={selectors.captures.dismissButton}
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
    </Card>
  );
}
