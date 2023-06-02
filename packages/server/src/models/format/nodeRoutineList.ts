import { NodeRoutineListModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "NodeRoutineList" as const;
export const NodeRoutineListFormat: Formatter<NodeRoutineListModelLogic> = {
    gqlRelMap: {
        __typename,
        items: "NodeRoutineListItem",
    },
    prismaRelMap: {
        __typename,
        node: "Node",
        items: "NodeRoutineListItem",
    },
    countFields: {},
};
