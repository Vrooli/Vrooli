import { ApiKeyModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ApiKey" as const;
export const ApiKeyFormat: Formatter<ApiKeyModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
    },
    countFields: {},
};
