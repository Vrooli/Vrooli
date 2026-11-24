import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { Crop, X } from "lucide-react";

import { useTranslation } from "../../i18n";
import { WbButton } from "../primitives";
import { useFocusTrap } from "../useFocusTrap";
import {
  RegionCaptureControls,
  type RegionCaptureAspect,
  type RegionCaptureDraft,
  type RegionCaptureField,
} from "./RegionCaptureControls";
import {
  MIN_REGION,
  applyAspect,
  clampRegion,
  moveRegion,
  regionFromDrag,
  resizeRegion,
  type Region,
  type RegionBounds,
  type ResizeHandle,
} from "./region";
import { ASPECT_RATIOS, type DragState, type Point } from "./regionCaptureGeometry";
import { RegionCaptureStage } from "./RegionCaptureStage";
import { captureName, encodeRegion } from "./captureRegionImage";
import "./region-capture.css";

type Phase = "idle" | "encoding" | "saving";

const KEY_STEP = 2;

export interface RegionCaptureProps {
  /** The upload being cropped. The URL is already absolute. */
  asset: { id: string; name: string; url: string };
  onImport: (file: File, fidelity: "captured") => void | Promise<void>;
  onClose: () => void;
}

/**
 * Region capture: an accessible dialog over the Assets pane. The person drags
 * a rectangle on the natural-size image, refines it with the keyboard or the
 * numeric fields, then the rectangle is rendered through `<canvas>` and handed
 * back as a PNG for the `captured` asset path.
 */
