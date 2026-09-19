import { createClient } from "@connectrpc/connect";
import { ReviewService, type DailySummary, type WeeklySummary, type ReviewReflection } from "@vrooli/proto-types/personal-planner/v1/review/review_pb";

import { transport } from "./client";

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
