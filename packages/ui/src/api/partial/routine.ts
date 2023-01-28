import { Routine, RoutineYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const routineYouPartial: GqlPartial<RoutineYou> = {
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

export const routinePartial: GqlPartial<Routine> = {
    __typename: 'Routine',
    common: {
        __define: {
            0: [require('./organization').organizationPartial, 'nav'],
            1: [require('./user').userPartial, 'nav'],
            2: [require('./tag').tagPartial, 'list'],
            3: [require('./label').labelPartial, 'list'],
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
        you: () => relPartial(routineYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(require('./routineVersion').routineVersionPartial, 'full', { omit: 'root' }),
        stats: () => relPartial(require('./statsRoutine').statsRoutinePartial, 'full'),
    },
    list: {
        versions: () => relPartial(require('./routineVersion').routineVersionPartial, 'list', { omit: 'root' }),
    },
    nav: {
        id: true,
        isInternal: true,
        isPrivate: true,
    }
}