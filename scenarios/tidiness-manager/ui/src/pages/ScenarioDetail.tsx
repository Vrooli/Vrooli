import { useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowUpDown, Play, FileText, AlertTriangle, Eye, Sparkles } from "lucide-react";
import { useState } from "react";
import { fetchScenarioDetail, type FileStats } from "../lib/api";
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
          <>
            <Button
              variant="outline"
              size="sm"
              onClick={() => navigate(`/scenario/${scenarioName}/issues`)}
            >
              <Eye className="h-4 w-4 mr-2" />
              View All Issues
            </Button>
            <Button
              variant="primary"
              size="sm"
              onClick={() => console.log("Trigger light scan")}
            >
              <Play className="h-4 w-4 mr-2" />
              Run Light Scan
            </Button>
          </>
        }
      />

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <Card data-testid="total-files-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Files</CardTitle>
            <FileText className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.total_files || 0}</div>
            <p className="text-xs text-slate-400 mt-1">{(stats?.total_lines || 0).toLocaleString()} total lines</p>
          </CardContent>
        </Card>

        <Card data-testid="total-issues-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Issues</CardTitle>
            <AlertTriangle className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalIssues}</div>
            <p className="text-xs text-slate-400 mt-1">
              {stats?.light_issues || 0} light · {stats?.ai_issues || 0} AI
            </p>
          </CardContent>
        </Card>

        <Card data-testid="long-files-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Long Files</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.long_files || 0}</div>
            <p className="text-xs text-slate-400 mt-1">Exceeding threshold</p>
          </CardContent>
        </Card>

        <Card data-testid="visit-coverage-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Visit Coverage</CardTitle>
            <Sparkles className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.visit_percent || 0}%</div>
            <p className="text-xs text-slate-400 mt-1">Files analyzed</p>
          </CardContent>
        </Card>
      </div>

      {/* Files Table */}
      <Card>
        <CardHeader>
          <CardTitle>Files</CardTitle>
          <CardDescription>
            Detailed file metrics and issue counts
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full" data-testid="file-table">
              <thead className="border-b border-white/10">
                <tr>
                  <th
                    className="text-left p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                    onClick={() => handleSort("path")}
                    data-testid="file-table-header-path"
                  >
                    <div className="flex items-center gap-2">
                      File Path
                      <ArrowUpDown className="h-4 w-4" />
                    </div>
                  </th>
                  <th
                    className="text-right p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                    onClick={() => handleSort("lines")}
                    data-testid="file-table-header-lines"
                  >
                    <div className="flex items-center justify-end gap-2">
                      Lines
                      <ArrowUpDown className="h-4 w-4" />
                    </div>
                  </th>
                  <th
                    className="text-right p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                    onClick={() => handleSort("totalIssues")}
                    data-testid="file-table-header-issues"
                  >
                    <div className="flex items-center justify-end gap-2">
                      Total Issues
                      <ArrowUpDown className="h-4 w-4" />
                    </div>
                  </th>
                  <th
                    className="text-right p-4 cursor-pointer hover:bg-white/5 transition-colors font-medium text-sm text-slate-400"
                    onClick={() => handleSort("visit_count")}
                    data-testid="file-table-header-visit-count"
                  >
                    <div className="flex items-center justify-end gap-2">
                      Visits
                      <ArrowUpDown className="h-4 w-4" />
                    </div>
                  </th>
                  <th className="text-right p-4 font-medium text-sm text-slate-400">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody>
                {sortedFiles.map((file) => (
                  <tr
                    key={file.path}
                    className="border-b border-white/5 hover:bg-white/5 transition-colors"
                    data-testid="file-table-row"
                  >
                    <td className="p-4">
                      <div className="flex items-center gap-2">
                        {file.is_long_file && (
                          <Badge variant="warning" size="sm">LONG</Badge>
                        )}
                        <span className="font-mono text-sm text-slate-200">{file.path}</span>
                      </div>
                    </td>
                    <td className="p-4 text-right">
                      <span className={cn(
                        "font-medium",
                        file.is_long_file ? "text-yellow-400" : "text-slate-300"
                      )}>
                        {file.lines}
                      </span>
                    </td>
                    <td className="p-4 text-right">
                      <div className="flex flex-col items-end gap-1">
                        <span className={cn(
                          "font-medium",
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
                    <td className="p-4 text-right">
                      <span className="text-slate-300">{file.visit_count}</span>
                    </td>
                    <td className="p-4 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/scenario/${scenarioName}/file/${encodeURIComponent(file.path)}`)}
                      >
                        View Details
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
