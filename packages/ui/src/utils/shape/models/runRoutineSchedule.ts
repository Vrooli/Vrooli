import { RunRoutineSchedule, RunRoutineScheduleCreateInput, RunRoutineScheduleUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type RunRoutineScheduleShape = Pick<RunRoutineSchedule, 'id'>

export const shapeRunRoutineSchedule: ShapeModel<RunRoutineScheduleShape, RunRoutineScheduleCreateInput, RunRoutineScheduleUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}