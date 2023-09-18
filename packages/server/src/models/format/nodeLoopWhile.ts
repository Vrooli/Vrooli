import { NodeLoopWhileModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NodeLoopWhileFormat: Formatter<NodeLoopWhileModelLogic> = {
    gqlRelMap: {
        __typename: "NodeLoopWhile",
    },
    prismaRelMap: {
        __typename: "NodeLoopWhile",
        loop: "NodeLoop",
    },
    countFields: {},
};
