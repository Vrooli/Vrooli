import { useMutation } from "@tanstack/react-query";
import { Zap, RefreshCw } from "lucide-react";
import { runProbes, type ProbeResponse } from "../lib/api";
import { Button } from "./ui/button";

function probeStatusBadge(status: string) {
  const colors: Record<string, string> = {
    up: "bg-green-500/10 text-green-400",
    down: "bg-red-500/10 text-red-400",
    timeout: "bg-yellow-500/10 text-yellow-400",
    error: "bg-red-500/10 text-red-400",
  };
  return colors[status] ?? "bg-slate-500/10 text-slate-400";
}

export function ProbeResults() {
  const { mutate, data, isPending } = useMutation({
    mutationFn: runProbes,
  });

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Zap className="h-5 w-5 text-slate-400" />
          <h2 className="text-lg font-semibold">Liveness Probes</h2>
        </div>
        <Button variant="outline" size="sm" onClick={() => mutate()} disabled={isPending}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isPending ? "animate-spin" : ""}`} />
          {isPending ? "Probing..." : "Run Probes"}
        </Button>
      </div>

      {!data && !isPending && (
        <p className="mt-4 text-slate-400">Click "Run Probes" to check all route liveness.</p>
      )}

      {data && (
        <>
          <div className="mt-4 flex gap-4 text-sm">
            <span className="text-green-400">{data.summary.up} up</span>
            <span className="text-red-400">{data.summary.down} down</span>
            <span className="text-slate-400">{data.summary.total} total</span>
          </div>

          <div className="mt-3 overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-white/10 text-left text-slate-400">
                  <th className="pb-2 pr-4">Subdomain</th>
                  <th className="pb-2 pr-4">Type</th>
                  <th className="pb-2 pr-4">Status</th>
                  <th className="pb-2 pr-4">Latency</th>
                  <th className="pb-2">Error</th>
                </tr>
              </thead>
              <tbody>
                {data.results.map((r, i) => (
                  <tr key={i} className="border-b border-white/5">
                    <td className="py-2 pr-4 font-medium text-slate-200">{r.subdomain}</td>
                    <td className="py-2 pr-4 text-slate-300">{r.probe_type}</td>
                    <td className="py-2 pr-4">
                      <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${probeStatusBadge(r.status)}`}>
                        {r.status}
                      </span>
                    </td>
                    <td className="py-2 pr-4 font-mono text-slate-300">
                      {r.latency_ms > 0 ? `${r.latency_ms}ms` : "—"}
                    </td>
                    <td className="py-2 text-slate-400 truncate max-w-[200px]">
                      {r.error_msg || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
