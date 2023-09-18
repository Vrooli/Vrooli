import { NodeLoopModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NodeLoopFormat: Formatter<NodeLoopModelLogic> = {
    gqlRelMap: {
        __typename: "NodeLoop",
        whiles: "NodeLoopWhile",
    },
    prismaRelMap: {
        __typename: "NodeLoop",
        node: "Node",
        whiles: "NodeLoopWhile",
    },
    countFields: {},
};
