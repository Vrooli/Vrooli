import { ShapeWrapper } from "types";
import { hasObjectChanged, InputShape, NodeLinkShape, NodeShape, OutputShape, ResourceListShape, shapeInputCreate, shapeInputUpdate, shapeNodeCreate, shapeNodeUpdate, shapeNodeLinkCreate, shapeNodeLinkUpdate, shapeOutputCreate, shapeOutputUpdate, shapeResourceListCreate, shapeResourceListUpdate, shapeTagCreate, shapeTagUpdate, TagShape } from "utils";
import { shapeCreateList, shapePrim, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type RoutineTranslationShape = Omit<ShapeWrapper<RoutineTranslation>, 'language' | 'instructions' | 'name'> & {
    id: string;
    language: RoutineTranslationCreateInput['language'];
    instructions: RoutineTranslationCreateInput['instructions'];
    name: RoutineTranslationCreateInput['name'];
}

export type RoutineShape = Omit<ShapeWrapper<Routine>, 'complexity' | 'simplicity' | 'inputs' | 'nodeLinks' | 'owner' | 'parent' | 'nodes' | 'outputs' | 'resourceLists' | 'runs' | 'tags' | 'translations'> & {
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
    project?: {
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
    name: item.name,
})

export const shapeRoutineTranslationUpdate = (
    original: RoutineTranslationShape,
    updated: RoutineTranslationShape
): RoutineTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        ...shapePrim(o, u, 'description'),
        ...shapePrim(o, u, 'instructions'),
        ...shapePrim(o, u, 'name'),
    }), 'id')

export const shapeRoutineCreate = (item: RoutineShape): RoutineCreateInput => ({
    id: item.id,
    isAutomatable: item.isAutomatable,
    isComplete: item.isComplete,
    isInternal: item.isInternal,
    isPrivate: item.isPrivate,
    // version: item.version,TODO
    parentId: item.parent?.id,
    projectId: item.project?.id,
    createdByUserId: item.owner?.__typename === 'User' ? item.owner.id : undefined,
    createdByOrganizationId: item.owner?.__typename === 'Organization' ? item.owner.id : undefined,
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
): RoutineUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    ...shapePrim(o, u, 'isAutomatable'),
    ...shapePrim(o, u, 'isComplete'),
    ...shapePrim(o, u, 'isInternal'),
    ...shapePrim(o, u, 'isPrivate'),
    // ...shapePrim(o, u, 'version'),TODO
    // ...shapePrim(o, u, 'parentId'),
    // ...shapePrim(o, u, 'projectId'),
    userId: u.owner?.__typename === 'User' ? u.owner.id : undefined,
    organizationId: u.owner?.__typename === 'Organization' ? u.owner.id : undefined,
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