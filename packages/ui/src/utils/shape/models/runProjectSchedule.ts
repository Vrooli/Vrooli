import { RunProjectSchedule, RunProjectScheduleCreateInput, RunProjectScheduleUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type RunProjectScheduleShape = Pick<RunProjectSchedule, 'id'>

export const shapeRunProjectSchedule: ShapeModel<RunProjectScheduleShape, RunProjectScheduleCreateInput, RunProjectScheduleUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}