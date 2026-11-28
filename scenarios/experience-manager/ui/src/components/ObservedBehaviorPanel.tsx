import { useMemo, useState } from "react";
import type { ChangeEvent } from "react";

type SessionRef = { id: string; build: string | null; startedAt: string; endedAt: string };
type Report = {
  schemaVersion: string;
  scenario: string;
  reportDigest?: string;
  builds: string[];
  sessionRefs?: SessionRef[];
  totals: { sessions: number; activeMs: number };
  window: { from: string | null; to: string | null; sessions: number };
  flows: Array<{ steps: string[]; count: number; medianDurationMs: number; totalDurationMs: number }>;
  friction: Array<{ kind: string; count: number }>;
  recommendations: Array<{ id: string; title: string; evidence: string; suggestion: string; severity: string }>;
};
type Envelope = {
  kind: string;
  schemaVersion: string;
  state: string;
  scenario: string;
  build: string | null;
  builds: string[];
  sessions: SessionRef[];
  expectedBuild?: string;
  reportDigest: string;
  window: { from: string | null; to: string | null; sessions: number };
};
type Advisory = { schemaVersion: string; findings: Array<{ schemaVersion: string; id: string; severity: string; kind: string; title: string; evidence: string; suggestion: string; route?: string; element?: { testId?: string; name?: string }; reportDigest: string; advisory: boolean }> };
type Candidates = { schemaVersion: string; candidates: Array<{ schemaVersion: string; id: string; kind: string; severity: string; route?: string; element?: { testId?: string; name?: string }; evidence: string; reportDigest: string; status: string; source: string; advisory: boolean }> };
type Packet = { schemaVersion: string; scenario: string; build: string | null; sessions: Array<{ id: string; build: string | null }>; window: { from: string | null; to: string | null; sessions: number }; headline: string; topFlow: null | { steps: string[]; count: number; medianDurationMs: number }; topFriction: null | Record<string, unknown>; timeCostMs: number; reportDigest: string; caveat: string };
type Loaded = { findings: Advisory["findings"]; candidates: Candidates["candidates"]; packet: Packet; envelope: Envelope };
type ReviewDecision = "" | "promote" | "decline";

