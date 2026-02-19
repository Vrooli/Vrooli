import { useQuery } from "@tanstack/react-query";
import { Shield, RefreshCw } from "lucide-react";
import { fetchAudit } from "../lib/api";
import { Button } from "./ui/button";

function auditBadge(status: string) {
  const colors: Record<string, string> = {
    compliant: "bg-green-500/10 text-green-400",
    mismatch: "bg-red-500/10 text-red-400",
    missing_scenario: "bg-yellow-500/10 text-yellow-400",
    missing_port: "bg-yellow-500/10 text-yellow-400",
  };
  return colors[status] ?? "bg-slate-500/10 text-slate-400";
}

export function AuditView() {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["audit"],
    queryFn: fetchAudit,
    refetchInterval: 60000,
  });

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-slate-400" />
          <h2 className="text-lg font-semibold">Port Compliance Audit</h2>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isFetching}>
          <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
        </Button>
      </div>

      {isLoading && <p className="mt-4 text-slate-400">Running audit...</p>}
      {error && <p className="mt-4 text-red-400">Failed to run audit</p>}

      {data && (
        <>
          <div className="mt-4 flex gap-4 text-sm">
            <span className="text-green-400">{data.compliant} compliant</span>
            {data.violations > 0 && (
              <span className="text-red-400">{data.violations} violation(s)</span>
            )}
            <span className="text-slate-400">{data.total} total</span>
          </div>

          {data.results.length > 0 && (
            <div className="mt-3 overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-white/10 text-left text-slate-400">
                    <th className="pb-2 pr-4">Subdomain</th>
                    <th className="pb-2 pr-4">Scenario</th>
                    <th className="pb-2 pr-4">Expected</th>
                    <th className="pb-2 pr-4">Actual</th>
                    <th className="pb-2 pr-4">Status</th>
                    <th className="pb-2">Detail</th>
                  </tr>
                </thead>
                <tbody>
                  {data.results.map((r, i) => (
                    <tr key={i} className="border-b border-white/5">
                      <td className="py-2 pr-4 font-medium text-slate-200">{r.subdomain}</td>
                      <td className="py-2 pr-4 text-slate-300">{r.scenario_name}</td>
                      <td className="py-2 pr-4 font-mono text-slate-300">{r.expected_port}</td>
                      <td className="py-2 pr-4 font-mono text-slate-300">{r.actual_port || "—"}</td>
                      <td className="py-2 pr-4">
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${auditBadge(r.status)}`}>
                          {r.status}
                        </span>
                      </td>
                      <td className="py-2 text-slate-400 truncate max-w-[200px]">{r.detail || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}
