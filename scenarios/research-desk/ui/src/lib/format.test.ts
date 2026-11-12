import { describe, expect, it } from "vitest";

import { kb, relative, toDate, tokens } from "./format";

describe("format", () => {
  const now = new Date("2026-11-11T12:00:00Z");

  it("says how long ago in the largest whole unit", () => {
    expect(relative(new Date("2026-11-11T11:59:30Z"), now)).toMatch(/30 seconds ago/);
    expect(relative(new Date("2026-11-11T09:00:00Z"), now)).toMatch(/3 hours ago/);
    expect(relative(new Date("2026-11-11T12:20:00Z"), now)).toMatch(/in 20 minutes/);
    expect(relative(new Date("2024-11-11T12:00:00Z"), now)).toMatch(/2 years ago/);
  });

  it("treats a missing or zero time as unknown, not as 1970", () => {
    expect(toDate(undefined)).toBeNull();
    expect(toDate({ seconds: 0n, nanos: 0 } as Parameters<typeof toDate>[0])).toBeNull();
  });

  it("states sizes as an agent reads them", () => {
    expect(kb(4200)).toBe("4.2 KB");
    expect(kb(36_000n)).toBe("36 KB");
    expect(tokens(36_000)).toBe("9,000");
  });
});
