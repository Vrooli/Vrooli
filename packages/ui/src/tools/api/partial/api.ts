import { Api, ApiYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const apiYou: GqlPartial<ApiYou> = {
    __typename: 'ApiYou',
    full: {
        canDelete: true,
        canStar: true,
        canTransfer: true,
        canUpdate: true,
        canRead: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    },
}

export const api: GqlPartial<Api> = {
    __typename: 'Api',
    common: {
        __define: {
            0: async () => rel((await import('./organization')).organization, 'nav'),
            1: async () => rel((await import('./user')).user, 'nav'),
            2: async () => rel((await import('./tag')).tag, 'list'),
            3: async () => rel((await import('./label')).label, 'list'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: { __use: 3 },
        owner: {
            __union: {
                Organization: 0,
                User: 1,
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: { __use: 2 },
        transfersCount: true,
        views: true,
        you: () => rel(apiYou, 'full'),
    },
    full: {
        parent: async () => rel((await import('./apiVersion')).apiVersion, 'nav'),
        versions: async () => rel((await import('./apiVersion')).apiVersion, 'full', { omit: 'root' }),
        stats: async () => rel((await import('./statsApi')).statsApi, 'full'),
    },
    list: {
        versions: async () => rel((await import('./apiVersion')).apiVersion, 'list', { omit: 'root' }),
    },
    nav: {
        id: true,
        isPrivate: true,
    }
}