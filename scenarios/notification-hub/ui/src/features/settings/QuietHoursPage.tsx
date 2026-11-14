import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Moon } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { ListGroup, ListItem } from "../../components/ListRow";
import { SkeletonRows } from "../../components/Skeleton";
import { strings } from "../../consts/strings";
import { deviceTimeZone } from "./format";
import { useTranslation } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { describeDays, describeHours } from "./describeSchedule";
import { Notice } from "./Notice";
import { OwnerGate } from "./OwnerGate";
import { quietWindowsQuery } from "./ownerQueries";
import { dayKey, groupSchedules, sortDays, type Schedule, type Weekday } from "./quietSchedule";
import { QuietScheduleEditor, type EditorResult } from "./QuietScheduleEditor";
import { RowButton } from "./RowButton";
import { SettingsScreen } from "./SettingsScreen";

/** Weekday hours for a first schedule: a reasonable night, not a blank form. */
function newSchedule(taken: readonly Weekday[]): Schedule {
  const free = ([1, 2, 3, 4, 5, 6, 0] as const).filter((day) => !taken.includes(day));
  return { days: sortDays(free.length === 7 ? [1, 2, 3, 4, 5] : free), start: "12:00", end: "13:00", timezone: deviceTimeZone(), criticalOverride: false };
}

export function QuietHoursPage() {
  const { t } = useTranslation();
  return (
    <SettingsScreen title={t(strings.settings.section.quietHours)} description={t(strings.settings.quietHours.description)} backToSettings>
      <OwnerGate><QuietHoursContent /></OwnerGate>
    </SettingsScreen>
  );
}

type Editing = { kind: "new" } | { kind: "edit"; schedule: Schedule };

function QuietHoursContent() {
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const windows = useQuery(quietWindowsQuery);
  const [editing, setEditing] = useState<Editing | null>(null);
  const [result, setResult] = useState<EditorResult | null>(null);

  if (windows.isPending) return <SkeletonRows rows={2} />;
  if (windows.isError && !windows.data) {
    return <QueryError error={windows.error} title={t(strings.settings.quietHours.loadFailed)} detail={t(strings.settings.quietHours.loadFailedDetail)} onRetry={() => void windows.refetch()} retrying={windows.isFetching} />;
  }
  const stored = windows.data.windows;
  const schedules = groupSchedules(stored);
  const used = schedules.flatMap((schedule) => schedule.days);
  const openEditor = (next: Editing) => { setResult(null); setEditing(next); };
  const add = (
    <Button className="min-h-11" type="button" variant="outline" onClick={() => openEditor({ kind: "new" })} data-testid={selectors.quietHours.add}>{t(strings.settings.quietHours.add)}</Button>
  );

  return (
    <>
      {result && (
        <Notice tone="success" testId={selectors.quietHours.message}>
          {t(result === "saved" ? strings.settings.quietHours.saved : strings.settings.quietHours.removed)}
        </Notice>
      )}
      {windows.isError && <Notice tone="warning">{t(strings.settings.quietHours.stale)}</Notice>}
      {schedules.length === 0 ? (
        <EmptyNotice icon={<Moon size={28} />} title={t(strings.settings.quietHours.emptyTitle)} detail={t(strings.settings.quietHours.emptyDetail)}>
          <div className="mt-2">{add}</div>
        </EmptyNotice>
      ) : (
        <>
          <ListGroup aria-label={t(strings.settings.section.quietHours)}>
            {schedules.map((schedule) => (
              <ListItem key={schedule.days.join()}>
                <RowButton
                  onClick={() => openEditor({ kind: "edit", schedule })}
                  data-testid={selectors.quietHours.schedule({ day: dayKey(schedule.days[0] ?? 0) })}
                  trailing={schedule.criticalOverride ? <span className="mt-0.5 shrink-0 rounded-full bg-app-surface-muted px-2 py-0.5 text-xs">{t(strings.settings.quietHours.criticalChip)}</span> : undefined}
                >
                  <span className="flex flex-col">
                    <span className="text-base font-semibold">{describeDays(schedule.days, lang, t)}</span>
                    <span className="text-sm text-app-muted-foreground">{describeHours(schedule, lang)}</span>
                    <span className="text-sm text-app-muted-foreground [overflow-wrap:anywhere]">{schedule.timezone}</span>
                  </span>
                </RowButton>
              </ListItem>
            ))}
          </ListGroup>
          {used.length < 7 && <div>{add}</div>}
        </>
      )}
      {editing && (
        <QuietScheduleEditor
          key={editing.kind === "edit" ? editing.schedule.days.join() : "new"}
          stored={stored}
          initial={editing.kind === "edit" ? editing.schedule : newSchedule(used)}
          existing={editing.kind === "edit"}
          takenElsewhere={editing.kind === "edit" ? used.filter((day) => !editing.schedule.days.includes(day)) : used}
          onClose={() => setEditing(null)}
          onDone={(done) => { setEditing(null); setResult(done); }}
        />
      )}
    </>
  );
}
