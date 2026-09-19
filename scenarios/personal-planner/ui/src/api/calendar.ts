import { createClient } from "@connectrpc/connect";
import { CalendarService, type ListAllocationsResponse, type ListTodayAllocationsResponse, type Allocation, type Routine, type RoutineOccurrence } from "@vrooli/proto-types/personal-planner/v1/calendar/calendar_pb";

import { transport } from "./client";

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
export async function carryForwardAllocation(input: { allocationId: string; targetLocalDate: string; startMinutes: number }): Promise<Allocation> {
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
export type { Allocation, Routine, RoutineOccurrence };