function canonical(value: unknown): string {
  if (value === null) return "null";
  if (typeof value === "number") return Number.isFinite(value) ? String(value) : "null";
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "string") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map((item) => canonical(item === undefined ? null : item)).join(",")}]`;
  if (typeof value === "object") {
    const record = value as Record<string, unknown>;
    const keys = Object.keys(record).filter((key) => record[key] !== undefined).sort();
    return `{${keys.map((key) => `${JSON.stringify(key)}:${canonical(record[key])}`).join(",")}}`;
  }
  return "null";
}

async function digestReport(report: Record<string, unknown>): Promise<string> {
  const { reportDigest: _ignored, ...body } = report;
  const bytes = new TextEncoder().encode(canonical(body));
  const hash = await crypto.subtle.digest("SHA-256", bytes);
  return `sha256:${Array.from(new Uint8Array(hash), (part) => part.toString(16).padStart(2, "0")).join("")}`;
}

async function readJson<T>(file: File | undefined, label: string): Promise<T> {
  if (!file) throw new Error(`Choose ${label}.`);
  return JSON.parse(await file.text()) as T;
}

function sameSessions(left: SessionRef[], right: SessionRef[]): boolean {
  return left.length === right.length && left.every((session, index) => {
    const other = right[index];
    return Boolean(other && session.id === other.id && session.build === other.build && session.startedAt === other.startedAt && session.endedAt === other.endedAt);
  });
}

async function validateBundle(files: { report?: File; envelope?: File; findings?: File; candidates?: File; packet?: File }, expectedScenario: string, expectedBuild: string): Promise<Loaded> {
  const [report, envelope, advisory, candidates, packet] = await Promise.all([
    readJson<Report & Record<string, unknown>>(files.report, "friction-report.json"),
    readJson<Envelope>(files.envelope, "observed-behavior.json"),
    readJson<Advisory>(files.findings, "advisory-findings.json"),
    readJson<Candidates>(files.candidates, "correction-candidates.json"),
    readJson<Packet>(files.packet, "packet-observed.json"),
  ]);
  if (report.schemaVersion !== "experience-friction-report/v1" || !Array.isArray(report.sessionRefs) || !Array.isArray(report.builds) || !report.totals || !Array.isArray(report.recommendations)) throw new Error("Unsupported or incomplete friction report.");
  const computedDigest = await digestReport(report);
  if (report.reportDigest !== computedDigest) throw new Error("Friction report digest does not match its content.");
  if (report.scenario !== expectedScenario || envelope.scenario !== expectedScenario || packet.scenario !== expectedScenario) throw new Error("Report scenario does not match this experience page.");
  if (envelope.kind !== "experience-observed-behavior" || envelope.schemaVersion !== "experience-observed-behavior/v1" || envelope.reportDigest !== computedDigest) throw new Error("Observed-behavior contract or report digest mismatch.");
  if (envelope.state !== "fresh" || envelope.build !== expectedBuild || envelope.expectedBuild !== expectedBuild) throw new Error("Evidence is not evaluated for the expected build.");
  if (report.totals.sessions === 0 || report.sessionRefs.length !== report.totals.sessions || new Set(report.sessionRefs.map((session) => session.id)).size !== report.sessionRefs.length || report.sessionRefs.some((session) => !/^[a-zA-Z0-9_-]{1,128}$/.test(session.id)) || envelope.window.sessions !== report.sessionRefs.length || report.window.sessions !== report.totals.sessions || envelope.window.from !== report.window.from || envelope.window.to !== report.window.to || !sameSessions(report.sessionRefs, envelope.sessions)) throw new Error("Session identity, window, or count mismatch.");
  if (report.builds.length !== 1 || report.builds[0] !== expectedBuild || envelope.builds.length !== 1 || envelope.builds[0] !== expectedBuild || report.sessionRefs.some((session) => !session.build || session.build !== expectedBuild)) throw new Error("Missing, stale, or mixed session build identity.");
  if (advisory.schemaVersion !== "experience-advisory-findings/v1" || !Array.isArray(advisory.findings) || advisory.findings.some((finding) => {
    const recommendation = report.recommendations.find((item) => `adv-${item.id}` === finding.id);
    return finding.schemaVersion !== "experience-advisory-finding/v1" || finding.advisory !== true || finding.reportDigest !== computedDigest || !recommendation || finding.title !== recommendation.title || finding.evidence !== recommendation.evidence || finding.suggestion !== recommendation.suggestion || finding.severity !== recommendation.severity;
  })) throw new Error("Advisory findings are malformed or not bound to this report.");
  const findingIds = new Set(advisory.findings.map((finding) => finding.id));
  const knownRecommendationIds = new Set(["dead-clicks", "rage-clicks", "repeated-actions", "dwell-without-action", "field-abandonment", "undeclared-flow", "declared-unobserved", "repeated-flow", "runtime-errors"]);
  if (report.recommendations.some((item) => knownRecommendationIds.has(item.id) && !findingIds.has(`adv-${item.id}`))) throw new Error("An advisory finding is missing from the bound findings artifact.");
  if (candidates.schemaVersion !== "experience-correction-candidates/v1" || !Array.isArray(candidates.candidates) || candidates.candidates.some((candidate) => candidate.schemaVersion !== "experience-correction-candidate/v1" || candidate.status !== "candidate" || candidate.source !== "observed" || candidate.advisory !== true || candidate.reportDigest !== computedDigest || typeof candidate.id !== "string" || typeof candidate.evidence !== "string")) throw new Error("Correction candidates are malformed or not bound to this report.");
  const expectedFlow = report.flows[0] ?? null;
  const expectedFriction = report.friction[0] ?? null;
  if (packet.schemaVersion !== "experience-packet-observed/v1" || packet.reportDigest !== computedDigest || packet.build !== expectedBuild || !packet.caveat || packet.sessions.length !== report.sessionRefs.length || packet.sessions.some((session, index) => session.id !== report.sessionRefs?.[index]?.id || session.build !== expectedBuild) || packet.window.from !== report.window.from || packet.window.to !== report.window.to || packet.window.sessions !== report.totals.sessions || packet.timeCostMs !== (expectedFlow?.totalDurationMs ?? report.totals.activeMs)) throw new Error("Packet block is malformed or not bound to this report and build.");
  if (canonical(packet.topFlow) !== canonical(expectedFlow) || canonical(packet.topFriction) !== canonical(expectedFriction)) throw new Error("Packet summary does not match the report flow and friction metrics.");
  return { findings: advisory.findings, candidates: candidates.candidates, packet, envelope };
}

export function ObservedBehaviorPanel({ scenario }: { scenario: string }) {
  const [expectedBuild, setExpectedBuild] = useState("");
  const [files, setFiles] = useState<{ report?: File; envelope?: File; findings?: File; candidates?: File; packet?: File }>({});
  const [loaded, setLoaded] = useState<Loaded>();
  const [dispositions, setDispositions] = useState<Record<string, { decision: ReviewDecision; reason: string }>>({});
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const onFile = (key: keyof typeof files) => (event: ChangeEvent<HTMLInputElement>) => {
    setLoaded(undefined);
    setError("");
    setDispositions({});
    setFiles((current) => ({ ...current, [key]: event.target.files?.[0] }));
  };
  const canLoad = useMemo(() => Boolean(expectedBuild.trim() && files.report && files.envelope && files.findings && files.candidates && files.packet), [expectedBuild, files]);
  const canExportDisposition = Boolean(loaded && loaded.candidates.length > 0 && loaded.candidates.every((candidate) => Boolean(dispositions[candidate.id]?.decision && dispositions[candidate.id]?.reason.trim())));
  const onExpectedBuildChange = (value: string) => {
    setLoaded(undefined);
    setError("");
    setDispositions({});
    setExpectedBuild(value);
  };
  const load = async () => {
    setBusy(true);
    setError("");
    setLoaded(undefined);
    setDispositions({});
    try { setLoaded(await validateBundle(files, scenario, expectedBuild.trim())); }
    catch (failure) { setError(failure instanceof Error ? failure.message : "Observed evidence could not be evaluated."); }
    finally { setBusy(false); }
  };
  const downloadDispositions = () => {
    if (!loaded || !canExportDisposition) return;
    const record = {
      schemaVersion: "experience-correction-review/v1",
      reportDigest: loaded.envelope.reportDigest,
      scenario: loaded.envelope.scenario,
      build: loaded.envelope.build,
      reviewerClass: "operator",
      reviewedAt: new Date().toISOString(),
      dispositions: loaded.candidates.map((candidate) => ({
        candidateId: candidate.id,
        decision: dispositions[candidate.id]!.decision,
        reason: dispositions[candidate.id]!.reason.trim(),
      })),
      note: "Review record only. This export does not write to the private EX1 corpus or promote gold labels.",
    };
    const url = URL.createObjectURL(new Blob([JSON.stringify(record, null, 2)], { type: "application/json" }));
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "experience-correction-review.json";
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  };

  return (
    <section aria-label="Observed behavior evidence" className="grid gap-3 rounded-panel border border-app-border bg-app-surface p-4">
      <div>
        <h2 className="font-semibold">Observed behavior · advisory</h2>
        <p className="mt-1 text-sm text-app-muted-foreground">Import the five local `vrooli-experience report` outputs. Evidence is shown only when session, scenario, build, schema, and digest agree. It never changes a verdict or image fidelity label.</p>
      </div>
      <label className="grid gap-1 text-sm">Build being reviewed<input aria-label="Expected build identity" value={expectedBuild} onChange={(event) => onExpectedBuildChange(event.target.value)} className="rounded-control border border-app-border bg-app-surface px-3 py-2" placeholder="sha256:…" /></label>
      <div className="grid gap-2 sm:grid-cols-2">
        <label className="grid gap-1 text-sm">friction-report.json<input aria-label="Friction report file" type="file" accept="application/json,.json" onChange={onFile("report")} /></label>
        <label className="grid gap-1 text-sm">observed-behavior.json<input aria-label="Observed behavior file" type="file" accept="application/json,.json" onChange={onFile("envelope")} /></label>
        <label className="grid gap-1 text-sm">advisory-findings.json<input aria-label="Advisory findings file" type="file" accept="application/json,.json" onChange={onFile("findings")} /></label>
        <label className="grid gap-1 text-sm">correction-candidates.json<input aria-label="Correction candidates file" type="file" accept="application/json,.json" onChange={onFile("candidates")} /></label>
        <label className="grid gap-1 text-sm">packet-observed.json<input aria-label="Observed packet file" type="file" accept="application/json,.json" onChange={onFile("packet")} /></label>
      </div>
      <button type="button" disabled={!canLoad || busy} onClick={() => void load()} className="w-fit rounded-control border border-app-border px-3 py-2 text-sm disabled:opacity-50">{busy ? "Checking evidence…" : "Evaluate observed evidence"}</button>
      {error ? <p role="status" className="text-sm text-app-warning">Not evaluated: {error}</p> : null}
      {loaded ? <div role="region" aria-label="Evaluated advisory evidence" className="grid gap-3 border-t border-app-border pt-3">
        <p className="text-sm">Advisory evidence for build <code>{loaded.envelope.build}</code>, {loaded.envelope.window.sessions} session(s), digest <code className="break-all">{loaded.envelope.reportDigest}</code>. Not a verdict.</p>
        <p className="text-xs text-app-muted-foreground">Bound session IDs: {loaded.envelope.sessions.map((session) => session.id).join(", ")}</p>
        {loaded.findings.length === 0 ? <p className="text-sm text-app-muted-foreground">No advisory findings in this report.</p> : <ul className="grid gap-2">{loaded.findings.map((finding) => <li key={finding.id} className="rounded-control border border-app-border p-3"><span className="text-xs font-semibold uppercase">{finding.severity} · advisory only</span><h3 className="mt-1 font-medium">{finding.title}</h3><p className="mt-1 text-sm">{finding.evidence}</p><p className="mt-1 text-sm text-app-muted-foreground">{finding.suggestion}</p><p className="mt-1 text-xs text-app-muted-foreground">{finding.route ?? "route not bound"} · {finding.element?.testId ?? finding.element?.name ?? "element not bound"}</p></li>)}</ul>}
        <section aria-label="EX1 correction candidate review" className="grid gap-3 rounded-panel border border-app-border p-3"><div><h3 className="font-semibold">Correction candidates · review required</h3><p className="text-sm text-app-muted-foreground">These are questions, not corrections. A recorded review downloads a separate disposition file and never edits the private corpus.</p></div>
          {loaded.candidates.length === 0 ? <p className="text-sm text-app-muted-foreground">No correction candidates in this report.</p> : loaded.candidates.map((candidate) => <article key={candidate.id} className="grid gap-2 border-t border-app-border pt-3"><p className="text-xs">{candidate.severity} · candidate · {candidate.kind} · <code>{candidate.id}</code></p><p className="text-sm">{candidate.evidence}</p><p className="text-xs text-app-muted-foreground">{candidate.route ?? "route not bound"} · {candidate.element?.testId ?? candidate.element?.name ?? "element not bound"}</p><label className="grid gap-1 text-sm">Review decision<select aria-label={`Review decision ${candidate.id}`} value={dispositions[candidate.id]?.decision ?? ""} onChange={(event) => setDispositions((current) => ({ ...current, [candidate.id]: { decision: event.target.value as ReviewDecision, reason: current[candidate.id]?.reason ?? "" } }))} className="rounded-control border border-app-border bg-app-surface px-3 py-2"><option value="">Needs human review</option><option value="promote">Recommend promote to gold correction</option><option value="decline">Decline as gold correction</option></select></label><label className="grid gap-1 text-sm">Reason<textarea aria-label={`Review reason ${candidate.id}`} value={dispositions[candidate.id]?.reason ?? ""} onChange={(event) => setDispositions((current) => ({ ...current, [candidate.id]: { decision: current[candidate.id]?.decision ?? "", reason: event.target.value } }))} rows={2} className="rounded-control border border-app-border bg-app-surface px-3 py-2" /></label></article>)}
          {loaded.candidates.length > 0 ? <button type="button" disabled={!canExportDisposition} onClick={downloadDispositions} className="w-fit rounded-control border border-app-border px-3 py-2 text-sm disabled:opacity-50">Download review disposition (no corpus write)</button> : null}
        </section>
        <aside role="region" aria-label="Optional observed packet block" className="rounded-control bg-app-surface-muted p-3"><h3 className="font-medium">Optional packet context</h3><p className="mt-1 text-sm">{loaded.packet.headline}</p><p className="mt-1 text-sm">Time cost: {Math.round(loaded.packet.timeCostMs / 1000)}s · friction: {loaded.packet.topFriction ? `${loaded.packet.topFriction.kind} (${loaded.packet.topFriction.count})` : "none"}</p><p className="mt-1 text-xs text-app-muted-foreground">{loaded.packet.caveat}</p></aside>
      </div> : null}
    </section>
  );
}
