import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Terminal } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import type { ActivitySource } from "../../api/sessionActivity";
import type { StateSlot } from "./resolveStateSlot";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-state-slot

const SPEAKER_BY_HARNESS: Readonly<Record<string, string>> = {
  claude: strings.messagesPane.speaker.claude,
  codex: strings.messagesPane.speaker.codex,
  grok: strings.messagesPane.speaker.grok,
  opencode: strings.messagesPane.speaker.opencode,
};

const SOURCE_LABEL: Readonly<Record<ActivitySource, string>> = {
  screen: strings.messagesPane.stateSlot.source.screen,
  output_clock: strings.messagesPane.stateSlot.source.output_clock,
  hook: strings.messagesPane.stateSlot.source.hook,
  harness_event: strings.messagesPane.stateSlot.source.harness_event,
};

/** A translator narrowed to string keys and results, for computed keys. */
type Say = (key: string, options?: Record<string, unknown>) => string;

function secondsSince(iso: string | undefined, now: number): number | null {
  if (!iso) return null;
  const at = Date.parse(iso);
  return Number.isFinite(at) ? Math.max(0, Math.floor((now - at) / 1000)) : null;
}

function formatDuration(totalSeconds: number, say: Say): string {
  const labels = strings.messagesPane.stateSlot;
  if (totalSeconds < 60) return say(labels.seconds, { s: totalSeconds });
  if (totalSeconds < 3600) return say(labels.minutes, { m: Math.floor(totalSeconds / 60), s: totalSeconds % 60 });
  return say(labels.hours, { h: Math.floor(totalSeconds / 3600), m: Math.floor((totalSeconds % 3600) / 60) });
}

/**
 * The current time, re-read once a second while `enabled`. This re-renders
 * elapsed-time labels; it never reads or polls session state.
 */
function useSecondTick(enabled: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!enabled) return undefined;
    setNow(Date.now());
    const id = setInterval(() => { setNow(Date.now()); }, 1000);
    return () => { clearInterval(id); };
  }, [enabled]);
  return now;
}

interface SessionStateSlotProps {
  slot: StateSlot | null;
  /** Switches the pane to its terminal. Absent when there is no terminal to go to. */
  onOpenTerminal?: () => void;
  /** Answers the answerable prompt with an option key, or dismisses it with null; rejects with the reason it could not. */
  onAnswer?: (optionKey: string | null) => Promise<void>;
}

/**
 * The one presentation of the session's activity under the last message: a
 * working strip, or a card saying the agent is waiting on the user — detected
 * (level 1), with the prompt read (level 2), or answerable (level 3: its
 * option buttons answer through onAnswer, and Cancel when the prompt allows it).
 */
