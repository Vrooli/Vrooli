import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, XCircle } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import {
  countCoverage,
  countFailed,
  DEFAULT_WINDOW_TOKEN,
  TimeWindowToken,
  WINDOW_TOKENS,
  type WindowToken,
} from "../../api/measures";

/**
 * Map each offered window token to its i18n label key. Keeping this as an
 * exhaustive `Record<WindowToken, …>` means adding a token to `WINDOW_TOKENS`
 * without giving it a label is a compile-time error.
 */
const WINDOW_LABEL_KEY = {
  [TimeWindowToken.THIS_WEEK]: strings.measures.window.thisWeek,
  [TimeWindowToken.LAST_7D]: strings.measures.window.last7d,
  [TimeWindowToken.LAST_30D]: strings.measures.window.last30d,
  [TimeWindowToken.THIS_MONTH]: strings.measures.window.thisMonth,
  [TimeWindowToken.LAST_MONTH]: strings.measures.window.lastMonth,
  [TimeWindowToken.THIS_QUARTER]: strings.measures.window.thisQuarter,
} as const satisfies Record<WindowToken, string>;

/**
 * MeasuresCard surfaces measures-health's own declared measures — the
 * failed/passed validation counts over its `validation_run` history — for a
 * selectable time window. It calls the two read-only `MeasuresService` RPCs
 * (`countFailedValidations` / `countValidationCoverage`) and mirrors the fleet
 * surfaces' loading/error conventions. The window selector drives the query
 * key, so changing the window refetches both counts.
 */
export function MeasuresCard() {
  const { t } = useTranslation();
  const [token, setToken] = useState<WindowToken>(DEFAULT_WINDOW_TOKEN);

  const query = useQuery({
    queryKey: ["measures", token],
    queryFn: async () => {
      const [failed, passed] = await Promise.all([countFailed(token), countCoverage(token)]);
      return { failed, passed };
    },
  });

  return (
    <section
      data-testid={selectors.measures.card}
      aria-label={t(strings.measures.title)}
      className="rounded-xl border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-sm font-medium text-app-muted-foreground">{t(strings.measures.title)}</h2>
        <label className="ms-auto flex items-center gap-2 text-xs text-app-muted-foreground">
          <span>{t(strings.measures.windowLabel)}</span>
          <select
            data-testid={selectors.measures.windowSelect}
            aria-label={t(strings.measures.windowLabel)}
            value={token}
            onChange={(e) => {
              const next = WINDOW_TOKENS.find((value) => String(value) === e.target.value);
              if (next !== undefined) setToken(next);
            }}
            className="rounded-control border border-app-border bg-app-background px-2 py-1 text-app-foreground"
          >
            {WINDOW_TOKENS.map((value) => (
              <option key={value} value={value}>
                {t(WINDOW_LABEL_KEY[value])}
              </option>
            ))}
          </select>
        </label>
      </div>

      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.measures.description)}</p>

      {query.isLoading && (
        <p data-testid={selectors.measures.loading} className="mt-4 text-app-muted-foreground">
          {t(strings.measures.loading)}
        </p>
      )}

      {query.error && (
        <p data-testid={selectors.measures.error} className="mt-4 text-red-400">
          {errorMessage(query.error, t)}
        </p>
      )}

      {query.data && (
        <dl className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="rounded-lg border border-red-500/40 bg-red-500/10 p-3">
            <dd
              data-testid={selectors.measures.failedValue}
              className="flex items-center gap-2 text-red-300"
            >
              <XCircle aria-hidden="true" className="h-4 w-4" />
              <span className="text-sm font-medium">
                {t(strings.measures.failedLabel, { count: Number(query.data.failed) })}
              </span>
            </dd>
          </div>
          <div className="rounded-lg border border-emerald-500/40 bg-emerald-500/10 p-3">
            <dd
              data-testid={selectors.measures.passedValue}
              className="flex items-center gap-2 text-emerald-300"
            >
              <CheckCircle2 aria-hidden="true" className="h-4 w-4" />
              <span className="text-sm font-medium">
                {t(strings.measures.passedLabel, { count: Number(query.data.passed) })}
              </span>
            </dd>
          </div>
        </dl>
      )}
    </section>
  );
}
