import { Standard, StandardCreateInput, StandardUpdateInput, StandardVersionTranslation, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { hasObjectChanged, ResourceListShape, shapeCreateList, createPrims, shapeResourceListCreate, shapeResourceListUpdate, shapeTagCreate, shapeTagUpdate, shapeUpdate, shapeUpdateList, updatePrims, TagShape } from "utils";


export type StandardShape = Omit<OmitCalculated<Standard>, 'props' | 'type' | 'name' | 'resourceLists' | 'tags' | 'translations' | 'creator'> & {
    id: string;
    props: StandardCreateInput['props'];
    type: StandardCreateInput['type'];
    name: StandardCreateInput['name'];
    resourceLists?: ResourceListShape[];
    tags?: TagShape[];
    translations: StandardVersionTranslationShape[];
    creator?: {
        __typename: 'User' | 'Organization';
        id: string;
    } | null;
}

export type StandardShapeUpdate = Omit<StandardShape, 'default' | 'isInternal' | 'name' | 'props' | 'yup' | 'type' | 'version' | 'creator'>;

export const shapeStandard: ShapeModel<StandardShape, StandardCreateInput, StandardUpdateInput> = {
    create: (item) => ({
        id: item.id,
        default: item.default + '', // Make sure default is a string
        isInternal: item.isInternal,
        isPrivate: item.isPrivate,
        name: item.name,
        props: item.props,
        yup: item.yup,
        type: item.type,
        // version: item.version,TODO
        // parentId: u.parent?.id, TODO
        createdByUserId: item.creator?.__typename === 'User' ? item.creator.id : undefined,
        createdByOrganizationId: item.creator?.__typename === 'Organization' ? item.creator.id : undefined,
        ...shapeCreateList(item, 'translations', shapeStandardTranslationCreate),
        ...shapeCreateList(item, 'resourceLists', shapeResourceListCreate),
        ...shapeCreateList(item, 'tags', shapeTagCreate),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isPrivate'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeStandardTranslationCreate, shapeStandardTranslationUpdate, 'id'),
        ...shapeUpdateList(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate, 'id'),
        ...shapeUpdateList(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate, 'tag', true, true),
    }) //TODO
}