import type { Transition } from "@vrooli/proto-types/swarm-manager/v1/domain/transition_pb";
import { Card } from "../ui/card";
import { ToggleButtons } from "./ToggleButtons";
import type { AutonomyGateMode, Settings } from "../../types/settings";
import type { StatsResponse } from "../../types/stats";

const MODES: { value: AutonomyGateMode; label: string }[] = [
  { value: "manual", label: "Manual" },
  { value: "suggest", label: "Suggest" },
  { value: "auto", label: "Automatic" },
];

export interface AutonomyTabProps {
  form: Settings;
  patch: (updates: Partial<Settings>) => void;
  transitions?: readonly Transition[];
  stats?: StatsResponse;
}

/** Renders the transition catalog's human gates and their live policy mode. */
export function AutonomyTab({ form, patch, transitions = [], stats }: AutonomyTabProps) {
  const gates = transitions.flatMap((transition) => {
    const value = transition as unknown as { humanGates?: readonly { id: string; defaultMode?: string; mode?: string; decides: string; threshold?: number; minSample?: number; acceptanceRate?: number; sampleSize?: number; readiness?: string }[] };
    return (value.humanGates ?? []).map((gate) => ({ transition, gate }));
  });

  return (
    <div className="space-y-4" data-testid="settings-autonomy-tab">
      <Card padding="sm">
        <h3 className="font-medium text-slate-100">Autonomy gates</h3>
        <p className="mt-1 text-sm text-slate-400">
          Every gate is declared by the transition it controls. Automatic mode is only available
          when the transition policy accepts the operator&apos;s configured override.
        </p>
      </Card>

      {gates.length === 0 ? (
        <Card padding="sm"><p className="text-sm text-slate-400">No declared human gates found.</p></Card>
      ) : gates.map(({ transition, gate }) => {
        const mode = normalizeGateMode(form.autonomyGateModes[gate.id] ?? gate.mode ?? gate.defaultMode);
        const evidence = stats?.agent.recommendation_acceptance_by_gate?.[gate.id];
        const sample = gate.sampleSize ?? evidence?.sample_size ?? 0;
        const rate = gate.acceptanceRate ?? evidence?.rate ?? 0;
        const minSample = gate.minSample ?? 5;
        const threshold = gate.threshold ?? 0;
        const readiness = gate.readiness ?? (sample < minSample ? "insufficient-sample" : rate >= threshold ? "ready" : "below-threshold");
        return (
          <Card key={`${transition.key}:${gate.id}`} padding="sm" data-testid="autonomy-gate-row">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="min-w-0 flex-1">
                <p className="font-medium text-slate-100">{gate.id}</p>
                <p className="text-xs text-cyan-300">{transition.key}</p>
                <p className="mt-2 text-sm text-slate-300">{gate.decides}</p>
              </div>
              <div className="min-w-[18rem]">
                <p className="text-xs uppercase tracking-wide text-slate-500">Mode</p>
                <ToggleButtons
                  value={mode}
                  options={MODES}
                  onChange={(next) => patch({
                    autonomyGateModes: { ...form.autonomyGateModes, [gate.id]: next },
                  })}
                />
              </div>
            </div>
            <div className="mt-3 grid gap-2 text-xs text-slate-400 sm:grid-cols-3">
              <span>Attributed sample: {sample}</span>
              <span>Acceptance: {(rate * 100).toFixed(0)}% · Threshold: {(threshold * 100).toFixed(0)}%</span>
              <span className={readiness === "ready" ? "text-emerald-300" : "text-amber-300"}>
                Readiness: {readiness.replace("-", " ")}
              </span>
            </div>
          </Card>
        );
      })}
    </div>
  );
}

function normalizeGateMode(value?: string): AutonomyGateMode {
  return value === "auto" || value === "suggest" ? value : "manual";
}
