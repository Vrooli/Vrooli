import { useQuery } from "@tanstack/react-query";
import { useTimeWindow } from "../../hooks/useTimeWindow";

interface RunClassResponse {
  classes: Array<{ class: string; run_count: number; success_count: number; failed_count: number }>;
  executed_denominator: number;
  missing_model_runs: number;
  missing_model_rate: number;
  excluded_classes: string[];
}

async function fetchRunClasses(preset: string): Promise<RunClassResponse> {
	const response = await fetch(`/api/v1/stats/run-classes?preset=${encodeURIComponent(preset)}`, { cache: "no-store" });
  if (!response.ok) throw new Error(`Run-class API error ${response.status}`);
  return response.json() as Promise<RunClassResponse>;
}

export function RunClassBreakdown() {
  const { filter } = useTimeWindow();
  const query = useQuery({ queryKey: ["stats", "run-classes", filter.preset], queryFn: () => fetchRunClasses(filter.preset ?? "7d"), staleTime: 30_000, refetchInterval: 60_000 });
  if (query.isLoading) return <div className="rounded-lg border border-border bg-card/40 p-4 text-sm text-muted-foreground">Loading run classes…</div>;
  if (query.error) return <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-4 text-sm text-red-500">Run classes: {query.error.message}</div>;
  const data = query.data;
  return (
    <section className="rounded-lg border border-border bg-card/50 p-4" data-testid="run-class-breakdown">
      <div className="flex items-baseline justify-between gap-2">
        <div><h3 className="text-sm font-semibold">Run classes</h3><p className="text-xs text-muted-foreground">Executed measures exclude imported and interactive runs.</p></div>
        <span className="text-xs text-muted-foreground">denominator {data?.executed_denominator ?? 0}</span>
      </div>
      <div className="mt-3 grid gap-2 sm:grid-cols-3">
        {(data?.classes ?? []).map((item) => <div key={item.class} className="rounded border border-border/60 px-3 py-2 text-xs"><div className="font-medium">{item.class || "unclassified"}</div><div className="text-muted-foreground">{item.run_count} runs · {item.success_count} success · {item.failed_count} failed</div></div>)}
      </div>
      <p className="mt-3 text-xs text-muted-foreground">Residual missing-model rate: {((data?.missing_model_rate ?? 0) * 100).toFixed(2)}% ({data?.missing_model_runs ?? 0} runs)</p>
    </section>
  );
}
