import { NodeLinkWhenModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "NodeLinkWhen" as const;
export const NodeLinkWhenFormat: Formatter<NodeLinkWhenModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        link: "NodeLink",
    },
    countFields: {},
};
