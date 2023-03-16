import { RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type RunRoutineStepShape = Pick<RunRoutineStep, 'id'> & {
    __typename?: 'RunRoutineStep';
}

export const shapeRunRoutineStep: ShapeModel<RunRoutineStepShape, RunRoutineStepCreateInput, RunRoutineStepUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}