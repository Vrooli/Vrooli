import { useSearchParams } from "react-router-dom";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { ManualImport } from "./ManualImport";
import { CoverageChip, UsageCoverage } from "./UsageCoverage";
import { UsageDetails } from "./UsageDetails";
import { UsageEvidence } from "./UsageEvidence";
import { UsageFriction } from "./UsageFriction";
import { UsageOverview } from "./UsageOverview";

export const USAGE_TABS = ["overview", "evidence", "friction", "details", "coverage"] as const;
export type UsageTab = (typeof USAGE_TABS)[number];

const TAB_LABELS: Record<UsageTab, string> = { overview: "Overview", evidence: "Evidence", friction: "Friction", details: "Details", coverage: "Coverage" };

function isUsageTab(value: string | null): value is UsageTab {
  return USAGE_TABS.includes(value as UsageTab);
}

/** Usage: what the subscriptions have left, what is using them, the friction that wastes them, and how complete the data is. */
export function UsagePage() {
  const [params, setParams] = useSearchParams();
  const requested = params.get("tab");
  const tab: UsageTab = isUsageTab(requested) ? requested : "overview";
  const selectTab = (next: string) => {
    if (!isUsageTab(next)) return;
    // Each tab owns its own query parameters; switching starts the new tab clean.
    setParams(next === "overview" ? {} : { tab: next }, { replace: true });
  };

  return (
    // The shell's main area does not scroll; each page owns its scroll container.
    <div className="h-full overflow-y-auto overflow-x-hidden p-4 sm:p-6">
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
          evidence: <div className="pt-4"><UsageEvidence /></div>,
          friction: <div className="pt-4"><UsageFriction /></div>,
          details: <div className="pt-4"><UsageDetails /></div>,
          coverage: (
            <div className="flex flex-col gap-6 pt-4">
              <UsageCoverage />
              <ManualImport />
            </div>
          ),
        }}
      />
    </div>
    </div>
  );
}
