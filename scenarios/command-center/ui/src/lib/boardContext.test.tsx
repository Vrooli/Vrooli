import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { parseSamples, useBoardController } from "./boardContext";

describe("board context", () => {
  it("defaults the audience mode to mark, so a screenshot carries its own legend", () => {
    expect(parseSamples(null)).toBe("mark");
    expect(parseSamples("nonsense")).toBe("mark");
    expect(parseSamples("hide")).toBe("hide");
    expect(parseSamples("full")).toBe("full");
  });
  it("refuses to run outside the controller", () => {
    expect(() => renderHook(() => useBoardController())).toThrow(/inside BoardController/);
  });
});
