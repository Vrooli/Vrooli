/**
 * ActionButton — a control that says what it is about to do.
 *
 * Operator actions in this app range from flipping a status field to
 * dispatching an autonomous agent that runs for minutes and spends tokens.
 * They were all rendered as the same cyan button with a label, so the only way
 * to find out which kind you had was to press it.
 *
 * This button reads the action's consequence class (see `lib/action-semantics`,
 * which derives it from the server's transition registry) and encodes it:
 *
 *   agent_workflow / agent_session → a bot glyph beside the label
 *   destructive                    → rose styling and a trailing ellipsis
 *   state_change / navigation      → the action's own icon, unadorned
 *
 * The bot glyph is the important one. It is the difference between "this
 * updates a field" and "this starts an agent", visible before the click.
 */

import { Bot, Loader2, MessagesSquare } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { cn } from "../../lib/utils";
import {
  CONSEQUENCE_META,
  consequenceOf,
  type ConsequenceClass,
  type ConsequenceInput,
} from "../../lib/action-semantics";

export interface ActionButtonProps extends ConsequenceInput {
  label: string;
  onClick: () => void;
  /** The action's own icon, shown when it carries no agent marker. */
  icon?: LucideIcon;
  pending?: boolean;
  disabled?: boolean;
  /** Label shown while the action is in flight. */
  pendingLabel?: string;
  size?: "sm" | "default";
  className?: string;
  "data-testid"?: string;
  /** Overrides the derived tooltip. */
  title?: string;
}

const CONSEQUENCE_ICON: Partial<Record<ConsequenceClass, LucideIcon>> = {
  agent_workflow: Bot,
  agent_session: MessagesSquare,
};

const BASE = "inline-flex items-center justify-center gap-1.5 rounded-full font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-60";

const TONE: Record<"destructive" | "agent" | "plain", string> = {
  destructive: "bg-rose-600/90 text-white hover:bg-rose-600 focus-visible:ring-rose-500/50",
  agent: "bg-cyan-600/90 text-white hover:bg-cyan-600 focus-visible:ring-cyan-500/40",
  plain: "bg-slate-50 text-slate-900 hover:bg-slate-100 focus-visible:ring-cyan-500/40",
};

const SIZE = {
  sm: "h-9 px-4 text-sm",
  default: "h-11 px-5 text-sm",
};

export function ActionButton({
  label,
  onClick,
  icon,
  pending = false,
  disabled = false,
  pendingLabel = "Working…",
  size = "default",
  className,
  title,
  actionId,
  transitionKind,
  destructive,
  "data-testid": testId,
}: ActionButtonProps) {
  const consequence = consequenceOf({ actionId, transitionKind, destructive });
  const meta = CONSEQUENCE_META[consequence];
  const ConsequenceIcon = CONSEQUENCE_ICON[consequence];
  // The agent marker replaces the action's own icon rather than sitting beside
  // it: two glyphs on a small button read as decoration, and the one that
  // matters is the one saying an agent is about to run.
  const Icon = ConsequenceIcon ?? icon;
  const tone = consequence === "destructive" ? "destructive" : meta.spawnsAgent ? "agent" : "plain";

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled || pending}
      // The tooltip carries the consequence in words, for anyone who does not
      // read the glyph and for screen readers via the accessible description.
      title={title ?? meta.hint}
      data-testid={testId}
      data-consequence={consequence}
      className={cn(BASE, SIZE[size], TONE[tone], className)}
    >
      {pending
        ? <Loader2 className="h-4 w-4 shrink-0 animate-spin motion-reduce:animate-none" aria-hidden />
        : Icon ? <Icon className="h-4 w-4 shrink-0" aria-hidden /> : null}
      <span className="truncate">
        {pending ? pendingLabel : `${label}${consequence === "destructive" ? "…" : ""}`}
      </span>
    </button>
  );
}

/**
 * A standalone marker for surfaces that list actions without rendering a
 * button for each — a queue row, a menu item, a confirmation summary.
 */
export function ConsequenceBadge({ actionId, transitionKind, destructive, className }: ConsequenceInput & { className?: string }) {
  const consequence = consequenceOf({ actionId, transitionKind, destructive });
  const meta = CONSEQUENCE_META[consequence];
  if (!meta.spawnsAgent) return null;
  const Icon = CONSEQUENCE_ICON[consequence] ?? Bot;

  return (
    <span
      className={cn("inline-flex items-center gap-1 rounded-full border border-cyan-500/30 bg-cyan-500/10 px-2 py-0.5 text-[11px] font-medium text-cyan-300", className)}
      data-testid="consequence-badge"
      data-consequence={consequence}
    >
      <Icon className="h-3 w-3" aria-hidden />
      {consequence === "agent_session" ? "Agent session" : "Agent run"}
    </span>
  );
}
