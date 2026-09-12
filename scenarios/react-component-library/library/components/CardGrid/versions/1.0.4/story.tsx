import { CardGrid } from "./CardGrid";
import { createElement } from "react";
export function RaisedRecords({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(CardGrid, {
    ...args,
    children: [
      createElement("article", { key: "record-1" }, "Raised record"),
      createElement("article", { key: "record-2" }, "Raised record"),
    ],
  } as never);
}
