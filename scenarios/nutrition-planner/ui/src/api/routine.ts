import { createClient } from "@connectrpc/connect";
import { RoutineService, type Occurrence, type Template } from "@vrooli/proto-types/nutrition-planner/v1/routine/routine_pb";
import { transport } from "./client";

const client = createClient(RoutineService, transport);

export type RoutineTemplate = Template;
export type RoutineOccurrence = Occurrence;

export async function listRoutineTemplates(workspaceId: string): Promise<RoutineTemplate[]> {
  const response = await client.listTemplates({ workspaceId });
  return response.templates;
}

export async function createRoutineTemplate(input: {
  workspaceId: string;
  slotName: string;
  recipeId?: string;
  quantity: string;
  weekdays: number[];
  startDate: string;
  endDate?: string;
  mode: string;
  active: boolean;
}): Promise<RoutineTemplate> {
  const response = await client.createTemplate({ ...input, recipeId: input.recipeId ?? "", endDate: input.endDate ?? "" });
  if (!response.template) throw new Error("The API returned no routine template.");
  return response.template;
}

export async function generateRoutineOccurrences(input: { workspaceId: string; fromDate: string; toDate: string }): Promise<RoutineOccurrence[]> {
  const response = await client.generateOccurrences(input);
  return response.occurrences;
}
