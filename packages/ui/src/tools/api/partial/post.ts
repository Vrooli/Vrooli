import { Post, PostTranslation } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const postTranslation: GqlPartial<PostTranslation> = {
    __typename: 'PostTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const post: GqlPartial<Post> = {
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
        resourceList: async () => rel((await import('./resourceList')).resourceList, 'full'),
        translations: () => rel(postTranslation, 'full'),
    },
    list: {
        resourceList: async () => rel((await import('./resourceList')).resourceList, 'list'),
        translations: () => rel(postTranslation, 'list'),
    },
    nav: {
        id: true,
        translations: () => rel(postTranslation, 'list'),
    }
}