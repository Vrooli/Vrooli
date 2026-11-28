import { useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Button } from "@vrooli/react-component-library/Button/2";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Switch } from "@vrooli/react-component-library/Switch/1";
import { recipientsClient } from "../../api/notifications";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { dialogPrefixes, selectors } from "../../consts/selectors";
import { deviceTimeZone, formatClock, formatList, timeZoneChoices, weekdayName } from "./format";
import { describeDays } from "./describeSchedule";
import { Notice } from "./Notice";
import { quietWindowsQuery, readFresh } from "./ownerQueries";
import {
  WEEKDAYS,
  WEEKEND,
  WEEK_ORDER,
  applyChanges,
  dayKey,
  isOvernight,
  planChanges,
  scheduleIssue,
  sortDays,
  type Schedule,
  type ScheduleIssue,
  type StoredWindow,
  type Weekday,
  type WindowChange,
} from "./quietSchedule";

export type EditorResult = "saved" | "removed";

const ISSUE_COPY = {
  noDays: strings.settings.quietHours.issue.noDays,
  badTime: strings.settings.quietHours.issue.badTime,
  sameTime: strings.settings.quietHours.issue.sameTime,
  badZone: strings.settings.quietHours.issue.badZone,
} as const satisfies Record<ScheduleIssue, string>;

const FIELD = "min-h-11 w-full rounded-control border border-app-border bg-app-background px-3 text-base text-app-foreground focus-visible:outline focus-visible:outline-2";

async function writeChange(change: WindowChange): Promise<void> {
  if (change.kind === "delete") {
    await recipientsClient.deleteQuietWindow({ id: change.id });
    return;
  }
  await recipientsClient.setQuietWindow({ weekday: change.weekday, ...change.settings });
}

/**
 * One schedule, edited in a sheet over the list: which days, from when to when,
 * in which time zone, and whether critical decisions may still come through.
 * Saving trusts only what the Hub reads back, and says exactly which days did
 * not land, with the draft still in hand to try again.
 */
