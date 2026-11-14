/** @vrooliComponentSource data-display.data-table */
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { adoptionsClient, LocalStatus, type Adoption } from "../../api/adoptions";
import { errorMessage } from "../../lib/errorMessage";
import { CreateAdoptionDialog } from "./CreateAdoptionDialog";
import { AdoptionSuggestions } from "./AdoptionSuggestions";
import { AdoptionsTable, adoptionStatusKey } from "./AdoptionsTable";

const EMPTY_ADOPTIONS: Adoption[] = [];

/**
 * AdoptionsCard renders the adoption registry: every soft-linked
 * library component → target scenario mapping plus its drift status.
 * Surface for req 08 (AD-001..AD-003).
 */
export function AdoptionsCard({
  componentId,
  suggestionsOnly = false,
}: {
  componentId?: string;
  suggestionsOnly?: boolean;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [scenario, setScenario] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const navigate = useNavigate();

  const adoptionsQuery = useQuery({
    queryKey: ["adoptions", { scenario, componentId }],
    queryFn: () => adoptionsClient.listAdoptions({ scenario, componentId, limit: 0 }),
  });

  const refreshMutation = useMutation({
    mutationFn: () => adoptionsClient.refreshAdoptions({}),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["adoptions"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adoptionsClient.deleteAdoption({ id }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["adoptions"] });
    },
  });

  const adoptions = adoptionsQuery.data?.adoptions ?? EMPTY_ADOPTIONS;
  const suggestionsQuery = useQuery({
    queryKey: ["adoptions", "suggestions", scenario],
    queryFn: () => adoptionsClient.suggestAdoptions({ scenario, limit: 8 }),
  });

  const summary = useMemo(() => {
    const acc = { current: 0, behind: 0, modified: 0, unknown: 0 };
    for (const a of adoptions) {
      const k = adoptionStatusKey(a.libraryVersionStatus);
      if (k === "current") acc.current++;
      else if (k === "behind") acc.behind++;
      else if (k === "unknown") acc.unknown++;
      if (a.localStatus === LocalStatus.MODIFIED) acc.modified++;
    }
    return acc;
  }, [adoptions]);

  if (suggestionsOnly) {
    const suggestions = suggestionsQuery.data?.suggestions ?? [];
    return (
      <section
        data-testid="adoption-suggestions"
        className="rounded-panel border border-app-border bg-app-surface p-space-sm"
      >
        <h3 className="text-sm font-semibold text-app-foreground">
          {t(strings.adoptions.suggestions.title)}
        </h3>
        <p className="mt-space-3xs text-xs text-app-muted-foreground">
          {t(strings.adoptions.suggestions.subtitle)}
        </p>
        <AdoptionSuggestions
          suggestions={suggestions}
          loading={suggestionsQuery.isLoading}
          onAdopt={(item) =>
            void navigate(
              `/?action=adopt&assetId=${encodeURIComponent(item.componentId)}&targetScenario=${encodeURIComponent(item.scenario)}`,
            )
          }
          compact
        />
      </section>
    );
  }

  return (
    <section
      data-testid={selectors.adoptions.card}
      aria-label={t(strings.adoptions.title)}
      className="mt-space-sm rounded-xl border border-app-border bg-app-surface p-space-sm backdrop-blur-sm"
    >
      <div className="flex items-center justify-between gap-space-xs">
        <h2 className="text-sm font-medium text-app-muted-foreground">
          {t(strings.adoptions.title)}
        </h2>
        <div className="flex items-center gap-space-2xs">
          <Button
            data-testid={selectors.adoptions.createButton}
            variant="secondary"
            onClick={() => void navigate("/?action=adopt")}
          >
            {t(strings.adoptions.createAction)}
          </Button>
          <Button
            data-testid={selectors.adoptions.refreshButton}
            onClick={() => refreshMutation.mutate()}
            disabled={refreshMutation.isPending}
          >
            {refreshMutation.isPending
              ? t(strings.adoptions.refreshing)
              : t(strings.adoptions.refreshAction)}
          </Button>
        </div>
      </div>

      <div className="mt-space-xs grid grid-cols-1 gap-space-2xs sm:grid-cols-2">
        <label className="block text-xs text-app-muted-foreground">
          {t(strings.adoptions.scenarioFilterLabel)}
          <Input
            data-testid={selectors.adoptions.scenarioFilter}
            value={scenario}
            onChange={(e) => setScenario(e.target.value)}
            placeholder={t(strings.adoptions.scenarioFilterPlaceholder)}
            className="mt-space-3xs"
          />
        </label>
      </div>

      {adoptionsQuery.isLoading && (
        <p data-testid={selectors.adoptions.loading} className="mt-space-xs text-app-foreground">
          {t(strings.adoptions.loading)}
        </p>
      )}
      {adoptionsQuery.error && (
        <p data-testid={selectors.adoptions.error} className="mt-space-xs text-app-danger">
          {errorMessage(adoptionsQuery.error, t)}
        </p>
      )}
      {refreshMutation.error && (
        <p data-testid={selectors.adoptions.error} className="mt-space-xs text-app-danger">
          {errorMessage(refreshMutation.error, t)}
        </p>
      )}
      {deleteMutation.error && (
        <p data-testid={selectors.adoptions.error} className="mt-space-xs text-app-danger">
          {errorMessage(deleteMutation.error, t)}
        </p>
      )}

      {adoptions.length === 0 && !adoptionsQuery.isLoading && !adoptionsQuery.error && (
        <div data-testid={selectors.adoptions.empty} className="mt-space-xs">
          <EmptyState
            title={t(strings.adoptions.empty)}
            action={
              <div className="flex flex-wrap gap-space-2xs">
                <Button variant="secondary" onClick={() => void navigate("/?action=adopt")}>
                  {t(strings.adoptions.createAction)}
                </Button>
                <Button
                  onClick={() => refreshMutation.mutate()}
                  disabled={refreshMutation.isPending}
                >
                  {refreshMutation.isPending
                    ? t(strings.adoptions.refreshing)
                    : t(strings.adoptions.refreshAction)}
                </Button>
              </div>
            }
          />
        </div>
      )}

      {adoptions.length > 0 && (
        <>
          <p
            data-testid={selectors.adoptions.summary}
            className="mt-space-xs text-xs text-app-muted-foreground"
          >
            {t(strings.adoptions.summary, {
              count: adoptions.length,
              current: summary.current,
              behind: summary.behind,
              modified: summary.modified,
              unknown: summary.unknown,
            })}
          </p>
          <div data-testid={selectors.adoptions.list} className="mt-space-2xs">
            <AdoptionsTable
              adoptions={adoptions}
              onDelete={(id) => deleteMutation.mutate(id)}
              deletePending={deleteMutation.isPending}
            />
          </div>
        </>
      )}

      <section className="rounded-panel border border-app-border bg-app-surface p-space-sm">
        <h2 className="text-sm font-semibold text-app-foreground">
          {t(strings.adoptions.suggestions.title)}
        </h2>
        <p className="mt-space-3xs text-xs text-app-muted-foreground">
          {t(strings.adoptions.suggestions.subtitle)}
        </p>
        <AdoptionSuggestions
          suggestions={suggestionsQuery.data?.suggestions ?? []}
          loading={suggestionsQuery.isLoading}
          onAdopt={(item) =>
            void navigate(
              `/?action=adopt&assetId=${encodeURIComponent(item.componentId)}&targetScenario=${encodeURIComponent(item.scenario)}`,
            )
          }
        />
      </section>
      <details className="mt-space-xs text-xs text-app-muted-foreground">
        <summary className="cursor-pointer">Advanced local re-link</summary>
        <p className="mt-space-2xs">
          Use this only to record an existing local copy; normal adoption starts the guided
          adopt-assist workflow.
        </p>
        <Button
          variant="secondary"
          size="sm"
          className="mt-space-2xs"
          onClick={() => setCreateOpen(true)}
        >
          Open local re-link
        </Button>
      </details>
      <CreateAdoptionDialog open={createOpen} initial={null} onClose={() => setCreateOpen(false)} />
    </section>
  );
}
