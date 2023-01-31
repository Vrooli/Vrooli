import { ResourceList, ResourceListTranslation } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";
import { resource } from "./resource";

export const resourceListTranslation: GqlPartial<ResourceListTranslation> = {
    __typename: 'ResourceListTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const resourceList: GqlPartial<ResourceList> = {
    __typename: 'ResourceList',
    common: {
        id: true,
        created_at: true,
        translations: () => rel(resourceListTranslation, 'full'),
        resources: () => rel(resource, 'full'),
    },
    full: {},
    list: {},
}