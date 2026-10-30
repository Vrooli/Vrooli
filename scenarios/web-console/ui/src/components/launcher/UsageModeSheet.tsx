/**
 * Start or end a usage mode. It is opened from the launcher header chip. It
 * reads and writes only Web Console's API, so it works without Agent Manager;
 * "until reset" is offered only for a quota window whose reset is known.
 */
import { useMemo, useState } from "react";
import { BottomSheet } from "@vrooli/react-component-library/BottomSheet/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import type { UsageModeStart } from "../../api/usageMode";
import { useAgentUsage } from "./useAgentUsage";
import type { UsageModeController } from "./useUsageMode";

const HOUR_CHOICES = [1, 2, 4];
const OPTION = "min-h-11 rounded-lg border px-3 text-sm font-medium text-start transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent";

export function clockTime(value: string, locale?: string): string {
  const at = Date.parse(value);
  if (!Number.isFinite(at)) return "";
  const date = new Date(at);
  // A reset on another day needs the day, or "08:47 PM" reads as tonight.
  const sameDay = date.toDateString() === new Date().toDateString();
  return new Intl.DateTimeFormat(locale, sameDay ? { hour: "2-digit", minute: "2-digit" } : { weekday: "short", hour: "2-digit", minute: "2-digit" }).format(date);
}

export default function UsageModeSheet({ open, onClose, usageMode }: { open: boolean; onClose: () => void; usageMode: UsageModeController }) {
  const { t, i18n } = useTranslation();
  const { state, busy, error } = usageMode;
  const agents = state.agents;
  const [agentId, setAgentId] = useState("");
  const [choice, setChoice] = useState("minutes:120");
  const selected = agents.find((agent) => agent.agentId === agentId) ?? agents[0];
  const readings = useAgentUsage(open && state.canControl && Boolean(selected), "local", selected?.agentId ?? "");
  const usage = selected ? readings.usages[selected.agentId] : undefined;
  // Only a single, current, likely-matched window with a future reset is offered.
  const resets = useMemo(() => {
    if (!usage || usage.match !== "likely") return [];
    return usage.windows.filter((window) => window.resetsAt && Date.parse(window.resetsAt) > Date.now() &&
      usage.windows.filter((other) => other.windowMinutes === window.windowMinutes).length === 1);
  }, [usage]);
  const active = state.modes;
  const untilText = (mode: { untilKind: string; expiresAt?: string }) =>
    mode.untilKind === "off" || !mode.expiresAt ? t(strings.launcher.usageModeUntilOffShort) : t(strings.launcher.usageModeUntil, { time: clockTime(mode.expiresAt, i18n.resolvedLanguage) });

  const choices = [
    ...HOUR_CHOICES.map((hours) => ({ key: `minutes:${String(hours * 60)}`, label: t(strings.launcher.usageModeHours, { count: hours }) })),
    ...resets.map((window) => {
      const time = clockTime(window.resetsAt, i18n.resolvedLanguage);
      return { key: `reset:${String(window.windowMinutes)}`, label: window.windowMinutes === 10080 ? t(strings.launcher.usageModeUntilWeekly, { time }) : t(strings.launcher.usageModeUntilWindow, { hours: window.windowMinutes / 60, time }) };
    }),
    { key: "off", label: t(strings.launcher.usageModeUntilOff) },
  ];
  const effectiveChoice = choices.some((item) => item.key === choice) ? choice : "minutes:120";

  const request = (): UsageModeStart | null => {
    if (!selected) return null;
    const [kind, value] = effectiveChoice.split(":");
    if (kind === "off") return { agentId: selected.agentId, kind: "off" };
    if (kind === "reset") return { agentId: selected.agentId, kind: "reset", windowMinutes: Number(value) };
    return { agentId: selected.agentId, kind: "minutes", minutes: Number(value) };
  };

  return (
    <BottomSheet avoidKeyboard open={open} onClose={onClose} title={t(strings.launcher.usageModeSheetTitle)}
      closeLabel={t(strings.launcher.usageModeCloseLabel)} testId="usage-mode-sheet">
      <div className="space-y-4 p-4">
        <p className="text-sm text-wc-text-muted">{t(strings.launcher.usageModeExplain)}</p>
        {!state.canControl && <p data-testid="usage-mode-operator-only" className="text-sm text-wc-text-secondary">{t(strings.launcher.usageModeOperatorOnly)}</p>}
        {active.length > 0 && <section aria-label={t(strings.launcher.usageModeActive)} className="space-y-2">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted">{t(strings.launcher.usageModeActive)}</h3>
          {active.map((mode) => (
            <div key={mode.id} data-testid={`usage-mode-active-${mode.id}`} className="flex items-center gap-3 rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2">
              <span className="min-w-0 flex-1 text-sm text-wc-text-primary">{agents.find((agent) => agent.runner === mode.runner)?.label ?? mode.runner}<span className="block text-xs text-wc-text-muted">{untilText(mode)}</span></span>
              {state.canControl && <Button type="button" variant="secondary" size="md" data-testid="usage-mode-end" disabled={busy} className="min-h-11" onClick={() => { void usageMode.end(mode.id); }}>{t(strings.launcher.usageModeEnd)}</Button>}
            </div>))}
        </section>}
        {state.canControl && selected && <section className="space-y-3">
          <div role="radiogroup" aria-label={t(strings.launcher.usageModeFor)} className="flex flex-wrap gap-2">
            {agents.map((agent) => (
              <button key={agent.agentId} type="button" role="radio" aria-checked={agent.agentId === selected.agentId} data-testid={`usage-mode-agent-${agent.agentId}`}
                onClick={() => { setAgentId(agent.agentId); }}
                className={cn(OPTION, agent.agentId === selected.agentId ? "border-wc-accent bg-wc-accent/10 text-wc-text-primary" : "border-wc-default text-wc-text-secondary")}>{agent.label}</button>))}
          </div>
          <div role="radiogroup" aria-label={t(strings.launcher.usageModeDuration)} className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {choices.map((item) => (
              <button key={item.key} type="button" role="radio" aria-checked={item.key === effectiveChoice} data-testid={`usage-mode-choice-${item.key.replace(":", "-")}`}
                onClick={() => { setChoice(item.key); }}
                className={cn(OPTION, item.key === effectiveChoice ? "border-wc-accent bg-wc-accent/10 text-wc-text-primary" : "border-wc-default text-wc-text-secondary")}>{item.label}</button>))}
          </div>
          <p className="text-xs text-wc-text-muted">{selected.label}: {selected.usualModel} → {selected.lowerModel}</p>
          {resets.length === 0 && <p className="text-xs text-wc-text-muted">{t(strings.launcher.usageModeResetUnknown)}</p>}
          <Button type="button" variant="primary" size="md" data-testid="usage-mode-start" disabled={busy} className="min-h-11 w-full" onClick={() => {
            const body = request();
            if (body) void usageMode.start(body).then((ok) => { if (ok) onClose(); });
          }}>{t(strings.launcher.usageModeStart)}</Button>
        </section>}
        {error && <p role="alert" data-testid="usage-mode-error" className="text-sm text-wc-error-text">{t(strings.launcher.usageModeError)}</p>}
      </div>
    </BottomSheet>
  );
}
