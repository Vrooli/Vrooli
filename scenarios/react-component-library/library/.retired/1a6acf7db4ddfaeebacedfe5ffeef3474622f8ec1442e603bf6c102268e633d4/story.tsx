import { ButtonGroup } from "./ButtonGroup";
import { createElement } from "react";
// ButtonGroup 1.1.1 reuses the verified 1.0.0 specimen contract.
export { ButtonGroupStory } from "../1.0.0/story";

export function Default({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(ButtonGroup, {
    ...args,
    children: [
      createElement("button", { key: "actions" }, "Actions"),
      createElement("button", { key: "save" }, "Save"),
    ],
  } as never);
}
