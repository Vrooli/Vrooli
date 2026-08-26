import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1.1.0";
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
    <Card
      data-testid={selectors.notes.measure.card}
      aria-label={t(strings.notes.measure.title)}
    >
      <CardHeader>
        <CardTitle>{t(strings.notes.measure.title)}</CardTitle>
      </CardHeader>
      <CardContent>
      {countQuery.isLoading && (
        <p data-testid={selectors.notes.measure.loading} className="text-sm text-app-muted-foreground">
          {t(strings.notes.measure.loading)}
        </p>
      )}
      {countQuery.error && (
        <p data-testid={selectors.notes.measure.error} className="text-sm text-app-danger">
          {errorMessage(countQuery.error, t)}
        </p>
      )}
      {countQuery.data !== undefined && (
        <p data-testid={selectors.notes.measure.value} className="text-2xl font-semibold text-app-foreground">
          {t(strings.notes.measure.thisWeek, { count: countQuery.data })}
        </p>
      )}
      </CardContent>
    </Card>
  );
}
