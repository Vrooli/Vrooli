import { Post, PostTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const postTranslationPartial: GqlPartial<PostTranslation> = {
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
        resourceList: async () => relPartial((await import('./resourceList')).resourceListPartial, 'full'),
        translations: () => relPartial(postTranslationPartial, 'full'),
    },
    list: {
        resourceList: async () => relPartial((await import('./resourceList')).resourceListPartial, 'list'),
        translations: () => relPartial(postTranslationPartial, 'list'),
    },
    nav: {
        id: true,
        translations: () => relPartial(postTranslationPartial, 'list'),
    }
}