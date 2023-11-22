import { ResourceList, ResourceListCreateInput, ResourceListFor, ResourceListTranslation, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput, ResourceListUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { ResourceShape, shapeResource } from "./resource";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ResourceListTranslationShape = Pick<ResourceListTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ResourceListTranslation";
}

export type ResourceListShape = Pick<ResourceList, "id"> & {
    __typename?: "ResourceList";
    resources?: ResourceShape[] | null;
    translations?: ResourceListTranslationShape[] | null;
} & ({
    listForId: string;
    listForType: ResourceListFor | `${ResourceListFor}`;
} | {
    listFor: {
        __typename: ResourceListFor | `${ResourceListFor}`;
        id: string;
    };
})

export const shapeResourceListTranslation: ShapeModel<ResourceListTranslationShape, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};

export const shapeResourceList: ShapeModel<ResourceListShape, ResourceListCreateInput, ResourceListUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id");
        let listForConnect: string | undefined;
        let listForType: ResourceListFor | undefined;
        console.log("in shapeResourceListCreate", d, "listForId" in d, "listForType" in d, "listFor" in d);
        if ("listForId" in d && "listForType" in d) {
            listForConnect = d.listForId;
            listForType = d.listForType as ResourceListFor;
        } else if ("listFor" in d) {
            listForConnect = d.listFor.id;
            listForType = d.listFor.__typename as ResourceListFor;
        }
        return {
            ...prims,
            listForConnect: listForConnect as string,
            listForType: listForType as ResourceListFor,
            ...createRel(d, "resources", ["Create"], "many", shapeResource, (r) => ({ list: { id: prims.id }, ...r })),
            ...createRel(d, "translations", ["Create"], "many", shapeResourceListTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "resources", ["Create", "Update", "Delete"], "many", shapeResource, (r, i) => ({ list: { id: i.id }, ...r })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeResourceListTranslation),
    }),
};
