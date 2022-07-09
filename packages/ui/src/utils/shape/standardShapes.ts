import { StandardCreateInput, StandardTranslationCreateInput, StandardTranslationUpdateInput, StandardUpdateInput } from "graphql/generated/globalTypes";
import { ShapeWrapper, Standard, StandardTranslation } from "types";
import { hasObjectChanged, ObjectType, ResourceListShape, shapeResourceListCreate, shapeResourceListUpdate, shapeTagCreate, shapeTagUpdate, TagShape } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type StandardTranslationShape = Omit<ShapeWrapper<StandardTranslation>, 'language' | 'jsonVariable'> & {
    id: string;
    language: StandardTranslationCreateInput['language'];
    jsonVariable: StandardTranslationCreateInput['jsonVariable'];
}

export type StandardShape = Omit<ShapeWrapper<Standard>, 'props' | 'type' | 'name' | 'resourceLists' | 'tags' | 'translations' | 'creator'> & {
    id: string;
    props: StandardCreateInput['props'];
    type: StandardCreateInput['type'];
    name: StandardCreateInput['name'];
    resourceLists?: ResourceListShape[];
    tags?: TagShape[];
    translations: StandardTranslationShape[];
    creator?: {
        __typename: 'User' | 'Organization';
        id: string;
    } | null;
}

export type StandardShapeUpdate = Omit<StandardShape, 'default' | 'isInternal' | 'name' | 'props' | 'yup' | 'type' | 'version' | 'creator'>;

export const shapeStandardTranslationCreate = (item: StandardTranslationShape): StandardTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    jsonVariable: item.jsonVariable,
})

export const shapeStandardTranslationUpdate = (
    original: StandardTranslationShape,
    updated: StandardTranslationShape
): StandardTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
        jsonVariable: u.jsonVariable !== o.jsonVariable ? u.jsonVariable : undefined,
    }), 'id')

export const shapeStandardCreate = (item: StandardShape): StandardCreateInput => ({
    id: item.id,
    default: item.default,
    isInternal: item.isInternal,
    name: item.name,
    props: item.props,
    yup: item.yup,
    type: item.type,
    version: item.version,
    // parentId: u.parent?.id, TODO
    createdByUserId: item.creator?.__typename === ObjectType.User ? item.creator.id : undefined,
    createdByOrganizationId: item.creator?.__typename === ObjectType.Organization ? item.creator.id : undefined,
    ...shapeCreateList(item, 'translations', shapeStandardTranslationCreate),
    ...shapeCreateList(item, 'resourceLists', shapeResourceListCreate),
    ...shapeCreateList(item, 'tags', shapeTagCreate),
})

export const shapeStandardUpdate = (
    original: StandardShapeUpdate,
    updated: StandardShapeUpdate
): StandardUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        // makingAnonymous: updated.makingAnonymous, TODO
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeStandardTranslationCreate, shapeStandardTranslationUpdate, 'id'),
        ...shapeUpdateList(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate, 'id'),
        ...shapeUpdateList(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate, 'tag', true, true),
    }), 'id')