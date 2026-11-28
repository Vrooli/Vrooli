import { useEffect, useId, useRef, useState } from "react";
import { Check, Download, FileUp, TriangleAlert, X } from "lucide-react";

import { useTranslation } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { WbButton } from "../primitives";
import { useFocusTrap } from "../useFocusTrap";
import type { DesignDocument } from "../design-types";
import {
  applyCaptureOutlinePatch,
  validateCaptureOutlinePatch,
  type CaptureOutlinePatch,
} from "../captureOutlinePatch";
import type { AppliedPatch } from "./patch";
import { readAsText, sha256 } from "./capturePatchEvidence";
import "./capture-outline-patch.css";

const PATCH_LIMIT = 2 * 1024 * 1024;
const IMAGE_LIMIT = 15 * 1024 * 1024;

interface EvidenceImage {
  role: "source" | "preview";
  path: string;
  file: File;
  url: string;
}

function basename(path: string): string {
  return path.split("/").at(-1) ?? path;
}

async function readPatch(file: File): Promise<unknown> {
  if (file.size > PATCH_LIMIT) throw new Error("capturePatch.patchTooLarge");
  const text = await readAsText(file);
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    throw new Error("capturePatch.invalidJson");
  }
  validateCaptureOutlinePatch(parsed);
  return parsed;
}

export interface CaptureOutlinePatchDialogProps {
  design: DesignDocument;
  activeFrameId: string;
  readOnly: boolean;
  onClose: () => void;
  onApply: (patch: AppliedPatch, patchId: string, summary: string) => void;
}

