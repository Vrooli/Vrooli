import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { AlertCircle, CheckCircle, FileText } from "lucide-react";
import { fetchFileIssues, type Issue } from "../lib/api";
import { PageHeader } from "../components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Spinner } from "../components/ui/spinner";
import { Alert, AlertDescription } from "../components/ui/alert";
import { cn } from "../lib/utils";

export default function FileDetail() {
  const { scenarioName, "*": filePath } = useParams<{ scenarioName: string; "*": string }>();

  const { data: issues = [], isLoading, error } = useQuery({
    queryKey: ["file-issues", scenarioName, filePath],
    queryFn: () => fetchFileIssues(scenarioName!, filePath!),
    enabled: !!scenarioName && !!filePath,
  });

  const getSeverityIcon = (severity: Issue["severity"]) => {
    return severity === "error" ? AlertCircle : AlertCircle;
  };

  const getSeverityColor = (severity: Issue["severity"]) => {
    return severity === "error" ? "text-red-500" : "text-yellow-500";
  };

  if (error) {
    return (
      <div>
        <PageHeader title="File Detail" backTo={`/scenario/${scenarioName}`} />
        <Alert variant="destructive">
          <AlertDescription>Failed to load file issues: {(error as Error).message}</AlertDescription>
        </Alert>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Loading..." backTo={`/scenario/${scenarioName}`} />
        <div className="flex items-center justify-center p-12">
          <Spinner size="lg" />
        </div>
      </div>
    );
  }

  const openIssues = issues.filter(i => i.status === "open");
  const resolvedIssues = issues.filter(i => i.status === "resolved");

  return (
    <div>
      <PageHeader
        title={filePath || "File"}
        description={`${scenarioName} - ${issues.length} issues found`}
        backTo={`/scenario/${scenarioName}`}
      />

      {/* Summary */}
      <div className="grid gap-4 md:grid-cols-3 mb-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Issues</CardTitle>
            <FileText className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{issues.length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Open Issues</CardTitle>
            <AlertCircle className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{openIssues.length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Resolved</CardTitle>
            <CheckCircle className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{resolvedIssues.length}</div>
          </CardContent>
        </Card>
      </div>

      {/* Issues List */}
      <Card>
        <CardHeader>
          <CardTitle>Issues</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {issues.length === 0 ? (
            <div className="text-center py-8 text-slate-400">
              No issues found for this file
            </div>
          ) : (
            issues.map((issue, idx) => {
              const SeverityIcon = getSeverityIcon(issue.severity);
              return (
                <div
                  key={idx}
                  className="border border-white/10 rounded-lg p-4 hover:bg-white/5 transition-colors"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex items-start gap-3 flex-1">
                      <SeverityIcon className={cn("h-5 w-5 mt-0.5", getSeverityColor(issue.severity))} />
                      <div className="flex-1 space-y-2">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm text-slate-400">
                            Line {issue.line}:{issue.column}
                          </span>
                          <Badge variant={issue.severity === "error" ? "error" : "warning"} size="sm">
                            {issue.severity}
                          </Badge>
                          <Badge variant="ghost" size="sm">
                            {issue.category}
                          </Badge>
                        </div>
                        <p className="text-slate-200">{issue.message}</p>
                        {issue.rule && (
                          <p className="text-xs text-slate-500">Rule: {issue.rule}</p>
                        )}
                        <p className="text-xs text-slate-500">Tool: {issue.tool}</p>
                      </div>
                    </div>
                    <div className="flex flex-col gap-2">
                      <Button variant="outline" size="sm">
                        Mark Resolved
                      </Button>
                      <Button variant="ghost" size="sm">
                        Ignore
                      </Button>
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </CardContent>
      </Card>
    </div>
  );
}
