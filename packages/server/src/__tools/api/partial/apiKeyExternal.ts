import { type ApiKeyExternal } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const apiKeyExternal: ApiPartial<ApiKeyExternal> = {
    full: {
        id: true,
        disabledAt: true,
        name: true,
        service: true,
    },
};
