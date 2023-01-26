import { Standard, StandardYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const standardYouPartial: GqlPartial<StandardYou> = {
    __typename: 'StandardYou',
    common: {
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
    full: {},
    list: {},
}

export const standardPartial: GqlPartial<Standard> = {
    __typename: 'Standard',
    common: {
        id: true,
        created_at: true,
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
        you: () => relPartial(standardYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(require('./standardVersion').standardVersionPartial, 'full', { omit: 'root' }),
        stats: () => relPartial(require('./statsStandard').statsStandardPartial, 'full'),
    },
    list: {
        versions: () => relPartial(require('./standardVersion').standardVersionPartial, 'list', { omit: 'root' }),
    }
}