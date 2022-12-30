import { ResourceList, ResourceListCreateInput, ResourceListTranslation, ResourceListTranslationCreateInput, ResourceListUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged } from "./objectTools";
import { ResourceShape } from "./resource";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type ResourceListTranslationShape = OmitCalculated<ResourceListTranslation>

export type ResourceListShape = Omit<OmitCalculated<ResourceList>, 'translations' | 'resources'> & {
    id: string;
    resources: Omit<ResourceShape, 'listId'>[] | null;
    translations: ResourceListTranslationShape[] | null;
}

export const shapeResourceListTranslationCreate = (item: ResourceListTranslationShape): ResourceListTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description', 'name')

export const shapeResourceListTranslationUpdate = (o: ResourceListTranslationShape, u: ResourceListTranslationShape): ResourceListTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description', 'name'))

export const shapeResourceListCreate = (item: ResourceListShape): ResourceListCreateInput => ({
    ...shapeCreatePrims(item, 'id'),
    ...shapeCreateList(item, 'translations', shapeResourceListTranslationCreate),
    ...shapeCreateList({
        resources: item.resources?.map(r => ({
            ...r,
            listId: item.id,
        }))
    }, 'resources', shapeResourceCreate),
})

export const shapeResourceListUpdate = (o: ResourceListShape, u: ResourceListShape): ResourceListUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeResourceListTranslationCreate, shapeResourceListTranslationUpdate, 'id'),
        ...shapeUpdateList({
            resources: o.resources?.map(r => ({
                ...r,
                listId: o.id,
            }))
        }, {
            resources: u.resources?.map(r => ({
                ...r,
                listId: u.id,
            }))
        }, 'resources', hasObjectChanged, shapeResourceCreate, shapeResourceUpdate, 'id'),
    })