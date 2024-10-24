import { StandardVersion, StandardVersionCreateInput, StandardVersionTranslation, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput, StandardVersionUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { ProjectVersionDirectoryShape, shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { StandardShape, shapeStandard } from "./standard";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type StandardVersionTranslationShape = Pick<StandardVersionTranslation, "id" | "language" | "description" | "jsonVariable" | "name"> & {
    __typename?: "StandardVersionTranslation";
}

export type StandardVersionShape = Pick<StandardVersion, "id" | "isComplete" | "isPrivate" | "isFile" | "codeLanguage" | "default" | "props" | "yup" | "variant" | "versionLabel" | "versionNotes"> & {
    __typename: "StandardVersion";
    directoryListings?: CanConnect<ProjectVersionDirectoryShape>[] | null;
    root: StandardShape;
    resourceList?: ResourceListShape | null;
    translations?: StandardVersionTranslationShape[] | null;
}

export const shapeStandardVersionTranslation: ShapeModel<StandardVersionTranslationShape, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "jsonVariable", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "jsonVariable", "name")),
};

export const shapeStandardVersion: ShapeModel<StandardVersionShape, StandardVersionCreateInput, StandardVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "isComplete", "isPrivate", "isFile", "codeLanguage", "default", "props", "yup", "variant", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Create"], "many", shapeProjectVersionDirectory),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeStandard, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "StandardVersion" } })),
            ...createRel(d, "translations", ["Create"], "many", shapeStandardVersionTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isComplete", "isPrivate", "isFile", "codeLanguage", "default", "props", "yup", "variant", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeStandard),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "StandardVersion" } })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeStandardVersionTranslation),
    }),
};
