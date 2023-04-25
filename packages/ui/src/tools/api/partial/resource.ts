import { Resource, ResourceTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const resourceTranslation: GqlPartial<ResourceTranslation> = {
    __typename: 'ResourceTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const resource: GqlPartial<Resource> = {
    __typename: 'Resource',
    common: {
        id: true,
        index: true,
        link: true,
        usedFor: true,
        translations: () => rel(resourceTranslation, 'full'),
    },
    full: {},
    list: {},
}