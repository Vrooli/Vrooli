import { Resource, ResourceCreateInput, ResourceTranslation, ResourceTranslationCreateInput, ResourceTranslationUpdateInput, ResourceUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged } from "./objectTools";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type ResourceTranslationShape = OmitCalculated<ResourceTranslation>

export type ResourceShape = Omit<OmitCalculated<Resource>, 'translations' | 'link' | 'usedFor'> & {
    id: string;
    link: ResourceCreateInput['link'];
    listId: string;
    usedFor: ResourceCreateInput['usedFor'] | null;
    translations: ResourceTranslationShape[];
}

export const shapeResourceTranslationCreate = (item: ResourceTranslationShape): ResourceTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description', 'name')

export const shapeResourceTranslationUpdate = (o: ResourceTranslationShape, u: ResourceTranslationShape): ResourceTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description', 'name'))

export const shapeResourceCreate = (item: ResourceShape): ResourceCreateInput => ({
    ...shapeCreatePrims(item, 'id', 'index', 'link', 'usedFor'),
    listConnect: item.listId,
    ...shapeCreateList(item, 'translations', shapeResourceTranslationCreate),
})

export const shapeResourceUpdate = (o: ResourceShape, u: ResourceShape): ResourceUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'index', 'link', 'listId', 'usedFor'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeResourceTranslationCreate, shapeResourceTranslationUpdate, 'id'),
    })