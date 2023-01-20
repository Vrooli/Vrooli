import { ResourceList, ResourceListTranslation } from "@shared/consts";
import { GqlPartial } from "types";
import { resourcePartial } from "./resource";

export const resourceListTranslationPartial: GqlPartial<ResourceListTranslation> = {
    __typename: 'ResourceListTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}

export const resourceListPartial: GqlPartial<ResourceList> = {
    __typename: 'ResourceList',
    full: {
        id: true,
        created_at: true,
        translations: resourceListTranslationPartial.full,
        resources: resourcePartial.full,
    },
}