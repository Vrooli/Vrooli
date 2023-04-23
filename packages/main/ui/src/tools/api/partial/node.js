import { rel } from "../utils";
export const nodeTranslation = {
    __typename: "NodeTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};
export const node = {
    __typename: "Node",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        columnIndex: true,
        nodeType: true,
        rowIndex: true,
        end: async () => rel((await import("./nodeEnd")).nodeEnd, "full", { omit: "node" }),
        routineList: async () => rel((await import("./nodeRoutineList")).nodeRoutineList, "full", { omit: "node" }),
        routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "full", { omit: ["nodes", "nodeLinks"] }),
        translations: () => rel(nodeTranslation, "full"),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=node.js.map