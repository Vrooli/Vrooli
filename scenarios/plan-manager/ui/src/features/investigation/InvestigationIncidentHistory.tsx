import { useQuery } from "@tanstack/react-query";

import { listInvestigationIncidents, listInvestigationOccurrences } from "../../api/investigation";
import { Card, SectionPanel } from "../../components/Surfaces";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Read-only incident history. The panel consumes typed incident fields and
 * deliberately does not parse the diagnostic or trigger JSON projections.
 */
export function InvestigationIncidentHistory({ executionId, familyId }: { executionId?: string; familyId?: string } = {}) {
  const { t } = useTranslation();
  const incidents = useQuery({
    queryKey: ["investigation-incidents", executionId ?? "all", familyId ?? "all"],
    queryFn: () => listInvestigationIncidents({ executionId, familyId, limit: 50 }),
  });
  const occurrences = useQuery({
    queryKey: ["investigation-occurrences", executionId ?? "all"],
    queryFn: () => listInvestigationOccurrences({ executionId, limit: 50 }),
  });
  const suppressedOccurrences = occurrences.data?.filter((occurrence) => !occurrence.decision?.eligible) ?? [];

  return (
    <SectionPanel
      title={t(strings.pages.settings.investigationHistoryTitle)}
      headingId="investigation-incident-history-heading"
      description={t(strings.pages.settings.investigationHistoryDescription)}
    >
      {incidents.isPending ? <p className="text-sm text-app-muted-foreground">{t(strings.pages.settings.investigationHistoryLoading)}</p> : null}
      {incidents.isError ? <Card className="text-sm text-app-danger">{t(strings.pages.settings.investigationHistoryUnavailable)}</Card> : null}
      {incidents.data?.length === 0 ? <p className="text-sm text-app-muted-foreground">{t(strings.pages.settings.investigationHistoryEmpty)}</p> : null}
      {incidents.data && incidents.data.length > 0 ? (
        <div className="flex flex-col gap-2" data-testid="investigation-incident-history">
          {incidents.data.map((incident) => (
            <Card key={incident.incidentFingerprint} className="flex flex-col gap-2 text-sm">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="font-medium">{incident.state || t(strings.pages.settings.investigationUnknown)}</span>
                <span className="text-xs text-app-muted-foreground">{t(strings.pages.settings.investigationOccurrenceCount, { count: incident.occurrenceCount })}</span>
              </div>
              <div className="grid gap-2 sm:grid-cols-3">
                <div><span className="text-xs uppercase text-app-muted-foreground">{t(strings.pages.settings.investigationExecution)}</span><p className="break-all">{incident.executionId}</p></div>
                <div><span className="text-xs uppercase text-app-muted-foreground">{t(strings.pages.settings.investigationPhase)}</span><p className="break-all">{incident.phaseId} / {incident.phaseGeneration}</p></div>
                <div><span className="text-xs uppercase text-app-muted-foreground">{t(strings.pages.settings.investigationLink)}</span><p className="break-all">{incident.investigationId || t(strings.pages.settings.investigationNotLinked)}{incident.programStatus ? ` / ${incident.programStatus}` : ""}</p></div>
              </div>
              {incident.familyId ? <p className="text-xs text-app-muted-foreground">{t(strings.pages.settings.investigationFamily)}: {incident.familyId} · {t(strings.pages.settings.investigationSubjects)}: {incident.subjectExecutionIds.join(", ")}</p> : null}
              {incident.decision?.matches.length ? (
                <p className="text-xs text-app-muted-foreground">
                  {t(strings.pages.settings.investigationTrigger)}: {incident.decision.matches.map((match) => match.kind).join(", ")}
                </p>
              ) : null}
              {incident.occurrences.length > 0 ? (
                <ul className="flex flex-col gap-1 border-t border-app-border pt-2 text-xs text-app-muted-foreground">
                  {incident.occurrences.map((occurrence) => (
                    <li key={occurrence.occurrenceId}>
                      {occurrence.observedAt} · {occurrence.decision?.mode || t(strings.pages.settings.investigationUnknown)} · {occurrence.decision?.matches.map((match) => match.kind).join(", ") || t(strings.pages.settings.investigationUnknown)}{occurrence.decision?.queued ? ` · ${t(strings.pages.settings.investigationQueued)}` : ""}
                    </li>
                  ))}
                </ul>
              ) : null}
              {incident.dispatchError ? <p className="text-xs text-app-danger">{incident.dispatchError}</p> : null}
            </Card>
          ))}
        </div>
      ) : null}
      {suppressedOccurrences.length > 0 ? (
        <div className="flex flex-col gap-2" data-testid="investigation-suppressed-history">
          <h3 className="text-sm font-medium">{t(strings.pages.settings.investigationSuppressedTitle)}</h3>
          {suppressedOccurrences.map((occurrence) => (
            <Card key={occurrence.occurrenceId} className="text-sm">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="font-medium">{t(strings.pages.settings.investigationSuppressedReason)}</span>
                <span className="text-xs text-app-muted-foreground">{occurrence.observedAt}</span>
              </div>
              {occurrence.familyId || occurrence.sharedFailureRef ? <p className="break-all text-xs text-app-muted-foreground">{occurrence.familyId ? `${t(strings.pages.settings.investigationFamily)}: ${occurrence.familyId}` : ""}{occurrence.familyId && occurrence.sharedFailureRef ? " · " : ""}{occurrence.sharedFailureRef ? `${t(strings.pages.settings.investigationSharedFailure)}: ${occurrence.sharedFailureRef}` : ""}</p> : null}
              <p className="text-xs text-app-muted-foreground">{occurrence.decision?.reasons.join("; ") || t(strings.pages.settings.investigationUnknown)}</p>
            </Card>
          ))}
        </div>
      ) : null}
      {occurrences.data && suppressedOccurrences.length === 0 ? <p className="text-xs text-app-muted-foreground">{t(strings.pages.settings.investigationSuppressedEmpty)}</p> : null}
      <p className="text-xs text-app-muted-foreground">{t(strings.pages.settings.investigationNote)}</p>
    </SectionPanel>
  );
}