/** Local import, evidence check and apply surface for a capture-to-edit patch. */
export function CaptureOutlinePatchDialog({
  design,
  activeFrameId,
  readOnly,
  onClose,
  onApply,
}: CaptureOutlinePatchDialogProps) {
  const { t } = useTranslation();
  const patchInputId = useId();
  const evidenceInputId = useId();
  const evidenceUrls = useRef(new Set<string>());
  const trapRef = useFocusTrap<HTMLDivElement>(true, onClose);
  const [patch, setPatch] = useState<CaptureOutlinePatch | null>(null);
  const [images, setImages] = useState<EvidenceImage[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [working, setWorking] = useState(false);

  const sourceImage = images.find((image) => image.role === "source") ?? null;
  const previewImage = images.find((image) => image.role === "preview") ?? null;
  const evidenceReady = Boolean(sourceImage && previewImage);
  const evidenceHint = sourceImage && previewImage ? null : t(strings.workbench.studio.patchChooseEvidenceHint);

  const compositionMatches = Boolean(
    patch &&
    patch.composition.design_id === design.id &&
    patch.composition.base_revision === design.revision &&
    patch.target.design_frame_id === activeFrameId,
  );

  const resetEvidence = () => {
    for (const url of evidenceUrls.current) URL.revokeObjectURL(url);
    evidenceUrls.current.clear();
    setImages([]);
  };

  useEffect(() => () => {
    for (const url of evidenceUrls.current) URL.revokeObjectURL(url);
    evidenceUrls.current.clear();
  }, []);

  const choosePatch = async (file?: File) => {
    if (!file) return;
    setWorking(true);
    setError(null);
    resetEvidence();
    try {
      const value = await readPatch(file);
      setPatch(value as CaptureOutlinePatch);
    } catch (cause) {
      setPatch(null);
      const key = cause instanceof Error ? errorStringKey(cause.message) : null;
      setError(t(key ?? strings.workbench.studio.patchReadFailed));
    } finally {
      setWorking(false);
    }
  };

  const chooseEvidence = async (files: File[]) => {
    if (!patch || files.length === 0) return;
    setWorking(true);
    setError(null);
    resetEvidence();
    const accepted: EvidenceImage[] = [];
    try {
      for (const file of files) {
        if (file.size > IMAGE_LIMIT) throw new Error(t(strings.workbench.studio.patchImageTooLarge, { name: file.name }));
        if (!/^image\/(png|jpeg)$/.test(file.type)) throw new Error(t(strings.workbench.studio.patchImageType, { name: file.name }));
        const matched = [
          { role: "source" as const, evidence: patch.source.image },
          { role: "preview" as const, evidence: patch.preview.image },
        ].find(({ evidence }) => basename(evidence.path) === file.name);
        if (!matched) continue;
        if ((await sha256(file)) !== matched.evidence.sha256) {
          throw new Error(t(strings.workbench.studio.patchImageHashMismatch, { name: file.name }));
        }
        const url = URL.createObjectURL(file);
        evidenceUrls.current.add(url);
        accepted.push({ role: matched.role, path: matched.evidence.path, file, url });
      }
      if (accepted.length !== 2 || accepted[0]?.role === accepted[1]?.role) {
        for (const image of accepted) {
          URL.revokeObjectURL(image.url);
          evidenceUrls.current.delete(image.url);
        }
        throw new Error(t(strings.workbench.studio.patchBothImagesRequired));
      }
      setImages(accepted);
    } catch (cause) {
      for (const image of accepted) {
        URL.revokeObjectURL(image.url);
        evidenceUrls.current.delete(image.url);
      }
      setImages([]);
      const key = cause instanceof Error ? errorStringKey(cause.message) : null;
      setError(key ? t(key) : cause instanceof Error ? cause.message : t(strings.workbench.studio.patchEvidenceFailed));
    } finally {
      setWorking(false);
    }
  };

  const exportPatch = () => {
    if (!patch) return;
    const blob = new Blob([`${JSON.stringify(patch, null, 2)}\n`], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${patch.patch_id}.capture-outline-patch.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const apply = () => {
    if (!patch || !compositionMatches || readOnly || !evidenceReady) return;
    setError(null);
    try {
      const applied = applyCaptureOutlinePatch(design, patch);
      onApply(applied, patch.patch_id, patch.composition.summary);
    } catch {
      setError(t(strings.workbench.studio.patchApplyFailed));
    }
  };

  return (
    <div className="co-patch-layer">
      <button className="co-patch-backdrop" type="button" aria-label={t(strings.workbench.studio.closePatch)} onClick={onClose} />
      <section
        ref={trapRef}
        className="co-patch-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="co-patch-title"
        aria-busy={working}
        data-testid={selectors.workspaceStudio.capturePatchDialog}
      >
        <header className="co-patch-head">
          <div>
            <p className="co-patch-eyebrow">{t(strings.workbench.studio.patchEyebrow)}</p>
            <h2 id="co-patch-title">{t(strings.workbench.studio.patchTitle)}</h2>
            <p>{t(strings.workbench.studio.patchIntro)}</p>
          </div>
          <button type="button" className="studio-iconbtn" aria-label={t(strings.workbench.studio.closePatch)} onClick={onClose}><X aria-hidden size={16} /></button>
        </header>

        <div className="co-patch-content">
          <section className="co-patch-step" aria-labelledby="co-patch-file-heading">
            <div className="co-patch-step-head"><span>1</span><div><h3 id="co-patch-file-heading">{t(strings.workbench.studio.patchLoadHeading)}</h3><p>{t(strings.workbench.studio.patchLoadDescription)}</p></div></div>
            <input id={patchInputId} type="file" accept="application/json,.json" className="co-patch-file" aria-label={t(strings.workbench.studio.patchFileLabel)} data-testid={selectors.workspaceStudio.capturePatchFile} disabled={working} onChange={(event) => void choosePatch(event.currentTarget.files?.[0])} />
            <label className="wb-btn secondary sm co-patch-file-label" htmlFor={patchInputId}><FileUp aria-hidden size={14} />{patch ? t(strings.workbench.studio.patchReplace) : t(strings.workbench.studio.patchChooseJson)}</label>
            {patch ? <p className="co-patch-file-name" role="status">{patch.patch_id} · {patch.source.scenario} · {patch.source.route}</p> : <p className="studio-note">{t(strings.workbench.studio.patchLocalOnly)}</p>}
          </section>

          {patch ? (
            <>
              <section className="co-patch-step" aria-labelledby="co-patch-evidence-heading">
                <div className="co-patch-step-head"><span>2</span><div><h3 id="co-patch-evidence-heading">{t(strings.workbench.studio.patchEvidenceHeading)}</h3><p>{t(strings.workbench.studio.patchEvidenceDescription)}</p></div></div>
                <input id={evidenceInputId} type="file" accept="image/png,image/jpeg" multiple className="co-patch-file" aria-label={t(strings.workbench.studio.patchEvidenceLabel)} data-testid={selectors.workspaceStudio.capturePatchEvidence} disabled={working} onChange={(event) => void chooseEvidence(Array.from(event.currentTarget.files ?? []))} />
                <label className="wb-btn secondary sm co-patch-file-label" htmlFor={evidenceInputId}><FileUp aria-hidden size={14} />{evidenceReady ? t(strings.workbench.studio.patchReplaceImages) : t(strings.workbench.studio.patchChooseImages)}</label>
                <div className="co-patch-images">
                  {([
                    { label: t(strings.workbench.studio.patchCapturedBefore), image: sourceImage, path: patch.source.image.path, fidelity: t(strings.workbench.studio.patchCaptured) },
                    { label: t(strings.workbench.studio.patchComponentAfter), image: previewImage, path: patch.preview.image.path, fidelity: patch.preview.fidelity },
                  ] as const).map(({ label, image, path, fidelity }) => (
                    <figure key={label} className="co-patch-image">
                      <figcaption><span>{label}</span><small>{fidelity}</small></figcaption>
                      {image ? <img src={image.url} alt={t(strings.workbench.studio.patchEvidenceAlt, { label, region: patch.target.region_id })} /> : <div className="co-patch-image-empty"><span>{basename(path)}</span><small>{t(strings.workbench.studio.patchHashRequired)}</small></div>}
                      <small className="co-patch-evidence-path">{path}</small>
                    </figure>
                  ))}
                </div>
                {!evidenceReady ? <p className="studio-note">{evidenceHint}</p> : <p className="co-patch-verified"><Check aria-hidden size={14} />{t(strings.workbench.studio.patchHashesMatch)}</p>}
              </section>

              <section className="co-patch-step" aria-labelledby="co-patch-target-heading">
                <div className="co-patch-step-head"><span>3</span><div><h3 id="co-patch-target-heading">{t(strings.workbench.studio.patchTargetHeading)}</h3><p>{patch.composition.summary}</p></div></div>
                <dl className="co-patch-metadata">
                  <div><dt>{t(strings.workbench.studio.patchRegion)}</dt><dd>{patch.target.region_id}</dd></div>
                  <div><dt>{t(strings.workbench.studio.patchOutlineNode)}</dt><dd>{patch.target.outline_node_id}</dd></div>
                  <div><dt>{t(strings.workbench.studio.patchRclFrame)}</dt><dd>{patch.target.design_frame_id}</dd></div>
                  <div><dt>{t(strings.workbench.studio.patchRclNodes)}</dt><dd>{patch.target.design_node_ids.join(", ")}</dd></div>
                  <div><dt>{t(strings.workbench.studio.patchComponent)}</dt><dd>{patch.target.component_identity ?? t(strings.workbench.studio.patchNoRclMapping)}</dd></div>
                  <div><dt>{t(strings.workbench.studio.patchScenarioState)}</dt><dd>{patch.source.state_id} · {t(strings.workbench.studio.patchDataLabel)} {patch.source.data_id}</dd></div>
                  <div><dt>{t(strings.workbench.studio.patchRouteViewport)}</dt><dd>{patch.source.route} · {patch.source.viewport.width} × {patch.source.viewport.height}</dd></div>
                  <div><dt>{t(strings.workbench.studio.patchBuilds)}</dt><dd>{t(strings.workbench.studio.patchBuildSummary, { source: patch.source.build_id, preview: patch.preview.build_id })}</dd></div>
                </dl>
                {compositionMatches ? <p className="co-patch-verified"><Check aria-hidden size={14} />{t(strings.workbench.studio.patchRevisionMatches)}</p> : <p className="co-patch-warning" role="status"><TriangleAlert aria-hidden size={14} />{t(strings.workbench.studio.patchStaleTarget)}</p>}
              </section>
            </>
          ) : null}
        </div>

        {error ? <p className="co-patch-error" role="alert"><TriangleAlert aria-hidden size={14} />{error}</p> : null}
        <footer className="co-patch-footer">
          <WbButton type="button" onClick={onClose}>{t(strings.common.close)}</WbButton>
          {patch ? <WbButton type="button" onClick={exportPatch}><Download aria-hidden size={14} />{t(strings.workbench.studio.patchDownload)}</WbButton> : null}
          {patch ? <WbButton type="button" variant="primary" disabled={!compositionMatches || readOnly || !evidenceReady || working} onClick={apply}>{t(strings.workbench.studio.patchApply)}</WbButton> : null}
        </footer>
      </section>
    </div>
  );
}

type PatchErrorKey =
  | typeof strings.workbench.studio.patchTooLarge
  | typeof strings.workbench.studio.patchInvalidJson
  | typeof strings.workbench.studio.patchReadFailed
  | typeof strings.workbench.studio.patchEvidenceFailed;

function errorStringKey(message: string): PatchErrorKey | null {
  switch (message) {
    case "capturePatch.patchTooLarge": return strings.workbench.studio.patchTooLarge;
    case "capturePatch.invalidJson": return strings.workbench.studio.patchInvalidJson;
    case "capturePatch.patchReadFailed": return strings.workbench.studio.patchReadFailed;
    case "capturePatch.imageReadFailed": return strings.workbench.studio.patchEvidenceFailed;
    default: return null;
  }
}
