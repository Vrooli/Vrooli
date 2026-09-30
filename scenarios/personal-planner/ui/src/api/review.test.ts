import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ getDailySummary: vi.fn(), getWeeklySummary: vi.fn(), getReflection: vi.fn(), saveReflection: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { fetchAgentReadModel, fetchCalibration, fetchDailyReview, fetchGoalDrifts, fetchGoalVariances, fetchReminders, fetchReminderPreferences, fetchTodaySignals, fetchReflection, fetchWeeklyReview, fetchWin, saveReflection, saveReminderPreferences, saveWin } from "./review";

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
    client.getReflection.mockResolvedValueOnce({});
    await expect(fetchReflection()).rejects.toThrow("no reflection");
    client.saveReflection.mockResolvedValueOnce({});
    await expect(saveReflection({ localDate: "2026-09-19", text: reflectionText })).rejects.toThrow("no saved reflection");
  });

  it("reads the stored calibration projection", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({ period: "all_time", avg_error_percent: 25, over_ratio: .5, under_ratio: .5, by_category: {}, accuracy_trend: [50, 0], sample_size: 2, updated_at: "2026-09-19T12:00:00Z" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(fetchCalibration()).resolves.toMatchObject({ period: "all_time", avgErrorPercent: 25, sampleSize: 2, accuracyTrend: [50, 0] });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("surfaces an unavailable calibration response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("unavailable", { status: 503 })));
    await expect(fetchCalibration("last_30d")).rejects.toThrow("Calibration is unavailable");
  });

  it("maps the explicit agent read model", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({ generated_at: "now", estimation_bias: { period: "all_time", avg_error_percent: 10, over_ratio: 1, under_ratio: 0, by_category: {}, accuracy_trend: [10], sample_size: 1, updated_at: "now" } }), { status: 200 })));
    await expect(fetchAgentReadModel()).resolves.toMatchObject({ generatedAt: "now", estimationBias: { avgErrorPercent: 10, sampleSize: 1 } });
  });

  it("reads stored goal variance labels", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(JSON.stringify([{ goal_id: "goal-1", target_date: "2026-09-20", completed_date: "2026-09-18", delta_days: -2, label: "2 days early" }]), { status: 200 })));
    await expect(fetchGoalVariances()).resolves.toEqual([{ goalId: "goal-1", targetDate: "2026-09-20", completedDate: "2026-09-18", deltaDays: -2, label: "2 days early" }]);
  });

  it("reads overdue and momentum signals from the server", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({ overdue_count: 1, overdue_minutes: 45, momentum_days: 2, label: "1 overdue" }), { status: 200 })));
    await expect(fetchTodaySignals("2026-09-19")).resolves.toEqual({ overdueCount: 1, overdueMinutes: 45, momentumDays: 2, label: "1 overdue" });
  });

  it("reads goal drift nudges", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(JSON.stringify([{ goal_id: "goal-1", target_date: "2026-10-01", expected_basis_points: 5000, actual_basis_points: 1000, drift_basis_points: -4000, label: "Behind pace" }]), { status: 200 })));
    await expect(fetchGoalDrifts("2026-09-16")).resolves.toEqual([{ goalId: "goal-1", targetDate: "2026-10-01", expectedBasis: 5000, actualBasis: 1000, driftBasis: -4000, label: "Behind pace" }]);
  });

  it("maps actionable in-app reminders", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(JSON.stringify([{ id: "upcoming-a", kind: "upcoming", title: "Write", body: "Starts in 20 minutes", start_date: "2026-09-21", start_minutes: 600 }]), { status: 200 })));
    await expect(fetchReminders("2026-09-21")).resolves.toEqual([{ id: "upcoming-a", kind: "upcoming", title: "Write", body: "Starts in 20 minutes", startDate: "2026-09-21", startMin: 600 }]);
  });

  it("maps durable reminder preferences", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ enabled: true, quiet_start_minutes: 1320, quiet_end_minutes: 420, lead_minutes: 45, updated_at: "now" }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ enabled: false, quiet_start_minutes: 1200, quiet_end_minutes: 480, lead_minutes: 30, updated_at: "later" }), { status: 200 })));
    await expect(fetchReminderPreferences()).resolves.toEqual({ enabled: true, quietStartMinutes: 1320, quietEndMinutes: 420, leadMinutes: 45, updatedAt: "now" });
    await expect(saveReminderPreferences({ enabled: false, quietStartMinutes: 1200, quietEndMinutes: 480, leadMinutes: 30 })).resolves.toEqual({ enabled: false, quietStartMinutes: 1200, quietEndMinutes: 480, leadMinutes: 30, updatedAt: "later" });
    expect(fetch).toHaveBeenLastCalledWith(expect.stringContaining("reminder-preferences"), expect.objectContaining({ method: "PUT" }));
  });

  it("maps the durable daily win", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ local_date: "2026-09-21", text: "Kept the first hour clear.", updated_at: "now" }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ local_date: "2026-09-21", text: "Shipped the small thing.", updated_at: "later" }), { status: 200 })));
    await expect(fetchWin("2026-09-21")).resolves.toEqual({ localDate: "2026-09-21", text: "Kept the first hour clear.", updatedAt: "now" });
    await expect(saveWin({ localDate: "2026-09-21", text: "Shipped the small thing." })).resolves.toEqual({ localDate: "2026-09-21", text: "Shipped the small thing.", updatedAt: "later" });
  });

  it("accepts the legacy capitalized daily-win response shape", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ LocalDate: "2026-09-21", Text: "A durable signal.", UpdatedAt: "now" }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ LocalDate: "2026-09-21", Text: "Saved signal.", UpdatedAt: "later" }), { status: 200 })));
    await expect(fetchWin("2026-09-21")).resolves.toEqual({ localDate: "2026-09-21", text: "A durable signal.", updatedAt: "now" });
    await expect(saveWin({ localDate: "2026-09-21", text: "Saved signal." })).resolves.toEqual({ localDate: "2026-09-21", text: "Saved signal.", updatedAt: "later" });
  });
});
