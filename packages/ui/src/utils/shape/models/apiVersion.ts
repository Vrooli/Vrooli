import { ApiVersion, ApiVersionCreateInput, ApiVersionTranslation, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput, ApiVersionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { ApiShape, shapeApi } from "./api";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ApiVersionTranslationShape = Pick<ApiVersionTranslation, "id" | "language" | "details" | "name" | "summary"> & {
    __typename?: "ApiVersionTranslation";
}

export type ApiVersionShape = Pick<ApiVersion, "id" | "callLink" | "documentationLink" | "isComplete" | "isPrivate" | "versionLabel" | "versionNotes"> & {
    __typename: "ApiVersion";
    directoryListings?: { id: string }[] | null;
    resourceList?: Omit<ResourceListShape, "listFor"> | null;
    root?: { id: string } | ApiShape | null;
    translations?: ApiVersionTranslationShape[] | null;
}

export const shapeApiVersionTranslation: ShapeModel<ApiVersionTranslationShape, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "details", "name", "summary"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "details", "summary"), a),
};

export const shapeApiVersion: ShapeModel<ApiVersionShape, ApiVersionCreateInput, ApiVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "callLink", "documentationLink", "isComplete", "isPrivate", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Connect"], "many"),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "ApiVersion" } })),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeApi, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "translations", ["Create"], "many", shapeApiVersionTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "callLink", "documentationLink", "isComplete", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "ApiVersion" } })),
        ...updateRel(o, u, "root", ["Update"], "one", shapeApi),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeApiVersionTranslation),
    }, a),
};
