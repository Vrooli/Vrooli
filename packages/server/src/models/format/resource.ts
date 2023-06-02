import { ResourceModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Resource" as const;
export const ResourceFormat: Formatter<ResourceModelLogic> = {
    gqlRelMap: {
        __typename,
        list: "ResourceList",
    },
    prismaRelMap: {
        __typename,
        list: "ResourceList",
    },
    countFields: {},
};
