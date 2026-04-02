/**
 * ActionFeedItem — Individual actionable item card in the Command Post feed.
 *
 * Renders title, kind badge, attention reason badges, primary CTA, and snooze control.
 * Click on card body navigates to the detail page.
 */

import { Clock, Play, MessageCircle, Wrench, Sparkles, Eye } from "lucide-react";
import { Button } from "../ui/button";
import { BACKLOG_KIND_ICONS } from "../../types/constants";
import { SnoozePopover } from "./SnoozePopover";
import type { ActionableItem } from "../../lib/command-post-utils";
import type { AttentionReason } from "../../lib/feed";


interface ActionFeedItemProps {
  item: ActionableItem;
  onNavigate: () => void;
  onSnooze: (key: string, expiresAt: number) => void;
  onRun?: (kind: string, name: string) => void;
  onFollowUp?: (executionId: string) => void;
  onEnterDecisionStream?: () => void;
}

function reasonLabel(reason: AttentionReason): string {
  switch (reason.kind) {
    case "pending-decisions":
      return `${reason.count} decision${reason.count !== 1 ? "s" : ""}`;
    case "plan-ready":
      return "Plan ready";
    case "research-complete":
      return "Research complete";
  }
}

const CTA_CONFIG: Record<string, { label: string; icon: React.ElementType }> = {
  run: { label: "Run", icon: Play },
  workshop: { label: "Workshop", icon: MessageCircle },
  finalize: { label: "Finalize", icon: Sparkles },
  followUp: { label: "Follow Up", icon: Wrench },
  archive: { label: "Archive", icon: Eye },
};

function handleCtaClick(
  e: React.MouseEvent,
  item: ActionableItem,
  props: Pick<ActionFeedItemProps, "onRun" | "onFollowUp" | "onEnterDecisionStream" | "onNavigate">,
) {
  e.stopPropagation();
  const cta = item.primaryCta;
  if (cta === "run" && item.kind && item.name && props.onRun) {
    props.onRun(item.kind, item.name);
  } else if (cta === "followUp" && item.executionId && props.onFollowUp) {
    props.onFollowUp(item.executionId);
  } else if ((cta === "workshop" || cta === "finalize") && props.onEnterDecisionStream) {
    props.onEnterDecisionStream();
  } else {
    props.onNavigate();
  }
}

export function ActionFeedItem({
  item,
  onNavigate,
  onSnooze,
  onRun,
  onFollowUp,
  onEnterDecisionStream,
}: ActionFeedItemProps) {
  const KindIcon = item.kind ? BACKLOG_KIND_ICONS[item.kind] : null;
  const ctaConfig = item.primaryCta ? CTA_CONFIG[item.primaryCta as string] : null;

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onNavigate}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") onNavigate();
      }}
      className="flex cursor-pointer items-center gap-3 rounded-lg border border-slate-700/40 bg-slate-900/40 p-3 transition-colors hover:bg-slate-800/60"
      data-testid={`action-feed-item-${item.key}`}
    >
      {/* Kind icon */}
      {KindIcon && <KindIcon className="h-4 w-4 shrink-0 text-slate-400" />}

      {/* Content */}
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-slate-200">{item.title}</p>
        {item.reasons.length > 0 && (
          <div className="mt-1 flex flex-wrap gap-1">
            {item.reasons.map((reason, i) => (
              <span
                key={i}
                className="rounded-full bg-amber-500/15 px-2 py-0.5 text-xs text-amber-300"
              >
                {reasonLabel(reason)}
              </span>
            ))}
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex shrink-0 items-center gap-1">
        {ctaConfig && (
          <Button
            variant="outline"
            size="sm"
            onClick={(e) =>
              handleCtaClick(e, item, { onRun, onFollowUp, onEnterDecisionStream, onNavigate })
            }
            data-testid={`action-feed-cta-${item.key}`}
          >
            <ctaConfig.icon className="mr-1 h-3.5 w-3.5" />
            {ctaConfig.label}
          </Button>
        )}
        <SnoozePopover itemKey={item.key} onSnooze={onSnooze}>
          <Clock className="h-3.5 w-3.5" />
        </SnoozePopover>
      </div>
    </div>
  );
}
