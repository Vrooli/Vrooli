import { describe, expect, it } from "vitest";
import { formatFigure } from "./format";

describe("formatFigure", () => {
  it("groups integers and hides the count unit", () => {
    expect(formatFigure(1284, "integer", "count")).toEqual({ prefix: "", text: "1,284", suffix: "" });
  });
  it("compacts large figures and currencies", () => {
    expect(formatFigure(128400, "compact")).toEqual({ prefix: "", text: "128.4k", suffix: "" });
    expect(formatFigure(12400, "currency.compact", "usd")).toEqual({ prefix: "$", text: "12.4k", suffix: "" });
    expect(formatFigure(410, "currency", "usd")).toEqual({ prefix: "$", text: "410", suffix: "" });
    expect(formatFigure(2_400_000, "compact")).toEqual({ prefix: "", text: "2.4M", suffix: "" });
  });
  it("renders ratios as percentages, signed when asked", () => {
    expect(formatFigure(0.58, "percent")).toEqual({ prefix: "", text: "58", suffix: "%" });
    expect(formatFigure(0.0714, "percent.signed")).toEqual({ prefix: "+", text: "7.1", suffix: "%" });
    expect(formatFigure(-0.02, "percent.signed")).toEqual({ prefix: "−", text: "2", suffix: "%" });
  });
  it("keeps duration units beside the number", () => {
    expect(formatFigure(32.5379, "minutes")).toEqual({ prefix: "", text: "33", suffix: " min" });
    expect(formatFigure(14.3, "duration.days")).toEqual({ prefix: "", text: "14.3", suffix: " d" });
  });
});

describe("formatFigure defaults", () => {
  it("shows a non-count unit beside an unformatted figure and keeps one decimal for fractions", () => {
    expect(formatFigure(3.14159, undefined, "ratio")).toEqual({ prefix: "", text: "3.1", suffix: " ratio" });
    expect(formatFigure(7, undefined, "count")).toEqual({ prefix: "", text: "7", suffix: "" });
    expect(formatFigure(8.5, "minutes")).toEqual({ prefix: "", text: "8.5", suffix: " min" });
    expect(formatFigure(0.4, "percent")).toEqual({ prefix: "", text: "40", suffix: "%" });
    expect(formatFigure(0.05, "percent")).toEqual({ prefix: "", text: "5", suffix: "%" });
  });
});
