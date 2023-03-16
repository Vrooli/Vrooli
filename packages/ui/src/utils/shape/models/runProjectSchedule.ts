import { RunProjectSchedule, RunProjectScheduleCreateInput, RunProjectScheduleUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type RunProjectScheduleShape = Pick<RunProjectSchedule, 'id'> & {
    __typename?: 'RunProjectSchedule';
}

export const shapeRunProjectSchedule: ShapeModel<RunProjectScheduleShape, RunProjectScheduleCreateInput, RunProjectScheduleUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}