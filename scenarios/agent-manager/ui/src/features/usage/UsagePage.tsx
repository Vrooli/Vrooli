import { lazy, Suspense, type ReactNode } from "react";
import { useSearchParams } from "react-router-dom";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { ManualImport } from "./ManualImport";
import { UsageBurners } from "./UsageBurners";
import { CoverageChip, UsageCoverage } from "./UsageCoverage";
import { UsageRecommendations } from "./UsageRecommendations";
import { UsageWindows } from "./UsageWindows";

const StatsPage = lazy(async () => ({ default: (await import("../stats")).StatsPage }));

export const USAGE_TABS = ["overview", "coverage", "details"] as const;
export type UsageTab = (typeof USAGE_TABS)[number];

const TAB_LABELS: Record<UsageTab, string> = { overview: "Overview", coverage: "Coverage", details: "Details" };

function isUsageTab(value: string | null): value is UsageTab {
  return USAGE_TABS.includes(value as UsageTab);
}

/** A titled region of the Usage page. The heading stays visible above its content or empty state. */
function UsageSection({ id, title, description, className, children }: { id: string; title: string; description: string; className?: string; children: ReactNode }) {
  const headingId = `usage-${id}-heading`;
  return (
    <section aria-labelledby={headingId} data-testid={`usage-section-${id}`} className={`space-y-3 ${className ?? ""}`}>
      <div>
        <h2 id={headingId} className="text-base font-semibold">{title}</h2>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
      {children}
    </section>
  );
}

/**
 * Overview: the four questions in the approved mockup order. On a phone,
 * recommendations come before "what's using it" (same content, re-sequenced).
 */
function UsageOverview() {
  return (
    <div className="flex flex-col gap-6">
      <UsageSection id="windows" title="Subscription windows" description="Percent used, reset time and projected run-out for each provider limit.">
        <UsageWindows />
      </UsageSection>
      <UsageSection id="recommendations" className="order-2 lg:order-3" title="Recommendations" description="Ranked by estimated weekly quota saved. Each one links to the runs and numbers behind it.">
        <UsageRecommendations />
      </UsageSection>
      <UsageSection id="burners" className="order-3 lg:order-2" title="What's using it" description="Share of each window used, by session, run, profile, workload, model, runner, modality or tool kind.">
        <UsageBurners />
      </UsageSection>
      <UsageSection id="coverage-summary" className="order-4" title="Coverage" description="What Mizar has read compared with the session files on this machine. The numbers above are only as good as this.">
        <UsageCoverage />
      </UsageSection>
    </div>
  );
}

/** Usage replaces Stats as the analytics entry: overview, coverage (manual import for now) and details (the run analytics panels). */
export function UsagePage() {
  const [params, setParams] = useSearchParams();
  const requested = params.get("tab");
  const tab: UsageTab = isUsageTab(requested) ? requested : "overview";
  const selectTab = (next: string) => {
    if (!isUsageTab(next)) return;
    setParams((current) => {
      const updated = new URLSearchParams(current);
      if (next === "overview") updated.delete("tab");
      else updated.set("tab", next);
      return updated;
    }, { replace: true });
  };

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4" data-testid="usage-page">
      <PageHeader title="Usage" description="How much subscription capacity your agents use, what uses it, and what to change." actions={<CoverageChip />} />
      <Tabs
        mode="controlled"
        active={tab}
        onChange={selectTab}
        ariaLabel="Usage sections"
        items={USAGE_TABS.map((id) => ({ id, label: TAB_LABELS[id] }))}
        itemTestId={(id) => `usage-tab-${id}`}
        testId="usage-tabs"
        panels={{
          overview: <div className="pt-4"><UsageOverview /></div>,
          coverage: (
            <div className="flex flex-col gap-6 pt-4">
              <UsageSection id="coverage" title="Capture coverage" description="Indexed transcript files compared with what is on disk, with every uncaptured source named.">
                <UsageCoverage detailed />
              </UsageSection>
              <ManualImport />
            </div>
          ),
          details: (
            <div className="pt-4">
              <Suspense fallback={<p className="text-sm text-muted-foreground">Loading details…</p>}>
                <StatsPage />
              </Suspense>
            </div>
          ),
        }}
      />
    </div>
  );
}
