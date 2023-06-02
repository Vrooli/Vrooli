import { NodeLinkModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "NodeLink" as const;
export const NodeLinkFormat: Formatter<NodeLinkModelLogic> = {
    gqlRelMap: {
        __typename,
        whens: "NodeLinkWhen",
    },
    prismaRelMap: {
        __typename,
        from: "Node",
        to: "Node",
        routineVersion: "RoutineVersion",
        whens: "NodeLinkWhen",
    },
    countFields: {},
};
