import { useEffect, useState } from "react";
import { Star } from "lucide-react";

import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { MAX_RATING, MAX_RATING_NOTE, type JobRating } from "../../api/jobs";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { cn } from "../../lib/utils";
import { liveRatingClient, type RatingClient } from "./ratingClient";

export interface JobRatingControlProps {
  /** The finished job whose output is being rated. */
  jobId: string;
  client?: RatingClient;
  className?: string;
}

const SCORES = Array.from({ length: MAX_RATING }, (_, i) => i + 1);

/**
 * A small rating control for a finished job's output: five touch-sized stars and
 * an optional note. Picking a star saves immediately (one tap on a phone); the
 * note is saved with Save rating once typed. Ratings feed the model verdicts, so
 * the control says so. It never renders the image: it sits beside whatever the
 * result view shows (including a blurred sensitive result) and only deals in the
 * job id.
 */
export function JobRatingControl({ jobId, client = liveRatingClient, className }: JobRatingControlProps) {
  const { t } = useTranslation();
  const [rating, setRating] = useState<JobRating | undefined>(undefined);
  const [note, setNote] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Pick up a rating that already exists for this job (reopened result), and
  // reset when the control is pointed at a different job.
  useEffect(() => {
    let live = true;
    setRating(undefined);
    setNote("");
    setError(null);
    client
      .load(jobId)
      .then((existing) => {
        if (live && existing) {
          setRating(existing);
          setNote(existing.note);
        }
      })
      .catch(() => {
        // An unreadable existing rating is not an error worth showing: the
        // control still works and saving replaces it.
      });
    return () => {
      live = false;
    };
  }, [client, jobId]);

  const save = async (score: number, text: string) => {
    setPending(true);
    setError(null);
    try {
      const stored = await client.rate(jobId, score, text);
      setRating(stored ?? ({ jobId, score, note: text } as JobRating));
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
      await client.clear(jobId);
      setRating(undefined);
      setNote("");
    } catch (err) {
      setError(errorMessage(err, t));
    } finally {
      setPending(false);
    }
  };

  const score = rating?.score ?? 0;
  const noteChanged = rating !== undefined && note.trim() !== rating.note;

  return (
    <div
      data-testid={selectors.workspace.rating.root}
      className={cn("flex flex-col gap-2 rounded-control border border-app-border bg-app-surface p-3", className)}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
          {t(strings.workspace.rating.title)}
        </span>
        {rating ? (
          <span data-testid={selectors.workspace.rating.saved} className="text-xs text-app-success">
            {t(strings.workspace.rating.saved, { score })}
          </span>
        ) : null}
      </div>

      <div role="group" aria-label={t(strings.workspace.rating.groupLabel)} className="flex items-center gap-1">
        {SCORES.map((value) => (
          <button
            key={value}
            type="button"
            data-testid={selectors.workspace.ratingScore({ score: value })}
            aria-label={t(strings.workspace.rating.scoreLabel, { score: value })}
            aria-pressed={value === score}
            disabled={pending}
            onClick={() => void save(value, note)}
            // 44px square: the minimum comfortable touch target on a phone.
            className="flex h-11 w-11 items-center justify-center rounded-control text-app-warning hover:bg-app-surface-muted disabled:opacity-60"
          >
            <Star aria-hidden="true" className={cn("h-6 w-6", value <= score ? "fill-current" : "fill-transparent")} />
          </button>
        ))}
      </div>

      {rating ? (
        <div className="flex flex-col gap-2">
          <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            {t(strings.workspace.rating.noteLabel)}
            <textarea
              data-testid={selectors.workspace.rating.note}
              value={note}
              maxLength={MAX_RATING_NOTE}
              rows={2}
              placeholder={t(strings.workspace.rating.notePlaceholder)}
              onChange={(e) => setNote(e.target.value)}
              className="rounded-control border border-app-border bg-app-surface px-2 py-1.5 text-sm text-app-foreground"
            />
          </label>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              size="sm"
              data-testid={selectors.workspace.rating.save}
              disabled={pending || !noteChanged}
              onClick={() => void save(score, note)}
              shape="square"
            >
              {pending ? t(strings.workspace.rating.saving) : t(strings.workspace.rating.save)}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              data-testid={selectors.workspace.rating.clear}
              disabled={pending}
              onClick={() => void remove()}
              shape="square"
              className="border border-app-border text-app-foreground hover:bg-app-surface-muted"
            >
              {t(strings.workspace.rating.clear)}
            </Button>
          </div>
        </div>
      ) : null}

      <p className="text-[11px] text-app-muted-foreground">{t(strings.workspace.rating.hint)}</p>
      {error ? (
        <p data-testid={selectors.workspace.rating.error} role="alert" className="text-xs text-app-danger">
          {t(strings.workspace.rating.error)}: {error}
        </p>
      ) : null}
    </div>
  );
}
