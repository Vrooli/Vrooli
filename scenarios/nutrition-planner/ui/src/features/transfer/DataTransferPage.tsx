import { useEffect, useState } from "react";
import { ensureWorkspace } from "../../api/workspace";
import { applyRecipesImport, applyWorkspaceImport, exportGroceriesCSV, exportRecipePDF, exportRecipes, exportWeeklyPDF, exportWorkspace, previewRecipesImport, previewWorkspaceImport } from "../../api/portability";
import { listRecipes } from "../../api/recipes";
import { selectors } from "../../consts/selectors";

function messageFor(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

export function DataTransferPage() {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [preview, setPreview] = useState<{ kind: "workspace" | "recipes"; valid: boolean; format: string; schemaVersion: number; recordCount: number; recordKinds: string[]; recipeCount: number; duplicateCount: number; conflictCount: number; omissions: string[]; errors: string[] } | null>(null);
  const [pendingContent, setPendingContent] = useState("");
  const [conflictPolicy, setConflictPolicy] = useState("copy_incoming");
  const [workspace, setWorkspace] = useState<{ id: string; revision: bigint } | null>(null);
  const [recipes, setRecipes] = useState<{ id: string; name: string }[]>([]);
  const [recipeId, setRecipeId] = useState("");
  const [pageSize, setPageSize] = useState("A4");

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
        setPreview({ kind: "workspace", ...result, recipeCount: 0, duplicateCount: 0, conflictCount: 0 });
      }
    } catch (error) {
      setMessage(messageFor(error, "Unable to inspect workspace import."));
    } finally {
      setBusy(false);
    }
  }

  async function restore() {
    if (!workspace || !preview?.valid || !pendingContent) return;
    setBusy(true);
    try {
      if (preview.kind === "recipes") {
        const result = await applyRecipesImport({ workspaceId: workspace.id, expectedWorkspaceRevision: workspace.revision, contentJson: pendingContent, idempotencyKey: crypto.randomUUID(), conflictPolicy });
        setMessage(`Recipe import applied: ${result.recipesApplied} added, ${result.recipesSkipped} skipped.`);
      } else {
        const result = await applyWorkspaceImport({ workspaceId: workspace.id, expectedWorkspaceRevision: workspace.revision, contentJson: pendingContent, idempotencyKey: crypto.randomUUID() });
        setMessage(`Restore applied: ${result.recipesApplied} recipes. Checkpoint ${result.checkpointId} is available for recovery.`);
      }
      setPreview(null);
    } catch (error) {
      setMessage(messageFor(error, "Unable to apply workspace restore."));
    } finally {
      setBusy(false);
    }
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
    <div className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Inspect import before applying</h2><label className="mt-4 block text-sm font-medium text-slate-700" htmlFor="workspace-import">Choose a native daily JSON file</label><input id="workspace-import" className="mt-2 block min-h-11" type="file" accept="application/json,.json" onChange={(event) => void inspectFile(event.target.files?.[0])} disabled={busy} />{preview && <div className="mt-4 rounded border p-4" role={preview.valid ? "status" : "alert"}><p className="font-medium">{preview.valid ? "Import is valid and staged for review" : "Import cannot be applied"}</p>{preview.valid && <><p className="mt-2 text-sm text-slate-600">{preview.format} v{preview.schemaVersion}{preview.kind === "recipes" ? ` · ${preview.recipeCount} recipes · ${preview.duplicateCount} duplicates · ${preview.conflictCount} conflicts` : ` · ${preview.recordCount} records · kinds: ${preview.recordKinds.join(", ")}`}</p>{preview.kind === "recipes" && <label className="mt-3 block text-sm font-medium text-slate-700" htmlFor="recipe-conflict-policy">Conflict policy<select id="recipe-conflict-policy" className="mt-2 block min-h-11 rounded border bg-white px-3" value={conflictPolicy} onChange={(event) => setConflictPolicy(event.target.value)}><option value="copy_incoming">Copy incoming</option><option value="keep_existing">Keep existing</option><option value="replace">Replace existing revision</option></select></label>}<button type="button" className="mt-4 min-h-11 rounded bg-amber-700 px-4 font-medium text-white disabled:opacity-50" onClick={() => void restore()} disabled={busy}>{preview.kind === "recipes" ? "Apply staged recipe import" : "Restore this validated backup"}</button></>}{preview.omissions.length > 0 && <p className="mt-2 text-sm text-slate-600">Omissions: {preview.omissions.join(", ")}</p>}{preview.errors.length > 0 && <ul className="mt-2 list-disc pl-5 text-sm text-red-700">{preview.errors.map((error) => <li key={error}>{error}</li>)}</ul>}</div>}{message && <p className="mt-4 text-sm text-slate-600" role="status">{message}</p>}</div></section>;
}
