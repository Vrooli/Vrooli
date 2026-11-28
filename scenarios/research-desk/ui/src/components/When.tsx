import type { DerivedTime } from "@vrooli/proto-types/research-desk/v1/shared/evidence_pb";
import { RelativeTime } from "@vrooli/react-component-library/RelativeTime/1.0.5";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { absolute, relative, toDate } from "../lib/format";

/** A relative time with the absolute time as its accessible title. */
export function When({ date }: { date: Date }) {
  return <RelativeTime value={relative(date)} dateTime={date.toISOString()} title={absolute(date)} />;
}

/** The time represented by a read snapshot, separate from source-file mtimes. */
export function LastObserved({ date }: { date?: Date | null }) {
  const { t } = useTranslation();
  return (
    <p aria-label={t(strings.common.lastObserved)} className="text-caption text-app-muted-foreground">
      {t(strings.common.lastObserved)}{" "}
      {date ? <When date={date} /> : t(strings.common.unknown)}
    </p>
  );
}

/**
 * A derived time as the API states it: "at least 3 hours ago" when the walk
 * that found it was cut short, so a lower bound never reads as
 * the exact newest write. Renders nothing when the value is unknown.
 */
export function DerivedWhen({ time }: { time?: DerivedTime }) {
  const { t } = useTranslation();
  const date = toDate(time?.value);
  if (!date) return null;
  const atLeast = time?.atLeast ?? false;
  return (
    <>
      {atLeast ? `${t(strings.common.atLeast)} ` : null}
      <When date={date} />
    </>
  );
}
