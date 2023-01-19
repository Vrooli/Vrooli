import { Resource, ResourceTranslation } from "@shared/consts";
import { GqlPartial } from "types";

export const resourceTranslationPartial: GqlPartial<ResourceTranslation> = {
    __typename: 'ResourceTranslation',
    full: () => ({
        id: true,
        language: true,
        description: true,
        name: true,
    }),
}

export const resourcePartial: GqlPartial<Resource> = {
    __typename: 'Resource',
    full: () => ({
        id: true,
        index: true,
        link: true,
        usedFor: true,
        translations: resourceTranslationPartial.full,
    }),
}