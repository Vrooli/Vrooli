import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import {
  computeStaleness,
  deriveBaselineScope,
  resolveReferences,
  type StalenessReport,
} from "../../api/validation";
import { PlanSelect } from "../../components/PlanSelect";
import { StatusBadge } from "../../components/StatusBadge";
import { SectionPanel } from "../../components/Surfaces";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { type StringKey } from "../../consts/stringKey";
import { errorMessage } from "../../lib/errorMessage";
import { stalenessDescriptor } from "../../lib/planStatus";
import { useTranslation } from "../../i18n";
import {
  ReferenceResolution,
  type Reference,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import type { DeriveBaselineScopeResponse } from "@vrooli/proto-types/plan-manager/v1/validation/validation_pb";

const RESOLUTION_LABELS: Record<ReferenceResolution, StringKey> = {
  [ReferenceResolution.UNSPECIFIED]: strings.pages.validation.resolutionUnspecified,
  [ReferenceResolution.RESOLVED]: strings.pages.validation.resolutionResolved,
  [ReferenceResolution.UNRESOLVED]: strings.pages.validation.resolutionUnresolved,
  [ReferenceResolution.FUTURE]: strings.pages.validation.resolutionFuture,
  [ReferenceResolution.MISSING]: strings.pages.validation.resolutionMissing,
};

function DegradedNote() {
  const { t } = useTranslation();
  return (
    <p className="flex items-center gap-2 rounded-control bg-app-warning/10 px-3 py-2 text-xs text-app-warning">
      <AlertTriangle aria-hidden="true" className="h-4 w-4" />
      {t(strings.common.degraded)}
    </p>
  );
}

function CodeList({ items }: { items: readonly string[] }) {
  return (
    <ul className="flex flex-col gap-1">
      {items.map((c, i) => (
        <li key={`${c}-${i}`} className="break-words font-mono text-xs text-app-foreground">
          {c}
        </li>
      ))}
    </ul>
  );
}

/**
 * ValidationBoard — per-plan validation tooling. Each action resolves a
 * dependency-backed view and renders its result honestly: degraded (a composed
 * dependency was down) is surfaced, never hidden behind a false PASS.
 */
export function ValidationBoard() {
  const { t } = useTranslation();
  const [planId, setPlanId] = useState("");

  const [references, setReferences] = useState<{ refs: Reference[]; degraded: boolean } | null>(null);
  const [staleness, setStaleness] = useState<StalenessReport | null>(null);
  const [baseline, setBaseline] = useState<DeriveBaselineScopeResponse | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const disabled = planId.length === 0 || busy;

  const run = (fn: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    void (async () => {
      try {
        await fn();
      } catch (e) {
        setError(e);
      } finally {
        setBusy(false);
      }
    })();
  };

  return (
    <div className="flex flex-col gap-4">
      <SectionPanel title={t(strings.pages.validation.planLabel)} headingId="validation-plan-heading">
        <PlanSelect
          value={planId}
          onChange={setPlanId}
          label={t(strings.pages.validation.planLabel)}
          testId={selectors.validation.planSelect}
        />
        {error ? (
          <p role="alert" className="text-sm text-app-danger">
            {errorMessage(error, t)}
          </p>
        ) : null}
      </SectionPanel>

      <div className="grid gap-4 lg:grid-cols-2">
        <SectionPanel
          title={t(strings.pages.validation.referencesHeading)}
          headingId="validation-refs-heading"
          actions={
            <Button
              type="button"
              size="sm"
              variant="outline"
              data-testid={selectors.validation.resolveButton}
              disabled={disabled}
              onClick={() =>
                run(async () => {
                  const res = await resolveReferences(planId);
                  setReferences({ refs: res.references, degraded: res.degraded });
                })
              }
            >
              {t(strings.pages.validation.resolveReferences)}
            </Button>
          }
        >
          {references ? (
            <div data-testid={selectors.validation.references} className="flex flex-col gap-2">
              {references.degraded ? <DegradedNote /> : null}
              {references.refs.length === 0 ? (
                <p className="text-sm text-app-muted-foreground">
                  {t(strings.pages.validation.noReferences)}
                </p>
              ) : (
                <ul className="flex flex-col gap-2">
                  {references.refs.map((ref) => (
                    <li
                      key={ref.id}
                      className="flex flex-wrap items-center gap-2 rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
                    >
                      <span className="break-all font-mono text-xs text-app-foreground">
                        {ref.target}
                      </span>
                      <span className="text-xs text-app-muted-foreground">
                        {t(RESOLUTION_LABELS[ref.resolution])}
                      </span>
                      <StatusBadge descriptor={stalenessDescriptor(ref.staleness)} />
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ) : (
            <p className="text-sm text-app-muted-foreground">{t(strings.common.empty)}</p>
          )}
        </SectionPanel>

        <SectionPanel
          title={t(strings.pages.validation.stalenessHeading)}
          headingId="validation-staleness-heading"
          actions={
            <Button
              type="button"
              size="sm"
              variant="outline"
              data-testid={selectors.validation.stalenessButton}
              disabled={disabled}
              onClick={() =>
                run(async () => {
                  setStaleness(await computeStaleness(planId));
                })
              }
            >
              {t(strings.pages.validation.computeStaleness)}
            </Button>
          }
        >
          {staleness ? (
            <div data-testid={selectors.validation.staleness} className="flex flex-col gap-2">
              {staleness.degraded ? <DegradedNote /> : null}
              <div className="flex items-center gap-2 text-sm">
                <span className="text-app-muted-foreground">
                  {t(strings.pages.validation.overallStaleness)}
                </span>
                <StatusBadge descriptor={stalenessDescriptor(staleness.overall)} />
              </div>
            </div>
          ) : (
            <p className="text-sm text-app-muted-foreground">{t(strings.common.empty)}</p>
          )}
        </SectionPanel>

        <SectionPanel
          title={t(strings.pages.validation.baselineHeading)}
          headingId="validation-baseline-heading"
          actions={
            <Button
              type="button"
              size="sm"
              variant="outline"
              data-testid={selectors.validation.baselineButton}
              disabled={disabled}
              onClick={() =>
                run(async () => {
                  setBaseline(await deriveBaselineScope(planId));
                })
              }
            >
              {t(strings.pages.validation.deriveBaseline)}
            </Button>
          }
        >
          {baseline ? (
            <div data-testid={selectors.validation.baseline} className="flex flex-col gap-3 text-sm">
              <div>
                <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
                  {t(strings.pages.validation.baselineCommands)}
                </p>
                {baseline.commands.length > 0 ? (
                  <CodeList items={baseline.commands} />
                ) : (
                  <p className="text-sm text-app-muted-foreground">{t(strings.pages.validation.noBaseline)}</p>
                )}
              </div>
              {baseline.locations.length > 0 ? (
                <div>
                  <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
                    {t(strings.pages.validation.baselineLocations)}
                  </p>
                  <CodeList items={baseline.locations} />
                </div>
              ) : null}
            </div>
          ) : (
            <p className="text-sm text-app-muted-foreground">{t(strings.common.empty)}</p>
          )}
        </SectionPanel>

      </div>

      <SectionPanel title={t(strings.pages.validation.producerOwnedHeading)} headingId="validation-producer-owned-heading">
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.validation.producerOwnedDescription)}
        </p>
      </SectionPanel>
    </div>
  );
}
