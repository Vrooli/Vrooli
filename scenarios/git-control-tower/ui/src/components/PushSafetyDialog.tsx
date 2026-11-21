import { useEffect, useRef, useState } from "react";
import type { PushRecoveryArtifact, PushSafetyReport } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { getPushRecovery, inspectPushSafety, preparePushRecovery } from "../lib/api-push-safety";
import { useUntrackBinary } from "../lib/hooks";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { AlertTriangle, ArrowRight, GitBranch, HardDrive, Loader2, RefreshCw, ShieldCheck } from "lucide-react";
import { Button } from "./ui/button";

interface Props { repoId?: string; focus?: "staged" | "outgoing"; onClose: () => void; onPush: () => void }
const mib = (n: bigint) => `${(Number(n) / 1048576).toFixed(2)} MiB`;

// ignoreTargetForPath mirrors the server's ignore-ownership rule: the scenario /
// resource / package directory owns its own .gitignore, and a repo-root file
// falls back to the root. Kept in sync with tracked_binaries_service.go so the
// drawer's untrack lands the ignore rule in the convention-correct file.
const UNTRACK_OWNER_ROOTS = ["scenarios", "resources", "packages", "templates"];
function ignoreTargetForPath(path: string): { ownerDir: string; pattern: string } {
  const parts = path.split("/");
  const [root = "", scope = ""] = parts;
  if (parts.length >= 3 && UNTRACK_OWNER_ROOTS.includes(root)) {
    return { ownerDir: `${root}/${scope}`, pattern: `/${parts.slice(2).join("/")}` };
  }
  return { ownerDir: "", pattern: `/${path}` };
}

// UntrackKeepLocalButton untracks a staged file (removes it from the index and
// adds an ignore rule) while leaving it on disk. It is offered only for staged
// content: untracking stops a file from entering history, but cannot remove one
// already committed -- that needs the recovery-prepare flow.
function UntrackKeepLocalButton({ repoId, path, onDone }: { repoId?: string; path: string; onDone: () => void }) {
  const untrack = useUntrackBinary(repoId);
  const target = ignoreTargetForPath(path);
  return <div className="mt-1">
    <Button variant="outline" className="h-7 rounded-lg px-2 text-[11px]" disabled={untrack.isPending}
      onClick={() => untrack.mutate({ path, owner_dir: target.ownerDir, ignore_pattern: target.pattern }, { onSuccess: onDone })}>
      {untrack.isPending ? "Untracking…" : "Untrack (keep local)"}
    </Button>
    {untrack.isError && <p className="mt-1 text-[11px] text-red-400">Untrack failed: {untrack.error.message}</p>}
  </div>;
}

