import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Panel } from "../../components/ui/panel";
import { EmptyState } from "../../components/ui/empty-state";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { listModels, primarySubtag } from "../../services/models";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { FootprintPanel } from "./components/FootprintPanel";
import { ModelFilters } from "./components/ModelFilters";
import { ModelsTable } from "./components/ModelsTable";
import { ResolvePanel } from "./components/ResolvePanel";
import { EMPTY_MODEL_FILTERS, isUnfiltered, type ModelFiltersState } from "./types";

export function ModelsPage() {
  const { t } = useTranslation();
  const [filters, setFilters] = useState<ModelFiltersState>(EMPTY_MODEL_FILTERS);

  // Filters are part of the query key and forwarded to ListModelsRequest so
  // filtering is always proven against the API contract.
  const models = useQuery({
    queryKey: ["models", filters],
    queryFn: () =>
      listModels({
        capability: filters.capability,
        language: primarySubtag(filters.language),
        commercialOnly: filters.commercialOnly,
        onlyReady: filters.onlyReady,
      }),
  });

  return (
    <div className="flex max-w-6xl flex-col gap-4 md:gap-6">
      <PageHeader
        title={t(strings.models.title)}
        description={t(strings.models.description)}
      />

      <ModelFilters value={filters} onChange={setFilters} />

      {/* Footprint is only meaningful for the full registry; a filtered view
          would look like a capability had no default. */}
      {models.data?.ok && isUnfiltered(filters) ? (
        <FootprintPanel models={models.data.data} />
      ) : null}

      <Panel
        title={t(strings.models.registryTitle)}
        bodyless
        aria-label={t(strings.models.registryTitle)}
      >
        {models.isLoading ? (
          <div className="p-4">
            <LoadingRows rows={6} label={t(strings.common.loading)} />
          </div>
        ) : models.data && !models.data.ok ? (
          <div className="p-4">
            <ApiErrorState
              error={models.data.error}
              onRetry={() => void models.refetch()}
            />
          </div>
        ) : models.data?.ok && models.data.data.length === 0 ? (
          <div className="p-4">
            <EmptyState title={t(strings.models.empty)} />
          </div>
        ) : models.data?.ok ? (
          <ModelsTable models={models.data.data} />
        ) : null}
      </Panel>

      <ResolvePanel />
    </div>
  );
}
