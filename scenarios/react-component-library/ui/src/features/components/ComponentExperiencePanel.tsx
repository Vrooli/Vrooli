/** @vrooliComponentSource feedback.status-badge */
import type { ComponentExperience, ComponentExperienceEvidence } from "../../api/components";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

function evidenceForClaim(evidence: ComponentExperienceEvidence[], claimID: string) {
  return evidence.find((item) => item.claimId === claimID);
}

function tierTone(tier: string): "success" | "warning" | "info" {
  if (tier === "machine") return "success";
  if (tier === "manual") return "warning";
  return "info";
}

function verdictTone(verdict?: string): "success" | "warning" | "danger" | "neutral" {
  if (verdict === "passed") return "success";
  if (verdict === "failed") return "danger";
  if (verdict === "skipped") return "warning";
  return "neutral";
}

function evidenceIsStale(evidence?: ComponentExperienceEvidence) {
  if (!evidence?.checkedAt) return false;
  const checkedAt = Date.parse(evidence.checkedAt);
  return Number.isFinite(checkedAt) && Date.now() - checkedAt > 5 * 60_000;
}

export function ComponentExperiencePanel({
  experience,
  isLoading,
  isError,
}: {
  experience?: ComponentExperience;
  isLoading: boolean;
  isError?: boolean;
}) {
  const { t } = useTranslation();
  if (isLoading)
    return (
      <section
        data-testid="component-experience-panel"
        aria-labelledby="component-experience-title"
        className="rounded-lg border border-app-border bg-app-surface-muted p-space-xs"
      >
        <h3 id="component-experience-title" className="font-medium">
          {t(strings.componentDetail.experience.title)}
        </h3>
        <p className="mt-space-2xs text-xs text-app-muted-foreground">
          {t(strings.componentDetail.experience.loading)}
        </p>
      </section>
    );
  if (isError)
    return (
      <section
        data-testid="component-experience-panel"
        aria-labelledby="component-experience-title"
        className="rounded-lg border border-app-border bg-app-surface-muted p-space-xs"
      >
        <h3 id="component-experience-title" className="font-medium">
          {t(strings.componentDetail.experience.title)}
        </h3>
        <p role="alert" className="mt-space-2xs text-xs text-app-warning">
          {t(strings.componentDetail.experience.loadError)}
        </p>
      </section>
    );
  if (!experience || experience.evidenceStatus === "not-configured")
    return (
      <section
        data-testid="component-experience-panel"
        aria-labelledby="component-experience-title"
        className="rounded-lg border border-app-border bg-app-surface-muted p-space-xs"
      >
        <h3 id="component-experience-title" className="font-medium">
          {t(strings.componentDetail.experience.title)}
        </h3>
        <p className="mt-space-2xs text-xs text-app-muted-foreground">
          {experience?.evidenceMessage || t(strings.componentDetail.experience.notConfigured)}
        </p>
      </section>
    );
  const unavailable = experience.evidenceStatus !== "available";
  return (
    <section
      data-testid="component-experience-panel"
      aria-labelledby="component-experience-title"
      className="rounded-lg border border-app-border bg-app-surface-muted p-space-xs"
    >
      <div className="flex flex-wrap items-center justify-between gap-space-2xs">
        <div>
          <h3 id="component-experience-title" className="font-medium">
            {t(strings.componentDetail.experience.title)}
          </h3>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">{experience.purpose}</p>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            {t(strings.componentDetail.experience.identity, {
              contract: experience.contractId,
              version:
                experience.version || t(strings.componentDetail.experience.versionUnavailable),
            })}
          </p>
        </div>
        <StatusBadge tone={unavailable ? "warning" : "info"}>
          {unavailable
            ? t(strings.componentDetail.experience.unavailable)
            : t(strings.componentDetail.experience.records, { count: experience.evidence.length })}
        </StatusBadge>
      </div>
      {experience.evidenceMessage && (
        <p className="mt-space-2xs text-xs text-app-muted-foreground">
          {experience.evidenceMessage}
        </p>
      )}
      <div className="mt-space-xs">
        <h4 className="text-xs font-medium">{t(strings.componentDetail.experience.states)}</h4>
        <ul className="mt-space-3xs flex flex-wrap gap-space-2xs">
          {experience.states.map((state) => (
            <li key={state.id}>
              <StatusBadge tone="neutral">
                {state.id} · {state.exampleName}
              </StatusBadge>
            </li>
          ))}
        </ul>
      </div>
      <ul
        aria-label={t(strings.componentDetail.experience.claims)}
        className="mt-space-xs space-y-space-2xs"
      >
        {experience.claims.map((claim) => {
          const evidence = evidenceForClaim(experience.evidence, claim.id);
          const stale = evidenceIsStale(evidence);
          return (
            <li
              key={claim.id}
              className="rounded-control border border-app-border bg-app-background p-space-2xs"
            >
              <div className="flex flex-wrap items-center gap-space-2xs">
                <span className="text-xs font-medium">{claim.statement}</span>
                <StatusBadge tone={tierTone(claim.tier)}>{claim.tier}</StatusBadge>
                <StatusBadge tone={verdictTone(evidence?.verdict)}>
                  {evidence?.verdict || t(strings.componentDetail.experience.unproven)}
                </StatusBadge>
                {stale ? (
                  <StatusBadge tone="warning">
                    {t(strings.componentDetail.experience.stale)}
                  </StatusBadge>
                ) : null}
              </div>
              <p className="mt-space-3xs text-xs text-app-muted-foreground">
                {t(strings.componentDetail.experience.statesPrefix, {
                  states:
                    claim.states.join(", ") || t(strings.componentDetail.experience.allStates),
                })}
              </p>
              {evidence && (
                <p className="mt-space-3xs text-xs text-app-muted-foreground">
                  {evidence.viewport ? `${evidence.viewport} · ` : ""}
                  {evidence.checkedAt ||
                    t(strings.componentDetail.experience.captureTimeUnavailable)}
                  {evidence.captureRef &&
                  (evidence.captureRef.startsWith("http://") ||
                    evidence.captureRef.startsWith("https://")) ? (
                    <>
                      {" "}
                      ·{" "}
                      <a
                        href={evidence.captureRef}
                        className="text-app-primary underline-offset-2 hover:underline"
                      >
                        {t(strings.componentDetail.experience.openCapture)}
                      </a>
                    </>
                  ) : evidence.captureRef ? (
                    <>
                      {" "}
                      · <span className="font-mono">{evidence.captureRef}</span>
                    </>
                  ) : null}
                </p>
              )}
            </li>
          );
        })}
      </ul>
    </section>
  );
}
