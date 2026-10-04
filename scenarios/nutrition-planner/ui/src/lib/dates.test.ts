import { describe, expect, it } from "vitest";

import { localDateString } from "./dates";

describe("localDateString", () => {
  it("uses the browser-local calendar date instead of the UTC date", () => {
    const previousTimezone = process.env.TZ;
    process.env.TZ = "America/Los_Angeles";
    try {
      expect(localDateString(new Date("2026-10-04T06:30:00Z"))).toBe("2026-10-03");
    } finally {
      if (previousTimezone === undefined) delete process.env.TZ;
      else process.env.TZ = previousTimezone;
    }
  });
});
