import { createClient } from "@connectrpc/connect";
import { ReviewService, type DailySummary, type WeeklySummary, type ReviewReflection } from "@vrooli/proto-types/personal-planner/v1/review/review_pb";

import { API_BASE, transport } from "./client";

export type Calibration = {
  period: string;
  avgErrorPercent: number;
  overRatio: number;
  underRatio: number;
  byCategory: Record<string, number>;
  accuracyTrend: number[];
  sampleSize: number;
  updatedAt: string;
};

export type GoalVariance = { goalId: string; targetDate: string; completedDate: string; deltaDays: number; label: string };
export type TodaySignals = { overdueCount: number; overdueMinutes: number; momentumDays: number; label: string };
export type GoalDrift = { goalId: string; targetDate: string; expectedBasis: number; actualBasis: number; driftBasis: number; label: string };
export type Reminder = { id: string; kind: "upcoming" | "overdue" | "goal"; title: string; body: string; startDate: string; startMin: number };
export type ReminderPreferences = { enabled: boolean; quietStartMinutes: number; quietEndMinutes: number; leadMinutes: number; updatedAt: string };
export type AgentReadModel = { estimationBias: Calibration; generatedAt: string };
export type ReviewWin = { localDate: string; text: string; updatedAt: string };

const reviewClient = createClient(ReviewService, transport);

export async function fetchDailyReview(localDate = ""): Promise<DailySummary> {
  const response = await reviewClient.getDailySummary({ localDate });
  if (!response.summary) throw new Error("The server returned no daily review");
  return response.summary;
}

export async function fetchWeeklyReview(weekStartLocalDate = ""): Promise<WeeklySummary> {
  const response = await reviewClient.getWeeklySummary({ weekStartLocalDate });
  if (!response.summary) throw new Error("The server returned no weekly review");
  return response.summary;
}

export async function fetchReflection(localDate = ""): Promise<ReviewReflection> {
  const response = await reviewClient.getReflection({ localDate });
  if (!response.reflection) throw new Error("The server returned no reflection");
  return response.reflection;
}

export async function saveReflection(input: { localDate: string; text: string }): Promise<ReviewReflection> {
  const response = await reviewClient.saveReflection(input);
  if (!response.reflection) throw new Error("The server returned no saved reflection");
  return response.reflection;
}

export async function fetchCalibration(period = "all_time"): Promise<Calibration> {
  const response = await fetch(`${API_BASE}/api/v1/review/calibration?period=${encodeURIComponent(period)}`, { cache: "no-store" });
  if (!response.ok) throw new Error("Calibration is unavailable");
  const value = await response.json() as { period: string; avg_error_percent: number; over_ratio: number; under_ratio: number; by_category: Record<string, number>; accuracy_trend: number[]; sample_size: number; updated_at: string };
  return { period: value.period, avgErrorPercent: value.avg_error_percent, overRatio: value.over_ratio, underRatio: value.under_ratio, byCategory: value.by_category, accuracyTrend: value.accuracy_trend, sampleSize: value.sample_size, updatedAt: value.updated_at };
}

export async function fetchAgentReadModel(period = "all_time"): Promise<AgentReadModel> {
  const response = await fetch(`${API_BASE}/api/v1/agent/read-model?period=${encodeURIComponent(period)}`, { cache: "no-store" });
  if (!response.ok) throw new Error("Agent read model is unavailable");
  const value = await response.json() as { estimation_bias: { period: string; avg_error_percent: number; over_ratio: number; under_ratio: number; by_category: Record<string, number>; accuracy_trend: number[]; sample_size: number; updated_at: string }; generated_at: string };
  return { generatedAt: value.generated_at, estimationBias: { period: value.estimation_bias.period, avgErrorPercent: value.estimation_bias.avg_error_percent, overRatio: value.estimation_bias.over_ratio, underRatio: value.estimation_bias.under_ratio, byCategory: value.estimation_bias.by_category, accuracyTrend: value.estimation_bias.accuracy_trend, sampleSize: value.estimation_bias.sample_size, updatedAt: value.estimation_bias.updated_at } };
}

export async function fetchGoalVariances(): Promise<GoalVariance[]> {
  const response = await fetch(`${API_BASE}/api/v1/review/goal-variances`, { cache: "no-store" });
  if (!response.ok) throw new Error("Goal variance is unavailable");
  const values = await response.json() as Array<{ goal_id: string; target_date: string; completed_date: string; delta_days: number; label: string }>;
  return values.map((value) => ({ goalId: value.goal_id, targetDate: value.target_date, completedDate: value.completed_date, deltaDays: value.delta_days, label: value.label }));
}

