import { useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { settingsService, transitionService } from "../../services";

type Gate = {
  id: string;
  decides: string;
  mode?: string;
  acceptanceRate?: number;
  sampleSize?: number;
  threshold?: number;
  readiness?: string;
};

/** Shows a bounded, dismissible prompt for server-certified ready gates. */
export function AutonomyReadyPrompt() {
  const queryClient = useQueryClient();
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());
  const { data: transitions = [] } = useQuery({
    queryKey: ["transition-catalog"],
    queryFn: () => transitionService.list(),
  });
  const { data: settings } = useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
  });

  const ready = useMemo(() => transitions.flatMap((transition) => {
    const gates = (transition as unknown as { humanGates?: readonly Gate[] }).humanGates ?? [];
    return gates
      .filter((gate) => gate.readiness === "ready" && gate.mode !== "auto")
      .map((gate) => ({ transition, gate }));
  }).filter(({ gate }) => !dismissed.has(gate.id)), [dismissed, transitions]);

  if (ready.length === 0) return null;

  return (
    <div className="space-y-2 border-b border-cyan-900/50 bg-cyan-950/20 p-3" data-testid="autonomy-ready-prompts">
      {ready.map(({ transition, gate }) => {
        const rate = gate.acceptanceRate ?? 0;
        const sample = gate.sampleSize ?? 0;
        const threshold = gate.threshold ?? 0;
        return (
          <div key={`${transition.key}:${gate.id}`} className="rounded-md border border-cyan-800/60 bg-slate-900/70 p-3 text-sm">
            <p className="font-medium text-cyan-200">Gate ready: {gate.id}</p>
            <p className="mt-1 text-xs text-slate-300">{gate.decides}</p>
            <p className="mt-2 text-xs text-slate-400">
              Acceptance {(rate * 100).toFixed(1)}% · attributed sample {sample} · threshold {(threshold * 100).toFixed(1)}%
            </p>
            <div className="mt-2 flex gap-2">
              <button
                type="button"
                className="rounded bg-cyan-600 px-2 py-1 text-xs font-medium text-white hover:bg-cyan-500"
                onClick={() => {
                  void settingsService.update({
                    autonomyGateModes: { ...(settings?.autonomyGateModes ?? {}), [gate.id]: "auto" },
                  }).then(() => {
                    void queryClient.invalidateQueries({ queryKey: ["settings"] });
                    void queryClient.invalidateQueries({ queryKey: ["transition-catalog"] });
                  });
                }}
                disabled={!settings}
                data-testid={`autonomy-ready-flip-${gate.id}`}
              >
                Flip to automatic
              </button>
              <button
                type="button"
                className="rounded border border-slate-700 px-2 py-1 text-xs text-slate-300 hover:bg-slate-800"
                onClick={() => setDismissed((current) => new Set(current).add(gate.id))}
                data-testid={`autonomy-ready-dismiss-${gate.id}`}
              >
                Dismiss
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
}
