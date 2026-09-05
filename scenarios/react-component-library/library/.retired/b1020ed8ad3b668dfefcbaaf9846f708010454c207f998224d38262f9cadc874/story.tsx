import { Button } from "./Button";
import { createElement } from "react";
import { Save } from "lucide-react";
export function IconStart({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(Button, {
    ...args,
    children: [
      createElement(Save, { key: "save-icon", "aria-hidden": true }),
      "Save draft",
    ],
  } as never);
}
