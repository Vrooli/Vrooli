import { ProjectVersion, ProjectVersionCreateInput, ProjectVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type ProjectVersionShape = Pick<ProjectVersion, 'id'>

export const shapeProjectVersion: ShapeModel<ProjectVersionShape, ProjectVersionCreateInput, ProjectVersionUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}