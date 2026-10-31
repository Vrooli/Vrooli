import { createClient } from "@connectrpc/connect";
import { CookingService, type CookingSession, type Timer } from "@vrooli/proto-types/nutrition-planner/v1/cooking/cooking_pb";
import { transport } from "./client";

const client = createClient(CookingService, transport);
export type { CookingSession, Timer };

export async function startCookingSession(input: { workspaceId: string; sessionId: string; recipeId: string; recipeRevision: bigint; methodId: string; scale: string }): Promise<CookingSession> {
  const response = await client.startSession(input);
  if (!response.session) throw new Error("The API returned no cooking session.");
  return response.session;
}

export async function getCookingSession(workspaceId: string, sessionId: string): Promise<CookingSession> {
  const response = await client.getSession({ workspaceId, sessionId });
  if (!response.session) throw new Error("The API returned no cooking session.");
  return response.session;
}

export async function listCookingSessions(workspaceId: string): Promise<CookingSession[]> {
  const response = await client.listSessions({ workspaceId });
  return response.sessions;
}

export async function saveCookingSession(input: { workspaceId: string; sessionId: string; eventId: string; expectedVersion: bigint; currentStepIndex: number; completedSteps: string[]; timers: Timer[]; finish?: boolean; actualYield?: string; yieldUnit?: string }): Promise<CookingSession> {
  const response = await client.saveSession({ ...input, finish: input.finish ?? false, actualYield: input.actualYield ?? "", yieldUnit: input.yieldUnit ?? "" });
  if (!response.session) throw new Error("The API returned no updated cooking session.");
  return response.session;
}
