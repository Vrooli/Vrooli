import { describe, expect, it, vi } from "vitest";
import {
  applyChanges,
  groupSchedules,
  planChanges,
  scheduleIssue,
  type Schedule,
  type StoredWindow,
  type WindowChange,
} from "./quietSchedule";

const row = (weekday: number, over: Partial<StoredWindow> = {}): StoredWindow => ({
  id: `w${weekday}`,
  weekday,
  start: "12:00",
  end: "13:00",
  timezone: "Europe/London",
  criticalOverride: false,
  ...over,
});

const schedule = (over: Partial<Schedule> = {}): Schedule => ({
  days: [1, 2, 3, 4, 5],
  start: "12:00",
  end: "13:00",
  timezone: "Europe/London",
  criticalOverride: false,
  ...over,
});

describe("groupSchedules", () => {
  it("joins days that share hours, zone and critical setting, in Monday-first order", () => {
    const groups = groupSchedules([row(0), row(5), row(1), row(2)]);
    expect(groups).toHaveLength(1);
    expect(groups[0]?.days).toEqual([1, 2, 5, 0]);
  });

  it("keeps days with different hours, zones or critical settings apart", () => {
    const groups = groupSchedules([
      row(1),
      row(2, { end: "14:00" }),
      row(3, { timezone: "UTC" }),
      row(4, { criticalOverride: true }),
    ]);
    expect(groups.map((group) => group.days)).toEqual([[1], [2], [3], [4]]);
  });

  it("orders schedules by the first day they cover", () => {
    const groups = groupSchedules([row(6, { start: "09:00" }), row(2)]);
    expect(groups.map((group) => group.days[0])).toEqual([2, 6]);
  });

  it("ignores rows with an impossible weekday instead of inventing a day", () => {
    expect(groupSchedules([row(9), row(-1), row(1.5)])).toEqual([]);
  });
});

describe("scheduleIssue", () => {
  it("accepts a same-day window in a known zone", () => {
    expect(scheduleIssue(schedule())).toBeNull();
  });

  it("accepts overnight hours, which the Hub reads as ending the next morning", () => {
    expect(scheduleIssue(schedule({ start: "22:00", end: "07:00" }))).toBeNull();
  });

  it.each([
    [{ days: [] }, "noDays"],
    [{ start: "" }, "badTime"],
    [{ end: "24:30" }, "badTime"],
    [{ start: "13:00", end: "13:00" }, "sameTime"],
    [{ timezone: "Mars/Olympus" }, "badZone"],
  ] as const)("names the first problem in %j", (over, issue) => {
    expect(scheduleIssue(schedule(over as Partial<Schedule>))).toBe(issue);
  });
});

describe("planChanges", () => {
  it("writes every day of a new schedule", () => {
    const changes = planChanges([], schedule({ days: [1, 2] }), []);
    expect(changes.map((change) => [change.kind, change.weekday])).toEqual([["set", 1], ["set", 2]]);
  });

  it("skips days the Hub already holds with the same settings, so a retry redoes only what is missing", () => {
    const changes = planChanges([row(1), row(2)], schedule({ days: [1, 2, 3] }), []);
    expect(changes.map((change) => change.weekday)).toEqual([3]);
  });

  it("rewrites days whose settings changed", () => {
    const changes = planChanges([row(1), row(2)], schedule({ days: [1, 2], end: "14:00" }), [1, 2]);
    expect(changes.map((change) => [change.kind, change.weekday])).toEqual([["set", 1], ["set", 2]]);
  });

  it("deletes days that were dropped from the schedule, by their stored id", () => {
    const changes = planChanges([row(1), row(2), row(3)], schedule({ days: [1] }), [1, 2, 3]);
    expect(changes).toEqual([
      { kind: "delete", weekday: 2, id: "w2" },
      { kind: "delete", weekday: 3, id: "w3" },
    ]);
  });

  it("does not delete a dropped day the Hub no longer holds", () => {
    expect(planChanges([row(1)], schedule({ days: [1] }), [1, 2])).toEqual([]);
  });

  it("never touches days that belong to another schedule", () => {
    const changes = planChanges([row(1), row(6, { start: "09:00" })], schedule({ days: [1], end: "14:00" }), [1]);
    expect(changes.map((change) => change.weekday)).toEqual([1]);
  });
});

describe("applyChanges", () => {
  const set = (weekday: 1 | 2 | 3): WindowChange => ({ kind: "set", weekday, settings: { start: "12:00", end: "13:00", timezone: "Europe/London", criticalOverride: false } });

  it("confirms every day the latest read shows as applied", async () => {
    const write = vi.fn().mockResolvedValue(undefined);
    const outcome = await applyChanges([set(1), set(2)], { write, readLatest: () => Promise.resolve([row(1), row(2)]) });
    expect(outcome).toEqual({ applied: [1, 2], pending: [], readFailed: false });
    expect(write).toHaveBeenCalledTimes(2);
  });

  it("stops at the first failed write and leaves later days untouched", async () => {
    const write = vi.fn().mockResolvedValueOnce(undefined).mockRejectedValueOnce(new Error("offline"));
    const outcome = await applyChanges([set(1), set(2), set(3)], { write, readLatest: () => Promise.resolve([row(1)]) });
    expect(outcome).toEqual({ applied: [1], pending: [2, 3], readFailed: false });
    expect(write).toHaveBeenCalledTimes(2);
  });

  it("counts a day applied when its response was lost but the Hub holds it", async () => {
    const write = vi.fn().mockRejectedValue(new Error("response lost"));
    const outcome = await applyChanges([set(1)], { write, readLatest: () => Promise.resolve([row(1)]) });
    expect(outcome.applied).toEqual([1]);
  });

  it("reports an unreadable follow-up as unknown rather than as success", async () => {
    const outcome = await applyChanges([set(1)], { write: () => Promise.resolve(), readLatest: () => Promise.reject(new Error("offline")) });
    expect(outcome).toEqual({ applied: [], pending: [1], readFailed: true });
  });

  it("confirms a deletion only when the day is gone from the latest read", async () => {
    const remove: WindowChange = { kind: "delete", weekday: 1, id: "w1" };
    const gone = await applyChanges([remove], { write: () => Promise.resolve(), readLatest: () => Promise.resolve([]) });
    const stillThere = await applyChanges([remove], { write: () => Promise.resolve(), readLatest: () => Promise.resolve([row(1)]) });
    expect(gone.applied).toEqual([1]);
    expect(stillThere.pending).toEqual([1]);
  });
});
