import { Node, NodeTranslation } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const nodeTranslationPartial: GqlPartial<NodeTranslation> = {
    __typename: 'NodeTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}


export const nodePartial: GqlPartial<Node> = {
    __typename: 'Node',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        columnIndex: true,
        nodeType: true,
        rowIndex: true,
        // loopCreate: () => relPartial(require().nodeLoopCreatePartial, 'full', { omit: 'node' }),
        end: () => relPartial(require('./nodeEnd').nodeEndPartial, 'full', { omit: 'node' }),
        routineList: () => relPartial(require('./nodeRoutineList').nodeRoutineListPartial, 'full', { omit: 'node' }),
        routineVersion: () => relPartial(require('./routineVersion').routineVersionPartial, 'full', { omit: ['nodes', 'nodeLinks'] }),
        translations: () => relPartial(nodeTranslationPartial, 'full'),
    },
}