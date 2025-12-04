import { useMemo, useState } from "react";
import { Search, ListChecks } from "lucide-react";
import { Button } from "../components/ui/button";
import type { CampaignSummary } from "../lib/api";

interface CampaignsPanelProps {
  campaigns: CampaignSummary[];
  isLoading: boolean;
  search: string;
  onSearchChange: (value: string) => void;
  selectedScenario?: string;
  onSelectScenario: (name: string) => void;
  defaultBlockedTiers: number;
}

export const CampaignsPanel = ({
  campaigns,
  isLoading,
  search,
  onSearchChange,
  selectedScenario,
  onSelectScenario,
  defaultBlockedTiers
}: CampaignsPanelProps) => {
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");

  const filtered = useMemo(() => {
    const term = search.toLowerCase();
    return [...campaigns]
      .filter((campaign) => campaign.scenario.toLowerCase().includes(term))
      .sort((a, b) => (sortDir === "asc" ? a.scenario.localeCompare(b.scenario) : b.scenario.localeCompare(a.scenario)));
  }, [campaigns, search, sortDir]);

  return (
    <section className="rounded-3xl border border-white/10 bg-white/5 p-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="flex items-center gap-3">
          <div className="rounded-full border border-emerald-400/40 bg-emerald-500/10 p-2">
            <ListChecks className="h-5 w-5 text-emerald-200" />
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Campaigns</p>
            <p className="text-lg font-semibold text-white">Deployment readiness campaigns</p>
            <p className="text-sm text-white/60">Search, sort, and resume work per scenario.</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative w-56">
            <input
              type="text"
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder="Search scenarios"
              className="w-full rounded-2xl border border-white/10 bg-black/30 px-3 py-2 pr-10 text-sm text-white placeholder:text-white/40"
            />
            <Search className="pointer-events-none absolute right-3 top-2.5 h-4 w-4 text-white/40" />
          </div>
          <Button variant="outline" size="sm" onClick={() => setSortDir((dir) => (dir === "asc" ? "desc" : "asc"))}>
            Sort: {sortDir === "asc" ? "A→Z" : "Z→A"}
          </Button>
        </div>
      </div>

      <div className="mt-4 overflow-hidden rounded-2xl border border-white/10">
        <table className="min-w-full divide-y divide-white/10 text-sm">
          <thead className="bg-black/30 text-white/60">
            <tr>
              <th className="px-3 py-2 text-left">Scenario</th>
              <th className="px-3 py-2 text-left">Status</th>
              <th className="px-3 py-2 text-left">Coverage</th>
              <th className="px-3 py-2 text-left">Requires action</th>
              <th className="px-3 py-2 text-left">Next action</th>
              <th className="px-3 py-2 text-left">Select</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/10">
            {isLoading ? (
              <tr>
                <td colSpan={4} className="px-3 py-4 text-center text-sm text-white/60">
                  Loading campaigns...
                </td>
              </tr>
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-3 py-4 text-center text-sm text-white/60">
                  No campaigns match “{search}”.
                </td>
              </tr>
            ) : (
              filtered.map((campaign) => {
                const isActive = campaign.scenario === selectedScenario;
                const blockers = campaign.blockers > 0 ? campaign.blockers : defaultBlockedTiers;
                const coverage =
                  campaign.summary && campaign.summary.total_secrets > 0
                    ? `${campaign.summary.strategized_secrets}/${campaign.summary.total_secrets}`
                    : "—";
                const requires = campaign.summary ? campaign.summary.requires_action : blockers;
                return (
                  <tr key={campaign.id} className="bg-black/20">
                    <td className="px-3 py-2 font-semibold text-white">{campaign.scenario}</td>
                    <td className="px-3 py-2">
                      <span
                        className={`rounded-full border px-2 py-0.5 text-[11px] uppercase tracking-[0.2em] ${
                          blockers > 0 ? "border-amber-400/50 text-amber-100" : "border-emerald-400/50 text-emerald-100"
                        }`}
                      >
                        {blockers > 0 ? `${blockers} tier${blockers === 1 ? "" : "s"} blocked` : "Ready"}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-sm text-white/80">{coverage}</td>
                    <td className="px-3 py-2 text-sm text-amber-100">{requires ?? "—"}</td>
                    <td className="px-3 py-2 text-xs text-white/70">
                      {campaign.next_action || (blockers > 0 ? "Define strategies for blocked tiers" : "Generate manifest")}
                    </td>
                    <td className="px-3 py-2">
                      <Button
                        size="sm"
                        variant={isActive ? "secondary" : "outline"}
                        onClick={() => onSelectScenario(campaign.scenario)}
                      >
                        {isActive ? "Selected" : "Open"}
                      </Button>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
};
