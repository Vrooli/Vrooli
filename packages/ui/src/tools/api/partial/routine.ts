import { Routine, RoutineYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const routineYou: GqlPartial<RoutineYou> = {
    __typename: 'RoutineYou',
    common: {
        canComment: true,
        canDelete: true,
        canEdit: true,
        canStar: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    },
    full: {},
    list: {},
}

export const routine: GqlPartial<Routine> = {
    __typename: 'Routine',
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
        isInternal: true,
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
        you: () => rel(routineYou, 'full'),
    },
    full: {
        parent: async () => rel((await import('./routineVersion')).routineVersion, 'nav'),
        versions: async () => rel((await import('./routineVersion')).routineVersion, 'full', { omit: 'root' }),
        stats: async () => rel((await import('./statsRoutine')).statsRoutine, 'full'),
    },
    list: {
        versions: async () => rel((await import('./routineVersion')).routineVersion, 'list', { omit: 'root' }),
    },
    nav: {
        id: true,
        isInternal: true,
        isPrivate: true,
    }
}