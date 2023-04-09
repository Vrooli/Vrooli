import { RunProjectStep, RunProjectStepCreateInput, RunProjectStepUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type RunProjectStepShape = Pick<RunProjectStep, 'id'> & {
    __typename?: 'RunProjectStep';
}

export const shapeRunProjectStep: ShapeModel<RunProjectStepShape, RunProjectStepCreateInput, RunProjectStepUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}