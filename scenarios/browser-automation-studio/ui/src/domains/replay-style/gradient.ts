import type { CSSProperties } from 'react';
import type { ReplayGradientSpec } from './model';

export const DEFAULT_REPLAY_GRADIENT_SPEC: ReplayGradientSpec = {
  type: 'linear',
  angle: 135,
  stops: [
    { color: '#38bdf8', position: 0 },
    { color: '#818cf8', position: 100 },
  ],
};

export const gradientStopsToCss = (stops: ReplayGradientSpec['stops']): string =>
  stops
    .map((stop) => (typeof stop.position === 'number' ? `${stop.color} ${stop.position}%` : stop.color))
    .join(', ');

export const buildGradientCss = (spec: ReplayGradientSpec): string => {
  const angle = typeof spec.angle === 'number' ? spec.angle : 135;
  return `linear-gradient(${angle}deg, ${gradientStopsToCss(spec.stops)})`;
};

export const buildGradientPreviewStyle = (spec: ReplayGradientSpec): CSSProperties => ({
  backgroundImage: buildGradientCss(spec),
  backgroundColor: '#0b1120',
});