export async function fetchTodaySignals(localDate = ""): Promise<TodaySignals> {
  const response = await fetch(`${API_BASE}/api/v1/review/today-signals?date=${encodeURIComponent(localDate)}`, { cache: "no-store" });
  if (!response.ok) throw new Error("Today signals are unavailable");
  const value = await response.json() as { overdue_count: number; overdue_minutes: number; momentum_days: number; label: string };
  return { overdueCount: value.overdue_count, overdueMinutes: value.overdue_minutes, momentumDays: value.momentum_days, label: value.label };
}

export async function fetchGoalDrifts(localDate = ""): Promise<GoalDrift[]> {
  const response = await fetch(`${API_BASE}/api/v1/review/goal-drifts?date=${encodeURIComponent(localDate)}`, { cache: "no-store" });
  if (!response.ok) throw new Error("Goal drift is unavailable");
  const values = await response.json() as Array<{ goal_id: string; target_date: string; expected_basis_points: number; actual_basis_points: number; drift_basis_points: number; label: string }>;
  return values.map((value) => ({ goalId: value.goal_id, targetDate: value.target_date, expectedBasis: value.expected_basis_points, actualBasis: value.actual_basis_points, driftBasis: value.drift_basis_points, label: value.label }));
}

export async function fetchReminders(localDate = ""): Promise<Reminder[]> {
  const response = await fetch(`${API_BASE}/api/v1/review/reminders?date=${encodeURIComponent(localDate)}`, { cache: "no-store" });
  if (!response.ok) throw new Error("Reminders are unavailable");
  const values = await response.json() as Array<{ id: string; kind: Reminder["kind"]; title: string; body: string; start_date: string; start_minutes: number }>;
  return values.map((value) => ({ id: value.id, kind: value.kind, title: value.title, body: value.body, startDate: value.start_date, startMin: value.start_minutes }));
}

export async function fetchReminderPreferences(): Promise<ReminderPreferences> {
  const response = await fetch(`${API_BASE}/api/v1/review/reminder-preferences`, { cache: "no-store" });
  if (!response.ok) throw new Error("Reminder preferences are unavailable");
  const value = await response.json() as { enabled: boolean; quiet_start_minutes: number; quiet_end_minutes: number; lead_minutes: number; updated_at: string };
  return { enabled: value.enabled, quietStartMinutes: value.quiet_start_minutes, quietEndMinutes: value.quiet_end_minutes, leadMinutes: value.lead_minutes, updatedAt: value.updated_at };
}

export async function saveReminderPreferences(input: Omit<ReminderPreferences, "updatedAt">): Promise<ReminderPreferences> {
  const response = await fetch(`${API_BASE}/api/v1/review/reminder-preferences`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ enabled: input.enabled, quiet_start_minutes: input.quietStartMinutes, quiet_end_minutes: input.quietEndMinutes, lead_minutes: input.leadMinutes }) });
  if (!response.ok) throw new Error("Reminder preferences could not be saved");
  const value = await response.json() as { enabled: boolean; quiet_start_minutes: number; quiet_end_minutes: number; lead_minutes: number; updated_at: string };
  return { enabled: value.enabled, quietStartMinutes: value.quiet_start_minutes, quietEndMinutes: value.quiet_end_minutes, leadMinutes: value.lead_minutes, updatedAt: value.updated_at };
}

export async function fetchWin(localDate = ""): Promise<ReviewWin> {
  const response = await fetch(`${API_BASE}/api/v1/review/win?date=${encodeURIComponent(localDate)}`, { cache: "no-store" });
  if (!response.ok) throw new Error("The daily win is unavailable");
  const value = await response.json() as { local_date?: string; text?: string; updated_at?: string; LocalDate?: string; Text?: string; UpdatedAt?: string };
  return { localDate: value.local_date ?? value.LocalDate ?? localDate, text: value.text ?? value.Text ?? "", updatedAt: value.updated_at ?? value.UpdatedAt ?? "" };
}

export async function saveWin(input: { localDate: string; text: string }): Promise<ReviewWin> {
  const response = await fetch(`${API_BASE}/api/v1/review/win`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ local_date: input.localDate, text: input.text }) });
  if (!response.ok) throw new Error("The daily win could not be saved");
  const value = await response.json() as { local_date?: string; text?: string; updated_at?: string; LocalDate?: string; Text?: string; UpdatedAt?: string };
  return { localDate: value.local_date ?? value.LocalDate ?? input.localDate, text: value.text ?? value.Text ?? input.text, updatedAt: value.updated_at ?? value.UpdatedAt ?? "" };
}
