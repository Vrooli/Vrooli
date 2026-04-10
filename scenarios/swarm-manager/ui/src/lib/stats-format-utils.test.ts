import { describe, it, expect } from "vitest";
import { formatDelta, formatHours, formatRate, formatWeeksRemaining, toBarPercent } from "./stats-format-utils";

describe("formatHours", () => {
  it("returns '< 1 min' for zero", () => {
    expect(formatHours(0)).toBe("< 1 min");
  });

  it("returns '< 1 min' for negative values", () => {
    expect(formatHours(-1)).toBe("< 1 min");
  });

  it("returns '< 1 min' for very small values", () => {
    expect(formatHours(0.005)).toBe("< 1 min");
  });

  it("formats minutes for sub-hour values", () => {
    expect(formatHours(0.5)).toBe("30 min");
  });

  it("rounds minutes to nearest integer", () => {
    expect(formatHours(0.25)).toBe("15 min");
  });

  it("formats hours with one decimal", () => {
    expect(formatHours(2.5)).toBe("2.5 hrs");
  });

  it("formats large hour values", () => {
    expect(formatHours(48)).toBe("48.0 hrs");
  });

  it("formats exactly one hour", () => {
    expect(formatHours(1)).toBe("1.0 hrs");
  });
});

describe("formatRate", () => {
  it("returns '0%' for zero", () => {
    expect(formatRate(0)).toBe("0%");
  });

  it("returns '100%' for 1.0", () => {
    expect(formatRate(1)).toBe("100%");
  });

  it("formats mid-range rates with one decimal", () => {
    expect(formatRate(0.852)).toBe("85.2%");
  });

  it("respects custom decimal count", () => {
    expect(formatRate(0.8527, 2)).toBe("85.27%");
  });

  it("formats small rates", () => {
    expect(formatRate(0.01)).toBe("1.0%");
  });
});

describe("formatDelta", () => {
  it("prefixes positive values with +", () => {
    expect(formatDelta(12)).toBe("+12");
  });

  it("shows negative values with -", () => {
    expect(formatDelta(-3)).toBe("-3");
  });

  it("shows zero without prefix", () => {
    expect(formatDelta(0)).toBe("0");
  });
});

describe("formatWeeksRemaining", () => {
  it("returns 'Done' for zero", () => {
    expect(formatWeeksRemaining(0)).toBe("Done");
  });

  it("returns 'Done' for negative values", () => {
    expect(formatWeeksRemaining(-1)).toBe("Done");
  });

  it("returns '< 1 week' for fractional weeks", () => {
    expect(formatWeeksRemaining(0.3)).toBe("< 1 week");
  });

  it("formats weeks with tilde and one decimal", () => {
    expect(formatWeeksRemaining(8.6)).toBe("~8.6 weeks");
  });

  it("formats exactly one week", () => {
    expect(formatWeeksRemaining(1)).toBe("~1.0 weeks");
  });
});

describe("toBarPercent", () => {
  it("returns 0 when max is 0", () => {
    expect(toBarPercent(5, 0)).toBe(0);
  });

  it("returns 0 when max is negative", () => {
    expect(toBarPercent(5, -1)).toBe(0);
  });

  it("computes correct percentage", () => {
    expect(toBarPercent(3, 10)).toBe(30);
  });

  it("clamps to 100 when value exceeds max", () => {
    expect(toBarPercent(15, 10)).toBe(100);
  });

  it("clamps to 0 for negative values", () => {
    expect(toBarPercent(-5, 10)).toBe(0);
  });

  it("returns 100 for equal value and max", () => {
    expect(toBarPercent(10, 10)).toBe(100);
  });

  it("handles zero value correctly", () => {
    expect(toBarPercent(0, 10)).toBe(0);
  });
});
