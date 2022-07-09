import { RoutineCreateInput, RoutineTranslationCreateInput, RoutineTranslationUpdateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { Routine, RoutineTranslation, ShapeWrapper } from "types";
import { hasObjectChanged, InputShape, NodeLinkShape, NodeShape, ObjectType, OutputShape, ResourceListShape, shapeInputCreate, shapeInputUpdate, shapeNodeCreate, shapeNodeUpdate, shapeNodeLinkCreate, shapeNodeLinkUpdate, shapeOutputCreate, shapeOutputUpdate, shapeResourceListCreate, shapeResourceListUpdate, shapeTagCreate, shapeTagUpdate, TagShape } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type RoutineTranslationShape = Omit<ShapeWrapper<RoutineTranslation>, 'language' | 'instructions' | 'title'> & {
    id: string;
    language: RoutineTranslationCreateInput['language'];
    instructions: RoutineTranslationCreateInput['instructions'];
    title: RoutineTranslationCreateInput['title'];
}

export type RoutineShape = Omit<ShapeWrapper<Routine>, 'complexity' | 'simplicity' | 'inputs' | 'nodeLinks' | 'owner' | 'nodes' | 'outputs' | 'resourceLists' | 'runs' | 'tags'> & {
    id: string;
    inputs: InputShape[];
    nodeLinks?: NodeLinkShape[] | null;
    nodes?: Omit<NodeShape, 'routineId'>[] | null;
    outputs?: OutputShape[] | null;
    owner?: {
        __typename: 'User' | 'Organization';
        id: string;
    } | null;
    parent?: {
        id: string
    } | null;
    resourceLists?: ResourceListShape[] | null;
    tags?: TagShape[] | null;
    translations: RoutineTranslationShape[];
}

export const shapeRoutineTranslationCreate = (item: RoutineTranslationShape): RoutineTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    instructions: item.instructions,
    title: item.title,
})

export const shapeRoutineTranslationUpdate = (
    original: RoutineTranslationShape,
    updated: RoutineTranslationShape
): RoutineTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
        instructions: u.instructions !== o.instructions ? u.instructions : undefined,
        title: u.title !== o.title ? u.title : undefined,
    }), 'id')

export const shapeRoutineCreate = (item: RoutineShape): RoutineCreateInput => ({
    id: item.id,
    isAutomatable: item.isAutomatable,
    isComplete: item.isComplete,
    isInternal: item.isInternal,
    version: item.version,
    parentId: item.parent?.id,
    // projectId: item.p
    createdByUserId: item.owner?.__typename === ObjectType.User ? item.owner.id : undefined,
    createdByOrganizationId: item.owner?.__typename === ObjectType.Organization ? item.owner.id : undefined,
    ...shapeCreateList({
        nodes: item.nodes?.map(n => ({ ...n, routineId: item.id }))
    }, 'nodes', shapeNodeCreate),
    ...shapeCreateList(item, 'nodeLinks', shapeNodeLinkCreate),
    ...shapeCreateList(item, 'inputs', shapeInputCreate),
    ...shapeCreateList(item, 'outputs', shapeOutputCreate),
    ...shapeCreateList(item, 'translations', shapeRoutineTranslationCreate),
    ...shapeCreateList(item, 'resourceLists', shapeResourceListCreate),
    ...shapeCreateList(item, 'tags', shapeTagCreate),
})

export const shapeRoutineUpdate = (
    original: RoutineShape,
    updated: RoutineShape
): RoutineUpdateInput | undefined => 
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        isAutomatable: u.isAutomatable,
        isComplete: u.isComplete,
        isInternal: u.isInternal,
        version: u.version,
        parentId: u.parent?.id,
        // projectId: u.p
        userId: u.owner?.__typename === ObjectType.User ? u.owner.id : undefined,
        organizationId: u.owner?.__typename === ObjectType.Organization ? u.owner.id : undefined,
        ...shapeUpdateList({
            nodes: o.nodes?.map(n => ({ ...n, routineId: o.id }))
        }, {
            nodes: u.nodes?.map(n => ({ ...n, routineId: u.id }))
        }, 'nodes', hasObjectChanged, shapeNodeCreate, shapeNodeUpdate, 'id'),
        ...shapeUpdateList(o, u, 'nodeLinks', hasObjectChanged, shapeNodeLinkCreate, shapeNodeLinkUpdate, 'id'),
        ...shapeUpdateList(o, u, 'inputs', hasObjectChanged, shapeInputCreate, shapeInputUpdate, 'id'),
        ...shapeUpdateList(o, u, 'outputs', hasObjectChanged, shapeOutputCreate, shapeOutputUpdate, 'id'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeRoutineTranslationCreate, shapeRoutineTranslationUpdate, 'id'),
        ...shapeUpdateList(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate, 'id'),
        ...shapeUpdateList(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate, 'tag', true, true),
    }), 'id')