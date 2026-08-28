import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

import { fetchTemplateList } from "../api/templateDomain";
import { DetailError, DetailLoading } from "../components/detail/DetailStates";
import { EntityList, EntityRowBody } from "../components/list/EntityList";
import { Select, type SelectOption } from "@vrooli/react-component-library/Select/1.1.0";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { kindLabel } from "../lib/templateLabels";

const TITLE_ID = "template-list-heading";
const KIND_PARAM = "kind";

export function TemplateListPage() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const query = useQuery({ queryKey: ["template-list"] as const, queryFn: fetchTemplateList });

  const kindFilter = params.get(KIND_PARAM) ?? "";
  const setKind = (value: string) => {
    const next = new URLSearchParams(params);
    if (value) {
      next.set(KIND_PARAM, value);
    } else {
      next.delete(KIND_PARAM);
    }
    setParams(next, { replace: true });
  };

  const templates = useMemo(() => query.data ?? [], [query.data]);
  const kindOptions = useMemo(() => buildKindOptions(t(strings.templateList.filters.allKinds)), [t]);
  const filtered = useMemo(
    () => (kindFilter ? templates.filter((template) => String(template.kind) === kindFilter) : templates),
    [templates, kindFilter],
  );

  return (
    <section data-testid={selectors.pages.templateList} aria-labelledby={TITLE_ID} className="flex flex-col gap-4">
      <div className="min-w-0">
        <h2 id={TITLE_ID} className="text-2xl font-semibold">
          {t(strings.pages.templateList.title)}
        </h2>
        <p className="mt-1 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.templateList.description)}
        </p>
      </div>

      {query.isLoading && <DetailLoading testId={selectors.templateList.loading} />}
      {query.isError && <DetailError testId={selectors.templateList.error} />}

      {query.data && (
        <div data-testid={selectors.templateList.root} className="flex flex-col gap-3">
          <label className="flex max-w-xs flex-col gap-1">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.templateList.filters.kind)}
            </span>
            <Select
              data-testid={selectors.templateList.kindFilter}
              aria-label={t(strings.templateList.filters.kind)}
              options={kindOptions}
              value={kindFilter}
              onChange={(event) => setKind(event.target.value)}
            />
          </label>

          <EntityList
            rows={filtered}
            tableTestId={selectors.templateList.table}
            getRowKey={(row) => row.id}
            getRowHref={(row) => `/templates/${encodeURIComponent(row.id)}`}
            getRowTestId={(row) => selectors.templateList.row({ id: row.id })}
            getRowLabel={(row) => `${t(strings.templateList.viewEntry)}: ${row.displayName || row.id}`}
            renderRow={(row) => (
              <EntityRowBody
                primary={row.displayName || row.id}
                secondary={`${kindLabel(row.kind)} · ${row.version}`}
                trailing={
                  <StatusBadge tone={(row.versionLag?.lagCount ?? 0) > 0 ? "warning" : "success"}>{row.status}</StatusBadge>
                }
              />
            )}
            searchText={(row) => `${row.displayName} ${row.id} ${row.version}`}
            searchLabel={t(strings.templateList.search)}
            searchPlaceholder={t(strings.templateList.searchPlaceholder)}
            emptyLabel={t(strings.templateList.empty)}
          />
        </div>
      )}
    </section>
  );
}

function buildKindOptions(allLabel: string): SelectOption[] {
  return [
    { value: "", label: allLabel },
    { value: "1", label: kindLabel(1) },
    { value: "2", label: kindLabel(2) },
    { value: "3", label: kindLabel(3) },
  ];
}
