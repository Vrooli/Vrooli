import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Filter, AlertCircle } from "lucide-react";
import { fetchAllIssues, type Issue } from "../lib/api";
import { PageHeader } from "../components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Select } from "../components/ui/select";
import { Spinner } from "../components/ui/spinner";
import { Alert, AlertDescription } from "../components/ui/alert";
import { cn } from "../lib/utils";

export default function IssuesView() {
  const { scenarioName } = useParams<{ scenarioName: string }>();
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
        description="All code tidiness issues for this scenario"
        backTo={`/scenario/${scenarioName}`}
      />

      {/* Filters */}
      <Card className="mb-6">
        <CardHeader>
          <div className="flex items-center gap-2">
            <Filter className="h-4 w-4" />
            <CardTitle className="text-base">Filters</CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="text-sm text-slate-400 mb-2 block">Status</label>
              <Select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
                <option value="all">All</option>
                <option value="open">Open</option>
                <option value="resolved">Resolved</option>
                <option value="ignored">Ignored</option>
              </Select>
            </div>
            <div>
              <label className="text-sm text-slate-400 mb-2 block">Category</label>
              <Select value={categoryFilter} onChange={(e) => setCategoryFilter(e.target.value)}>
                <option value="all">All</option>
                <option value="lint">Lint</option>
                <option value="type">Type</option>
                <option value="ai">AI</option>
              </Select>
            </div>
            <div>
              <label className="text-sm text-slate-400 mb-2 block">Severity</label>
              <Select value={severityFilter} onChange={(e) => setSeverityFilter(e.target.value)}>
                <option value="all">All</option>
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
          <CardTitle>
            {isLoading ? "Loading..." : `${issues.length} Issues Found`}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center p-12">
              <Spinner size="lg" />
            </div>
          ) : issues.length === 0 ? (
            <div className="text-center py-8 text-slate-400">
              No issues match the current filters
            </div>
          ) : (
            <div className="space-y-3">
              {issues.map((issue, idx) => (
                <div
                  key={idx}
                  className="border border-white/10 rounded-lg p-4 hover:bg-white/5 transition-colors"
                >
                  <div className="flex items-start gap-3">
                    <AlertCircle className={cn(
                      "h-5 w-5 mt-0.5",
                      issue.severity === "error" ? "text-red-500" : "text-yellow-500"
                    )} />
                    <div className="flex-1 space-y-2">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-mono text-sm text-slate-400">
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
                      </div>
                      <p className="text-slate-200">{issue.message}</p>
                      {issue.rule && (
                        <p className="text-xs text-slate-500">Rule: {issue.rule} · Tool: {issue.tool}</p>
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
