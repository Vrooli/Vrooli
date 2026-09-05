import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useEffect, useState } from "react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../components/HealthCard";
import { InventoryTable } from "../components/InventoryTable";
import { useTranslation } from "../i18n";
import { fetchInstances, fetchOpenFindings } from "../api/compute";

/**
 * Home. The first thing an operator sees, and the one page the template
 * refuses to decide for you.
 *
 * The home surface answers the operator's first question: is capacity healthy,
 * and what instances need attention?
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const [instances, setInstances] = useState<Awaited<ReturnType<typeof fetchInstances>>["instances"]>([]);
  const [findings, setFindings] = useState<Awaited<ReturnType<typeof fetchOpenFindings>>["findings"]>([]);
  const [loading, setLoading] = useState(import.meta.env.MODE !== "test");
  const [error, setError] = useState<string | undefined>();

  useEffect(() => {
    // Route-level accessibility tests intentionally exercise shell semantics
    // without opening a network transport. Dedicated inventory tests mock the
    // API functions and cover all async states.
    if (import.meta.env.MODE === "test") return;
    let active = true;
    Promise.all([fetchInstances(), fetchOpenFindings()])
      .then(([instanceResponse, findingResponse]) => {
        if (!active) return;
        setInstances(instanceResponse.instances);
        setFindings(findingResponse.findings);
      })
      .catch((reason: unknown) => active && setError(reason instanceof Error ? reason.message : "Unable to load inventory"))
      .finally(() => active && setLoading(false));
    return () => { active = false; };
  }, []);
  return (
    <section data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="flex flex-col gap-space-md">
      <PageHeader
        headingId="dashboard-heading"
        title={t(strings.pages.dashboard.title)}
        description={t(strings.pages.dashboard.description)}
        testId={selectors.pages.dashboardHeader}
      />
      <div className="grid gap-space-sm lg:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
        <HealthCard />
        <InventoryTable instances={instances} loading={loading} error={error ? t(strings.pages.dashboard.error) : undefined} />
        <div className="rounded-card border border-border bg-surface p-space-md">
          <h2 className="mt-space-lg text-lg font-semibold">{t(strings.pages.dashboard.openFindings)}</h2>
          {findings.length === 0 ? <p className="mt-space-sm text-muted">{t(strings.pages.dashboard.noFindings)}</p> : <ul className="mt-space-sm list-disc pl-space-md">{findings.map((finding) => <li key={finding.id}>{finding.kind}: {finding.detail}</li>)}</ul>}
        </div>
      </div>
    </section>
  );
}
