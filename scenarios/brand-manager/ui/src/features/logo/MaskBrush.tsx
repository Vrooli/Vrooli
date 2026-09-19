import { useCallback, useEffect, useRef, useState } from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const CANVAS_EDGE = 512;
const DEFAULT_BRUSH = 24;

interface MaskBrushProps {
  /** Called with the exported mask PNG when the operator submits. */
  onSubmit: (mask: Blob) => void;
  disabled?: boolean;
}

/**
 * MaskBrush paints a white-on-black mask over a 512px canvas and exports it as a
 * PNG. The white region is what image-tools' object_removal removes. It degrades
 * to a no-op when a 2D context is unavailable (jsdom tests mock getContext).
 */
export function MaskBrush({ onSubmit, disabled = false }: MaskBrushProps) {
  const { t } = useTranslation();
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const drawing = useRef(false);
  const [brush, setBrush] = useState(DEFAULT_BRUSH);
  const [dirty, setDirty] = useState(false);

  const fillBlack = useCallback(() => {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    if (!canvas || !ctx) {
      return;
    }
    ctx.fillStyle = "#000000";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
  }, []);

  useEffect(() => {
    fillBlack();
  }, [fillBlack]);

  const strokeAt = useCallback(
    (clientX: number, clientY: number) => {
      const canvas = canvasRef.current;
      const ctx = canvas?.getContext("2d");
      if (!canvas || !ctx) {
        return;
      }
      const rect = canvas.getBoundingClientRect();
      const scaleX = canvas.width / rect.width;
      const scaleY = canvas.height / rect.height;
      const x = (clientX - rect.left) * scaleX;
      const y = (clientY - rect.top) * scaleY;
      ctx.fillStyle = "#ffffff";
      ctx.beginPath();
      ctx.arc(x, y, brush, 0, Math.PI * 2);
      ctx.fill();
      setDirty(true);
    },
    [brush],
  );

  const submit = () => {
    const canvas = canvasRef.current;
    if (!canvas) {
      return;
    }
    canvas.toBlob((blob) => {
      if (blob) {
        onSubmit(blob);
      }
    }, "image/png");
  };

  return (
    <div data-testid={selectors.logo.maskBrush} className="flex flex-col gap-2">
      <p className="text-xs text-slate-500">{t(strings.logo.maskHint)}</p>
      <canvas
        ref={canvasRef}
        width={CANVAS_EDGE}
        height={CANVAS_EDGE}
        className="h-48 w-48 cursor-crosshair rounded-lg border border-white/10 bg-black"
        onPointerDown={(event) => {
          drawing.current = true;
          strokeAt(event.clientX, event.clientY);
        }}
        onPointerMove={(event) => {
          if (drawing.current) {
            strokeAt(event.clientX, event.clientY);
          }
        }}
        onPointerUp={() => {
          drawing.current = false;
        }}
        onPointerLeave={() => {
          drawing.current = false;
        }}
      />
      <label className="flex items-center gap-2 text-xs text-slate-400">
        {t(strings.logo.maskBrushSize)}
        <input
          type="range"
          min={4}
          max={64}
          value={brush}
          onChange={(event) => setBrush(Number(event.target.value))}
        />
      </label>
      <div className="flex gap-2">
        <button
          type="button"
          disabled={disabled || !dirty}
          data-testid={selectors.logo.maskSubmit}
          className="rounded-control border border-cyan-400/40 px-2 py-1 text-xs text-cyan-200 disabled:opacity-50"
          onClick={submit}
        >
          {t(strings.logo.maskSubmit)}
        </button>
        <button
          type="button"
          className="rounded-control border border-white/10 px-2 py-1 text-xs text-slate-300"
          onClick={() => {
            fillBlack();
            setDirty(false);
          }}
        >
          {t(strings.logo.maskClear)}
        </button>
      </div>
    </div>
  );
}
