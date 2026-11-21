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
import "./region-capture.css";

type AspectId = "free" | "1:1" | "4:3" | "16:9";
type FieldKey = "x" | "y" | "w" | "h";
type Phase = "idle" | "encoding" | "saving";

const ASPECT_ORDER: readonly AspectId[] = ["free", "1:1", "4:3", "16:9"];
const ASPECT_RATIOS: Record<AspectId, number | null> = {
  free: null,
  "1:1": 1,
  "4:3": 4 / 3,
  "16:9": 16 / 9,
};
const HANDLES: readonly ResizeHandle[] = ["nw", "ne", "sw", "se"];
const KEY_STEP = 2;

interface Point {
  x: number;
  y: number;
}

interface DragState {
  mode: "new" | "move" | "resize";
  pointerId: number;
  start: Point;
  origin: Region;
  handle: ResizeHandle;
}

interface FieldDraft {
  key: FieldKey;
  value: string;
}

export interface RegionCaptureProps {
  /** The upload being cropped. The URL is already absolute. */
  asset: { id: string; name: string; url: string };
  onImport: (file: File, fidelity: "captured") => void | Promise<void>;
  onClose: () => void;
}

function percent(value: number, total: number): string {
  if (total <= 0) return "0%";
  return `${(value / total) * 100}%`;
}

function captureName(assetName: string): string {
  const base = assetName.replace(/\.[^./\\]+$/, "");
  return `capture-${base || "region"}.png`;
}

