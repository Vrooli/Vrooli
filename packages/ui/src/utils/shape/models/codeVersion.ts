import { CodeVersion, CodeVersionCreateInput, CodeVersionTranslation, CodeVersionTranslationCreateInput, CodeVersionTranslationUpdateInput, CodeVersionUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { CodeShape, shapeCode } from "./code";
import { ProjectVersionDirectoryShape, shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type CodeVersionTranslationShape = Pick<CodeVersionTranslation, "id" | "language" | "description" | "name" | "jsonVariable"> & {
    __typename?: "CodeVersionTranslation";
}

export type CodeVersionShape = Pick<CodeVersion, "id" | "calledByRoutineVersionsCount" | "codeLanguage" | "codeType" | "content" | "default" | "isComplete" | "isPrivate" | "versionLabel" | "versionNotes"> & {
    __typename: "CodeVersion";
    directoryListings?: ProjectVersionDirectoryShape[] | null;
    resourceList?: ResourceListShape | null;
    root?: CanConnect<CodeShape> | null;
    suggestedNextByCode?: CanConnect<CodeVersionShape>[] | null;
    translations?: CodeVersionTranslationShape[] | null;
}

export const shapeCodeVersionTranslation: ShapeModel<CodeVersionTranslationShape, CodeVersionTranslationCreateInput, CodeVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name", "jsonVariable"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "language", "description", "name", "jsonVariable"), a),
};

export const shapeCodeVersion: ShapeModel<CodeVersionShape, CodeVersionCreateInput, CodeVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "codeLanguage", "codeType", "content", "default", "isComplete", "isPrivate", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Create"], "many", shapeProjectVersionDirectory),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeCode, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "CodeVersion" } })),
            ...createRel(d, "translations", ["Create"], "many", shapeCodeVersionTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "codeLanguage", "content", "default", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeCode),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "CodeVersion" } })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeCodeVersionTranslation),
    }, a),
};
