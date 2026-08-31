import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { KeyboardEvent as ReactKeyboardEvent } from "react";
import { Check, ChevronDown, CircleAlert, Monitor, Plus, RefreshCw, Server, SlidersHorizontal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useEscapeKey } from "@vrooli/react-component-library/useEscapeKey/1";

import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { slugify } from "../../lib/slugify";
import type { TerminalTarget, TerminalTargetState } from "../../api/targets";

// [REQ:P0-014b] Compact Machine Selection

/**
 * Machine selection in one row.
 *
 * The interaction model is ported from system-monitor's MachinePicker, whose
 * two accessibility decisions are worth keeping and are easy to lose:
 *
 *   1. Closing returns focus to the trigger. Otherwise a keyboard reader who
 *      presses Escape is left with focus on nothing.
 *   2. The link action sits OUTSIDE the role="listbox" element. A button
 *      inside the listbox breaks the relationship assistive tech relies on,
 *      and the list scrolls while the action must not — a fleet large enough
 *      to scroll is exactly when "Link a machine…" must stay reachable.
 *
 * The markup is rebuilt against web-console's own tokens rather than imported:
 * scenarios do not share components across their boundaries.
 */

interface MachinePickerProps {
  targets: TerminalTarget[];
  selectedId: string;
  onSelect: (targetId: string) => void;
  /** Opens the machines surface, which owns linking and permissions. */
  onOpenMachines?: () => void;
  /**
   * Why the fleet looks the way it does — unconfigured, empty, or unreadable.
   *
   * This used to be a full-width amber card in the launcher, which spent a
   * quarter of the dialog explaining a list the operator had not opened yet.
   * It belongs where the list is: a footnote under the options, visible at the
   * moment it is relevant and costing nothing the rest of the time.
   */
  catalogMessage?: string | null;
  /** A remedy for the catalog state, when the server offered one. */
  catalogRecovery?: string | null;
  /** Re-probe the fleet. Rendered inside the menu, not beside the trigger. */
  onRefresh?: () => void | Promise<void>;
  refreshing?: boolean;
  disabled?: boolean;
}

/** Presence tone drives the LED colour and reads before any text does. */
function toneFor(target: TerminalTarget): "local" | "ready" | "warn" | "off" {
  if (target.kind === "local") return "local";
  const state: TerminalTargetState | undefined = target.state;
  if (state === "dispatchable") return "ready";
  if (state === "offline") return "off";
  return "warn";
}

const ledClass: Record<ReturnType<typeof toneFor>, string> = {
  local: "bg-wc-accent",
  ready: "bg-emerald-400",
  warn: "bg-amber-400",
  off: "bg-slate-500",
};

/**
 * The one-line summary under each machine's name. It carries the readiness
 * detail that used to need its own grid, which is what lets the full-width
 * target cards go.
 */
function metaFor(target: TerminalTarget, neverSeen: string, agentsReady: (ready: number, total: number) => string): string {
  const platform = [target.os, target.arch].filter(Boolean).join("/");
  const capabilityFacts = target.readiness?.filter((fact) => fact.key.startsWith("capability:")) ?? [];
  const capabilitySummary = capabilityFacts.length
    ? agentsReady(capabilityFacts.filter((fact) => fact.state === "ready").length, capabilityFacts.length)
    : undefined;
  const transportFailure = target.readiness?.find((fact) => !fact.passed && !fact.key.startsWith("capability:"));
  const capabilityFailure = capabilityFacts.find((fact) => fact.state !== "ready");
  if (target.kind === "local" && !capabilityFacts.length) return "Web Console host";
  if (transportFailure || capabilityFailure) return [platform, capabilitySummary, transportFailure?.label, capabilityFailure?.label].filter(Boolean).join(" · ");
  if (!target.available) {
    return [platform, capabilitySummary, target.recovery_action ?? target.failure_rung ?? neverSeen].filter(Boolean).join(" · ");
  }
  return [platform, capabilitySummary].filter(Boolean).join(" · ") || target.node_id || target.id;
}

