/**
 * CaptureTriage — Shared triage component for classified captures.
 *
 * Renders the suggestion list with per-item accept/edit/dismiss actions
 * and an "Accept all" batch action. Used by both CaptureCard (sidebar)
 * and CaptureDetailsPage (detail overlay).
 */

import { useState } from "react";
import { Check, CheckCheck, ChevronDown, ChevronUp, Loader2, Pencil, X } from "lucide-react";
import { Button } from "../ui/button";
import { TagList } from "../ui/tag-list";
import { backlogService } from "../../services/backlog-service";
import { captureService } from "../../services/capture-service";
import { useCaptureStore } from "../../stores/capture-store";
import { useBacklogStore } from "../../stores";
import { BACKLOG_KIND_LABELS } from "../../types";
import { selectors } from "../../consts/selectors";
import type { Capture, CaptureClassificationItem } from "../../types";
import type { BacklogFormValues } from "../../types";

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

interface CaptureTriageProps {
  capture: Capture;
  onEditItem?: (prefill: BacklogFormValues) => void;
  /** Called after all items are resolved (accepted or dismissed) and capture is removed. */
  onCaptureResolved?: () => void;
}

export function CaptureTriage({ capture, onEditItem, onCaptureResolved }: CaptureTriageProps) {
  const [acceptingIndex, setAcceptingIndex] = useState<number | null>(null);
  const [acceptedIndices, setAcceptedIndices] = useState<Set<number>>(new Set());
  const [dismissedIndices, setDismissedIndices] = useState<Set<number>>(new Set());
  const removeCapture = useCaptureStore((s) => s.removeCapture);
  const upsertBacklogItem = useBacklogStore((s) => s.upsertItem);

  const items = capture.classification?.items ?? [];
  const unresolvedCount = items.filter((_, i) => !acceptedIndices.has(i) && !dismissedIndices.has(i)).length;

  if (items.length === 0) return null;

  const resolveIfDone = async (nextAccepted: Set<number>, nextDismissed: Set<number>) => {
    if (items.every((_, i) => nextAccepted.has(i) || nextDismissed.has(i))) {
      await captureService.remove(capture.id);
      removeCapture(capture.id);
      onCaptureResolved?.();
    }
  };

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
        suggestedSkills: [],
      });
      upsertBacklogItem(created);

      const next = new Set(acceptedIndices);
      next.add(index);
      setAcceptedIndices(next);

      await resolveIfDone(next, dismissedIndices);
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

    resolveIfDone(acceptedIndices, next);
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

  if (unresolvedCount === 0) return null;

  return (
    <div>
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
  );
}
