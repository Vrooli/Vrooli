// DOC: docs/reference/api-endpoints.md#documentation-health
// DOC: docs/reference/api-endpoints.md#documentation-healing
import { AlertCircle, CheckCircle, Loader2, MoveRight, RefreshCw, Wrench, XCircle, Zap } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Button } from "../../../shared/ui/button";
import type { DocHealthViewModel, HealthTone } from "../../../shared/controllers/documentationController";
import { useDocAutoFix, useDocHealing } from "../../../shared/hooks/healingHooks";

const formatPercent = (value?: number) => {
  if (value === undefined || value === null || Number.isNaN(value)) return "—";
  return `${Math.round(value * 100)}%`;
};

const toneClasses: Record<HealthTone, string> = {
  good: "ko-tone-good",
  medium: "ko-tone-medium",
  poor: "ko-tone-poor",
};

export type HealthPanelProps = {
  scenarioName: string | null;
  healthViewModel: DocHealthViewModel;
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
  onRefresh: () => void;
};

export function HealthPanel({
  scenarioName,
  healthViewModel,
  isLoading,
  hasError,
  errorMessage,
  onRefresh,
}: HealthPanelProps) {
  const [isHealOpen, setIsHealOpen] = useState(false);
  const [selectedIssues, setSelectedIssues] = useState<string[]>([]);
  const [autoApprove, setAutoApprove] = useState(false);
  const [dryRun, setDryRun] = useState(false);
  const [rejectReason, setRejectReason] = useState("");
  const { job, isBusy, error: healError, actions } = useDocHealing(scenarioName);
  const { result: autoFixResult, isLoading: isAutoFixing, error: autoFixError, autoFix, clear: clearAutoFix } = useDocAutoFix(scenarioName);

  const issueOptions = useMemo(() => {
    const issues: string[] = [];
    healthViewModel.missingDocs.forEach((doc) => issues.push(`Missing: ${doc}`));
    healthViewModel.misplacedDocs.forEach((doc) =>
      issues.push(`Misplaced: ${doc.actualPath} -> ${doc.expectedPath}`)
    );
    healthViewModel.extraDocs.forEach((doc) => issues.push(`Extra: ${doc}`));
    return issues;
  }, [healthViewModel.extraDocs, healthViewModel.misplacedDocs, healthViewModel.missingDocs]);

  useEffect(() => {
    setSelectedIssues(issueOptions);
    setRejectReason("");
    setAutoApprove(false);
    setDryRun(false);
  }, [scenarioName, issueOptions]);

  const toggleIssue = (issue: string) => {
    setSelectedIssues((current) =>
      current.includes(issue) ? current.filter((item) => item !== issue) : [...current, issue]
    );
  };

  const handleStartHealing = async () => {
    if (!scenarioName) return;
    await actions.startHealing({
      scenario_name: scenarioName,
      issues: selectedIssues,
      auto_approve: autoApprove,
      dry_run: dryRun,
    });
  };

  const handleApprove = async () => {
    await actions.approve();
  };

  const handleReject = async () => {
    await actions.reject(undefined, rejectReason);
  };

  const resetJob = () => {
    actions.clearJob();
    setRejectReason("");
  };

  if (!scenarioName) {
    return <div className="ko-panel p-4 ko-text-sm ko-muted">Select a scenario to view health details.</div>;
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-10">
        <RefreshCw className="h-5 w-5 ko-icon animate-spin" />
        <span className="ml-3 ko-text-sm ko-muted">Loading doc health...</span>
      </div>
    );
  }

  if (hasError) {
    return (
      <div className="ko-alert ko-alert-danger">
        <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
        <div className="flex-1">
          <p className="ko-alert-title ko-text-danger-strong">Failed to load health</p>
          <p className="ko-text-sm ko-text-danger-muted mt-1">{errorMessage}</p>
          <Button onClick={onRefresh} variant="danger" className="mt-3">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const toneClass = toneClasses[healthViewModel.healthTone] ?? toneClasses.medium;
  const fixCategory = healthViewModel.fixCategory;
  const canHeal = healthViewModel.canAutoFix && healthViewModel.hasIssues;
  const healButtonLabel =
    job?.status === "running" || job?.status === "pending"
      ? "Healing in Progress"
      : job?.status === "needs_review"
        ? "Review Changes"
        : "Fix with Agent";

  return (
    <div className="ko-stack">
      <div className="flex items-center justify-between">
        <div>
          <p className="ko-text-sm ko-subtle">Scenario</p>
          <p className="font-semibold ko-text-strong">{scenarioName}</p>
        </div>
        <span className={`ko-health-badge ${toneClass}`}>{healthViewModel.healthScoreLabel}</span>
      </div>

      <div className="ko-card p-3 ko-text-xs ko-subtle">
        {healthViewModel.totalDocsLabel} • {healthViewModel.warningCount} warnings
      </div>

      {healthViewModel.hasIssues ? (
        <div className="ko-stack-sm">
          {healthViewModel.missingDocs.length > 0 && (
            <div className="ko-issue-block">
              <p className="ko-text-sm font-semibold ko-text-strong">
                Missing Docs <span className="ko-pill ko-pill-sm">agent</span>
              </p>
              <ul className="ko-list">
                {healthViewModel.missingDocs.map((doc) => (
                  <li key={doc}>{doc}</li>
                ))}
              </ul>
            </div>
          )}

          {healthViewModel.misplacedDocs.length > 0 && (
            <div className="ko-issue-block">
              <p className="ko-text-sm font-semibold ko-text-strong">
                Misplaced Docs <span className="ko-pill ko-pill-sm ko-pill-good">auto</span>
              </p>
              <ul className="ko-list">
                {healthViewModel.misplacedDocs.map((doc) => (
                  <li key={`${doc.actualPath}-${doc.expectedPath}`}>
                    <span className="ko-text-strong">{doc.actualPath}</span> → {doc.expectedPath}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {healthViewModel.extraDocs.length > 0 && (
            <div className="ko-issue-block">
              <p className="ko-text-sm font-semibold ko-text-strong">
                Extra Docs <span className="ko-pill ko-pill-sm">agent</span>
              </p>
              <ul className="ko-list">
                {healthViewModel.extraDocs.map((doc) => (
                  <li key={doc}>{doc}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      ) : (
        <div className="ko-panel p-4 ko-text-sm ko-muted">
          No documentation issues detected. Everything is aligned with the standard layout.
        </div>
      )}

      <div className="ko-card p-3 ko-stack-sm">
        <div className="flex items-center justify-between">
          <div>
            <p className="ko-text-sm font-semibold ko-text-strong">Heal Documentation</p>
            <p className="ko-text-xs ko-muted">
              {canHeal
                ? fixCategory === "all_auto"
                  ? "All issues can be auto-fixed by moving misplaced files."
                  : fixCategory === "mixed"
                    ? "Some issues can be auto-fixed; others need an agent."
                    : "Spawn an agent to repair documentation structure."
                : "All docs are aligned."}
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          {(fixCategory === "all_auto" || fixCategory === "mixed") && (
            <Button
              variant="outline"
              size="sm"
              disabled={isAutoFixing}
              onClick={() => autoFix()}
            >
              <Zap className="h-4 w-4 mr-2" />
              {isAutoFixing ? "Fixing..." : `Quick Fix (${healthViewModel.misplacedDocs.length} files)`}
            </Button>
          )}
          {(fixCategory === "all_agent" || fixCategory === "mixed") && (
            <Button
              variant="outline"
              size="sm"
              disabled={!canHeal && !job}
              onClick={() => setIsHealOpen(true)}
            >
              <Wrench className="h-4 w-4 mr-2" />
              {healButtonLabel}
            </Button>
          )}
          {fixCategory === "none" && job && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsHealOpen(true)}
            >
              <Wrench className="h-4 w-4 mr-2" />
              {healButtonLabel}
            </Button>
          )}
        </div>

        {autoFixError && (
          <div className="ko-alert ko-alert-danger">
            <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="ko-alert-title ko-text-danger-strong">Auto-fix error</p>
              <p className="ko-text-sm ko-text-danger-muted mt-1">{autoFixError.message}</p>
            </div>
          </div>
        )}

        {autoFixResult && (
          <div className="ko-stack-xs">
            <p className="ko-text-xs ko-muted">
              Health: {formatPercent(autoFixResult.health_before)} → {formatPercent(autoFixResult.health_after)}
            </p>
            {autoFixResult.moved.length > 0 && (
              <div className="ko-stack-xs">
                {autoFixResult.moved.map((m) => (
                  <div key={m.from_path} className="ko-text-xs flex items-center gap-1">
                    <MoveRight className="h-3 w-3" />
                    <span className="ko-text-strong">{m.from_path}</span> → {m.to_path}
                  </div>
                ))}
              </div>
            )}
            {autoFixResult.skipped.length > 0 && (
              <div className="ko-stack-xs">
                {autoFixResult.skipped.map((s) => (
                  <div key={s.from_path} className="ko-text-xs ko-muted">
                    Skipped {s.from_path}: {s.reason}
                  </div>
                ))}
              </div>
            )}
            <Button variant="outline" size="sm" onClick={() => { clearAutoFix(); onRefresh(); }}>
              Refresh Health
            </Button>
          </div>
        )}
      </div>

      {isHealOpen && (
        <div className="ko-modal-backdrop">
          <div className="ko-modal">
            <div className="ko-modal-header">
              <div>
                <p className="ko-text-sm ko-subtle">Documentation Healing</p>
                <p className="text-lg font-semibold ko-text-strong">{scenarioName}</p>
              </div>
              <Button variant="outline" size="sm" onClick={() => setIsHealOpen(false)}>
                Close
              </Button>
            </div>

            <div className="ko-modal-body ko-stack">
              {healError && (
                <div className="ko-alert ko-alert-danger">
                  <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
                  <div className="flex-1">
                    <p className="ko-alert-title ko-text-danger-strong">Healing error</p>
                    <p className="ko-text-sm ko-text-danger-muted mt-1">{healError.message}</p>
                  </div>
                </div>
              )}

              {!job && (
                <div className="ko-stack-sm">
                  <div className="ko-issue-block">
                    <p className="ko-text-sm font-semibold ko-text-strong">Select issues to fix</p>
                    {issueOptions.length === 0 ? (
                      <p className="ko-text-xs ko-muted mt-2">No issues detected.</p>
                    ) : (
                      <div className="ko-stack-xs mt-3">
                        {issueOptions.map((issue) => (
                          <label key={issue} className="ko-checkbox-row">
                            <input
                              type="checkbox"
                              checked={selectedIssues.includes(issue)}
                              onChange={() => toggleIssue(issue)}
                            />
                            <span>{issue}</span>
                          </label>
                        ))}
                      </div>
                    )}
                  </div>

                  <div className="ko-issue-block">
                    <p className="ko-text-sm font-semibold ko-text-strong">Options</p>
                    <div className="ko-stack-xs mt-3">
                      <label className="ko-checkbox-row">
                        <input
                          type="checkbox"
                          checked={autoApprove}
                          onChange={(event) => setAutoApprove(event.target.checked)}
                        />
                        <span>Auto-approve when health improves</span>
                      </label>
                      <label className="ko-checkbox-row">
                        <input
                          type="checkbox"
                          checked={dryRun}
                          onChange={(event) => setDryRun(event.target.checked)}
                        />
                        <span>Dry run (preview only)</span>
                      </label>
                    </div>
                  </div>

                  <div className="ko-modal-footer">
                    <Button
                      type="button"
                      onClick={handleStartHealing}
                      disabled={!scenarioName || selectedIssues.length === 0 || isBusy}
                    >
                      {isBusy ? "Starting..." : "Start Healing"}
                    </Button>
                  </div>
                </div>
              )}

              {job && (
                <div className="ko-stack-sm">
                  <div className="ko-card p-3 flex items-center justify-between">
                    <div>
                      <p className="ko-text-xs ko-muted">Status</p>
                      <p className="ko-text-sm font-semibold ko-text-strong">{job.status}</p>
                    </div>
                    {(job.status === "running" || job.status === "pending") && (
                      <span className="ko-inline-status">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        {job.progress || "Working..."}
                      </span>
                    )}
                    {job.status === "approved" && <CheckCircle className="h-5 w-5 ko-icon-strong" />}
                    {job.status === "rejected" && <XCircle className="h-5 w-5 ko-text-danger-strong" />}
                  </div>

                  <div className="ko-card p-3">
                    <p className="ko-text-xs ko-muted">Health</p>
                    <div className="flex items-center gap-4 mt-2">
                      <span className="ko-pill">Before {formatPercent(job.health_before)}</span>
                      <span className="ko-pill ko-pill-good">After {formatPercent(job.health_after)}</span>
                    </div>
                  </div>

                  {job.error && (
                    <div className="ko-alert ko-alert-danger">
                      <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
                      <div className="flex-1">
                        <p className="ko-alert-title ko-text-danger-strong">Job error</p>
                        <p className="ko-text-sm ko-text-danger-muted mt-1">{job.error}</p>
                      </div>
                    </div>
                  )}

                  {job.diff && job.diff.files.length > 0 && (
                    <div className="ko-stack-sm">
                      <div className="ko-issue-block">
                        <p className="ko-text-sm font-semibold ko-text-strong">Diff Summary</p>
                        <p className="ko-text-xs ko-muted mt-2">
                          {job.diff.summary || "Review the proposed documentation changes below."}
                        </p>
                      </div>
                      <div className="ko-diff-list">
                        {job.diff.files.map((file) => (
                          <div key={`${file.path}-${file.operation}`} className="ko-diff-card">
                            <div className="ko-diff-header">
                              <span className="ko-text-sm font-semibold">{file.path}</span>
                              <span className="ko-pill">{file.operation}</span>
                            </div>
                            <pre className="ko-diff-code">{file.diff || "Diff not available."}</pre>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {job.status === "needs_review" && (
                    <div className="ko-issue-block ko-stack-sm">
                      <p className="ko-text-sm font-semibold ko-text-strong">Approve changes</p>
                      <label className="ko-text-xs ko-muted">
                        Rejection reason (optional)
                        <input
                          className="ko-input mt-2"
                          value={rejectReason}
                          onChange={(event) => setRejectReason(event.target.value)}
                          placeholder="Reason for rejection"
                        />
                      </label>
                      <div className="flex gap-2">
                        <Button onClick={handleApprove} disabled={isBusy}>
                          Approve
                        </Button>
                        <Button onClick={handleReject} variant="danger" disabled={isBusy}>
                          Reject
                        </Button>
                      </div>
                    </div>
                  )}

                  {(job.status === "approved" || job.status === "rejected" || job.status === "failed") && (
                    <div className="ko-modal-footer">
                      <Button variant="outline" onClick={resetJob}>
                        Start New Healing
                      </Button>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
