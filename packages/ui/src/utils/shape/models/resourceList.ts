import { ResourceList, ResourceListCreateInput, ResourceListTranslation, ResourceListTranslationCreateInput, ResourceListUpdateInput } from "@shared/consts";
import { ResourceShape } from "./resource";
import { createPrims, shapeUpdate, updatePrims } from "utils";
import { ShapeModel } from "types";

export type ResourceListTranslationShape = Pick<ResourceListTranslation, 'id' | 'language' | 'description' | 'name'>

export type ResourceListShape = Omit<OmitCalculated<ResourceList>, 'translations' | 'resources'> & {
    id: string;
    resources: Omit<ResourceShape, 'listId'>[] | null;
    translations: ResourceListTranslationShape[] | null;
}

export const shapeResourceListTranslation: ShapeModel<ResourceListTranslationShape, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'))
}

export const shapeResourceList: ShapeModel<ResourceListShape, ResourceListCreateInput, ResourceListUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id'),
        ...shapeCreateList(item, 'translations', shapeResourceListTranslationCreate),
        ...shapeCreateList({
            resources: item.resources?.map(r => ({
                ...r,
                listId: item.id,
            }))
        }, 'resources', shapeResourceCreate),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id'),
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
}