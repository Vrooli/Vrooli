import type { ModelPolicyDriftSnapshot } from "../api/types";

export function ModelPolicyDriftStatus({ snapshot }: { snapshot?: ModelPolicyDriftSnapshot }) {
  const status = snapshot?.status ?? "not_measured";
  const label = status.replace("_", " ");
  return (
    <section className="mb-4 rounded-lg border border-border bg-card/40 p-4" data-testid="model-policy-drift-status">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold">Model-policy drift safeguard</h2>
          <p className="text-xs text-muted-foreground">Scheduled live-catalog detection and scenario-qa reporting.</p>
        </div>
        <span className="rounded-full border px-2 py-1 text-xs uppercase tracking-wide">{label}</span>
      </div>
      <p className="mt-2 text-xs text-muted-foreground">
        Last detect: {snapshot?.last_run ? new Date(snapshot.last_run).toLocaleString() : "not measured"} · measured {snapshot?.measured ?? 0}/{snapshot?.total ?? 4} runners · every {snapshot?.interval_hours ?? 168}h · findings {snapshot?.findings?.length ?? 0}
      </p>
    </section>
  );
}
