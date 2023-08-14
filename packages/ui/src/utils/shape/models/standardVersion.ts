import { StandardVersion, StandardVersionCreateInput, StandardVersionTranslation, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput, StandardVersionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { shapeStandard, StandardShape } from "./standard";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
import { updateTranslationPrims } from "./tools/updateTranslationPrims";

export type StandardVersionTranslationShape = Pick<StandardVersionTranslation, "id" | "language" | "description" | "jsonVariable" | "name"> & {
    __typename?: "StandardVersionTranslation";
}

export type StandardVersionShape = Pick<StandardVersion, "id" | "isComplete" | "isPrivate" | "isFile" | "default" | "props" | "yup" | "standardType" | "versionLabel" | "versionNotes"> & {
    __typename: "StandardVersion";
    directoryListings?: { id: string }[] | null;
    root: StandardShape;
    resourceList?: { id: string } | ResourceListShape | null;
    translations?: StandardVersionTranslationShape[] | null;
}

export const shapeStandardVersionTranslation: ShapeModel<StandardVersionTranslationShape, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "jsonVariable", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "jsonVariable", "name"), a),
};

export const shapeStandardVersion: ShapeModel<StandardVersionShape, StandardVersionCreateInput, StandardVersionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isComplete", "isPrivate", "isFile", "default", "props", "yup", "standardType", "versionLabel", "versionNotes"),
        ...createRel(d, "directoryListings", ["Create"], "many", shapeProjectVersionDirectory),
        ...createRel(d, "root", ["Connect", "Create"], "one", shapeStandard, (r) => ({ ...r, isPrivate: d.isPrivate })),
        ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: d.id, __typename: "StandardVersion" } })),
        ...createRel(d, "translations", ["Create"], "many", shapeStandardVersionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isComplete", "isPrivate", "isFile", "default", "props", "yup", "standardType", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeStandard),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "StandardVersion" } })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeStandardVersionTranslation),
    }, a),
};
