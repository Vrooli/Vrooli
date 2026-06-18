import { useQuery } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { countNotesInWindow } from "../../api/notes";
import { errorMessage } from "../../lib/errorMessage";

const NOTES_MEASURE_QUERY_KEY = ["notes", "measure", "this_week"] as const;

/**
 * NotesMeasureCard surfaces the `notes count` measure result — how many notes
 * were created this week — as a result card. It is the canonical reference for
 * presenting a Measures answer in a scenario UI: the same measure that
 * search-hub can answer from natural language ("how many notes this week"),
 * rendered directly. New scenarios copy this card for their own measure when
 * they replace the notes example domain.
 */
export function NotesMeasureCard() {
  const { t } = useTranslation();

  const countQuery = useQuery({
    queryKey: NOTES_MEASURE_QUERY_KEY,
    queryFn: () => countNotesInWindow(),
  });

  return (
    <section
      data-testid={selectors.notes.measure.card}
      aria-label={t(strings.notes.measure.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.notes.measure.title)}</h2>
      {countQuery.isLoading && (
        <p data-testid={selectors.notes.measure.loading} className="mt-2 text-slate-200">
          {t(strings.notes.measure.loading)}
        </p>
      )}
      {countQuery.error && (
        <p data-testid={selectors.notes.measure.error} className="mt-2 text-red-400">
          {errorMessage(countQuery.error, t)}
        </p>
      )}
      {countQuery.data !== undefined && (
        <p data-testid={selectors.notes.measure.value} className="mt-2 text-2xl font-semibold text-slate-100">
          {t(strings.notes.measure.thisWeek, { count: countQuery.data })}
        </p>
      )}
    </section>
  );
}
