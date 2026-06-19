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

/**
 * AuditPanel renders the port-compliance findings: each manifested route's
 * scenario service.json compared against the manifest's expected port. A
 * drifted or ranged UI port silently breaks ingress, so the violation count is
 * surfaced prominently.
 */
export function AuditPanel() {
  const { t } = useTranslation();

  const auditQuery = useQuery({ queryKey: AUDIT_KEY, queryFn: () => auditClient.runAudit({}) });

  const results = auditQuery.data?.results ?? [];
  const violationCount = auditQuery.data?.violationCount ?? 0;

  return (
    <section data-testid={selectors.audit.panel} className="flex flex-col gap-4">
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

      <QueryState
        isLoading={auditQuery.isLoading}
        error={auditQuery.error}
        isEmpty={results.length === 0}
        loadingLabel={t(strings.audit.loading)}
        errorLabel={t(strings.audit.error)}
        emptyLabel={t(strings.audit.empty)}
      >
        <div className="overflow-x-auto rounded-panel border border-app-border">
          <table data-testid={selectors.audit.table} className="w-full text-left text-sm">
            <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="px-3 py-2">{t(strings.audit.colScenario)}</th>
                <th className="px-3 py-2">{t(strings.audit.colSubdomain)}</th>
                <th className="px-3 py-2">{t(strings.audit.colExpected)}</th>
                <th className="px-3 py-2">{t(strings.audit.colActual)}</th>
                <th className="px-3 py-2">{t(strings.audit.colStatus)}</th>
                <th className="px-3 py-2">{t(strings.audit.colDetail)}</th>
              </tr>
            </thead>
            <tbody>
              {results.map((result: PortAuditResult) => (
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
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </QueryState>
    </section>
  );
}