/** The grant chip: what the caller is actually allowed to do here. */
function grantFor(target: TerminalTarget): string | undefined {
  if (target.kind === "local") return undefined;
  const scope = target.readiness?.find((fact) => fact.key === "bridge_scope")?.detail;
  if (scope) return scope;
  return target.available ? undefined : "No grant";
}

export default function MachinePicker({
  targets,
  selectedId,
  onSelect,
  onOpenMachines,
  catalogMessage,
  catalogRecovery,
  onRefresh,
  refreshing = false,
  disabled = false,
}: MachinePickerProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const rootRef = useRef<HTMLDivElement | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  const close = useCallback(() => {
    setOpen(false);
    triggerRef.current?.focus();
  }, []);

  useEscapeKey(open, close);

  useEffect(() => {
    if (!open) return undefined;
    const onPointerDown = (event: PointerEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    return () => { document.removeEventListener("pointerdown", onPointerDown); };
  }, [open]);

  const ordered = useMemo(() => {
    const local = targets.filter((target) => target.kind === "local");
    const remote = targets.filter((target) => target.kind !== "local");
    return [...local, ...remote];
  }, [targets]);

  const selected = ordered.find((target) => target.id === selectedId) ?? ordered[0];

  useEffect(() => {
    if (!open) return;
    listRef.current?.querySelector<HTMLElement>('[data-active="true"]')?.focus();
  }, [open, activeIndex]);

  const neverSeen = t(strings.terminalLauncher.neverSeen);
  const agentsReady = (ready: number, total: number) => t(strings.terminalLauncher.agentsReady, { ready, total });

  const handleListKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (ordered.length === 0) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((prev) => (prev + 1) % ordered.length);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((prev) => (prev - 1 + ordered.length) % ordered.length);
    } else if (event.key === "Home") {
      event.preventDefault();
      setActiveIndex(0);
    } else if (event.key === "End") {
      event.preventDefault();
      setActiveIndex(ordered.length - 1);
    }
  };

  const choose = (targetId: string) => {
	const target = ordered.find((candidate) => candidate.id === targetId);
	if (!target?.available) return;
    onSelect(targetId);
    close();
  };

  if (!selected) return null;

  return (
    <div ref={rootRef} className="relative min-w-0 flex-1 basis-[13rem]">
      <button
        ref={triggerRef}
        type="button"
        data-testid="launcher-machine-picker"
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={`${t(strings.launcher.machine)}: ${selected.label}`}
        onClick={() => {
          // Open onto the current selection, set before the menu mounts, so
          // the first arrow keypress moves from where the reader already is.
          const index = ordered.findIndex((target) => target.id === selectedId);
          setActiveIndex(index >= 0 ? index : 0);
          setOpen((prev) => !prev);
        }}
        className="flex min-h-11 w-full items-center gap-2 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-start text-sm text-wc-text-primary transition hover:border-wc-accent disabled:cursor-not-allowed disabled:opacity-50"
      >
        <span className={cn("h-2 w-2 shrink-0 rounded-full", ledClass[toneFor(selected)])} aria-hidden />
        <span className="min-w-0 flex-1 truncate">{selected.label}</span>
        <span className="shrink-0 truncate text-xs text-wc-text-faint">{metaFor(selected, neverSeen, agentsReady)}</span>
        <ChevronDown className={cn("h-4 w-4 shrink-0 text-wc-text-faint transition", open && "rotate-180")} aria-hidden />
      </button>

      {open && (
        <div
          data-testid="launcher-machine-menu"
          className="absolute inset-x-0 top-full z-30 mt-1 overflow-hidden rounded-lg border border-wc-default bg-wc-surface-raised shadow-xl"
        >
          <div className="flex items-center gap-2 px-3 py-1.5">
            <span className="min-w-0 flex-1 truncate text-[11px] font-semibold uppercase tracking-wider text-wc-text-faint">
              {t(strings.launcher.chooseMachine)}
            </span>
            {onRefresh && (
              <button
                type="button"
                data-testid="launcher-target-refresh"
                aria-label={t(strings.terminalLauncher.refreshTargets)}
                title={t(strings.terminalLauncher.refreshTargets)}
                disabled={refreshing}
                onClick={() => { void onRefresh(); }}
                className="shrink-0 rounded-full p-1.5 text-wc-text-faint transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-60"
              >
                <RefreshCw className={cn("h-3.5 w-3.5", refreshing && "animate-spin")} aria-hidden />
              </button>
            )}
          </div>

          {/* The listbox is exactly the options and nothing else: a wrapper
              between it and its options, or the link action inside it, would
              break the relationship assistive tech relies on. The list
              scrolls; the link action below does not. */}
          <div
            ref={listRef}
            role="listbox"
            data-testid="launcher-machine-list"
            aria-label={t(strings.launcher.machine)}
            onKeyDown={handleListKeyDown}
            className="max-h-64 overflow-y-auto"
          >
            {ordered.map((target, index) => {
              const isSelected = target.id === selected.id;
              const grant = grantFor(target);
              return (
                <button
                  key={target.id}
                  type="button"
                  role="option"
                  aria-selected={isSelected}
                  aria-disabled={!target.available}
                  title={!target.available ? (target.failure_rung ?? target.recovery_action ?? "This machine cannot host a terminal session") : undefined}
                  disabled={!target.available}
                  tabIndex={index === activeIndex ? 0 : -1}
                  data-active={index === activeIndex}
                  data-testid={`launcher-machine-option-${slugify(target.id)}`}
                  onClick={() => { choose(target.id); }}
                  onFocus={() => { setActiveIndex(index); }}
                  className={cn(
                    "flex w-full items-center gap-2.5 px-3 py-2 text-start transition",
                    isSelected ? "bg-wc-accent/10" : "hover:bg-wc-surface-input",
                    !target.available && "cursor-not-allowed opacity-60",
                  )}
                >
                  <span className={cn("h-2 w-2 shrink-0 rounded-full", ledClass[toneFor(target)])} aria-hidden />
                  {target.kind === "local"
                    ? <Monitor className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />
                    : <Server className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />}
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm text-wc-text-primary">{target.label}</span>
                    <span className="block truncate text-[11px] text-wc-text-faint">{metaFor(target, neverSeen, agentsReady)}</span>
                  </span>
                  {isSelected
                    ? <Check className="h-4 w-4 shrink-0 text-wc-accent" aria-hidden />
                    : grant
                      ? <span className="shrink-0 rounded-full border border-wc-default px-1.5 py-0.5 text-[10px] text-wc-text-faint">{grant}</span>
                      : null}
                </button>
              );
            })}
          </div>

          {/* Why the list reads the way it does, stated under the list rather
              than in a card above it. */}
          {catalogMessage && (
            <div
              data-testid="launcher-target-catalog-state"
              className="flex gap-2 border-t border-wc-default px-3 py-2 text-[11px] text-wc-text-muted"
            >
              <CircleAlert className="mt-0.5 h-3.5 w-3.5 shrink-0 text-amber-400" aria-hidden />
              <span className="min-w-0">
                {catalogMessage}
                {catalogRecovery && <span className="block text-wc-text-faint">{catalogRecovery}</span>}
              </span>
            </div>
          )}

          {/* Linking and administering are two actions on one row: both are
              "go somewhere else", and neither deserves a card in the dialog. */}
          {onOpenMachines && (
            <div className="flex items-center gap-1 border-t border-wc-default p-1">
              <button
                type="button"
                data-testid="launcher-machine-link"
                onClick={() => {
                  close();
                  onOpenMachines();
                }}
                className="flex min-h-11 flex-1 items-center gap-2 rounded-lg px-2 text-start text-sm text-wc-accent transition hover:bg-wc-surface-input"
              >
                <Plus className="h-4 w-4 shrink-0" aria-hidden />
                {t(strings.launcher.linkMachine)}
              </button>
              <button
                type="button"
                data-testid="launcher-machine-manage"
                onClick={() => {
                  close();
                  onOpenMachines();
                }}
                className="flex min-h-11 shrink-0 items-center gap-1.5 rounded-lg px-2 text-sm text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary"
              >
                <SlidersHorizontal className="h-4 w-4 shrink-0" aria-hidden />
                {t(strings.machines.openFromLauncher)}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
