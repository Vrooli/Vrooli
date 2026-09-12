import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { FileCog } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { manifestClient, ConvergenceTarget } from "../../api/manifest";
import { errorMessage } from "../../lib/errorMessage";
import { ROUTES } from "../../routes.generated";
import { Card } from "../../shared/ui/primitives/Card";
import { Badge } from "../../shared/ui/primitives/Badge";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { LoadingSkeleton } from "../../shared/ui/composites/LoadingSkeleton";

/**
 * Manifests index — tabular view of every (skill, golden) manifest.
 * Rows link to the manifest editor.
 */
export function ManifestsIndex() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const listQuery = useQuery({
    queryKey: ["manifests"],
    queryFn: () => manifestClient.listManifests({}),
  });

  const manifests = listQuery.data?.manifests ?? [];

  return (
    <section
      data-testid={selectors.manifests.surface}
      className="flex flex-col gap-4"
    >
      <PanelHeader
        title={t(strings.manifests.title)}
        description={t(strings.manifests.subtitle)}
      />

      {listQuery.isLoading ? (
        <LoadingSkeleton
          data-testid={selectors.manifests.loading}
          variant="card"
          count={3}
        />
      ) : null}

      {listQuery.error ? (
        <p
          data-testid={selectors.manifests.error}
          className="text-sm text-status-failure"
        >
          {errorMessage(listQuery.error, t)}
        </p>
      ) : null}

      {!listQuery.isLoading && manifests.length === 0 && !listQuery.error ? (
        <EmptyState
          testId={selectors.manifests.empty}
          icon={<FileCog className="h-8 w-8" aria-hidden />}
          title={t(strings.manifests.empty)}
          description={t(strings.manifests.emptyDescription)}
        />
      ) : null}

      {manifests.length > 0 ? (
        <ul
          data-testid={selectors.manifests.list}
          className="flex flex-col gap-2"
        >
          {manifests.map((m) => (
            <li key={`${m.skillId}-${m.goldenSlug}`}>
              <button
                type="button"
                data-testid={selectors.manifests.row}
                onClick={() =>
                  void navigate(ROUTES.manifestEditor(m.skillId, m.goldenSlug))
                }
                className="w-full text-left"
              >
                <Card
                  surface="raised"
                  className="transition-colors hover:border-app-accent"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-app-foreground">
                        {t(strings.manifests.rowLabel, {
                          skillId: m.skillId,
                          goldenSlug: m.goldenSlug,
                        })}
                      </p>
                      <p className="truncate text-xs text-app-muted-foreground">
                        {t(strings.manifests.pinningLabel, {
                          template: m.templateVersionPinned || "—",
                          skill: m.skillVersionPinned || "—",
                        })}
                      </p>
                    </div>
                    <Badge variant="neutral">
                      {m.convergenceTarget === ConvergenceTarget.EMPTY_DIFF
                        ? t(strings.manifests.convergenceEmptyDiff)
                        : t(strings.manifests.convergenceNone)}
                    </Badge>
                  </div>
                </Card>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}
