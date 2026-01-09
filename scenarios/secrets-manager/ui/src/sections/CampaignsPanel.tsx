import { useMemo, useState } from "react";
import { Search, ListChecks, ChevronDown, ChevronRight } from "lucide-react";
import { Button } from "../components/ui/button";
import { HelpDialog } from "../components/ui/HelpDialog";
import type { CampaignSummary } from "../lib/api";

interface CampaignsPanelProps {
  campaigns: CampaignSummary[];
  isLoading: boolean;
  search: string;
  onSearchChange: (value: string) => void;
  selectedScenario?: string;
  onSelectScenario: (name: string) => void;
  defaultBlockedTiers: number;
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
}

export const CampaignsPanel = ({
  campaigns,
  isLoading,
  search,
  onSearchChange,
  selectedScenario,
  onSelectScenario,
  defaultBlockedTiers,
  isCollapsed = false,
  onToggleCollapse
}: CampaignsPanelProps) => {
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");

  const filtered = useMemo(() => {
    const term = search.toLowerCase();
    return [...campaigns]
      .filter((campaign) => campaign.scenario.toLowerCase().includes(term))
      .sort((a, b) => (sortDir === "asc" ? a.scenario.localeCompare(b.scenario) : b.scenario.localeCompare(a.scenario)));
  }, [campaigns, search, sortDir]);

  const aggregate = useMemo(
    () =>
      filtered.reduce(
        (acc, campaign) => {
          const summary = campaign.summary;
          acc.count += 1;
          acc.covered += summary?.strategized_secrets ?? 0;
          acc.total += summary?.total_secrets ?? 0;
          const blockers = summary?.requires_action ?? (campaign.blockers > 0 ? campaign.blockers : defaultBlockedTiers);
          acc.blockers += blockers;
          return acc;
        },
        { count: 0, covered: 0, total: 0, blockers: 0 }
      ),
    [filtered, defaultBlockedTiers]
  );

  return (
    <section id="anchor-campaigns" className="rounded-3xl border border-white/10 bg-white/5 overflow-hidden">
      {/* Collapsible header */}
      <div
        className={`flex items-center gap-3 px-6 py-4 ${onToggleCollapse ? "cursor-pointer hover:bg-white/5" : ""} ${isCollapsed ? "" : "border-b border-white/10"}`}
        onClick={onToggleCollapse}
      >
        {onToggleCollapse && (
          <div className="flex items-center justify-center">
            {isCollapsed ? (
              <ChevronRight className="h-5 w-5 text-white/40" />
            ) : (
              <ChevronDown className="h-5 w-5 text-white/40" />
            )}
          </div>
        )}
        <div className="rounded-full border border-emerald-400/40 bg-emerald-500/10 p-2">
          <ListChecks className="h-5 w-5 text-emerald-200" />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Campaigns</p>
            <HelpDialog title="What are Campaigns?">
              <p>
                A <strong className="text-white">campaign</strong> represents a deployment-readiness workflow for a specific scenario.
                Each campaign tracks progress toward making a scenario ready for deployment across different tiers.
              </p>
              <ul className="mt-3 space-y-2">
                <li>
                  <strong className="text-white">Status:</strong> Shows how many deployment tiers are currently blocked
                  (e.g., "2 tiers blocked" means 2 tiers lack required secret strategies)
                </li>
                <li>
                  <strong className="text-white">Coverage:</strong> Ratio of secrets with defined strategies vs total required secrets
                </li>
                <li>
                  <strong className="text-white">Requires action:</strong> Number of secrets that still need attention before deployment
                </li>
                <li>
                  <strong className="text-white">Next action:</strong> Suggested next step to unblock the scenario
                </li>
              </ul>
              <p className="mt-3">
                Select a campaign to view its readiness details and generate deployment manifests.
              </p>
            </HelpDialog>
          </div>
          <p className="text-lg font-semibold text-white">Deployment readiness campaigns</p>
          {isCollapsed && selectedScenario && (
            <p className="text-sm text-emerald-300/80">Selected: {selectedScenario}</p>
          )}
          {!isCollapsed && (
            <p className="text-sm text-white/60">Search, sort, and resume work per scenario.</p>
          )}
        </div>
        {/* Summary badges visible when collapsed */}
        {isCollapsed && (
          <div className="flex gap-2">
            <div className="rounded-xl border border-white/10 bg-black/30 px-2 py-1 text-xs text-white/70">
              {aggregate.count} campaigns
            </div>
            {aggregate.blockers > 0 && (
              <div className="rounded-xl border border-amber-400/30 bg-amber-500/10 px-2 py-1 text-xs text-amber-100">
                {aggregate.blockers} blockers
              </div>
            )}
          </div>
        )}
      </div>

      {/* Collapsible content */}
      {!isCollapsed && (
        <div className="p-6 pt-4">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
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

          <div className="mt-3 flex flex-wrap gap-3 text-sm text-white/70">
            <div className="rounded-xl border border-white/10 bg-black/30 px-3 py-2">
              Campaigns: <span className="font-semibold text-white">{aggregate.count}</span>
            </div>
            <div className="rounded-xl border border-emerald-400/30 bg-emerald-500/10 px-3 py-2">
              Coverage:{" "}
              <span className="font-semibold text-white">
                {aggregate.total > 0 ? `${aggregate.covered}/${aggregate.total}` : "—"}
              </span>
            </div>
            <div className="rounded-xl border border-amber-400/30 bg-amber-500/10 px-3 py-2">
              Blockers: <span className="font-semibold text-amber-100">{aggregate.blockers}</span>
            </div>
          </div>

          <div className="mt-4 max-h-[400px] overflow-y-auto rounded-2xl border border-white/10">
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
                    <td colSpan={6} className="px-3 py-4 text-center text-sm text-white/60">
                      Loading campaigns...
                    </td>
                  </tr>
                ) : filtered.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-3 py-4 text-center text-sm text-white/60">
                      No campaigns match "{search}".
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
        </div>
      )}
    </section>
  );
};
