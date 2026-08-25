import { describe, expect, it } from "vitest";
import {
  ARROW_UP_BYTES,
  CSI,
  CSI_REVERSE_TAB,
  ESC,
  mouseWheelSequence,
} from "./terminalKeys";
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
});
