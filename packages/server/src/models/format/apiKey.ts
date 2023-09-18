import { ApiKeyModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ApiKeyFormat: Formatter<ApiKeyModelLogic> = {
    gqlRelMap: {
        __typename: "ApiKey",
    },
    prismaRelMap: {
        __typename: "ApiKey",
    },
    countFields: {},
};
