import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ getDailySummary: vi.fn(), getWeeklySummary: vi.fn(), getReflection: vi.fn(), saveReflection: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { fetchDailyReview, fetchReflection, fetchWeeklyReview, saveReflection } from "./review";

afterEach(() => vi.clearAllMocks());

describe("review API transport", () => {
  it("requests the daily summary and rejects an empty response", async () => {
    const summary = { localDate: "2026-09-19", recordedActiveMinutes: 25n } as never;
    client.getDailySummary.mockResolvedValueOnce({ summary });
    expect(await fetchDailyReview("2026-09-19")).toBe(summary);
    expect(client.getDailySummary).toHaveBeenCalledWith({ localDate: "2026-09-19" });
    client.getDailySummary.mockResolvedValueOnce({});
    await expect(fetchDailyReview()).rejects.toThrow("no daily review");
  });

  it("requests the weekly summary and rejects an empty response", async () => {
    const summary = { weekStartLocalDate: "2026-09-14", plannedMinutes: 120n } as never;
    client.getWeeklySummary.mockResolvedValueOnce({ summary });
    expect(await fetchWeeklyReview("2026-09-14")).toBe(summary);
    expect(client.getWeeklySummary).toHaveBeenCalledWith({ weekStartLocalDate: "2026-09-14" });
    client.getWeeklySummary.mockResolvedValueOnce({});
    await expect(fetchWeeklyReview()).rejects.toThrow("no weekly review");
  });

  it("reads and saves the optional reflection", async () => {
    const reflectionText = "Protect the first hour.";
    const reflection = { localDate: "2026-09-19", text: reflectionText } as never;
    client.getReflection.mockResolvedValueOnce({ reflection });
    expect(await fetchReflection("2026-09-19")).toBe(reflection);
    expect(client.getReflection).toHaveBeenCalledWith({ localDate: "2026-09-19" });
    client.saveReflection.mockResolvedValueOnce({ reflection });
    expect(await saveReflection({ localDate: "2026-09-19", text: reflectionText })).toBe(reflection);
    expect(client.saveReflection).toHaveBeenCalledWith({ localDate: "2026-09-19", text: reflectionText });
  });
});
