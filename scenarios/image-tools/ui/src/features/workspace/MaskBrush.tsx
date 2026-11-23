import { useCallback, useRef, useState } from "react";

import { Slider } from "../../components/ui/slider";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SensitiveImage } from "../sensitive/SensitiveImage";
import { MaskSelectByName } from "./MaskSelectByName";
import type { NamedSelectionClient } from "./useNamedSelection";

export interface MaskBrushProps {
  /** The image being masked (the current canvas image), or null. */
  imageUrl: string | null;
  /** Emits the painted/uploaded mask as a PNG File, or null when cleared. */
  onMask: (mask: File | null) => void;
  /** The picture itself, so a part of it can be selected by name; omit to offer painting only. */
  image?: File | null;
  /** Injected seam for the by-name selection (tests); the live service otherwise. */
  selectionClient?: NamedSelectionClient;
}

const MIN_BRUSH = 8;
const MAX_BRUSH = 96;
const DEFAULT_BRUSH = 36;

/**
 * Paint-a-mask tool for the masked generation ops (inpaint / object removal).
 * The user brushes white over the region to change; the exported PNG is that
 * region in white on black — the mask contract the backend's multipart edge
 * expects. An "upload a mask" file input is the accessible, non-pointer
 * fallback (the brush itself needs a 2D canvas, which a keyboard user or a
 * headless test can't drive), so no path is pointer-only.
 */
