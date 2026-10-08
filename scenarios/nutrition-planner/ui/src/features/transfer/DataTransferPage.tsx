import { useEffect, useState } from "react";
import { ensureWorkspace } from "../../api/workspace";
import { applyRecipesImport, applyWorkspaceImport, exportGroceriesCSV, exportRecipePDF, exportRecipes, exportWeeklyPDF, exportWorkspace, getRestoreCheckpoint, recoverRestoreCheckpoint, previewRecipesImport, previewWorkspaceImport } from "../../api/portability";
import { listRecipes } from "../../api/recipes";
import { selectors } from "../../consts/selectors";

function messageFor(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

const RESTORE_FAMILIES = [
  ["workspace", "Workspace"], ["recipe", "Recipes"], ["recipe_revision", "Recipe history"], ["plan", "Plan"],
  ["shopping_state", "Shopping state"], ["nutrition_target", "Nutrition targets"], ["profile", "Profile"],
  ["intake_event", "Intake events"], ["supplement_schedule", "Supplement schedules"],
  ["supplement_schedule_revision", "Supplement schedule history"],
] as const;

export function DataTransferPage() {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [preview, setPreview] = useState<{ kind: "workspace" | "recipes"; valid: boolean; format: string; schemaVersion: number; recordCount: number; recordKinds: string[]; recipeCount: number; duplicateCount: number; conflictCount: number; omissions: string[]; errors: string[]; currentCounts?: Record<string, number>; incomingCounts?: Record<string, number>; declarations?: string[]; comparisonError?: string } | null>(null);
  const [pendingContent, setPendingContent] = useState("");
  const [conflictPolicy, setConflictPolicy] = useState("copy_incoming");
  const [workspace, setWorkspace] = useState<{ id: string; revision: bigint } | null>(null);
  const [recipes, setRecipes] = useState<{ id: string; name: string }[]>([]);
  const [recipeId, setRecipeId] = useState("");
  const [pageSize, setPageSize] = useState("A4");
  const [checkpoint, setCheckpoint] = useState<{ checkpointId: string; createdAt: string; restoreRevision: bigint; recipeCount: number; planIncluded: boolean; omissions: string[] } | null>(null);
  const [checkpointId, setCheckpointId] = useState("");
  const [recoveryConfirmed, setRecoveryConfirmed] = useState(false);
  const [recoveryKey, setRecoveryKey] = useState("");

  useEffect(() => {
    void ensureWorkspace().then(async (current) => {
      setWorkspace(current);
      const available = await listRecipes(current.id);
      setRecipes(available.map((recipe) => ({ id: recipe.id, name: recipe.name })));
    }).catch((error: unknown) => setMessage(messageFor(error, "Unable to load transfer options.")));
  }, []);

  function saveText(filename: string, content: string, type: string) {
    const url = URL.createObjectURL(new Blob([content], { type }));
    const anchor = document.createElement("a"); anchor.href = url; anchor.download = filename; anchor.click(); URL.revokeObjectURL(url);
  }

  function saveBytes(filename: string, content: Uint8Array, type: string) {
    const url = URL.createObjectURL(new Blob([content.buffer as ArrayBuffer], { type }));
    const anchor = document.createElement("a"); anchor.href = url; anchor.download = filename; anchor.click(); URL.revokeObjectURL(url);
  }

  async function exportRecipeCollection() {
    setBusy(true);
    try { const current = await ensureWorkspace(); const result = await exportRecipes(current.id); saveText(result.filename, result.content, "application/json"); setMessage(`Recipes ready. Omissions: ${result.omissions.length}.`); }
    catch (error) { setMessage(messageFor(error, "Unable to export recipes.")); }
    finally { setBusy(false); }
  }

  async function exportGroceries() {
    setBusy(true);
    try { const current = await ensureWorkspace(); const result = await exportGroceriesCSV({ workspaceId: current.id, expectedRevision: current.revision }); saveText(result.filename, result.content, "text/csv"); setMessage("Grocery CSV ready."); }
    catch (error) { setMessage(messageFor(error, "Unable to export groceries.")); }
    finally { setBusy(false); }
  }

  async function exportWeek() {
    setBusy(true);
    try { const current = await ensureWorkspace(); const result = await exportWeeklyPDF({ workspaceId: current.id, expectedRevision: current.revision, pageSize }); saveBytes(result.filename, result.content, "application/pdf"); setMessage("Weekly PDF ready."); }
    catch (error) { setMessage(messageFor(error, "Unable to export the weekly PDF.")); }
    finally { setBusy(false); }
  }

  async function exportRecipe() {
    if (!recipeId) { setMessage("Choose a recipe before exporting its PDF."); return; }
    setBusy(true);
    try { const current = await ensureWorkspace(); const result = await exportRecipePDF({ workspaceId: current.id, recipeId, pageSize }); saveBytes(result.filename, result.content, "application/pdf"); setMessage("Recipe PDF ready."); }
    catch (error) { setMessage(messageFor(error, "Unable to export the recipe PDF.")); }
    finally { setBusy(false); }
  }

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
      setMessage(messageFor(error, "Unable to export workspace."));
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
      let format = "";
      try { format = (JSON.parse(contentJson) as { format?: string }).format ?? ""; } catch { /* server returns the authoritative validation error */ }
      if (format === "daily.recipes") {
        const result = await previewRecipesImport({ workspaceId: workspace.id, contentJson });
        setPreview({ kind: "recipes", ...result, recordCount: result.recipeCount, recordKinds: [], omissions: [] });
      } else {
        const result = await previewWorkspaceImport({ workspaceId: workspace.id, contentJson });
        if (!result.valid) {
          setPreview({ kind: "workspace", ...result, recipeCount: 0, duplicateCount: 0, conflictCount: 0 });
          return;
        }
        const staged = JSON.parse(contentJson) as { manifest?: { recordKinds?: string[] }; records?: { kind?: string }[] };
        try {
          const currentExport = await exportWorkspace(workspace.id);
          const currentSnapshot = JSON.parse(currentExport.content) as { format?: string; manifest?: { recordKinds?: string[] }; records?: { kind?: string; id?: string; revision?: number }[] };
          const currentWorkspace = currentSnapshot.records?.find((record) => record.kind === "workspace");
          if (currentSnapshot.format !== "daily.workspace" || !currentSnapshot.manifest?.recordKinds || RESTORE_FAMILIES.some(([kind]) => !currentSnapshot.manifest?.recordKinds?.includes(kind)) || !currentSnapshot.records || currentWorkspace?.id !== workspace.id || typeof currentWorkspace.revision !== "number" || BigInt(currentWorkspace.revision) !== workspace.revision) throw new Error("The current workspace export does not match the active revision.");
          const kinds = ["workspace", "recipe", "recipe_revision", "plan", "shopping_state", "nutrition_target", "profile", "intake_event", "supplement_schedule", "supplement_schedule_revision"];
          const count = (records: { kind?: string }[] | undefined, kind: string) => records?.filter((record) => record.kind === kind).length ?? 0;
          const declarations = result.recordKinds;
          setPreview({ kind: "workspace", ...result, recipeCount: 0, duplicateCount: 0, conflictCount: 0, currentCounts: Object.fromEntries(kinds.map((kind) => [kind, count(currentSnapshot.records, kind)])), incomingCounts: Object.fromEntries(kinds.map((kind) => [kind, count(staged.records, kind)])), declarations });
        } catch (error) {
          setPreview({ kind: "workspace", ...result, recipeCount: 0, duplicateCount: 0, conflictCount: 0, comparisonError: `Current workspace could not be compared safely (${messageFor(error, "comparison unavailable")}). Retry by selecting the backup again.` });
        }
      }
    } catch (error) {
      setMessage(messageFor(error, "Unable to inspect workspace import."));
    } finally {
      setBusy(false);
    }
  }

  async function restore() {
    if (!workspace || !preview?.valid || preview.comparisonError || (preview.kind === "workspace" && !preview.currentCounts) || !pendingContent) return;
    setBusy(true);
    try {
      if (preview.kind === "recipes") {
        const result = await applyRecipesImport({ workspaceId: workspace.id, expectedWorkspaceRevision: workspace.revision, contentJson: pendingContent, idempotencyKey: crypto.randomUUID(), conflictPolicy });
        setMessage(`Recipe import applied: ${result.recipesApplied} added, ${result.recipesSkipped} skipped.`);
      } else {
        const result = await applyWorkspaceImport({ workspaceId: workspace.id, expectedWorkspaceRevision: workspace.revision, contentJson: pendingContent, idempotencyKey: crypto.randomUUID() });
        setCheckpointId(result.checkpointId);
        setWorkspace({ id: workspace.id, revision: result.workspaceRevision });
        try {
          const summary = await getRestoreCheckpoint({ workspaceId: workspace.id, checkpointId: result.checkpointId });
          setCheckpoint(summary);
          setRecoveryConfirmed(false);
          setRecoveryKey("");
          setMessage(`Workspace restore applied at revision ${result.workspaceRevision}. Its pre-restore checkpoint is ready for review.`);
        } catch (error) {
          setCheckpoint(null);
          setMessage(`Workspace restore applied at revision ${result.workspaceRevision}, but checkpoint details could not be loaded: ${messageFor(error, "inspection unavailable")}. The checkpoint ID is retained so you can retry review.`);
        }
      }
      setPreview(null);
    } catch (error) {
      setMessage(messageFor(error, "Unable to apply workspace restore."));
    } finally {
      setBusy(false);
    }
  }

  async function reviewCheckpoint() {
    if (!workspace || !checkpointId) return;
    setBusy(true);
    try { setCheckpoint(await getRestoreCheckpoint({ workspaceId: workspace.id, checkpointId })); setMessage("Saved pre-restore checkpoint details loaded for review."); }
    catch (error) { setMessage(`Checkpoint review failed: ${messageFor(error, "inspection unavailable")}. The checkpoint remains available to retry.`); }
    finally { setBusy(false); }
  }

  async function recoverCheckpoint() {
    if (!workspace || !checkpoint || !recoveryConfirmed) return;
    setBusy(true);
    const key = recoveryKey || crypto.randomUUID();
    setRecoveryKey(key);
    try {
      const current = await ensureWorkspace();
      setWorkspace({ id: current.id, revision: current.revision });
      const result = await recoverRestoreCheckpoint({ workspaceId: current.id, checkpointId: checkpoint.checkpointId, expectedWorkspaceRevision: current.revision, idempotencyKey: key });
      setWorkspace({ id: current.id, revision: result.workspaceRevision });
      setMessage(`Recovery complete at revision ${result.workspaceRevision}: saved supported workspace records restored, including ${result.recipesRestored} recipes and history${result.planRestored ? " and the saved plan" : "; no saved plan was present"}, shopping state, nutrition targets, profile, intake events, and supplement schedules. Unsupported domains remain omitted: ${result.omissions.join(", ")}.`);
      setCheckpoint(null);
      setCheckpointId("");
      setRecoveryConfirmed(false);
      setRecoveryKey("");
    } catch (error) {
      const text = messageFor(error, "Unable to recover the checkpoint.");
      if (/stale|revision/i.test(text)) {
        try { const fresh = await ensureWorkspace(); setWorkspace({ id: fresh.id, revision: fresh.revision }); } catch { /* keep the recovery review available */ }
        setMessage("The workspace changed since this checkpoint was reviewed. The latest revision has been refreshed; review and confirm recovery again.");
        setRecoveryConfirmed(false);
      } else {
        setMessage(`${text} The checkpoint remains available for review; retrying uses the same operation key.`);
      }
    } finally { setBusy(false); }
  }

  return <section data-testid={selectors.pages.transfer} aria-labelledby="transfer-heading" className="flex flex-col gap-6">
    <header><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">Data / Transfer</p><h1 id="transfer-heading" className="text-3xl font-semibold text-slate-900">Export and restore your data</h1><p className="mt-2 text-slate-600">Your own data exports do not require a provider or subscription. Inspect a backup before any live write; authority, credentials, and unsupported operational domains are never imported.</p></header>
    <div className="grid gap-4 md:grid-cols-2">
      <article className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Recipes JSON</h2><p className="mt-2 text-sm text-slate-600">Lossless native recipe collection with explicit omissions.</p><button type="button" className="mt-4 min-h-11 rounded border px-4 disabled:opacity-50" onClick={() => void exportRecipeCollection()} disabled={busy}>Export recipes</button></article>
      <article className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Workspace backup</h2><p className="mt-2 text-sm text-slate-600">Supported workspace records plus manifest and recovery context.</p><button type="button" className="mt-4 min-h-11 rounded bg-blue-700 px-4 font-medium text-white disabled:opacity-50" onClick={() => void downloadBackup()} disabled={busy}>{busy ? "Working…" : "Download workspace backup"}</button></article>
      <article className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Weekly PDF</h2><p className="mt-2 text-sm text-slate-600">Printable plan, unresolved slots, and shopping context.</p><label className="mt-4 block text-sm font-medium text-slate-700" htmlFor="pdf-page-size">Paper size</label><select id="pdf-page-size" aria-label="PDF paper size" className="mt-2 min-h-11 rounded border bg-white px-3" value={pageSize} onChange={(event) => setPageSize(event.target.value)} disabled={busy}><option value="A4">A4</option><option value="LETTER">US Letter</option></select><button type="button" className="mt-4 min-h-11 rounded border px-4 disabled:opacity-50" onClick={() => void exportWeek()} disabled={busy}>Export weekly PDF</button></article>
      <article className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Grocery CSV</h2><p className="mt-2 text-sm text-slate-600">Revision-pinned checklist with unknowns and formula-safe text.</p><button type="button" className="mt-4 min-h-11 rounded border px-4 disabled:opacity-50" onClick={() => void exportGroceries()} disabled={busy}>Export grocery CSV</button></article>
      <article className="rounded-lg border bg-white p-5 md:col-span-2"><h2 className="font-semibold">Recipe PDF</h2><p className="mt-2 text-sm text-slate-600">A legible recipe artifact with ingredients, map, yield, and method.</p><div className="mt-4 flex flex-wrap gap-3"><select aria-label="Recipe for PDF" className="min-h-11 rounded border bg-white px-3" value={recipeId} onChange={(event) => setRecipeId(event.target.value)} disabled={busy}><option value="">Choose a recipe</option>{recipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.name}</option>)}</select><button type="button" className="min-h-11 rounded border px-4 disabled:opacity-50" onClick={() => void exportRecipe()} disabled={busy}>Export recipe PDF</button></div></article>
    </div>
    {checkpointId && <section className="rounded-lg border border-amber-300 bg-amber-50 p-5" aria-labelledby="restore-checkpoint-heading">
      <h2 id="restore-checkpoint-heading" className="font-semibold">Pre-restore checkpoint ready for recovery</h2>
      {!checkpoint && <><p className="mt-2 text-sm text-slate-700">Checkpoint {checkpointId} exists, but its summary has not been loaded yet.</p><button type="button" className="mt-3 min-h-11 rounded border px-4 disabled:opacity-50" onClick={() => void reviewCheckpoint()} disabled={busy}>Review saved checkpoint</button></>}
      {checkpoint && <>
      <p className="mt-2 text-sm text-slate-700">Saved {checkpoint.createdAt} at workspace revision {String(checkpoint.restoreRevision)}. It contains {checkpoint.recipeCount} recipes and {checkpoint.planIncluded ? "the saved plan" : "no saved plan"}.</p>
      <p className="mt-2 text-sm text-slate-700">Recovery restores the checkpoint’s supported workspace records, including recipes and history, plan, shopping state, nutrition targets, profile, intake events, and supplement schedules. Unsupported domains remain omitted: {checkpoint.omissions.join(", ")}.</p>
      <label className="mt-3 flex min-h-11 items-center gap-2 text-sm font-medium"><input type="checkbox" checked={recoveryConfirmed} onChange={(event) => setRecoveryConfirmed(event.target.checked)} disabled={busy} />I reviewed this checkpoint and explicitly want to restore it over the current workspace.</label>
      <button type="button" className="mt-3 min-h-11 rounded bg-amber-800 px-4 font-medium text-white disabled:opacity-50" onClick={() => void recoverCheckpoint()} disabled={busy || !recoveryConfirmed}>Recover this checkpoint</button>
      </>}
    </section>}
    <div className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Inspect import before applying</h2><label className="mt-4 block text-sm font-medium text-slate-700" htmlFor="workspace-import">Choose a native daily JSON file</label><input id="workspace-import" className="mt-2 block min-h-11" type="file" accept="application/json,.json" onChange={(event) => void inspectFile(event.target.files?.[0])} disabled={busy} />{preview && <div className="mt-4 rounded border p-4" role={preview.valid ? "status" : "alert"}><p className="font-medium">{preview.valid ? "Import is valid and staged for review" : "Import cannot be applied"}</p>{preview.valid && <><p className="mt-2 text-sm text-slate-600">{preview.format} v{preview.schemaVersion}{preview.kind === "recipes" ? ` · ${preview.recipeCount} recipes · ${preview.duplicateCount} duplicates · ${preview.conflictCount} conflicts` : ` · ${preview.recordCount} records`}</p>{preview.kind === "workspace" && preview.currentCounts && preview.incomingCounts && <><h3 className="mt-3 font-medium">Workspace restore impact</h3><div className="mt-2 grid grid-cols-[1fr_auto_auto_1fr] gap-x-3 text-sm"><span>Record family</span><span>Current</span><span>Incoming</span><span>Action</span>{RESTORE_FAMILIES.map(([kind, label]) => { const declared = preview.declarations?.includes(kind); const action = declared ? preview.incomingCounts?.[kind] === 0 ? "Clear" : "Replace" : "Preserve"; return <div key={kind} className="contents"><span>{label}</span><span>{preview.currentCounts?.[kind] ?? 0}</span><span>{declared ? preview.incomingCounts?.[kind] ?? 0 : "—"}</span><span>{action}</span></div>; })}</div><p className="mt-2 text-sm text-slate-600">An omitted family in a legacy backup is preserved. Unsupported domains are not imported: {preview.omissions.join(", ") || "none declared"}.</p></>}{preview.kind === "recipes" && <label className="mt-3 block text-sm font-medium text-slate-700" htmlFor="recipe-conflict-policy">Conflict policy<select id="recipe-conflict-policy" className="mt-2 block min-h-11 rounded border bg-white px-3" value={conflictPolicy} onChange={(event) => setConflictPolicy(event.target.value)}><option value="copy_incoming">Copy incoming</option><option value="keep_existing">Keep existing</option><option value="replace">Replace existing revision</option></select></label>}<button type="button" className="mt-4 min-h-11 rounded bg-amber-700 px-4 font-medium text-white disabled:opacity-50" onClick={() => void restore()} disabled={busy || Boolean(preview.comparisonError) || (preview.kind === "workspace" && !preview.currentCounts)}>{preview.kind === "recipes" ? "Apply staged recipe import" : "Restore this validated backup"}</button></>}{preview.comparisonError && <p role="alert" className="mt-2 text-sm text-red-700">{preview.comparisonError}</p>}{preview.omissions.length > 0 && <p className="mt-2 text-sm text-slate-600">Unsupported domains omitted and preserved: {preview.omissions.join(", ")}.</p>}{preview.errors.length > 0 && <ul className="mt-2 list-disc pl-5 text-sm text-red-700">{preview.errors.map((error) => <li key={error}>{error}</li>)}</ul>}</div>}{message && <p className="mt-4 text-sm text-slate-600" role="status">{message}</p>}</div></section>;
}
