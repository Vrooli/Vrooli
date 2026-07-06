/**
 * PlanCardView — renders one plan-board card (item / gate / outcome) on the
 * shared BoardCard primitive. Phase 2 is intentionally action-light: cards
 * navigate to their detail surface; menus and levers land with the
 * Operations/Command-Post absorption phases.
 */

import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  backlogDetailPath,
  captureDetailPath,
  executionDetailPath,
} from "../../../app/routes/route-paths";
import { BoardCard, type BoardCardTone } from "../../../components/cards/BoardCard";
import { cn } from "../../../lib/utils";
import { formatRelativeTime } from "../../../lib";
import { gateActionLabel, outcomeGlyph, waveBadgeLabel } from "../lib/plan-presentation";
import { CYCLE_WAVE, type PlanCardData } from "../types";
import { PlanCardMenu } from "./PlanCardMenu";
import { usePlanCardActions } from "./plan-card-actions-context";

const GATE_TONE: Record<string, BoardCardTone> = {
  decide: "attention",
  review: "attention",
  classify: "attention",
  workshop: "neutral",
};

const OUTCOME_TONE: Record<string, BoardCardTone> = {
  ok: "positive",
  failed: "negative",
  needs_review: "attention",
  needs_followup: "attention",
};

function cardTone(card: PlanCardData): BoardCardTone {
  if (card.cardType === "gate" && card.gate) {
    return GATE_TONE[card.gate.kind] ?? "attention";
  }
  if (card.cardType === "outcome") {
    return OUTCOME_TONE[card.outcome] ?? "neutral";
  }
  if (card.action === "run") return "active";
  return "neutral";
}

function cardSubtitle(card: PlanCardData): string {
  const parts: string[] = [];
  if (card.cardType === "gate" && card.gate) {
    parts.push(gateActionLabel(card.gate));
    if (card.unblocks > 0) {
      parts.push(`unblocks ${card.unblocks}`);
    }
  } else if (card.cardType === "outcome") {
    parts.push(card.status.replaceAll("_", " "));
    if (card.finishedAt) parts.push(formatRelativeTime(card.finishedAt));
  } else {
    if (card.itemKind) parts.push(card.itemKind);
    parts.push(card.action === "none" ? card.status.replaceAll("_", " ") : card.action);
    if (card.unblocks > 0) parts.push(`unblocks ${card.unblocks}`);
  }
  if (card.initiative) parts.push(card.initiative);
  return parts.join(" · ");
}

export interface PlanCardViewProps {
  card: PlanCardData;
  /** Show the wave badge (Later column). */
  showWave?: boolean;
  /** Render dimmed (snoozed card with show-snoozed on). */
  dimmed?: boolean;
}

export function PlanCardView({ card, showWave = false, dimmed = false }: PlanCardViewProps) {
  const navigate = useNavigate();
  const actions = usePlanCardActions();

  const handleOpen = useCallback(() => {
    // Decide gates open the decision drawer scoped to the item; other
    // cards navigate to their owning detail surface.
    if (card.cardType === "gate" && card.gate?.kind === "decide" && actions && card.itemKind && card.itemName) {
      actions.openDecisions(`${card.itemKind}/${card.itemName}`);
      return;
    }
    if (card.executionId) {
      navigate(executionDetailPath(card.executionId));
      return;
    }
    if (card.id.startsWith("capture/")) {
      navigate(captureDetailPath(card.id.slice("capture/".length)));
      return;
    }
    if (card.itemKind && card.itemName) {
      navigate(backlogDetailPath(card.itemKind, card.itemName));
    }
  }, [card, navigate, actions]);

  const badges = (
    <>
      {card.cardType === "gate" && card.gate ? (
        <span
          className="rounded bg-amber-500/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-amber-300"
          data-testid="plan-card-gate-badge"
        >
          {card.gate.kind}
          {card.gate.count > 1 ? ` ${card.gate.count}` : ""}
        </span>
      ) : null}
      {card.cardType === "outcome" ? (
        <span className="text-sm" aria-hidden data-testid="plan-card-outcome-glyph">
          {outcomeGlyph(card.outcome)}
        </span>
      ) : null}
      {card.cardType === "item" && card.effort ? (
        <span
          className="rounded bg-indigo-500/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-indigo-300"
          data-testid="plan-card-effort-badge"
          title={`effort ${card.effort}`}
        >
          {card.effort}
        </span>
      ) : null}
      {showWave ? (
        <span
          className={cn(
            "rounded px-1.5 py-0.5 text-[10px] font-medium",
            card.wave === CYCLE_WAVE
              ? "bg-rose-500/15 text-rose-300"
              : "bg-slate-700/60 text-slate-400",
          )}
          data-testid="plan-card-wave-badge"
        >
          {waveBadgeLabel(card.wave)}
        </span>
      ) : null}
    </>
  );

  return (
    <BoardCard
      title={card.title}
      subtitle={cardSubtitle(card)}
      tone={cardTone(card)}
      badges={badges}
      action={<PlanCardMenu card={card} />}
      onClick={handleOpen}
      dimmed={dimmed}
      testId={`plan-card-${card.id}`}
    />
  );
}
