import { ApiKey } from "@local/shared";
import { ApiPartial } from "../types.js";

export const apiKey: ApiPartial<ApiKey> = {
    full: {
        id: true,
        creditsUsed: true,
        creditsUsedBeforeLimit: true,
        stopAtLimit: true,
        absoluteMax: true,
        resetsAt: true,
    },
};
