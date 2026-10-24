import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  AuditStatus,
  type PortAuditResult,
} from "@vrooli/proto-types/tunnel-manager/v1/audit/audit_pb";

import { Button } from "../../components/ui/button";
import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge, type BadgeTone } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

import { auditClient } from "../../api/audit";

const AUDIT_KEY = ["audit"] as const;
type AuditFilter =
  | "all"
  | "violations"
  | "compliant"
  | "mismatch"
  | "missing-scenario"
  | "missing-port";

type AuditStatusKey = (typeof strings.audit.status)[keyof typeof strings.audit.status];

function auditStatusLabel(status: AuditStatus): AuditStatusKey {
  switch (status) {
    case AuditStatus.COMPLIANT:
      return strings.audit.status.compliant;
    case AuditStatus.MISMATCH:
      return strings.audit.status.mismatch;
    case AuditStatus.MISSING_SCENARIO:
      return strings.audit.status.missingScenario;
    case AuditStatus.MISSING_PORT:
      return strings.audit.status.missingPort;
    default:
      return strings.audit.status.unknown;
  }
}

function auditStatusTone(status: AuditStatus): BadgeTone {
  switch (status) {
    case AuditStatus.COMPLIANT:
      return "success";
    case AuditStatus.MISMATCH:
      return "danger";
    case AuditStatus.MISSING_SCENARIO:
    case AuditStatus.MISSING_PORT:
      return "warning";
    default:
      return "neutral";
  }
}

function auditRemediation(
  status: AuditStatus,
): (typeof strings.audit.remediation)[keyof typeof strings.audit.remediation] {
  switch (status) {
    case AuditStatus.COMPLIANT:
      return strings.audit.remediation.compliant;
    case AuditStatus.MISMATCH:
      return strings.audit.remediation.mismatch;
    case AuditStatus.MISSING_SCENARIO:
      return strings.audit.remediation.missingScenario;
    case AuditStatus.MISSING_PORT:
      return strings.audit.remediation.missingPort;
    default:
      return strings.audit.remediation.unknown;
  }
}

function matchesFilter(result: PortAuditResult, filter: AuditFilter): boolean {
  switch (filter) {
    case "violations":
      return result.status !== AuditStatus.COMPLIANT;
    case "compliant":
      return result.status === AuditStatus.COMPLIANT;
    case "mismatch":
      return result.status === AuditStatus.MISMATCH;
    case "missing-scenario":
      return result.status === AuditStatus.MISSING_SCENARIO;
    case "missing-port":
      return result.status === AuditStatus.MISSING_PORT;
    default:
      return true;
  }
}

/**
 * AuditPanel renders the port-compliance findings: each manifested route's
 * scenario service.json compared against the manifest's expected port. A
 * drifted or ranged UI port silently breaks ingress, so the violation count is
 * surfaced prominently.
 */
