import { describe, expect, it } from "vitest";

import { scheduleAppliesOnLocalDate, type SupplementSchedule } from "./supplement";

const schedule: SupplementSchedule = {
  id: "s1", revision: 1n, productRevisionId: "product:p:1", dose: "2", doseUnit: "capsule",
  weekdays: [0], startDate: "2026-10-04", endDate: "2026-10-04", paused: false, confirmed: true, createdAt: "",
};

describe("scheduleAppliesOnLocalDate", () => {
  it("uses the supplied calendar date and inclusive schedule bounds", () => {
    expect(scheduleAppliesOnLocalDate(schedule, "2026-10-04")).toBe(true);
    expect(scheduleAppliesOnLocalDate(schedule, "2026-10-05")).toBe(false);
    expect(scheduleAppliesOnLocalDate(schedule, "2026-10-03")).toBe(false);
  });

  it("excludes paused and unconfirmed schedules", () => {
    expect(scheduleAppliesOnLocalDate({ ...schedule, paused: true }, "2026-10-04")).toBe(false);
    expect(scheduleAppliesOnLocalDate({ ...schedule, confirmed: false }, "2026-10-04")).toBe(false);
  });
});
