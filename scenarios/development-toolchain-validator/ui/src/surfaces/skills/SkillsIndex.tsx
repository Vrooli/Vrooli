import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Wrench } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { skillCatalogClient } from "../../api/skillCatalog";
import { errorMessage } from "../../lib/errorMessage";
import { ROUTES } from "../../routes.generated";
import { Card } from "../../shared/ui/primitives/Card";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { LoadingSkeleton } from "../../shared/ui/composites/LoadingSkeleton";

/**
 * Skills index surface — lists every skill known to the catalog with
 * version + content-hash. Click a row to drill into per-skill detail.
 */
export function SkillsIndex() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const listQuery = useQuery({
    queryKey: ["skills"],
    queryFn: () => skillCatalogClient.listSkills({}),
  });

  const skills = listQuery.data?.skills ?? [];

  return (
    <section
      data-testid={selectors.skills.surface}
      className="flex flex-col gap-4"
    >
      <PanelHeader
        title={t(strings.skills.title)}
        description={t(strings.skills.subtitle)}
      />

      {listQuery.isLoading ? (
        <LoadingSkeleton
          data-testid={selectors.skills.loading}
          variant="card"
          count={3}
        />
      ) : null}

      {listQuery.error ? (
        <p
          data-testid={selectors.skills.error}
          className="text-sm text-status-failure"
        >
          {errorMessage(listQuery.error, t)}
        </p>
      ) : null}

      {!listQuery.isLoading && skills.length === 0 && !listQuery.error ? (
        <EmptyState
          testId={selectors.skills.empty}
          icon={<Wrench className="h-8 w-8" aria-hidden />}
          title={t(strings.skills.empty)}
          description={t(strings.skills.emptyDescription)}
        />
      ) : null}

      {skills.length > 0 ? (
        <ul data-testid={selectors.skills.list} className="flex flex-col gap-2">
          {skills.map((s) => (
            <li key={s.id}>
              <button
                type="button"
                data-testid={selectors.skills.row}
                onClick={() => void navigate(ROUTES.skillDetail(s.id))}
                className="w-full text-left"
              >
                <Card
                  surface="raised"
                  className="transition-colors hover:border-app-accent"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-app-foreground">
                        {s.id}
                      </p>
                      <p className="truncate text-xs text-app-muted-foreground">
                        {t(strings.skills.rowLabel, {
                          id: s.id,
                          version: s.version,
                        })}
                      </p>
                      <p className="mt-1 truncate font-mono text-[11px] text-app-muted-foreground">
                        {s.contentHash}
                      </p>
                    </div>
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
