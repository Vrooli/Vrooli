import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { Button } from "@vrooli/react-component-library/Button/2";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { uploadSourceMedia, projectsClient } from "../../api/projects";
import { cancelPhotographicOperation, getPhotographicPNG, readPhotographicOperation, submitPhotographicScene, waitPhotographicOperation, type PhotographicOperation, type PhotographicRequest } from "../../api/photographic-scene";

type Saved = PhotographicRequest & { version: 1; media?: { id: string; sha256: string } };
const storageKey = (projectId: string) => `device-studio:photographic-scene:v1:${projectId}`;
function readSaved(projectId: string): Saved | undefined {
  try {
    const raw = localStorage.getItem(storageKey(projectId)); if (!raw) return undefined;
    const value = JSON.parse(raw) as Saved;
    if (value.version !== 1 || !value.operation_id || typeof value.prompt !== "string" || !((value.width === 768 && value.height === 512) || (value.width === 512 && value.height === 768))) return undefined;
    return value;
  } catch { return undefined; }
}

export function PhotographicScenePanel({ projectId }: { projectId: string }) {
  return <PhotographicScenePanelState key={projectId} projectId={projectId} />;
}

function PhotographicScenePanelState({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [saved, setSaved] = useState<Saved>();
  const [operation, setOperation] = useState<PhotographicOperation>();
  const [prompt, setPrompt] = useState("");
  const [width, setWidth] = useState<768 | 512>(768);
  const [height, setHeight] = useState<512 | 768>(512);
  const [storageReady, setStorageReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const requestGeneration = useRef(0);
  const [unknown, setUnknown] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    const restored = readSaved(projectId);
    if (restored) {
      setSaved(restored); setPrompt(restored.prompt); setWidth(restored.width); setHeight(restored.height);
      const importedMedia = restored.media;
      if (importedMedia) void projectsClient.getSourceMedia({ projectId, id: importedMedia.id }).then((response) => {
        if (response.sourceMedia?.sha256 === importedMedia.sha256) setMessage(t(strings.photographic_scene.saved, { mediaId: importedMedia.id, sha256: importedMedia.sha256 }));
      }).catch(() => undefined);
      void readPhotographicOperation(projectId, restored.operation_id).then((status) => { setOperation(status); setUnknown(false); }).catch(() => setUnknown(true));
    }
    setStorageReady(true);
  // Restore exactly once per project. No effect here submits or resumes a job.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  function persist(value: Saved): boolean {
    try { localStorage.setItem(storageKey(projectId), JSON.stringify(value)); setSaved(value); return true; }
    catch { setMessage(t(strings.photographic_scene.storageUnavailable)); return false; }
  }
  async function submit() {
    const request: PhotographicRequest = saved ? saved : { operation_id: crypto.randomUUID(), prompt: prompt.trim(), width, height };
    if (!saved && (!request.prompt || new TextEncoder().encode(request.prompt).length > 800)) return;
    const record: Saved = { ...request, version: 1 };
    if (!persist(record)) return;
    setBusy(true); setUnknown(false); setMessage("");
    try {
      const status = await submitPhotographicScene(projectId, request);
      setOperation(status); setMessage("");
    } catch (error) {
      if (error instanceof ApiError && error.status === 422) {
        try { localStorage.removeItem(storageKey(projectId)); } catch { /* the request was still definitively rejected */ }
        setSaved(undefined); setOperation(undefined); setUnknown(false);
        setMessage(t(strings.photographic_scene.rejected));
      } else {
        setUnknown(true); setMessage(t(strings.photographic_scene.operationUnknown));
      }
    } finally { setBusy(false); }
  }
  async function refresh() {
    if (!saved || waiting) return; setBusy(true);
    const generation = ++requestGeneration.current;
    try { const status = await readPhotographicOperation(projectId, saved.operation_id); if (generation === requestGeneration.current) { setOperation(status); setUnknown(false); } }
    catch { if (generation === requestGeneration.current) { setUnknown(true); setMessage(t(strings.photographic_scene.operationUnknown)); } }
    finally { setBusy(false); }
  }
  async function waitOnce() {
    if (!saved || busy || waiting) return; setWaiting(true);
    const generation = ++requestGeneration.current;
    try { const status = await waitPhotographicOperation(projectId, saved.operation_id); if (generation === requestGeneration.current) { setOperation(status); setUnknown(false); } }
    catch { if (generation === requestGeneration.current) { setUnknown(true); setMessage(t(strings.photographic_scene.operationUnknown)); } }
    finally { setWaiting(false); }
  }
  async function cancel() {
    if (!saved || busy) return; setBusy(true);
    const generation = ++requestGeneration.current;
    try { const status = await cancelPhotographicOperation(projectId, saved.operation_id); if (generation === requestGeneration.current) { setOperation(status); setUnknown(false); } }
    catch { if (generation === requestGeneration.current) { setUnknown(true); setMessage(t(strings.photographic_scene.operationUnknown)); } }
    finally { setBusy(false); }
  }
  async function importResult() {
    if (!saved || !operation?.job_id || operation.state !== "succeeded") return;
    setBusy(true); setMessage(t(strings.photographic_scene.saving));
    try {
      const blob = await getPhotographicPNG(projectId, saved.operation_id, saved.width, saved.height);
      const file = new File([blob], `device-studio-generated-${operation.job_id}.png`, { type: "image/png" });
      const media = await uploadSourceMedia(projectId, file);
      await queryClient.invalidateQueries({ queryKey: ["project-source-media", projectId] });
      const next = { ...saved, media: { id: media.id, sha256: media.sha256 } };
      persist(next);
      setMessage(t(strings.photographic_scene.saved, { mediaId: media.id, sha256: media.sha256 }));
    } catch { setMessage(t(strings.photographic_scene.importFailed)); }
    finally { setBusy(false); }
  }
  function startDifferent() {
    if (busy || waiting || unknown || !operation || ["queued", "running"].includes(operation.state)) return;
    try { localStorage.removeItem(storageKey(projectId)); } catch { return; }
    setSaved(undefined); setOperation(undefined); setPrompt(""); setWidth(768); setHeight(512); setUnknown(false); setMessage("");
  }

  const locked = Boolean(saved);
  const promptBytes = new TextEncoder().encode(prompt).length;
  const canImport = operation?.state === "succeeded" && !saved?.media;
  return <section className="rounded-panel border border-app-border bg-app-surface p-space-md" aria-labelledby="photo-scene-title">
    <h2 id="photo-scene-title" className="text-lg font-semibold">{t(strings.photographic_scene.title)}</h2>
    <p className="mt-1 text-sm text-app-muted">{t(strings.photographic_scene.description)}</p>
    <p className="mt-2 rounded-control bg-app-muted/20 p-space-sm text-sm">{t(strings.photographic_scene.disclosure)}</p>
    {!locked && <div className="mt-space-sm flex flex-col gap-space-sm">
      {message && <p role="status">{message}</p>}
      <label className="flex flex-col gap-1 text-sm font-medium">{t(strings.photographic_scene.prompt)}
        <textarea className="min-h-24 rounded-control border border-app-border bg-app-surface px-3 py-2 text-base" value={prompt} onChange={(event) => setPrompt(event.target.value)} maxLength={800} disabled={!storageReady || busy} />
      </label>
      <p className="text-xs text-app-muted">{t(strings.photographic_scene.promptBytes, { bytes: promptBytes })}</p>
      <label className="flex flex-col gap-1 text-sm font-medium">{t(strings.photographic_scene.dimensions)}
        <select className="min-h-11 rounded-control border border-app-border bg-app-surface px-3" value={`${width}x${height}`} onChange={(event) => { const landscape = event.target.value === "768x512"; setWidth(landscape ? 768 : 512); setHeight(landscape ? 512 : 768); }} disabled={!storageReady || busy}>
          <option value="768x512">{t(strings.photographic_scene.landscape)}</option><option value="512x768">{t(strings.photographic_scene.portrait)}</option>
        </select>
      </label>
      <Button className="min-h-11 self-start" onClick={() => void submit()} disabled={!storageReady || busy || !prompt.trim() || promptBytes > 800}>{t(strings.photographic_scene.generate)}</Button>
    </div>}
    {saved && <div className="mt-space-sm flex flex-wrap items-center gap-space-sm">
      {operation && <p role="status">{t(strings.photographic_scene.accepted, { jobId: operation.job_id, state: operation.state, progress: operation.progress })}</p>}
      {operation?.model_id && <p>{t(strings.photographic_scene.model, { model: operation.model_id })}</p>}
      {operation?.tier && <p>{t(strings.photographic_scene.tier, { tier: operation.tier })}</p>}
      {unknown && <p role="alert">{message || t(strings.photographic_scene.operationUnknown)}</p>}
        <Button variant="outline" className="min-h-11" onClick={() => void refresh()} disabled={busy || waiting}>{t(strings.photographic_scene.refresh)}</Button>
      {operation && ["queued", "running"].includes(operation.state) && <>
        <Button variant="outline" className="min-h-11" onClick={() => void waitOnce()} disabled={busy || waiting}>{t(strings.photographic_scene.wait)}</Button>
        <Button variant="outline" className="min-h-11" onClick={() => void cancel()} disabled={busy}>{t(strings.photographic_scene.cancel)}</Button>
      </>}
      {unknown && <Button variant="outline" className="min-h-11" onClick={() => void submit()} disabled={busy}>{t(strings.photographic_scene.retry)}</Button>}
      {operation?.state === "failed" && <p role="alert">{t(strings.photographic_scene.failed)}</p>}
      {canImport && <Button className="min-h-11" onClick={() => void importResult()} disabled={busy}>{t(strings.photographic_scene.save)}</Button>}
      {saved.media && <p role="status">{message || t(strings.photographic_scene.saved, { mediaId: saved.media.id, sha256: saved.media.sha256 })}</p>}
      {message && !unknown && !saved.media && <p role="status">{message}</p>}
      <Button variant="outline" className="min-h-11" onClick={startDifferent} disabled={busy || waiting || unknown || !operation || ["queued", "running"].includes(operation.state)}>{t(strings.photographic_scene.newImage)}</Button>
    </div>}
  </section>;
}
