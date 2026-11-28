import { beforeEach, describe, expect, it } from "vitest";
import { i18n, setLocale } from "../../i18n";
import { describeDays, describeHours } from "./describeSchedule";

describe("describeDays", () => {
  beforeEach(async () => { await setLocale("en"); });
  const say = (days: Parameters<typeof describeDays>[0], lang = "en") => describeDays(days, lang, i18n.t.bind(i18n));

  it("names the common shapes", () => {
    expect(say([0, 1, 2, 3, 4, 5, 6])).toBe("Every day");
    expect(say([5, 4, 3, 2, 1])).toBe("Weekdays");
    expect(say([0, 6])).toBe("Weekends");
  });

  it("collapses runs of three or more days into a range, in Monday-first order", () => {
    expect(say([1, 2, 3])).toBe("Mon–Wed");
    expect(say([1, 2, 3, 5])).toBe("Mon–Wed and Fri");
    expect(say([5, 6, 0])).toBe("Fri–Sun");
  });

  it("lists one or two days by name", () => {
    expect(say([1])).toBe("Mon");
    expect(say([1, 3])).toBe("Mon and Wed");
  });

  it("reads days in the reader's language", () => {
    expect(say([1, 3], "ja")).toBe("月、水");
  });
});

describe("describeHours", () => {
  it("writes both ends as the reader's clock shows them", () => {
    expect(describeHours({ start: "09:00", end: "17:30" }, "en")).toBe("9:00 AM–5:30 PM");
    expect(describeHours({ start: "09:00", end: "17:30" }, "ja")).toBe("9:00–17:30");
  });
});
