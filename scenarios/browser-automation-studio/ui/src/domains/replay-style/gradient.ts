import type { CSSProperties } from 'react';
import type { ReplayGradientSpec, ReplayGradientStop } from './model';

export const DEFAULT_REPLAY_GRADIENT_SPEC: ReplayGradientSpec = {
  type: 'linear',
  angle: 135,
  stops: [
    { color: '#38bdf8', position: 0 },
    { color: '#818cf8', position: 100 },
  ],
};

const clampPercent = (value: number): number => Math.min(100, Math.max(0, value));

export const gradientStopsToCss = (stops: ReplayGradientSpec['stops']): string =>
  stops
    .map((stop) => (typeof stop.position === 'number' ? `${stop.color} ${stop.position}%` : stop.color))
    .join(', ');

export const buildGradientCss = (spec: ReplayGradientSpec): string => {
  const type = spec.type === 'radial' ? 'radial' : 'linear';
  if (type === 'radial') {
    const center = spec.center ?? { x: 50, y: 50 };
    return `radial-gradient(circle at ${clampPercent(center.x)}% ${clampPercent(center.y)}%, ${gradientStopsToCss(spec.stops)})`;
  }
  const angle = typeof spec.angle === 'number' ? spec.angle : 135;
  return `linear-gradient(${angle}deg, ${gradientStopsToCss(spec.stops)})`;
};

export const buildGradientPreviewStyle = (spec: ReplayGradientSpec): CSSProperties => ({
  backgroundImage: buildGradientCss(spec),
  backgroundColor: '#0b1120',
});

const splitGradientArgs = (value: string): string[] => {
  const parts: string[] = [];
  let current = '';
  let depth = 0;

  for (let i = 0; i < value.length; i += 1) {
    const char = value[i];
    if (char === '(') {
      depth += 1;
    } else if (char === ')') {
      depth = Math.max(0, depth - 1);
    } else if (char === ',' && depth === 0) {
      parts.push(current.trim());
      current = '';
      continue;
    }
    current += char;
  }
  if (current.trim()) {
    parts.push(current.trim());
  }
  return parts;
};

const parseStop = (raw: string): ReplayGradientStop | null => {
  const trimmed = raw.trim();
  if (!trimmed) {
    return null;
  }
  const match = trimmed.match(/^(.*)\s+(-?\d+(?:\.\d+)?)%$/);
  if (match) {
    const color = match[1].trim();
    const position = Number.parseFloat(match[2]);
    if (!color || !Number.isFinite(position)) {
      return null;
    }
    return { color, position: clampPercent(position) };
  }
  return { color: trimmed };
};

const isGradientConfigToken = (value: string): boolean =>
  /\bat\b/i.test(value) || /\bcircle\b|\bellipse\b|\bclosest\b|\bfarthest\b/i.test(value);

const parseLinearDirection = (value: string): number | null => {
  const normalized = value.trim().toLowerCase();
  if (!normalized.startsWith('to ')) {
    return null;
  }
  const direction = normalized.slice(3).trim();
  switch (direction) {
    case 'top':
      return 0;
    case 'top right':
    case 'right top':
      return 45;
    case 'right':
      return 90;
    case 'bottom right':
    case 'right bottom':
      return 135;
    case 'bottom':
      return 180;
    case 'bottom left':
    case 'left bottom':
      return 225;
    case 'left':
      return 270;
    case 'top left':
    case 'left top':
      return 315;
    default:
      return null;
  }
};

export const parseGradientCss = (value: string): ReplayGradientSpec | null => {
  const trimmed = value.trim();
  const match = trimmed.match(/^(linear-gradient|radial-gradient)\((.*)\)$/i);
  if (!match) {
    return null;
  }

  const type = match[1].toLowerCase().startsWith('radial') ? 'radial' : 'linear';
  const args = splitGradientArgs(match[2]);
  if (args.length < 2) {
    return null;
  }

  let angle: number | undefined;
  let center: { x: number; y: number } | undefined;
  let stopStartIndex = 0;

  if (type === 'linear') {
    const angleMatch = args[0].trim().match(/^(-?\d+(?:\.\d+)?)deg$/i);
    if (angleMatch) {
      angle = Number.parseFloat(angleMatch[1]);
      stopStartIndex = 1;
    } else {
      const directionAngle = parseLinearDirection(args[0]);
      if (typeof directionAngle === 'number') {
        angle = directionAngle;
        stopStartIndex = 1;
      }
    }
  } else {
    const firstArg = args[0].trim();
    if (isGradientConfigToken(firstArg)) {
      const centerMatch = firstArg.match(/at\s+(-?\d+(?:\.\d+)?)%\s+(-?\d+(?:\.\d+)?)%/i);
      if (centerMatch) {
        const x = Number.parseFloat(centerMatch[1]);
        const y = Number.parseFloat(centerMatch[2]);
        if (Number.isFinite(x) && Number.isFinite(y)) {
          center = { x: clampPercent(x), y: clampPercent(y) };
        }
      }
      stopStartIndex = 1;
    }
  }

  const stops = args.slice(stopStartIndex)
    .map(parseStop)
    .filter((stop): stop is ReplayGradientStop => Boolean(stop));

  if (stops.length < 2) {
    return null;
  }

  return {
    type,
    angle: type === 'linear' && typeof angle === 'number' ? angle : undefined,
    center: type === 'radial' ? center : undefined,
    stops,
  };
};

export const normalizeGradientStops = (
  stops: ReplayGradientStop[] | undefined,
  fallback: ReplayGradientStop[] = DEFAULT_REPLAY_GRADIENT_SPEC.stops,
  maxStops = 4,
): ReplayGradientStop[] => {
  const base = (stops ?? []).filter((stop) => stop && typeof stop.color === 'string' && stop.color.trim().length > 0);
  const source = (base.length >= 2 ? base : fallback).slice(0, maxStops);
  const count = source.length;
  return source.map((stop, index) => {
    const position = typeof stop.position === 'number'
      ? clampPercent(stop.position)
      : count > 1
        ? Math.round((index / (count - 1)) * 100)
        : index === 0 ? 0 : 100;
    return { color: stop.color, position };
  });
};
