import { CommandCenterShell } from "./CommandCenterShell";
import { createElement } from "react";
export function ReadyWorkspace({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(CommandCenterShell, {
    ...args,
    children: createElement("section", {}, "Recent runs"),
    controls: createElement("button", {}, "ready"),
    navigation: createElement("nav", {}, "Recent runs"),
  } as never);
}

export function PartialWorkspace({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(CommandCenterShell, {
    ...args,
    navigation: createElement("nav", {}, "Some information is unavailable."),
  } as never);
}

export function LoadingWorkspace({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(CommandCenterShell, {
    ...args,
    loading: createElement("p", {}, "loading"),
    navigation: createElement("nav", {}, "loading"),
  } as never);
}

export function EmptyWorkspace({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(CommandCenterShell, {
    ...args,
    empty: createElement("p", {}, "empty"),
    navigation: createElement("nav", {}, "empty"),
  } as never);
}

export function ErrorWorkspace({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(CommandCenterShell, {
    ...args,
    error: createElement("p", {}, "error"),
    navigation: createElement("nav", {}, "error"),
  } as never);
}
