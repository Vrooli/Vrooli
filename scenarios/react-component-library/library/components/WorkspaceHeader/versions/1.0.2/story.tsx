import { WorkspaceHeader } from "./WorkspaceHeader";
import { createElement } from "react";
export function Basic({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(WorkspaceHeader, {
    ...args,
    primaryAction: createElement("button", {}, "Component Library"),
  } as never);
}
