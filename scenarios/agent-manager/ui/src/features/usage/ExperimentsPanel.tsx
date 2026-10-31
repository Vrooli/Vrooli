import { Link } from "react-router-dom";
import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageExperiment, UsageExperimentArm } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { useExperimentActions, useUsageExperiments } from "./api";
import { assumedAcceptanceText, experimentSentence, horizonText, ratioText, VERDICT_TEXT, verdictOf } from "./experiments";
import { EVIDENCE_LABELS, known, type EvidenceDesign } from "./evidence";
import { startErrorText } from "./startErrorText";

function ArmLine({ arm }: { arm: UsageExperimentArm }) {
  const cost = known(arm.pointsPerAccepted) ? `${arm.pointsPerAccepted.value.toFixed(2)} pts (${arm.pointsPerAccepted.low.toFixed(2)} to ${arm.pointsPerAccepted.high.toFixed(2)})` : "unknown";
  const rework = known(arm.reworkRate) ? `${Math.round(arm.reworkRate.value * 100)}% (${Math.round(arm.reworkRate.low * 100)}% to ${Math.round(arm.reworkRate.high * 100)}%)` : "unknown";
  return (
    <tr data-testid={`experiment-arm-${arm.arm}`}>
      <th scope="row" className="py-1 pr-3 text-left font-medium capitalize">{arm.arm}</th>
      <td className="py-1 pr-3 tabular-nums">{Number(arm.units)} launches, {Number(arm.accepted)} accepted</td>
      <td className="py-1 pr-3 tabular-nums">{cost} per accepted outcome</td>
      <td className="py-1 tabular-nums">rework {rework}{Number(arm.exceptions) > 0 ? `, ${Number(arm.exceptions)} ${Number(arm.exceptions) === 1 ? "exception" : "exceptions"}` : ""}</td>
    </tr>
  );
}

function Recommendation({ experiment }: { experiment: UsageExperiment }) {
  const { propose } = useExperimentActions();
  const r = experiment.recommendation;
  if (!r?.available) return null;
  const decided = r.decisionState !== "";
  return (
    <section className="flex flex-col gap-2 rounded-md border border-border p-3" data-testid="experiment-recommendation" aria-label="Role recommendation">
      <h4 className="m-0 text-sm font-semibold">Recommendation: make {r.toModel} the default for {r.role}</h4>
      <p className="m-0 text-sm text-muted-foreground">{r.evidence} This is an ask, not a change. Nothing is applied until a verified person approves, and even then Agent Manager does not edit resource files.</p>
      {r.policyOps.length > 0 ? (
        <div>
          <p className="m-0 text-xs text-muted-foreground" data-testid="experiment-policy-file">The exact change, in <code>{r.policyFile}</code>:</p>
          <pre className="m-0 overflow-x-auto rounded bg-muted/40 p-2 text-xs" data-testid="experiment-policy-ops">{r.policyOps.join("\n")}</pre>
        </div>
      ) : null}
      {r.policyNote ? <p className="m-0 text-xs text-muted-foreground">{r.policyNote}</p> : null}
      {decided ? (
        <p className="m-0 text-sm" data-testid="experiment-decision" role="status">Decision {r.decisionState}.{r.decisionNote ? ` ${r.decisionNote}` : ""}</p>
      ) : (
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="secondary" className="min-h-11 sm:min-h-8" disabled={propose.isPending || propose.isSuccess} onClick={() => propose.mutate(experiment.modeId)} data-testid="experiment-propose">
            {propose.isSuccess ? "Approval requested" : propose.isPending ? "Asking" : "Ask me to approve this"}
          </Button>
          <span className="text-xs text-muted-foreground">Raises a decision for a verified human.</span>
        </div>
      )}
      {propose.error ? <p role="alert" className="m-0 text-sm text-destructive" data-testid="experiment-propose-error">{startErrorText(propose.error, "ask for this approval")}</p> : null}
    </section>
  );
}

