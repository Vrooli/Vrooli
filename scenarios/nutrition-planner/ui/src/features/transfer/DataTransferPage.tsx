import { useState } from "react";
import { ensureWorkspace } from "../../api/workspace";
import { applyWorkspaceImport, exportWorkspace, previewWorkspaceImport } from "../../api/portability";

export function DataTransferPage() {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [preview, setPreview] = useState<{ valid: boolean; format: string; schemaVersion: number; recordCount: number; recordKinds: string[]; omissions: string[]; errors: string[] } | null>(null);
  const [pendingContent, setPendingContent] = useState("");
  const [workspace, setWorkspace] = useState<{ id: string; revision: bigint } | null>(null);

  async function downloadBackup() {
    setBusy(true);
    try {
      const workspace = await ensureWorkspace();
      const result = await exportWorkspace(workspace.id);
      const url = URL.createObjectURL(new Blob([result.content], { type: "application/json" }));
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = result.filename;
      anchor.click();
      URL.revokeObjectURL(url);
      setMessage(`Backup ready. Omitted domains: ${result.omissions.length}.`);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Unable to export workspace.");
    } finally {
      setBusy(false);
    }
  }

  async function inspectFile(file: File | undefined) {
    if (!file) return;
    setBusy(true);
    setPreview(null);
    try {
      const workspace = await ensureWorkspace();
      const contentJson = await file.text();
      setWorkspace({ id: workspace.id, revision: workspace.revision });
      setPendingContent(contentJson);
      setPreview(await previewWorkspaceImport({ workspaceId: workspace.id, contentJson }));
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Unable to inspect workspace import.");
    } finally {
      setBusy(false);
    }
  }

  async function restore() {
    if (!workspace || !preview?.valid || !pendingContent) return;
    setBusy(true);
    try {
      const result = await applyWorkspaceImport({ workspaceId: workspace.id, expectedWorkspaceRevision: workspace.revision, contentJson: pendingContent, idempotencyKey: crypto.randomUUID() });
      setMessage(`Restore applied: ${result.recipesApplied} recipes. Checkpoint ${result.checkpointId} is available for recovery.`);
      setPreview(null);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Unable to apply workspace restore.");
    } finally {
      setBusy(false);
    }
  }

  return <section aria-labelledby="transfer-heading" className="flex flex-col gap-6"><header><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">Data / Transfer</p><h1 id="transfer-heading" className="text-3xl font-semibold text-slate-900">Backup and restore</h1><p className="mt-2 text-slate-600">Export your supported workspace data or inspect a backup before any live write. Authority, credentials, and unsupported operational domains are never imported.</p></header><div className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Export backup</h2><button type="button" className="mt-4 min-h-11 rounded bg-blue-700 px-4 font-medium text-white disabled:opacity-50" onClick={() => void downloadBackup()} disabled={busy}>{busy ? "Working…" : "Download workspace backup"}</button></div><div className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Inspect backup before restore</h2><label className="mt-4 block text-sm font-medium text-slate-700" htmlFor="workspace-import">Choose a daily.workspace JSON file</label><input id="workspace-import" className="mt-2 block min-h-11" type="file" accept="application/json,.json" onChange={(event) => void inspectFile(event.target.files?.[0])} disabled={busy} />{preview && <div className="mt-4 rounded border p-4" role={preview.valid ? "status" : "alert"}><p className="font-medium">{preview.valid ? "Import is valid and staged for review" : "Import cannot be applied"}</p>{preview.valid && <><p className="mt-2 text-sm text-slate-600">{preview.format} v{preview.schemaVersion} · {preview.recordCount} records · kinds: {preview.recordKinds.join(", ")}</p><button type="button" className="mt-4 min-h-11 rounded bg-amber-700 px-4 font-medium text-white disabled:opacity-50" onClick={() => void restore()} disabled={busy}>Restore this validated backup</button></>}{preview.omissions.length > 0 && <p className="mt-2 text-sm text-slate-600">Omissions: {preview.omissions.join(", ")}</p>}{preview.errors.length > 0 && <ul className="mt-2 list-disc pl-5 text-sm text-red-700">{preview.errors.map((error) => <li key={error}>{error}</li>)}</ul>}</div>}{message && <p className="mt-4 text-sm text-slate-600" role="status">{message}</p>}</div></section>;
}
