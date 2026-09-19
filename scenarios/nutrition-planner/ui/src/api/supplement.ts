import { createClient } from "@connectrpc/connect";
import { SupplementService } from "@vrooli/proto-types/nutrition-planner/v1/supplement/supplement_pb";
import { transport } from "./client";

const client = createClient(SupplementService, transport);

export type SupplementSchedule = { id: string; revision: bigint; productRevisionId: string; dose: string; doseUnit: string; weekdays: number[]; startDate: string; endDate: string; paused: boolean; confirmed: boolean; createdAt: string };

function mapSchedule(schedule: SupplementSchedule): SupplementSchedule { return { ...schedule, weekdays: [...schedule.weekdays] }; }

export async function listSupplementSchedules(workspaceId: string): Promise<SupplementSchedule[]> {
  const response = await client.listSchedules({ workspaceId });
  return response.schedules.map((schedule) => mapSchedule(schedule));
}

export async function createSupplementSchedule(input: { workspaceId: string; productRevisionId: string; dose: string; doseUnit: string; weekdays: number[]; startDate: string; endDate?: string; confirmed: boolean }): Promise<SupplementSchedule> {
  const response = await client.createSchedule({ ...input, endDate: input.endDate ?? "" });
  if (!response.schedule) throw new Error("The API returned no supplement schedule.");
  return mapSchedule(response.schedule);
}

export async function updateSupplementSchedule(input: { workspaceId: string; id: string; expectedRevision: bigint; dose: string; doseUnit: string; weekdays: number[]; startDate: string; endDate?: string; paused: boolean; confirmed: boolean }): Promise<SupplementSchedule> {
  const response = await client.updateSchedule({ ...input, endDate: input.endDate ?? "" });
  if (!response.schedule) throw new Error("The API returned no updated supplement schedule.");
  return mapSchedule(response.schedule);
}
