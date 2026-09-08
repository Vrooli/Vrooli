import { useEffect, useRef, useState } from "react";
import type { PushRecoveryArtifact, PushSafetyReport } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { getPushRecovery, inspectPushSafety, preparePushRecovery } from "../lib/api-push-safety";
import { AlertTriangle, ArrowRight, GitBranch, HardDrive, Loader2, RefreshCw, ShieldCheck, X } from "lucide-react";
import { Button } from "./ui/button";

interface Props { repoId?: string; onClose: () => void; onPush: () => void }
const mib = (n: bigint) => `${(Number(n) / 1048576).toFixed(2)} MiB`;

// Read-only inspection is separate from explicit artifact preparation. This
// component has no activation, reset, stash, or publication recovery operation.
export function PushSafetyDialog({ repoId, onClose, onPush }: Props) {
  const [report, setReport] = useState<PushSafetyReport>();
  const [artifact, setArtifact] = useState<PushRecoveryArtifact>();
  const [busy, setBusy] = useState<"inspection" | "preparation" | "status" | null>("inspection");
  const [error, setError] = useState("");
  const [consent, setConsent] = useState(false);
  const [operationId, setOperationId] = useState("");
  const dialog = useRef<HTMLDivElement>(null);
  const generation = useRef(0);

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    dialog.current?.focus();
    return () => { previous?.focus(); };
  }, []);

  useEffect(() => {
    const sequence = generation;
    const token = ++sequence.current;
    setReport(undefined); setArtifact(undefined); setOperationId(""); setConsent(false); setError(""); setBusy("inspection");
    inspectPushSafety(repoId).then((value) => {
      if (token === generation.current) setReport(value);
    }).catch((e: unknown) => { if (token === generation.current) setError(e instanceof Error ? e.message : "Inspection unavailable."); })
      .finally(() => { if (token === generation.current) setBusy(null); });
    return () => { sequence.current++; };
  }, [repoId]);

  const refresh = async () => {
    const token = ++generation.current;
    setBusy("inspection"); setError(""); setReport(undefined); setConsent(false);
    try { const value = await inspectPushSafety(repoId); if (token === generation.current) setReport(value); }
    catch (e) { if (token === generation.current) setError(e instanceof Error ? e.message : "Inspection unavailable."); }
    finally { if (token === generation.current) setBusy(null); }
  };
  const checkPreparation = async () => {
    if (busy) return;
    const token = ++generation.current;
    setBusy("status"); setError(""); setArtifact(undefined);
    try { const value = await getPushRecovery(operationId.trim(), repoId); if (token === generation.current) setArtifact(value); }
    catch (e) { if (token === generation.current) setError(e instanceof Error ? e.message : "Preparation status unavailable."); }
    finally { if (token === generation.current) setBusy(null); }
  };
  const prepare = async () => {
    if (!report || !consent || busy) return;
    const token = ++generation.current;
    setBusy("preparation"); setError(""); setOperationId(report.fingerprint);
    try { const result = await preparePushRecovery(report, repoId); if (token === generation.current) setArtifact(result); }
    catch (e) { if (token === generation.current) { setError(e instanceof Error ? e.message : "Preparation did not complete."); setConsent(false); } }
    finally { if (token === generation.current) setBusy(null); }
  };

  return <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/70 p-3 backdrop-blur-sm sm:p-6">
    <div ref={dialog} role="dialog" aria-modal="true" aria-labelledby="push-safety-title" tabIndex={-1}
      className="flex max-h-[90dvh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-slate-700/70 bg-slate-900 text-slate-200 shadow-2xl shadow-black/50 outline-none"
      onKeyDown={(event) => {
        if (event.key === "Escape") onClose();
        if (event.key === "Tab") {
          const items = dialog.current?.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), summary, [tabindex="0"]');
          const first = items?.[0]; const last = items?.[items.length - 1];
          if (event.shiftKey && (document.activeElement === first || document.activeElement === dialog.current)) { event.preventDefault(); last?.focus(); }
          else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
        }
      }}>
      <header className="flex shrink-0 items-start gap-3 border-b border-slate-800 bg-slate-950/40 px-5 py-4">
        <div className="rounded-xl border border-amber-500/20 bg-amber-500/10 p-2.5 text-amber-300"><ShieldCheck className="h-5 w-5" /></div>
        <div className="min-w-0 flex-1">
          <p className="text-[10px] font-semibold uppercase tracking-[0.18em] text-amber-300/80">Push safety review</p>
          <h2 id="push-safety-title" className="mt-1 text-lg font-semibold tracking-tight text-slate-50">{report?.state === "blocked" ? "Push blocked by oversized files" : "Review outgoing history"}</h2>
          <p className="mt-1 text-xs leading-relaxed text-slate-400">Review the files and choose how to prepare your history.</p>
        </div>
        <button type="button" onClick={onClose} aria-label="Close recovery review" className="rounded-lg p-1.5 text-slate-500 transition hover:bg-slate-800 hover:text-slate-200"><X className="h-4 w-4" /></button>
      </header>
      <div className="min-h-0 space-y-4 overflow-y-auto p-5 text-xs leading-relaxed">
      <div className="flex items-start gap-2.5 rounded-xl border border-blue-900/60 bg-blue-950/20 p-3 text-blue-200/80">
        <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-blue-300" />
        <p><span className="mb-0.5 block font-medium text-blue-100">Your local commits remain available.</span>Inspection and preparation leave your active branch, staged changes, and working files unchanged.</p>
      </div>
      {busy && <p role="status" className="flex items-start gap-2 rounded-xl border border-slate-800 bg-slate-950/40 p-3 text-slate-300"><Loader2 className="mt-0.5 h-4 w-4 shrink-0 animate-spin" />{busy === "inspection" ? "Checking the live destination and outgoing history…" : busy === "status" ? "Finding retained preparation and checking bundle contents. Large bundles can take time to read…" : "Preparing an isolated copy and verifying recovery bundles. This can take several minutes and requires disk space for copies of committed history. Closing this screen does not cancel the server-owned preparation; reopen it and check preparation status."}</p>}
      {error && <p role="alert" className="break-words rounded-xl border border-amber-800/50 bg-amber-950/20 p-3 text-amber-200">{error}</p>}
      {report && <>
        <p role="status" className="text-slate-400">{report.reason}</p>
        <ol aria-label="Recovery steps" className="grid grid-cols-3 gap-2 text-[11px]">
          {["Review snapshot", "Prepare with approval", "Apply · unavailable"].map((step, index) => <li key={step} className={`flex items-center gap-2 rounded-lg px-2 py-2 ${index === 0 ? "bg-blue-500/10 text-blue-200" : "text-slate-500"}`}><span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-current/20 text-[10px]">{index + 1}</span>{step}</li>)}
        </ol>
        <div className="grid gap-2 sm:grid-cols-2">
          <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3"><p className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500"><GitBranch className="h-3.5 w-3.5" />Destination</p><p className="mt-1 break-all font-medium text-slate-100">{report.remote}/{report.branch}</p><p className="mt-0.5 text-slate-500">{report.commits.length} outgoing commits</p></div>
          <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3"><p className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500"><HardDrive className="h-3.5 w-3.5" />File size limit</p><p className="mt-1 font-medium text-slate-100">{report.limit > 0n ? mib(report.limit) : "Host limit unknown"}</p><p className="mt-0.5 text-slate-500">Includes files in earlier commits</p></div>
        </div>
        <details className="text-[11px] text-slate-500"><summary className="cursor-pointer hover:text-slate-300">Snapshot identifiers</summary><p className="mt-2 break-all font-mono">Source: {report.head || "unavailable"}<br />Remote base: {report.base || "no existing branch confirmed"}</p></details>
        <h3 className="text-xs font-semibold text-slate-100">Outgoing history</h3>
        {report.files.map((file) => <div key={file.oid} className="space-y-2 rounded-xl border border-amber-800/40 bg-amber-950/10 p-3">
          {file.paths.map((path) => <p key={path} className="break-all whitespace-pre-wrap font-mono text-[11px] text-slate-200">{path}</p>)}
          <p className="inline-flex rounded-md bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-200">{mib(file.bytes)} · {file.blocked ? "exceeds limit" : "large file warning"}</p>
          <p className="text-[11px] text-slate-500">First observed in outgoing history: {file.commits[0]?.slice(0, 12) ?? "unavailable"}. Present in {file.commits.length} outgoing commits.</p>
        </div>)}
        <details className="rounded-xl border border-slate-800 bg-slate-950/25 p-3 text-slate-400"><summary className="cursor-pointer font-medium text-slate-300">Staged content (not part of recovery preparation)</summary>
          <p>{report.stagedComplete ? report.stagedReason : "Staged file-size check unverified. Check staged content before committing."}</p>
          {report.stagedFiles?.map(file => <p key={file.oid + file.paths.join()} className="break-all text-amber-200">{file.paths.join(", ")} · {mib(file.bytes)} · {file.blocked ? "Would block push if committed" : "Large staged file"}</p>)}
          <p>Local commits remain allowed. Preparation only handles the blocked paths in committed outgoing history above; it does not change your index.</p>
        </details>
        {report.state === "blocked" && <>
          <p className="flex items-start gap-2 text-slate-400"><AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-400/70" />Deleting the file in a new commit or adding an ignore rule alone will not remove it from earlier commits.</p>
          <p className="text-slate-400">{report.recoveryReason}</p>
          <details className="rounded-xl border border-slate-800 bg-slate-950/25 p-3 text-slate-400"><summary className="cursor-pointer font-medium text-slate-300">Recovery choices and safeguards</summary>
            <ul className="list-disc pl-5 space-y-2 mt-2">
              <li>Generated artifact: keep the local file and prepare history without the exact listed paths. All versions of those paths are excluded from the outgoing commits.</li>
              <li>Versioned asset: plan a Git LFS migration instead. It also changes commit IDs and requires LFS storage. This workflow does not perform LFS migration.</li>
              <li>Other agents can keep editing during preparation. A new commit or remote change requires a fresh preview.</li>
              <li>Original and replacement commits are retained separately. Messages, authors, timestamps, and other paths are verified. Rewritten signatures cannot remain valid.</li>
              <li>These bundles protect committed history only. Before application, pause shared-checkout writers and capture a consistent backup of the index and live files, including untracked and relevant ignored files.</li>
              <li>Preparation does not add an ignore rule or change packaging. Review those changes before applying. Application and rollback need separate review; this screen cannot activate or push a repaired branch.</li>
            </ul>
          </details>
          {report.canPrepare && artifact?.fingerprint !== report.fingerprint && <label className="flex cursor-pointer items-start gap-3 rounded-xl border border-blue-800/50 bg-blue-950/20 p-3 text-xs leading-relaxed text-blue-100"><input className="mt-0.5 h-4 w-4 shrink-0 accent-blue-500" type="checkbox" checked={consent} disabled={Boolean(busy)} onChange={(e) => setConsent(e.target.checked)} />I approve preparing replacement commits without the blocked paths listed under outgoing history. My active workspace will remain unchanged; these artifacts do not back up uncommitted work.</label>}
        </>}
      </>}
      {artifact && <section aria-label="Recovery preparation result" className="space-y-2 rounded-xl border border-slate-700 bg-slate-950/40 p-3">
        <h3 className="font-semibold">{artifact.state === "prepared" ? "Prepared artifacts checked — not applied" : `Recovery state: ${artifact.state}`}</h3>
        <p>{artifact.message}</p>
        <p className="text-xs break-all">Operation ID: {artifact.fingerprint || "none"}</p>
        {artifact.head && <p className="text-xs break-all">Prepared source: {artifact.head}<br />Recorded remote base: {artifact.base || "unavailable"}</p>}
        <p>These artifacts do not back up uncommitted work.</p>
        <p className="text-sm break-all">Original history: {artifact.originalBundle || "not yet verified"}<br />Replacement history: {artifact.repairedBundle || "not yet verified"}</p>
        {artifact.signaturesRemoved && <p>Commit signatures were removed from replacement commits. Re-sign and renew evidence tied to old commit IDs before publication.</p>}
        <ul className="font-mono text-xs">{artifact.mappings.map((m) => <li key={m.original}>{m.original.slice(0, 12)} → {m.replacement.slice(0, 12)}</li>)}</ul>
      </section>}
      <section aria-label="Find retained preparation" className="space-y-2 rounded-xl border border-slate-800 bg-slate-950/25 p-3">
        <p className="font-medium text-slate-300">Already prepared a recovery?</p><label className="block text-[11px] text-slate-400">Operation ID (optional)
          <input className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-950/60 px-3 py-2 font-mono text-xs text-slate-200 outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500" value={operationId} disabled={Boolean(busy)} onChange={(event) => setOperationId(event.target.value)} />
        </label>
        <p className="text-[11px] text-slate-500">Leave blank to find this repository’s most recently updated preparation, even after new commits. Enter a saved operation ID to check an older one. Checking status does not start or apply recovery.</p>
        <Button variant="outline" className="h-8 rounded-lg text-xs" disabled={Boolean(busy)} onClick={() => void checkPreparation()}>Check preparation status</Button>
      </section>
      </div>
      <footer className="flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-slate-800 bg-slate-950/40 px-5 py-4">
        <Button variant="outline" className="mr-auto h-9 gap-2 rounded-xl border-transparent text-xs text-slate-400" disabled={Boolean(busy)} onClick={() => void refresh()}><RefreshCw className="h-3.5 w-3.5" />Refresh inspection</Button>
        <Button variant="outline" className="h-9 rounded-xl text-xs" onClick={onClose}>Close</Button>
        {report?.canPrepare && artifact?.fingerprint !== report.fingerprint && <Button className="h-9 gap-2 rounded-xl bg-blue-600 text-xs text-white shadow-lg shadow-blue-950/40 hover:bg-blue-500" disabled={Boolean(busy) || !consent} onClick={() => void prepare()}>Prepare isolated recovery<ArrowRight className="h-3.5 w-3.5" /></Button>}
        {report?.complete && report.state !== "blocked" && <Button className="h-9 gap-2 rounded-xl bg-blue-600 text-xs text-white shadow-lg shadow-blue-950/40 hover:bg-blue-500" disabled={Boolean(busy)} onClick={onPush}>Continue to push</Button>}
      </footer>
    </div>
  </div>;
}
