import { EmptyState } from "./EmptyState";
import { createElement } from "react";
import { GitBranch, PackageSearch } from "lucide-react";
export function NoComponents({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(EmptyState, {
    ...args,
    icon: createElement(PackageSearch, { "aria-hidden": true }),
  } as never);
}

export function WithAction({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(EmptyState, {
    ...args,
    icon: createElement(GitBranch, { "aria-hidden": true }),
  } as never);
}
