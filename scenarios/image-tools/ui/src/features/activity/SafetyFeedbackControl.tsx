import { useEffect, useState } from "react";

import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import type { Job } from "../../api/jobs";
import {
  feedbackChoice,
  type FeedbackChoice,
  type SafetyFeedbackCase,
} from "../../api/safetyFeedback";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { cn } from "../../lib/utils";
import { liveSafetyFeedbackClient, type SafetyFeedbackClient } from "../safety/safetyFeedbackClient";
import { safetyVerdictKind, safetyVerdictOf } from "../safety/safetyVerdict";

export interface SafetyFeedbackControlProps {
  /** The reviewed job. The control renders nothing when it only passed. */
  job: Job;
  client?: SafetyFeedbackClient;
  className?: string;
}

const CHOICES: FeedbackChoice[] = ["legitimate", "confirmed", "uncertain"];

const CHOICE_LABEL = {
  legitimate: strings.reviewerFeedback.legitimate,
  confirmed: strings.reviewerFeedback.confirmed,
  uncertain: strings.reviewerFeedback.uncertain,
} as const;

const CHOICE_SAVED = {
  legitimate: strings.reviewerFeedback.savedLegitimate,
  confirmed: strings.reviewerFeedback.savedConfirmed,
  uncertain: strings.reviewerFeedback.savedUncertain,
} as const;

/**
 * The false-positive capture control for one reviewed job: three one-tap
 * judgements (legitimate / correct / unsure) and an optional note. It labels the
 * reviewer's evidence for calibration; it never restores a refused output or
 * unblurs a sensitive result. Job-level by design; `image-tools safety feedback
 * <job> --ref <ref>` labels one variation.
 */
export function SafetyFeedbackControl({ job, client = liveSafetyFeedbackClient, className }: SafetyFeedbackControlProps) {
  const { t } = useTranslation();
  const verdict = safetyVerdictOf(job);
  const [existing, setExisting] = useState<SafetyFeedbackCase | null>(null);
  const [note, setNote] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let live = true;
    setExisting(null);
    setNote("");
    setError(null);
    client
      .load(job.id)
      .then((stored) => {
        if (live && stored) {
          setExisting(stored);
          setNote(stored.note);
        }
      })
      .catch(() => {
        // An unreadable judgement is not worth blocking on: the control still
        // works and saving replaces it.
      });
    return () => {
      live = false;
    };
  }, [client, job.id]);

  if (!verdict) return null;

  const choice = existing ? feedbackChoice(existing.label) : null;
  const kind = safetyVerdictKind(verdict);
  const noteChanged = existing !== null && note.trim() !== existing.note;
  const noteSaveChoice = choice ?? "legitimate";

  const save = async (next: FeedbackChoice, text: string) => {
    setPending(true);
    setError(null);
    try {
      const stored = await client.record(job.id, next, text);
      if (stored) {
        setExisting(stored);
        setNote(stored.note);
      } else {
        setExisting(null);
      }
    } catch (err) {
      setError(errorMessage(err, t));
    } finally {
      setPending(false);
    }
  };

  const remove = async () => {
    setPending(true);
    setError(null);
    try {
      await client.clear(job.id);
      setExisting(null);
      setNote("");
    } catch (err) {
      setError(errorMessage(err, t));
    } finally {
      setPending(false);
    }
  };

  return (
    <div
      data-testid={selectors.activity.safetyFeedback.root}
      className={cn("mt-2 flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-2", className)}
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
          {t(strings.reviewerFeedback.title)}
        </span>
        <span className="rounded-pill bg-app-danger/10 px-2 py-0.5 text-[11px] font-medium text-app-danger">
          {t(strings.reviewerFeedback[kind])}
        </span>
        <span className="min-w-0 flex-1 truncate text-xs text-app-muted-foreground">
          {t(strings.reviewerFeedback.reason, { reason: verdict.reasonCode || "—" })}
        </span>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs text-app-muted-foreground">{t(strings.reviewerFeedback.prompt)}</span>
        {CHOICES.map((value) => (
          <Button
            key={value}
            type="button"
            size="sm"
            variant={value === "legitimate" ? "primary" : "ghost"}
            shape="square"
            data-testid={selectors.activity.safetyFeedback[value]}
            aria-pressed={choice === value}
            disabled={pending}
            onClick={() => void save(value, note)}
            className={cn(
              "border border-app-border",
              value === "legitimate"
                ? "bg-app-success/10 text-app-success hover:bg-app-success/20"
                : "text-app-foreground hover:bg-app-surface",
              choice === value && "ring-2 ring-app-primary/40",
            )}
          >
            {t(CHOICE_LABEL[value])}
          </Button>
        ))}
      </div>

      {choice ? (
        <span
          data-testid={selectors.activity.safetyFeedback.saved}
          className="text-xs font-medium text-app-success"
        >
          {t(CHOICE_SAVED[choice])}
        </span>
      ) : null}

      {choice ? (
        <div className="flex flex-col gap-2">
          <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            {t(strings.reviewerFeedback.noteLabel)}
            <textarea
              data-testid={selectors.activity.safetyFeedback.note}
              value={note}
              rows={2}
              onChange={(event) => setNote(event.target.value)}
              placeholder={t(strings.reviewerFeedback.notePlaceholder)}
              className="rounded-control border border-app-border bg-app-surface px-2 py-1.5 text-sm text-app-foreground"
            />
          </label>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              size="sm"
              data-testid={selectors.activity.safetyFeedback.save}
              disabled={pending || !noteChanged}
              onClick={() => void save(noteSaveChoice, note)}
              shape="square"
            >
              {pending ? t(strings.reviewerFeedback.saving) : t(strings.reviewerFeedback.saveNote)}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              data-testid={selectors.activity.safetyFeedback.remove}
              disabled={pending}
              onClick={() => void remove()}
              shape="square"
              className="border border-app-border text-app-foreground hover:bg-app-surface"
            >
              {t(strings.reviewerFeedback.remove)}
            </Button>
          </div>
        </div>
      ) : null}

      <p className="text-[11px] text-app-muted-foreground">{t(strings.reviewerFeedback.hint)}</p>
      {error ? (
        <p data-testid={selectors.activity.safetyFeedback.error} role="alert" className="text-xs text-app-danger">
          {t(strings.reviewerFeedback.error)}: {error}
        </p>
      ) : null}
    </div>
  );
}
