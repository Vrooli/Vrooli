import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Flag, Layers3, Target } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { getProgress, listMilestones, listSectors } from "../../api/techTree";

const tiers = ["foundation", "operational", "analytics", "integration", "digital_twin"];

export function RoadmapPage() {
  const { t } = useTranslation();
  const sectorsQuery = useQuery({ queryKey: ["roadmap-sectors"], queryFn: listSectors });
  const milestonesQuery = useQuery({ queryKey: ["roadmap-milestones"], queryFn: listMilestones });
  const progressQuery = useQuery({ queryKey: ["roadmap-progress"], queryFn: getProgress });

  const sectors = sectorsQuery.data?.sectors ?? [];
  const milestones = milestonesQuery.data?.milestones ?? [];
  const buckets = progressQuery.data?.buckets ?? [];

  return (
    <section data-testid={selectors.pages.roadmap} className="flex flex-col gap-5">
      <div>
        <p className="text-sm font-medium uppercase text-app-muted-foreground">{t(strings.roadmap.eyebrow)}</p>
        <h2 className="text-3xl font-semibold">{t(strings.roadmap.title)}</h2>
        <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.roadmap.description)}
        </p>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <SummaryCard icon={<Layers3 aria-hidden className="h-5 w-5" />} label={t(strings.roadmap.metrics.sectors)} value={sectors.length} />
        <SummaryCard icon={<Target aria-hidden className="h-5 w-5" />} label={t(strings.roadmap.metrics.milestones)} value={milestones.length} />
        <SummaryCard icon={<Flag aria-hidden className="h-5 w-5" />} label={t(strings.roadmap.metrics.buckets)} value={buckets.length} />
      </div>

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="rounded-lg border border-app-border bg-app-surface p-4">
          <h3 className="text-lg font-semibold">{t(strings.roadmap.tiers.title)}</h3>
          {progressQuery.isLoading && <p className="mt-3 text-sm text-app-muted-foreground">{t(strings.roadmap.states.loadingProgress)}</p>}
          <div className="mt-4 overflow-x-auto">
            <table className="w-full min-w-[680px] text-left text-sm">
              <thead className="text-xs uppercase text-app-muted-foreground">
                <tr>
                  <th className="py-2">{t(strings.roadmap.tiers.sectorColumn)}</th>
                  {tiers.map((tier) => <th key={tier} className="px-2 py-2">{tier.replace("_", " ")}</th>)}
                </tr>
              </thead>
              <tbody>
                {(sectors.length ? sectors : [{ slug: "unassigned", name: t(strings.roadmap.fallbacks.unassigned), description: "" }]).map((sector) => (
                  <tr key={sector.slug} className="border-t border-app-border">
                    <td className="py-3 pr-3 font-medium">{sector.name || sector.slug}</td>
                    {tiers.map((tier) => {
                      const bucket = buckets.find((entry) => entry.sector === sector.slug && entry.tier === tier);
                      return (
                        <td key={`${sector.slug}-${tier}`} className="px-2 py-3">
                          <ProgressPill planned={bucket?.planned ?? 0} live={bucket?.live ?? 0} stable={bucket?.stable ?? 0} />
                        </td>
                      );
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="space-y-4">
          <div className="rounded-lg border border-app-border bg-app-surface p-4">
            <h3 className="text-lg font-semibold">{t(strings.roadmap.sectors.title)}</h3>
            <div className="mt-3 space-y-3">
              {sectors.map((sector) => (
                <div key={sector.slug} className="rounded-md border border-app-border p-3">
                  <p className="font-medium">{sector.name || sector.slug}</p>
                  <p className="mt-1 text-sm text-app-muted-foreground">{sector.description || t(strings.roadmap.fallbacks.noDescription)}</p>
                </div>
              ))}
              {!sectorsQuery.isLoading && sectors.length === 0 && <p className="text-sm text-app-muted-foreground">{t(strings.roadmap.states.noSectors)}</p>}
            </div>
          </div>

          <div className="rounded-lg border border-app-border bg-app-surface p-4">
            <h3 className="text-lg font-semibold">{t(strings.roadmap.milestones.title)}</h3>
            <div className="mt-3 space-y-3">
              {milestones.map((milestone) => (
                <div key={milestone.id} className="rounded-md border border-app-border p-3">
                  <p className="font-medium">{milestone.name || milestone.id}</p>
                  <p className="mt-1 text-sm text-app-muted-foreground">{milestone.description || t(strings.roadmap.fallbacks.noDescription)}</p>
                  <p className="mt-2 text-xs text-app-muted-foreground">
                    {t(strings.roadmap.milestones.requiredScenarios, { count: milestone.requiredScenarios.length })}
                  </p>
                </div>
              ))}
              {!milestonesQuery.isLoading && milestones.length === 0 && <p className="text-sm text-app-muted-foreground">{t(strings.roadmap.states.noMilestones)}</p>}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function SummaryCard({ icon, label, value }: { icon: ReactNode; label: string; value: number }) {
  return (
    <div className="rounded-lg border border-app-border bg-app-surface p-4">
      <div className="flex items-center gap-3">
        <span className="rounded-md bg-app-primary/10 p-2 text-app-primary">{icon}</span>
        <div>
          <p className="text-xs uppercase text-app-muted-foreground">{label}</p>
          <p className="text-2xl font-semibold">{value}</p>
        </div>
      </div>
    </div>
  );
}

function ProgressPill({ planned, live, stable }: { planned: number; live: number; stable: number }) {
  const { t } = useTranslation();
  const total = planned + live + stable;
  return (
    <div className="rounded-md bg-black/10 px-2 py-1 text-xs">
      <span className="font-medium">{total}</span>
      <span className="ml-1 text-app-muted-foreground">
        {t(strings.roadmap.tiers.progressCounts, { planned, live, stable })}
      </span>
    </div>
  );
}
