import { useEffect, useMemo, useState } from "react";
import { Loader2, Send, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";

import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import type { HandoffResult } from "../../lib/handoff";
import { targetKey, textForTarget, type HandoffTarget } from "../../hooks/useHandoff";
import { Button } from "../ui/button";

// [REQ:P0-014d] Handoff Between Sessions In A Group
//
// The composer imports the send path and nothing else. It has no knowledge of
// capture rules: a suggestion chip opens this same surface, already populated,
// through the ordinary props below.

export interface HandoffComposerProps {
  open: boolean;
  onClose: () => void;
  /** Where the handoff is coming from, for the title. */
  sourceLabel: string;
  /** The text being carried: a path, a passage, or empty. Never classified. */
  payload: string;
  targets: HandoffTarget[];
  /** Pre-select these target keys — a suggestion chip names one. */
  initialSelection?: string[];
  onSend: (targets: HandoffTarget[], textFor: (target: HandoffTarget) => string) => Promise<HandoffResult[]>;
}

/**
 * Compose and send a handoff.
 *
 * Two rules the surface exists to enforce:
 *
 *   1. Nothing is dispatched without the operator seeing the exact text. Every
 *      selected target's message is rendered and editable before Send.
 *   2. The result is reported per target, and `queued` is shown as its own
 *      state. Collapsing it into success would tell the operator their
 *      message arrived when it is still sitting in a pending queue.
 */
export default function HandoffComposer({
  open,
  onClose,
  sourceLabel,
  payload,
  targets,
  initialSelection,
  onSend,
}: HandoffComposerProps) {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<string[]>(initialSelection ?? []);
  // Per-target overrides. Absent means "use the rendered default", so a target
  // the operator never touched follows its role's prompt even if the payload
  // changes underneath.
  const [edited, setEdited] = useState<Record<string, string>>({});
  const [sending, setSending] = useState(false);
  const [results, setResults] = useState<HandoffResult[] | null>(null);

  useEffect(() => {
    if (!open) return;
    // Default to the only target when there is exactly one: the common case
    // should not cost a selection click.
    const only = targets.length === 1 ? targets[0] : undefined;
    const preset = initialSelection ?? (only ? [targetKey(only)] : []);
    setSelected(preset);
    setEdited({});
    setResults(null);
    setSending(false);
  }, [open, initialSelection, targets]);

  const selectedTargets = useMemo(
    () => targets.filter((target) => selected.includes(targetKey(target))),
    [selected, targets],
  );

  const textFor = (target: HandoffTarget): string => {
    const key = targetKey(target);
    return edited[key] ?? textForTarget(target, payload);
  };

  const toggle = (key: string) => {
    setSelected((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]));
  };

  const handleSend = async () => {
    if (selectedTargets.length === 0 || sending) return;
    setSending(true);
    try {
      const outcome = await onSend(selectedTargets, textFor);
      setResults(outcome);
      // The text is kept on any non-delivery so the operator can retry
      // without retyping. Closing here would throw it away.
      if (outcome.every((r) => r.status === "sent")) onClose();
    } finally {
      setSending(false);
    }
  };

  if (!open) return null;

  return (
    <ResponsiveDialog
      avoidKeyboard
      open
      onClose={onClose}
      size="md"
      closeLabel={t(strings.handoff.close)}
      title={t(strings.handoff.handOffTitle, { name: sourceLabel })}
      testId="handoff-composer"
    >
      <div className="flex h-full flex-col">
        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
          <div className="rounded-lg border border-wc-default bg-wc-surface-base/50 px-3 py-2">
            <div className="text-[11px] font-semibold uppercase tracking-wider text-wc-text-faint">
              {t(strings.handoff.payloadLabel)}
            </div>
            <div className="mt-0.5 break-all text-sm text-wc-text-secondary">
              {payload || t(strings.handoff.noPayload)}
            </div>
          </div>

          {targets.length === 0 ? (
            <p data-testid="handoff-no-targets" className="py-6 text-center text-sm text-wc-text-muted">
              {t(strings.handoff.noTargets)}
            </p>
          ) : (
            <>
              <section className="space-y-2">
                <div className="text-xs font-semibold uppercase tracking-wider text-wc-text-faint">
                  {t(strings.handoff.targets)}
                </div>
                <div data-testid="handoff-targets" className="space-y-1.5">
                  {targets.map((target) => {
                    const key = targetKey(target);
                    const isWaiting = target.kind !== "session";
                    return (
                      <label
                        key={key}
                        data-testid={`handoff-target-${key}`}
                        className={cn(
                          "flex cursor-pointer items-center gap-2.5 rounded-lg border px-3 py-2 transition",
                          selected.includes(key)
                            ? "border-wc-accent bg-wc-accent/10"
                            : "border-wc-default bg-wc-surface-input hover:border-wc-accent/50",
                        )}
                      >
                        <Checkbox
                          label=""
                          checked={selected.includes(key)}
                          onChange={() => { toggle(key); }}
                        />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-sm text-wc-text-primary">{target.label}</span>
                          {isWaiting && (
                            <span className="block text-[11px] text-wc-text-faint">
                              {t(strings.handoff.startsFirst)}
                            </span>
                          )}
                        </span>
                        {isWaiting && (
                          <span className="shrink-0 rounded border border-dashed border-wc-default px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-wc-text-faint">
                            {t(strings.handoff.waitingChip)}
                          </span>
                        )}
                      </label>
                    );
                  })}
                </div>
              </section>

              {/* One field for the common single-target case; one per target
                  only when the selected targets carry different prompts, so
                  the ordinary path does not pay for the fan-out path. */}
              {selectedTargets.length > 0 && (
                <section className="space-y-2">
                  <div className="text-xs font-semibold uppercase tracking-wider text-wc-text-faint">
                    {t(strings.handoff.message)}
                  </div>
                  {selectedTargets.map((target) => {
                    const key = targetKey(target);
                    return (
                      <div key={key} className="space-y-1">
                        {selectedTargets.length > 1 && (
                          <div className="text-[11px] text-wc-text-faint">
                            {t(strings.handoff.messageFor, { label: target.label })}
                          </div>
                        )}
                        <textarea
                          data-testid={selectedTargets.length === 1 ? "handoff-message" : `handoff-message-${key}`}
                          value={textFor(target)}
                          onChange={(event) => {
                            setEdited((prev) => ({ ...prev, [key]: event.target.value }));
                          }}
                          rows={selectedTargets.length > 1 ? 2 : 4}
                          className="w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 text-sm text-wc-text-primary outline-none transition focus:border-wc-accent"
                        />
                      </div>
                    );
                  })}
                </section>
              )}
            </>
          )}

          {results && (
            <section data-testid="handoff-results" className="space-y-1.5 rounded-lg border border-wc-default bg-wc-surface-base/50 p-3">
              {results.map((result) => (
                <div
                  key={result.targetId}
                  data-testid={`handoff-result-${result.targetId}`}
                  data-status={result.status}
                  className={cn(
                    "text-xs",
                    result.status === "sent" && "text-emerald-300",
                    result.status === "queued" && "text-amber-300",
                    result.status === "failed" && "text-rose-300",
                  )}
                >
                  {result.status === "sent" && t(strings.handoff.resultSent, { label: result.label })}
                  {result.status === "queued" && t(strings.handoff.resultQueued, { label: result.label })}
                  {result.status === "failed" && t(strings.handoff.resultFailed, { label: result.label, reason: result.reason ?? "" })}
                </div>
              ))}
              {results.some((r) => r.status !== "sent") && (
                <p className="pt-1 text-[11px] text-wc-text-faint">{t(strings.handoff.keptText)}</p>
              )}
            </section>
          )}
        </div>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t border-wc-default p-3">
          <Button variant="outline" size="sm" onClick={onClose}>
            <X className="me-1.5 h-4 w-4" aria-hidden />
            {t(strings.handoff.close)}
          </Button>
          <Button
            data-testid="handoff-send"
            size="sm"
            onClick={() => { void handleSend(); }}
            disabled={selectedTargets.length === 0 || sending}
          >
            {sending
              ? <Loader2 className="me-1.5 h-4 w-4 animate-spin" aria-hidden />
              : <Send className="me-1.5 h-4 w-4" aria-hidden />}
            {sending ? t(strings.handoff.sending) : t(strings.handoff.send)}
          </Button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}
