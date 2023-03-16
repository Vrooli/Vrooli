import { RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type RunRoutineShape = Pick<RunRoutine, 'id'> & {
    __typename?: 'RunRoutine';
}

export const shapeRunRoutine: ShapeModel<RunRoutineShape, RunRoutineCreateInput, RunRoutineUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}