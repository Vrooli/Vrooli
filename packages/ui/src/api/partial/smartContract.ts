import { SmartContract, SmartContractYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const smartContractYouPartial: GqlPartial<SmartContractYou> = {
    __typename: 'SmartContractYou',
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

export const smartContractPartial: GqlPartial<SmartContract> = {
    __typename: 'SmartContract',
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
        you: () => relPartial(smartContractYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(require('./smartContractVersion').smartContractVersionPartial, 'full', { omit: 'root' }),
        stats: () => relPartial(require('./statsSmartContract').statsSmartContractPartial, 'full'),
    },
    list: {
        versions: () => relPartial(require('./smartContractVersion').smartContractVersionPartial, 'list'),
    }
}