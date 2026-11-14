import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueries } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card } from "@vrooli/react-component-library/Card/1";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { RadioGroup } from "@vrooli/react-component-library/RadioGroup/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageExperimentPlan } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { CALL_OPTIONS, useExperimentActions, useUsageOutcomes } from "./api";
import { rowsByKey } from "./evidence";
import { headline, horizonText, isExemptRole, learnRoles, paceText, shareWords, type LearnRole } from "./experiments";
import { measuresClient } from "./measuresClient";
import { startErrorText } from "./startErrorText";

const PACE_DAYS = 30;
const SPLITS = [
  { value: "50", label: "Half and half", description: "Fastest answer" },
  { value: "20", label: "Mostly usual", description: "80/20: slower, less saved" },
] as const;

const roleId = (role: LearnRole) => `${role.runner}|${role.key}`;

/**
 * Usage › Learn mode setup (mockup F): which roles to test, the split, and an
 * honest, ranged time to an answer read from each role's own pace and variance
 * before anything starts. Starting is a verified-operator write the API enforces.
 */
export function LearnSetupPage() {
  const navigate = useNavigate();
  const outcomes = useUsageOutcomes("last_30d", "role");
  const roles = useMemo(() => learnRoles(rowsByKey(outcomes.data?.rows ?? []), PACE_DAYS), [outcomes.data]);
  const [chosen, setChosen] = useState<ReadonlySet<string>>(new Set());
  const [split, setSplit] = useState<string>("50");
  const [sessions, setSessions] = useState(false);
  const { start } = useExperimentActions();
  const selected = roles.filter((role) => chosen.has(roleId(role)) && !role.exempt);
  const percent = Number(split);

  const plans = useQueries({
    queries: selected.map((role) => ({
      queryKey: ["usage", "experiments", "plan", role.runner, role.key, percent] as const,
      queryFn: () => measuresClient.planUsageExperiment({ runner: role.runner, role: role.key, splitPercent: percent }, CALL_OPTIONS),
      staleTime: 300_000,
      retry: false,
    })),
  });
  const planned = plans.flatMap((query) => (query.data?.plan ? [query.data.plan] : []));
  const planning = plans.some((query) => query.isLoading);
  const unreachable = planned.some((plan) => !plan.detect?.reached && Number(plan.pilotUnits) > 0);
  const lead = unreachable ? { text: "No verdict is reachable yet: a chosen role's finished work records no accepted outcome.", open: true, bar: undefined } : headline(planned);
  const toggle = (role: LearnRole) => setChosen((current) => {
    const next = new Set(current);
    if (next.has(roleId(role))) next.delete(roleId(role));
    else next.add(roleId(role));
    return next;
  });

  const submit = async () => {
    // One comparison per role: each has its own pace, variance and verdict.
    for (const role of selected) {
      try {
        await start.mutateAsync({ runner: role.runner, role: role.key, split: percent, sessions });
      } catch {
        return; // the error is shown; roles already started stay started
      }
    }
    navigate("/usage?tab=evidence");
  };

  return (
    <div className="h-full overflow-y-auto overflow-x-hidden p-4 sm:p-6">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-4" data-testid="learn-setup">
        <PageHeader title="Find out properly, by chance assignment" description="New launches of the chosen roles are split by chance between the usual model and the cheaper one, so the work itself cannot have picked the model. The result is read once, at a planned size, and is only as good as how acceptance is recorded. The wait is estimated, not promised." />
        <Card className="flex flex-col gap-4 p-4 sm:p-5">
          <div className="flex items-center justify-between gap-2">
            <h2 className="m-0 text-base font-semibold">Set up Learn mode</h2>
            <StatusBadge tone="neutral">Not started</StatusBadge>
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="flex min-w-0 flex-col gap-4">
              <section className="flex flex-col gap-2" aria-label="Which roles to test">
                <h3 className="m-0 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Which roles to test</h3>
                {outcomes.isLoading ? <p className="m-0 text-sm text-muted-foreground">Reading your roles.</p> : null}
                {outcomes.error ? <p role="alert" className="m-0 text-sm text-muted-foreground" data-testid="learn-roles-error">Your roles could not be read, so nothing can be chosen yet.</p> : null}
                {!outcomes.isLoading && !outcomes.error && roles.length === 0 ? <p className="m-0 text-sm text-muted-foreground" data-testid="learn-no-roles">No role has launched managed work in the last 30 days, so there is nothing to compare yet.</p> : null}
                {roles.map((role) => (
                  <div key={roleId(role)} className="rounded-md border border-border p-2" data-testid="learn-role">
                    <Checkbox
                      label={role.key}
                      description={role.exempt ? "Left out: reviews, judgment and fallback keep their model by default" : `${role.runner}, ${paceText(role.perWeek)} (sample from the last 30 days)`}
                      checked={chosen.has(roleId(role)) && !role.exempt}
                      disabled={role.exempt}
                      data-testid={`learn-role-${role.key}`}
                      onCheckedChange={() => toggle(role)}
                    />
                  </div>
                ))}
              </section>
              <RadioGroup label="Split" variant="card" value={split} onValueChange={setSplit} options={SPLITS.map((option) => ({ value: option.value, label: option.label, description: option.description, testId: `learn-split-${option.value}` }))} />
              <Checkbox
                label="Also include my own sessions"
                description="Sessions you start through Vrooli join as randomized day blocks, labelled when they launch. You can still pick a model yourself once; it is counted and stays in its block."
                checked={sessions}
                data-testid="learn-sessions"
                onCheckedChange={(next) => setSessions(next === true)}
              />
            </div>
            <div className="flex min-w-0 flex-col gap-3">
              <section className="flex flex-col gap-2 rounded-md border border-border p-3" data-testid="learn-estimate" aria-live="polite" aria-label="How long until an answer">
                <h3 className="m-0 text-xs font-semibold uppercase tracking-wide text-muted-foreground">How long until an answer</h3>
                <p className="m-0 text-base font-medium" data-testid="learn-headline">{planning ? "Estimating from your recent work." : lead.text}</p>
                {lead.bar && !planning ? (
                  <div className="relative h-1.5 rounded bg-muted" aria-hidden="true">
                    <div className="absolute inset-y-0 rounded bg-primary" style={{ left: `${lead.bar.from * 100}%`, width: `${Math.max(4, (lead.bar.to - lead.bar.from) * 100)}%` }} />
                  </div>
                ) : null}
                {selected.map((role, index) => <RoleEstimate key={roleId(role)} role={role} plan={plans[index]?.data?.plan} refusal={plans[index]?.data?.refusal} />)}
                <p className="m-0 text-xs text-muted-foreground">
                  The estimate comes from a simulation of each role's own recent pace and spread, updated whenever you open this page. Unknown outcomes and quiet weeks stretch it. Only large differences can be seen at this volume.
                </p>
              </section>
              <p className="m-0 rounded-md border-l-4 border-warning bg-warning/10 p-3 text-sm" data-testid="learn-cost-warning">
                While it runs, {shareWords(percent)} of these launches use the cheaper model, whatever its quality. You can stop at any time and you keep what was learned. It pauses by itself when a quota window is forecast to run out.
              </p>
              <section className="rounded-md border border-border p-3 text-sm" aria-label="Fine print in plain words">
                <h3 className="m-0 mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Fine print in plain words</h3>
                <p className="m-0">Running sessions are never changed. Choosing a model yourself still wins. Results appear in Evidence, marked by how work was assigned. Checking early cannot produce a verdict: it is read once, when both sides reach the planned size, and it can read inconclusive. A role default changes only if you approve a proposal, and even then you apply the change yourself.</p>
              </section>
              {start.error ? <p role="alert" className="m-0 text-sm text-destructive" data-testid="learn-start-error">{startErrorText(start.error, "start Learn mode")}</p> : null}
              <div className="flex flex-wrap justify-end gap-2">
                <Button variant="ghost" className="min-h-11 sm:min-h-8" onClick={() => navigate("/usage?tab=evidence")} data-testid="learn-cancel">Cancel</Button>
                <Button variant="primary" className="min-h-11 sm:min-h-8" disabled={selected.length === 0 || start.isPending || planning} onClick={() => void submit()} data-testid="learn-start">{start.isPending ? "Starting" : "Start Learn mode"}</Button>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
}

function RoleEstimate({ role, plan, refusal }: { role: LearnRole; plan: UsageExperimentPlan | undefined; refusal: string | undefined }) {
  if (refusal || isExemptRole(role.key)) return <p className="m-0 text-xs text-muted-foreground" data-testid="learn-role-estimate">{role.key}: {refusal || "left out"}</p>;
  if (!plan) return <p className="m-0 text-xs text-muted-foreground" data-testid="learn-role-estimate">{role.key}: no estimate yet.</p>;
  const planned = Number(plan.minAcceptedPerArm);
  return (
    <p className="m-0 text-xs text-muted-foreground" data-testid="learn-role-estimate">
      {role.key}: the verdict is read once, when each side holds {planned} accepted outcomes (an approval, a passing validation or a positive verdict). A launch that only finished does not count, and {Math.round(plan.acceptedShare * 100)}% of its finished launches record one.
      {plan.unitsPerDay > 0 ? ` At ${(plan.unitsPerDay * 7).toFixed(1)} finished launches a week` : " With no finished launches"}{plan.detect?.reached ? ` that takes ${horizonText(plan.detect)}. A true halving in cost would then read cheaper about ${Math.round(plan.powerAtLook * 100)}% of the time, and no difference would read "about the same" about ${Math.round(plan.sameAtLook * 100)}% of the time; otherwise it reads inconclusive.` : ", no verdict is reachable from this role's work until it records accepted outcomes."}
      {plan.completedOnly?.reached ? ` Counting launches that only finished would take ${horizonText(plan.completedOnly)}: a weaker reading, never a verdict.` : ""}
      {plan.varianceAssumed ? ` The spread was assumed because only ${Number(plan.pilotUnits)} launches have finished.` : ""}
      {plan.detectableIn90Days > 0 ? ` A ${Math.round(plan.detectableIn90Days * 100)}% or larger drop could be seen within 90 days.` : ""}
    </p>
  );
}
