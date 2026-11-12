import type { DerivedTime } from "@vrooli/proto-types/research-desk/v1/shared/evidence_pb";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { absolute, relative, toDate } from "../lib/format";

/** A relative time with the absolute time as its accessible title. */
export function When({ date }: { date: Date }) {
  return (
    <time dateTime={date.toISOString()} title={absolute(date)}>
      {relative(date)}
    </time>
  );
}

/**
 * A derived time as the API states it: "at least 3 hours ago" when the walk
 * that found it was cut short (rulings F3), so a lower bound never reads as
 * the exact newest write. Renders nothing when the value is unknown.
 */
export function DerivedWhen({ time }: { time?: DerivedTime }) {
  const { t } = useTranslation();
  const date = toDate(time?.value);
  if (!date) return null;
  const atLeast = time?.evidence?.reason.startsWith("at least") ?? false;
  return (
    <>
      {atLeast ? `${t(strings.common.atLeast)} ` : null}
      <When date={date} />
    </>
  );
}
