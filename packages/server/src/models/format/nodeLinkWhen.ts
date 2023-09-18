import { NodeLinkWhenModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NodeLinkWhenFormat: Formatter<NodeLinkWhenModelLogic> = {
    gqlRelMap: {
        __typename: "NodeLinkWhen",
    },
    prismaRelMap: {
        __typename: "NodeLinkWhen",
        link: "NodeLink",
    },
    countFields: {},
};
