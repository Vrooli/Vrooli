import type { Color } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";

export type RGB = Pick<Color, "red" | "green" | "blue">;

export function colorHex(color?: RGB): string | undefined {
  if (!color) return undefined;
  return `#${[color.red, color.green, color.blue].map((channel) => Math.max(0, Math.min(255, channel)).toString(16).padStart(2, "0")).join("")}`.toUpperCase();
}

export function scaledChannels(color: RGB, brightnessPercent: number): RGB {
  const brightness = Math.max(0, Math.min(100, brightnessPercent)) / 100;
  return {
    red: Math.round(color.red * brightness),
    green: Math.round(color.green * brightness),
    blue: Math.round(color.blue * brightness),
  };
}

export function rgbLabel(color: RGB): string {
  return `${color.red}, ${color.green}, ${color.blue}`;
}
