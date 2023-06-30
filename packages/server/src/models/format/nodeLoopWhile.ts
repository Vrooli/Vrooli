import { NodeLoopWhileModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "NodeLoopWhile" as const;
export const NodeLoopWhileFormat: Formatter<NodeLoopWhileModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        loop: "NodeLoop",
    },
    countFields: {},
};
