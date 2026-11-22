import { useCallback, useEffect, useState } from "react";

import type { PublishRecord } from "@vrooli/proto-types/content-desk/v1/ledger/ledger_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";

import { ledgerClient } from "../api/ledger";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type Result = {
  record: PublishRecord;
  readings: Array<{ metric: string; value: number; state: string; sampleCount: number; lastObservedAt: string }>;
  hasMeasurements: boolean;
  measurementError: boolean;
};

export function LearnPage() {
  const { t } = useTranslation();
  const [results, setResults] = useState<Result[]>([]);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    setState("loading");
    try {
      const response = await ledgerClient.listPublishRecords({});
      const next = await Promise.all(response.publishRecords.map(async (record): Promise<Result> => {
        try {
          const metrics = await ledgerClient.getDraftMetrics({ draftId: record.draftId, staleAfterDays: 30 });
          return { record, readings: metrics.readings, hasMeasurements: metrics.hasMeasurements, measurementError: false };
        } catch {
          return { record, readings: [], hasMeasurements: false, measurementError: true };
        }
      }));
      setResults(next);
      setError("");
      setState("ready");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t(strings.pages.learn.loadError));
      setState("error");
    }
  }, [t]);

  useEffect(() => { void refresh(); }, [refresh]);

  return <main className="mx-auto w-full max-w-5xl space-y-6 px-4 py-6 sm:px-6 lg:px-8" data-experience-surface="learn" data-testid="page-learn">
    <header>
      <p className="text-sm font-semibold uppercase tracking-[0.1em] text-app-primary">{t(strings.pages.learn.eyebrow)}</p>
      <h1 className="mt-1 font-serif text-4xl font-medium tracking-tight">{t(strings.pages.learn.title)}</h1>
      <p className="mt-2 max-w-2xl text-sm text-app-muted-foreground">{t(strings.pages.learn.description)}</p>
    </header>
    <Card>
      <CardHeader><CardTitle as="h2">{t(strings.pages.learn.resultsTitle)}</CardTitle></CardHeader>
      <CardContent className="space-y-4">
        {state === "loading" ? <p aria-live="polite">{t(strings.pages.learn.loading)}</p> : null}
        {state === "error" ? <section role="alert" className="rounded-control border border-app-danger/40 p-4"><p>{error}</p><Button className="mt-3" variant="secondary" onClick={() => void refresh()}>{t(strings.pages.learn.retry)}</Button></section> : null}
        {state === "ready" && results.length === 0 ? <section className="rounded-control border border-dashed border-app-border p-5"><h3 className="font-semibold">{t(strings.pages.learn.emptyTitle)}</h3><p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.learn.emptyDescription)}</p></section> : null}
        {state === "ready" && results.length > 0 ? <ul className="grid gap-4">
          {results.map(({ record, readings, hasMeasurements, measurementError }) => <li key={record.id}>
            <article className="rounded-panel border border-app-border bg-app-surface p-4">
              <h3 className="break-all font-medium">{record.draftId}</h3>
              <p className="mt-1 break-all text-sm text-app-muted-foreground">{record.publishedUrl || record.platformPostId || t(strings.pages.learn.noReceiptLink)}</p>
              {measurementError ? <p className="mt-3 text-sm text-app-danger">{t(strings.pages.learn.measurementsUnavailable)}</p> : null}
              {!measurementError && !hasMeasurements ? <p className="mt-3 text-sm text-app-muted-foreground">{t(strings.pages.learn.unknownMeasurements)}</p> : null}
              {hasMeasurements ? <ul className="mt-3 grid gap-2 sm:grid-cols-2">
                {readings.map((reading) => <li key={reading.metric} className="rounded-control border border-app-border p-3">
                  <p className="font-medium">{reading.metric}: {reading.value}</p>
                  <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.pages.learn.readingMeta, { state: reading.state, count: reading.sampleCount, observed: reading.lastObservedAt || t(strings.pages.learn.unknownTime) })}</p>
                </li>)}
              </ul> : null}
            </article>
          </li>)}
        </ul> : null}
        <p className="text-xs text-app-muted-foreground">{t(strings.pages.learn.limits)}</p>
      </CardContent>
    </Card>
  </main>;
}
