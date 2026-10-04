import { createClient } from "@connectrpc/connect";
import { CalendarService, type ListAllocationsResponse, type ListTodayAllocationsResponse, type Allocation, type Routine, type RoutineOccurrence, type PlacementProposal, type ScheduleProposal } from "@vrooli/proto-types/personal-planner/v1/calendar/calendar_pb";

import { API_BASE, transport } from "./client";

export type CalendarEvent = {
  id: string;
  title: string;
  subject: string;
  notes: string;
  availability: "busy" | "free";
  timezone: string;
  all_day: boolean;
  start_date: string;
  end_date_exclusive: string;
  start_at: string;
  end_at: string;
  provider: string;
  provider_calendar_id: string;
  provider_event_id: string;
  occurrence_id: string;
  revision: number;
  created_at: string;
  updated_at: string;
};

export type CalendarEventInput = Omit<CalendarEvent, "id" | "revision" | "created_at" | "updated_at" | "provider" | "provider_calendar_id" | "provider_event_id" | "occurrence_id">;

async function eventRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, { ...init, headers: { "Content-Type": "application/json", ...init?.headers } });
  if (!response.ok) {
    const detail = (await response.text()).trim();
    throw new Error(detail || `Calendar event request failed (${response.status})`);
  }
  return await response.json() as T;
}

export async function fetchCalendarEvents(startLocalDate: string, endLocalDate: string): Promise<CalendarEvent[]> {
  const query = new URLSearchParams({ start_local_date: startLocalDate, end_local_date: endLocalDate });
  const response = await eventRequest<{ events: CalendarEvent[] }>(`/api/v1/calendar/events?${query.toString()}`);
  return response.events;
}

export async function fetchCalendarEvent(id: string): Promise<CalendarEvent> {
  return await eventRequest<CalendarEvent>(`/api/v1/calendar/events/${encodeURIComponent(id)}`);
}

export async function lookupCalendarEventByIdempotencyKey(key: string): Promise<CalendarEvent | null> {
  const response = await fetch(`${API_BASE}/api/v1/calendar/event-commands/${encodeURIComponent(key)}`);
  if (response.status === 404) return null;
  if (!response.ok) throw new Error((await response.text()).trim() || `Calendar event status failed (${response.status})`);
  return await response.json() as CalendarEvent;
}

export async function createCalendarEvent(event: CalendarEventInput, idempotencyKey: string): Promise<CalendarEvent> {
  return await eventRequest<CalendarEvent>("/api/v1/calendar/events", { method: "POST", body: JSON.stringify({ ...event, idempotency_key: idempotencyKey }) });
}

export async function updateCalendarEvent(id: string, event: CalendarEventInput, expectedRevision: number): Promise<CalendarEvent> {
  return await eventRequest<CalendarEvent>(`/api/v1/calendar/events/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify({ ...event, expected_revision: expectedRevision }) });
}

export const calendarClient = createClient(CalendarService, transport);
export async function fetchTodayAllocations(localDate = ""): Promise<ListTodayAllocationsResponse> { return calendarClient.listTodayAllocations({ localDate }); }
export async function fetchAllocations(startLocalDate: string, endLocalDate: string): Promise<ListAllocationsResponse> {
  return calendarClient.listAllocations({ startLocalDate, endLocalDate });
}
export async function createAllocation(input: { workItemId: string; localDate: string; startMinutes: number; durationMinutes: number }): Promise<Allocation> {
  const response = await calendarClient.createAllocation(input);
  if (!response.allocation) throw new Error("allocation was not returned");
  return response.allocation;
}
export async function previewAllocation(input: { workItemId: string; localDate: string; startMinutes: number; durationMinutes: number }): Promise<PlacementProposal> {
  const response = await calendarClient.previewAllocation(input);
  if (!response.proposal) throw new Error("placement proposal was not returned");
  return response.proposal;
}
export async function applyAllocationProposal(input: { proposalId: string; expectedRevision: bigint; idempotencyKey: string }): Promise<Allocation> {
  const response = await calendarClient.applyAllocationProposal(input);
  if (!response.allocation) throw new Error("applied allocation was not returned");
  return response.allocation;
}
export async function previewSchedule(input: { localDate: string; startMinutes: number; workItemIds: string[] }): Promise<ScheduleProposal> {
  const response = await calendarClient.previewSchedule(input);
  if (!response.proposal) throw new Error("schedule proposal was not returned");
  return response.proposal;
}
export async function applyScheduleProposal(input: { proposalId: string; expectedRevision: bigint; idempotencyKey: string }): Promise<Allocation[]> {
  const response = await calendarClient.applyScheduleProposal(input);
  return response.allocations;
}
export async function carryForwardAllocation(input: { allocationId: string; targetLocalDate: string; startMinutes: number; reasonCode?: "interrupted" | "underestimated" | "blocked" | "deprioritized" | "external" }): Promise<Allocation> {
  if (input.reasonCode) {
    const response = await fetch(`${API_BASE}/api/v1/calendar/carry-forward`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ allocation_id: input.allocationId, target_local_date: input.targetLocalDate, start_minutes: input.startMinutes, reason_code: input.reasonCode }) });
    if (!response.ok) throw new Error("The allocation could not be carried forward");
    return await response.json() as Allocation;
  }
  const response = await calendarClient.carryForwardAllocation(input);
  if (!response.allocation) throw new Error("carried allocation was not returned");
  return response.allocation;
}
export async function fetchRoutines(): Promise<Routine[]> { const response = await calendarClient.listRoutines({}); return response.routines; }
export async function createRoutine(input: { title: string; kind: string; timezone: string; startDate: string; endDate: string; weekdays: number[]; startMinute: number; durationMinutes: number; frequencyPerWeek: number }): Promise<Routine> {
  const response = await calendarClient.createRoutine(input);
  if (!response.routine) throw new Error("routine was not returned");
  return response.routine;
}
export async function fetchRoutineOccurrences(startLocalDate: string, endLocalDate: string): Promise<RoutineOccurrence[]> {
  const response = await calendarClient.listRoutineOccurrences({ startLocalDate, endLocalDate });
  return response.occurrences;
}
export async function skipRoutineOccurrence(input: { routineId: string; localDate: string; expectedRevision: bigint }): Promise<void> {
  const response = await calendarClient.skipRoutineOccurrence(input);
  if (!response.skipped) throw new Error("routine occurrence was not skipped");
}
export async function rescheduleRoutineOccurrence(input: { routineId: string; localDate: string; startMinute: number; expectedRevision: bigint }): Promise<void> {
  const response = await calendarClient.rescheduleRoutineOccurrence(input);
  if (!response.rescheduled) throw new Error("routine occurrence was not rescheduled");
}
export type { Allocation, Routine, RoutineOccurrence, PlacementProposal, ScheduleProposal };
