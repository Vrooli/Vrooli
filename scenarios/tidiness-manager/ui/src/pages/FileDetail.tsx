import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { AlertCircle, CheckCircle, FileText, Info } from "lucide-react";
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
        <Alert variant="destructive" role="alert" aria-live="assertive">
          <AlertCircle className="h-4 w-4" aria-hidden="true" />
          <AlertDescription>
            <p className="font-semibold mb-2">Failed to load file issues</p>
            <p className="text-sm">{(error as Error).message}</p>
            <div className="mt-3 pt-3 border-t border-red-500/20">
              <p className="text-xs text-red-200/80 mb-2">Troubleshooting:</p>
              <ul className="text-xs text-red-200/70 space-y-1">
                <li>• Verify the file path is correct</li>
                <li>• Check if the scenario exists: <code className="bg-black/30 px-1 rounded">tidiness-manager issues {scenarioName}</code></li>
                <li>• Ensure the API is running and healthy</li>
              </ul>
            </div>
          </AlertDescription>
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
        description={
          <div className="space-y-1">
            <p>{scenarioName} - {issues.length} issues found</p>
            <p className="text-xs text-slate-500 font-mono">
              CLI: tidiness-manager issues {scenarioName} --file="{filePath}"
            </p>
          </div>
        }
        backTo={`/scenario/${scenarioName}`}
      />

      {/* Summary */}
      <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 md:grid-cols-3 mb-6">
        <Card data-testid="total-issues-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <div className="flex items-center gap-1">
              <CardTitle className="text-sm font-medium">Total Issues</CardTitle>
              <Info
                className="h-3 w-3 text-slate-500 cursor-help"
                title="All issues found in this file from light scans (lint/type) and AI analysis"
                aria-hidden="true"
              />
            </div>
            <FileText className="h-4 w-4 text-slate-400" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${issues.length} total issues`}>{issues.length}</div>
            <p className="text-xs text-slate-500 mt-1" id="total-issues-desc">
              All issues found in this file
            </p>
          </CardContent>
        </Card>

        <Card data-testid="open-issues-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <div className="flex items-center gap-1">
              <CardTitle className="text-sm font-medium">Open Issues</CardTitle>
              <Info
                className="h-3 w-3 text-slate-500 cursor-help"
                title="Issues that require attention. Mark as resolved when fixed or ignore if not applicable."
                aria-hidden="true"
              />
            </div>
            <AlertCircle className="h-4 w-4 text-yellow-500" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${openIssues.length} open issues`}>{openIssues.length}</div>
            <p className="text-xs text-slate-500 mt-1" id="open-issues-desc">
              Issues requiring attention
            </p>
          </CardContent>
        </Card>

        <Card data-testid="resolved-issues-card">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <div className="flex items-center gap-1">
              <CardTitle className="text-sm font-medium">Resolved</CardTitle>
              <Info
                className="h-3 w-3 text-slate-500 cursor-help"
                title="Issues that have been fixed or marked as resolved"
                aria-hidden="true"
              />
            </div>
            <CheckCircle className="h-4 w-4 text-green-500" aria-hidden="true" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold" aria-label={`${resolvedIssues.length} resolved issues`}>{resolvedIssues.length}</div>
            <p className="text-xs text-slate-500 mt-1" id="resolved-issues-desc">
              Fixed or dismissed issues
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Issues List */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <CardTitle>Issues</CardTitle>
            <Info
              className="h-4 w-4 text-slate-500 cursor-help"
              title="Issues detected by linters, type checkers, or AI analysis. Line:Column indicates exact location in source file."
            />
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {issues.length === 0 ? (
            <div className="text-center py-12 space-y-4" role="status" aria-live="polite">
              <CheckCircle className="h-12 w-12 text-green-500 mx-auto" aria-hidden="true" />
              <div className="space-y-2">
                <p className="text-slate-400 font-semibold text-lg">No issues found</p>
                <p className="text-sm text-slate-500">
                  This file passed all linting, type checking, and AI analysis
                </p>
              </div>
              <div className="mt-4 pt-4 border-t border-white/10 max-w-md mx-auto">
                <p className="text-xs text-slate-600 mb-2">Next steps:</p>
                <ul className="text-xs text-slate-500 space-y-1 text-left">
                  <li>• Run a new scan to check for changes</li>
                  <li>• Review other files in {scenarioName}</li>
                  <li>• Check the global dashboard for pending issues</li>
                </ul>
              </div>
            </div>
          ) : (
            issues.map((issue, idx) => {
              const SeverityIcon = getSeverityIcon(issue.severity);
              return (
                <div
                  key={idx}
                  className="border border-white/10 rounded-lg p-4 hover:bg-white/5 transition-colors focus-within:ring-2 focus-within:ring-blue-500/50"
                  role="article"
                  aria-labelledby={`issue-${idx}-location`}
                  aria-describedby={`issue-${idx}-message`}
                >
                  <div className="flex flex-col md:flex-row items-start gap-4">
                    <div className="flex items-start gap-3 flex-1 w-full">
                      <SeverityIcon
                        className={cn("h-5 w-5 mt-0.5 flex-shrink-0", getSeverityColor(issue.severity))}
                        aria-label={`${issue.severity} severity`}
                      />
                      <div className="flex-1 space-y-2 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span
                            id={`issue-${idx}-location`}
                            className="font-mono text-sm text-slate-400"
                            title="Line and column number in source file where this issue occurs"
                          >
                            Line {issue.line}:{issue.column}
                          </span>
                          <Badge variant={issue.severity === "error" ? "error" : "warning"} size="sm">
                            {issue.severity}
                          </Badge>
                          <Badge variant="ghost" size="sm">
                            {issue.category}
                          </Badge>
                        </div>
                        <p id={`issue-${idx}-message`} className="text-slate-200 break-words">{issue.message}</p>
                        {issue.rule && (
                          <p className="text-xs text-slate-500">Rule: {issue.rule}</p>
                        )}
                        <p className="text-xs text-slate-500">Tool: {issue.tool}</p>
                      </div>
                    </div>
                    <div className="flex md:flex-col gap-2 w-full md:w-auto">
                      <Button
                        variant="outline"
                        size="sm"
                        className="flex-1 md:flex-none focus:ring-2 focus:ring-green-500/50"
                        title="Mark this issue as resolved after fixing the code"
                        aria-label={`Mark issue at line ${issue.line}:${issue.column} as resolved`}
                      >
                        Mark Resolved
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="flex-1 md:flex-none focus:ring-2 focus:ring-slate-500/50"
                        title="Ignore this issue if it's not applicable or a false positive"
                        aria-label={`Ignore issue at line ${issue.line}:${issue.column}`}
                      >
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
