import { ResourceList, ResourceListCreateInput, ResourceListTranslation, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput, ResourceListUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ResourceShape, shapeResource } from "./resource";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ResourceListTranslationShape = Pick<ResourceListTranslation, 'id' | 'language' | 'description' | 'name'> & {
    __typename?: 'ResourceListTranslation';
}

export type ResourceListShape = Pick<ResourceList, 'id'> & {
    __typename?: 'ResourceList';
    resources?: ResourceShape[] | null;
    translations?: ResourceListTranslationShape[] | null;
}

export const shapeResourceListTranslation: ShapeModel<ResourceListTranslationShape, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'name'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'))
}

export const shapeResourceList: ShapeModel<ResourceListShape, ResourceListCreateInput, ResourceListUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id'),
        ...createRel(d, 'resources', ['Create'], 'many', shapeResource, (r) => ({ list: { id: d.id }, ...r })),
        ...createRel(d, 'translations', ['Create'], 'many', shapeResourceListTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id'),
        ...updateRel(o, u, 'resources', ['Create', 'Update', 'Delete'], 'many', shapeResource, (r, i) => ({ list: { id: i.id} , ...r })),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeResourceListTranslation),
    })
}