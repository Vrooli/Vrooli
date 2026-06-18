import { useEffect, useRef, useState } from "react";
import { Loader2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { RunOpImageResult } from "../../api/ops";
import { BeforeAfter } from "./BeforeAfter";
import { CropOverlay } from "./CropOverlay";
import type { Rect, Size } from "./cropMath";

/** Live progress for an async op resolving in the canvas (Enhance / Create). */
export interface CanvasProgress {
  percent: number;
  message: string;
}

const ZOOM_MIN = 0.1;
const ZOOM_MAX = 8;
const ZOOM_STEP = 1.25;

const clampZoom = (z: number) => Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, z));

/** Crop-overlay wiring; non-null only when the crop op is the active edit. */
export interface CanvasCrop {
  rect: Rect;
  onChange: (rect: Rect) => void;
  /** Called once the image's natural size is known (to seed a full-image box). */
  onNatural: (natural: Size) => void;
}

export interface WorkspaceCanvasProps {
  baseUrl: string | null;
  previewUrl: string | null;
  hasSteps: boolean;
  currentResult: RunOpImageResult | null;
  metadata: string | null;
  onFile: (file: File) => void;
  /** When set, draw a draggable crop box mapped 1:1 over the image. */
  crop?: CanvasCrop | null;
  /** When set, overlay an async op's live progress over the image. */
  progress?: CanvasProgress | null;
  /**
   * Increments when an async op (Enhance/Create) lands a result; bumping it
   * auto-engages the before/after compare so the change is immediately visible.
   */
  compareSignal?: number;
}

/**
 * The work surface: the loaded image on a transparency checkerboard, with
 * zoom / fit / actual-size, an optional before↔after compare, and a metadata
 * panel for read ops. The empty state is the single upload affordance (drag,
 * click, or mobile capture). Pan is native scroll of the viewport container
 * (never a focus-scroll, which would cross the iframe boundary); height uses
 * the parent's height chain (`h-full`/`min-h`), not a viewport-height class.
 */
