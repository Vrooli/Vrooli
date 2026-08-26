import { ResizableSidebar } from "./ResizableSidebar";
import { createElement } from "react";
export function Default({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(ResizableSidebar, {
    ...args,
    children: createElement("p", {}, "Filters"),
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
  return createElement(ResizableSidebar, {
    ...args,
    children: createElement("p", {}, "Content"),
  } as never);
}