function encodeRegion(image: HTMLImageElement, region: Region): Promise<Blob> {
  return new Promise((resolve, reject) => {
    const canvas = document.createElement("canvas");
    canvas.width = region.w;
    canvas.height = region.h;
    const context = canvas.getContext("2d");
    if (!context) {
      reject(new Error("canvas-unavailable"));
      return;
    }
    context.drawImage(
      image,
      region.x,
      region.y,
      region.w,
      region.h,
      0,
      0,
      region.w,
      region.h,
    );
    try {
      canvas.toBlob(
        (blob) => {
          if (blob) resolve(blob);
          else reject(new Error("png-encode-failed"));
        },
        "image/png",
      );
    } catch (cause) {
      reject(cause instanceof Error ? cause : new Error("png-encode-failed"));
    }
  });
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
  const [aspect, setAspect] = useState<AspectId>("free");
  const [draft, setDraft] = useState<FieldDraft | null>(null);
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
        constrainRegion(
          resizeRegion(drag.origin, drag.handle, dx, dy, natural, MIN_REGION),
        ),
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

  const chooseAspect = (id: AspectId) => {
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

  const applyFieldValue = (key: FieldKey, value: number) => {
    if (!natural || !region) return;
    const next: Region = { ...region };
    if (key === "x") next.x = value;
    else if (key === "y") next.y = value;
    else if (key === "w") next.w = value;
    else next.h = value;
    setRegion(
      key === "x" || key === "y"
        ? clampRegion(next, natural, MIN_REGION)
        : constrainRegion(next),
    );
  };

  const changeField = (key: FieldKey, raw: string) => {
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

  const fieldLabels: Record<FieldKey, string> = {
    x: t("workbench.studio.regionX", { defaultValue: "X" }),
    y: t("workbench.studio.regionY", { defaultValue: "Y" }),
    w: t("workbench.studio.regionW", { defaultValue: "W" }),
    h: t("workbench.studio.regionH", { defaultValue: "H" }),
  };
  const aspectLabels: Record<AspectId, string> = {
    free: t("workbench.studio.aspectFree", { defaultValue: "Free" }),
    "1:1": t("workbench.studio.aspectSquare", { defaultValue: "1:1" }),
    "4:3": t("workbench.studio.aspectWide", { defaultValue: "4:3" }),
    "16:9": t("workbench.studio.aspectCinema", { defaultValue: "16:9" }),
  };

  const marqueeValue = region && natural
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
              defaultValue: "Drag on the image to select a region. Arrow keys move it by 2 px; hold Shift to resize.",
            })}
          </p>

          <div
            ref={stageRef}
            className="rc-stage"
            data-testid="region-capture-stage"
            onPointerDown={(event) => beginDrag(event, "new", "se")}
            onPointerMove={onStagePointerMove}
            onPointerUp={endDrag}
            onPointerCancel={endDrag}
            onLostPointerCapture={() => {
              dragRef.current = null;
            }}
          >
            <img
              ref={imageRef}
              className="rc-image"
              src={asset.url}
              alt={t("workbench.studio.cropImageAlt", {
                defaultValue: "{{name}}, full image for region capture",
                name: asset.name,
              })}
              draggable={false}
              onLoad={handleImageLoad}
              onError={() => {
                setImageFailed(true);
                setNatural(null);
                setRegion(null);
              }}
              data-testid="region-capture-image"
            />

            {region && natural ? (
              <div
                ref={marqueeRef}
                className="rc-marquee"
                role="slider"
                tabIndex={0}
                aria-label={t("workbench.studio.regionMarquee", {
                  defaultValue: "Selected region. Arrow keys move it; Shift plus arrow keys resize it.",
                })}
                aria-valuemin={0}
                aria-valuemax={Math.max(natural.width, natural.height)}
                aria-valuenow={region.w}
                aria-valuetext={marqueeValue}
                data-testid="region-capture-marquee"
                style={{
                  left: percent(region.x, natural.width),
                  top: percent(region.y, natural.height),
                  width: percent(region.w, natural.width),
                  height: percent(region.h, natural.height),
                }}
                onKeyDown={onMarqueeKeyDown}
                onPointerDown={(event) => {
                  marqueeRef.current?.focus();
                  beginDrag(event, regionCoversImage ? "new" : "move", "se");
                }}
              >
                {HANDLES.map((handle) => (
                  <span
                    key={handle}
                    className={`rc-handle rc-handle-${handle}`}
                    aria-hidden="true"
                    data-testid={`region-capture-handle-${handle}`}
                    onPointerDown={(event) => beginDrag(event, "resize", handle)}
                  />
                ))}
              </div>
            ) : null}

            {!natural && !imageFailed ? (
              <span className="rc-loading" role="status">
                {t("workbench.studio.regionLoadingImage", { defaultValue: "Loading image…" })}
              </span>
            ) : null}
          </div>

          {imageFailed ? (
            <p className="rc-error" role="alert" data-testid="region-capture-image-error">
              {t("workbench.studio.regionImageFailed", {
                defaultValue: "This image could not be loaded for cropping. Close and try another asset.",
              })}
            </p>
          ) : null}

          {error ? (
            <p className="rc-error" role="alert" data-testid="region-capture-error">
              {error}
            </p>
          ) : null}

          <div className="rc-controls">
            <div className="rc-aspect-group" role="group" aria-label={t("workbench.studio.aspectGroup", { defaultValue: "Aspect ratio" })}>
              {ASPECT_ORDER.map((id) => (
                <button
                  key={id}
                  type="button"
                  className="rc-aspect"
                  aria-pressed={aspect === id}
                  disabled={!region || imageFailed || busy}
                  onClick={() => chooseAspect(id)}
                  data-testid={`region-capture-aspect-${id}`}
                >
                  {aspectLabels[id]}
                </button>
              ))}
            </div>

            <div className="rc-fields">
              {(["x", "y", "w", "h"] as const).map((key) => (
                <label key={key} className="rc-field">
                  <span>{fieldLabels[key]}</span>
                  <input
                    type="number"
                    min={0}
                    step={1}
                    inputMode="numeric"
                    value={draft?.key === key ? draft.value : (region?.[key] ?? 0)}
                    disabled={!region || imageFailed || busy}
                    onChange={(event) => changeField(key, event.target.value)}
                    onBlur={commitDraft}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") event.currentTarget.blur();
                    }}
                    data-testid={`region-capture-${key}`}
                  />
                </label>
              ))}
            </div>
          </div>
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
