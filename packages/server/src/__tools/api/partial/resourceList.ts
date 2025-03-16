import { ResourceList, ResourceListTranslation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";
import { resource } from "./resource.js";

export const resourceListTranslation: ApiPartial<ResourceListTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const resourceList: ApiPartial<ResourceList> = {
    common: {
        id: true,
        created_at: true,
        translations: () => rel(resourceListTranslation, "full"),
        resources: () => rel(resource, "full"),
    },
};
