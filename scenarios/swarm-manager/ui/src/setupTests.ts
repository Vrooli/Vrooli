import "@testing-library/jest-dom";
import React from "react";
import { vi } from "vitest";

const originalConsoleLog = console.log.bind(console);
console.log = (...args: unknown[]) => {
  if (typeof args[0] === "string" && args[0].startsWith("[api-base]")) {
    return;
  }
  originalConsoleLog(...args);
};

vi.mock("@monaco-editor/react", () => ({
  __esModule: true,
  default: ({
    value = "",
    onChange,
    "data-testid": testId,
  }: {
    value?: string;
    onChange?: (value?: string) => void;
    "data-testid"?: string;
  }) =>
    React.createElement("textarea", {
      "data-testid": testId ?? "monaco-editor",
      value,
      onChange: (event: React.ChangeEvent<HTMLTextAreaElement>) =>
        onChange?.(event.target.value),
    }),
  DiffEditor: ({
    original,
    modified,
    "data-testid": testId,
  }: {
    original?: string;
    modified?: string;
    "data-testid"?: string;
  }) =>
    React.createElement("div", {
      "data-testid": testId ?? "monaco-diff-editor",
      "data-original": original ?? "",
      "data-modified": modified ?? "",
    }),
}));
