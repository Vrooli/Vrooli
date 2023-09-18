import { ResourceModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ResourceFormat: Formatter<ResourceModelLogic> = {
    gqlRelMap: {
        __typename: "Resource",
        list: "ResourceList",
    },
    prismaRelMap: {
        __typename: "Resource",
        list: "ResourceList",
    },
    countFields: {},
};
