import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { ArrowUpDown, AlertCircle, CheckCircle2, FileWarning, Activity, TrendingUp } from "lucide-react";
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
    queryFn: fetchScenarioStats,
  });

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
        <Alert variant="destructive">
          <AlertDescription>Failed to load scenarios: {(error as Error).message}</AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title="Dashboard"
        description="Monitor code tidiness across all scenarios"
      />

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Scenarios</CardTitle>
            <Activity className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalScenarios}</div>
            <p className="text-xs text-slate-400 mt-1">Being monitored</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Issues</CardTitle>
            <AlertCircle className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalIssues}</div>
            <p className="text-xs text-slate-400 mt-1">Across all scenarios</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Long Files</CardTitle>
            <FileWarning className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalLongFiles}</div>
            <p className="text-xs text-slate-400 mt-1">Exceeding threshold</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Campaigns</CardTitle>
            <TrendingUp className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{activeCampaigns}</div>
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
            <div className="flex items-center justify-center p-12">
              <Spinner size="lg" />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full" data-testid="scenario-table">
                <thead className="border-b border-white/10">
                  <tr>
                    <th
                      className="text-left p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                      onClick={() => handleSort("scenario")}
                      data-testid="scenario-table-header-name"
                    >
                      <div className="flex items-center gap-2">
                        Scenario
                        <ArrowUpDown className="h-4 w-4" />
                      </div>
                    </th>
                    <th className="text-left p-4 font-medium text-sm text-slate-400">
                      Health
                    </th>
                    <th
                      className="text-right p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                      onClick={() => handleSort("light_issues")}
                      data-testid="scenario-table-header-light-issues"
                    >
                      <div className="flex items-center justify-end gap-2">
                        Light Issues
                        <ArrowUpDown className="h-4 w-4" />
                      </div>
                    </th>
                    <th
                      className="text-right p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                      onClick={() => handleSort("ai_issues")}
                      data-testid="scenario-table-header-ai-issues"
                    >
                      <div className="flex items-center justify-end gap-2">
                        AI Issues
                        <ArrowUpDown className="h-4 w-4" />
                      </div>
                    </th>
                    <th
                      className="text-right p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                      onClick={() => handleSort("long_files")}
                      data-testid="scenario-table-header-long-files"
                    >
                      <div className="flex items-center justify-end gap-2">
                        Long Files
                        <ArrowUpDown className="h-4 w-4" />
                      </div>
                    </th>
                    <th
                      className="text-right p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                      onClick={() => handleSort("visit_percent")}
                      data-testid="scenario-table-header-visit-percent"
                    >
                      <div className="flex items-center justify-end gap-2">
                        Visit %
                        <ArrowUpDown className="h-4 w-4" />
                      </div>
                    </th>
                    <th
                      className="text-left p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                      onClick={() => handleSort("campaign_status")}
                      data-testid="scenario-table-header-campaign-status"
                    >
                      <div className="flex items-center gap-2">
                        Campaign
                        <ArrowUpDown className="h-4 w-4" />
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
                        data-testid="scenario-table-row"
                      >
                        <td className="p-4">
                          <div>
                            <div className="font-medium text-slate-100">{scenario.scenario}</div>
                            <div className="text-xs text-slate-500">
                              {scenario.total_files} files · {scenario.total_lines.toLocaleString()} lines
                            </div>
                          </div>
                        </td>
                        <td className="p-4">
                          <div className="flex items-center gap-2">
                            <HealthIcon className={cn("h-4 w-4", health.color)} />
                            <span className={cn("text-sm font-medium", health.color)}>{health.label}</span>
                          </div>
                        </td>
                        <td className="p-4 text-right">
                          <span className={cn("font-medium", scenario.light_issues > 10 ? "text-red-400" : "text-slate-300")}>
                            {scenario.light_issues}
                          </span>
                        </td>
                        <td className="p-4 text-right">
                          <span className={cn("font-medium", scenario.ai_issues > 5 ? "text-red-400" : "text-slate-300")}>
                            {scenario.ai_issues}
                          </span>
                        </td>
                        <td className="p-4 text-right">
                          <span className={cn("font-medium", scenario.long_files > 5 ? "text-yellow-400" : "text-slate-300")}>
                            {scenario.long_files}
                          </span>
                        </td>
                        <td className="p-4 text-right">
                          <div className="flex items-center justify-end gap-2">
                            <div className="w-16 h-2 bg-slate-800 rounded-full overflow-hidden">
                              <div
                                className={cn(
                                  "h-full transition-all",
                                  scenario.visit_percent >= 80 ? "bg-green-500" :
                                  scenario.visit_percent >= 50 ? "bg-yellow-500" : "bg-red-500"
                                )}
                                style={{ width: `${scenario.visit_percent}%` }}
                              />
                            </div>
                            <span className="text-sm font-medium text-slate-300 w-10 text-right">
                              {scenario.visit_percent}%
                            </span>
                          </div>
                        </td>
                        <td className="p-4">{getCampaignBadge(scenario.campaign_status)}</td>
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
