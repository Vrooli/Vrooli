/* eslint-disable no-restricted-syntax, @typescript-eslint/use-unknown-in-catch-callback-variable, @typescript-eslint/no-misused-promises */
import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { listStrategies, validateFlow, type Strategy } from "../api/deviceControl";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const starter = JSON.stringify({ id: "smoke-flow", name: "Smoke flow", steps: [{ id: "observe", kind: "observe", required_capabilities: ["screenshot"], timeout_ms: 1000 }] }, null, 2);

export function FlowsPage() {
  const { t } = useTranslation();
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [strategy, setStrategy] = useState("");
  const [flow, setFlow] = useState(starter);
  const [report, setReport] = useState<{ runnable?: boolean; gaps?: string[]; warnings?: string[] }>();
  const [error, setError] = useState("");

  useEffect(() => {
    listStrategies().then(({ strategies: values }) => { setStrategies(values); setStrategy(values[0]?.id ?? ""); }).catch((e: Error) => setError(e.message));
  }, []);

  async function validate() {
    try { setError(""); setReport(await validateFlow(strategy, JSON.parse(flow))); }
    catch (e) { setError(e instanceof Error ? e.message : "Invalid flow JSON"); }
  }

  return (
    <section className="flex flex-col gap-4" aria-labelledby="flows-heading">
      <h2 id="flows-heading" className="text-2xl font-semibold">{t(strings.pages.flows.title)}</h2>
      <p className="text-app-muted-foreground">Validate required capabilities before a lease is acquired.</p>
      <div className="grid gap-4 lg:grid-cols-2">
        <Card><CardHeader><CardTitle>Flow definition</CardTitle></CardHeader><CardContent className="flex flex-col gap-3">
          <label htmlFor="flow-strategy">Strategy</label>
          <select id="flow-strategy" value={strategy} onChange={(e) => setStrategy(e.target.value)} className="rounded-md border bg-transparent p-2">
            {strategies.map((item) => <option key={item.id} value={item.id}>{item.id} · {item.status}</option>)}
          </select>
          <label htmlFor="flow-json">JSON</label>
          <textarea id="flow-json" value={flow} onChange={(e) => setFlow(e.target.value)} rows={14} className="rounded-md border bg-transparent p-3 font-mono text-sm" />
          <Button onClick={validate}>Validate before run</Button>
          {error && <p role="alert" className="text-red-600">{error}</p>}
        </CardContent></Card>
        <Card><CardHeader><CardTitle>Capability gap report</CardTitle></CardHeader><CardContent>
          {report ? <div role="status"><p className={report.runnable ? "text-emerald-600" : "text-amber-600"}>{report.runnable ? "Runnable" : "Blocked before execution"}</p>{(report.gaps ?? []).map((gap) => <p key={gap} className="mt-2 text-sm">{gap}</p>)}{(report.warnings ?? []).map((warning) => <p key={warning} className="mt-2 text-sm text-app-muted-foreground">{warning}</p>)}</div> : <p className="text-app-muted-foreground">No validation run yet.</p>}
        </CardContent></Card>
      </div>
    </section>
  );
}
