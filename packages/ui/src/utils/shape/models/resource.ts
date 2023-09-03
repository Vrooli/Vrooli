import { addHttps, Resource, ResourceCreateInput, ResourceTranslation, ResourceTranslationCreateInput, ResourceTranslationUpdateInput, ResourceUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ResourceTranslationShape = Pick<ResourceTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ResourceTranslation";
}

export type ResourceShape = Pick<Resource, "id" | "index" | "link" | "usedFor"> & {
    __typename: "Resource";
    list: { id: string } | ResourceListShape;
    translations: ResourceTranslationShape[];
}

export const shapeResourceTranslation: ShapeModel<ResourceTranslationShape, ResourceTranslationCreateInput, ResourceTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};

export const shapeResource: ShapeModel<ResourceShape, ResourceCreateInput, ResourceUpdateInput> = {
    create: (d) => ({
        // Make sure link is properly shaped
        link: addHttps(d.link),
        ...createPrims(d, "id", "index", "usedFor"),
        ...createRel(d, "list", ["Connect", "Create"], "one", shapeResourceList),
        ...createRel(d, "translations", ["Create"], "many", shapeResourceTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        // Make sure link is properly shaped
        link: o.link !== u.link ? addHttps(u.link) : undefined,
        ...updatePrims(o, u, "id", "index", "usedFor"),
        ...updateRel(o, u, "list", ["Connect", "Create"], "one", shapeResourceList),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeResourceTranslation),
    }),
};
