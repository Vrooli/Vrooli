import { describe, expect, it } from "vitest";

import { formatMinorMoney } from "./formatMoney";

describe("formatMinorMoney", () => {
  it("uses the currency's minor unit precision", () => {
    expect(formatMinorMoney(1250, "USD", "en-US", "minor units")).toBe("$12.50");
    expect(formatMinorMoney(1250, "JPY", "en-US", "minor units")).toBe("¥1,250");
    expect(formatMinorMoney(1250, "KWD", "en-US", "minor units")).toBe("KWD 1.250");
  });

  it("keeps an unknown currency amount explicit instead of guessing its exponent", () => {
    expect(formatMinorMoney(1250, "bad", "en-US", "minor units")).toBe("BAD 1250 minor units");
  });
});
