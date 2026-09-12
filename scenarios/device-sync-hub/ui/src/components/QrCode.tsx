import { useMemo } from "react";

import { encodeQr } from "../lib/qr";

interface QrCodeProps {
  /** Payload to encode (a pairing code). */
  value: string;
  /** Rendered pixel size of the square. */
  size?: number;
  className?: string;
  "data-testid"?: string;
  "aria-label"?: string;
}

/**
 * Dependency-free QR rendering: encodes `value` to a module matrix (see
 * `lib/qr.ts`) and draws it as a single SVG `<path>` so it stays crisp at any
 * size and adds no DOM weight per module. Falls back to nothing if encoding
 * fails (e.g. an over-long payload) — the caller always also shows the code as
 * text, so the QR is an affordance, never the only path.
 */
export function QrCode({ value, size = 192, className, ...rest }: QrCodeProps) {
  const matrix = useMemo(() => {
    try {
      return encodeQr(value);
    } catch {
      return null;
    }
  }, [value]);

  if (!matrix) return null;

  const count = matrix.length;
  const quiet = 2;
  const dim = count + quiet * 2;
  let d = "";
  for (let r = 0; r < count; r++) {
    for (let c = 0; c < count; c++) {
      if (matrix[r]?.[c]) {
        d += `M${c + quiet} ${r + quiet}h1v1h-1z`;
      }
    }
  }

  return (
    <svg
      role="img"
      width={size}
      height={size}
      viewBox={`0 0 ${dim} ${dim}`}
      shapeRendering="crispEdges"
      className={className}
      {...rest}
    >
      <rect width={dim} height={dim} fill="#ffffff" />
      <path d={d} fill="#000000" />
    </svg>
  );
}
