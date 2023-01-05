import { Routine, RoutineCreateInput, RoutineUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { NodeLinkShape, NodeShape, OutputShape, ResourceListShape, TagShape, createPrims, updatePrims, shapeUpdate, shapeRoutineVersionInput, shapeNodeLink, shapeRoutineVersionOutput, shapeRoutineVersionTranslation, shapeResourceList, updateRel, createRel, shapeTag, shapeResourceListTranslation, RoutineVersionInputShape, updateOwner, createOwner, shapeNode } from "utils";


export type RoutineShape = Omit<OmitCalculated<Routine>, 'complexity' | 'simplicity' | 'inputs' | 'nodeLinks' | 'owner' | 'parent' | 'nodes' | 'outputs' | 'resourceLists' | 'runs' | 'tags' | 'translations'> & {
    id: string;
    inputs: RoutineVersionInputShape[];
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
    tags?: ({ tag: string } | TagShape)[] | null;
    translations: RoutineVersionTranslationShape[];
}

export const shapeRoutine: ShapeModel<RoutineShape, RoutineCreateInput, RoutineUpdateInput> = {
    create: (d) => ({
        id: d.id,
        isAutomatable: d.isAutomatable,
        isComplete: d.isComplete,
        isInternal: d.isInternal,
        isPrivate: d.isPrivate,
        // version: d.version,TODO
        parentId: d.parent?.id,
        projectId: d.project?.id,
        ...createOwner(d),
        ...createRel(d, 'nodes', ['Create'], 'many', shapeNode, (n) => ({ ...n, routineVersion: { id: d.id } })),
        ...createRel(d, 'nodeLinks', ['Create'], 'many', shapeNodeLink),
        ...createRel(d, 'inputs', ['Create'], 'many', shapeRoutineVersionInput),
        ...createRel(d, 'outputs', ['Create'], 'many', shapeRoutineVersionOutput),
        ...createRel(d, 'resourceList', ['Create'], 'one', shapeResourceList),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createRel(d, 'translations', ['Create'], 'many', shapeRoutineVersionTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isAutomatable', 'isComplete', 'isInternal', 'isPrivate'),
        // ...shapePrim(o, u, 'version'),TODO
        // ...shapePrim(o, u, 'parentId'),
        // ...shapePrim(o, u, 'projectId'),
        ...updateOwner(o, u),
        ...updateRel(o, u, 'nodes', ['Create', 'Update', 'Delete'], 'many', shapeNode, (d, i) => ({ ...d, routineVersion: { id: i.id } })),
        ...updateRel(o, u, 'nodeLinks', ['Create', 'Update', 'Delete'], 'many', shapeNodeLink),
        ...updateRel(o, u, 'inputs', ['Create', 'Update', 'Delete'], 'many', shapeRoutineVersionInput),
        ...updateRel(o, u, 'outputs',['Create', 'Update', 'Delete'], 'many', shapeRoutineVersionOutput),
        ...updateRel(o, u, 'resourceList', ['Create', 'Update'], 'one', shapeResourceList),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeResourceListTranslation),
    })
}