import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, X } from "lucide-react";

import { listCandidateFindings, triageFinding } from "../../api/execution";
import { AsyncBoundary } from "../../components/AsyncBoundary";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { FindingTriage } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const triageKeys = {
  candidates: ["triage", "candidates"] as const,
};

/**
 * TriageBoard — candidate findings awaiting operator triage. Each finding can be
 * promoted to a real bug or dismissed as a false positive. Findings are filed as
 * CANDIDATE in-flow and never auto-promoted, so this is the human gate.
 */
export function TriageBoard() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const findings = useQuery({
    queryKey: triageKeys.candidates,
    queryFn: () => listCandidateFindings(),
  });

  const triage = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: FindingTriage }) =>
      triageFinding(id, decision),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: triageKeys.candidates });
    },
  });

  return (
    <AsyncBoundary
      isLoading={findings.isLoading}
      error={findings.error}
      isEmpty={(findings.data?.length ?? 0) === 0}
      testIdPrefix={selectors.triage.list}
      emptyLabel={t(strings.pages.triage.empty)}
    >
      <ul data-testid={selectors.triage.list} className="flex flex-col gap-2">
        {(findings.data ?? []).map((finding) => (
          <li
            key={finding.id}
            data-testid={selectors.triage.row({ id: finding.id })}
            className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4 sm:flex-row sm:items-start sm:justify-between"
          >
            <div className="flex min-w-0 flex-col gap-1">
              <p className="font-medium text-app-foreground">{finding.title}</p>
              {finding.detail ? (
                <p className="text-sm text-app-muted-foreground">{finding.detail}</p>
              ) : null}
              <div className="flex flex-wrap items-center gap-3 text-xs text-app-muted-foreground">
                {finding.phaseId ? (
                  <span>
                    {t(strings.pages.triage.phase)}{" "}
                    <span className="font-mono">{finding.phaseId}</span>
                  </span>
                ) : null}
                {finding.recordedAt ? (
                  <span>
                    {t(strings.pages.triage.recordedAt)}{" "}
                    {formatDate(new Date(finding.recordedAt), { dateStyle: "medium", timeStyle: "short" })}
                  </span>
                ) : null}
              </div>
            </div>
            <div className="flex shrink-0 gap-2">
              <Button
                type="button"
                size="sm"
                data-testid={selectors.triage.promote({ id: finding.id })}
                disabled={triage.isPending}
                onClick={() => triage.mutate({ id: finding.id, decision: FindingTriage.PROMOTED })}
              >
                <Check aria-hidden="true" className="me-1.5 h-4 w-4" />
                {t(strings.pages.triage.promote)}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                data-testid={selectors.triage.dismiss({ id: finding.id })}
                disabled={triage.isPending}
                onClick={() => triage.mutate({ id: finding.id, decision: FindingTriage.DISMISSED })}
              >
                <X aria-hidden="true" className="me-1.5 h-4 w-4" />
                {t(strings.pages.triage.dismiss)}
              </Button>
            </div>
          </li>
        ))}
      </ul>
    </AsyncBoundary>
  );
}
