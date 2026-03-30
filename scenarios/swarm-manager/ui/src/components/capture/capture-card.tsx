/**
 * Capture Card
 *
 * Displays a raw capture with classification triage.
 *
 * States:
 * 1. Classifying: spinner + "Classifying..." below title
 * 2. Classified: title + compact suggested items with accept/edit/dismiss
 * 3. Classified (no-op): title + "Nothing actionable detected" + dismiss
 * 4. Failed: title + error with retry
 */

import { useState } from "react";
import { Check, CheckCheck, ChevronDown, ChevronUp, Loader2, Pencil, RefreshCw, X } from "lucide-react";
import { Button } from "../ui/button";
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
  className?: string;
}

function toSlug(title: string) {
  return title.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}

function SuggestionRow({
  item,
  isAccepting,
  onAccept,
  onEdit,
  onDismiss,
}: {
  item: CaptureClassificationItem;
  isAccepting: boolean;
  onAccept: () => void;
  onEdit?: () => void;
  onDismiss: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const hasDetail = !!(item.description || item.tags.length > 0);

  return (
    <div className="group/row">
      <div className="flex flex-wrap items-start gap-x-2 gap-y-1 py-1">
        <div className="flex min-w-0 flex-1 flex-wrap items-baseline gap-x-2 gap-y-0.5">
          <span className="shrink-0 rounded bg-slate-700/80 px-1.5 py-0.5 text-[11px] font-medium text-slate-300">
            {BACKLOG_KIND_LABELS[item.kind] ?? item.kind}
          </span>
          <span className="text-sm text-slate-200">{item.title}</span>
          <span className="shrink-0 text-[11px] text-slate-500">P{item.priority}</span>
          {item.confidence >= 0.8 && (
            <span className="shrink-0 text-[11px] text-emerald-500">{Math.round(item.confidence * 100)}%</span>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-0.5">
          {hasDetail && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setExpanded(!expanded)}
              className="opacity-60 hover:opacity-100"
              aria-label={expanded ? "Collapse details" : "Expand details"}
            >
              {expanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={onAccept}
            disabled={isAccepting}
            title="Accept"
            data-testid={selectors.captures.itemAcceptButton}
          >
            {isAccepting ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Check className="h-3.5 w-3.5 text-emerald-400" />
            )}
          </Button>
          {onEdit && (
            <Button
              variant="ghost"
              size="icon"
              onClick={onEdit}
              title="Edit before adding"
              data-testid={selectors.captures.itemEditButton}
            >
              <Pencil className="h-3.5 w-3.5" />
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={onDismiss}
            title="Dismiss"
            data-testid={selectors.captures.itemDismissButton}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      {expanded && (
        <div className="ml-6 pb-1.5">
          {item.description && (
            <p className="text-xs text-slate-400 line-clamp-2">{item.description}</p>
          )}
          {item.tags.length > 0 && (
            <TagList tags={item.tags} maxTags={4} className="mt-1" />
          )}
        </div>
      )}
    </div>
  );
}

export function CaptureCard({ capture, onEditItem, className }: CaptureCardProps) {
  const [acceptingIndex, setAcceptingIndex] = useState<number | null>(null);
  const [acceptedIndices, setAcceptedIndices] = useState<Set<number>>(new Set());
  const [dismissedIndices, setDismissedIndices] = useState<Set<number>>(new Set());
  const [isRetrying, setIsRetrying] = useState(false);
  const removeCapture = useCaptureStore((s) => s.removeCapture);
  const updateCapture = useCaptureStore((s) => s.updateCapture);
  const upsertBacklogItem = useBacklogStore((s) => s.upsertItem);

  const items = capture.classification?.items ?? [];
  const allResolved = items.length > 0 && items.every((_, i) => acceptedIndices.has(i) || dismissedIndices.has(i));
  const unresolvedCount = items.filter((_, i) => !acceptedIndices.has(i) && !dismissedIndices.has(i)).length;

  const handleAcceptItem = async (item: CaptureClassificationItem, index: number) => {
    setAcceptingIndex(index);
    try {
      const created = await backlogService.create({
        name: toSlug(item.title),
        title: item.title,
        description: item.description,
        kind: item.kind,
        status: "backlog",
        priority: item.priority,
        tags: item.tags,
      });
      upsertBacklogItem(created);
      // Auto-initialize now happens server-side in the Create endpoint.

      const next = new Set(acceptedIndices);
      next.add(index);
      setAcceptedIndices(next);

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
        const item = items[i];
        if (!item) continue;
        await handleAcceptItem(item, i);
      }
    }
  };

  const handleDismissItem = (index: number) => {
    const next = new Set(dismissedIndices);
    next.add(index);
    setDismissedIndices(next);

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
        name: toSlug(item.title),
        title: item.title,
        description: item.description,
        kind: item.kind,
        status: "backlog",
        priority: item.priority,
        tags: item.tags,
      });
    }
  };

  const statusDotColor =
    capture.status === "classifying" ? "bg-cyan-400" :
    capture.status === "failed" ? "bg-red-400" :
    capture.status === "classified" && items.length > 0 ? "bg-emerald-400" :
    "bg-slate-500";

  return (
    <div
      className={className}
      data-testid={selectors.captures.card}
    >
      {/* Header: status dot + capture badge (left), timestamp + dismiss (right) */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          <span className={`inline-block h-2 w-2 rounded-full ${statusDotColor}`} />
          <span className="rounded-full bg-violet-500/20 px-2 py-0.5 text-[11px] font-medium text-violet-300">
            Capture
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="text-[11px] text-slate-600">{formatRelativeTime(capture.created)}</span>
          <button
            onClick={handleDismissCapture}
            className="shrink-0 rounded p-0.5 text-slate-600 transition-colors hover:bg-slate-700 hover:text-slate-400"
            title="Dismiss capture"
            data-testid={selectors.captures.dismissButton}
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
      {/* Title: original capture text */}
      <h3 className="mt-3 font-medium text-slate-100">{capture.text}</h3>

      {/* Classifying */}
      {capture.status === "classifying" && (
        <div className="mt-1 flex items-center gap-1.5 text-xs text-cyan-400">
          <Loader2 className="h-3 w-3 animate-spin" />
          Classifying...
        </div>
      )}

      {/* Failed */}
      {capture.status === "failed" && (
        <div className="mt-1 flex items-center gap-2">
          <span className="text-xs text-red-400">Classification failed</span>
          <button
            onClick={handleRetry}
            disabled={isRetrying}
            className="inline-flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200 disabled:opacity-50"
            data-testid={selectors.captures.retryButton}
          >
            <RefreshCw className={`h-3 w-3 ${isRetrying ? "animate-spin" : ""}`} />
            Retry
          </button>
        </div>
      )}

      {/* No-op: nothing actionable */}
      {capture.status === "classified" && items.length === 0 && (
        <div className="mt-1 flex items-center justify-between">
          <span className="text-xs text-slate-500 italic">Nothing actionable detected</span>
          <button
            onClick={handleDismissCapture}
            className="text-xs text-slate-500 hover:text-slate-300"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Suggestions */}
      {capture.status === "classified" && items.length > 0 && !allResolved && (
        <div className="mt-1.5">
          <div className="flex items-center gap-2 mb-0.5">
            <span className="text-[11px] text-slate-500">
              {unresolvedCount} suggestion{unresolvedCount !== 1 ? "s" : ""}
            </span>
            {unresolvedCount > 1 && (
              <button
                onClick={handleAcceptAll}
                className="inline-flex items-center gap-1 text-[11px] text-cyan-400 hover:text-cyan-300"
                data-testid={selectors.captures.acceptAllButton}
              >
                <CheckCheck className="h-3 w-3" />
                Accept all
              </button>
            )}
          </div>

          <div className="divide-y divide-slate-700/40">
            {items.map((item, index) => {
              if (acceptedIndices.has(index) || dismissedIndices.has(index)) return null;
              return (
                <SuggestionRow
                  key={index}
                  item={item}
                  isAccepting={acceptingIndex === index}
                  onAccept={() => handleAcceptItem(item, index)}
                  onEdit={onEditItem ? () => handleEditItem(item, index) : undefined}
                  onDismiss={() => handleDismissItem(index)}
                />
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
