import { Routine, RoutineYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { labelPartial } from "./label";
import { organizationPartial } from "./organization";
import { statsRoutinePartial } from "./statsRoutine";
import { tagPartial } from "./tag";
import { userPartial } from "./user";

export const routineYouPartial: GqlPartial<RoutineYou> = {
    __typename: 'RoutineYou',
    full: () => ({
        canComment: true,
        canDelete: true,
        canEdit: true,
        canStar: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    }),
}

export const routinePartial: GqlPartial<Routine> = {
    __typename: 'Routine',
    common: () => ({
        id: true,
        created_at: true,
        isInternal: true,
        isPrivate: true,
        issuesCount: true,
        labels: relPartial(labelPartial, 'list'),
        owner: {
            __union: {
                Organization: relPartial(organizationPartial, 'nav'),
                User: relPartial(userPartial, 'nav'),
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: relPartial(tagPartial, 'list'),
        transfersCount: true,
        views: true,
        you: relPartial(routineYouPartial, 'full'),
    }),
    full: () => ({
        versions: relPartial(routineVersionPartial, 'full', { omit: 'root' }),
        stats: relPartial(statsRoutinePartial, 'full'),
    }),
    list: () => ({
        versions: relPartial(routineVersionPartial, 'list'),
    })
}