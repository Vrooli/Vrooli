import { Node, NodeTranslation } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { nodeEndPartial } from "./nodeEnd";
import { nodeRoutineListPartial } from "./nodeRoutineList";
import { routineVersionPartial } from "./routineVersion";

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
        // loopCreate: () => relPartial(nodeLoopCreatePartial, 'full', { omit: 'node' }),
        end: () => relPartial(nodeEndPartial, 'full', { omit: 'node' }),
        routineList: () => relPartial(nodeRoutineListPartial, 'full', { omit: 'node' }),
        routineVersion: () => relPartial(routineVersionPartial, 'full', { omit: ['nodes', 'nodeLinks'] }),
        translations: () => relPartial(nodeTranslationPartial, 'full'),
    },
}