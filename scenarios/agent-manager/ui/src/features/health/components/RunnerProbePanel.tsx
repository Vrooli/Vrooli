import { useState } from "react";
import { CheckCircle2, Loader2, XCircle, Zap } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { useHealth, probeRunner } from "../../../hooks/useApi";
import { formatDuration, runnerLabel } from "../../../lib/format";
import { jsonValueToPlain, runnerTypeFromSlug } from "../../../lib/utils";
import type { JsonValue, ProbeResult } from "../../../types";

interface Dependency { name: string; slug: string; available: boolean; message?: string }

function readDependency(value?: JsonValue): { available: boolean; message?: string } {
  const parsed = jsonValueToPlain(value) as Record<string, unknown> | undefined;
  return { available: parsed?.status === "healthy", message: typeof parsed?.error === "string" ? parsed.error : undefined };
}

function ProbeRow({ dependency }: { dependency: Dependency }) {
  const runnerType = runnerTypeFromSlug(dependency.slug);
  const [probing, setProbing] = useState(false);
  const [result, setResult] = useState<ProbeResult | null>(null);
  const [failure, setFailure] = useState<string | null>(null);

  const probe = async () => {
    if (runnerType === undefined || probing) return;
    setProbing(true);
    setResult(null);
    setFailure(null);
    try {
      setResult(await probeRunner(runnerType));
    } catch (error) {
      setFailure((error as Error).message);
    } finally {
      setProbing(false);
    }
  };

  const outcome = failure ?? (result ? result.error || (result.success ? "" : "Probe failed") : null);
  return (
    <li className="flex flex-col gap-2 border-b border-border px-4 py-3 last:border-b-0" data-testid={`runner-probe-${dependency.slug}`}>
      <div className="flex min-w-0 items-center gap-3">
        {dependency.available ? <CheckCircle2 aria-hidden="true" className="h-4 w-4 shrink-0 text-success" /> : <XCircle aria-hidden="true" className="h-4 w-4 shrink-0 text-destructive" />}
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium">{dependency.name}</p>
          {dependency.message ? <p className="break-words text-xs text-muted-foreground">{dependency.message}</p> : null}
        </div>
        <StatusBadge tone={dependency.available ? "success" : "danger"}>{dependency.available ? "Available" : "Unavailable"}</StatusBadge>
        {runnerType !== undefined ? (
          <Button variant="secondary" size="sm" onClick={() => void probe()} disabled={probing} className="min-h-11 gap-1 sm:min-h-8" title="Send a test request to verify the agent responds">
            {probing ? <Loader2 aria-hidden="true" className="h-3 w-3 animate-spin" /> : <Zap aria-hidden="true" className="h-3 w-3" />}
            Probe
          </Button>
        ) : null}
      </div>
      {result || failure ? (
        <p role="status" className={`rounded border px-3 py-2 text-xs ${result?.success ? "border-success/20 bg-success/10 text-success" : "border-destructive/20 bg-destructive/10 text-destructive"}`}>
          <span className="font-medium">{result?.success ? "Probe succeeded" : "Probe failed"}</span>
          {result ? <span className="text-muted-foreground"> · {formatDuration(Number(result.latencyMs))}</span> : null}
          {outcome ? <span className="mt-1 block break-words opacity-80">{outcome}</span> : null}
        </p>
      ) : null}
    </li>
  );
}

/** Live checks: what the API reports for the sandbox and each runner, with a probe that sends a real test request. */
export function RunnerProbePanel() {
  const health = useHealth();
  const dependencies = health.data?.dependencies ?? {};
  const rows: Dependency[] = [
    ...(dependencies.sandbox ? [{ name: "Workspace sandbox", slug: "sandbox", ...readDependency(dependencies.sandbox) }] : []),
    ...Object.entries(dependencies)
      .filter(([name]) => name.startsWith("runner_"))
      .map(([name, value]) => ({ name: runnerLabel(name.replace("runner_", "")), slug: name.replace("runner_", ""), ...readDependency(value) })),
  ];
  if (rows.length === 0) return null;
  return (
    <Card className="overflow-hidden" data-testid="runner-probe-panel">
      <CardHeader className="px-4 py-3"><CardTitle className="text-base">Live checks</CardTitle></CardHeader>
      <CardContent className="p-0"><ul>{rows.map((row) => <ProbeRow key={row.slug} dependency={row} />)}</ul></CardContent>
    </Card>
  );
}
