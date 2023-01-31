import { Node, NodeTranslation } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const nodeTranslation: GqlPartial<NodeTranslation> = {
    __typename: 'NodeTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}


export const node: GqlPartial<Node> = {
    __typename: 'Node',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        columnIndex: true,
        nodeType: true,
        rowIndex: true,
        // loopCreate: async () => relPartial((await import('./nodeLoop').nodeLoopCreatePartial, 'full', { omit: 'node' }),
        end: async () => rel((await import('./nodeEnd')).nodeEnd, 'full', { omit: 'node' }),
        routineList: async () => rel((await import('./nodeRoutineList')).nodeRoutineList, 'full', { omit: 'node' }),
        routineVersion: async () => rel((await import('./routineVersion')).routineVersion, 'full', { omit: ['nodes', 'nodeLinks'] }),
        translations: () => rel(nodeTranslation, 'full'),
    },
    full: {},
    list: {},
}