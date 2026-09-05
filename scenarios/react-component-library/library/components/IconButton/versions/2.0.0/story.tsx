import { IconButton } from "./IconButton";
import { createElement, type ElementType } from "react";
import { Circle, Play, Settings, Trash2, X } from "lucide-react";
function iconButton(args: Record<string, unknown>, Icon: ElementType, label: string) {
  return createElement(IconButton, {
    ...args,
    "aria-label": label,
    children: createElement(Icon, { "aria-hidden": true }),
  } as never);
}
export function VariantPrimary({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Play, "Play");
}

export function VariantSecondary({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Settings, "Settings");
}

export function VariantGhost({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, X, "Close");
}

export function VariantDanger({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Trash2, "Delete");
}

export function SizeXs({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Circle, "XS");
}

export function SizeSm({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Circle, "SM");
}

export function SizeMd({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Circle, "MD");
}

export function SizeLg({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Circle, "LG");
}

export function SizeXl({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Circle, "XL");
}

export function SizeIcon({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Circle, "Icon");
}

export function DensityComfortable({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Circle, "Comfortable");
}

export function DensityCompact({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return iconButton(args, Circle, "Compact");
}
