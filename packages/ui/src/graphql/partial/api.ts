import { Api, ApiYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { apiVersionPartial } from "./apiVersion";
import { labelPartial } from "./label";
import { organizationPartial } from "./organization";
import { statsApiPartial } from "./statsApi";
import { tagPartial } from "./tag";
import { userPartial } from "./user";

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
        you: relPartial(apiYouPartial, 'full'),
    }),
    full: () => ({
        versions: relPartial(apiVersionPartial, 'full', { omit: 'root' }),
        stats: relPartial(statsApiPartial, 'full'),
    }),
    list: () => ({
        versions: relPartial(apiVersionPartial, 'list'),
    })
}