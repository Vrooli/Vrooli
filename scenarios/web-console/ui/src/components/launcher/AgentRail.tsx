// DOC: docs/reference/configuration.md#launcher-shortcuts
/**
 * The agent choice: a horizontal rail of harness cards with a peek at the next.
 *
 * Cards are one height whether loading, loaded or without usage. Their height is
 * `slots` usage rows, and `slots` is decided before any reading arrives, so
 * nothing under a thumb moves when one does.
 */
import type { KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";

import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { slugify } from "../../lib/slugify";
import { HorizontalRail } from "../HorizontalRail";
import AgentMark from "./AgentMark";
import { agentAppearance } from "./agentAppearance";
import { cardSelectable, type AgentCard } from "./agentGrid";
import { CardUsage } from "./AgentUsage";
import { handleRadioKeys } from "./radioKeys";
import { FOCUS_RING } from "./controls";
import type { AgentUsage } from "./useAgentUsage";

/** Fixed so the peek and the snap land the same on every card. */
const CARD = "flex flex-col gap-2.5 rounded-[var(--radius-panel)] border p-2.5 text-start transition-colors";

export interface AgentRailProps {
  cards: readonly AgentCard[];
  targetKind: string;
  selectedAgentId: string | null;
  onSelect: (agentId: string) => void;
  disabled: boolean;
  usages: Readonly<Record<string, AgentUsage | null>>;
  usageLoading: boolean;
  slots: number;
  declaredWindows: (agentId: string) => number;
  needsAction: (agentId: string) => boolean;
  /** Lay the cards out as a grid, for a surface wide enough to show them all. */
  wrap?: boolean;
}

export function AgentRail({ cards, targetKind, selectedAgentId, onSelect, disabled, usages, usageLoading, slots, declaredWindows, needsAction, wrap = false }: AgentRailProps) {
  const { t } = useTranslation();
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => { handleRadioKeys(event, (index) => { const card = cards[index]; if (card) onSelect(card.agentID); }); };
  const items = (
    <>
      {cards.map((card) => {
        const selected = card.agentID === selectedAgentId;
        const flagged = needsAction(card.agentID);
        return (
          <button key={card.key} type="button" role="radio" aria-checked={selected} tabIndex={selected ? 0 : -1}
            data-testid={`launcher-agent-${slugify(card.label)}`} data-capability-id={card.agentID} data-capability-state={card.state} data-capability-version={card.version}
            disabled={disabled || !cardSelectable(card, targetKind)} title={card.command}
            onClick={() => { onSelect(card.agentID); }}
            className={cn(CARD, wrap ? "min-w-0" : "w-[164px] flex-none", FOCUS_RING, "disabled:cursor-not-allowed disabled:opacity-60",
              selected ? "border-wc-accent bg-wc-accent/10 shadow-[inset_0_0_0_1px_rgb(var(--wc-accent))]" : "border-wc-default bg-wc-surface-input hover:border-wc-hover")}>
            <span className="flex min-w-0 items-center gap-[7px]">
              <AgentMark mark={card.agentID} appearance={agentAppearance(card.agentID)} />
              <span className="min-w-0 flex-1 truncate text-[13px] font-semibold text-wc-text-primary">{card.label}</span>
              {flagged && <span data-testid={`launcher-agent-flag-${slugify(card.key)}`} className="h-2 w-2 flex-none rounded-full bg-amber-400"><span className="sr-only">{t(strings.launcher.agentHasAction)}</span></span>}
            </span>
            <CardUsage usage={usages[card.agentID]} loading={usageLoading} slots={slots} declared={declaredWindows(card.agentID)} />
          </button>
        );
      })}
    </>
  );
  if (wrap) {
    return <div data-testid="launcher-agent-grid" role="radiogroup" aria-label={t(strings.launcher.chooseAgent)} onKeyDown={onKeyDown} className="grid grid-cols-3 gap-2 px-4">{items}</div>;
  }
  return <HorizontalRail testId="launcher-agent-grid" role="radiogroup" aria-label={t(strings.launcher.chooseAgent)} gapClassName="gap-2" gutterClassName="ps-4 scroll-ps-4" onKeyDown={onKeyDown}>{items}</HorizontalRail>;
}

/** The rail's heading, with the way into Manage and a count of what needs doing there. */
export function AgentRailHeader({ actionCount, onManage, manageRef }: { actionCount: number; onManage: () => void; manageRef: React.Ref<HTMLButtonElement> }) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-between gap-2 px-4 pt-4">
      <h3 id="launcher-agents-heading" className="text-[11px] font-semibold uppercase tracking-[0.08em] text-wc-text-muted">{t(strings.launcher.agents)}</h3>
      <button ref={manageRef} type="button" data-testid="launcher-manage-agents" onClick={onManage}
        aria-label={actionCount > 0 ? t(strings.launcher.manageWithActions, { count: actionCount }) : t(strings.launcher.manage)}
        className={cn("flex min-h-11 items-center gap-1.5 rounded-[var(--radius-control)] px-1 text-[13px] font-semibold text-wc-accent", FOCUS_RING)}>
        {t(strings.launcher.manage)}
        {actionCount > 0 && <span data-testid="launcher-manage-count" className="min-w-[18px] rounded-full bg-amber-400 px-[5px] text-center text-[11px] font-bold leading-[18px] tabular-nums text-amber-950">{actionCount}</span>}
      </button>
    </div>
  );
}
