import { ResourceList, ResourceListTranslation } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";
import { resource } from "./resource";

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