// Read-only inspection is separate from explicit artifact preparation. This
// component has no activation, reset, stash, or publication recovery operation.
export function PushSafetyDialog({ repoId, focus = "outgoing", onClose, onPush }: Props) {
  const [report, setReport] = useState<PushSafetyReport>();
  const [artifact, setArtifact] = useState<PushRecoveryArtifact>();
  const [busy, setBusy] = useState<"inspection" | "preparation" | "status" | null>("inspection");
  const [error, setError] = useState("");
  const [consent, setConsent] = useState(false);
  const [operationId, setOperationId] = useState("");
  const generation = useRef(0);

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

  // When opened from a staged-file review link, present the staged index — the
  // content that actually triggered the warning — instead of outgoing history,
  // which is a separate (and here empty) set of already-committed files.
  const stagedFiles = report?.stagedFiles ?? [];
  const stagedLargest = stagedFiles.reduce((m, f) => (f.bytes > m ? f.bytes : m), 0n);
  const stagedBlocked = stagedFiles.some((f) => f.blocked);
  const showStaged = focus === "staged" && Boolean(report?.stagedComplete) && stagedFiles.length > 0;

  return <ResponsiveDialog
    open
    onClose={onClose}
    title={showStaged ? (stagedBlocked ? "Staged file would block push" : "Review staged files") : report?.state === "blocked" ? "Push blocked by oversized files" : "Review outgoing history"}
    ariaLabel="Push safety review"
    closeLabel="Close recovery review"
    size="lg"
    contentPadding="none"
    avoidKeyboard
    testId="push-safety-dialog"
    footer={
      <div className="flex flex-nowrap items-center justify-end gap-2">
        <Button variant="outline" className="mr-auto h-9 gap-2 rounded-xl border-transparent px-2 text-xs text-slate-400" disabled={Boolean(busy)} onClick={() => void refresh()} aria-label="Refresh inspection"><RefreshCw className="h-3.5 w-3.5" /><span className="hidden sm:inline">Refresh inspection</span></Button>
        <Button variant="outline" className="h-9 shrink-0 rounded-xl text-xs" onClick={onClose}>Close</Button>
        {report?.canPrepare && artifact?.fingerprint !== report.fingerprint && <Button className="h-9 shrink-0 gap-2 rounded-xl bg-blue-600 text-xs text-white shadow-lg shadow-blue-950/40 hover:bg-blue-500" disabled={Boolean(busy) || !consent} onClick={() => void prepare()}>Prepare recovery<ArrowRight className="h-3.5 w-3.5" /></Button>}
        {!showStaged && report?.complete && report.state !== "blocked" && <Button className="h-9 shrink-0 gap-2 rounded-xl bg-blue-600 text-xs text-white shadow-lg shadow-blue-950/40 hover:bg-blue-500" disabled={Boolean(busy)} onClick={onPush}>Continue to push</Button>}
      </div>
    }
  >
      <div className="space-y-4 p-5 text-xs leading-relaxed">
      <p className="text-xs leading-relaxed text-slate-400">{showStaged ? "Review the staged files that would be included in your next commit." : "Review the files and choose how to prepare your history."}</p>
      <div className="flex items-start gap-2.5 rounded-xl border border-blue-900/60 bg-blue-950/20 p-3 text-blue-200/80">
        <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-blue-300" />
        <p><span className="mb-0.5 block font-medium text-blue-100">Your local commits remain available.</span>Inspection and preparation leave your active branch, staged changes, and working files unchanged.</p>
      </div>
      {busy && <p role="status" className="flex items-start gap-2 rounded-xl border border-slate-800 bg-slate-950/40 p-3 text-slate-300"><Loader2 className="mt-0.5 h-4 w-4 shrink-0 animate-spin" />{busy === "inspection" ? "Checking the live destination and outgoing history…" : busy === "status" ? "Finding retained preparation and checking bundle contents. Large bundles can take time to read…" : "Preparing an isolated copy and verifying recovery bundles. This can take several minutes and requires disk space for copies of committed history. Closing this screen does not cancel the server-owned preparation; reopen it and check preparation status."}</p>}
      {error && <p role="alert" className="break-words rounded-xl border border-amber-800/50 bg-amber-950/20 p-3 text-amber-200">{error}</p>}
      {report && (showStaged ? <>
        <p role="status" className="text-slate-400">{report.stagedReason}</p>
        <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
          <p className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500"><HardDrive className="h-3.5 w-3.5" />Largest staged file</p>
          {report.limit > 0n ? (() => {
            const pct = Math.min(100, Math.round((Number(stagedLargest) / Number(report.limit)) * 100));
            const over = stagedLargest > report.limit;
            const near = pct >= 80;
            return <>
              <p className="mt-1 font-medium text-slate-100">{mib(stagedLargest)} <span className="font-normal text-slate-500">of {mib(report.limit)} limit</span></p>
              <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-slate-800" role="meter" aria-label="Largest staged file vs limit" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}>
                <div className={`h-full rounded-full ${over ? "bg-red-500" : near ? "bg-amber-400" : "bg-emerald-500"}`} style={{ width: `${Math.max(pct, 2)}%` }} />
              </div>
            </>;
          })() : <p className="mt-1 font-medium text-slate-100">{mib(stagedLargest)} <span className="font-normal text-slate-500">· host limit unknown</span></p>}
        </div>
        <div className="space-y-2">
          {[...stagedFiles].sort((a, b) => (b.bytes > a.bytes ? 1 : b.bytes < a.bytes ? -1 : 0)).map((file) => <div key={file.oid + file.paths.join()} className={`space-y-1 rounded-xl border p-3 ${file.blocked ? "border-red-800/40 bg-red-950/10" : "border-amber-800/40 bg-amber-950/10"}`}>
            {file.paths.map((path) => <p key={path} className="break-all whitespace-pre-wrap font-mono text-[11px] text-slate-200">{path}</p>)}
            <p className={`inline-flex rounded-md px-2 py-0.5 text-[11px] font-medium ${file.blocked ? "bg-red-500/10 text-red-200" : "bg-amber-500/10 text-amber-200"}`}>{mib(file.bytes)} · {file.blocked ? `exceeds ${report.limit > 0n ? mib(report.limit) : "the"} limit — would block push if committed` : "large staged file"}</p>
            {file.paths[0] && <UntrackKeepLocalButton repoId={repoId} path={file.paths[0]} onDone={() => void refresh()} />}
          </div>)}
        </div>
        <div className="space-y-2 rounded-xl border border-slate-800 bg-slate-950/25 p-3 text-slate-400">
          <p className="font-medium text-slate-300">How to resolve before committing</p>
          <p>These files are staged in the index only — not in any commit yet, so no history rewrite is needed. Any one of these clears the block:</p>
          <ul className="list-disc space-y-1 pl-5">
            <li>Unstage the file, then add it to an ignore rule if it should never be tracked (for example an experiment output or test fixture).</li>
            <li>Keep it as a local-only generated artifact that lives outside Git.</li>
            <li>For a versioned large asset, plan a Git LFS migration before committing it.</li>
          </ul>
          <p className="flex items-start gap-2"><ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-blue-300" />Reviewing here changes nothing. Your staged changes and working files are unchanged.</p>
        </div>
      </> : <>
        <p role="status" className="text-slate-400">{report.reason}</p>
        {report.state === "blocked" && <ol aria-label="Recovery steps" className="grid grid-cols-3 gap-2 text-[11px]">
          {["Review snapshot", "Prepare with approval", "Apply · unavailable"].map((step, index) => <li key={step} className={`flex items-center gap-2 rounded-lg px-2 py-2 ${index === 0 ? "bg-blue-500/10 text-blue-200" : "text-slate-500"}`}><span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-current/20 text-[10px]">{index + 1}</span>{step}</li>)}
        </ol>}
        <div className="grid gap-2 sm:grid-cols-2">
          <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3"><p className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500"><GitBranch className="h-3.5 w-3.5" />Destination</p><p className="mt-1 break-all font-medium text-slate-100">{report.remote}/{report.branch}</p><p className="mt-0.5 text-slate-500">{report.commits.length} outgoing commits</p></div>
          <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
            <p className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500"><HardDrive className="h-3.5 w-3.5" />Largest file</p>
            {report.limit > 0n ? (() => {
              const pct = Math.min(100, Math.round((Number(report.largestBlobBytes) / Number(report.limit)) * 100));
              const over = report.largestBlobBytes > report.limit;
              const near = pct >= 80;
              return <>
                <p className="mt-1 font-medium text-slate-100">{mib(report.largestBlobBytes)} <span className="font-normal text-slate-500">of {mib(report.limit)} limit</span></p>
                <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-slate-800" role="meter" aria-label="Largest file vs limit" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}>
                  <div className={`h-full rounded-full ${over ? "bg-red-500" : near ? "bg-amber-400" : "bg-emerald-500"}`} style={{ width: `${Math.max(pct, 2)}%` }} />
                </div>
                <p className="mt-1 text-slate-500">{mib(report.outgoingBytes)} total across {report.commits.length} commits</p>
              </>;
            })() : <>
              <p className="mt-1 font-medium text-slate-100">{mib(report.largestBlobBytes)}</p>
              <p className="mt-0.5 text-slate-500">{mib(report.outgoingBytes)} total · host limit unknown</p>
            </>}
          </div>
        </div>
        <details className="text-[11px] text-slate-500"><summary className="cursor-pointer hover:text-slate-300">Snapshot identifiers</summary><p className="mt-2 break-all font-mono">Source: {report.head || "unavailable"}<br />Remote base: {report.base || "no existing branch confirmed"}</p></details>
        {report.files.length > 0 && <details open={report.state === "blocked"} className="rounded-xl border border-slate-800 bg-slate-950/25 p-3 text-slate-400">
          <summary className="cursor-pointer font-medium text-slate-300">Large files in outgoing history ({report.files.length})</summary>
          <div className="mt-2 space-y-2">
          {report.files.map((file) => <div key={file.oid} className="space-y-2 rounded-xl border border-amber-800/40 bg-amber-950/10 p-3">
            {file.paths.map((path) => <p key={path} className="break-all whitespace-pre-wrap font-mono text-[11px] text-slate-200">{path}</p>)}
            <p className="inline-flex rounded-md bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-200">{mib(file.bytes)} · {file.blocked ? "exceeds limit" : "large file warning"}</p>
            <p className="text-[11px] text-slate-500">First observed in outgoing history: {file.commits[0]?.slice(0, 12) ?? "unavailable"}. Present in {file.commits.length} outgoing commits.</p>
          </div>)}
          </div>
        </details>}
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
      </>)}
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
      <details open={Boolean(artifact)} className="rounded-xl border border-slate-800 bg-slate-950/25 p-3">
        <summary className="cursor-pointer font-medium text-slate-300">Already prepared a recovery?</summary>
        <div className="mt-2 space-y-2">
        <label className="block text-[11px] text-slate-400">Operation ID (optional)
          <input className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-950/60 px-3 py-2 font-mono text-xs text-slate-200 outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500" value={operationId} disabled={Boolean(busy)} onChange={(event) => setOperationId(event.target.value)} />
        </label>
        <p className="text-[11px] text-slate-500">Leave blank to find this repository’s most recently updated preparation, even after new commits. Enter a saved operation ID to check an older one. Checking status does not start or apply recovery.</p>
        <Button variant="outline" className="h-8 rounded-lg text-xs" disabled={Boolean(busy)} onClick={() => void checkPreparation()}>Check preparation status</Button>
        </div>
      </details>
      </div>
  </ResponsiveDialog>;
}
