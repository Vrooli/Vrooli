import { useCallback, useMemo, useState } from "react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { Grid2X2, Minus, Plus, RotateCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { formatBytes } from "../format";
import { useImageInspection } from "../useImageInspection";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, PreviewButton, PreviewLoading, PreviewToolbar, previewSelectClass } from "./ArtifactChrome";
import { PreviewNotice } from "./shared";

export function ImagePreview(props: PreviewRendererProps) {
  return <ImageInspector key={`${props.model.previewId}:${props.model.blobHref}`} {...props} />;
}

function ImageInspector({ model }: PreviewRendererProps) {
  const { t } = useTranslation();
  const c = strings.artifactPreview;
  const [image, setImage] = useState({ width: 0, height: 0 });
  const [failed, setFailed] = useState(false);
  const [rotation, setRotation] = useState(0);
  const [background, setBackground] = useState(0);
  const { ref, rect } = useResizeObserver<HTMLDivElement>();
  const width = rect?.width ?? 0, height = rect?.height ?? 0;
  const stage = useMemo(() => ({ width, height }), [width, height]);
  const inspect = useImageInspection(stage, image, rotation);
  const stageElement = inspect.stageElement;
  const attachStage = useCallback((node: HTMLDivElement | null) => { ref(node); stageElement.current = node; }, [ref, stageElement]);
  const ready = image.width > 0 && !failed;
  const zoom = inspect.fitMode ? "fit" : String(inspect.view.scale);
  const scales = [0.25, 0.5, 1, 2, 4, 8];
  const backgroundLabels = [t(c.transparency), t(c.whiteBackground), t(c.darkBackground)];
  const metadata = [ready ? `${String(image.width)} × ${String(image.height)}` : "", model.basename.split(".").pop()?.toUpperCase(), formatBytes(model.sizeBytes), ready ? `${String(Math.round(inspect.view.scale * 100))}%` : ""].filter(Boolean).join(" · ");
  return <ArtifactFrame testId="file-preview-image">
    <PreviewToolbar label={t(c.imageControls)}>
      <div className="flex min-w-0 items-center gap-1">
        <PreviewButton label={t(c.zoomOut)} disabled={!ready || inspect.view.scale <= 0.01} onClick={() => { inspect.zoom(inspect.view.scale / 1.5); }}><Minus /></PreviewButton>
        <select aria-label={t(c.imageZoom)} value={zoom} disabled={!ready} className={`${previewSelectClass} w-20`} onChange={(event) => { if (event.target.value === "fit") inspect.reset(); else inspect.zoom(Number(event.target.value)); }}>
          <option value="fit">{t(c.fit)}</option>
          {!inspect.fitMode && !scales.includes(inspect.view.scale) && <option value={String(inspect.view.scale)}>{Math.round(inspect.view.scale * 100)}%</option>}
          {scales.map((scale) => <option key={scale} value={String(scale)}>{scale * 100}%</option>)}
        </select>
        <PreviewButton label={t(c.zoomIn)} disabled={!ready || inspect.view.scale >= 8} onClick={() => { inspect.zoom(inspect.view.scale * 1.5); }}><Plus /></PreviewButton>
      </div>
      <div className="flex items-center">
        <PreviewButton label={t(c.rotate)} disabled={!ready} onClick={() => { setRotation((value) => (value + 90) % 360); inspect.reset(); }}><RotateCw /></PreviewButton>
        <PreviewButton label={t(c.background, { name: backgroundLabels[background] })} onClick={() => { setBackground((value) => (value + 1) % 3); }}><Grid2X2 /></PreviewButton>
      </div>
    </PreviewToolbar>
    <div ref={attachStage}
      role="group" aria-label={t(c.imageStage)} tabIndex={0} data-testid={selectors.artifactPreview.imageStage}
      className={`relative min-h-0 flex-1 overflow-hidden outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent ${background === 0 ? "wc-image-checkerboard" : background === 1 ? "bg-white" : "bg-black"} ${ready ? "cursor-grab active:cursor-grabbing" : ""}`}
      style={{ touchAction: "none" }} {...inspect.pointerEvents}
      onDoubleClick={() => { if (ready) { if (inspect.fitMode) inspect.zoom(1); else inspect.reset(); } }}
      onKeyDown={(event) => {
        if (!ready) return;
        if (event.key === "+" || event.key === "=") inspect.zoom(inspect.view.scale * 1.5);
        else if (event.key === "-") inspect.zoom(inspect.view.scale / 1.5);
        else if (event.key === "0") inspect.reset();
        else if (event.key === "ArrowLeft") inspect.pan(40, 0);
        else if (event.key === "ArrowRight") inspect.pan(-40, 0);
        else if (event.key === "ArrowUp") inspect.pan(0, 40);
        else if (event.key === "ArrowDown") inspect.pan(0, -40);
        else return;
        event.preventDefault(); event.stopPropagation();
      }}>
      {!ready && !failed && <PreviewLoading label={t(c.loadingImage)} />}
      {failed ? <div role="alert" className="flex h-full items-center justify-center p-4"><PreviewNotice message={t(strings.messagesFileViewer.mediaLoadError)} /></div> : <img
        src={model.blobHref} alt={model.basename} draggable={false}
        onLoad={(event) => { setImage({ width: event.currentTarget.naturalWidth, height: event.currentTarget.naturalHeight }); }}
        onError={() => { setFailed(true); }} className="pointer-events-none absolute left-1/2 top-1/2 max-w-none select-none"
        style={{ width: image.width || undefined, height: image.height || undefined, opacity: ready ? 1 : 0, transform: `translate(-50%, -50%) translate3d(${String(inspect.view.x)}px, ${String(inspect.view.y)}px, 0) scale(${String(inspect.view.scale)}) rotate(${String(rotation)}deg)`, willChange: "transform" }} />}
    </div>
    <ArtifactFooter model={model} metadata={metadata} />
  </ArtifactFrame>;
}
