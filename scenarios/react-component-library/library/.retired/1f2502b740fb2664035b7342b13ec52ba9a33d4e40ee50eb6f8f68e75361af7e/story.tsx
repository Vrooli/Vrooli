import { DrawerShell } from "./DrawerShell";
import { createElement } from "react";
export function FullOpen({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(DrawerShell, {
    ...args,
    onClose: (...eventArgs: unknown[]) => log("close", ...eventArgs),
  } as never);
}

export function CompactOpen({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(DrawerShell, {
    ...args,
    onClose: (...eventArgs: unknown[]) => log("close", ...eventArgs),
  } as never);
}

export function WithHeaderActions({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(DrawerShell, {
    ...args,
    headerActions: createElement("button", {}, "Save"),
    onClose: (...eventArgs: unknown[]) => log("close", ...eventArgs),
  } as never);
}

export function WithHeaderExtra({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(DrawerShell, {
    ...args,
    headerExtra: createElement("p", {}, "Updated 2 minutes ago"),
    onClose: (...eventArgs: unknown[]) => log("close", ...eventArgs),
  } as never);
}

export function KeyboardAware({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(DrawerShell, {
    ...args,
    onClose: (...eventArgs: unknown[]) => log("close", ...eventArgs),
  } as never);
}
