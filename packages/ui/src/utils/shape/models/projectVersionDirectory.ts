import { ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type ProjectVersionDirectoryShape = Pick<ProjectVersionDirectory, 'id'>

export const shapeProjectVersionDirectory: ShapeModel<ProjectVersionDirectoryShape, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}