import { useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { DataState } from "../../components/feedback/DataState";
import { EVIDENCE_WINDOWS, useOperatorSignalsDefined, useUsageOutcomes, useUsageParity, type EvidenceWindow } from "./api";
import { BreakEvenBar } from "./BreakEvenBar";
import { EvidenceChart } from "./EvidenceChart";
import { ModelComparison, OperatorSignals, QualityByMode, SessionCoverageList } from "./EvidenceTables";
import { breakEven, EVIDENCE_LABELS, evidenceView, type EvidenceState } from "./evidence";
import { ExperimentsPanel } from "./ExperimentsPanel";
import { ModeBeforeAfter } from "./ModeBeforeAfter";

const WINDOW_TEXT: Record<EvidenceWindow, { chip: string; phrase: string }> = {
  last_7d: { chip: "7 days", phrase: "last 7 days" },
  last_30d: { chip: "30 days", phrase: "last 30 days" },
};

const PILL_TONE: Record<EvidenceState, StatusTone> = { no_mode: "neutral", not_enough_data: "neutral", observational: "warning", randomized: "success", day_block_randomized: "success" };

function Section({ title, hint, children, testId }: { title: string; hint?: string; children: React.ReactNode; testId: string }) {
  return (
    <Card className="flex flex-col gap-3 p-4 sm:p-5" data-testid={testId}>
      <div>
        <h3 className="m-0 text-base font-semibold">{title}</h3>
        {hint ? <p className="m-0 text-xs text-muted-foreground">{hint}</p> : null}
      </div>
      {children}
    </Card>
  );
}

/** Usage › Evidence: did a mode save usage, what did it cost, and how sure is that. Conclusion first, charts under it. */
export function UsageEvidence() {
  const [window, setWindow] = useState<EvidenceWindow>("last_30d");
  const byModel = useUsageOutcomes(window, "model");
  const byMode = useUsageOutcomes(window, "mode");
  const parity = useUsageParity(window);
  const signals = useOperatorSignalsDefined();
  const response = byModel.data;
  const view = response ? evidenceView(response, WINDOW_TEXT[window].phrase) : undefined;
  const breakEvenValue = breakEven(view?.headline);
  return (
    <div className="flex flex-col gap-4" data-testid="usage-evidence">
      <div className="flex flex-wrap items-center gap-2" role="group" aria-label="Period">
        {EVIDENCE_WINDOWS.map((id) => (
          <Button key={id} variant={id === window ? "primary" : "secondary"} className="min-h-11 sm:min-h-8" aria-pressed={id === window} onClick={() => setWindow(id)} data-testid={`evidence-window-${id}`}>{WINDOW_TEXT[id].chip}</Button>
        ))}
        <span className="text-xs text-muted-foreground">Same figures as <code>agent-manager usage outcomes --window {window}</code></span>
      </div>
      <DataState
        loading={byModel.isLoading}
        error={byModel.error}
        isEmpty={!response || !view}
        subject="Evidence"
        onRetry={() => void byModel.refetch()}
        empty={<EmptyState title="No evidence yet" description="The run ledger has not reported anything for this period." />}
      >
        {response && view ? (
          <>
            <Card className="flex flex-col gap-3 p-4 sm:p-5" data-testid="evidence-conclusion" aria-live="polite">
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge tone={PILL_TONE[view.state]}><span data-testid="evidence-pill">{EVIDENCE_LABELS[view.state]}</span></StatusBadge>
                <span className="text-xs text-muted-foreground">{WINDOW_TEXT[window].phrase}, {Number(response.units)} work units, {Number(response.completedUnits)} only finished with acceptance unknown, {Number(response.unknownUnits)} with an unknown outcome</span>
              </div>
              <p className="m-0 text-base font-medium sm:text-lg" data-testid="evidence-text">{view.conclusion}</p>
              <p className="m-0 text-sm text-muted-foreground" data-testid="evidence-strength">{view.strength}</p>
              {view.state === "no_mode" ? (
                <div className="flex flex-wrap items-center gap-2">
                  <Link to="/usage" className="inline-flex min-h-11 items-center text-sm text-primary hover:underline sm:min-h-8" data-testid="evidence-start-mode">Start a mode from Overview</Link>
                </div>
              ) : null}
              {response.notes.map((note) => <p key={note} className="m-0 text-xs text-muted-foreground">{note}</p>)}
            </Card>

            <ExperimentsPanel />

            {breakEvenValue ? (
              <Section title="Break-even" hint="Price and extra use on the same units. Reference tokens weigh input 1, cached 0.1, cache writes 1.25 and output 5 for every model: a common yardstick, not any model's price." testId="evidence-break-even">
                <BreakEvenBar value={breakEvenValue} />
              </Section>
            ) : (
              <Section title="Break-even" testId="evidence-break-even">
                <p className="m-0 text-sm text-muted-foreground" data-testid="break-even-missing">Needs two models of one runner on one path with enough accepted outcomes. Until then no price ratio is compared with measured use.</p>
              </Section>
            )}

            <Section title="Usage over time" hint="Quota points by the day a run started, with mode periods shaded" testId="evidence-over-time">
              <EvidenceChart series={response.series} periods={response.modePeriods} windowStart={response.windowStart} windowEnd={response.windowEnd} />
            </Section>

            <Section title="Models compared" hint="By path, because an operator can correct only work they watch. 90% ranges from resampling work units. Accepted needs an approval, a passing validation or a positive verdict; a unit that only finished counts toward the weaker cost per completed." testId="evidence-models">
              <ModelComparison rows={response.rows} />
            </Section>

            <div className="grid items-start gap-4 lg:grid-cols-2">
              <Section title="Quality, mode on and off" hint="Redone work and how often you stepped in" testId="evidence-quality">
                <DataState loading={byMode.isLoading} error={byMode.error} isEmpty={!byMode.data} subject="Quality" compact onRetry={() => void byMode.refetch()} empty={<p className="m-0 text-sm text-muted-foreground">Not available.</p>}>
                  <QualityByMode rows={byMode.data?.rows ?? []} periods={response.modePeriods} />
                </DataState>
                <h4 className="m-0 text-sm font-medium">Operator signals</h4>
                <OperatorSignals defined={signals.data} />
              </Section>
              <Section title="How much of your work this covers" hint="By how the session started" testId="evidence-coverage">
                <DataState loading={parity.isLoading} error={parity.error} isEmpty={!parity.data} subject="Coverage" compact onRetry={() => void parity.refetch()} empty={<p className="m-0 text-sm text-muted-foreground">Not available.</p>}>
                  <SessionCoverageList rows={parity.data?.rows ?? []} />
                </DataState>
              </Section>
            </div>

            <Section title="Each time a mode ran" hint="The span before it, as long as the mode lasted, against the mode itself" testId="evidence-before-after">
              <ModeBeforeAfter periods={response.modePeriods} now={response.generatedAt} />
            </Section>
          </>
        ) : null}
      </DataState>
    </div>
  );
}
