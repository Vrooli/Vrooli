import { ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type ProjectVersionDirectoryShape = Pick<ProjectVersionDirectory, 'id'> & {
    __typename?: 'ProjectVersionDirectory';
}

export const shapeProjectVersionDirectory: ShapeModel<ProjectVersionDirectoryShape, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}