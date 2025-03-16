import { ApiKey, ApiKeyCreated } from "@local/shared";
import { ApiPartial } from "../types.js";

export const apiKey: ApiPartial<ApiKey> = {
    full: {
        id: true,
        creditsUsed: true,
        disabledAt: true,
        limitHard: true,
        limitSoft: true,
        name: true,
        stopAtLimit: true,
    },
};

export const apiKeyCreated: ApiPartial<ApiKeyCreated> = {
    full: {
        ...apiKey.full,
        key: true,
    },
};
