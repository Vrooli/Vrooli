import { RunProjectSchedule, RunProjectScheduleCreateInput, RunProjectScheduleUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type RunProjectScheduleShape = Pick<RunProjectSchedule, 'id'>

export const shapeRunProjectSchedule: ShapeModel<RunProjectScheduleShape, RunProjectScheduleCreateInput, RunProjectScheduleUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}