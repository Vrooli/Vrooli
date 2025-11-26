import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { ArrowUpDown, AlertCircle, CheckCircle2, FileWarning, Activity, TrendingUp, HelpCircle, FolderOpen, Info, Terminal, Sparkles } from "lucide-react";
import { useState } from "react";
import { fetchScenarioStats, type ScenarioStats } from "../lib/api";
import { PageHeader } from "../components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Spinner } from "../components/ui/spinner";
import { Alert, AlertDescription } from "../components/ui/alert";
import { cn } from "../lib/utils";

type SortField = "scenario" | "light_issues" | "ai_issues" | "long_files" | "visit_percent" | "campaign_status";

export default function Dashboard() {
  const navigate = useNavigate();
  const [sortField, setSortField] = useState<SortField>("light_issues");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");

  const { data: scenarios = [], isLoading, error } = useQuery({
    queryKey: ["scenarios"],
    queryFn: async () => {
      console.log('[Dashboard] Fetching scenario stats...');
      const result = await fetchScenarioStats();
      console.log('[Dashboard] Fetched scenarios:', result.length);
      return result;
    },
  });

  console.log('[Dashboard] Render - isLoading:', isLoading, 'error:', error, 'scenarios:', scenarios.length);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDirection("desc");
    }
  };

  const sortedScenarios = [...scenarios].sort((a, b) => {
    const aValue = a[sortField];
    const bValue = b[sortField];
    const multiplier = sortDirection === "asc" ? 1 : -1;

    if (typeof aValue === "string" && typeof bValue === "string") {
      return multiplier * aValue.localeCompare(bValue);
    }
    return multiplier * ((aValue as number) - (bValue as number));
  });

  const getCampaignBadge = (status: ScenarioStats["campaign_status"]) => {
    const badges: Record<ScenarioStats["campaign_status"], { text: string; variant: "default" | "active" | "success" | "warning" | "error" }> = {
      none: { text: "No Campaign", variant: "default" },
      active: { text: "Active", variant: "active" },
      completed: { text: "Completed", variant: "success" },
      paused: { text: "Paused", variant: "warning" },
      error: { text: "Error", variant: "error" },
    };
    const badge = badges[status];
    return <Badge variant={badge.variant}>{badge.text}</Badge>;
  };

  const getHealthStatus = (scenario: ScenarioStats) => {
    const totalIssues = scenario.light_issues + scenario.ai_issues;
    if (totalIssues === 0 && scenario.long_files === 0) return { label: "Excellent", color: "text-green-500", icon: CheckCircle2 };
    if (totalIssues < 5 && scenario.long_files < 3) return { label: "Good", color: "text-blue-500", icon: CheckCircle2 };
    if (totalIssues < 15 && scenario.long_files < 8) return { label: "Needs Attention", color: "text-yellow-500", icon: AlertCircle };
    return { label: "Critical", color: "text-red-500", icon: FileWarning };
  };

  // Calculate summary stats
  const totalScenarios = scenarios.length;
  const totalIssues = scenarios.reduce((sum, s) => sum + s.light_issues + s.ai_issues, 0);
  const totalLongFiles = scenarios.reduce((sum, s) => sum + s.long_files, 0);
  const activeCampaigns = scenarios.filter(s => s.campaign_status === "active").length;

  if (error) {
    return (
      <div>
        <PageHeader title="Dashboard" description="Monitor code tidiness across all scenarios" />
        <Alert variant="destructive" role="alert" aria-live="assertive">
          <AlertDescription>Failed to load scenarios: {(error as Error).message}</AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div data-testid="dashboard-page">
      <PageHeader
        title="Dashboard"
        description="Monitor code tidiness across all scenarios"
      />

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <Card data-testid="total-scenarios-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Scenarios</CardTitle>
            <Activity className="h-4 w-4 text-slate-400" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${totalScenarios} scenarios being monitored`}>{totalScenarios}</div>
            <p className="text-xs text-slate-400 mt-1">Being monitored</p>
          </CardContent>
        </Card>

        <Card data-testid="total-issues-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Issues</CardTitle>
            <AlertCircle className="h-4 w-4 text-yellow-500" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${totalIssues} total issues across all scenarios`}>{totalIssues}</div>
            <p className="text-xs text-slate-400 mt-1">Across all scenarios</p>
          </CardContent>
        </Card>

        <Card data-testid="long-files-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Long Files</CardTitle>
            <FileWarning className="h-4 w-4 text-red-500" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${totalLongFiles} files exceeding length threshold`}>{totalLongFiles}</div>
            <p className="text-xs text-slate-400 mt-1">Exceeding threshold</p>
          </CardContent>
        </Card>

        <Card data-testid="active-campaigns-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Campaigns</CardTitle>
            <TrendingUp className="h-4 w-4 text-blue-500" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${activeCampaigns} active campaigns improving code health`}>{activeCampaigns}</div>
            <p className="text-xs text-slate-400 mt-1">Improving code health</p>
          </CardContent>
        </Card>
      </div>

      {/* Scenarios Table */}
      <Card>
        <CardHeader>
          <CardTitle>Scenarios</CardTitle>
          <CardDescription>Code health status for all monitored scenarios</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="flex items-center justify-center p-12" role="status" aria-live="polite" aria-label="Loading scenarios">
              <Spinner size="lg" />
            </div>
          ) : sortedScenarios.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 px-4 text-center">
              <FolderOpen className="h-16 w-16 text-slate-500 mb-4" />
              <h3 className="text-xl font-semibold text-slate-200 mb-2">
                No Scenarios Scanned Yet
              </h3>
              <p className="text-slate-400 max-w-md mb-6">
                Tidiness Manager orchestrates code cleanliness through light scanning (lint/type checks) and smart scanning (AI-powered deep analysis). Get started below.
              </p>

              {/* Agent-First Workflow */}
              <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-6 max-w-2xl text-left space-y-4 mb-4">
                <div className="flex items-start gap-3">
                  <Terminal className="h-5 w-5 text-blue-400 mt-0.5 flex-shrink-0" />
                  <div className="flex-1">
                    <p className="text-sm font-semibold text-blue-200 mb-3">Recommended: Agent CLI Workflow</p>
                    <div className="space-y-3">
                      <div>
                        <p className="text-xs text-slate-400 mb-1">1. Run a light scan (fast, cheap):</p>
                        <code className="text-xs bg-slate-950 px-3 py-2 rounded block text-slate-300 font-mono">
                          tidiness-manager scan &lt;scenario-name&gt; --type light
                        </code>
                      </div>
                      <div>
                        <p className="text-xs text-slate-400 mb-1">2. Query top issues for refactoring:</p>
                        <code className="text-xs bg-slate-950 px-3 py-2 rounded block text-slate-300 font-mono">
                          tidiness-manager issues &lt;scenario-name&gt; --category lint --limit 10
                        </code>
                      </div>
                      <div>
                        <p className="text-xs text-slate-400 mb-1">3. (Optional) Start systematic campaign:</p>
                        <code className="text-xs bg-slate-950 px-3 py-2 rounded block text-slate-300 font-mono">
                          tidiness-manager campaigns start &lt;scenario-name&gt; --max-sessions 10
                        </code>
                      </div>
                    </div>
                    <p className="text-xs text-slate-500 mt-3 pt-3 border-t border-blue-500/20">
                      <span className="font-medium">Why CLI-first?</span> Agents get structured JSON output, avoid API rate limits, and can batch operations. The UI reflects all CLI actions automatically.
                    </p>
                  </div>
                </div>
              </div>

              {/* Human-Friendly Alternative */}
              <div className="bg-slate-900/50 border border-slate-800 rounded-lg p-5 max-w-2xl text-left">
                <div className="flex items-start gap-3">
                  <Info className="h-5 w-5 text-slate-400 mt-0.5 flex-shrink-0" />
                  <div className="flex-1">
                    <p className="text-sm font-semibold text-slate-300 mb-2">For Humans:</p>
                    <p className="text-xs text-slate-400 leading-relaxed">
                      Navigate to a scenario's detail page to trigger scans manually, or visit the <a href="/campaigns" className="text-blue-400 hover:text-blue-300 underline">Campaigns tab</a> to configure automated analysis. Results appear on this dashboard as they complete.
                    </p>
                  </div>
                </div>
              </div>

              <div className="mt-6 pt-4 border-t border-slate-800 max-w-2xl">
                <p className="text-xs text-slate-500 flex items-center justify-center gap-2">
                  <Sparkles className="h-3.5 w-3.5" />
                  <span>Tip: Use visited-tracker integration to ensure comprehensive coverage without redundant work across sessions</span>
                </p>
              </div>
            </div>
          ) : (
            <div className="overflow-x-auto -mx-4 sm:mx-0">
              <table className="w-full min-w-[640px]" data-testid="scenario-table" aria-label="Scenarios tidiness overview table">
                <thead className="border-b border-white/10">
                  <tr>
                    <th
                      className="text-left p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                      onClick={() => handleSort("scenario")}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("scenario"); } }}
                      tabIndex={0}
                      aria-label={`Sort by scenario name, currently ${sortField === "scenario" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                      data-testid="scenario-table-header-name"
                      title="Sort by scenario name"
                    >
                      <div className="flex items-center gap-2">
                        Scenario
                        <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                      </div>
                    </th>
                    <th className="text-left p-2 sm:p-4 font-medium text-xs sm:text-sm text-slate-400" title="Overall code health status: Excellent (no issues), Good (<5 issues, <3 long files), Needs Attention (<15 issues, <8 long files), Critical (higher)">
                      <div className="flex items-center gap-1 group cursor-help">
                        Health
                        <Info className="h-3 w-3 sm:h-3.5 sm:w-3.5 text-slate-500 group-hover:text-slate-300 transition-colors" aria-label="Info: Health status criteria" />
                      </div>
                    </th>
                    <th
                      className="text-right p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                      onClick={() => handleSort("light_issues")}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("light_issues"); } }}
                      tabIndex={0}
                      role="button"
                      aria-label={`Sort by light issues, currently ${sortField === "light_issues" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                      data-testid="scenario-table-header-light-issues"
                      title="Lint and type checking issues - Sort by count"
                    >
                      <div className="flex items-center justify-end gap-2">
                        Light Issues
                        <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                      </div>
                    </th>
                    <th
                      className="text-right p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                      onClick={() => handleSort("ai_issues")}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("ai_issues"); } }}
                      tabIndex={0}
                      role="button"
                      aria-label={`Sort by AI issues, currently ${sortField === "ai_issues" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                      data-testid="scenario-table-header-ai-issues"
                      title="AI-detected code quality issues - Sort by count"
                    >
                      <div className="flex items-center justify-end gap-2">
                        <span className="hidden sm:inline">AI Issues</span>
                        <span className="sm:hidden">AI</span>
                        <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                      </div>
                    </th>
                    <th
                      className="text-right p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                      onClick={() => handleSort("long_files")}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("long_files"); } }}
                      tabIndex={0}
                      role="button"
                      aria-label={`Sort by long files, currently ${sortField === "long_files" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                      data-testid="scenario-table-header-long-files"
                      title="Files exceeding line count threshold - Sort by count"
                    >
                      <div className="flex items-center justify-end gap-2">
                        <span className="hidden sm:inline">Long Files</span>
                        <span className="sm:hidden">Long</span>
                        <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                      </div>
                    </th>
                    <th
                      className="text-right p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                      onClick={() => handleSort("visit_percent")}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("visit_percent"); } }}
                      tabIndex={0}
                      role="button"
                      aria-label={`Sort by visit percentage, currently ${sortField === "visit_percent" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                      data-testid="scenario-table-header-visit-percent"
                      title="Percentage of files reviewed in active campaigns - Sort by percentage"
                    >
                      <div className="flex items-center justify-end gap-2">
                        Visit %
                        <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                      </div>
                    </th>
                    <th
                      className="text-left p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                      onClick={() => handleSort("campaign_status")}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("campaign_status"); } }}
                      tabIndex={0}
                      role="button"
                      aria-label={`Sort by campaign status, currently ${sortField === "campaign_status" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                      data-testid="scenario-table-header-campaign-status"
                      title="Auto-campaign status for systematic file scanning - Sort by status"
                    >
                      <div className="flex items-center gap-2">
                        Campaign
                        <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                      </div>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {sortedScenarios.map((scenario) => {
                    const health = getHealthStatus(scenario);
                    const HealthIcon = health.icon;
                    return (
                      <tr
                        key={scenario.scenario}
                        className="border-b border-white/5 hover:bg-white/5 cursor-pointer transition-colors"
                        onClick={() => navigate(`/scenario/${scenario.scenario}`)}
                        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); navigate(`/scenario/${scenario.scenario}`); } }}
                        tabIndex={0}
                        data-testid="scenario-table-row"
                      >
                        <td className="p-2 sm:p-4">
                          <div>
                            <div className="font-medium text-sm sm:text-base text-slate-100">{scenario.scenario}</div>
                            <div className="text-xs text-slate-400">
                              {scenario.total_files} files · {scenario.total_lines.toLocaleString()} lines
                            </div>
                          </div>
                        </td>
                        <td className="p-2 sm:p-4">
                          <div className="flex items-center gap-1 sm:gap-2" role="status" aria-label={`Code health: ${health.label}`}>
                            <HealthIcon className={cn("h-3 w-3 sm:h-4 sm:w-4", health.color)} aria-hidden="true" />
                            <span className={cn("text-xs sm:text-sm font-medium", health.color)}>{health.label}</span>
                          </div>
                        </td>
                        <td className="p-2 sm:p-4 text-right">
                          <span className={cn("text-sm font-medium", scenario.light_issues > 10 ? "text-red-400" : "text-slate-300")}>
                            {scenario.light_issues}
                          </span>
                        </td>
                        <td className="p-2 sm:p-4 text-right">
                          <span className={cn("text-sm font-medium", scenario.ai_issues > 5 ? "text-red-400" : "text-slate-300")}>
                            {scenario.ai_issues}
                          </span>
                        </td>
                        <td className="p-2 sm:p-4 text-right">
                          <span className={cn("text-sm font-medium", scenario.long_files > 5 ? "text-yellow-400" : "text-slate-300")}>
                            {scenario.long_files}
                          </span>
                        </td>
                        <td className="p-2 sm:p-4 text-right">
                          <div className="flex items-center justify-end gap-1 sm:gap-2">
                            <div
                              className="w-12 sm:w-16 h-2 bg-slate-800 rounded-full overflow-hidden"
                              role="progressbar"
                              aria-valuenow={scenario.visit_percent}
                              aria-valuemin={0}
                              aria-valuemax={100}
                              aria-label={`Campaign progress: ${scenario.visit_percent}% of files visited`}
                            >
                              <div
                                className={cn(
                                  "h-full transition-all",
                                  scenario.visit_percent >= 80 ? "bg-green-500" :
                                  scenario.visit_percent >= 50 ? "bg-yellow-500" : "bg-red-500"
                                )}
                                style={{ width: `${scenario.visit_percent}%` }}
                              />
                            </div>
                            <span className="text-xs sm:text-sm font-medium text-slate-300 w-8 sm:w-10 text-right" aria-hidden="true">
                              {scenario.visit_percent}%
                            </span>
                          </div>
                        </td>
                        <td className="p-2 sm:p-4">{getCampaignBadge(scenario.campaign_status)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
