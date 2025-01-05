import { ApiKey } from "@local/shared";
import { GqlPartial } from "../types";

export const apiKey: GqlPartial<ApiKey> = {
    full: {
        id: true,
        creditsUsed: true,
        creditsUsedBeforeLimit: true,
        stopAtLimit: true,
        absoluteMax: true,
        resetsAt: true,
    },
};
