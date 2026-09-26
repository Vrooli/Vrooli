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

export function WithHeaderContent({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(DrawerShell, {
    ...args,
    headerExtra: createElement("p", {}, "Updated just now"),
    onClose: (...eventArgs: unknown[]) => log("close", ...eventArgs),
  } as never);
}
