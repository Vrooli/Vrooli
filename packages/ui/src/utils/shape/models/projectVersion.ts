import { ProjectVersion, ProjectVersionCreateInput, ProjectVersionTranslation, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput, ProjectVersionUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { ProjectShape, shapeProject } from "./project";
import { ProjectVersionDirectoryShape, shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { ResourceListShape } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ProjectVersionTranslationShape = Pick<ProjectVersionTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ProjectVersionTranslation";
}

export type ProjectVersionShape = Pick<ProjectVersion, "id" | "isComplete" | "isPrivate" | "versionLabel" | "versionNotes"> & {
    __typename: "ProjectVersion";
    directories?: ProjectVersionDirectoryShape[] | null;
    resourceList?: CanConnect<ResourceListShape> | null;
    root?: CanConnect<ProjectShape> | null;
    suggestedNextByProject?: CanConnect<ProjectShape>[] | null;
    translations?: ProjectVersionTranslationShape[] | null;
}

export const shapeProjectVersionTranslation: ShapeModel<ProjectVersionTranslationShape, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name"), a),
};

export const shapeProjectVersion: ShapeModel<ProjectVersionShape, ProjectVersionCreateInput, ProjectVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "isComplete", "isPrivate", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directories", ["Create"], "many", shapeProjectVersionDirectory),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeProject, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "suggestedNextByProject", ["Connect"], "many"),
            ...createRel(d, "translations", ["Create"], "many", shapeProjectVersionTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directories", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeProject),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeProjectVersionTranslation),
        ...updateRel(o, u, "suggestedNextByProject", ["Connect", "Disconnect"], "many"),
    }, a),
};
