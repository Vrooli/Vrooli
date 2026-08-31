import { AssetDetailShell } from "./AssetDetailShell";
import { createElement } from "react";
export function ReadyDetail({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(AssetDetailShell, {
    ...args,
    activity: createElement("p", {}, "Validated by 12 checks."),
    preview: createElement("div", {}, "ready"),
  } as never);
}

export function ActivityPartial({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(AssetDetailShell, {
    ...args,
    metadata: createElement("p", {}, "Some information is unavailable."),
    preview: createElement("div", {}, "Some information is unavailable."),
  } as never);
}

export function ActivityLoading({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(AssetDetailShell, {
    ...args,
    activityLoading: createElement("p", {}, "loading"),
    metadata: createElement("p", {}, "loading"),
    preview: createElement("div", {}, "loading"),
  } as never);
}

export function ActivityEmpty({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(AssetDetailShell, {
    ...args,
    activityEmpty: createElement("p", {}, "empty"),
    metadata: createElement("p", {}, "empty"),
    preview: createElement("div", {}, "empty"),
  } as never);
}

export function ActivityError({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(AssetDetailShell, {
    ...args,
    activityError: createElement("p", {}, "error"),
    metadata: createElement("p", {}, "error"),
    preview: createElement("div", {}, "error"),
  } as never);
}
