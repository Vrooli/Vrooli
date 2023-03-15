import { ApiKey } from "@shared/consts";
import { GqlPartial } from "../types";

export const apiKey: GqlPartial<ApiKey> = {
    __typename: 'ApiKey',
    full: {
        id: true,
        creditsUsed: true,
        creditsUsedBeforeLimit: true,
        stopAtLimit: true,
        absoluteMax: true,
        resetsAt: true,
    },
}