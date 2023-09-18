import { NodeLinkModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NodeLinkFormat: Formatter<NodeLinkModelLogic> = {
    gqlRelMap: {
        __typename: "NodeLink",
        whens: "NodeLinkWhen",
    },
    prismaRelMap: {
        __typename: "NodeLink",
        from: "Node",
        to: "Node",
        routineVersion: "RoutineVersion",
        whens: "NodeLinkWhen",
    },
    countFields: {},
};
