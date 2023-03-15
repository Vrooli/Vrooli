import { ProjectVersion, ProjectVersionCreateInput, ProjectVersionTranslation, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput, ProjectVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ProjectShape, shapeProject } from "./project";
import { ProjectVersionDirectoryShape, shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ProjectVersionTranslationShape = Pick<ProjectVersionTranslation, 'id' | 'language' | 'description' | 'name'>

export type ProjectVersionShape = Pick<ProjectVersion, 'id' | 'isComplete' | 'isPrivate' | 'versionLabel' | 'versionNotes'> & {
    directoryListings?: ProjectVersionDirectoryShape[] | null;
    root?: { id: string } | ProjectShape | null;
    suggestedNextByProject?: { id: string }[] | null;
    translations?: ProjectVersionTranslationShape[] | null;
}

export const shapeProjectVersionTranslation: ShapeModel<ProjectVersionTranslationShape, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'name'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'), a)
}

export const shapeProjectVersion: ShapeModel<ProjectVersionShape, ProjectVersionCreateInput, ProjectVersionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'isComplete', 'isPrivate', 'versionLabel', 'versionNotes'),
        ...createRel(d, 'directoryListings', ['Create'], 'many', shapeProjectVersionDirectory),
        ...createRel(d, 'root', ['Connect', 'Create'], 'one', shapeProject),
        ...createRel(d, 'suggestedNextByProject', ['Connect'], 'many'),
        ...createRel(d, 'translations', ['Create'], 'many', shapeProjectVersionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isPrivate', 'versionLabel', 'versionNotes'),
        ...updateRel(o, u, 'directoryListings', ['Create', 'Update', 'Delete'], 'many', shapeProjectVersionDirectory),
        ...updateRel(o, u, 'root', ['Update'], 'one', shapeProject),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeProjectVersionTranslation),
        ...updateRel(o, u, 'suggestedNextByProject', ['Connect', 'Disconnect'], 'many')
    }, a)
}