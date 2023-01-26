import { Api, ApiYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const apiYouPartial: GqlPartial<ApiYou> = {
    __typename: 'ApiYou',
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

export const apiPartial: GqlPartial<Api> = {
    __typename: 'Api',
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
        you: () => relPartial(apiYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(require('./apiVersion').apiVersionPartial, 'full', { omit: 'root' }),
        stats: () => relPartial(require('./statsApi').statsApiPartial, 'full'),
    },
    list: {
        versions: () => relPartial(require('./apiVersion').apiVersionPartial, 'list', { omit: 'root' }),
    }
}