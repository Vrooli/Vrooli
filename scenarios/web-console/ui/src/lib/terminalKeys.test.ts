import { describe, expect, it } from "vitest";
import {
  ARROW_UP_BYTES,
  CSI,
  CSI_REVERSE_TAB,
  ESC,
  NAMED_KEY_BYTES,
  isMouseTrackingSequence,
  mouseWheelSequence,
  namedKeySequence,
} from "./terminalKeys";
import sharedKeyMap from "../../../shared/terminal-keymap.json";
import { applyModifiers } from "../consts/toolbar-keys";

describe("terminal key encoding", () => {
  it("encodes modified tab and arrows", () => {
    expect(applyModifiers("\t", { ctrl: false, alt: false, shift: true })).toEqual({ data: CSI_REVERSE_TAB, consumed: true });
    expect(applyModifiers("\t", { ctrl: false, alt: true, shift: true })).toEqual({ data: ESC + CSI_REVERSE_TAB, consumed: true });
    expect(applyModifiers(ARROW_UP_BYTES, { ctrl: true, alt: true, shift: true })).toEqual({ data: `${CSI}1;8A`, consumed: true });
    expect(applyModifiers("x", { ctrl: false, alt: false, shift: false })).toEqual({ data: "x", consumed: false });
  });

  it("encodes control and alt bytes and wheel coordinates", () => {
    expect(applyModifiers("c", { ctrl: true, alt: false, shift: false })).toEqual({ data: "\x03", consumed: true });
    expect(applyModifiers("x", { ctrl: false, alt: true, shift: true })).toEqual({ data: "\x1bX", consumed: true });
    expect(applyModifiers("long", { ctrl: true, alt: false, shift: false })).toEqual({ data: "long", consumed: true });
    expect(mouseWheelSequence(true, 0, 1)).toBe(`${CSI}<64;1;2M`);
    expect(mouseWheelSequence(false, 2, 3)).toBe(`${CSI}<65;3;4M`);
  });

  it("recognizes SGR and X10 mouse reports without matching other CSI input", () => {
    expect(isMouseTrackingSequence("\x1b[<64;1;2M")).toBe(true);
    expect(isMouseTrackingSequence("\x1b[<65;3;4m")).toBe(true);
    expect(isMouseTrackingSequence("\x1b[Mabc")).toBe(true);
    expect(isMouseTrackingSequence("\x1b[A")).toBe(false);
    expect(isMouseTrackingSequence("\x1b[11~")).toBe(false);
    expect(isMouseTrackingSequence("\x1b[?25l")).toBe(false);
  });

  it("uses the shared named-key map and applies the alt prefix", () => {
    expect(NAMED_KEY_BYTES.get("f12")).toBe("\x1b[24~");
    expect(namedKeySequence(" F12 ", { alt: true })).toBe("\x1b\x1b[24~");
    expect(namedKeySequence("unknown")).toBeUndefined();
  });

  it("covers every entry in the shared cross-language key table", () => {
    for (const entry of sharedKeyMap) {
      expect(Array.from(NAMED_KEY_BYTES.get(entry.name) ?? "", (char) => char.charCodeAt(0))).toEqual(entry.bytes);
    }
  });
});
