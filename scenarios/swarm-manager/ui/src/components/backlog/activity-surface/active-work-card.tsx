import { ExternalLink, LoaderCircle, OctagonX, PlayCircle } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { defaultApiClient } from "../../../lib/api-client";
import { API_ENDPOINTS } from "../../../lib/api-endpoints";
import { Button } from "../../ui/button";

type Progress = { current_node: string; slice_count: number; turns: number; tokens: number; cost_usd: number; updated_at?: string };

export function ActiveWorkCard({ executionId, agentManagerUrl, onStopped }: { executionId: string; agentManagerUrl: string | null; onStopped: () => void }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["execution-progress", executionId],
    queryFn: () => defaultApiClient.get<Progress>(API_ENDPOINTS.executionProgress(executionId)),
    refetchInterval: 6_000,
  });
  const stop = async () => {
    await defaultApiClient.post(API_ENDPOINTS.executionCancel(executionId), { reason: "Stopped from backlog activity" });
    onStopped();
  };
  return <section className="rounded-xl border border-cyan-400/25 bg-cyan-400/[0.06] p-4 shadow-sm" aria-label="Active work">
    <div className="flex flex-wrap items-start justify-between gap-3"><div className="flex items-start gap-3"><span className="mt-0.5 rounded-lg bg-cyan-400/15 p-2 text-cyan-200"><PlayCircle className="h-5 w-5" /></span><div><h2 className="text-sm font-semibold text-white">Executing plan</h2><p className="mt-1 text-xs text-slate-300">{isLoading ? "Reading live workflow progress…" : error ? "Live progress is temporarily unavailable; the execution remains active." : `Currently at ${data?.current_node || "the next workflow step"}.`}</p></div></div><div className="flex items-center gap-2"><Button variant="outline" size="sm" onClick={() => void stop()}><OctagonX className="mr-1.5 h-3.5 w-3.5" />Stop</Button>{agentManagerUrl ? <Button asChild variant="outline" size="sm"><a href={`${agentManagerUrl.replace(/\/$/, "")}/workflow-executions/${executionId}`} target="_blank" rel="noreferrer">View workflow<ExternalLink className="ml-1.5 h-3.5 w-3.5" /></a></Button> : null}</div></div>
    <div className="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-4">{[{ label: "Slices", value: data ? data.slice_count : "—" }, { label: "Turns", value: data ? data.turns : "—" }, { label: "Tokens", value: data ? data.tokens.toLocaleString() : "—" }, { label: "Cost", value: data ? `$${data.cost_usd.toFixed(2)}` : "—" }].map((metric) => <div key={metric.label} className="rounded-lg border border-white/10 bg-slate-950/30 px-3 py-2"><div className="text-[11px] text-slate-400">{metric.label}</div><div className="mt-0.5 text-sm font-medium text-slate-100">{metric.value}</div></div>)}</div>
    {isLoading ? <div className="mt-3 flex items-center gap-2 text-xs text-cyan-100"><LoaderCircle className="h-3.5 w-3.5 animate-spin" />Updating every six seconds while this workflow is active.</div> : null}
  </section>;
}
