import { Resource, ResourceCreateInput, ResourceTranslation, ResourceTranslationCreateInput, ResourceTranslationUpdateInput, ResourceUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, shapeUpdate, updatePrims } from "utils";

export type ResourceTranslationShape = Pick<ResourceTranslation, 'id' | 'language' | 'description' | 'name'>

export type ResourceShape = Omit<OmitCalculated<Resource>, 'translations' | 'link' | 'usedFor'> & {
    id: string;
    link: ResourceCreateInput['link'];
    listId: string;
    usedFor: ResourceCreateInput['usedFor'] | null;
    translations: ResourceTranslationShape[];
}

export const shapeResourceTranslation: ShapeModel<ResourceTranslationShape, ResourceTranslationCreateInput, ResourceTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'))
}

export const shapeResource: ShapeModel<ResourceShape, ResourceCreateInput, ResourceUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id', 'index', 'link', 'usedFor'),
        listConnect: item.listId,
        ...shapeCreateList(item, 'translations', shapeResourceTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'index', 'link', 'listId', 'usedFor'),
        ...shapeUpdateList(o, u, 'translations', shapeResourceTranslation, 'id'),
    })
}