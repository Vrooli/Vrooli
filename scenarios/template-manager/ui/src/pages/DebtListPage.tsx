import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

import { fetchDebtLedger, type DebtEntry, type TemplateRecord } from "../api/templateDomain";
import { DetailError, DetailLoading } from "../components/detail/DetailStates";
import { EntityList, EntityRowBody } from "../components/list/EntityList";
import { Select, type SelectOption } from "../components/ui/select";
import { StatusBadge } from "../components/ui/status-badge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { debtStatusTone } from "../lib/templateLabels";

const TITLE_ID = "debt-list-heading";
const TEMPLATE_PARAM = "template";
const STATUS_PARAM = "status";

export function DebtListPage() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const query = useQuery({ queryKey: ["debt-ledger"] as const, queryFn: fetchDebtLedger });

  const templateFilter = params.get(TEMPLATE_PARAM) ?? "";
  const statusFilter = params.get(STATUS_PARAM) ?? "";

  const setFilter = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    if (value) {
      next.set(key, value);
    } else {
      next.delete(key);
    }
    setParams(next, { replace: true });
  };

  const entries = useMemo(() => query.data?.entries ?? [], [query.data]);
  const templates = useMemo(() => query.data?.templates ?? [], [query.data]);

  const templateOptions = useMemo(
    () => buildTemplateOptions(templates, entries, t(strings.debtList.filters.allTemplates)),
    [templates, entries, t],
  );
  const statusOptions = useMemo(
    () => buildStatusOptions(entries, t(strings.debtList.filters.allStatuses)),
    [entries, t],
  );

  const filtered = useMemo(
    () =>
      entries.filter(
        (entry) =>
          (!templateFilter || entry.templateId === templateFilter) &&
          (!statusFilter || entry.status === statusFilter),
      ),
    [entries, templateFilter, statusFilter],
  );

  return (
    <section
      data-testid={selectors.pages.debtList}
      aria-labelledby={TITLE_ID}
      className="flex flex-col gap-4"
    >
      <div className="min-w-0">
        <h2 id={TITLE_ID} className="text-2xl font-semibold">
          {t(strings.pages.debtList.title)}
        </h2>
        <p className="mt-1 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.debtList.description)}
        </p>
      </div>

      {query.isLoading && <DetailLoading testId={selectors.debtList.loading} />}
      {query.isError && <DetailError testId={selectors.debtList.error} />}

      {query.data && (
        <div data-testid={selectors.debtList.root} className="flex flex-col gap-3">
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-app-muted-foreground">
                {t(strings.debtList.filters.template)}
              </span>
              <Select
                data-testid={selectors.debtList.templateFilter}
                aria-label={t(strings.debtList.filters.template)}
                options={templateOptions}
                value={templateFilter}
                onChange={(event) => setFilter(TEMPLATE_PARAM, event.target.value)}
              />
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-app-muted-foreground">
                {t(strings.debtList.filters.status)}
              </span>
              <Select
                data-testid={selectors.debtList.statusFilter}
                aria-label={t(strings.debtList.filters.status)}
                options={statusOptions}
                value={statusFilter}
                onChange={(event) => setFilter(STATUS_PARAM, event.target.value)}
              />
            </label>
          </div>

          <EntityList
            rows={filtered}
            tableTestId={selectors.debtList.table}
            getRowKey={(row) => row.key}
            getRowHref={(row) => `/debt/${encodeURIComponent(row.key)}`}
            getRowTestId={(row) => selectors.debtList.row({ key: row.key })}
            getRowLabel={(row) => `${t(strings.debtList.viewEntry)}: ${row.title || row.key}`}
            renderRow={(row) => (
              <EntityRowBody
                primary={row.title || row.key}
                secondary={`${row.templateId} · ${row.severity}`}
                trailing={<StatusBadge tone={debtStatusTone(row.status)}>{row.status}</StatusBadge>}
              />
            )}
            searchText={(row) => `${row.title} ${row.key} ${row.templateId}`}
            searchLabel={t(strings.debtList.search)}
            searchPlaceholder={t(strings.debtList.searchPlaceholder)}
            emptyLabel={t(strings.debtList.empty)}
          />
        </div>
      )}
    </section>
  );
}

function buildTemplateOptions(
  templates: TemplateRecord[],
  entries: DebtEntry[],
  allLabel: string,
): SelectOption[] {
  const labels = new Map<string, string>();
  for (const template of templates) {
    labels.set(template.id, template.displayName || template.id);
  }
  // Include any template referenced by debt but absent from the registry list.
  for (const entry of entries) {
    if (entry.templateId && !labels.has(entry.templateId)) {
      labels.set(entry.templateId, entry.templateId);
    }
  }
  const options = [...labels.entries()]
    .sort((a, b) => a[1].localeCompare(b[1]))
    .map(([value, label]) => ({ value, label }));
  return [{ value: "", label: allLabel }, ...options];
}

function buildStatusOptions(entries: DebtEntry[], allLabel: string): SelectOption[] {
  const statuses = [...new Set(entries.map((entry) => entry.status).filter(Boolean))].sort();
  return [{ value: "", label: allLabel }, ...statuses.map((value) => ({ value, label: value }))];
}
