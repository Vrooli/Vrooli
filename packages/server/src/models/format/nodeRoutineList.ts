import { NodeRoutineListModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NodeRoutineListFormat: Formatter<NodeRoutineListModelLogic> = {
    gqlRelMap: {
        __typename: "NodeRoutineList",
        items: "NodeRoutineListItem",
    },
    prismaRelMap: {
        __typename: "NodeRoutineList",
        node: "Node",
        items: "NodeRoutineListItem",
    },
    countFields: {},
};
