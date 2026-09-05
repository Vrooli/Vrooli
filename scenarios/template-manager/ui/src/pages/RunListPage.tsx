import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

import { fetchValidationRunList } from "../api/templateDomain";
import { DetailError, DetailLoading } from "../components/detail/DetailStates";
import { EntityList, EntityRowBody } from "../components/list/EntityList";
import { Select, type SelectOption } from "@vrooli/react-component-library/Select/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { modeLabel, runStatusTone } from "../lib/templateLabels";

const TITLE_ID = "run-list-heading";
const STATUS_PARAM = "status";

export function RunListPage() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const query = useQuery({ queryKey: ["run-list"] as const, queryFn: fetchValidationRunList });

  const statusFilter = params.get(STATUS_PARAM) ?? "";
  const setStatus = (value: string) => {
    const next = new URLSearchParams(params);
    if (value) {
      next.set(STATUS_PARAM, value);
    } else {
      next.delete(STATUS_PARAM);
    }
    setParams(next, { replace: true });
  };

  const runs = useMemo(() => query.data ?? [], [query.data]);
  const statusOptions = useMemo(
    () => buildStatusOptions(runs.map((run) => run.status), t(strings.runList.filters.allStatuses)),
    [runs, t],
  );
  const filtered = useMemo(
    () => (statusFilter ? runs.filter((run) => run.status === statusFilter) : runs),
    [runs, statusFilter],
  );

  return (
    <section data-testid={selectors.pages.runList} aria-labelledby={TITLE_ID} className="flex flex-col gap-4">
      <div className="min-w-0">
        <h2 id={TITLE_ID} className="text-2xl font-semibold">
          {t(strings.pages.runList.title)}
        </h2>
        <p className="mt-1 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.runList.description)}
        </p>
      </div>

      {query.isLoading && <DetailLoading testId={selectors.runList.loading} />}
      {query.isError && <DetailError testId={selectors.runList.error} />}

      {query.data && (
        <div data-testid={selectors.runList.root} className="flex flex-col gap-3">
          <label className="flex max-w-xs flex-col gap-1">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.runList.filters.status)}
            </span>
            <Select
              data-testid={selectors.runList.statusFilter}
              aria-label={t(strings.runList.filters.status)}
              options={statusOptions}
              value={statusFilter}
              onChange={(event) => setStatus(event.target.value)}
            />
          </label>

          <EntityList
            rows={filtered}
            tableTestId={selectors.runList.table}
            getRowKey={(row) => row.id}
            getRowHref={(row) => `/runs/${encodeURIComponent(row.id)}`}
            getRowTestId={(row) => selectors.runList.row({ id: row.id })}
            getRowLabel={(row) => `${t(strings.runList.viewEntry)}: ${row.id}`}
            renderRow={(row) => (
              <EntityRowBody
                primary={row.id}
                secondary={`${row.templateId} · ${modeLabel(row.mode)} · ${row.findings.length}`}
                trailing={<StatusBadge tone={runStatusTone(row.status)}>{row.status}</StatusBadge>}
              />
            )}
            searchText={(row) => `${row.id} ${row.templateId}`}
            searchLabel={t(strings.runList.search)}
            searchPlaceholder={t(strings.runList.searchPlaceholder)}
            emptyLabel={t(strings.runList.empty)}
          />
        </div>
      )}
    </section>
  );
}

function buildStatusOptions(statuses: string[], allLabel: string): SelectOption[] {
  const distinct = [...new Set(statuses.filter(Boolean))].sort();
  return [{ value: "", label: allLabel }, ...distinct.map((value) => ({ value, label: value }))];
}
