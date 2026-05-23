import { Link, Navigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath, encodeScenarioPath } from "../hooks/useScenarioPath";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { manifestClient } from "../api/manifest";
import { useQuery } from "@tanstack/react-query";

export function TargetApplyPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();

  const domains = useQuery({
    queryKey: ["manifest", "domains", scenario ?? ""],
    queryFn: () => manifestClient.listDomains({ scenario: scenario ?? "" }),
    enabled: scenario !== null,
  });

  if (scenario === null) return <Navigate to="/" replace />;

  return (
    <section
      data-testid={selectors.pages.targetApply}
      aria-labelledby="target-apply-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-apply-heading" className="text-xl font-semibold">
          {t(strings.pages.targetApply.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.targetApply.description)}
        </p>
      </header>

      {domains.isPending ? (
        <LoadingState label={t(strings.pages.targetApply.loading)} />
      ) : domains.isError ? (
        <ErrorState
          title={t(strings.pages.targetApply.errorTitle)}
          message={domains.error instanceof Error ? domains.error.message : String(domains.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void domains.refetch();
          }}
        />
      ) : domains.data.domains.length === 0 ? (
        <EmptyState title={t(strings.pages.targetApply.selectDomainPrompt)} />
      ) : (
        <ul className="flex flex-col gap-2">
          {domains.data.domains.map((d) => (
            <li key={d.name}>
              <Link
                to={`/targets/${encodeScenarioPath(scenario)}/apply/${encodeURIComponent(d.name)}`}
                data-testid={selectors.features.apply.plan.domainLink({ key: d.name })}
                className="block rounded-panel border border-app-border bg-app-surface p-3 text-sm hover:bg-app-surface-muted"
              >
                <span className="font-semibold">{d.name}</span>
                <span className="ml-2 text-app-muted-foreground">
                  {t(strings.pages.targetManifest.columns.paths)}: {d.paths.length}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
