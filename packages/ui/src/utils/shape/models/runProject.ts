import { RunProject, RunProjectCreateInput, RunProjectUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type RunProjectShape = Pick<RunProject, 'id'>

export const shapeRunProject: ShapeModel<RunProjectShape, RunProjectCreateInput, RunProjectUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}