export function QuietScheduleEditor({ stored, initial, existing, takenElsewhere, onClose, onDone }: {
  stored: readonly StoredWindow[];
  initial: Schedule;
  existing: boolean;
  takenElsewhere: readonly Weekday[];
  onClose: () => void;
  onDone: (result: EditorResult) => void;
}) {
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<Schedule>(initial);
  const [saving, setSaving] = useState(false);
  const [problem, setProblem] = useState<{ pending: Weekday[]; applied: Weekday[]; readFailed: boolean } | null>(null);
  const [confirm, setConfirm] = useState<"discard" | "remove" | null>(null);
  const firstFieldRef = useRef<HTMLInputElement>(null);

  const issue = scheduleIssue(draft);
  const dirty = JSON.stringify(draft) !== JSON.stringify(initial);
  const zones = useMemo(() => timeZoneChoices(draft.timezone, deviceTimeZone()), [draft.timezone]);
  const patch = (next: Partial<Schedule>) => { setProblem(null); setDraft((current) => ({ ...current, ...next })); };
  const toggle = (day: Weekday) => patch({ days: sortDays(draft.days.includes(day) ? draft.days.filter((d) => d !== day) : [...draft.days, day]) });
  const pick = (days: readonly Weekday[]) => patch({ days: sortDays(days.filter((day) => !takenElsewhere.includes(day))) });
  const dayList = (days: readonly Weekday[]) => formatList(days.map((day) => weekdayName(day, lang, "long")), lang);

  const run = async (changes: WindowChange[], done: EditorResult) => {
    setSaving(true);
    setProblem(null);
    const outcome = await applyChanges(changes, {
      write: writeChange,
      readLatest: async () => (await readFresh(queryClient, quietWindowsQuery)).windows,
    });
    setSaving(false);
    if (outcome.pending.length === 0 && !outcome.readFailed) onDone(done);
    else setProblem(outcome);
  };
  // Planned against what the Hub holds now, so trying again redoes only what is missing.
  const save = () => void run(planChanges(stored, draft, initial.days), "saved");
  const remove = () => { setConfirm(null); void run(planChanges(stored, { ...draft, days: [] }, initial.days), "removed"); };
  const requestClose = () => (dirty && !saving ? setConfirm("discard") : onClose());

  const unsavedDays = problem ? dayList(problem.pending) : "";
  return (
    <>
      <ResponsiveDialog
        open
        onClose={requestClose}
        title={t(existing ? strings.settings.quietHours.editTitle : strings.settings.quietHours.addTitle)}
        closeLabel={t(strings.settings.action.close)}
        size="md"
        contentPadding="comfortable"
        initialFocusRef={firstFieldRef}
        testId={selectors.quietHours.editor}
        footer={(
          <div className="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center">
            {existing && (
              <Button type="button" variant="ghost" className="min-h-11 text-app-danger sm:me-auto" disabled={saving} onClick={() => setConfirm("remove")} data-testid={selectors.quietHours.remove}>
                {t(strings.settings.quietHours.remove)}
              </Button>
            )}
            <Button type="button" variant="outline" disabled={saving} onClick={requestClose}>{t(strings.settings.action.cancel)}</Button>
            <Button type="button" variant="primary" disabled={saving || issue !== null || !dirty} onClick={save} data-testid={selectors.quietHours.save}>
              {saving ? t(strings.settings.action.saving) : t(strings.settings.action.save)}
            </Button>
          </div>
        )}
      >
        <form className="flex flex-col gap-5" onSubmit={(event) => { event.preventDefault(); if (!saving && issue === null && dirty) save(); }}>
          <fieldset className="flex flex-col gap-2">
            <legend className="mb-1 text-sm font-semibold">{t(strings.settings.quietHours.days)}</legend>
            <div className="grid grid-cols-7 gap-1.5">
              {WEEK_ORDER.map((day) => {
                const taken = takenElsewhere.includes(day);
                const on = draft.days.includes(day);
                return (
                  <button
                    key={day}
                    type="button"
                    aria-pressed={on}
                    aria-label={weekdayName(day, lang, "long")}
                    disabled={taken}
                    title={taken ? t(strings.settings.quietHours.dayTaken) : undefined}
                    onClick={() => toggle(day)}
                    data-testid={selectors.quietHours.day({ day: dayKey(day) })}
                    className={`min-h-11 min-w-0 rounded-control border px-0.5 text-xs font-medium transition-colors duration-quick focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 disabled:border-dashed disabled:opacity-40 ${on ? "border-app-primary bg-app-primary text-app-primary-foreground" : "border-app-border bg-app-surface"}`}
                  >
                    {weekdayName(day, lang, "short")}
                  </button>
                );
              })}
            </div>
            <div className="flex flex-wrap gap-x-4">
              {([["weekdays", WEEKDAYS], ["weekends", WEEKEND], ["every", WEEK_ORDER]] as const).map(([key, days]) => (
                <button key={key} type="button" onClick={() => pick(days)} className="min-h-11 text-sm font-medium text-app-primary underline-offset-4 hover:underline focus-visible:outline focus-visible:outline-2">
                  {t(strings.settings.days[key])}
                </button>
              ))}
            </div>
            {takenElsewhere.length > 0 && <p className="text-sm text-app-muted-foreground">{t(strings.settings.quietHours.takenNote, { days: describeDays(takenElsewhere, lang, t) })}</p>}
          </fieldset>

          <div className="grid grid-cols-2 gap-3">
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t(strings.settings.quietHours.from)}
              <input ref={firstFieldRef} type="time" required value={draft.start} onChange={(event) => patch({ start: event.target.value })} aria-invalid={issue === "badTime" || issue === "sameTime"} data-testid={selectors.quietHours.start} className={FIELD} />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t(strings.settings.quietHours.until)}
              <input type="time" required value={draft.end} onChange={(event) => patch({ end: event.target.value })} aria-invalid={issue === "badTime" || issue === "sameTime"} data-testid={selectors.quietHours.end} className={FIELD} />
            </label>
          </div>

          {issue === null && isOvernight(draft) && (
            <p className="text-sm text-app-muted-foreground" data-testid={selectors.quietHours.overnight}>
              {t(strings.settings.quietHours.overnightNote, { end: formatClock(draft.end, lang) })}
            </p>
          )}

          <label className="flex flex-col gap-1 text-sm font-semibold">
            {t(strings.settings.quietHours.timezone)}
            <Select
              value={draft.timezone}
              options={zones.map((zone) => ({ value: zone, label: zone }))}
              onValueChange={(timezone) => patch({ timezone })}
              data-testid={selectors.quietHours.timezone}
            />
            <span className="text-sm font-normal text-app-muted-foreground">{t(strings.settings.quietHours.timezoneHelp)}</span>
          </label>

          <div className="flex items-start justify-between gap-4">
            <div className="flex min-w-0 flex-col">
              <span id="quiet-critical-label" className="text-sm font-semibold">{t(strings.settings.quietHours.critical)}</span>
              <span id="quiet-critical-hint" className="text-sm text-app-muted-foreground">{t(strings.settings.quietHours.criticalHint)}</span>
            </div>
            <Switch checked={draft.criticalOverride} onCheckedChange={(criticalOverride) => patch({ criticalOverride })} aria-labelledby="quiet-critical-label" aria-describedby="quiet-critical-hint" data-testid={selectors.quietHours.critical} />
          </div>

          {issue !== null && dirty && <p role="alert" className="text-sm text-app-danger" data-testid={selectors.quietHours.issue}>{t(ISSUE_COPY[issue])}</p>}
          {problem && (
            <Notice tone="danger" title={t(problem.readFailed ? strings.settings.quietHours.unconfirmedTitle : strings.settings.quietHours.partialTitle)} testId={selectors.quietHours.problem}>
              {problem.readFailed
                ? t(strings.settings.quietHours.unconfirmedBody)
                : (
                  <>
                    <p>{t(strings.settings.quietHours.partialBody, { days: unsavedDays })}</p>
                    {problem.applied.length > 0 && <p>{t(strings.settings.quietHours.partialSaved, { days: dayList(problem.applied) })}</p>}
                  </>
                )}
            </Notice>
          )}
        </form>
      </ResponsiveDialog>

      <AlertDialog
        open={confirm === "discard"}
        title={t(strings.settings.unsaved.title)}
        description={t(strings.settings.unsaved.body)}
        destructive
        confirmLabel={t(strings.settings.unsaved.leave)}
        cancelLabel={t(strings.settings.unsaved.stay)}
        closeLabel={t(strings.settings.unsaved.stay)}
        onConfirm={onClose}
        onCancel={() => setConfirm(null)}
      />
      <AlertDialog
        testIdPrefix={dialogPrefixes.quietRemoval}
        open={confirm === "remove"}
        title={t(strings.settings.quietHours.removeTitle, { days: describeDays(initial.days, lang, t) })}
        description={t(strings.settings.quietHours.removeBody)}
        destructive
        confirmLabel={t(strings.settings.quietHours.removeConfirm)}
        cancelLabel={t(strings.settings.action.cancel)}
        closeLabel={t(strings.settings.action.cancel)}
        onConfirm={remove}
        onCancel={() => setConfirm(null)}
      />
    </>
  );
}
