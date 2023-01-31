import { Resource, ResourceTranslation } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const resourceTranslationPartial: GqlPartial<ResourceTranslation> = {
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

export const resourcePartial: GqlPartial<Resource> = {
    __typename: 'Resource',
    common: {
        id: true,
        index: true,
        link: true,
        usedFor: true,
        translations: () => relPartial(resourceTranslationPartial, 'full'),
    },
    full: {},
    list: {},
}