function ExperimentCard({ experiment }: { experiment: UsageExperiment }) {
  const verdict = verdictOf(experiment.verdict);
  const design: EvidenceDesign = experiment.design in EVIDENCE_LABELS ? (experiment.design as EvidenceDesign) : "observational";
  const assumed = assumedAcceptanceText(experiment);
  const state = experiment.paused ? `Paused: ${experiment.pausedReason}` : experiment.active ? "Running" : "Ended";
  return (
    <article className="flex flex-col gap-2 border-t border-border pt-3 first:border-t-0 first:pt-0" data-testid="experiment-card">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge tone={design === "observational" ? "neutral" : "success"}><span data-testid="experiment-design">{EVIDENCE_LABELS[design]}</span></StatusBadge>
        <StatusBadge tone={verdict === "not_enough_data" || verdict === "withheld" ? "neutral" : "info"}><span data-testid="experiment-verdict">{VERDICT_TEXT[verdict]}</span></StatusBadge>
        <span className="text-xs text-muted-foreground">{experiment.role} on {experiment.runner}, {experiment.splitPercent}% to {experiment.challengerModel || "the challenger"} · {state}{experiment.sessionsOptIn ? " · your own sessions included in day blocks" : ""}</span>
      </div>
      <p className="m-0 text-base font-medium" data-testid="experiment-sentence">{experimentSentence(experiment)}</p>
      <p className="m-0 text-sm text-muted-foreground" data-testid="experiment-reason">{experiment.reason}</p>
      {assumed ? <p className="m-0 text-xs text-muted-foreground" data-testid="experiment-assumed">{assumed}</p> : null}
      <table className="w-full text-xs">
        <tbody>
          {experiment.incumbent ? <ArmLine arm={experiment.incumbent} /> : null}
          {experiment.challenger ? <ArmLine arm={experiment.challenger} /> : null}
        </tbody>
      </table>
      {known(experiment.costRatio) ? (
        <p className="m-0 text-xs text-muted-foreground" data-testid="experiment-stats">
          Challenger over incumbent cost per accepted outcome: {ratioText(experiment.costRatio.value)} (90% range {ratioText(experiment.costRatio.low)} to {ratioText(experiment.costRatio.high)}), permutation test p={experiment.costPValue.toFixed(3)}.
          {known(experiment.reworkDifference) ? ` Rework differs by ${(experiment.reworkDifference.value * 100).toFixed(0)} points (${(experiment.reworkDifference.low * 100).toFixed(0)} to ${(experiment.reworkDifference.high * 100).toFixed(0)}), p=${experiment.reworkPValue.toFixed(3)}.` : ""}
        </p>
      ) : null}
      {verdict === "not_enough_data" ? (
        <p className="m-0 text-sm" data-testid="experiment-needed">
          {Number(experiment.needAccepted) > 0 ? `Still needed: ${Number(experiment.needAccepted)} more accepted outcomes in the thinner arm. ` : ""}
          Expected time to a verdict: {horizonText(experiment.plan?.detect)} for a halving of cost{experiment.plan?.varianceAssumed ? " (variance assumed: too little finished work to measure it)" : ""}.
        </p>
      ) : null}
      <Recommendation experiment={experiment} />
    </article>
  );
}

/** Usage › Evidence: every Learn comparison, its design, verdict or the data still needed, and the role recommendation. */
export function ExperimentsPanel() {
  const query = useUsageExperiments();
  const experiments = query.data?.experiments ?? [];
  return (
    <section className="flex flex-col gap-3 rounded-lg border border-border bg-card p-4 sm:p-5" data-testid="evidence-experiments" aria-label="Learn comparisons">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h3 className="m-0 text-base font-semibold">Learn comparisons</h3>
          <p className="m-0 text-xs text-muted-foreground">Learn mode assigns a role's new launches between its usual model and a cheaper one. Each card says how its launches were assigned and what its acceptance rests on; only an assignment on record per launch is called randomized.</p>
        </div>
        <Link to="/usage/learn" className="inline-flex min-h-11 items-center text-sm text-primary hover:underline sm:min-h-8" data-testid="evidence-learn-setup">Set up Learn mode</Link>
      </div>
      {query.isLoading ? <p className="m-0 text-sm text-muted-foreground">Reading comparisons.</p> : null}
      {query.error ? <p className="m-0 text-sm text-muted-foreground" data-testid="experiments-unavailable">Comparisons are not available right now.</p> : null}
      {!query.isLoading && !query.error && experiments.length === 0 ? <p className="m-0 text-sm text-muted-foreground" data-testid="experiments-none">No comparison has run. Set one up to find out properly whether a cheaper model earns its place for a role.</p> : null}
      {experiments.map((experiment) => <ExperimentCard key={experiment.modeId} experiment={experiment} />)}
    </section>
  );
}
