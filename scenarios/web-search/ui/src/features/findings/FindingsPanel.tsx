import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { findingsClient } from "../../api/clients";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { FindingStatus } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";
import { AddFindingForm } from "./AddFindingForm";
import { FindingCard } from "./FindingCard";

type Filter = "all" | "active" | "disputed" | "superseded";

const FILTER_STATUS: Record<Filter, FindingStatus> = {
  all: FindingStatus.UNSPECIFIED,
  active: FindingStatus.ACTIVE,
  disputed: FindingStatus.DISPUTED,
  superseded: FindingStatus.SUPERSEDED,
};

const FILTER_LABEL = {
  all: strings.findings.filterAll,
  active: strings.findings.filterActive,
  disputed: strings.findings.filterDisputed,
  superseded: strings.findings.filterSuperseded,
} as const satisfies Record<Filter, string>;

const FILTERS: readonly Filter[] = ["all", "active", "disputed", "superseded"];

/**
 * FindingsPanel is the findings-management surface: a status filter (with an
 * include-superseded toggle), the live list of findings, a prune action
 * (dry-run first), and the manual Add form. List data is fetched per
 * (filter, includeArchived) and mutations invalidate it to stay consistent.
 */
export function FindingsPanel() {
  const { t } = useTranslation();
  const [filter, setFilter] = useState<Filter>("all");
  // The "superseded" filter inherently needs archived rows; otherwise the
  // toggle is the user's choice.
  const [includeArchivedPref, setIncludeArchivedPref] = useState(false);
  const includeArchived = filter === "superseded" || includeArchivedPref;

  const findingsKey = ["findings", filter, includeArchived] as const;

  const query = useQuery({
    queryKey: findingsKey,
    queryFn: async () =>
      findingsClient.listFindings({
        status: FILTER_STATUS[filter],
        includeArchived,
        limit: 100,
      }),
  });

  const [pruneMessage, setPruneMessage] = useState<string | null>(null);
  const pruneMutation = useMutation({
    mutationFn: async (dryRun: boolean) => {
      const res = await findingsClient.pruneFindings({ dryRun });
      return { res, dryRun };
    },
    onSuccess: ({ res, dryRun }) => {
      const count = res.pruned;
      setPruneMessage(
        dryRun
          ? t(strings.findings.pruneDryRunResult, { count })
          : t(strings.findings.pruneApplied, { count }),
      );
      if (!dryRun) {
        void query.refetch();
      }
    },
  });

  const findings = query.data?.findings ?? [];

  return (
    <div data-testid={selectors.findings.panel} className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <div role="radiogroup" aria-label={t(strings.findings.filterLabel)} className="flex flex-wrap gap-2">
          {FILTERS.map((f) => (
            <button
              key={f}
              type="button"
              role="radio"
              aria-checked={filter === f}
              onClick={() => setFilter(f)}
              className={
                filter === f
                  ? "rounded-control bg-app-primary px-3 py-1 text-sm text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {t(FILTER_LABEL[f])}
            </button>
          ))}
        </div>
        <label className="flex items-center gap-2 text-sm text-app-foreground">
          <input
            type="checkbox"
            data-testid={selectors.findings.includeArchived}
            checked={includeArchived}
            disabled={filter === "superseded"}
            onChange={(e) => setIncludeArchivedPref(e.target.checked)}
            className="h-4 w-4 accent-app-primary"
          />
          {t(strings.findings.includeArchivedLabel)}
        </label>
      </div>

      <section
        aria-labelledby="prune-heading"
        className="flex flex-wrap items-center gap-3 rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 id="prune-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.findings.pruneHeading)}
        </h3>
        <Button
          data-testid={selectors.findings.pruneDryRun}
          variant="outline"
          size="sm"
          disabled={pruneMutation.isPending}
          onClick={() => pruneMutation.mutate(true)}
        >
          {t(strings.findings.pruneDryRun)}
        </Button>
        <Button
          data-testid={selectors.findings.pruneApply}
          size="sm"
          disabled={pruneMutation.isPending}
          onClick={() => pruneMutation.mutate(false)}
        >
          {t(strings.findings.pruneApply)}
        </Button>
        {pruneMutation.error != null && (
          <p className="text-sm text-app-danger">
            {t(strings.findings.pruneError, { message: errorMessage(pruneMutation.error, t) })}
          </p>
        )}
        {pruneMessage != null && pruneMutation.error == null && (
          <p data-testid={selectors.findings.pruneResult} className="text-sm text-app-muted-foreground">
            {pruneMessage}
          </p>
        )}
      </section>

      {query.isLoading && (
        <p data-testid={selectors.findings.loading} className="text-sm text-app-muted-foreground">
          {t(strings.findings.loading)}
        </p>
      )}
      {query.error != null && (
        <p data-testid={selectors.findings.error} className="text-sm text-app-danger">
          {t(strings.findings.error, { message: errorMessage(query.error, t) })}
        </p>
      )}
      {query.data && findings.length === 0 && (
        <p data-testid={selectors.findings.empty} className="text-sm text-app-muted-foreground">
          {t(strings.findings.empty)}
        </p>
      )}
      {findings.length > 0 && (
        <ul data-testid={selectors.findings.list} className="flex flex-col gap-3">
          {findings.map((finding) => (
            <FindingCard key={finding.id} finding={finding} findingsKey={findingsKey} />
          ))}
        </ul>
      )}

      <AddFindingForm findingsKey={findingsKey} />
    </div>
  );
}
