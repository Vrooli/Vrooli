import { Button } from "./Button";
import { createElement } from "react";
import { Trash2 } from "lucide-react";
export function DangerIcon({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(Button, {
    ...args,
    icon: createElement(Trash2, { "aria-hidden": true }),
  } as never);
}
