import { Api, ApiYou } from "@shared/consts";
import { GqlPartial } from "types";
import { apiVersionPartial } from "./apiVersion";
import { organizationPartial } from "./organization";
import { tagPartial } from "./tag";

export const apiYouPartial: GqlPartial<ApiYou> = {
    __typename: 'ApiYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
        canStar: true,
        canTransfer: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    }),
}

export const apiPartial: GqlPartial<Api> = {
    __typename: 'Api',
    common: () => ({
        id: true,
        created_at: true,
        isPrivate: true,
        issuesCount: true,
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: tagPartial.list,
        transfersCount: true,
        views: true,
        you: apiYouPartial.full,
    }),
    full: () => ({
        owner: {
            __union: {
                Organization: organizationPartial.nav,
                User: userPartial.nav
            }
        },
        versions: without(apiVersionPartial.full, 'root'),
        labels: labelPartial.list,
        stats: apiStatsPartial.full,
    }),
    list: () => ({
        versions: apiVersionPartial.list,
        labels: labelPartial.list,
    })
}