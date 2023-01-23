import { Post, PostTranslation } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const postTranslationPartial: GqlPartial<PostTranslation> = {
    __typename: 'PostTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}

export const postPartial: GqlPartial<Post> = {
    __typename: 'Post',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        commentsCount: true,
        repostsCount: true,
        score: true,
        stars: true, 
        views: true,
    },
    full: {
        resourceList: () => relPartial(require('./resourceList').resourceListPartial, 'full'),
        translations: () => relPartial(postTranslationPartial, 'full'),
    },
    list: {
        resourceList: () => relPartial(require('./resourceList').resourceListPartial, 'list'),
        translations: () => relPartial(postTranslationPartial, 'list'),
    }
}