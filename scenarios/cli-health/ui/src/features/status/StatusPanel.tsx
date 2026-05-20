import { useMutation, useQuery } from "@tanstack/react-query";

import { reindexClient, searchClient } from "../../api/clients";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { ReindexResponse } from "@vrooli/proto-types/cli-health/v1/reindex/reindex_pb";
import type { StatusResponse } from "@vrooli/proto-types/cli-health/v1/search/search_pb";

const formatTimestamp = (ts: string) => {
  if (!ts) return null;
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return ts;
  return d.toISOString().replace("T", " ").replace(/\.\d+Z$/, "Z");
};

export function StatusPanel() {
  const { t } = useTranslation();

  const status = useQuery<StatusResponse>({
    queryKey: ["search-status"],
    queryFn: () => searchClient.status({}),
    refetchInterval: 30_000,
  });

  const reindex = useMutation<ReindexResponse>({
    mutationFn: () => reindexClient.reindex({ scenario: "", dryRun: false }),
    onSuccess: () => {
      void status.refetch();
    },
  });

  const yesNo = (b: boolean) => (b ? t(strings.status.yes) : t(strings.status.no));

  return (
    <section
      data-testid={selectors.status.card}
      className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-lg font-semibold">{t(strings.status.title)}</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.status.description)}</p>

      {status.isLoading && (
        <p data-testid={selectors.status.loading} className="mt-3 text-sm text-app-muted-foreground">
          {t(strings.status.loading)}
        </p>
      )}
      {status.error && (
        <p data-testid={selectors.status.error} className="mt-3 text-sm text-red-400">
          {t(strings.status.error, { message: status.error.message })}
        </p>
      )}
      {status.data && (
        <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <dt className="text-app-muted-foreground">{t(strings.status.title)}:</dt>
          <dd data-testid={selectors.status.available}>
            {status.data.available
              ? t(strings.status.availableYes)
              : t(strings.status.availableNo)}
          </dd>
          <dt className="text-app-muted-foreground">{t(strings.status.ollamaLabel)}</dt>
          <dd data-testid={selectors.status.ollama}>{yesNo(status.data.ollama)}</dd>
          <dt className="text-app-muted-foreground">{t(strings.status.qdrantLabel)}</dt>
          <dd data-testid={selectors.status.qdrant}>{yesNo(status.data.qdrant)}</dd>
          <dt className="text-app-muted-foreground">{t(strings.status.indexedLabel)}</dt>
          <dd data-testid={selectors.status.indexed}>{status.data.indexedCount}</dd>
          <dt className="text-app-muted-foreground">{t(strings.status.lastReconcileLabel)}</dt>
          <dd data-testid={selectors.status.lastReconcile}>
            {formatTimestamp(status.data.lastReconcileAt) ?? t(strings.status.never)}
            {status.data.lastReconcileOutcome && (
              <span className="ms-2 text-app-muted-foreground">
                ({status.data.lastReconcileOutcome})
              </span>
            )}
          </dd>
        </dl>
      )}

      <div className="mt-3">
        <Button
          data-testid={selectors.status.reindex}
          onClick={() => reindex.mutate()}
          disabled={reindex.isPending}
        >
          {reindex.isPending ? t(strings.status.reindexing) : t(strings.status.reindex)}
        </Button>
        {reindex.data && (
          <p data-testid={selectors.status.reindexed} className="mt-2 text-xs text-app-muted-foreground">
            {t(strings.status.reindexed, {
              jobId: reindex.data.jobId,
              upserts: reindex.data.plannedUpserts,
              deletes: reindex.data.plannedDeletes,
            })}
          </p>
        )}
      </div>
    </section>
  );
}
