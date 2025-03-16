import { ApiKeyExternal } from "@local/shared";
import { ApiPartial } from "../types.js";

export const apiKeyExternal: ApiPartial<ApiKeyExternal> = {
    full: {
        id: true,
        disabledAt: true,
        name: true,
        service: true,
    },
};
