import { useMemo, useState } from "react";

import { ScenarioPicker } from "../../components/ScenarioPicker";
import { StatusChip } from "../../components/StatusChip";
import { DiffView } from "../../components/DiffView";
import {
  findingDocPath,
  groupFindings,
  strippedCode,
  type FindingGroup,
} from "./findingsModel";
import { useApplyFix, useFindings, usePreviewFix } from "./useFindings";
import { useRecentScenarios } from "../../hooks/useRecentScenarios";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import type { ContractFinding } from "@vrooli/proto-types/business-health/v1/contract/contract_pb";

const severityIsError = (severity: string): boolean => severity.toLowerCase() === "error";

/**
 * Findings surface — validate a chosen scenario's business contract and list
 * its findings grouped by capability (or severity), each with its remediation
 * doc deep-link and, where a deterministic fixer exists, a preview-then-apply
 * affordance. Apply is always explicit and confirmed by a second click.
 */
export function FindingsPage() {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState("");
  const { recents, remember, clear } = useRecentScenarios();

  const query = useFindings(scenario);
  const response = query.data;
  const report = response?.report;

  const previewFix = usePreviewFix(scenario);
  const applyFix = useApplyFix(scenario);
  // The rule code whose fix panel is currently open (only one at a time).
  const [activeFix, setActiveFix] = useState<string | null>(null);

  const groups = useMemo<FindingGroup[]>(() => groupFindings(report), [report]);
  const findingCount = report?.findings.length ?? 0;

  const choose = (slug: string) => {
    setScenario(slug);
    setActiveFix(null);
    previewFix.reset();
    applyFix.reset();
    remember(slug);
  };

  const openPreview = (code: string) => {
    const rule = strippedCode(code);
    setActiveFix(rule);
    applyFix.reset();
    previewFix.mutate([rule]);
  };

  const confirmApply = () => {
    if (!activeFix) return;
    applyFix.mutate([activeFix]);
  };

  return (
    <section
      data-testid={selectors.pages.findings}
      aria-labelledby="findings-heading"
      className="flex min-h-0 flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="findings-heading" className="text-2xl font-semibold text-app-foreground">
          {t(strings.findings.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">{t(strings.findings.description)}</p>
      </header>

      <ScenarioPicker
        onSelect={choose}
        recents={recents}
        onClearRecents={clear}
        initialValue={scenario}
      />

      {scenario === "" && (
        <p className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground">
          {t(strings.common.chooseScenario)}
        </p>
      )}

      {scenario !== "" && query.isLoading && (
        <p
          data-testid={selectors.findings.loading}
          className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground"
        >
          {t(strings.findings.loading)}
        </p>
      )}

      {scenario !== "" && query.isError && (
        <div
          data-testid={selectors.findings.error}
          role="alert"
          className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
        >
          <p>{t(strings.findings.error)}</p>
          <p className="mt-1 text-xs opacity-80">{errorMessage(query.error, t)}</p>
        </div>
      )}

      {response && (
        <div className="flex flex-col gap-4">
          <p
            role="status"
            className="rounded-panel border border-app-border bg-app-surface p-3 text-sm text-app-foreground"
          >
            {t(strings.findings.verdict, { status: response.status })}
          </p>

          {response.degradedReason && (
            <p
              role="status"
              className="rounded-panel border border-app-warning/40 bg-app-warning/10 p-3 text-xs text-app-warning"
            >
              {t(strings.findings.degraded, { reason: response.degradedReason })}
            </p>
          )}

          {findingCount === 0 ? (
            <p
              data-testid={selectors.findings.empty}
              className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground"
            >
              {t(strings.findings.empty)}
            </p>
          ) : (
            <div data-testid={selectors.findings.list} className="flex flex-col gap-6">
              {groups.map((group) => (
                <section key={group.key} className="flex flex-col gap-3">
                  <h3 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
                    {group.label}
                  </h3>
                  <ul className="flex flex-col gap-3">
                    {group.findings.map((finding) => (
                      <FindingRow
                        key={finding.code}
                        finding={finding}
                        active={activeFix === strippedCode(finding.code)}
                        preview={previewFix}
                        apply={applyFix}
                        onPreview={() => openPreview(finding.code)}
                        onApply={confirmApply}
                      />
                    ))}
                  </ul>
                </section>
              ))}
            </div>
          )}
        </div>
      )}
    </section>
  );
}

interface FindingRowProps {
  readonly finding: ContractFinding;
  readonly active: boolean;
  readonly preview: ReturnType<typeof usePreviewFix>;
  readonly apply: ReturnType<typeof useApplyFix>;
  readonly onPreview: () => void;
  readonly onApply: () => void;
}

function FindingRow({ finding, active, preview, apply, onPreview, onApply }: FindingRowProps) {
  const { t } = useTranslation();
  const docPath = findingDocPath(finding.code);
  const isError = severityIsError(finding.severity);

  return (
    <li
      data-testid={selectors.findings.item({ code: finding.code })}
      className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-center gap-2">
        <StatusChip tone={isError ? "danger" : "warning"}>{finding.severity}</StatusChip>
        <span className="font-mono text-xs text-app-muted-foreground">
          {strippedCode(finding.code)}
        </span>
        <span className="min-w-0 flex-1 text-sm font-medium text-app-foreground">
          {finding.title}
        </span>
        {finding.autofixAvailable && (
          <StatusChip tone="info">{t(strings.findings.autofix)}</StatusChip>
        )}
      </div>

      {finding.message && <p className="text-sm text-app-foreground">{finding.message}</p>}

      {finding.location && (
        <p className="font-mono text-xs text-app-muted-foreground">
          {t(strings.findings.location, { location: finding.location })}
        </p>
      )}

      <div
        data-testid={selectors.findings.remediation}
        className="rounded-control border border-app-border bg-app-surface-muted p-3"
      >
        <h4 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
          {t(strings.findings.remediationHeading)}
        </h4>
        {finding.remediation && (
          <p className="mt-1 text-sm text-app-foreground">{finding.remediation}</p>
        )}
        <a
          data-testid={selectors.findings.docLink}
          href={docPath}
          className="mt-2 inline-block font-mono text-xs text-app-primary underline-offset-2 hover:underline"
        >
          {t(strings.findings.docLink, { path: docPath })}
        </a>
      </div>

      {finding.autofixAvailable && (
        <div className="flex flex-col gap-3">
          <button
            type="button"
            data-testid={selectors.findings.previewFix}
            onClick={onPreview}
            disabled={active && preview.isPending}
            className="self-start rounded-control border border-app-border bg-app-surface px-3 py-1.5 text-sm text-app-foreground hover:bg-app-surface-muted disabled:opacity-60"
          >
            {active && preview.isPending
              ? t(strings.findings.previewLoading)
              : t(strings.findings.previewFix)}
          </button>

          {active && !preview.isPending && (
            <FixPanel preview={preview} apply={apply} onApply={onApply} />
          )}
        </div>
      )}
    </li>
  );
}

interface FixPanelProps {
  readonly preview: ReturnType<typeof usePreviewFix>;
  readonly apply: ReturnType<typeof useApplyFix>;
  readonly onApply: () => void;
}

function FixPanel({ preview, apply, onApply }: FixPanelProps) {
  const { t } = useTranslation();

  if (preview.isError) {
    return (
      <p className="text-xs text-app-danger">{errorMessage(preview.error, t)}</p>
    );
  }

  const result = preview.data;
  if (!result) return null;

  if (result.kind === "unimplemented") {
    return (
      <p data-testid={selectors.findings.fixMessages} className="text-sm text-app-muted-foreground">
        {t(strings.findings.fixEmpty)}
      </p>
    );
  }

  const { candidates, messages } = result.response;

  if (candidates.length === 0) {
    return (
      <p data-testid={selectors.findings.fixMessages} className="text-sm text-app-muted-foreground">
        {t(strings.findings.noFixable)}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      {candidates.map((candidate, index) => (
        <DiffView
          key={`${candidate.filePath}-${index}`}
          data-testid={selectors.findings.fixDiff}
          before={candidate.before}
          after={candidate.after}
          path={candidate.filePath}
        />
      ))}

      {messages.length > 0 && (
        <ul
          data-testid={selectors.findings.fixMessages}
          className="flex flex-col gap-1 text-xs text-app-muted-foreground"
        >
          {messages.map((message, index) => (
            <li key={index}>{message}</li>
          ))}
        </ul>
      )}

      <button
        type="button"
        data-testid={selectors.findings.applyFix}
        onClick={onApply}
        disabled={apply.isPending}
        className="self-start rounded-control border border-app-primary bg-app-primary/10 px-3 py-1.5 text-sm font-medium text-app-primary hover:bg-app-primary/20 disabled:opacity-60"
      >
        {apply.isPending ? t(strings.findings.applying) : t(strings.findings.applyFix)}
      </button>

      {apply.isSuccess && (
        <p role="status" className="text-xs text-app-success">
          {t(strings.findings.fixApplied, { count: apply.data.candidates.length })}
        </p>
      )}
    </div>
  );
}
