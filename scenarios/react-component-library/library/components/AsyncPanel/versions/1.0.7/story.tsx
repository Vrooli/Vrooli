import { AsyncPanel } from "./AsyncPanel";
import { createElement } from "react";
export function Ready({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(AsyncPanel, {
    ...args,
    children: createElement("p", {}, "Recent activity"),
  } as never);
}
