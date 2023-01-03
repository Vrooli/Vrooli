import { Routine, RoutineCreateInput, RoutineUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { hasObjectChanged, InputShape, NodeLinkShape, NodeShape, OutputShape, ResourceListShape, shapeInputCreate, shapeInputUpdate, shapeNodeCreate, shapeNodeUpdate, shapeNodeLinkCreate, shapeNodeLinkUpdate, shapeOutputCreate, shapeOutputUpdate, shapeResourceListCreate, shapeResourceListUpdate, shapeTagCreate, shapeTagUpdate, TagShape, createPrims, updatePrims, shapeUpdate } from "utils";


export type RoutineShape = Omit<OmitCalculated<Routine>, 'complexity' | 'simplicity' | 'inputs' | 'nodeLinks' | 'owner' | 'parent' | 'nodes' | 'outputs' | 'resourceLists' | 'runs' | 'tags' | 'translations'> & {
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
    translations: RoutineVersionTranslationShape[];
}

export const shapeRoutine: ShapeModel<RoutineShape, RoutineCreateInput, RoutineUpdateInput> = {
    create: (item) => ({
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
        ...createRel({
            nodes: item.nodes?.map(n => ({ ...n, routineId: item.id }))
        }, 'nodes', shapeNodeCreate),
        ...createRel(item, 'nodeLinks', shapeNodeLinkCreate),
        ...createRel(item, 'inputs', shapeInputCreate),
        ...createRel(item, 'outputs', shapeOutputCreate),
        ...createRel(item, 'translations', shapeRoutineTranslationCreate),
        ...createRel(item, 'resourceLists', shapeResourceListCreate),
        ...createRel(item, 'tags', shapeTagCreate),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isAutomatable', 'isComplete', 'isInternal', 'isPrivate'),
        // ...shapePrim(o, u, 'version'),TODO
        // ...shapePrim(o, u, 'parentId'),
        // ...shapePrim(o, u, 'projectId'),
        userId: u.owner?.__typename === 'User' ? u.owner.id : undefined,
        organizationId: u.owner?.__typename === 'Organization' ? u.owner.id : undefined,
        ...updateRel({
            nodes: o.nodes?.map(n => ({ ...n, routineId: o.id }))
        }, {
            nodes: u.nodes?.map(n => ({ ...n, routineId: u.id }))
        }, 'nodes', hasObjectChanged, shapeNodeCreate, shapeNodeUpdate, 'id'),
        ...updateRel(o, u, 'nodeLinks', hasObjectChanged, shapeNodeLinkCreate, shapeNodeLinkUpdate, 'id'),
        ...updateRel(o, u, 'inputs', hasObjectChanged, shapeInputCreate, shapeInputUpdate, 'id'),
        ...updateRel(o, u, 'outputs', hasObjectChanged, shapeOutputCreate, shapeOutputUpdate, 'id'),
        ...updateRel(o, u, 'translations', hasObjectChanged, shapeRoutineTranslationCreate, shapeRoutineTranslationUpdate, 'id'),
        ...updateRel(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate, 'id'),
        ...updateRel(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate, 'tag', true, true),
    })
}