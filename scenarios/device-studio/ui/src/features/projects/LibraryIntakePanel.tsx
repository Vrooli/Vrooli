import { useEffect, useRef, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import type { IntakeCandidateRevision } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { correctLocalGeometryMaterial, downloadLocalGeometryCandidate, getLocalGeometryMaterials, importLocalGeometryCandidate, listLocalGeometryCandidates, MAX_LOCAL_GEOMETRY_BYTES, type MaterialCorrectionRequest } from "../../api/library-intake";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

function newCandidateID() {
  const suffix = globalThis.crypto?.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
  return `intake-${suffix}`;
}

function parseMaterialTuple(raw: string, count: number): number[] | undefined {
  const fields = raw.split(",").map((field) => field.trim());
  if (fields.length !== count || fields.some((field) => field === "")) return undefined;
  return fields.map(Number);
}

function MaterialEditor({ candidate, onComplete }: { candidate: IntakeCandidateRevision; onComplete: () => void }) {
  const { t } = useTranslation();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [roughness, setRoughness] = useState("");
  const [metallic, setMetallic] = useState("");
  const [baseColor, setBaseColor] = useState("");
  const [emissive, setEmissive] = useState("");
  const [childID, setChildID] = useState(newCandidateID);
  const [pending, setPending] = useState<MaterialCorrectionRequest>();
  const [error, setError] = useState("");
  const inventory = useQuery({ queryKey: ["local-geometry-materials", candidate.id, candidate.revision, candidate.revisionSha256], queryFn: () => getLocalGeometryMaterials(candidate), retry: false });
  const storageKey = `device-studio-material-correction:${candidate.id}@${candidate.revision}:${candidate.revisionSha256}`;
  useEffect(() => {
    try {
      const saved = localStorage.getItem(storageKey);
      if (saved) {
        const value = JSON.parse(saved) as MaterialCorrectionRequest;
        if (value.parent_candidate_id === candidate.id && value.parent_revision === candidate.revision && value.parent_revision_sha256 === candidate.revisionSha256) {
          setPending(value); setChildID(value.child_candidate_id); setSelectedIndex(value.material_index);
        }
      }
    } catch { setError(t(strings.physical.intakeMaterialStorageFailure)); }
  }, [candidate.id, candidate.revision, candidate.revisionSha256, storageKey, t]);
  const selected = inventory.data?.materials.find((material) => material.index === selectedIndex);
  const correction = useMutation({ mutationFn: correctLocalGeometryMaterial, onSuccess: () => {
    try { localStorage.removeItem(storageKey); } catch { /* completed server result remains visible */ }
    setPending(undefined); setError(""); setChildID(newCandidateID()); onComplete();
  }, onError: (cause) => setError(cause instanceof Error ? cause.message : t(strings.physical.intakeFailure)) });
  const apply = () => {
    if (pending) { correction.mutate(pending); return; }
    if (!selected || !selected.editable) { setError(t(strings.physical.intakeMaterialUnsupported)); return; }
    const request: MaterialCorrectionRequest = { parent_candidate_id: candidate.id, parent_revision: candidate.revision, parent_revision_sha256: candidate.revisionSha256, child_candidate_id: childID, material_index: selected.index, material_name: selected.name };
    if (roughness.trim()) request.roughness_factor = Number(roughness);
    if (metallic.trim()) request.metallic_factor = Number(metallic);
    if (baseColor.trim()) {
      const values = parseMaterialTuple(baseColor, 4);
      if (!values) { setError(t(strings.physical.intakeMaterialInvalid)); return; }
      request.base_color_factor = values as [number, number, number, number];
    }
    if (emissive.trim()) {
      const values = parseMaterialTuple(emissive, 3);
      if (!values) { setError(t(strings.physical.intakeMaterialInvalid)); return; }
      request.emissive_factor = values as [number, number, number];
    }
    if (![request.roughness_factor, request.metallic_factor, ...(request.base_color_factor ?? []), ...(request.emissive_factor ?? [])].every((value) => value === undefined || (Number.isFinite(value) && value >= 0 && value <= 1))) { setError(t(strings.physical.intakeMaterialInvalid)); return; }
    if (![request.roughness_factor, request.metallic_factor, request.base_color_factor, request.emissive_factor].some((value) => value !== undefined)) { setError(t(strings.physical.intakeMaterialNoChange)); return; }
    try { localStorage.setItem(storageKey, JSON.stringify(request)); }
    catch { setError(t(strings.physical.intakeMaterialStorageFailure)); return; }
    setPending(request); correction.mutate(request);
  };
  if (inventory.isLoading) return <p>{t(strings.physical.intakeMaterialLoading)}</p>;
  if (inventory.error) return <p role="alert">{t(strings.physical.intakeMaterialUnavailable)}</p>;
  return <div className="physical-shot__material-editor" data-testid={`intake-material-editor-${candidate.id}`}>
    <label>{t(strings.physical.intakeMaterialSelect)}<select className="physical-shot__input" value={selectedIndex} disabled={Boolean(pending)} onChange={(event) => { const material = inventory.data?.materials.find((item) => item.index === Number(event.target.value)); if (material) { setSelectedIndex(material.index); setRoughness(""); setMetallic(""); setBaseColor(""); setEmissive(""); } }}>
      {inventory.data?.materials.map((material) => <option key={material.index} value={material.index}>[{material.index}] {material.name || `Material ${material.index}`} {material.editable ? "" : "· unsupported"}</option>)}
    </select></label>
    {selected && <p>{t(strings.physical.intakeMaterialCurrent, { roughness: selected.roughness_factor, metallic: selected.metallic_factor })}</p>}
    <label>{t(strings.physical.intakeMaterialRoughness)}<input className="physical-shot__input" type="number" min="0" max="1" step="0.01" disabled={Boolean(pending)} value={roughness} onChange={(event) => setRoughness(event.target.value)} /></label>
    <label>{t(strings.physical.intakeMaterialMetallic)}<input className="physical-shot__input" type="number" min="0" max="1" step="0.01" disabled={Boolean(pending)} value={metallic} onChange={(event) => setMetallic(event.target.value)} /></label>
    <label>{t(strings.physical.intakeMaterialBaseColor)}<input className="physical-shot__input" disabled={Boolean(pending)} placeholder="R, G, B, A" value={baseColor} onChange={(event) => setBaseColor(event.target.value)} /></label>
    <label>{t(strings.physical.intakeMaterialEmissive)}<input className="physical-shot__input" disabled={Boolean(pending)} placeholder="R, G, B" value={emissive} onChange={(event) => setEmissive(event.target.value)} /></label>
    {pending && <p>{t(strings.physical.intakeMaterialPending)}</p>}
    {error && <p role="alert">{error}</p>}
    {correction.data?.material_correction_lineage && <details><summary>{t(strings.physical.intakeMaterialLineage)}</summary><p>{correction.data.parent_candidate_id}@{correction.data.parent_revision} · parent SHA-256 {correction.data.parent_sha256}</p><p>{t(strings.physical.intakeMaterialChildHash)} {correction.data.original_sha256}</p><pre>{JSON.stringify(correction.data.material_correction_lineage, null, 2)}</pre></details>}
    <Button type="button" className="min-h-11" disabled={correction.isPending || (!pending && !selected?.editable)} onClick={apply}>{pending ? t(strings.physical.intakeMaterialRetry) : correction.isPending ? t(strings.physical.intakeMaterialApplying) : t(strings.physical.intakeMaterialApply)}</Button>
    {pending && <Button type="button" variant="outline" className="min-h-11" disabled={correction.isPending} onClick={() => { try { localStorage.removeItem(storageKey); setPending(undefined); setError(""); setChildID(newCandidateID()); } catch { setError(t(strings.physical.intakeMaterialStorageFailure)); } }}>{t(strings.physical.intakeMaterialDiscard)}</Button>}
  </div>;
}

export function LibraryIntakePanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const fileInput = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File>();
  const [candidateId, setCandidateId] = useState(newCandidateID);
  const [displayName, setDisplayName] = useState("");
  const [sourceLabel, setSourceLabel] = useState("");
  const [licenseClaim, setLicenseClaim] = useState("");
  const [attribution, setAttribution] = useState("");
  const [downloadError, setDownloadError] = useState("");
  const [materialCandidate, setMaterialCandidate] = useState<string>();
  const candidates = useInfiniteQuery({
    queryKey: ["local-geometry-intake"],
    queryFn: ({ pageParam }) => listLocalGeometryCandidates(pageParam),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.nextAfterCandidateId || undefined,
    retry: false,
  });
  const upload = useMutation({
    mutationFn: () => {
      if (!file) throw new Error("Choose a GLB file first.");
      return importLocalGeometryCandidate(file, { candidateId, displayName, sourceLabel, licenseClaim, attribution });
    },
    onSuccess: async () => {
      setFile(undefined);
      if (fileInput.current) fileInput.current.value = "";
      setCandidateId(newCandidateID());
      setDisplayName("");
      setSourceLabel("");
      setLicenseClaim("");
      setAttribution("");
      await queryClient.invalidateQueries({ queryKey: ["local-geometry-intake"] });
    },
  });
  const importReady = Boolean(file && file.size > 0 && file.size <= MAX_LOCAL_GEOMETRY_BYTES && displayName.trim() && sourceLabel.trim() && candidateId.trim());
  const download = async (candidate: IntakeCandidateRevision) => {
    setDownloadError("");
    try {
      const blob = await downloadLocalGeometryCandidate(candidate);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = candidate.originalFilename;
      anchor.click();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) {
      setDownloadError(error instanceof Error ? error.message : t(strings.physical.intakeFailure));
    }
  };

  return <Card className="physical-shot__intake" data-testid="physical-library-intake">
    <CardHeader><CardTitle as="h3">{t(strings.physical.intakeTitle)}</CardTitle><p>{t(strings.physical.intakeHelp)}</p></CardHeader>
    <CardContent className="physical-shot__fields">
      <label>{t(strings.physical.intakeFile)}<input ref={fileInput} className="physical-shot__input" type="file" accept=".glb,model/gltf-binary" onChange={(event) => setFile(event.currentTarget.files?.[0])} /></label>
      {file && <p>{file.name} · {file.size.toLocaleString()} bytes</p>}
      <label>{t(strings.physical.intakeDisplayName)}<input className="physical-shot__input" value={displayName} maxLength={120} onChange={(event) => setDisplayName(event.target.value)} /></label>
      <label>{t(strings.physical.intakeSource)}<input className="physical-shot__input" value={sourceLabel} maxLength={200} onChange={(event) => setSourceLabel(event.target.value)} /></label>
      <label>{t(strings.physical.intakeLicense)}<input className="physical-shot__input" value={licenseClaim} maxLength={200} onChange={(event) => setLicenseClaim(event.target.value)} /></label>
      <label>{t(strings.physical.intakeAttribution)}<input className="physical-shot__input" value={attribution} maxLength={200} onChange={(event) => setAttribution(event.target.value)} /></label>
      <p className="physical-shot__candidate">{t(strings.physical.intakeUnreviewed)}</p>
      <Button type="button" className="studio-primary min-h-11" disabled={!importReady || upload.isPending} onClick={() => upload.mutate()}>{upload.isPending ? t(strings.physical.intakeUploading) : t(strings.physical.intakeImport)}</Button>
      <details>
        <summary>{t(strings.physical.intakeTechnicalDetails)}</summary>
        <label>{t(strings.physical.intakeCandidateId)}<input className="physical-shot__input" value={candidateId} maxLength={128} onChange={(event) => setCandidateId(event.target.value)} /></label>
      </details>
      {upload.error && <p role="alert">{upload.error.message}</p>}
      {candidates.error && <p role="alert">{t(strings.physical.intakeListUnavailable)}</p>}
      {downloadError && <p role="alert">{downloadError}</p>}
      {candidates.data && <div><h4>{t(strings.physical.intakeCandidates)}</h4>{candidates.data.pages.flatMap((page) => page.candidates).length === 0 ? <p>{t(strings.physical.intakeEmpty)}</p> : <ul>{candidates.data.pages.flatMap((page) => page.candidates).map((candidate) => <li key={`${candidate.id}@${candidate.revision}`}>
        <strong>{candidate.displayName}</strong> · {candidate.originalFilename} · {candidate.sizeBytes.toString()} bytes<br />
        {t(strings.physical.intakeStatus, { review: candidate.reviewStatus, render: candidate.renderStatus })} · SHA-256 {candidate.originalSha256.slice(0, 12)}…
        {candidate.inspection && <span> · {candidate.inspection.supported ? t(strings.physical.intakeSupported) : t(strings.physical.intakeUnsupported)}{candidate.inspection.finding ? ` · ${candidate.inspection.finding}` : ""}</span>}
        <Button type="button" variant="outline" className="min-h-11" onClick={() => void download(candidate)}>{t(strings.physical.intakeDownload)}</Button>
        <Button type="button" variant="outline" className="min-h-11" onClick={() => setMaterialCandidate((current) => current === candidate.id ? undefined : candidate.id)}>{materialCandidate === candidate.id ? t(strings.physical.intakeMaterialClose) : t(strings.physical.intakeMaterialEdit)}</Button>
        {materialCandidate === candidate.id && <MaterialEditor candidate={candidate} onComplete={() => { void queryClient.invalidateQueries({ queryKey: ["local-geometry-intake"] }); }} />}
      </li>)}</ul>}{candidates.hasNextPage && <Button type="button" variant="outline" className="min-h-11" disabled={candidates.isFetchingNextPage} onClick={() => void candidates.fetchNextPage()}>{t(strings.physical.intakeLoadMore)}</Button>}</div>}
    </CardContent>
  </Card>;
}
