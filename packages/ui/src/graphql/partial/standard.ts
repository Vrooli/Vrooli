import { Standard, StandardYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { labelPartial } from "./label";
import { organizationPartial } from "./organization";
import { standardVersionPartial } from "./standardVersion";
import { statsStandardPartial } from "./statsStandard";
import { tagPartial } from "./tag";
import { userPartial } from "./user";

export const standardYouPartial: GqlPartial<StandardYou> = {
    __typename: 'StandardYou',
    full: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canTransfer: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    },
}

export const standardPartial: GqlPartial<Standard> = {
    __typename: 'Standard',
    common: {
        id: true,
        created_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: () => relPartial(labelPartial, 'list'),
        owner: {
            __union: {
                Organization: () => relPartial(organizationPartial, 'nav'),
                User: () => relPartial(userPartial, 'nav'),
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: () => relPartial(tagPartial, 'list'),
        transfersCount: true,
        views: true,
        you: () => relPartial(standardYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(standardVersionPartial, 'full', { omit: 'root' }),
        stats: () => relPartial(statsStandardPartial, 'full'),
    },
    list: {
        versions: () => relPartial(standardVersionPartial, 'list'),
    }
}