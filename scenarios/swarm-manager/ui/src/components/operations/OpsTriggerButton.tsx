/**
 * OpsTriggerButton — always-visible entry point to the Operations Center.
 *
 * Lives in the sidebar header and the graph HUD as a peer to the other
 * navigation buttons. Two visual variants keep the trigger at home in
 * both contexts:
 *
 *   - `compact` — slim pill used in the sidebar header. Mirrors the
 *     metric/badge density of the surrounding header buttons.
 *   - `hud`     — bordered button used in the graph HUD row. Visually
 *     matches the other HUD pills (Settings / Stats / Help) so the
 *     trigger reads as a peer button rather than a notification chip.
 *
 * Both variants always render — when no agents are running the label
 * reads "0 agents" rather than collapsing. The button is the canonical
 * "where do I go to see what's running?" surface; hiding it on idle
 * defeats that purpose. Click navigates to `/operations` via
 * react-router's `<Link>` so middle-click and modifier-click open the
 * page in a new tab the way operators expect from a navigation control.
 *
 * The agent count comes from `useOperationsStore` so the trigger and the
 * Operations Center page agree on the value to within one polling tick.
 * AppShell mounts a global operations poll (see `useOperationsPolling`)
 * so the count stays fresh wherever the trigger renders.
 */

import { Bot } from "lucide-react";
import { Link } from "react-router-dom";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import { selectActiveCount, useOperationsStore } from "../../stores/operations-store";

export type OpsTriggerVariant = "compact" | "hud";

export interface OpsTriggerButtonProps {
  /**
   * Visual variant: `compact` for the sidebar header, `hud` for the graph HUD.
   * Both variants share the same `data-testid` so workflow tooling can locate
   * the trigger regardless of layout context.
   */
  variant: OpsTriggerVariant;
  /** Additional class names — pass HUD-specific responsive utilities here. */
  className?: string;
}

function pluralize(count: number): string {
  return count === 1 ? "agent" : "agents";
}

export function OpsTriggerButton({ variant, className }: OpsTriggerButtonProps) {
  const count = useOperationsStore(selectActiveCount);
  const label = `${count} ${pluralize(count)}`;
  const ariaLabel = `Operations Center · ${label}`;
  const title = `${label} — open Operations Center`;

  if (variant === "compact") {
    return (
      <Link
        to="/operations"
        className={cn(
          "flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium transition-colors",
          count > 0
            ? "bg-emerald-500/15 text-emerald-300 hover:bg-emerald-500/25"
            : "bg-slate-800/60 text-slate-400 hover:bg-slate-700/60",
          className,
        )}
        aria-label={ariaLabel}
        title={title}
        data-testid={selectors.layout.opsTriggerButton}
        data-variant="compact"
      >
        <Bot
          className={cn("h-3.5 w-3.5", count > 0 && "animate-pulse")}
          aria-hidden="true"
        />
        <span>{label}</span>
      </Link>
    );
  }

  return (
    <Link
      to="/operations"
      className={cn(
        "flex items-center gap-1.5 rounded-lg border border-slate-700/60 bg-slate-900/80 px-2.5 py-1.5 text-sm text-slate-100 transition-colors hover:bg-slate-800/80",
        className,
      )}
      aria-label={ariaLabel}
      title={title}
      data-testid={selectors.layout.opsTriggerButton}
      data-variant="hud"
    >
      <Bot
        className={cn("h-4 w-4 text-cyan-300", count > 0 && "animate-pulse")}
        aria-hidden="true"
      />
      <span className="rounded-full bg-cyan-500/20 px-1.5 py-0.5 text-xs text-cyan-200">
        {count}
      </span>
      <span className="hidden text-xs text-slate-300 sm:inline">
        {pluralize(count)}
      </span>
    </Link>
  );
}
