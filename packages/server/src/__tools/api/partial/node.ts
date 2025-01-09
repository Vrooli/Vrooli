import { Node, NodeTranslation } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const nodeTranslation: ApiPartial<NodeTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};


export const node: ApiPartial<Node> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        columnIndex: true,
        nodeType: true,
        rowIndex: true,
        // loopCreate: async () => relPartial((await import('./nodeLoop').nodeLoopCreatePartial, 'full', { omit: 'node' }),
        end: async () => rel((await import("./nodeEnd")).nodeEnd, "full", { omit: "node" }),
        routineList: async () => rel((await import("./nodeRoutineList")).nodeRoutineList, "full", { omit: "node" }),
        routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "full", { omit: ["nodes", "nodeLinks"] }),
        translations: () => rel(nodeTranslation, "full"),
    },
};