export function SessionStateSlot({ slot, onOpenTerminal, onAnswer }: SessionStateSlotProps) {
  const { t } = useTranslation();
  // Computed keys (speaker, source, duration units) all name string entries.
  const say = t as unknown as Say;
  const now = useSecondTick(slot !== null);
  // The prompt (by hash) an answer is in flight or done for, and why the last
  // one failed; a new prompt has a new hash, which clears both.
  const [answered, setAnswered] = useState<{ hash: string; error: string | null } | null>(null);
  if (!slot) return null;
  const labels = strings.messagesPane.stateSlot;
  const frame = "mx-3 mb-3 mt-1 [overflow-anchor:none]";

  if (slot.kind === "working") {
    const elapsed = secondsSince(slot.elapsedFrom, now) ?? 0;
    const lastOutput = secondsSince(slot.lastOutputAt, now);
    return (
      <div
        data-testid="messages-state-slot"
        data-kind="working"
        role="status"
        aria-label={t(labels.label)}
        className={cn(frame, "flex flex-wrap items-center gap-x-2 gap-y-1 rounded-md px-3 py-2 text-xs text-wc-text-secondary")}
      >
        <span aria-hidden className="h-2 w-2 shrink-0 animate-pulse rounded-full bg-wc-accent" />
        <span className="font-medium text-wc-text-primary">{t(labels.working, { elapsed: formatDuration(elapsed, say) })}</span>
        {lastOutput !== null && <span>· {t(labels.lastOutput, { ago: formatDuration(lastOutput, say) })}</span>}
      </div>
    );
  }

  const promptHash = "prompt" in slot ? slot.prompt.hash ?? "" : "";
  const settled = answered?.hash === promptHash ? answered : null;
  const busy = settled !== null && settled.error === null;
  const answerError = settled?.error ?? null;
  const answer = (optionKey: string | null) => {
    if (!onAnswer) return;
    setAnswered({ hash: promptHash, error: null });
    onAnswer(optionKey).catch((error: unknown) => {
      setAnswered({ hash: promptHash, error: error instanceof Error ? error.message : String(error) });
    });
  };

  const speaker = t((SPEAKER_BY_HARNESS[slot.harness] ?? strings.messagesPane.speaker.agent) as never);
  const detected = t(labels.detected, {
    ago: formatDuration(secondsSince(slot.since, now) ?? 0, say),
    source: t(SOURCE_LABEL[slot.source] as never),
  });
  const terminalLabel = slot.kind === "waiting-rendered" ? labels.answerInTerminal : labels.openTerminal;

  return (
    <section
      data-testid="messages-state-slot"
      data-kind={slot.kind}
      aria-label={t(labels.label)}
      className={cn(frame, "rounded-lg border border-wc-accent bg-wc-surface-raised p-3 shadow-sm")}
    >
      {slot.kind === "waiting-detected" ? (
        <p className="text-sm font-medium text-wc-text-primary">{t(labels.asking, { speaker })}</p>
      ) : (
        <>
          <p className="text-xs font-medium text-wc-text-secondary">{t(labels.asking, { speaker })}</p>
          <p className="mt-1 whitespace-pre-wrap break-words text-sm text-wc-text-primary">{slot.prompt.text}</p>
          {slot.prompt.options.length > 0 && (
            <div role="group" aria-label={t(labels.options)} className="mt-2 flex flex-wrap gap-1.5">
              {slot.prompt.options.map((option) => (slot.kind === "waiting-answerable" ? (
                <Button
                  key={option.key}
                  type="button"
                  size="sm"
                  variant={option.selected ? "secondary" : "ghost"}
                  disabled={busy || !onAnswer}
                  onClick={() => { answer(option.key); }}
                  data-testid={`state-slot-option-${option.key}`}
                  className="min-h-11 md:min-h-8"
                >
                  {option.label}
                  <span className="ms-1.5 text-xs opacity-70">{t(labels.sendOption, { key: option.key })}</span>
                </Button>
              ) : (
                <span
                  key={option.key}
                  data-testid={`state-slot-option-${option.key}`}
                  data-selected={option.selected}
                  aria-current={option.selected ? "true" : undefined}
                  className={cn(
                    "rounded-full border px-2.5 py-1 text-xs",
                    option.selected ? "border-wc-accent text-wc-text-primary" : "border-wc-default text-wc-text-secondary",
                  )}
                >
                  {option.label}
                </span>
              )))}
            </div>
          )}
          {slot.kind === "waiting-answerable" && slot.prompt.cancellable && (
            <Button
              type="button"
              size="sm"
              variant="ghost"
              disabled={busy || !onAnswer}
              onClick={() => { answer(null); }}
              data-testid="state-slot-cancel"
              className="mt-1.5 min-h-11 md:min-h-8"
            >
              {t(labels.cancel)}
            </Button>
          )}
          {answerError && (
            <p data-testid="state-slot-answer-error" role="alert" className="mt-1.5 text-xs text-red-400">
              {t(labels.answerFailed, { reason: answerError })}
            </p>
          )}
          {slot.prompt.freeTextHint && (
            <p data-testid="state-slot-free-text" className="mt-1.5 text-xs text-wc-text-secondary">
              {t(labels.freeText, { hint: slot.prompt.freeTextHint })}
            </p>
          )}
        </>
      )}
      <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
        <span className="text-xs text-wc-text-secondary">{detected}</span>
        {onOpenTerminal && (
          <Button
            type="button"
            size="sm"
            variant="secondary"
            onClick={onOpenTerminal}
            data-testid={slot.kind === "waiting-rendered" ? "state-slot-answer-in-terminal" : "state-slot-open-terminal"}
            className="min-h-11 md:min-h-8"
          >
            <Terminal aria-hidden className="me-1.5 h-3.5 w-3.5" />
            {t(terminalLabel)}
          </Button>
        )}
      </div>
    </section>
  );
}
