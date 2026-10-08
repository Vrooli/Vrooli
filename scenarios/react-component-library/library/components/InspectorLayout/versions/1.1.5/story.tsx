import { InspectorLayout } from "./InspectorLayout";
import { createElement } from "react";
export function ReadyInspector({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(InspectorLayout, {
    ...args,
    canvas: createElement("p", {}, "Workflow canvas"),
    inspector: createElement("p", {}, "ready"),
    toolbar: createElement("button", {}, "Workflow canvas"),
  } as never);
}

export function EmptyInspector({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(InspectorLayout, {
    ...args,
    canvas: createElement("p", {}, "Nothing to show yet."),
  } as never);
}

export function LoadingInspector({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(InspectorLayout, {
    ...args,
    canvas: createElement("p", {}, "loading"),
    inspector: createElement("p", {}, "loading"),
  } as never);
}

export function PartialInspector({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(InspectorLayout, {
    ...args,
    canvas: createElement("p", {}, "partial"),
    inspector: createElement("p", {}, "partial"),
  } as never);
}

export function ErrorInspector({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(InspectorLayout, {
    ...args,
    canvas: createElement("p", {}, "error"),
    inspector: createElement("p", {}, "error"),
  } as never);
}
