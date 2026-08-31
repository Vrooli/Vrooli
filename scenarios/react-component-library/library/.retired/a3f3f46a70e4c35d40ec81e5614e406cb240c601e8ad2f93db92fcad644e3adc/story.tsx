import { AppShell } from "./AppShell";
import { createElement } from "react";
export function Default({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(AppShell, {
    ...args,
    children: createElement("section", {}, "Workspace content"),
    navigation: createElement("nav", {}, "Application workspace"),
  } as never);
}

export function LongContent({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(AppShell, {
    ...args,
    children: createElement("section", {}, "Application workspace"),
    navigation: createElement("nav", {}, "app-shell-main"),
  } as never);
}
