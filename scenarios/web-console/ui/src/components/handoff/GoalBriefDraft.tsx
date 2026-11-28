import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import { buildGoalBrief, validateGoalBrief } from "@vrooli/goal-brief";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";

export function GoalBriefDraft({ onUse }: { onUse: (message: string) => void }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [destination, setDestination] = useState("");
  const [stance, setStance] = useState("");
  const [charterPath, setCharterPath] = useState("");
  const draft = useMemo(() => buildGoalBrief({ destination, stance, charterPath }), [destination, stance, charterPath]);
  const errors = useMemo(() => validateGoalBrief(draft), [draft]);

  return (
    <section className="space-y-2 rounded-lg border border-wc-default bg-wc-surface-base/40 p-3" data-testid="goal-brief-draft">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="m-0 text-sm font-semibold text-wc-text-primary">{t(strings.handoff.goalBriefTitle)}</p>
          <p className="text-xs text-wc-text-muted">{t(strings.handoff.goalBriefHint)}</p>
        </div>
        <button type="button" className="min-h-10 shrink-0 rounded border border-wc-default px-3 text-sm text-wc-text-secondary" aria-expanded={open} data-testid="goal-brief-toggle" onClick={() => { setOpen((value) => !value); }}>
          {open ? t(strings.handoff.goalBriefHide) : t(strings.handoff.goalBriefOpen)}
        </button>
      </div>
      {open ? (
        <div className="space-y-3">
          <label className="block space-y-1 text-xs font-medium text-wc-text-secondary">
            <span>{t(strings.handoff.goalBriefDestination)}</span>
            <Input data-testid="goal-brief-destination" value={destination} onChange={(event) => { setDestination(event.target.value); }} />
          </label>
          <label className="block space-y-1 text-xs font-medium text-wc-text-secondary">
            <span>{t(strings.handoff.goalBriefStance)}</span>
            <Textarea data-testid="goal-brief-stance" rows={3} value={stance} onChange={(event) => { setStance(event.target.value); }} />
          </label>
          <label className="block space-y-1 text-xs font-medium text-wc-text-secondary">
            <span>{t(strings.handoff.goalBriefCharter)}</span>
            <Input data-testid="goal-brief-charter" value={charterPath} onChange={(event) => { setCharterPath(event.target.value); }} />
          </label>
          {errors.length > 0 ? (
            <ul className="m-0 list-disc space-y-1 ps-5 text-xs text-amber-200" data-testid="goal-brief-errors" aria-live="polite">
              {errors.map((error) => <li key={error}>{error}</li>)}
            </ul>
          ) : (
            <div className="space-y-2">
              <div className="rounded border border-wc-default bg-wc-surface p-2">
                <div className="flex justify-between gap-2 text-[11px] text-wc-text-muted"><span>{t(strings.handoff.goalBriefKernel)}</span><span data-testid="goal-brief-kernel-count">{Array.from(draft.kernel).length}/600</span></div>
                <pre className="m-0 whitespace-pre-wrap break-words pt-1 font-mono text-xs text-wc-text-secondary">{draft.kernel}</pre>
              </div>
              <details>
                <summary className="cursor-pointer text-xs text-wc-text-muted">{t(strings.handoff.goalBriefStanceMessage)}</summary>
                <pre className="mt-2 whitespace-pre-wrap break-words rounded border border-wc-default bg-wc-surface p-2 text-xs text-wc-text-secondary">{draft.stanceMessage}</pre>
              </details>
              <button type="button" className="min-h-10 rounded bg-wc-accent px-3 text-sm font-semibold text-white" data-testid="goal-brief-use" onClick={() => { onUse(draft.message); }}>
                {t(strings.handoff.goalBriefUse)}
              </button>
              <p className="text-[11px] text-wc-text-faint">{t(strings.handoff.goalBriefReview)}</p>
            </div>
          )}
        </div>
      ) : null}
    </section>
  );
}
