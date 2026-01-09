import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Filter, AlertCircle, CheckCircle2, FolderOpen, X, XCircle, Ban, Info, Terminal, Download, Copy } from "lucide-react";
import { fetchAllIssues, updateIssueStatus, type Issue } from "../lib/api";
import { useToast } from "../components/ui/toast";
import { PageHeader } from "../components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Select } from "../components/ui/select";
import { Button } from "../components/ui/button";
import { Spinner } from "../components/ui/spinner";
import { Alert, AlertDescription } from "../components/ui/alert";
import { cn } from "../lib/utils";

export default function IssuesView() {
  const { scenarioName } = useParams<{ scenarioName: string }>();
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const [statusFilter, setStatusFilter] = useState<string>("open");
  const [categoryFilter, setCategoryFilter] = useState<string>("all");
  const [severityFilter, setSeverityFilter] = useState<string>("all");

  const { data: issues = [], isLoading, error } = useQuery({
    queryKey: ["all-issues", scenarioName, statusFilter, categoryFilter, severityFilter],
    queryFn: () => fetchAllIssues(scenarioName!, {
      status: statusFilter !== "all" ? statusFilter : undefined,
      category: categoryFilter !== "all" ? categoryFilter : undefined,
      severity: severityFilter !== "all" ? severityFilter : undefined,
    }),
    enabled: !!scenarioName,
  });

  const updateMutation = useMutation({
    mutationFn: ({ issueId, status }: { issueId: number; status: "open" | "resolved" | "ignored" }) =>
      updateIssueStatus(issueId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["all-issues", scenarioName] });
    },
  });

  const handleResolve = (issueId: number | undefined) => {
    if (!issueId) return;
    updateMutation.mutate({ issueId, status: "resolved" });
  };

  const handleIgnore = (issueId: number | undefined) => {
    if (!issueId) return;
    updateMutation.mutate({ issueId, status: "ignored" });
  };

  const hasActiveFilters = statusFilter !== "open" || categoryFilter !== "all" || severityFilter !== "all";

  const clearFilters = () => {
    setStatusFilter("open");
    setCategoryFilter("all");
    setSeverityFilter("all");
  };

  const exportToJSON = () => {
    const exportData = {
      scenario: scenarioName,
      exported_at: new Date().toISOString(),
      filters: {
        status: statusFilter !== "all" ? statusFilter : "all",
        category: categoryFilter !== "all" ? categoryFilter : "all",
        severity: severityFilter !== "all" ? severityFilter : "all",
      },
      total_issues: issues.length,
      issues: issues.map(issue => ({
        id: issue.id,
        file: issue.file,
        line: issue.line,
        column: issue.column,
        severity: issue.severity,
        category: issue.category,
        message: issue.message,
        rule: issue.rule,
        tool: issue.tool,
        status: issue.status,
      })),
    };

    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `tidiness-issues-${scenarioName}-${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    showToast(`Exported ${issues.length} issues to JSON file`, "success");
  };

  const copyJSONToClipboard = async () => {
    const exportData = {
      scenario: scenarioName,
      filters: {
        status: statusFilter !== "all" ? statusFilter : "all",
        category: categoryFilter !== "all" ? categoryFilter : "all",
        severity: severityFilter !== "all" ? severityFilter : "all",
      },
      total_issues: issues.length,
      issues: issues.map(issue => ({
        id: issue.id,
        file: issue.file,
        line: issue.line,
        column: issue.column,
        severity: issue.severity,
        category: issue.category,
        message: issue.message,
        rule: issue.rule,
        tool: issue.tool,
        status: issue.status,
      })),
    };

    try {
      await navigator.clipboard.writeText(JSON.stringify(exportData, null, 2));
      showToast(`Copied ${issues.length} issues as JSON to clipboard`, "success");
    } catch (err) {
      showToast("Failed to copy to clipboard", "error");
    }
  };

  if (error) {
    return (
      <div>
        <PageHeader title="Issues" backTo={`/scenario/${scenarioName}`} />
        <Alert variant="destructive">
          <AlertDescription>Failed to load issues: {(error as Error).message}</AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title={`Issues - ${scenarioName}`}
        description="All code tidiness issues detected by light scanning (lint/type) and smart scanning (AI)"
        backTo={`/scenario/${scenarioName}`}
        actions={
          <div className="flex items-center gap-2 flex-wrap">
            <div className="bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-xs text-slate-400 flex items-center gap-2">
              <Terminal className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">
                CLI: <code className="text-slate-300">tidiness-manager issues {scenarioName} --category lint --limit 20</code>
              </span>
              <span className="sm:hidden">
                CLI available
              </span>
            </div>
          </div>
        }
      />

      {/* Filters */}
      <Card className="mb-6">
        <CardHeader>
          <div className="flex items-center justify-between flex-wrap gap-2">
            <div className="flex items-center gap-2 flex-wrap">
              <Filter className="h-4 w-4" />
              <CardTitle className="text-sm sm:text-base">Filter Issues</CardTitle>
              {hasActiveFilters && (
                <Badge variant="ghost" size="sm">
                  {issues.length} {issues.length === 1 ? 'result' : 'results'}
                </Badge>
              )}
            </div>
            {hasActiveFilters && (
              <Button
                variant="ghost"
                size="sm"
                onClick={clearFilters}
                className="text-slate-400 hover:text-slate-200"
              >
                <X className="h-4 w-4 mr-1" />
                <span className="hidden sm:inline">Clear filters</span>
                <span className="sm:hidden">Clear</span>
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <div>
              <label htmlFor="status-filter" className="text-sm text-slate-400 mb-2 flex items-center gap-1 group cursor-help" title="Open: Needs attention • Resolved: Fixed by developer • Ignored: Intentional or false positive">
                Status
                <Info className="h-3 w-3 text-slate-500 group-hover:text-slate-300 transition-colors" />
              </label>
              <Select
                id="status-filter"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                aria-label="Filter by issue status"
              >
                <option value="all">All Statuses</option>
                <option value="open">Open</option>
                <option value="resolved">Resolved</option>
                <option value="ignored">Ignored</option>
              </Select>
            </div>
            <div>
              <label htmlFor="category-filter" className="text-sm text-slate-400 mb-2 flex items-center gap-1 group cursor-help" title="Lint: ESLint/golangci-lint violations • Type: TypeScript/Go type errors • AI: Claude-detected code smells">
                Category
                <Info className="h-3 w-3 text-slate-500 group-hover:text-slate-300 transition-colors" />
              </label>
              <Select
                id="category-filter"
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
                aria-label="Filter by issue category"
              >
                <option value="all">All Categories</option>
                <option value="lint">Lint</option>
                <option value="type">Type</option>
                <option value="ai">AI</option>
              </Select>
            </div>
            <div>
              <label htmlFor="severity-filter" className="text-sm text-slate-400 mb-2 block">
                Severity
              </label>
              <Select
                id="severity-filter"
                value={severityFilter}
                onChange={(e) => setSeverityFilter(e.target.value)}
                aria-label="Filter by severity level"
              >
                <option value="all">All Severities</option>
                <option value="error">Error</option>
                <option value="warning">Warning</option>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Issues List */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-4 flex-wrap">
            <CardTitle>
              {isLoading ? "Loading..." : `${issues.length} Issues Found`}
            </CardTitle>
            {!isLoading && issues.length > 0 && (
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={copyJSONToClipboard}
                  title="Copy issues as JSON to clipboard for agent consumption"
                  className="flex-1 sm:flex-none"
                >
                  <Copy className="h-4 w-4 sm:mr-2" />
                  <span className="hidden sm:inline">Copy JSON</span>
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={exportToJSON}
                  title="Download issues as JSON file for programmatic processing"
                  className="flex-1 sm:flex-none"
                >
                  <Download className="h-4 w-4 sm:mr-2" />
                  <span className="hidden sm:inline">Export JSON</span>
                </Button>
              </div>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center p-12">
              <Spinner size="lg" />
            </div>
          ) : issues.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
              {statusFilter === "all" && categoryFilter === "all" && severityFilter === "all" ? (
                <>
                  <CheckCircle2 className="h-12 w-12 text-green-500 mb-4" />
                  <h3 className="text-lg font-semibold text-slate-200 mb-2">
                    No Issues Found
                  </h3>
                  <p className="text-slate-400 max-w-md mb-4">
                    Great news! This scenario has no code tidiness issues detected. All files pass linting and type checks.
                  </p>
                  <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 max-w-lg text-left">
                    <p className="text-xs text-blue-200 mb-2 font-medium">For ongoing monitoring:</p>
                    <code className="text-xs bg-slate-950 px-3 py-2 rounded block text-slate-300 font-mono">
                      tidiness-manager campaigns start {scenarioName} --max-sessions 10
                    </code>
                    <p className="text-xs text-slate-500 mt-2">
                      Creates systematic campaign to analyze all files with visited-tracker, surfacing deeper code quality patterns over time.
                    </p>
                  </div>
                </>
              ) : (
                <>
                  <FolderOpen className="h-12 w-12 text-slate-500 mb-4" />
                  <h3 className="text-lg font-semibold text-slate-200 mb-2">
                    No Matching Issues
                  </h3>
                  <p className="text-slate-400 max-w-md mb-3">
                    No issues match the current filters. Try adjusting your filter selection above.
                  </p>
                  <div className="text-xs text-slate-500 space-y-1">
                    <p>Current filters: Status={statusFilter}, Category={categoryFilter}, Severity={severityFilter}</p>
                    <p className="font-mono">
                      CLI equivalent: tidiness-manager issues {scenarioName} --status {statusFilter} --category {categoryFilter} --severity {severityFilter}
                    </p>
                  </div>
                </>
              )}
            </div>
          ) : (
            <div className="space-y-3">
              {issues.map((issue, idx) => (
                <div
                  key={issue.id || idx}
                  className="border border-white/10 rounded-lg p-3 sm:p-4 hover:bg-white/5 transition-colors"
                >
                  <div className="flex items-start gap-2 sm:gap-3">
                    <AlertCircle className={cn(
                      "h-4 w-4 sm:h-5 sm:w-5 mt-0.5 flex-shrink-0",
                      issue.severity === "error" ? "text-red-500" : "text-yellow-500"
                    )} />
                    <div className="flex-1 space-y-2 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-mono text-xs sm:text-sm text-slate-400 break-all">
                          {issue.file}:{issue.line}:{issue.column}
                        </span>
                        <Badge variant={issue.severity === "error" ? "error" : "warning"} size="sm">
                          {issue.severity}
                        </Badge>
                        <Badge variant="ghost" size="sm">
                          {issue.category}
                        </Badge>
                        {issue.status === "resolved" && (
                          <Badge variant="success" size="sm">Resolved</Badge>
                        )}
                        {issue.status === "ignored" && (
                          <Badge variant="ghost" size="sm">Ignored</Badge>
                        )}
                      </div>
                      <p className="text-sm sm:text-base text-slate-200">{issue.message}</p>
                      {issue.rule && (
                        <p className="text-xs text-slate-500">Rule: {issue.rule} · Tool: {issue.tool}</p>
                      )}
                      {issue.status === "open" && issue.id && (
                        <div className="flex items-center gap-2 pt-2 flex-wrap">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleResolve(issue.id)}
                            disabled={updateMutation.isPending}
                            className="text-green-400 hover:text-green-300 hover:bg-green-400/10 flex-1 sm:flex-none"
                            title="Mark this issue as fixed in the codebase"
                          >
                            <CheckCircle2 className="h-4 w-4 sm:mr-1" />
                            <span className="hidden sm:inline">Mark Resolved</span>
                            <span className="sm:hidden ml-1">Resolve</span>
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleIgnore(issue.id)}
                            disabled={updateMutation.isPending}
                            className="text-slate-400 hover:text-slate-300 hover:bg-slate-400/10 flex-1 sm:flex-none"
                            title="Ignore this issue (intentional pattern or false positive)"
                          >
                            <Ban className="h-4 w-4 sm:mr-1" />
                            <span className="hidden sm:inline">Ignore</span>
                            <span className="sm:hidden ml-1">Ignore</span>
                          </Button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
