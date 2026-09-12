// DOC: docs/reference/configuration.md#launcher-shortcuts
/**
 * The "Agents" block of the New Terminal dialog.
 *
 * Extracted from TerminalLauncher because the grid now carries four card
 * states, an install action, and a reorder mode — three concerns that have
 * nothing to do with destinations, appearance, or session policy, which is
 * everything else that dialog owns.
 *
 * Two structural decisions worth knowing before editing:
 *
 *   A card is a <div>, not a <button>. The install affordance is a control
 *   inside the card, and a button inside a button is invalid markup that
 *   browsers resolve by dropping one of them. The card's body is its own
 *   button; the rail beneath it is another.
 *
 *   Every state renders at the same footprint, and there is exactly ONE rail.
 *   Install, progress, and every outcome are alternatives chosen by railFor,
 *   not independent conditionals — which is what let a card show "Install"
 *   and "Installed" at the same time. Acting on one card also never reflows
 *   the grid, because the rail's footprint does not change.
 *
 * [REQ:P0-006a] Terminal Launch Flow UI
 * [REQ:P0-014a] Launcher Destination And Appearance Disclosure
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Check, CircleAlert, HelpCircle } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../../consts/strings";
import type { ShortcutEntry } from "../../consts/shortcuts";
import type { InstallOutcome } from "../../api/capabilities";
import type { TerminalTarget } from "../../api/targets";
import { slugify } from "../../lib/slugify";
import { cn } from "../../lib/classnames";
import AgentMark, { ReorderGrip } from "./AgentMark";
import { MISSING_APPEARANCE, agentAppearance } from "./agentAppearance";
import {
  applyAgentOrderToShortcuts,
  buildAgentGrid,
  cardInstalls,
  cardLaunches,
  type AgentCard,
} from "./agentGrid";
import { useListReorder } from "./useListReorder";

/** Shared card shell. Overflow hidden so the rail clips to the rounded corner. */
const CARD_SHELL =
  "relative flex min-w-0 flex-col overflow-hidden rounded-xl border border-wc-default bg-wc-surface-input transition";

/** The pressable body of a card. */
const CARD_BODY =
  "flex min-h-[54px] w-full flex-col gap-1.5 px-2.5 pb-2 pt-2.5 text-start disabled:cursor-not-allowed";

/** How long the transient "Installed" confirmation stays on the card. */
const CONFIRMATION_MS = 6000;

/**
 * The one rail a card can carry. "none" is a real member rather than an
 * absent value so every branch has to be answered.
 */
type RailKind = "none" | "install" | "progress" | "installed" | "unconfirmed" | "failed" | "unsupported";

/** Drops keys from a record without mutating it or deleting computed keys. */
function without(source: Record<string, InstallOutcome>, keys: string[]): Record<string, InstallOutcome> {
  const drop = new Set(keys);
  return Object.fromEntries(Object.entries(source).filter(([key]) => !drop.has(key)));
}

/** Shared rail geometry, so switching between rails never resizes the card. */
// Border colour is deliberately NOT set here: each rail supplies its own, and
// two border-colour utilities on one element resolve by stylesheet order
// rather than class order, so the shared one would win unpredictably.
const RAIL_BASE = "w-full border-t px-2 py-1.5 text-center text-[11.5px] font-semibold transition";

/** Status dot colour per state — semantic, deliberately not the accent hue. */
const DOT_CLASS: Record<AgentCard["state"], string> = {
  ready: "bg-wc-success",
  missing: "bg-wc-warning",
  installing: "bg-wc-accent",
  "not-applicable": "bg-wc-text-faint",
  unknown: "bg-wc-text-faint",
};

