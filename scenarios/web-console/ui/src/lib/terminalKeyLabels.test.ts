import { describe, expect, it } from "vitest";
import { decodeInputLabel } from "./terminalKeyLabels";

describe("decodeInputLabel", () => {
  it.each([
    ["hello", [{ kind: "text", label: "hello" }]],
    ["\x1b[A", [{ kind: "key", label: "Arrow Up" }]],
    ["\r", [{ kind: "key", label: "Enter" }]],
    ["\t", [{ kind: "key", label: "Tab" }]],
    ["\x7f", [{ kind: "key", label: "Backspace" }]],
    ["\x1b", [{ kind: "key", label: "Escape" }]],
    ["\x03", [{ kind: "key", label: "Ctrl+C" }]],
  ])("decodes %j", (data, expected) => {
    expect(decodeInputLabel(data)).toEqual(expected);
  });

  it("uses a short hex label for unknown escape sequences", () => {
    expect(decodeInputLabel("\x1b[99~")).toEqual([{ kind: "unknown", label: "Esc [99~" }]);
  });
});
