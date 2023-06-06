import { NodeLoopModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "NodeLoop" as const;
export const NodeLoopFormat: Formatter<NodeLoopModelLogic> = {
    gqlRelMap: {
        __typename,
        whiles: "NodeLoopWhile",
    },
    prismaRelMap: {
        __typename,
        node: "Node",
        whiles: "NodeLoopWhile",
    },
    countFields: {},
};
