import { ApiVersion, ApiVersionCreateInput, ApiVersionTranslation, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput, ApiVersionUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { ApiShape, shapeApi } from "./api";
import { ProjectVersionDirectoryShape } from "./projectVersionDirectory";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ApiVersionTranslationShape = Pick<ApiVersionTranslation, "id" | "language" | "details" | "name" | "summary"> & {
    __typename?: "ApiVersionTranslation";
}

export type ApiVersionShape = Pick<ApiVersion, "id" | "callLink" | "documentationLink" | "isComplete" | "isPrivate" | "schemaLanguage" | "schemaText" | "versionLabel" | "versionNotes"> & {
    __typename: "ApiVersion";
    directoryListings?: CanConnect<ProjectVersionDirectoryShape>[] | null;
    resourceList?: ResourceListShape | null;
    root?: CanConnect<ApiShape> | null;
    translations?: ApiVersionTranslationShape[] | null;
}

export const shapeApiVersionTranslation: ShapeModel<ApiVersionTranslationShape, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "details", "name", "summary"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "details", "summary")),
};

export const shapeApiVersion: ShapeModel<ApiVersionShape, ApiVersionCreateInput, ApiVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "callLink", "documentationLink", "isComplete", "isPrivate", "schemaLanguage", "schemaText", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Connect"], "many"),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "ApiVersion" } })),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeApi, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "translations", ["Create"], "many", shapeApiVersionTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "callLink", "documentationLink", "isComplete", "isPrivate", "schemaLanguage", "schemaText", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "ApiVersion" } })),
        ...updateRel(o, u, "root", ["Update"], "one", shapeApi),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeApiVersionTranslation),
    }),
};