export function MaskBrush({ imageUrl, onMask, image = null, selectionClient }: MaskBrushProps) {
  const { t } = useTranslation();
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const naturalRef = useRef<{ width: number; height: number }>({ width: 0, height: 0 });
  const paintingRef = useRef(false);
  const lastRef = useRef<{ x: number; y: number } | null>(null);

  const [brush, setBrush] = useState(DEFAULT_BRUSH);
  const [painted, setPainted] = useState(false);
  const [uploadedName, setUploadedName] = useState<string | null>(null);
  const [selectedPhrase, setSelectedPhrase] = useState<string | null>(null);

  // Size the paint layer to the image's natural pixels so the exported mask
  // lines up with the backend's full-resolution input.
  const onImageLoad = useCallback((event: React.SyntheticEvent<HTMLImageElement>) => {
    const img = event.currentTarget;
    naturalRef.current = { width: img.naturalWidth, height: img.naturalHeight };
    const canvas = canvasRef.current;
    if (canvas) {
      canvas.width = img.naturalWidth;
      canvas.height = img.naturalHeight;
    }
  }, []);

  const exportMask = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas || canvas.width === 0) {
      return;
    }
    // Composite the white strokes over black: a fresh canvas, black fill, then
    // the paint layer on top → white-on-black mask.
    const out = document.createElement("canvas");
    out.width = canvas.width;
    out.height = canvas.height;
    const ctx = out.getContext("2d");
    if (!ctx) {
      return;
    }
    ctx.fillStyle = "#000000";
    ctx.fillRect(0, 0, out.width, out.height);
    ctx.drawImage(canvas, 0, 0);
    out.toBlob((blob) => {
      if (blob) {
        onMask(new File([blob], "mask.png", { type: "image/png" }));
      }
    }, "image/png");
  }, [onMask]);

  const pointAt = (event: React.PointerEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas) {
      return null;
    }
    const rect = canvas.getBoundingClientRect();
    if (rect.width === 0 || rect.height === 0) {
      return null;
    }
    return {
      x: ((event.clientX - rect.left) / rect.width) * canvas.width,
      y: ((event.clientY - rect.top) / rect.height) * canvas.height,
    };
  };

  const stroke = (from: { x: number; y: number } | null, to: { x: number; y: number }) => {
    const ctx = canvasRef.current?.getContext("2d");
    if (!ctx) {
      return;
    }
    ctx.strokeStyle = "#ffffff";
    ctx.fillStyle = "#ffffff";
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctx.lineWidth = brush;
    ctx.beginPath();
    ctx.arc(to.x, to.y, brush / 2, 0, Math.PI * 2);
    ctx.fill();
    if (from) {
      ctx.beginPath();
      ctx.moveTo(from.x, from.y);
      ctx.lineTo(to.x, to.y);
      ctx.stroke();
    }
  };

  const onPointerDown = (event: React.PointerEvent<HTMLCanvasElement>) => {
    if (uploadedName) {
      return;
    }
    const point = pointAt(event);
    if (!point) {
      return;
    }
    event.currentTarget.setPointerCapture(event.pointerId);
    paintingRef.current = true;
    lastRef.current = point;
    stroke(null, point);
    setPainted(true);
  };

  const onPointerMove = (event: React.PointerEvent<HTMLCanvasElement>) => {
    if (!paintingRef.current) {
      return;
    }
    const point = pointAt(event);
    if (!point) {
      return;
    }
    stroke(lastRef.current, point);
    lastRef.current = point;
  };

  const endStroke = () => {
    if (!paintingRef.current) {
      return;
    }
    paintingRef.current = false;
    lastRef.current = null;
    exportMask();
  };

  const clear = () => {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    if (canvas && ctx) {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
    }
    setPainted(false);
    setUploadedName(null);
    setSelectedPhrase(null);
    onMask(null);
  };

  // A selection by name lands on the paint layer (white where it selects), so the
  // brush can add to it; without a canvas it is simply the mask.
  const applySelection = async (selected: File, phrase: string) => {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    setSelectedPhrase(phrase);
    setPainted(true);
    if (!canvas || !ctx || canvas.width === 0 || typeof createImageBitmap !== "function") {
      onMask(selected);
      return;
    }
    try {
      const bitmap = await createImageBitmap(selected);
      const layer = document.createElement("canvas");
      layer.width = canvas.width;
      layer.height = canvas.height;
      const layerCtx = layer.getContext("2d");
      if (!layerCtx) {
        onMask(selected);
        return;
      }
      layerCtx.drawImage(bitmap, 0, 0, layer.width, layer.height);
      const pixels = layerCtx.getImageData(0, 0, layer.width, layer.height);
      const data = pixels.data;
      for (let i = 0; i < data.length; i += 4) {
        // White-on-black mask in, white-on-transparent paint out.
        data[i + 3] = Math.round(((data[i] ?? 0) + (data[i + 1] ?? 0) + (data[i + 2] ?? 0)) / 3);
        data[i] = data[i + 1] = data[i + 2] = 255;
      }
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.putImageData(pixels, 0, 0);
      bitmap.close();
      exportMask();
    } catch {
      onMask(selected);
    }
  };

  const onUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setUploadedName(file.name);
      setPainted(true);
      onMask(file);
    }
  };

  const status = uploadedName
    ? t(strings.workspace.mask.uploaded, { name: uploadedName })
    : selectedPhrase
      ? t(strings.workspace.mask.selected, { phrase: selectedPhrase })
      : painted
      ? t(strings.workspace.mask.painted)
      : t(strings.workspace.mask.empty);

  return (
    <section
      data-testid={selectors.workspace.mask.root}
      aria-label={t(strings.workspace.mask.title)}
      className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-3"
    >
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-app-foreground">
          {t(strings.workspace.mask.title)}
        </span>
        <button
          type="button"
          data-testid={selectors.workspace.mask.clear}
          onClick={clear}
          className="text-xs text-app-primary underline"
        >
          {t(strings.workspace.mask.clear)}
        </button>
      </div>
      <p className="text-xs text-app-muted-foreground">{t(strings.workspace.mask.hint)}</p>

      {!uploadedName && (
        <MaskSelectByName image={image} client={selectionClient} onSelected={(selected, phrase) => void applySelection(selected, phrase)} />
      )}

      {/* The box hugs the picture so the paint layer (inset-0) covers exactly the pixels
          it edits; a full-width box letterboxed portrait photos and skewed every stroke. */}
      {imageUrl && !uploadedName && (
        <div className="relative mx-auto w-fit max-w-full overflow-hidden rounded-control border border-app-border">
          <SensitiveImage
            src={imageUrl}
            interactive={false}
            alt=""
            aria-hidden="true"
            onLoad={onImageLoad}
            className="block h-auto max-h-[55vh] w-auto max-w-full"
          />
          <canvas
            ref={canvasRef}
            data-testid={selectors.workspace.mask.canvas}
            aria-label={t(strings.workspace.mask.title)}
            onPointerDown={onPointerDown}
            onPointerMove={onPointerMove}
            onPointerUp={endStroke}
            onPointerLeave={endStroke}
            className="absolute inset-0 h-full w-full cursor-crosshair opacity-50 touch-none"
          />
        </div>
      )}

      {!uploadedName && (
        <Slider
          label={t(strings.workspace.mask.brushSize)}
          value={brush}
          min={MIN_BRUSH}
          max={MAX_BRUSH}
          unit="px"
          defaultValue={DEFAULT_BRUSH}
          resetLabel={t(strings.workspace.control.reset)}
          onChange={setBrush}
          data-testid={selectors.workspace.mask.brushSize}
        />
      )}

      <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
        <span>{t(strings.workspace.mask.uploadLabel)}</span>
        <input
          data-testid={selectors.workspace.mask.upload}
          type="file"
          accept="image/*"
          onChange={onUpload}
          className="block w-full text-xs text-app-muted-foreground file:mr-3 file:rounded-control file:border-0 file:bg-app-primary file:px-3 file:py-2 file:text-app-primary-foreground"
        />
      </label>

      <p data-testid={selectors.workspace.mask.status} className="text-xs text-app-muted-foreground">
        {status}
      </p>
    </section>
  );
}
