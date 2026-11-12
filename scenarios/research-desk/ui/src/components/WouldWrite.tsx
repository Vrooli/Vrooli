import type { UseQueryResult } from "@tanstack/react-query";

import type { EntryReading } from "../api/activity";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";
import { Exact } from "./Confirm";

/**
 * The exact text a write would append, as the API's dry run returned it
 * (rulings P5), and when the supervisor will read it (V4). The desk never
 * rebuilds the file's grammar itself; while the dry run is pending or refused
 * the confirmation says so instead of guessing.
 */
export function WouldWrite({ label, query }: { label: string; query: UseQueryResult<{ text: string; reading?: EntryReading }> }) {
  const { t } = useTranslation();
  if (query.isError) return <p className="text-body-sm text-app-danger">{errorMessage(query.error, t)}</p>;
  if (!query.data) return <p className="text-body-sm text-app-muted-foreground">{t(strings.entry.preparing)}</p>;
  return (
    <>
      <Exact label={label} text={query.data.text} />
      {query.data.reading?.message ? (
        <p className="text-body-sm">
          <span className="font-semibold">{t(strings.entry.readWhen)}</span> {query.data.reading.message}
        </p>
      ) : null}
    </>
  );
}
