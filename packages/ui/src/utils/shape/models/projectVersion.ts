import { ProjectVersion, ProjectVersionCreateInput, ProjectVersionTranslation, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput, ProjectVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ProjectShape, shapeProject } from "./project";
import { ProjectVersionDirectoryShape, shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ProjectVersionTranslationShape = Pick<ProjectVersionTranslation, 'id' | 'language' | 'description' | 'name'>

export type ProjectVersionShape = Pick<ProjectVersion, 'id' | 'isLatest' | 'isPrivate' | 'versionIndex' | 'versionLabel' | 'versionNotes'> & {
    directoryListings?: ProjectVersionDirectoryShape[];
    root: { id: string } | ProjectShape;
    suggestedNextByProject?: { id: string }[];
    translations?: ProjectVersionTranslationShape[];
}

export const shapeProjectVersionTranslation: ShapeModel<ProjectVersionTranslationShape, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'))
}

export const shapeProjectVersion: ShapeModel<ProjectVersionShape, ProjectVersionCreateInput, ProjectVersionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'isLatest', 'isPrivate', 'versionIndex', 'versionLabel', 'versionNotes'),
        ...createRel(d, 'directoryListings', ['Create'], 'many', shapeProjectVersionDirectory),
        ...createRel(d, 'root', ['Connect', 'Create'], 'one', shapeProject),
        ...createRel(d, 'suggestedNextByProject', ['Connect'], 'many'),
        ...createRel(d, 'translations', ['Create'], 'many', shapeProjectVersionTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isLatest', 'isPrivate', 'versionIndex', 'versionLabel', 'versionNotes'),
        ...updateRel(o, u, 'directoryListings', ['Create', 'Update', 'Delete'], 'many', shapeProjectVersionDirectory),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeProjectVersionTranslation),
        ...updateRel(o, u, 'suggestedNextByProject', ['Connect', 'Disconnect'], 'many')
    })
}