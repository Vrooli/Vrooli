import { Page } from "./Page";
import { createElement } from "react";
export function Default({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(Page, {
    ...args,
    children: createElement("p", {}, "Page content"),
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
  return createElement(Page, {
    ...args,
    children: createElement("p", {}, "Content"),
  } as never);
}
