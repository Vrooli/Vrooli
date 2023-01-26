import { Routine, RoutineYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const routineYouPartial: GqlPartial<RoutineYou> = {
    __typename: 'RoutineYou',
    full: {
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
}

export const routinePartial: GqlPartial<Routine> = {
    __typename: 'Routine',
    common: {
        id: true,
        created_at: true,
        isInternal: true,
        isPrivate: true,
        issuesCount: true,
        labels: () => relPartial(require('./label').labelPartial, 'list'),
        owner: {
            __union: {
                Organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
                User: () => relPartial(require('./user').userPartial, 'nav'),
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: () => relPartial(require('./tag').tagPartial, 'list'),
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
    }
}