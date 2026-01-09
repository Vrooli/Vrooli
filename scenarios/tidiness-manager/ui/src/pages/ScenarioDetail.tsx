import { useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowUpDown, Play, FileText, AlertTriangle, Eye, Sparkles, Info, Terminal } from "lucide-react";
import { useState } from "react";
import { fetchScenarioDetail, type FileStats } from "../lib/api";
import { useToast } from "../components/ui/toast";
import { PageHeader } from "../components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Spinner } from "../components/ui/spinner";
import { Alert, AlertDescription } from "../components/ui/alert";
import { cn } from "../lib/utils";

type SortField = "path" | "lines" | "totalIssues" | "visit_count";

export default function ScenarioDetail() {
  const { scenarioName } = useParams<{ scenarioName: string }>();
  const navigate = useNavigate();
  const { showToast } = useToast();
  const [sortField, setSortField] = useState<SortField>("totalIssues");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");

  const { data, isLoading, error } = useQuery({
    queryKey: ["scenario-detail", scenarioName],
    queryFn: () => fetchScenarioDetail(scenarioName!),
    enabled: !!scenarioName,
  });

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDirection("desc");
    }
  };

  const filesWithTotals = (data?.files || []).map((file) => ({
    ...file,
    totalIssues: file.lint_issues + file.type_issues + file.ai_issues,
  }));

  const sortedFiles = [...filesWithTotals].sort((a, b) => {
    const multiplier = sortDirection === "asc" ? 1 : -1;

    if (sortField === "path") {
      return multiplier * a.path.localeCompare(b.path);
    }
    return multiplier * (a[sortField] - b[sortField]);
  });

  if (error) {
    return (
      <div>
        <PageHeader title="Scenario Detail" backTo="/" />
        <Alert variant="destructive">
          <AlertDescription>Failed to load scenario: {(error as Error).message}</AlertDescription>
        </Alert>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Loading..." backTo="/" />
        <div className="flex items-center justify-center p-12">
          <Spinner size="lg" />
        </div>
      </div>
    );
  }

  const stats = data?.stats;
  const totalIssues = (stats?.light_issues || 0) + (stats?.ai_issues || 0);

  return (
    <div>
      <PageHeader
        title={scenarioName || "Scenario"}
        description={`Code tidiness overview for ${scenarioName}`}
        backTo="/"
        actions={
          <div className="flex gap-2 flex-wrap">
            <Button
              variant="outline"
              size="sm"
              onClick={() => navigate(`/scenario/${scenarioName}/issues`)}
              title="View all issues with filters for status, category, and severity. Mark issues as resolved or ignored."
              className="flex-1 sm:flex-none"
            >
              <Eye className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">View All Issues</span>
              <span className="sm:hidden ml-2">Issues</span>
            </Button>
            <Button
              variant="primary"
              size="sm"
              onClick={async () => {
                console.log("Trigger light scan for", scenarioName);
                // TODO: Wire to API endpoint POST /scan/light when backend ready
                const cliCommand = `tidiness-manager scan ${scenarioName} --type light --wait`;

                // Copy command to clipboard automatically
                try {
                  await navigator.clipboard.writeText(cliCommand);
                  showToast(
                    `CLI command copied to clipboard:\n\n${cliCommand}\n\nRun this command to trigger a light scan (completes in 60-120s).\nRefresh this page afterward to see results.\n\nNote: Direct UI triggering requires API wiring (POST /api/v1/scan/light).`,
                    "info",
                    8000
                  );
                } catch (err) {
                  showToast(
                    `To trigger a light scan:\n\n${cliCommand}\n\nAPI integration coming soon for direct UI triggering.`,
                    "info",
                    8000
                  );
                }
              }}
              title="Run make lint and make type, parse outputs into structured issues. Completes in 60-120 seconds for most scenarios. Click to copy CLI command."
              className="flex-1 sm:flex-none"
            >
              <Play className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">Run Light Scan</span>
              <span className="sm:hidden ml-2">Scan</span>
            </Button>
          </div>
        }
      />

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <Card data-testid="total-files-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Files</CardTitle>
            <FileText className="h-4 w-4 text-slate-400" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${stats?.total_files || 0} total files with ${(stats?.total_lines || 0).toLocaleString()} lines`}>{stats?.total_files || 0}</div>
            <p className="text-xs text-slate-400 mt-1">{(stats?.total_lines || 0).toLocaleString()} total lines</p>
          </CardContent>
        </Card>

        <Card data-testid="total-issues-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Issues</CardTitle>
            <AlertTriangle className="h-4 w-4 text-yellow-500" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${totalIssues} total issues: ${stats?.light_issues || 0} light and ${stats?.ai_issues || 0} AI`}>{totalIssues}</div>
            <p className="text-xs text-slate-400 mt-1">
              {stats?.light_issues || 0} light · {stats?.ai_issues || 0} AI
            </p>
          </CardContent>
        </Card>

        <Card data-testid="long-files-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Long Files</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${stats?.long_files || 0} files exceeding length threshold`}>{stats?.long_files || 0}</div>
            <p className="text-xs text-slate-400 mt-1">Exceeding threshold</p>
          </CardContent>
        </Card>

        <Card data-testid="visit-coverage-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-1 group cursor-help" title="Percentage of files analyzed by visited-tracker campaigns. Higher coverage ensures comprehensive tidiness analysis without redundant work.">
              Visit Coverage
              <Info className="h-3 w-3 text-slate-500 group-hover:text-slate-300 transition-colors" aria-label="Info: Visit coverage explanation" />
            </CardTitle>
            <Sparkles className="h-4 w-4 text-blue-500" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${stats?.visit_percent || 0}% of files analyzed by campaigns`}>{stats?.visit_percent || 0}%</div>
            <p className="text-xs text-slate-400 mt-1">Files analyzed</p>
          </CardContent>
        </Card>
      </div>

      {/* Files Table */}
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1">
              <CardTitle>Files</CardTitle>
              <CardDescription>
                Detailed file metrics and issue counts. Click column headers to sort.
              </CardDescription>
            </div>
            <div className="bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-xs text-slate-400 flex items-center gap-2">
              <Terminal className="h-3.5 w-3.5" />
              <span>
                CLI: <code className="text-slate-300">tidiness-manager issues {scenarioName} --limit 10</code>
              </span>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <div className="overflow-x-auto -mx-4 sm:mx-0">
            <table className="w-full min-w-[600px]" data-testid="file-table" role="table" aria-label="Files in scenario with issues and metrics">
              <thead className="border-b border-white/10" role="rowgroup">
                <tr>
                  <th
                    className="text-left p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                    onClick={() => handleSort("path")}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("path"); } }}
                    tabIndex={0}
                    role="button"
                    aria-label={`Sort by file path, currently ${sortField === "path" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                    data-testid="file-table-header-path"
                  >
                    <div className="flex items-center gap-2">
                      File Path
                      <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                    </div>
                  </th>
                  <th
                    className="text-right p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                    onClick={() => handleSort("lines")}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("lines"); } }}
                    tabIndex={0}
                    role="button"
                    aria-label={`Sort by line count, currently ${sortField === "lines" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                    data-testid="file-table-header-lines"
                  >
                    <div className="flex items-center justify-end gap-2">
                      Lines
                      <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                    </div>
                  </th>
                  <th
                    className="text-right p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                    onClick={() => handleSort("totalIssues")}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("totalIssues"); } }}
                    tabIndex={0}
                    role="button"
                    aria-label={`Sort by total issues, currently ${sortField === "totalIssues" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                    data-testid="file-table-header-issues"
                  >
                    <div className="flex items-center justify-end gap-2">
                      <span className="hidden sm:inline">Total Issues</span>
                      <span className="sm:hidden">Issues</span>
                      <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                    </div>
                  </th>
                  <th
                    className="text-right p-2 sm:p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-xs sm:text-sm text-slate-400"
                    onClick={() => handleSort("visit_count")}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSort("visit_count"); } }}
                    tabIndex={0}
                    role="button"
                    aria-label={`Sort by visit count, currently ${sortField === "visit_count" ? (sortDirection === "asc" ? "ascending" : "descending") : "not sorted"}`}
                    data-testid="file-table-header-visit-count"
                    title="Number of times this file has been analyzed by smart scans. Used by visited-tracker to prioritize stale files."
                  >
                    <div className="flex items-center justify-end gap-2">
                      Visits
                      <ArrowUpDown className="h-4 w-4" aria-hidden="true" />
                    </div>
                  </th>
                  <th className="text-right p-2 sm:p-4 font-medium text-xs sm:text-sm text-slate-400">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody role="rowgroup">
                {sortedFiles.map((file) => (
                  <tr
                    key={file.path}
                    className="border-b border-white/5 hover:bg-white/5 transition-colors"
                    data-testid="file-table-row"
                  >
                    <td className="p-2 sm:p-4">
                      <div className="flex items-center gap-2 flex-wrap">
                        {file.is_long_file && (
                          <Badge variant="warning" size="sm">LONG</Badge>
                        )}
                        <span className="font-mono text-xs sm:text-sm text-slate-200 break-all">{file.path}</span>
                      </div>
                    </td>
                    <td className="p-2 sm:p-4 text-right">
                      <span className={cn(
                        "text-sm font-medium",
                        file.is_long_file ? "text-yellow-400" : "text-slate-300"
                      )}>
                        {file.lines}
                      </span>
                    </td>
                    <td className="p-2 sm:p-4 text-right">
                      <div className="flex flex-col items-end gap-1">
                        <span className={cn(
                          "text-sm font-medium",
                          file.totalIssues > 5 ? "text-red-400" : "text-slate-300"
                        )}>
                          {file.totalIssues}
                        </span>
                        {file.totalIssues > 0 && (
                          <span className="text-xs text-slate-500">
                            {file.lint_issues}L · {file.type_issues}T · {file.ai_issues}AI
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="p-2 sm:p-4 text-right">
                      <span className="text-sm text-slate-300">{file.visit_count}</span>
                    </td>
                    <td className="p-2 sm:p-4 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/scenario/${scenarioName}/file/${encodeURIComponent(file.path)}`)}
                        className="text-xs sm:text-sm"
                      >
                        <span className="hidden sm:inline">View Details</span>
                        <span className="sm:hidden">View</span>
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