export function RegionCapture({ asset, onImport, onClose }: RegionCaptureProps) {
  const { t } = useTranslation();
  const titleId = useId();
  const trapRef = useFocusTrap<HTMLDivElement>(true, onClose);
  const imageRef = useRef<HTMLImageElement>(null);
  const stageRef = useRef<HTMLDivElement>(null);
  const marqueeRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<DragState | null>(null);

  const [natural, setNatural] = useState<RegionBounds | null>(null);
  const [region, setRegion] = useState<Region | null>(null);
  const [aspect, setAspect] = useState<RegionCaptureAspect>("free");
  const [draft, setDraft] = useState<RegionCaptureDraft | null>(null);
  const [phase, setPhase] = useState<Phase>("idle");
  const [error, setError] = useState<string | null>(null);
  const [imageFailed, setImageFailed] = useState(false);

  const busy = phase !== "idle";
  const ratio = ASPECT_RATIOS[aspect];

  const handleImageLoad = useCallback(() => {
    const image = imageRef.current;
    if (!image) return;
    const width = image.naturalWidth;
    const height = image.naturalHeight;
    if (width <= 0 || height <= 0) {
      setImageFailed(true);
      return;
    }
    const size = { width, height };
    setImageFailed(false);
    setNatural(size);
    setRegion((current) =>
      clampRegion(current ?? { x: 0, y: 0, w: width, h: height }, size, MIN_REGION),
    );
  }, []);

  // A cached image can finish before React attaches the load listener.
  useEffect(() => {
    const image = imageRef.current;
    if (image?.complete && image.naturalWidth > 0) handleImageLoad();
  }, [handleImageLoad]);

  const pointFromEvent = (clientX: number, clientY: number): Point | null => {
    const image = imageRef.current;
    if (!image || !natural) return null;
    const rect = image.getBoundingClientRect();
    const x = rect.width > 0 ? ((clientX - rect.left) / rect.width) * natural.width : 0;
    const y = rect.height > 0 ? ((clientY - rect.top) / rect.height) * natural.height : 0;
    return {
      x: Math.min(Math.max(x, 0), natural.width),
      y: Math.min(Math.max(y, 0), natural.height),
    };
  };

  const constrainRegion = (next: Region): Region => {
    if (!natural) return next;
    return ratio === null
      ? clampRegion(next, natural, MIN_REGION)
      : applyAspect(next, ratio, natural, "center", MIN_REGION);
  };

  const beginDrag = (
    event: ReactPointerEvent<HTMLElement>,
    mode: DragState["mode"],
    handle: ResizeHandle,
  ) => {
    if (event.button !== 0 || !natural || !region || busy) return;
    const point = pointFromEvent(event.clientX, event.clientY);
    const stage = stageRef.current;
    if (!point || !stage) return;
    dragRef.current = {
      mode,
      pointerId: event.pointerId,
      start: point,
      origin: region,
      handle,
    };
    stage.setPointerCapture(event.pointerId);
    event.preventDefault();
    event.stopPropagation();
    setDraft(null);
    if (mode === "new") {
      setRegion(constrainRegion(regionFromDrag(point, point, natural, MIN_REGION)));
    }
  };

  const onStagePointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId || !natural) return;
    const point = pointFromEvent(event.clientX, event.clientY);
    if (!point) return;
    const dx = point.x - drag.start.x;
    const dy = point.y - drag.start.y;
    if (drag.mode === "new") {
      setRegion(constrainRegion(regionFromDrag(drag.start, point, natural, MIN_REGION)));
    } else if (drag.mode === "move") {
      setRegion(moveRegion(drag.origin, dx, dy, natural));
    } else {
      setRegion(
        constrainRegion(resizeRegion(drag.origin, drag.handle, dx, dy, natural, MIN_REGION)),
      );
    }
  };

  const endDrag = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (dragRef.current?.pointerId !== event.pointerId) return;
    dragRef.current = null;
    const stage = stageRef.current;
    if (stage?.hasPointerCapture(event.pointerId)) {
      stage.releasePointerCapture(event.pointerId);
    }
  };

  const onMarqueeKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (!natural || !region || busy) return;
    let dx = 0;
    let dy = 0;
    if (event.key === "ArrowLeft") dx = -KEY_STEP;
    else if (event.key === "ArrowRight") dx = KEY_STEP;
    else if (event.key === "ArrowUp") dy = -KEY_STEP;
    else if (event.key === "ArrowDown") dy = KEY_STEP;
    else return;
    event.preventDefault();
    setDraft(null);
    setRegion(
      event.shiftKey
        ? constrainRegion(resizeRegion(region, "se", dx, dy, natural, MIN_REGION))
        : moveRegion(region, dx, dy, natural),
    );
  };

  const chooseAspect = (id: RegionCaptureAspect) => {
    setAspect(id);
    setDraft(null);
    if (!region || !natural) return;
    const nextRatio = ASPECT_RATIOS[id];
    setRegion(
      nextRatio === null
        ? clampRegion(region, natural, MIN_REGION)
        : applyAspect(region, nextRatio, natural, "center", MIN_REGION),
    );
  };

  const applyFieldValue = (key: RegionCaptureField, value: number) => {
    if (!natural || !region) return;
    const next: Region = { ...region };
    if (key === "x") next.x = value;
    else if (key === "y") next.y = value;
    else if (key === "w") next.w = value;
    else next.h = value;
    setRegion(
      key === "x" || key === "y" ? clampRegion(next, natural, MIN_REGION) : constrainRegion(next),
    );
  };

  const changeField = (key: RegionCaptureField, raw: string) => {
    setDraft({ key, value: raw });
    const value = Number.parseInt(raw, 10);
    if (raw.trim() !== "" && Number.isFinite(value)) applyFieldValue(key, value);
  };

  const commitDraft = () => {
    if (!draft) return;
    const value = Number.parseInt(draft.value, 10);
    if (Number.isFinite(value)) applyFieldValue(draft.key, value);
    setDraft(null);
  };

  const capture = async () => {
    const image = imageRef.current;
    if (!image || !natural || !region || busy) return;
    setError(null);
    setPhase("encoding");
    try {
      const blob = await encodeRegion(image, region);
      setPhase("saving");
      const file = new File([blob], captureName(asset.name), { type: "image/png" });
      await onImport(file, "captured");
      onClose();
    } catch {
      setError(
        t("workbench.studio.regionCaptureFailed", {
          defaultValue: "The region could not be captured. Try again or pick a smaller area.",
        }),
      );
      setPhase("idle");
    }
  };

  const marqueeValue =
    region && natural
      ? t("workbench.studio.regionValue", {
          defaultValue: "x {{x}}, y {{y}}, {{w}} by {{h}} pixels",
          x: region.x,
          y: region.y,
          w: region.w,
          h: region.h,
        })
      : "";

  const regionCoversImage =
    region !== null &&
    natural !== null &&
    region.x === 0 &&
    region.y === 0 &&
    region.w === natural.width &&
    region.h === natural.height;

  const primaryLabel =
    phase === "encoding"
      ? t("workbench.studio.regionCapturing", { defaultValue: "Capturing…" })
      : phase === "saving"
        ? t("workbench.studio.regionSaving", { defaultValue: "Saving capture…" })
        : t("workbench.studio.captureRegion", { defaultValue: "Capture region" });

  return (
    <div className="rc-layer">
      <div className="rc-backdrop" aria-hidden="true" onPointerDown={() => onClose()} />
      <div
        ref={trapRef}
        className="rc-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-busy={busy}
        data-testid="region-capture"
      >
        <header className="rc-head">
          <h2 id={titleId} className="rc-title">
            {t("workbench.studio.cropRegionTitle", {
              defaultValue: "Crop a region from {{name}}",
              name: asset.name,
            })}
          </h2>
          <button
            type="button"
            className="rc-close"
            aria-label={t("workbench.studio.closeRegionCapture", {
              defaultValue: "Close region capture",
            })}
            onClick={onClose}
            data-testid="region-capture-close"
          >
            <X aria-hidden size={14} />
          </button>
        </header>

        <div className="rc-body">
          <p className="rc-intro">
            {t("workbench.studio.cropRegionHint", {
              defaultValue:
                "Drag on the image to select a region. Arrow keys move it by 2 px; hold Shift to resize.",
            })}
          </p>

          <RegionCaptureStage
            asset={asset}
            imageRef={imageRef}
            stageRef={stageRef}
            marqueeRef={marqueeRef}
            natural={natural}
            region={region}
            imageFailed={imageFailed}
            marqueeValue={marqueeValue}
            regionCoversImage={regionCoversImage}
            imageAlt={t("workbench.studio.cropImageAlt", {
              defaultValue: "{{name}}, full image for region capture",
              name: asset.name,
            })}
            marqueeLabel={t("workbench.studio.regionMarquee", {
              defaultValue: "Selected region. Arrow keys move it; Shift plus arrow keys resize it.",
            })}
            loadingLabel={t("workbench.studio.regionLoadingImage", {
              defaultValue: "Loading image…",
            })}
            onImageLoad={handleImageLoad}
            onImageError={() => {
              setImageFailed(true);
              setNatural(null);
              setRegion(null);
            }}
            onBeginDrag={beginDrag}
            onPointerMove={onStagePointerMove}
            onPointerEnd={endDrag}
            onLostPointerCapture={() => {
              dragRef.current = null;
            }}
            onMarqueeKeyDown={onMarqueeKeyDown}
          />

          {imageFailed ? (
            <p className="rc-error" role="alert" data-testid="region-capture-image-error">
              {t("workbench.studio.regionImageFailed", {
                defaultValue:
                  "This image could not be loaded for cropping. Close and try another asset.",
              })}
            </p>
          ) : null}

          {error ? (
            <p className="rc-error" role="alert" data-testid="region-capture-error">
              {error}
            </p>
          ) : null}

          <RegionCaptureControls
            aspect={aspect}
            draft={draft}
            imageFailed={imageFailed}
            busy={busy}
            region={region}
            onChooseAspect={chooseAspect}
            onChangeField={changeField}
            onCommitDraft={commitDraft}
          />
        </div>

        <footer className="rc-foot">
          <span className="rc-hint">
            {t("workbench.studio.regionKeyboardHint", {
              defaultValue: "Arrow keys move · Shift + arrows resize",
            })}
          </span>
          <div className="rc-actions">
            <WbButton variant="ghost" size="sm" onClick={onClose} disabled={busy}>
              {t("workbench.studio.cancel", { defaultValue: "Cancel" })}
            </WbButton>
            <WbButton
              variant="primary"
              size="sm"
              onClick={() => void capture()}
              disabled={!region || imageFailed || busy}
              data-testid="region-capture-confirm"
            >
              <Crop aria-hidden size={13} />
              {primaryLabel}
            </WbButton>
          </div>
        </footer>
      </div>
    </div>
  );
}
