import { createClient } from "@connectrpc/connect";
import { FocusService, type ActualCorrection, type FocusSession } from "@vrooli/proto-types/personal-planner/v1/focus/focus_pb";

import { transport } from "./client";

export const focusClient = createClient(FocusService, transport);

export type Actual = {
  id: string;
  workItemId: string;
  title: string;
  localDate: string;
  reportedMinutes: bigint;
  certainty: string;
  note: string;
  createdAtUnixSeconds: bigint;
  revision: bigint;
};

export async function fetchCurrentFocus(): Promise<FocusSession | null> {
  const response = await focusClient.getCurrentSession({});
  return response.hasSession ? response.session ?? null : null;
}

export async function startFocus(input: { workItemId?: string; title: string; mode?: string }): Promise<FocusSession> {
  const response = await focusClient.startFocus({ workItemId: input.workItemId ?? "", title: input.title, mode: input.mode ?? "open" });
  if (!response.session) throw new Error("Focus service returned no session");
  return response.session;
}

export async function pauseFocus(session: FocusSession): Promise<FocusSession> {
  const response = await focusClient.pauseFocus({ sessionId: session.id, expectedRevision: session.revision });
  if (!response.session) throw new Error("Focus service returned no session");
  return response.session;
}

export async function resumeFocus(session: FocusSession): Promise<FocusSession> {
  const response = await focusClient.resumeFocus({ sessionId: session.id, expectedRevision: session.revision });
  if (!response.session) throw new Error("Focus service returned no session");
  return response.session;
}

export async function endFocus(session: FocusSession): Promise<FocusSession> {
  const response = await focusClient.endFocus({ sessionId: session.id, expectedRevision: session.revision });
  if (!response.session) throw new Error("Focus service returned no session");
  return response.session;
}

export async function fetchActuals(localDate = ""): Promise<Actual[]> {
  const response = await focusClient.listActuals({ localDate });
  return response.actuals;
}

export async function fetchActualCorrections(input: { actualId?: string; localDate?: string; limit?: number } = {}): Promise<ActualCorrection[]> {
  const response = await focusClient.listActualCorrections({ actualId: input.actualId ?? "", localDate: input.localDate ?? "", limit: input.limit ?? 100 });
  return response.corrections;
}

export async function recordManualActual(input: { title: string; localDate: string; reportedMinutes: number; workItemId?: string; certainty?: string; note?: string }): Promise<Actual> {
  const response = await focusClient.recordManualActual({ workItemId: input.workItemId ?? "", title: input.title, localDate: input.localDate, reportedMinutes: BigInt(input.reportedMinutes), certainty: input.certainty ?? "user_reported_approximate", note: input.note ?? "" });
  if (!response.actual) throw new Error("Focus service returned no actual");
  return response.actual;
}

export async function correctActual(actual: Actual, input: { reportedMinutes: number; certainty?: string; note?: string }): Promise<Actual> {
  const response = await focusClient.correctActual({ id: actual.id, expectedRevision: actual.revision, reportedMinutes: BigInt(input.reportedMinutes), certainty: input.certainty ?? actual.certainty, note: input.note ?? actual.note });
  if (!response.actual) throw new Error("Focus service returned no actual");
  return response.actual;
}

export type { FocusSession };