export interface AgentGridSectionProps {
  shortcuts: readonly ShortcutEntry[];
  target: TerminalTarget;
  /** True when nothing in the grid may be pressed (creating, offline, …). */
  disabled: boolean;
  onLaunch: (command?: string) => void;
  /**
   * Runs the governed installer and resolves with the machine's own verdict.
   * The card never infers success from the call resolving: "the installer ran"
   * and "the agent is there" are different claims, and only the second one is
   * an install the operator can act on.
   */
  onInstall?: (agentID: string) => Promise<InstallOutcome>;
  onEditShortcuts?: () => void;
  /** Persists a new order. Absent means the grid offers no reorder mode. */
  onReorder?: (next: ShortcutEntry[]) => void | Promise<void>;
}

export default function AgentGridSection({
  shortcuts,
  target,
  disabled,
  onLaunch,
  onInstall,
  onEditShortcuts,
  onReorder,
}: AgentGridSectionProps) {
  const { t } = useTranslation();
  const [reordering, setReordering] = useState(false);
  const [installing, setInstalling] = useState<string[]>([]);
  const [outcomes, setOutcomes] = useState<Record<string, InstallOutcome>>({});
  const confirmationTimers = useRef<number[]>([]);

  useEffect(() => () => {
    for (const timer of confirmationTimers.current) window.clearTimeout(timer);
  }, []);

  const { agents, commands } = useMemo(
    () => buildAgentGrid({ readiness: target.readiness, shortcuts, installing }),
    [installing, shortcuts, target.readiness],
  );

  const commitOrder = useCallback((next: AgentCard[]) => {
    if (!onReorder) return;
    void onReorder(applyAgentOrderToShortcuts(shortcuts, next));
  }, [onReorder, shortcuts]);

  const reorder = useListReorder<AgentCard>({ source: agents, onCommit: commitOrder, enabled: reordering });

  const startInstall = useCallback(async (agentID: string) => {
    if (!onInstall) return;
    // Clear any previous verdict first: a retry that still showed the last
    // failure underneath its own progress bar would be reporting two answers.
    setOutcomes((current) => (agentID in current ? without(current, [agentID]) : current));
    setInstalling((current) => (current.includes(agentID) ? current : [...current, agentID]));
    let outcome: InstallOutcome;
    try {
      outcome = await onInstall(agentID);
    } catch (error) {
      // A transport failure is a failed install with no message from the
      // machine, which is still more than the operator had before.
      console.error("capability install failed", agentID, error);
      outcome = { status: "failed" };
    }
    setInstalling((current) => current.filter((id) => id !== agentID));
    setOutcomes((current) => ({ ...current, [agentID]: outcome }));
    // Only the success is transient. It answers "did that work?" and then gets
    // out of the way of the card's real state. A failure or an unconfirmed
    // install carries the only explanation the operator will get, so it stays
    // until they retry it or the machine reports the agent.
    if (outcome.status === "installed") {
      confirmationTimers.current.push(window.setTimeout(() => {
        setOutcomes((current) => (current[agentID]?.status === "installed" ? without(current, [agentID]) : current));
      }, CONFIRMATION_MS));
    }
  }, [onInstall]);

  // An outcome describes a machine state at a moment. Once the machine reports
  // the agent as present, the stored verdict is stale by definition — an
  // "unconfirmed" rail left over a working agent is its own kind of lie.
  const readyAgentKey = agents.filter((card) => card.state === "ready").map((card) => card.agentID).join(",");
  useEffect(() => {
    const ready = new Set(readyAgentKey ? readyAgentKey.split(",") : []);
    setOutcomes((current) => {
      const stale = Object.keys(current).filter((id) => ready.has(id) && current[id]?.status !== "installed");
      return stale.length === 0 ? current : without(current, stale);
    });
  }, [readyAgentKey]);

  const statusLine = (card: AgentCard): string => {
    switch (card.state) {
      case "ready":
        return card.version || t(strings.launcher.agentReady);
      case "missing":
        return t(strings.launcher.agentMissing);
      case "installing":
        return t(strings.launcher.agentInstalling);
      case "not-applicable":
        return card.detail || t(strings.launcher.agentUnsupported);
      default:
        return card.description || card.command;
    }
  };

  /**
   * The single rail a card may show, chosen once.
   *
   * The order is the whole point: a decided outcome outranks the install
   * offer, so "Install" and "Installed" cannot appear together no matter what
   * the catalog says. That pairing is what an operator saw when a relayed
   * install completed against a machine that never reported the agent.
   */
  const railFor = (card: AgentCard, installs: boolean): RailKind => {
    if (reordering) return "none";
    if (card.state === "installing") return "progress";
    const outcome = outcomes[card.agentID];
    // A machine can refuse an install the catalog still thinks is offerable —
    // no published build for its platform. Falling through to "none" would
    // remove the button and say nothing, so the refusal gets its own rail.
    if (outcome) return outcome.status === "not_applicable" ? "unsupported" : outcome.status;
    return installs ? "install" : "none";
  };

  const renderRail = (card: AgentCard, rail: RailKind) => {
    const testID = slugify(card.key);
    switch (rail) {
      case "install":
        return (
          <button
            type="button"
            data-testid={`launcher-agent-install-${testID}`}
            disabled={disabled}
            onClick={() => { void startInstall(card.agentID); }}
            className={cn(RAIL_BASE, "border-wc-default bg-wc-warning/10 text-wc-warning hover:bg-wc-warning/20 disabled:cursor-not-allowed disabled:opacity-50")}
          >
            {t(strings.launcher.installAgent)}
          </button>
        );
      case "progress":
        return (
          <span
            role="progressbar"
            aria-label={t(strings.launcher.agentInstalling)}
            data-testid={`launcher-agent-progress-${testID}`}
            className="block h-[3px] w-full overflow-hidden bg-wc-surface-base"
          >
            <span className="block h-full w-2/5 animate-pulse rounded-e-full bg-wc-accent" />
          </span>
        );
      case "installed":
        return (
          <span
            data-testid={`launcher-agent-installed-${testID}`}
            className={cn(RAIL_BASE, "flex items-center justify-center gap-1 border-wc-success/30 bg-wc-success/10 text-wc-success")}
          >
            <Check className="h-3 w-3" aria-hidden />
            {t(strings.launcher.agentInstalled)}
          </span>
        );
      // Unconfirmed and failed are both retryable and both carry the machine's
      // own words in the tooltip, which is the only place the reason exists.
      case "unconfirmed":
      case "failed": {
        const failed = rail === "failed";
        return (
          <button
            type="button"
            data-testid={`launcher-agent-${failed ? "install-failed" : "install-unconfirmed"}-${testID}`}
            title={outcomes[card.agentID]?.message}
            disabled={disabled}
            onClick={() => { void startInstall(card.agentID); }}
            className={cn(
              RAIL_BASE,
              "flex items-center justify-center gap-1 disabled:cursor-not-allowed disabled:opacity-50",
              failed
                ? "border-wc-error-text/30 bg-wc-error-text/10 text-wc-error-text hover:bg-wc-error-text/20"
                : "border-wc-warning/30 bg-wc-warning/10 text-wc-warning hover:bg-wc-warning/20",
            )}
          >
            {failed ? <CircleAlert className="h-3 w-3 shrink-0" aria-hidden /> : <HelpCircle className="h-3 w-3 shrink-0" aria-hidden />}
            <span className="truncate">{failed ? t(strings.launcher.installFailed) : t(strings.launcher.installUnconfirmed)}</span>
            <span className="shrink-0 opacity-70">· {t(strings.launcher.installRetry)}</span>
          </button>
        );
      }
      case "unsupported":
        return (
          <span
            data-testid={`launcher-agent-install-unsupported-${testID}`}
            title={outcomes[card.agentID]?.message}
            className={cn(RAIL_BASE, "border-wc-default bg-wc-surface-base/60 text-wc-text-faint")}
          >
            {t(strings.launcher.agentUnsupported)}
          </span>
        );
      default:
        return null;
    }
  };

  const renderAgentCard = (card: AgentCard, index: number) => {
    const launches = cardLaunches(card);
    const installs = cardInstalls(card) && Boolean(onInstall);
    const inert = card.state === "not-applicable" || (card.state === "missing" && !onInstall);
    const lifted = reordering && reorder.draggingIndex === index && reorder.active;
    const isTarget = reordering && reorder.active && reorder.targetIndex === index && reorder.draggingIndex !== index;
    const rail = railFor(card, installs);

    return (
      <div
        key={card.key}
        ref={(node) => { reorder.registerItem(index, node); }}
        data-testid={`launcher-agent-card-${slugify(card.key)}`}
        data-agent-state={card.state}
        className={cn(
          CARD_SHELL,
          inert && "opacity-50",
          card.state === "missing" && !inert && "border-wc-warning/40",
          isTarget && "border-wc-accent border-dashed",
          lifted && "z-10 rotate-[-1.2deg] scale-[1.03] border-wc-accent shadow-2xl",
        )}
      >
        <button
          type="button"
          data-testid={`launcher-agent-${slugify(card.label)}`}
          data-capability-id={card.agentID || undefined}
          data-capability-state={card.state}
          data-capability-version={card.version}
          className={cn(CARD_BODY, reordering && "ps-7", !reordering && launches && "hover:bg-wc-surface-input/70")}
          title={reordering ? undefined : card.command}
          disabled={disabled || reordering || !launches}
          aria-label={reordering ? undefined : `${card.label} — ${statusLine(card)}`}
          onClick={() => { if (launches) onLaunch(card.command); }}
        >
          <span className="flex min-w-0 items-center gap-2">
            <AgentMark
              mark={card.agentID || "command"}
              muted={inert}
              appearance={card.state === "missing" && !inert ? MISSING_APPEARANCE : agentAppearance(card.agentID)}
            />
            <span className="min-w-0 flex-1 truncate text-[13.5px] font-semibold text-wc-text-primary">{card.label}</span>
            {reordering && (
              <span className="shrink-0 rounded border border-wc-default px-1 font-mono text-[9.5px] text-wc-text-faint">{index + 1}</span>
            )}
          </span>
          <span className="flex min-w-0 items-center gap-1.5">
            <span className={cn("h-[5px] w-[5px] shrink-0 rounded-full", DOT_CLASS[card.state])} aria-hidden />
            <span className="min-w-0 truncate font-mono text-[10.5px] text-wc-text-faint">{statusLine(card)}</span>
          </span>
        </button>

        {reordering && (
          <span
            role="button"
            tabIndex={0}
            aria-label={t(strings.launcher.reorderGrip, { name: card.label })}
            data-testid={`launcher-agent-grip-${slugify(card.key)}`}
            className="absolute inset-y-0 start-0 flex w-6 cursor-grab items-center justify-center touch-none"
            onPointerDown={(event) => { reorder.onGripPointerDown(index, event); }}
            onKeyDown={(event) => {
              // Alt+arrow rather than bare arrows: the grid is a normal tab
              // stop and bare arrows belong to the browser's own navigation.
              if (!event.altKey) return;
              const delta = event.key === "ArrowUp" || event.key === "ArrowLeft" ? -1 : event.key === "ArrowDown" || event.key === "ArrowRight" ? 1 : 0;
              if (delta === 0) return;
              event.preventDefault();
              reorder.moveFocused(index, delta);
            }}
          >
            <ReorderGrip active={lifted} />
          </span>
        )}

        {renderRail(card, rail)}
      </div>
    );
  };

  return (
    <section aria-labelledby="launcher-actions-heading" className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <h3 id="launcher-actions-heading" className="text-sm font-semibold text-wc-text-primary">
          {t(strings.launcher.agents)}
        </h3>
        <div className="flex items-center gap-3">
          {onReorder && agents.length > 1 && (
            <button
              type="button"
              data-testid="launcher-reorder-toggle"
              className="text-xs font-semibold text-wc-accent"
              aria-pressed={reordering}
              onClick={() => {
                setReordering((value) => {
                  if (value) reorder.reset();
                  return !value;
                });
              }}
            >
              {reordering ? t(strings.launcher.reorderDone) : t(strings.launcher.reorder)}
            </button>
          )}
        </div>
      </div>

      {/* Two columns at every width. A single column of full-width rows is a
          list you read one entry at a time; the grid is one you scan. */}
      <div data-testid="launcher-agent-grid" className="grid grid-cols-2 items-start gap-2">
        {(reordering ? reorder.items : agents).map(renderAgentCard)}

        {!reordering && (
          <button
            type="button"
            data-testid="launcher-empty-shell"
            onClick={() => { onLaunch(); }}
            disabled={disabled}
            className={cn(CARD_SHELL, CARD_BODY, "border-wc-accent/40 hover:bg-wc-surface-input/70 disabled:opacity-50")}
          >
            <span className="flex min-w-0 items-center gap-2">
              <AgentMark mark="shell" appearance={{ plate: "#0f2c38", ink: "rgb(var(--wc-accent))" }} />
              <span className="min-w-0 flex-1 truncate text-[13.5px] font-semibold text-wc-text-primary">
                {t(strings.terminalLauncher.emptyShell)}
              </span>
            </span>
            <span className="min-w-0 truncate font-mono text-[10.5px] text-wc-text-faint">
              {t(strings.launcher.emptyShellStatus)}
            </span>
          </button>
        )}

        {!reordering && onEditShortcuts && (
          <button
            type="button"
            data-testid="launcher-edit-shortcuts"
            onClick={onEditShortcuts}
            className={cn(CARD_SHELL, CARD_BODY, "justify-center border-dashed bg-transparent hover:border-wc-accent")}
          >
            <span className="flex min-w-0 items-center gap-2">
              <AgentMark mark="edit" appearance={{ plate: "transparent", ink: "rgb(var(--wc-text-faint))" }} />
              <span className="min-w-0 flex-1 truncate text-[13.5px] font-medium text-wc-text-secondary">
                {t(strings.launcher.editShortcuts)}
              </span>
            </span>
            <span className="min-w-0 truncate font-mono text-[10.5px] text-wc-text-faint">
              {t(strings.launcher.editShortcutsStatus)}
            </span>
          </button>
        )}
      </div>

      {reordering && (
        <p data-testid="launcher-reorder-hint" className="flex items-start gap-2 rounded-lg border border-wc-default/60 bg-wc-surface-base/40 px-2.5 py-2 text-[11.5px] text-wc-text-muted">
          {t(strings.launcher.reorderHint)}
        </p>
      )}

      {/* Anything the operator added to their shortcut list that is not one of
          the agents above still gets a row, so the grid can never hide their
          own entries. */}
      {!reordering && commands.length > 0 && (
        <div className="space-y-2">
          <div className="px-1 text-xs font-semibold uppercase tracking-wider text-wc-text-faint">
            {t(strings.terminalLauncher.shortcuts)}
          </div>
          {commands.map((card) => (
            <button
              key={card.key}
              type="button"
              data-testid={`launcher-shortcut-${slugify(card.label)}`}
              onClick={() => { onLaunch(card.command); }}
              disabled={disabled}
              title={card.command}
              className="flex min-h-11 w-full items-center gap-2.5 rounded-xl border border-wc-default bg-wc-surface-input px-3 py-2 text-start transition hover:border-wc-accent disabled:cursor-not-allowed disabled:opacity-50"
            >
              <AgentMark mark="command" />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-[13.5px] font-medium text-wc-text-primary">{card.label}</span>
                <span className="block truncate font-mono text-[10.5px] text-wc-text-faint">{card.description || card.command}</span>
              </span>
            </button>
          ))}
        </div>
      )}

      {/* Kept mounted so a screen reader announces an install finishing even
          when the transient confirmation has already gone. */}
      <span aria-live="polite" className="sr-only">
        {installing.length > 0 ? t(strings.launcher.agentInstalling) : ""}
      </span>
    </section>
  );
}
