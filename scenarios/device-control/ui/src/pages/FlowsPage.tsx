/* eslint-disable @typescript-eslint/use-unknown-in-catch-callback-variable, @typescript-eslint/no-misused-promises */
import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { acquireSession, killSession, listDevices, listStrategies, releaseSession, runFlow, validateFlow, type Device, type Session, type Strategy } from "../api/deviceControl";
import { API_BASE } from "../api/client";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { selectors } from "../consts/selectors";
import { buildApiUrl } from "@vrooli/api-base";

const starter = JSON.stringify({ id: "smoke-flow", name: "Smoke flow", steps: [{ id: "observe", kind: "observe", required_capabilities: ["screenshot"], timeout_ms: 1000 }] }, null, 2);

export function FlowsPage() {
  const { t } = useTranslation();
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [devices, setDevices] = useState<Device[]>([]);
  const [device, setDevice] = useState("");
  const [run, setRun] = useState<{ disposition: string; chapters: Array<{ id: string; disposition: string; message: string }>; resolutions?: Array<{ target: string; rung: string; confidence: number }>; evidence: Array<{ id: string; kind: string; checksum?: string; size_bytes: number; redaction_verified: boolean; applied_rules?: string[]; recording_method?: string; effective_fps?: number; disposition?: string; disposition_reason?: string }> }>();
  const [strategy, setStrategy] = useState("");
  const [flow, setFlow] = useState(starter);
  const [report, setReport] = useState<{ runnable?: boolean; gaps?: string[]; warnings?: string[] }>();
  const [activeSession, setActiveSession] = useState<Session>();
  const [running, setRunning] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([listStrategies(), listDevices()]).then(([strategyResult, deviceResult]) => { setStrategies(strategyResult.strategies); setStrategy(strategyResult.strategies[0]?.id ?? ""); setDevices(deviceResult.devices); setDevice(deviceResult.devices.find((item) => item.status === "available")?.id ?? ""); }).catch((e: Error) => setError(e.message));
  }, []);

  async function validate() {
    try { setError(""); setReport(await validateFlow(strategy, JSON.parse(flow))); }
    catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.flows.invalidFlow)); }
  }
  async function execute() {
    try { setError(""); setRunning(true); const definition: unknown = JSON.parse(flow) as unknown; const lease = await acquireSession(device, "browser-operator"); setActiveSession(lease.session); try { setRun(await runFlow(device, "browser-operator", lease.session.lease_token ?? "", definition)); } finally { setActiveSession(undefined); await releaseSession(lease.session.id).catch(() => undefined); } }
    catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.flows.flowFailed)); }
    finally { setRunning(false); }
  }
  async function killActiveSession() { if (!activeSession) return; try { await killSession(activeSession.id); setActiveSession(undefined); } catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.flows.sessionKillFailed)); } }

  return (
    <section data-testid={selectors.pages.flows} className="flex flex-col gap-4" aria-labelledby="flows-heading">
      <h2 id="flows-heading" className="text-2xl font-semibold">{t(strings.pages.flows.title)}</h2>
      <p className="text-app-muted-foreground">{t(strings.pages.flows.description)}</p>
      <div className="grid gap-4 lg:grid-cols-2">
        <Card><CardHeader><CardTitle>{t(strings.pages.flows.flowDefinition)}</CardTitle></CardHeader><CardContent className="flex flex-col gap-3">
          <label htmlFor="flow-strategy">{t(strings.pages.flows.strategy)}</label>
          <select id="flow-strategy" data-testid={selectors.pages.flowStrategy} value={strategy} onChange={(e) => setStrategy(e.target.value)} className="rounded-md border bg-transparent p-2">
            {strategies.map((item) => <option key={item.id} value={item.id}>{t(strings.pages.flows.strategyOption, { id: item.id, status: item.status })}</option>)}
          </select>
          <label htmlFor="flow-device">{t(strings.pages.flows.device)}</label>
          <select id="flow-device" data-testid={selectors.pages.flowDevice} value={device} onChange={(e) => setDevice(e.target.value)} className="rounded-md border bg-transparent p-2">{devices.filter((item) => item.status === "available").map((item) => <option key={item.id} value={item.id}>{t(strings.pages.flows.deviceOption, { name: item.model || item.name, id: item.serial || item.id })}</option>)}</select>
          <label htmlFor="flow-json">{t(strings.pages.flows.json)}</label>
          <textarea id="flow-json" data-testid={selectors.pages.flowDefinition} value={flow} onChange={(e) => setFlow(e.target.value)} rows={14} className="rounded-md border bg-transparent p-3 font-mono text-base" />
          <div className="flex gap-2"><Button data-testid={selectors.pages.flowValidate} onClick={validate}>{t(strings.pages.flows.validate)}</Button><Button data-testid={selectors.pages.flowRun} onClick={() => void execute()} disabled={running || !device}>{running ? t(strings.pages.flows.running) : t(strings.pages.flows.acquireAndRun)}</Button></div>
          {activeSession && <div data-testid={selectors.pages.flowActiveSession} className="rounded-md border border-app-destructive/40 p-3" role="status"><p className="font-medium">{t(strings.pages.flows.liveSession, { id: activeSession.id })}</p><p className="text-sm text-app-muted-foreground">{t(strings.pages.flows.killAvailable)}</p><Button data-testid={selectors.pages.flowKillSession} className="mt-2" onClick={() => void killActiveSession()} aria-label={t(strings.pages.flows.killActiveSession)}>{t(strings.pages.flows.killActiveSession)}</Button></div>}
          {error && <p data-testid={selectors.pages.flowError} role="alert" className="text-red-600">{error}</p>}
        </CardContent></Card>
        <Card data-testid={selectors.pages.flowGapReport}><CardHeader><CardTitle>{t(strings.pages.flows.capabilityGapReport)}</CardTitle></CardHeader><CardContent>
          {report ? <div role="status"><p className={report.runnable ? "text-emerald-600" : "text-amber-600"}>{report.runnable ? t(strings.pages.flows.runnable) : t(strings.pages.flows.blockedBeforeExecution)}</p>{(report.gaps ?? []).map((gap) => <p key={gap} className="mt-2 text-sm">{gap}</p>)}{(report.warnings ?? []).map((warning) => <p key={warning} className="mt-2 text-sm text-app-muted-foreground">{warning}</p>)}</div> : <p className="text-app-muted-foreground">{t(strings.pages.flows.noValidation)}</p>}
        </CardContent></Card>
        <Card data-testid={selectors.pages.flowRunReview}>
          <CardHeader><CardTitle>{t(strings.pages.flows.runReview)}</CardTitle></CardHeader>
          <CardContent>
            {run ? (
              <div className="flex flex-col gap-2">
                <p role="status">{run.disposition}</p>
                {run.chapters.map((chapter) => (
                  <p key={chapter.id} className="text-sm">
                    {t(strings.pages.flows.chapterSummary, { id: chapter.id, disposition: chapter.disposition, message: chapter.message })}
                  </p>
                ))}
                {(run.resolutions ?? []).map((resolution) => (
                  <p key={`${resolution.target}-${resolution.rung}`} className="text-sm">
                    {t(strings.pages.flows.resolutionSummary, { target: resolution.target, rung: resolution.rung, confidence: resolution.confidence.toFixed(2) })}
                  </p>
                ))}
                <p className="font-medium">{t(strings.pages.flows.retainedEvidence, { count: run.evidence.length })}</p>
                {run.evidence.map((ref) => (
                  <div key={ref.id} className={`rounded-md border p-2 ${ref.disposition === "degraded" ? "border-amber-500" : ""}`}>
                    <p className="text-sm">
                      <span>{ref.id}</span> <span aria-hidden="true">·</span> <span>{ref.kind}</span> <span aria-hidden="true">·</span>
                      <span>{ref.checksum ?? t(strings.pages.flows.checksumUnavailable)}</span> <span aria-hidden="true">·</span>
                      <span>{ref.size_bytes}</span> {t(strings.pages.flows.bytes)} <span aria-hidden="true">·</span>
                      <span>{t(strings.pages.flows.redactionVerified, { verified: String(ref.redaction_verified) })}</span>
                      <span aria-hidden="true">·</span> {(ref.applied_rules ?? []).join(", ")}
                    </p>
                    {ref.disposition === "degraded" && (
                      <p role="status" className="font-medium text-amber-700">
                        {t(strings.pages.flows.degradedEvidence, { reason: ref.disposition_reason ?? t(strings.pages.flows.dispositionReasonUnavailable) })}
                      </p>
                    )}
                    {ref.kind === "image" && <img data-testid={selectors.pages.flowEvidenceImage} src={buildApiUrl(`/api/v1/evidence/${encodeURIComponent(ref.id)}`, { baseUrl: API_BASE })} alt={t(strings.pages.flows.retainedEvidenceAlt, { id: ref.id })} className="mt-2 max-h-80 w-auto rounded border" />}
                  </div>
                ))}
              </div>
            ) : <p className="text-app-muted-foreground">{t(strings.pages.flows.noRun)}</p>}
          </CardContent>
        </Card>
      </div>
    </section>
  );
}