export function WorkspaceCanvas({
  baseUrl,
  previewUrl,
  hasSteps,
  currentResult,
  metadata,
  onFile,
  crop = null,
  progress = null,
  compareSignal = 0,
}: WorkspaceCanvasProps) {
  const { t } = useTranslation();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const [zoom, setZoom] = useState(1);
  const [fit, setFit] = useState(true);
  const [comparing, setComparing] = useState(false);
  const [natural, setNatural] = useState<Size | null>(null);
  const [client, setClient] = useState<Size | null>(null);

  // Auto-engage before/after when an async op lands a result, so the user sees
  // the change without reaching for the compare toggle.
  useEffect(() => {
    if (compareSignal > 0 && hasSteps) {
      setComparing(true);
    }
  }, [compareSignal, hasSteps]);

  // Track the rendered image's on-screen box so the crop overlay maps 1:1.
  // ResizeObserver keeps the box correct across zoom/fit and viewport changes.
  useEffect(() => {
    const img = imgRef.current;
    if (!crop || !img) {
      return;
    }
    const measure = () => setClient({ width: img.clientWidth, height: img.clientHeight });
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(img);
    return () => observer.disconnect();
  }, [crop, baseUrl, previewUrl, fit, zoom]);

  const zoomIn = () => {
    setFit(false);
    setZoom((z) => clampZoom(z * ZOOM_STEP));
  };
  const zoomOut = () => {
    setFit(false);
    setZoom((z) => clampZoom(z / ZOOM_STEP));
  };
  const actualSize = () => {
    setFit(false);
    setZoom(1);
  };
  const toFit = () => {
    setFit(true);
    setZoom(1);
  };

  const openPicker = () => fileInputRef.current?.click();
  const onDrop = (event: React.DragEvent<HTMLButtonElement>) => {
    event.preventDefault();
    const file = event.dataTransfer.files[0];
    if (file) {
      onFile(file);
    }
  };

  const percent = Math.round((fit ? 1 : zoom) * 100);
  const showCompare = comparing && hasSteps && baseUrl !== null && previewUrl !== null;

  return (
    <div
      data-testid={selectors.workspace.canvas.root}
      className="flex min-h-0 flex-1 flex-col gap-2"
    >
      <div className="flex flex-wrap items-center gap-1">
        <Button
          variant="outline"
          size="sm"
          type="button"
          data-testid={selectors.workspace.canvas.zoomOut}
          aria-label={t(strings.workspace.canvas.zoomOut)}
          onClick={zoomOut}
          disabled={baseUrl === null}
        >
          <span aria-hidden="true">−</span>
        </Button>
        <span className="min-w-[3.5rem] text-center text-xs tabular-nums text-app-muted-foreground">
          {t(strings.workspace.canvas.zoomLevel, { percent })}
        </span>
        <Button
          variant="outline"
          size="sm"
          type="button"
          data-testid={selectors.workspace.canvas.zoomIn}
          aria-label={t(strings.workspace.canvas.zoomIn)}
          onClick={zoomIn}
          disabled={baseUrl === null}
        >
          <span aria-hidden="true">+</span>
        </Button>
        <Button
          variant="outline"
          size="sm"
          type="button"
          data-testid={selectors.workspace.canvas.zoomActual}
          onClick={actualSize}
          disabled={baseUrl === null}
        >
          {t(strings.workspace.canvas.zoomActual)}
        </Button>
        <Button
          variant="outline"
          size="sm"
          type="button"
          data-testid={selectors.workspace.canvas.zoomFit}
          onClick={toFit}
          disabled={baseUrl === null}
        >
          {t(strings.workspace.canvas.zoomFit)}
        </Button>
        {hasSteps && (
          <Button
            variant="outline"
            size="sm"
            type="button"
            data-testid={selectors.workspace.canvas.compareToggle}
            aria-pressed={comparing}
            onClick={() => setComparing((c) => !c)}
          >
            {t(strings.workspace.canvas.compareToggle)}
          </Button>
        )}
        {baseUrl !== null && (
          <Button
            variant="outline"
            size="sm"
            type="button"
            data-testid={selectors.workspace.canvas.replace}
            onClick={openPicker}
          >
            {t(strings.workspace.canvas.replace)}
          </Button>
        )}
      </div>

      <div className="canvas-checker relative flex min-h-64 flex-1 items-center justify-center overflow-auto rounded-panel border border-app-border p-3">
        {baseUrl === null ? (
          <button
            type="button"
            data-testid={selectors.workspace.dropzone}
            onClick={openPicker}
            onDrop={onDrop}
            onDragOver={(e) => e.preventDefault()}
            className="flex h-full w-full flex-col items-center justify-center gap-1 rounded-panel border border-dashed border-app-border bg-app-surface/70 p-6 text-center text-sm text-app-muted-foreground hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          >
            <span className="font-medium text-app-foreground">{t(strings.workspace.uploadLabel)}</span>
            <span>{t(strings.workspace.canvas.empty)}</span>
          </button>
        ) : showCompare ? (
          <BeforeAfter beforeUrl={baseUrl} afterUrl={previewUrl} />
        ) : (
          <div className="relative inline-block">
            <img
              ref={imgRef}
              data-testid={selectors.workspace.canvas.image}
              src={previewUrl ?? baseUrl}
              alt={hasSteps ? t(strings.workspace.resultLabel) : t(strings.workspace.originalLabel)}
              className={fit ? "max-h-full max-w-full object-contain" : "h-auto"}
              style={fit ? undefined : { width: `${percent}%`, maxWidth: "none" }}
              onLoad={(e) => {
                const el = e.currentTarget;
                const size = { width: el.naturalWidth, height: el.naturalHeight };
                setNatural(size);
                setClient({ width: el.clientWidth, height: el.clientHeight });
                crop?.onNatural(size);
              }}
            />
            {crop && natural && client ? (
              <CropOverlay
                natural={natural}
                client={client}
                rect={crop.rect}
                onChange={crop.onChange}
              />
            ) : null}
          </div>
        )}
        {progress && baseUrl !== null && (
          <div
            data-testid={selectors.workspace.canvas.progress}
            aria-live="polite"
            className="absolute inset-0 z-10 flex flex-col items-center justify-center gap-2 rounded-panel bg-app-background/75"
          >
            <Loader2 aria-hidden="true" className="h-6 w-6 animate-spin text-app-brand" />
            <span className="text-sm font-medium text-app-foreground">{progress.percent}%</span>
            {progress.message && (
              <span className="text-xs text-app-muted-foreground">{progress.message}</span>
            )}
          </div>
        )}
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          capture="environment"
          data-testid={selectors.workspace.fileInput}
          aria-label={t(strings.workspace.uploadLabel)}
          className="sr-only"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) {
              onFile(file);
            }
          }}
        />
      </div>

      {currentResult && (
        <p
          data-testid={selectors.workspace.canvas.meta}
          className="text-xs text-app-muted-foreground"
        >
          {t(strings.workspace.resultMeta, {
            width: currentResult.width,
            height: currentResult.height,
            format: currentResult.format,
          })}
        </p>
      )}

      {metadata && (
        <figure className="flex flex-col gap-1">
          <figcaption className="text-xs text-app-muted-foreground">
            {t(strings.workspace.metadataLabel)}
          </figcaption>
          <pre
            data-testid={selectors.workspace.canvas.metadataOutput}
            className="max-h-48 overflow-auto rounded-panel border border-app-border bg-app-surface-muted p-3 text-xs text-app-foreground"
          >
            {metadata}
          </pre>
        </figure>
      )}
    </div>
  );
}
