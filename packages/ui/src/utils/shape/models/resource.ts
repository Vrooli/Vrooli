import { Resource, ResourceCreateInput, ResourceTranslation, ResourceTranslationCreateInput, ResourceTranslationUpdateInput, ResourceUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "utils";

export type ResourceTranslationShape = Pick<ResourceTranslation, 'id' | 'language' | 'description' | 'name'>

export type ResourceShape = Pick<Resource, 'id' | 'index' | 'link' | 'usedFor'> & {
    list: { __typename?: 'ResourceList', id: string };
    translations: ResourceTranslationShape[];
}

export const shapeResourceTranslation: ShapeModel<ResourceTranslationShape, ResourceTranslationCreateInput, ResourceTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'name'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'))
}

export const shapeResource: ShapeModel<ResourceShape, ResourceCreateInput, ResourceUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'index', 'link', 'usedFor'),
        ...createRel(d, 'list', ['Connect'], 'one'),
        ...createRel(d, 'translations', ['Create'], 'many', shapeResourceTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'index', 'link', 'usedFor'),
        ...updateRel(o, u, 'list', ['Connect'], 'one'),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeResourceTranslation),
    })
}