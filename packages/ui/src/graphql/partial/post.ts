import { Post, PostTranslation } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { resourceListPartial } from "./resourceList";

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
        resourceList: () => relPartial(resourceListPartial, 'full'),
        translations: () => relPartial(postTranslationPartial, 'full'),
    },
    list: {
        resourceList: () => relPartial(resourceListPartial, 'list'),
        translations: () => relPartial(postTranslationPartial, 'list'),
    }
}