export function AuditPanel() {
  const { t } = useTranslation();
  const [filter, setFilter] = useState<AuditFilter>("all");

  const auditQuery = useQuery({ queryKey: AUDIT_KEY, queryFn: () => auditClient.runAudit({}) });

  const results = auditQuery.data?.results ?? [];
  const violationCount = auditQuery.data?.violationCount ?? 0;
  const compliantCount = results.filter((result) => result.status === AuditStatus.COMPLIANT).length;
  const filteredResults = results.filter((result) => matchesFilter(result, filter));
  const experienceState = auditQuery.isLoading ? "loading" : auditQuery.error ? "error" : results.length === 0 ? "empty" : "ready";

  return (
    <section data-testid={selectors.audit.panel} data-experience-surface="audit-results" data-experience-state={experienceState} className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h2 className="text-lg font-semibold">{t(strings.audit.heading)}</h2>
          {auditQuery.data && (
            <StatusBadge
              tone={violationCount > 0 ? "danger" : "success"}
              data-testid={selectors.audit.violationCount}
            >
              {t(strings.audit.violations, { count: violationCount })}
            </StatusBadge>
          )}
        </div>
        <Button
          data-testid={selectors.audit.runButton}
          disabled={auditQuery.isFetching}
          onClick={() => void auditQuery.refetch()}
        >
          {t(strings.audit.runButton)}
        </Button>
      </div>

      <div data-testid={selectors.audit.summary} className="grid gap-3 sm:grid-cols-3">
        <AuditSummaryStat
          label={t(strings.audit.totalRoutes)}
          value={auditQuery.error ? t(strings.common.notAvailable) : results.length}
        />
        <AuditSummaryStat
          label={t(strings.audit.compliantRoutes)}
          value={auditQuery.error ? t(strings.common.notAvailable) : compliantCount}
          tone={results.length > 0 && violationCount === 0 ? "success" : "neutral"}
        />
        <AuditSummaryStat
          label={t(strings.audit.routesToFix)}
          value={auditQuery.error ? t(strings.common.notAvailable) : violationCount}
          tone={violationCount > 0 ? "danger" : "success"}
        />
      </div>

      {results.length > 0 && (
        <label className="flex w-full flex-col gap-1 text-sm text-app-muted-foreground sm:max-w-xs">
          <span>{t(strings.audit.filterLabel)}</span>
          <select
            data-testid={selectors.audit.statusFilter}
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-3 py-2 text-app-foreground"
            value={filter}
            onChange={(event) => setFilter(event.target.value as AuditFilter)}
          >
            <option value="all">{t(strings.audit.filter.all)}</option>
            <option value="violations">{t(strings.audit.filter.violations)}</option>
            <option value="compliant">{t(strings.audit.filter.compliant)}</option>
            <option value="mismatch">{t(strings.audit.filter.mismatch)}</option>
            <option value="missing-scenario">{t(strings.audit.filter.missingScenario)}</option>
            <option value="missing-port">{t(strings.audit.filter.missingPort)}</option>
          </select>
        </label>
      )}

      <QueryState
        isLoading={auditQuery.isLoading}
        error={auditQuery.error}
        isEmpty={results.length === 0}
        loadingLabel={t(strings.audit.loading)}
        errorLabel={t(strings.audit.error)}
        emptyLabel={t(strings.audit.empty)}
      >
        {filteredResults.length === 0 ? (
          <p
            data-testid={selectors.queryState.empty}
            className="rounded-panel bg-app-surface-muted p-4 text-sm text-app-muted-foreground"
          >
            {t(strings.audit.filterEmpty)}
          </p>
        ) : (
          <div data-testid={selectors.audit.table} className="min-w-0 w-full max-w-full overflow-hidden rounded-panel border border-app-border">
            <div className="hidden overflow-x-auto md:block">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
                <tr>
                  <th className="px-3 py-2">{t(strings.audit.colScenario)}</th>
                  <th className="px-3 py-2">{t(strings.audit.colSubdomain)}</th>
                  <th className="px-3 py-2">{t(strings.audit.colExpected)}</th>
                  <th className="px-3 py-2">{t(strings.audit.colActual)}</th>
                  <th className="px-3 py-2">{t(strings.audit.colStatus)}</th>
                  <th className="px-3 py-2">{t(strings.audit.colDetail)}</th>
                  <th className="px-3 py-2">{t(strings.audit.colRemediation)}</th>
                </tr>
              </thead>
              <tbody>
                {filteredResults.map((result: PortAuditResult) => (
                  <tr
                    key={result.subdomain}
                    data-testid={selectors.audit.row}
                    className="border-b border-app-border last:border-0"
                  >
                    <td className="px-3 py-2 font-medium">{result.scenario}</td>
                    <td className="px-3 py-2">{result.subdomain}</td>
                    <td className="px-3 py-2 tabular-nums">{result.expectedPort}</td>
                    <td className="px-3 py-2 tabular-nums">{result.actualPort || "—"}</td>
                    <td className="px-3 py-2">
                      <StatusBadge tone={auditStatusTone(result.status)} data-testid={selectors.audit.statusBadge}>
                        {t(auditStatusLabel(result.status))}
                      </StatusBadge>
                    </td>
                    <td className="px-3 py-2 text-app-muted-foreground">{result.detail}</td>
                    <td
                      data-testid={selectors.audit.remediation}
                      className="max-w-sm px-3 py-2 text-app-muted-foreground"
                    >
                      {t(auditRemediation(result.status))}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            </div>
            <div className="grid gap-3 p-3 md:hidden">
              {filteredResults.map((result: PortAuditResult) => (
                <article key={result.subdomain} className="min-w-0 w-full max-w-full break-words rounded-control border border-app-border bg-app-surface-muted p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="break-words font-medium">{result.scenario}</p>
                      <p className="truncate text-xs text-app-muted-foreground">{result.subdomain}</p>
                    </div>
                    <StatusBadge tone={auditStatusTone(result.status)}>{t(auditStatusLabel(result.status))}</StatusBadge>
                  </div>
                  <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
                    <div><dt className="text-app-muted-foreground">{t(strings.audit.colExpected)}</dt><dd className="font-medium tabular-nums">{result.expectedPort}</dd></div>
                    <div><dt className="text-app-muted-foreground">{t(strings.audit.colActual)}</dt><dd className="font-medium tabular-nums">{result.actualPort || "—"}</dd></div>
                  </dl>
                  <p className="mt-3 text-sm text-app-muted-foreground">{result.detail}</p>
                  <p className="mt-2 text-sm text-app-muted-foreground">{t(auditRemediation(result.status))}</p>
                </article>
              ))}
            </div>
          </div>
        )}
      </QueryState>
    </section>
  );
}

function AuditSummaryStat({
  label,
  value,
  tone = "neutral",
}: {
  label: string;
  value: number | string;
  tone?: "success" | "danger" | "neutral";
}) {
  const toneClass = {
    success: "text-app-success",
    danger: "text-app-danger",
    neutral: "text-app-foreground",
  }[tone];

  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-3">
      <div className="text-xs uppercase text-app-muted-foreground">{label}</div>
      <div className={`mt-1 text-lg font-semibold tabular-nums ${toneClass}`}>{value}</div>
    </div>
  );
}
