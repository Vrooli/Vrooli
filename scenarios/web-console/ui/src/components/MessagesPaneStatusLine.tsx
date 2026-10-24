import { AlertCircle, Check, Loader2 } from "lucide-react";
import { cn } from "../lib/classnames";

/**
 * The Messages pane's one-line status region.
 *
 * This replaced two independently-rendered banners that could — and did — stack
 * on top of each other and contradict: a full-width "Live updates disconnected"
 * card sitting directly above an equally-heavy "Up to date" card. Two boxes of
 * content weight, saying opposite things, pushing the conversation down the
 * screen.
 *
 * The rules that follow from that:
 *
 *   - At most one message is ever shown, chosen by priority. A transient
 *     confirmation must never appear while something is actually wrong.
 *   - The region reserves no height when idle and stays a single line when
 *     active, so showing it does not shove the transcript around.
 *   - Weight tracks severity: a confirmation is quiet text, a fault is tinted.
 */

import type { MessagesPaneStatus } from "../lib/messagesPaneStatus";

interface MessagesPaneStatusLineProps {
  status: MessagesPaneStatus | null;
}

export default function MessagesPaneStatusLine({ status }: MessagesPaneStatusLineProps) {
  if (!status) return null;

  const icon =
    status.kind === "error" ? <AlertCircle className="h-3 w-3 shrink-0" aria-hidden="true" />
      : status.kind === "disconnected" ? <Loader2 className="h-3 w-3 shrink-0 animate-spin" aria-hidden="true" />
        : <Check className="h-3 w-3 shrink-0" aria-hidden="true" />;

  return (
    <div
      data-testid="messages-status-line"
      data-status-kind={status.kind}
      role="status"
      aria-live="polite"
      className={cn(
        "flex items-center gap-1.5 px-2 py-1 text-[11px] leading-none",
        status.kind === "error" && "text-wc-error-text",
        status.kind === "disconnected" && "text-wc-text-secondary",
        status.kind === "success" && "text-wc-text-muted",
      )}
    >
      {icon}
      <span className="truncate">{status.text}</span>
    </div>
  );
}
