import { useMemo } from 'react';
import { encodeQr } from '../lib/qrcode';

interface QrCodeProps {
  value: string;
  /** Rendered edge length in CSS pixels. */
  size?: number;
  label: string;
  className?: string;
}

/** Crisp SVG QR code with the quiet zone scanners need. */
export function QrCode({ value, size = 200, label, className }: QrCodeProps) {
  const path = useMemo(() => {
    const { size: modules, modules: grid } = encodeQr(value);
    let d = '';
    grid.forEach((row, y) => {
      row.forEach((dark, x) => {
        if (dark) d += `M${String(x + 4)} ${String(y + 4)}h1v1h-1z`;
      });
    });
    return { d, dimension: modules + 8 };
  }, [value]);
  // Whole device pixels per module keep module edges sharp; blurred edges are
  // what makes on-screen QR codes fail to scan.
  const rendered = Math.max(1, Math.round(size / path.dimension)) * path.dimension;
  return (
    <svg
      className={className}
      width={rendered}
      height={rendered}
      viewBox={`0 0 ${String(path.dimension)} ${String(path.dimension)}`}
      role="img"
      aria-label={label}
      shapeRendering="crispEdges"
    >
      <rect width={path.dimension} height={path.dimension} fill="#ffffff" />
      <path d={path.d} fill="#0b1728" />
    </svg>
  );
}
