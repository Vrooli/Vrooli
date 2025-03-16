import { Resource, ResourceTranslation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const resourceTranslation: ApiPartial<ResourceTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const resource: ApiPartial<Resource> = {
    common: {
        id: true,
        index: true,
        link: true,
        usedFor: true,
        translations: () => rel(resourceTranslation, "full"),
    },
};
