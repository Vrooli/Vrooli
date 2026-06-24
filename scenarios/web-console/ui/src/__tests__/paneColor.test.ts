import { describe, it, expect } from "vitest";
import {
  isHexColor,
  parsePaneColor,
  serializePaneColor,
  paneColorStyle,
  nextGroupColor,
} from "../lib/paneColor";
import { HEADER_COLORS } from "../consts/config";

describe("parsePaneColor", () => {
  it("treats transparent / empty / nullish as transparent", () => {
    for (const value of ["transparent", "", null, undefined]) {
      expect(parsePaneColor(value)).toEqual({ colors: [], isTransparent: true });
    }
  });

  it("parses a legacy single hex color", () => {
    expect(parsePaneColor("#ff6b6b")).toEqual({ colors: ["#ff6b6b"], isTransparent: false });
  });

  it("parses a two-color stripe", () => {
    expect(parsePaneColor("#ff6b6b|#4dabf7")).toEqual({
      colors: ["#ff6b6b", "#4dabf7"],
      isTransparent: false,
    });
  });

  it("caps at two colors", () => {
    expect(parsePaneColor("#111111|#222222|#333333").colors).toEqual([
      "#111111",
      "#222222",
    ]);
  });

  it("drops malformed parts and keeps valid ones", () => {
    expect(parsePaneColor("notacolor|#4dabf7")).toEqual({
      colors: ["#4dabf7"],
      isTransparent: false,
    });
  });

  it("degrades fully-malformed input to transparent", () => {
    expect(parsePaneColor("garbage")).toEqual({ colors: [], isTransparent: true });
  });
});

describe("serializePaneColor", () => {
  it("round-trips a single color", () => {
    expect(serializePaneColor(["#ff6b6b"])).toBe("#ff6b6b");
  });

  it("round-trips two colors with the | delimiter", () => {
    expect(serializePaneColor(["#ff6b6b", "#4dabf7"])).toBe("#ff6b6b|#4dabf7");
  });

  it("returns transparent for an empty/invalid set", () => {
    expect(serializePaneColor([])).toBe("transparent");
    expect(serializePaneColor(["nope"])).toBe("transparent");
  });

  it("caps at two colors", () => {
    expect(serializePaneColor(["#111111", "#222222", "#333333"])).toBe(
      "#111111|#222222",
    );
  });

  it("is the inverse of parse for legacy single hex and transparent", () => {
    for (const value of ["#ff6b6b", "transparent"]) {
      const parsed = parsePaneColor(value);
      expect(serializePaneColor(parsed.colors)).toBe(value);
    }
  });
});

describe("paneColorStyle", () => {
  it("returns undefined for transparent/malformed", () => {
    expect(paneColorStyle("transparent", "bar")).toBeUndefined();
    expect(paneColorStyle("garbage", "header")).toBeUndefined();
  });

  it("returns a solid backgroundColor for a single color", () => {
    expect(paneColorStyle("#ff6b6b", "bar")).toEqual({ backgroundColor: "#ff6b6b" });
    expect(paneColorStyle("#ff6b6b", "header")).toEqual({ backgroundColor: "#ff6b6b" });
  });

  it("returns stacked bands for the bar variant with two colors", () => {
    const style = paneColorStyle("#ff6b6b|#4dabf7", "bar");
    expect(style?.backgroundImage).toContain("linear-gradient(180deg");
    expect(style?.backgroundImage).toContain("#ff6b6b 0 50%");
    expect(style?.backgroundImage).toContain("#4dabf7 50% 100%");
  });

  it("returns a candy-cane for the header variant with two colors", () => {
    const style = paneColorStyle("#ff6b6b|#4dabf7", "header");
    expect(style?.backgroundImage).toContain("repeating-linear-gradient(45deg");
    expect(style?.backgroundImage).toContain("#ff6b6b 0 10px");
    expect(style?.backgroundImage).toContain("#4dabf7 10px 20px");
  });
});

describe("nextGroupColor", () => {
  it("returns the first palette color when none are used", () => {
    expect(nextGroupColor([])).toBe(HEADER_COLORS[0]);
  });

  it("avoids colors already in use", () => {
    const used = [HEADER_COLORS[0], HEADER_COLORS[1]];
    expect(nextGroupColor(used)).toBe(HEADER_COLORS[2]);
  });

  it("wraps by count when every palette color is used", () => {
    const allUsed = [...HEADER_COLORS];
    // count === length → index 0
    expect(nextGroupColor(allUsed)).toBe(HEADER_COLORS[0]);
    // one extra beyond the full palette → index 1
    expect(nextGroupColor([...allUsed, "#000000"])).toBe(HEADER_COLORS[1]);
  });
});

describe("isHexColor", () => {
  it("accepts 3- and 6-digit hex", () => {
    expect(isHexColor("#fff")).toBe(true);
    expect(isHexColor("#ff6b6b")).toBe(true);
  });
  it("rejects non-hex", () => {
    expect(isHexColor("transparent")).toBe(false);
    expect(isHexColor("ff6b6b")).toBe(false);
    expect(isHexColor("#ff6b6b|#4dabf7")).toBe(false);
  });
});
